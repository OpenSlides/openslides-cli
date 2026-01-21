package actions

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRemoveInstance_DirectoryExists(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "remove-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	projectDir := filepath.Join(tmpDir, "test-instance")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("Failed to create project dir: %v", err)
	}

	secretsDir := filepath.Join(projectDir, "secrets")
	if err := os.MkdirAll(secretsDir, 0755); err != nil {
		t.Fatalf("Failed to create secrets dir: %v", err)
	}

	testFile := filepath.Join(secretsDir, "test_secret")
	if err := os.WriteFile(testFile, []byte("secret"), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	t.Cleanup(func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Errorf("Failed to remove temp dir %s: %v", tmpDir, err)
		}
	})

	err = removeInstance(projectDir, true)
	if err != nil {
		t.Fatalf("removeInstance failed: %v", err)
	}

	if _, err := os.Stat(projectDir); !os.IsNotExist(err) {
		t.Errorf("Project directory still exists after removal")
	}
}

func TestRemoveInstance_DirectoryNotExist(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "remove-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Errorf("Failed to remove temp dir %s: %v", tmpDir, err)
		}
	})

	nonExistentDir := filepath.Join(tmpDir, "does-not-exist")

	err = removeInstance(nonExistentDir, true)
	if err == nil {
		t.Error("Expected error when removing non-existent directory, got nil")
	}

	expectedMsg := "does not exist"
	if err != nil && !contains(err.Error(), expectedMsg) {
		t.Errorf("Error should mention directory doesn't exist, got: %v", err)
	}
}

func TestRemoveInstance_NotADirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "remove-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Errorf("Failed to remove temp dir %s: %v", tmpDir, err)
		}
	})

	testFile := filepath.Join(tmpDir, "test-file")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	err = removeInstance(testFile, true)
	if err == nil {
		t.Error("Expected error when removing a file instead of directory, got nil")
	}

	expectedMsg := "not a directory"
	if err != nil && !contains(err.Error(), expectedMsg) {
		t.Errorf("Error should mention it's not a directory, got: %v", err)
	}
}

func TestRemoveInstance_RemovesNestedStructure(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "remove-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	projectDir := filepath.Join(tmpDir, "complex-instance")

	dirs := []string{
		filepath.Join(projectDir, "secrets"),
		filepath.Join(projectDir, "config"),
		filepath.Join(projectDir, "data", "postgres"),
		filepath.Join(projectDir, "data", "redis"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create dir %s: %v", dir, err)
		}
	}

	files := []string{
		filepath.Join(projectDir, "secrets", "postgres_password"),
		filepath.Join(projectDir, "secrets", "superadmin"),
		filepath.Join(projectDir, "config", "docker-compose.yml"),
		filepath.Join(projectDir, "data", "postgres", "pg_data.db"),
		filepath.Join(projectDir, "data", "redis", "dump.rdb"),
	}

	for _, file := range files {
		if err := os.WriteFile(file, []byte("test data"), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", file, err)
		}
	}

	t.Cleanup(func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Errorf("Failed to remove temp dir %s: %v", tmpDir, err)
		}
	})

	err = removeInstance(projectDir, true)
	if err != nil {
		t.Fatalf("removeInstance failed: %v", err)
	}

	if _, err := os.Stat(projectDir); !os.IsNotExist(err) {
		t.Error("Project directory still exists after removal")
	}

	if _, err := os.Stat(tmpDir); os.IsNotExist(err) {
		t.Error("Parent directory should still exist")
	}
}

func TestRemoveInstance_WithForceFlag(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "remove-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	projectDir := filepath.Join(tmpDir, "test-instance")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("Failed to create project dir: %v", err)
	}

	t.Cleanup(func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Errorf("Failed to remove temp dir %s: %v", tmpDir, err)
		}
	})

	err = removeInstance(projectDir, true)
	if err != nil {
		t.Fatalf("removeInstance with force=true failed: %v", err)
	}

	if _, err := os.Stat(projectDir); !os.IsNotExist(err) {
		t.Error("Project directory still exists after removal")
	}
}

func TestRemoveInstance_EmptyDirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "remove-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	projectDir := filepath.Join(tmpDir, "empty-instance")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("Failed to create project dir: %v", err)
	}

	t.Cleanup(func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Errorf("Failed to remove temp dir %s: %v", tmpDir, err)
		}
	})

	err = removeInstance(projectDir, true)
	if err != nil {
		t.Fatalf("removeInstance failed on empty directory: %v", err)
	}

	if _, err := os.Stat(projectDir); !os.IsNotExist(err) {
		t.Error("Empty directory still exists after removal")
	}
}

func TestRemoveInstance_WithSymlinks(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "remove-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	projectDir := filepath.Join(tmpDir, "test-instance")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("Failed to create project dir: %v", err)
	}

	targetFile := filepath.Join(tmpDir, "target.txt")
	if err := os.WriteFile(targetFile, []byte("target"), 0644); err != nil {
		t.Fatalf("Failed to create target file: %v", err)
	}

	symlinkPath := filepath.Join(projectDir, "link")
	if err := os.Symlink(targetFile, symlinkPath); err != nil {
		t.Fatalf("Failed to create symlink: %v", err)
	}

	t.Cleanup(func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Errorf("Failed to remove temp dir %s: %v", tmpDir, err)
		}
	})

	err = removeInstance(projectDir, true)
	if err != nil {
		t.Fatalf("removeInstance failed: %v", err)
	}

	if _, err := os.Stat(projectDir); !os.IsNotExist(err) {
		t.Error("Project directory still exists after removal")
	}

	if _, err := os.Stat(targetFile); os.IsNotExist(err) {
		t.Error("Symlink target should not be deleted")
	}
}
