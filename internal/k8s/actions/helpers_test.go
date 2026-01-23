package actions

import (
	"os"
	"testing"
)

func TestExtractNamespace(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple directory",
			input:    "my-instance",
			expected: "my-instance",
		},
		{
			name:     "directory with dots",
			input:    "my.instance",
			expected: "myinstance",
		},
		{
			name:     "full path with dots",
			input:    "/home/user/projects/my.instance",
			expected: "myinstance",
		},
		{
			name:     "nested path without dots",
			input:    "/var/lib/openslides/prod-instance",
			expected: "prod-instance",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractNamespace(tt.input)
			if result != tt.expected {
				t.Errorf("extractNamespace(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFileExists(t *testing.T) {
	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "test-file-*")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "existing file",
			path:     tmpFile.Name(),
			expected: true,
		},
		{
			name:     "non-existing file",
			path:     "/tmp/definitely-does-not-exist-12345",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fileExists(tt.path)
			if result != tt.expected {
				t.Errorf("fileExists(%q) = %v, want %v", tt.path, result, tt.expected)
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
		{
			name:     "yaml extension",
			filename: "deployment.yaml",
			expected: true,
		},
		{
			name:     "yml extension",
			filename: "service.yml",
			expected: true,
		},
		{
			name:     "json file",
			filename: "config.json",
			expected: false,
		},
		{
			name:     "no extension",
			filename: "Makefile",
			expected: false,
		},
		{
			name:     "yaml in path but not extension",
			filename: "/path/yaml/file.txt",
			expected: false,
		},
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
