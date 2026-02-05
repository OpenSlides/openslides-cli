package utils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/OpenSlides/openslides-cli/internal/constants"
)

func TestReadFromFileOrStdin(t *testing.T) {
	t.Run("read from file", func(t *testing.T) {
		tmpfile, err := os.CreateTemp("", "test-*.txt")
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if err := os.Remove(tmpfile.Name()); err != nil {
				t.Logf("warning: failed to remove temp file: %v", err)
			}
		})

		content := "test content"
		if _, err := tmpfile.WriteString(content); err != nil {
			t.Fatalf("failed to write test content: %v", err)
		}
		if err := tmpfile.Close(); err != nil {
			t.Fatalf("failed to close temp file: %v", err)
		}

		data, err := ReadFromFileOrStdin(tmpfile.Name())
		if err != nil {
			t.Errorf("ReadFromFileOrStdin() error = %v", err)
		}
		if string(data) != content {
			t.Errorf("ReadFromFileOrStdin() = %s, want %s", string(data), content)
		}
	})

	t.Run("file not found", func(t *testing.T) {
		_, err := ReadFromFileOrStdin("nonexistent-file.txt")
		if err == nil {
			t.Error("Expected error for nonexistent file")
		}
	})
}

func TestReadInputOrFileOrStdin(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		filename string
		wantErr  bool
	}{
		{"both empty", "", "", true},
		{"both provided", "input", "file", true},
		{"only input", "test input", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ReadInputOrFileOrStdin(tt.input, tt.filename)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadInputOrFileOrStdin() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

	t.Run("with input", func(t *testing.T) {
		input := "test data"
		data, err := ReadInputOrFileOrStdin(input, "")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if string(data) != input {
			t.Errorf("Got %s, want %s", string(data), input)
		}
	})
}

func TestReadPassword(t *testing.T) {
	t.Run("valid password file", func(t *testing.T) {
		tmpfile, err := os.CreateTemp("", "password-*.txt")
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if err := os.Remove(tmpfile.Name()); err != nil {
				t.Logf("warning: failed to remove temp file: %v", err)
			}
		})

		password := "secret123"
		if _, err := tmpfile.WriteString(password); err != nil {
			t.Fatalf("failed to write password: %v", err)
		}
		if err := tmpfile.Close(); err != nil {
			t.Fatalf("failed to close temp file: %v", err)
		}

		result, err := ReadPassword(tmpfile.Name())
		if err != nil {
			t.Errorf("ReadPassword() error = %v", err)
		}
		if result != password {
			t.Errorf("ReadPassword() = %s, want %s", result, password)
		}
	})

	t.Run("file not found", func(t *testing.T) {
		_, err := ReadPassword("nonexistent-password.txt")
		if err == nil {
			t.Error("Expected error for nonexistent file")
		}
	})
}

func TestCreateFile(t *testing.T) {
	tmpdir := t.TempDir()

	t.Run("create new file", func(t *testing.T) {
		content := []byte("test content")
		err := CreateFile(tmpdir, false, "test.txt", content, constants.StackFilePerm)
		if err != nil {
			t.Errorf("CreateFile() error = %v", err)
		}

		data, err := os.ReadFile(filepath.Join(tmpdir, "test.txt"))
		if err != nil {
			t.Fatalf("failed to read created file: %v", err)
		}
		if string(data) != string(content) {
			t.Errorf("File content = %s, want %s", string(data), string(content))
		}

		// Verify permissions
		fileInfo, err := os.Stat(filepath.Join(tmpdir, "test.txt"))
		if err != nil {
			t.Fatalf("failed to stat file: %v", err)
		}
		if fileInfo.Mode().Perm() != constants.StackFilePerm {
			t.Errorf("File permissions = %v, want %v", fileInfo.Mode().Perm(), constants.StackFilePerm)
		}
	})

	t.Run("don't overwrite without force", func(t *testing.T) {
		filename := "existing.txt"
		original := []byte("original")
		if err := CreateFile(tmpdir, false, filename, original, constants.StackFilePerm); err != nil {
			t.Fatalf("failed to create initial file: %v", err)
		}

		newContent := []byte("new content")
		if err := CreateFile(tmpdir, false, filename, newContent, constants.StackFilePerm); err != nil {
			t.Fatalf("CreateFile() error = %v", err)
		}

		data, err := os.ReadFile(filepath.Join(tmpdir, filename))
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}
		if string(data) != string(original) {
			t.Error("File was overwritten without force flag")
		}
	})

	t.Run("overwrite with force", func(t *testing.T) {
		filename := "force.txt"
		original := []byte("original")
		if err := CreateFile(tmpdir, true, filename, original, constants.StackFilePerm); err != nil {
			t.Fatalf("failed to create initial file: %v", err)
		}

		newContent := []byte("new content")
		if err := CreateFile(tmpdir, true, filename, newContent, constants.StackFilePerm); err != nil {
			t.Fatalf("CreateFile() error = %v", err)
		}

		data, err := os.ReadFile(filepath.Join(tmpdir, filename))
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}
		if string(data) != string(newContent) {
			t.Error("File was not overwritten with force flag")
		}
	})

	t.Run("create secret file with secret permissions", func(t *testing.T) {
		filename := "secret.txt"
		content := []byte("super secret")
		err := CreateFile(tmpdir, false, filename, content, constants.SecretFilePerm)
		if err != nil {
			t.Errorf("CreateFile() error = %v", err)
		}

		// Verify permissions
		fileInfo, err := os.Stat(filepath.Join(tmpdir, filename))
		if err != nil {
			t.Fatalf("failed to stat file: %v", err)
		}
		if fileInfo.Mode().Perm() != constants.SecretFilePerm {
			t.Errorf("Secret file permissions = %v, want %v", fileInfo.Mode().Perm(), constants.SecretFilePerm)
		}
	})

	t.Run("different permissions for different file types", func(t *testing.T) {
		// Create a manifest file with stack permissions
		manifestFile := "deployment.yaml"
		if err := CreateFile(tmpdir, false, manifestFile, []byte("manifest"), constants.StackFilePerm); err != nil {
			t.Fatalf("failed to create manifest file: %v", err)
		}

		// Create a secret file with secret permissions
		secretFile := "password"
		if err := CreateFile(tmpdir, false, secretFile, []byte("secret"), constants.SecretFilePerm); err != nil {
			t.Fatalf("failed to create secret file: %v", err)
		}

		// Verify manifest permissions (0644)
		manifestInfo, err := os.Stat(filepath.Join(tmpdir, manifestFile))
		if err != nil {
			t.Fatalf("failed to stat manifest: %v", err)
		}
		if manifestInfo.Mode().Perm() != constants.StackFilePerm {
			t.Errorf("Manifest permissions = %v, want %v", manifestInfo.Mode().Perm(), constants.StackFilePerm)
		}

		// Verify secret permissions (0600)
		secretInfo, err := os.Stat(filepath.Join(tmpdir, secretFile))
		if err != nil {
			t.Fatalf("failed to stat secret: %v", err)
		}
		if secretInfo.Mode().Perm() != constants.SecretFilePerm {
			t.Errorf("Secret permissions = %v, want %v", secretInfo.Mode().Perm(), constants.SecretFilePerm)
		}
	})
}
