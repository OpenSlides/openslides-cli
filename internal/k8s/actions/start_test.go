package actions

import (
	"os"
	"path/filepath"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
)

func TestFileExists(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
		setup    func(t *testing.T) string
		cleanup  func(path string)
	}{
		{
			name:     "file exists",
			expected: true,
			setup: func(t *testing.T) string {
				tmpFile := filepath.Join(t.TempDir(), "test.yaml")
				if err := os.WriteFile(tmpFile, []byte("test"), 0644); err != nil {
					t.Fatal(err)
				}
				return tmpFile
			},
		},
		{
			name:     "file does not exist",
			path:     "/nonexistent/path/file.yaml",
			expected: false,
		},
		{
			name:     "empty path",
			path:     "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.path
			if tt.setup != nil {
				path = tt.setup(t)
			}

			result := fileExists(path)
			if result != tt.expected {
				t.Errorf("fileExists(%q) = %v, want %v", path, result, tt.expected)
			}
		})
	}
}

func TestIsYAMLFile(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected bool
	}{
		{"yaml extension", "deployment.yaml", true},
		{"yml extension", "service.yml", true},
		{"json file", "config.json", false},
		{"txt file", "readme.txt", false},
		{"no extension", "Dockerfile", false},
		{"multiple dots yaml", "my.config.yaml", true},
		{"multiple dots yml", "my.config.yml", true},
		{"uppercase YAML", "file.YAML", false}, // filepath.Ext is case-sensitive
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isYAMLFile(tt.filename)
			if result != tt.expected {
				t.Errorf("isYAMLFile(%q) = %v, want %v", tt.filename, result, tt.expected)
			}
		})
	}
}

func TestApplyDirectory_FileFiltering(t *testing.T) {
	// Create temp directory with mixed files
	tmpDir := t.TempDir()

	// Create test files
	files := map[string]string{
		"deployment.yaml":  "apiVersion: apps/v1\nkind: Deployment",
		"service.yml":      "apiVersion: v1\nkind: Service",
		"config.json":      `{"key": "value"}`,
		"README.md":        "# Documentation",
		"script.sh":        "#!/bin/bash",
		".hidden.yaml":     "apiVersion: v1\nkind: Secret",
		"nested/deep.yaml": "apiVersion: v1\nkind: ConfigMap",
	}

	yamlCount := 0
	for filename, content := range files {
		path := filepath.Join(tmpDir, filename)

		// Create subdirectory if needed
		dir := filepath.Dir(path)
		if dir != tmpDir {
			if err := os.MkdirAll(dir, 0755); err != nil {
				t.Fatal(err)
			}
		}

		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		// Count YAML files
		if isYAMLFile(filename) {
			yamlCount++
		}
	}

	// Read directory and verify filtering logic
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	filteredCount := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if isYAMLFile(entry.Name()) {
			filteredCount++
		}
	}

	// We expect 3 YAML files at top level: deployment.yaml, service.yml, .hidden.yaml
	expectedYAMLFiles := 3
	if filteredCount != expectedYAMLFiles {
		t.Errorf("Expected %d YAML files, found %d", expectedYAMLFiles, filteredCount)
	}
}

func TestApplyManifest_NamespaceExtraction(t *testing.T) {
	tests := []struct {
		name              string
		manifestContent   string
		expectedNamespace string
		expectError       bool
	}{
		{
			name: "namespace resource",
			manifestContent: `apiVersion: v1
kind: Namespace
metadata:
  name: test-namespace`,
			expectedNamespace: "test-namespace",
			expectError:       false,
		},
		{
			name: "namespaced resource with namespace",
			manifestContent: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-deployment
  namespace: my-namespace`,
			expectedNamespace: "my-namespace",
			expectError:       false,
		},
		{
			name: "namespaced resource without namespace",
			manifestContent: `apiVersion: v1
kind: Service
metadata:
  name: test-service`,
			expectedNamespace: "",
			expectError:       false,
		},
		{
			name:              "invalid yaml",
			manifestContent:   "not: valid: yaml: content:",
			expectedNamespace: "",
			expectError:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse YAML
			var obj unstructured.Unstructured
			err := yaml.Unmarshal([]byte(tt.manifestContent), &obj)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Extract namespace
			namespace := obj.GetNamespace()
			if namespace == "" && obj.GetKind() == "Namespace" {
				namespace = obj.GetName()
			}

			if namespace != tt.expectedNamespace {
				t.Errorf("Expected namespace %q, got %q", tt.expectedNamespace, namespace)
			}
		})
	}
}

func TestTLSSecretPath(t *testing.T) {
	// Verify the constant matches expected path
	expected := "secrets/tls-letsencrypt-secret.yaml"
	if tlsCertSecretYAML != expected {
		t.Errorf("tlsCertSecretYAML = %q, want %q", tlsCertSecretYAML, expected)
	}

	// Test path construction
	projectDir := "/path/to/project"
	fullPath := filepath.Join(projectDir, tlsCertSecretYAML)
	expectedPath := "/path/to/project/secrets/tls-letsencrypt-secret.yaml"

	if fullPath != expectedPath {
		t.Errorf("Path construction failed: got %q, want %q", fullPath, expectedPath)
	}
}

func TestStartCmd_Flags(t *testing.T) {
	cmd := StartCmd()

	// Verify command exists
	if cmd == nil {
		t.Fatal("StartCmd() returned nil")
	}

	// Verify command name
	if cmd.Use != "start <project-dir>" {
		t.Errorf("Command use = %q, want %q", cmd.Use, "start <project-dir>")
	}

	// Verify flags exist
	tests := []struct {
		flagName     string
		defaultValue string
	}{
		{"kubeconfig", ""},
		{"skip-ready-check", "false"},
		{"timeout", "5m0s"},
	}

	for _, tt := range tests {
		t.Run(tt.flagName, func(t *testing.T) {
			flag := cmd.Flags().Lookup(tt.flagName)
			if flag == nil {
				t.Fatalf("Flag %q not found", tt.flagName)
			}

			if flag.DefValue != tt.defaultValue {
				t.Errorf("Flag %q default = %q, want %q", tt.flagName, flag.DefValue, tt.defaultValue)
			}
		})
	}
}

func TestStartCmd_Args(t *testing.T) {
	cmd := StartCmd()

	// Test with no args (should fail)
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	if err == nil {
		t.Error("Expected error with no args, got nil")
	}

	// Test with correct number of args (should pass validation, may fail execution)
	cmd.SetArgs([]string{"./test-dir"})
	// We don't execute because it would try to connect to k8s
	// Just verify args are accepted
	if err := cmd.Args(cmd, []string{"./test-dir"}); err != nil {
		t.Errorf("Args validation failed with valid args: %v", err)
	}

	// Test with too many args (should fail)
	if err := cmd.Args(cmd, []string{"./test-dir", "extra"}); err == nil {
		t.Error("Expected error with too many args, got nil")
	}
}
