package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/OpenSlides/openslides-cli/internal/constants"
)

func TestNewConfig(t *testing.T) {
	t.Run("empty config list", func(t *testing.T) {
		cfg, err := NewConfig([]string{})
		if err != nil {
			t.Errorf("NewConfig() error = %v", err)
		}
		if cfg == nil {
			t.Fatal("Expected config, got nil")
		}

		// With no config files, should just be an empty map
		if len(cfg) != 0 {
			t.Errorf("Expected empty config, got %d keys", len(cfg))
		}
	})

	t.Run("with custom config", func(t *testing.T) {
		tmpfile, err := os.CreateTemp("", "config-*.yml")
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if err := os.Remove(tmpfile.Name()); err != nil {
				t.Logf("warning: failed to remove temp file: %v", err)
			}
		})

		customConfig := `---
url: test.example.com
stackName: test-stack
host: 192.168.1.1
port: 9000
defaults:
  tag: custom-tag
  containerRegistry: example.com/registry
`
		if _, err := tmpfile.WriteString(customConfig); err != nil {
			t.Fatalf("failed to write test config: %v", err)
		}
		if err := tmpfile.Close(); err != nil {
			t.Fatalf("failed to close temp file: %v", err)
		}

		cfg, err := NewConfig([]string{tmpfile.Name()})
		if err != nil {
			t.Errorf("NewConfig() error = %v", err)
		}
		if cfg["url"] != "test.example.com" {
			t.Errorf("Expected url test.example.com, got %v", cfg["url"])
		}
		if cfg["host"] != "192.168.1.1" {
			t.Errorf("Expected host 192.168.1.1, got %v", cfg["host"])
		}
		if cfg["port"] != float64(9000) { // YAML unmarshals numbers as float64
			t.Errorf("Expected port 9000, got %v", cfg["port"])
		}

		// Check nested defaults
		defaults, ok := cfg["defaults"].(map[string]any)
		if !ok {
			t.Fatal("Expected defaults to be a map")
		}
		if defaults["tag"] != "custom-tag" {
			t.Errorf("Expected tag custom-tag, got %v", defaults["tag"])
		}
		if defaults["containerRegistry"] != "example.com/registry" {
			t.Errorf("Expected containerRegistry example.com/registry, got %v", defaults["containerRegistry"])
		}
	})

	t.Run("merge multiple configs", func(t *testing.T) {
		tmpdir := t.TempDir()

		config1 := filepath.Join(tmpdir, "config1.yml")
		if err := os.WriteFile(config1, []byte("host: 127.0.0.1\nport: 8000\n"), constants.StackFilePerm); err != nil {
			t.Fatalf("failed to write config1: %v", err)
		}

		config2 := filepath.Join(tmpdir, "config2.yml")
		if err := os.WriteFile(config2, []byte("port: 9000\nstackName: test-stack\n"), constants.StackFilePerm); err != nil {
			t.Fatalf("failed to write config2: %v", err)
		}

		cfg, err := NewConfig([]string{config1, config2})
		if err != nil {
			t.Errorf("NewConfig() error = %v", err)
		}

		// With mergo.WithOverride, later configs override existing keys
		if cfg["port"] != float64(9000) {
			t.Errorf("Expected port 9000 (from config2), got %v", cfg["port"])
		}
		if cfg["stackName"] != "test-stack" {
			t.Errorf("Expected stackName from config2, got %v", cfg["stackName"])
		}
		// host should still be present from config1 (new key added)
		if cfg["host"] != "127.0.0.1" {
			t.Errorf("Expected host 127.0.0.1 (from config1), got %v", cfg["host"])
		}
	})

	t.Run("deep merge nested structures", func(t *testing.T) {
		tmpdir := t.TempDir()

		config1 := filepath.Join(tmpdir, "config1.yml")
		if err := os.WriteFile(config1, []byte(`
services:
  client:
    tag: latest
    replicas: 3
`), constants.StackFilePerm); err != nil {
			t.Fatalf("failed to write config1: %v", err)
		}

		config2 := filepath.Join(tmpdir, "config2.yml")
		if err := os.WriteFile(config2, []byte(`
services:
  client:
    foo: bar
    replicas: 5
`), constants.StackFilePerm); err != nil {
			t.Fatalf("failed to write config2: %v", err)
		}

		cfg, err := NewConfig([]string{config1, config2})
		if err != nil {
			t.Errorf("NewConfig() error = %v", err)
		}

		// Verify deep merge worked
		services, ok := cfg["services"].(map[string]any)
		if !ok {
			t.Fatal("Expected services to be a map")
		}

		client, ok := services["client"].(map[string]any)
		if !ok {
			t.Fatal("Expected client to be a map")
		}

		// Should have both tag (from config1) and foo (from config2) - keys added
		if client["tag"] != "latest" {
			t.Errorf("Expected tag latest, got %v", client["tag"])
		}
		if client["foo"] != "bar" {
			t.Errorf("Expected foo bar, got %v", client["foo"])
		}
		// replicas should be overridden by config2
		if client["replicas"] != float64(5) {
			t.Errorf("Expected replicas 5, got %v", client["replicas"])
		}
	})

	t.Run("arbitrary nested keys", func(t *testing.T) {
		tmpdir := t.TempDir()

		config1 := filepath.Join(tmpdir, "config1.yml")
		if err := os.WriteFile(config1, []byte(`
services:
  client:
    tag: latest
`), constants.StackFilePerm); err != nil {
			t.Fatalf("failed to write config1: %v", err)
		}

		config2 := filepath.Join(tmpdir, "config2.yml")
		if err := os.WriteFile(config2, []byte(`
services:
  client:
    custom:
      deeply:
        nested:
          value: 42
`), constants.StackFilePerm); err != nil {
			t.Fatalf("failed to write config2: %v", err)
		}

		cfg, err := NewConfig([]string{config1, config2})
		if err != nil {
			t.Errorf("NewConfig() error = %v", err)
		}

		// Navigate to the deeply nested value
		services := cfg["services"].(map[string]any)
		client := services["client"].(map[string]any)
		custom := client["custom"].(map[string]any)
		deeply := custom["deeply"].(map[string]any)
		nested := deeply["nested"].(map[string]any)

		if nested["value"] != float64(42) {
			t.Errorf("Expected deeply nested value 42, got %v", nested["value"])
		}

		// Original tag should still be there (new keys added)
		if client["tag"] != "latest" {
			t.Errorf("Expected tag latest, got %v", client["tag"])
		}
	})

	t.Run("deep merge with defaults and services", func(t *testing.T) {
		tmpdir := t.TempDir()

		// Simulates default-config.yml - no services section needed!
		config1 := filepath.Join(tmpdir, "default.yml")
		if err := os.WriteFile(config1, []byte(`
defaults:
  containerRegistry: registry.example.com
  tag: latest
`), constants.StackFilePerm); err != nil {
			t.Fatalf("failed to write config1: %v", err)
		}

		// Simulates environment-specific config - only define services that need custom config
		config2 := filepath.Join(tmpdir, "production.yml")
		if err := os.WriteFile(config2, []byte(`
defaults:
  tag: 4.2.0
services:
  postgres:
    password: super-secret
  projector:
    replicas: 3
`), constants.StackFilePerm); err != nil {
			t.Fatalf("failed to write config2: %v", err)
		}

		cfg, err := NewConfig([]string{config1, config2})
		if err != nil {
			t.Errorf("NewConfig() error = %v", err)
		}

		// Check defaults were merged (tag overridden, containerRegistry preserved)
		defaults := cfg["defaults"].(map[string]any)
		if defaults["containerRegistry"] != "registry.example.com" {
			t.Errorf("Expected containerRegistry preserved, got %v", defaults["containerRegistry"])
		}
		if defaults["tag"] != "4.2.0" {
			t.Errorf("Expected tag overridden to 4.2.0, got %v", defaults["tag"])
		}

		// Check services were added
		services := cfg["services"].(map[string]any)

		// postgres should have password
		postgres := services["postgres"].(map[string]any)
		if postgres["password"] != "super-secret" {
			t.Errorf("Expected postgres password, got %v", postgres["password"])
		}

		// projector should have replicas
		projector := services["projector"].(map[string]any)
		if projector["replicas"] != float64(3) {
			t.Errorf("Expected projector replicas 3, got %v", projector["replicas"])
		}

		// auth service doesn't exist at all - that's fine with 'or' pattern
		if _, exists := services["auth"]; exists {
			t.Errorf("auth service should not exist, but it do")
		}
	})

	t.Run("invalid YAML file", func(t *testing.T) {
		tmpfile, err := os.CreateTemp("", "config-*.yml")
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if err := os.Remove(tmpfile.Name()); err != nil {
				t.Logf("warning: failed to remove temp file: %v", err)
			}
		})

		if _, err := tmpfile.WriteString("invalid: yaml: content:"); err != nil {
			t.Fatalf("failed to write invalid yaml: %v", err)
		}
		if err := tmpfile.Close(); err != nil {
			t.Fatalf("failed to close temp file: %v", err)
		}

		_, err = NewConfig([]string{tmpfile.Name()})
		if err == nil {
			t.Error("Expected error for invalid YAML")
		}
	})

	t.Run("nonexistent file", func(t *testing.T) {
		_, err := NewConfig([]string{"nonexistent.yml"})
		if err == nil {
			t.Error("Expected error for nonexistent file")
		}
	})
}

func TestGetFilename(t *testing.T) {
	t.Run("with filename in config", func(t *testing.T) {
		cfg := map[string]any{
			"filename": "custom.yml",
		}
		result := getFilename(cfg)
		if result != "custom.yml" {
			t.Errorf("Expected custom.yml, got %s", result)
		}
	})

	t.Run("without filename in config", func(t *testing.T) {
		cfg := map[string]any{
			"other": "value",
		}
		result := getFilename(cfg)
		if result != constants.DefaultConfigFile {
			t.Errorf("Expected %s, got %s", constants.DefaultConfigFile, result)
		}
	})

	t.Run("empty filename in config", func(t *testing.T) {
		cfg := map[string]any{
			"filename": "",
		}
		result := getFilename(cfg)
		if result != constants.DefaultConfigFile {
			t.Errorf("Expected %s for empty filename, got %s", constants.DefaultConfigFile, result)
		}
	})

	t.Run("wrong type for filename", func(t *testing.T) {
		cfg := map[string]any{
			"filename": 123,
		}
		result := getFilename(cfg)
		if result != constants.DefaultConfigFile {
			t.Errorf("Expected %s for non-string filename, got %s", constants.DefaultConfigFile, result)
		}
	})
}

func TestTemplateFunctions(t *testing.T) {
	tmpdir := t.TempDir()
	secretsDir := filepath.Join(tmpdir, constants.SecretsDirName)
	if err := os.MkdirAll(secretsDir, constants.SecretsDirPerm); err != nil {
		t.Fatalf("failed to create secrets dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(secretsDir, "test_secret"), []byte("secret123"), constants.SecretFilePerm); err != nil {
		t.Fatalf("failed to write test secret: %v", err)
	}

	tf := &TemplateFunctions{baseDir: tmpdir}

	t.Run("readSecret", func(t *testing.T) {
		result, err := tf.ReadSecret("test_secret")
		if err != nil {
			t.Errorf("ReadSecret() error = %v", err)
		}
		if result == "" {
			t.Error("Expected non-empty result")
		}
		// Should be base64 encoded
		if !strings.Contains(result, "c2VjcmV0MTIz") { // base64 of "secret123"
			t.Error("Expected base64 encoded secret")
		}
	})

	t.Run("readSecret not found", func(t *testing.T) {
		_, err := tf.ReadSecret("nonexistent")
		if err == nil {
			t.Error("Expected error for nonexistent secret")
		}
	})
}

func TestMarshalContent(t *testing.T) {
	t.Run("marshal map", func(t *testing.T) {
		data := map[string]string{
			"key1": "value1",
			"key2": "value2",
		}
		result, err := marshalContent(2, data)
		if err != nil {
			t.Errorf("marshalContent() error = %v", err)
		}
		if !strings.Contains(result, "key1") || !strings.Contains(result, "value1") {
			t.Error("Expected marshaled content")
		}
		// Check indentation (lines should start with 2 spaces)
		lines := strings.SplitSeq(result, "\n")
		for line := range lines {
			if line != "" && !strings.HasPrefix(line, "  ") {
				t.Errorf("Expected 2-space indentation, got: %q", line)
			}
		}
	})

	t.Run("marshal nested structure", func(t *testing.T) {
		data := map[string]any{
			"outer": map[string]string{
				"inner": "value",
			},
		}
		result, err := marshalContent(4, data)
		if err != nil {
			t.Errorf("marshalContent() error = %v", err)
		}
		if !strings.Contains(result, "outer") {
			t.Error("Expected outer key in marshaled content")
		}
	})
}

func TestEnvMapToK8S(t *testing.T) {
	t.Run("converts map to K8S env format", func(t *testing.T) {
		env := map[string]any{
			"VAR1":    "value1",
			"VAR2":    "value2",
			"PORT":    9000,
			"ENABLED": true,
		}

		result := envMapToK8S(env)

		if len(result) != 4 {
			t.Errorf("Expected 4 items, got %d", len(result))
		}

		// Check format and values are converted to strings
		foundVars := make(map[string]string)
		for _, item := range result {
			name, okName := item["name"]
			value, okValue := item["value"]
			if !okName || !okValue {
				t.Error("Expected 'name' and 'value' keys")
			}
			foundVars[name] = value
		}

		if foundVars["VAR1"] != "value1" {
			t.Errorf("Expected VAR1=value1, got %s", foundVars["VAR1"])
		}
		if foundVars["PORT"] != "9000" {
			t.Errorf("Expected PORT=9000, got %s", foundVars["PORT"])
		}
		if foundVars["ENABLED"] != "true" {
			t.Errorf("Expected ENABLED=true, got %s", foundVars["ENABLED"])
		}
	})

	t.Run("handles empty map", func(t *testing.T) {
		env := map[string]any{}

		result := envMapToK8S(env)

		if len(result) != 0 {
			t.Errorf("Expected 0 items, got %d", len(result))
		}
	})
}

func TestCreateDirAndFiles(t *testing.T) {
	tmpdir := t.TempDir()

	t.Run("invalid template path", func(t *testing.T) {
		cfg := map[string]any{
			"filename": constants.DefaultConfigFile,
		}
		err := CreateDirAndFiles(tmpdir, false, "nonexistent-template", cfg)
		if err == nil {
			t.Error("Expected error for nonexistent template")
		}
	})

	t.Run("template file", func(t *testing.T) {
		tplFile := filepath.Join(tmpdir, "template.yml")
		if err := os.WriteFile(tplFile, []byte("test: {{ .url }}"), constants.StackFilePerm); err != nil {
			t.Fatalf("failed to write template: %v", err)
		}

		outDir := filepath.Join(tmpdir, "output1")
		cfg := map[string]any{
			"filename": "result.yml",
			"url":      "example.com",
		}

		err := CreateDirAndFiles(outDir, true, tplFile, cfg)
		if err != nil {
			t.Errorf("CreateDirAndFiles() error = %v", err)
		}

		result, err := os.ReadFile(filepath.Join(outDir, "result.yml"))
		if err != nil {
			t.Fatalf("failed to read result file: %v", err)
		}
		if !strings.Contains(string(result), "example.com") {
			t.Error("Expected template to be processed")
		}
	})

	t.Run("template directory", func(t *testing.T) {
		tplDir := filepath.Join(tmpdir, "templates")
		if err := os.MkdirAll(tplDir, constants.StackDirPerm); err != nil {
			t.Fatalf("failed to create template dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(tplDir, "file1.yml"), []byte("content1"), constants.StackFilePerm); err != nil {
			t.Fatalf("failed to write file1: %v", err)
		}
		if err := os.WriteFile(filepath.Join(tplDir, "file2.yml"), []byte("content2"), constants.StackFilePerm); err != nil {
			t.Fatalf("failed to write file2: %v", err)
		}

		outDir := filepath.Join(tmpdir, "output2")
		cfg := map[string]any{
			"filename": constants.DefaultConfigFile,
		}

		err := CreateDirAndFiles(outDir, true, tplDir, cfg)
		if err != nil {
			t.Errorf("CreateDirAndFiles() error = %v", err)
		}

		// Check both files were created
		if _, err := os.Stat(filepath.Join(outDir, "file1.yml")); err != nil {
			t.Error("Expected file1.yml to be created")
		}
		if _, err := os.Stat(filepath.Join(outDir, "file2.yml")); err != nil {
			t.Error("Expected file2.yml to be created")
		}
	})

	t.Run("template with nested config access", func(t *testing.T) {
		tplFile := filepath.Join(tmpdir, "nested-template.yml")
		if err := os.WriteFile(tplFile, []byte("foo: {{ .services.client.foo }}\ntag: {{ .services.client.tag }}"), constants.StackFilePerm); err != nil {
			t.Fatalf("failed to write template: %v", err)
		}

		outDir := filepath.Join(tmpdir, "output3")
		cfg := map[string]any{
			"filename": "nested-result.yml",
			"services": map[string]any{
				"client": map[string]any{
					"foo": "bar",
					"tag": "latest",
				},
			},
		}

		err := CreateDirAndFiles(outDir, true, tplFile, cfg)
		if err != nil {
			t.Errorf("CreateDirAndFiles() error = %v", err)
		}

		result, err := os.ReadFile(filepath.Join(outDir, "nested-result.yml"))
		if err != nil {
			t.Fatalf("failed to read result: %v", err)
		}
		content := string(result)
		if !strings.Contains(content, "foo: bar") {
			t.Error("Expected nested foo value")
		}
		if !strings.Contains(content, "tag: latest") {
			t.Error("Expected nested tag value")
		}
	})

	t.Run("template with or operator fallback pattern", func(t *testing.T) {
		tplFile := filepath.Join(tmpdir, "or-template.yml")
		if err := os.WriteFile(tplFile, []byte(`tag: {{ or .services.auth.tag .defaults.tag }}
registry: {{ or .services.auth.containerRegistry .defaults.containerRegistry }}`), constants.StackFilePerm); err != nil {
			t.Fatalf("failed to write template: %v", err)
		}

		outDir := filepath.Join(tmpdir, "output4")
		cfg := map[string]any{
			"filename": "or-result.yml",
			"defaults": map[string]any{
				"tag":               "4.2.21",
				"containerRegistry": "registry.example.com",
			},
			"services": map[string]any{
				"auth": map[string]any{
					"tag": "4.2.0",
				},
			},
		}

		err := CreateDirAndFiles(outDir, true, tplFile, cfg)
		if err != nil {
			t.Errorf("CreateDirAndFiles() error = %v", err)
		}

		result, err := os.ReadFile(filepath.Join(outDir, "or-result.yml"))
		if err != nil {
			t.Fatalf("failed to read result: %v", err)
		}
		content := string(result)

		if !strings.Contains(content, "tag: 4.2.0") {
			t.Error("Expected tag from service config")
		}
		if !strings.Contains(content, "registry: registry.example.com") {
			t.Error("Expected registry from defaults")
		}
	})

	t.Run("template with direct or pattern - recommended approach", func(t *testing.T) {
		tplFile := filepath.Join(tmpdir, "or-pattern-template.yml")
		templateContent := `auth:
  image: {{ or .services.auth.containerRegistry .defaults.containerRegistry }}/openslides-auth:{{ or .services.auth.tag .defaults.tag }}
  replicas: {{ or .services.auth.replicas 1 }}

projector:
  image: {{ or .services.projector.containerRegistry .defaults.containerRegistry }}/openslides-projector:{{ or .services.projector.tag .defaults.tag }}
  replicas: {{ or .services.projector.replicas 1 }}`
		if err := os.WriteFile(tplFile, []byte(templateContent), constants.StackFilePerm); err != nil {
			t.Fatalf("failed to write template: %v", err)
		}

		outDir := filepath.Join(tmpdir, "output5")
		cfg := map[string]any{
			"filename": "or-pattern-result.yml",
			"defaults": map[string]any{
				"tag":               "4.2.21",
				"containerRegistry": "registry.example.com",
			},
			"services": map[string]any{
				"auth": map[string]any{
					"tag":      "4.2.0",
					"replicas": 3,
				},
			},
		}

		err := CreateDirAndFiles(outDir, true, tplFile, cfg)
		if err != nil {
			t.Errorf("CreateDirAndFiles() error = %v", err)
		}

		result, err := os.ReadFile(filepath.Join(outDir, "or-pattern-result.yml"))
		if err != nil {
			t.Fatalf("failed to read result: %v", err)
		}
		content := string(result)

		if !strings.Contains(content, ":4.2.0") {
			t.Error("Expected auth tag 4.2.0 from service config")
		}
		if !strings.Contains(content, "registry.example.com/openslides-auth") {
			t.Error("Expected auth registry from defaults")
		}
		if !strings.Contains(content, "replicas: 3") {
			t.Error("Expected auth replicas from service config")
		}

		if !strings.Contains(content, ":4.2.21") {
			t.Error("Expected projector tag 4.2.21 from defaults")
		}
		if !strings.Contains(content, "registry.example.com/openslides-projector") {
			t.Error("Expected projector registry from defaults")
		}
		if !strings.Contains(content, "replicas: 1") {
			t.Error("Expected projector default replicas")
		}
	})
}
