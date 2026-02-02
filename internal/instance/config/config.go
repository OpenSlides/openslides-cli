package config

import (
	"bytes"
	_ "embed"
	"encoding/base64"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/OpenSlides/openslides-cli/internal/constants"
	"github.com/OpenSlides/openslides-cli/internal/logger"
	"github.com/OpenSlides/openslides-cli/internal/utils"

	"dario.cat/mergo"
	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"
)

const (
	ConfigHelp      = "(Re)creates deployment configuration files"
	ConfigHelpExtra = `(Re)creates deployment configuration files from templates.

Generates deployment files (Docker Compose or Kubernetes manifests) using
templates and YAML configuration files. Multiple config files are deep-merged
in order, with later files overriding earlier ones.

Template functions available:
  • marshalContent - Marshal YAML content with indentation
  • envMapToK8S     - Convert environment map to Kubernetes format
  • readSecret     - Read and base64-encode secrets from secrets/ directory

Examples:
  osmanage config ./my.instance.dir.org
  osmanage config ./my.instance.dir.org --template ./custom.tmpl --config ./config.yaml
  osmanage config ./my.instance.dir.org -t ./k8s-templates -c base.yaml -c overrides.yaml
  osmanage config ./my.instance.dir.org --force`
)

// Cmd returns the subcommand.
func Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config <instance-dir>",
		Short: ConfigHelp,
		Long:  ConfigHelp + "\n\n" + ConfigHelpExtra,
		Args:  cobra.ExactArgs(1),
	}

	force := cmd.Flags().BoolP("force", "f", false, "overwrite existing files")
	customTemplate := cmd.Flags().StringP("template", "t", "", "custom template file or directory")
	configFiles := cmd.Flags().StringArrayP("config", "c", nil, "custom YAML config file (can be used multiple times)")
	cmd.MarkFlagsRequiredTogether("template", "config")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		logger.Info("=== CONFIG ===")

		baseDir := args[0]
		logger.Debug("Base directory: %s", args[0])
		logger.Debug("Config files: %v", *configFiles)

		config, err := NewConfig(*configFiles)
		if err != nil {
			return fmt.Errorf("parsing configuration: %w", err)
		}

		if err := CreateDirAndFiles(baseDir, *force, *customTemplate, config); err != nil {
			return fmt.Errorf("creating deployment files: %w", err)
		}

		logger.Info("Config files created successfully")
		fmt.Printf("Configuration created in: %s\n", baseDir)
		return nil
	}

	return cmd
}

// NewConfig creates a configuration map by deep-merging all given files in order.
// Later files override existing keys and add new keys.
func NewConfig(configFileNames []string) (map[string]any, error) {
	logger.Debug("Loading configuration from %d file(s)", len(configFileNames))

	config := make(map[string]any)

	for _, filename := range configFileNames {
		logger.Debug("Reading config file: %s", filename)
		data, err := os.ReadFile(filename)
		if err != nil {
			return nil, fmt.Errorf("reading config file %q: %w", filename, err)
		}

		var fileConfig map[string]any
		if err := yaml.Unmarshal(data, &fileConfig); err != nil {
			return nil, fmt.Errorf("unmarshaling YAML from %q: %w", filename, err)
		}

		// Deep merge fileConfig into config
		if err := mergo.Merge(&config, fileConfig, mergo.WithOverride); err != nil {
			return nil, fmt.Errorf("merging config from %q: %w", filename, err)
		}
	}

	return config, nil
}

// CreateDirAndFiles creates the base directory and (re-)creates the deployment
// files according to the given template. Use a truthy value for force to
// override existing files.
func CreateDirAndFiles(baseDir string, force bool, customTemplate string, cfg map[string]any) error {
	logger.Debug("Creating deployment files - custom: %s", customTemplate)
	fileInfo, err := os.Stat(customTemplate)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("template file or directory %q does not exist", customTemplate)
		}
		return fmt.Errorf("checking file info of %q: %w", customTemplate, err)
	}

	if fileInfo.IsDir() {
		return createFromTemplateDir(baseDir, force, customTemplate, cfg)
	}

	return createFromTemplateFile(baseDir, force, customTemplate, cfg)
}

func createFromTemplateFile(baseDir string, force bool, tplFile string, cfg map[string]any) error {
	logger.Debug("Using custom template file: %s", tplFile)

	data, err := os.ReadFile(tplFile)
	if err != nil {
		return fmt.Errorf("reading template file: %w", err)
	}

	if err := os.MkdirAll(baseDir, constants.InstanceDirPerm); err != nil {
		return fmt.Errorf("creating instance directory: %w", err)
	}

	// Extract filename from config if present, otherwise use a default
	filename := filepath.Join(baseDir, getFilename(cfg))
	return createDeploymentFile(filename, force, data, cfg, baseDir)
}

func createFromTemplateDir(baseDir string, force bool, tplDir string, cfg map[string]any) error {
	logger.Debug("Using custom template directory: %s", tplDir)

	tplFS := os.DirFS(tplDir)

	if err := os.MkdirAll(baseDir, constants.InstanceDirPerm); err != nil {
		return fmt.Errorf("creating instance directory: %w", err)
	}

	return createFromFS(baseDir, force, tplFS, cfg)
}

func createFromFS(baseDir string, force bool, tplFS fs.FS, cfg map[string]any) error {
	return fs.WalkDir(tplFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		targetPath := filepath.Join(baseDir, path)

		if d.IsDir() {
			// Use appropriate permissions based on directory name
			perm := getDirPermissions(filepath.Base(targetPath))
			logger.Debug("Creating directory: %s (perms: %04o)", targetPath, perm)
			return os.MkdirAll(targetPath, perm)
		}

		logger.Debug("Processing template: %s", path)
		data, err := fs.ReadFile(tplFS, path)
		if err != nil {
			return fmt.Errorf("reading template %q: %w", path, err)
		}

		return createDeploymentFile(targetPath, force, data, cfg, baseDir)
	})
}

// getDirPermissions returns appropriate permissions based on directory name
func getDirPermissions(dirName string) fs.FileMode {
	switch dirName {
	case constants.SecretsDirName:
		return constants.SecretsDirPerm
	case constants.StackDirName:
		return constants.StackDirPerm
	default:
		return constants.InstanceDirPerm
	}
}

func createDeploymentFile(filename string, force bool, tplData []byte, cfg map[string]any, baseDir string) error {
	tf := &TemplateFunctions{baseDir: baseDir}
	tmpl, err := template.New("deployment").Funcs(tf.GetFuncMap()).Parse(string(tplData))
	if err != nil {
		return fmt.Errorf("parsing template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, cfg); err != nil {
		return fmt.Errorf("executing template: %w", err)
	}

	dir := filepath.Dir(filename)
	name := filepath.Base(filename)
	return utils.CreateFile(dir, force, name, buf.Bytes(), constants.StackFilePerm)
}

// getFilename extracts the filename from config, or returns a default
func getFilename(cfg map[string]any) string {
	if fn, ok := cfg["filename"].(string); ok && fn != "" {
		return fn
	}
	return "output.yml"
}

type TemplateFunctions struct {
	baseDir string
}

// GetFuncMap returns the template function map with context
func (tf *TemplateFunctions) GetFuncMap() template.FuncMap {
	return template.FuncMap{
		"marshalContent": marshalContent,
		"envMapToK8S":    envMapToK8S,
		"readSecret":     tf.ReadSecret,
	}
}

// ReadSecret reads a secret file from the secrets directory and returns it base64 encoded
func (tf *TemplateFunctions) ReadSecret(name string) (string, error) {
	secretPath := filepath.Join(tf.baseDir, constants.SecretsDirName, name)
	data, err := os.ReadFile(secretPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("secret %q does not exist - ensure secrets are created before deployment files", name)
		}
		return "", fmt.Errorf("reading secret %q: %w", name, err)
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

func marshalContent(ws int, v any) (string, error) {
	data, err := yaml.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("marshalling content: %w", err)
	}
	result := "\n"
	for line := range strings.SplitSeq(string(data), "\n") {
		if len(line) != 0 {
			result += fmt.Sprintf("%s%s\n", strings.Repeat(" ", ws), line)
		}
	}
	return strings.TrimRight(result, "\n"), nil
}

func envMapToK8S(v map[string]any) []map[string]string {
	// Handle map[string]any (from YAML unmarshaling)
	var list []map[string]string
	for key, value := range v {
		// Convert value to string
		strValue := fmt.Sprintf("%v", value)
		list = append(list, map[string]string{
			"name":  key,
			"value": strValue,
		})
	}
	return list
}
