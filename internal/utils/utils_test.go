package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadFromFileOrStdin(t *testing.T) {
	t.Run("read from file", func(t *testing.T) {
		tmpfile, err := os.CreateTemp("", "test-*.txt")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpfile.Name())

		content := "test content"
		if _, err := tmpfile.WriteString(content); err != nil {
			t.Fatal(err)
		}
		tmpfile.Close()

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
		defer os.Remove(tmpfile.Name())

		password := "secret123"
		tmpfile.WriteString(password)
		tmpfile.Close()

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
		err := CreateFile(tmpdir, false, "test.txt", content)
		if err != nil {
			t.Errorf("CreateFile() error = %v", err)
		}

		data, err := os.ReadFile(filepath.Join(tmpdir, "test.txt"))
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != string(content) {
			t.Errorf("File content = %s, want %s", string(data), string(content))
		}
	})

	t.Run("don't overwrite without force", func(t *testing.T) {
		filename := "existing.txt"
		original := []byte("original")
		CreateFile(tmpdir, false, filename, original)

		newContent := []byte("new content")
		CreateFile(tmpdir, false, filename, newContent)

		data, _ := os.ReadFile(filepath.Join(tmpdir, filename))
		if string(data) != string(original) {
			t.Error("File was overwritten without force flag")
		}
	})

	t.Run("overwrite with force", func(t *testing.T) {
		filename := "force.txt"
		original := []byte("original")
		CreateFile(tmpdir, true, filename, original)

		newContent := []byte("new content")
		CreateFile(tmpdir, true, filename, newContent)

		data, _ := os.ReadFile(filepath.Join(tmpdir, filename))
		if string(data) != string(newContent) {
			t.Error("File was not overwritten with force flag")
		}
	})
}
