package utils

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/OpenSlides/openslides-cli/internal/logger"
)

// CreateFile creates a file in the given directory with the given content.
// Use a truthy value for force to override an existing file.
func CreateFile(dir string, force bool, name string, content []byte, perm fs.FileMode) error {
	p := path.Join(dir, name)

	pExists, err := FileExists(p)
	if err != nil {
		return fmt.Errorf("checking file existance: %w", err)
	}
	if !force && pExists {
		// No force-mode and file already exists, so skip this file.
		return nil
	}

	if err := os.WriteFile(p, content, perm); err != nil {
		return fmt.Errorf("creating and writing to file %q: %w", p, err)
	}
	return nil
}

// fileExists is a small helper function to check if a file already exists. It is not
// save in concurrent usage.
func FileExists(p string) (bool, error) {
	_, err := os.Stat(p)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, fmt.Errorf("checking existance of file %s: %w", p, err)
}

// ReadInputOrFileOrStdin reads data from either a positional argument, a file, or stdin
func ReadInputOrFileOrStdin(input, filename string) ([]byte, error) {
	logger.Debug("Reading input: direct=%v, file=%s", input != "", filename)

	if input == "" && filename == "" {
		return nil, fmt.Errorf("either input or filename must be provided")
	}
	if input != "" && filename != "" {
		return nil, fmt.Errorf("cannot provide both input and filename")
	}

	if input != "" {
		logger.Debug("Using direct input (%d bytes)", len(input))
		return []byte(input), nil
	}

	return ReadFromFileOrStdin(filename)
}

// ReadFromFileOrStdin reads from a file or stdin if filename is "-"
func ReadFromFileOrStdin(filename string) ([]byte, error) {
	logger.Debug("Reading from file or stdin: %s", filename)

	var reader io.Reader
	if filename == "-" {
		logger.Debug("Reading from stdin")
		reader = os.Stdin
	} else {
		logger.Debug("Reading from file: %s", filename)
		f, err := os.Open(filename)
		if err != nil {
			return nil, fmt.Errorf("opening file %q: %w", filename, err)
		}
		defer func() { _ = f.Close() }()
		reader = f
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("reading data: %w", err)
	}

	logger.Debug("Read %d bytes", len(data))
	return data, nil
}

// ReadPassword reads password from a file
func ReadPassword(passwordFile string) (string, error) {
	logger.Debug("Reading password from: %s", passwordFile)

	data, err := os.ReadFile(passwordFile)
	if err != nil {
		logger.Error("Failed to read password file: %v", err)
		return "", fmt.Errorf("reading password file %q: %w", passwordFile, err)
	}

	password := string(data)
	logger.Debug("Password read successfully (%d bytes)", len(password))
	return password, nil
}

// extractNamespace gets the namespace from instance directory path
// Example: "/real/path/to/my.instance.dir.url" -> "myinstancedirurl"
func ExtractNamespace(instanceDir string) string {
	dirName := filepath.Base(instanceDir)
	namespace := strings.ReplaceAll(dirName, ".", "")
	return namespace
}

// isYAMLFile checks if filename has YAML extension
func IsYAMLFile(filename string) bool {
	ext := filepath.Ext(filename)
	return ext == ".yaml" || ext == ".yml"
}
