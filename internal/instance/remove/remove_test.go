package remove

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/OpenSlides/openslides-cli/internal/constants"
)

func TestRemoveInstance_DirectoryExists(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "remove-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	instanceDir := filepath.Join(tmpDir, "test-instance")
	if err := os.MkdirAll(instanceDir, constants.InstanceDirPerm); err != nil {
		t.Fatalf("Failed to create instance dir: %v", err)
	}

	secretsDir := filepath.Join(instanceDir, constants.SecretsDirName)
	if err := os.MkdirAll(secretsDir, constants.SecretsDirPerm); err != nil {
		t.Fatalf("Failed to create secrets dir: %v", err)
	}

	testFile := filepath.Join(secretsDir, "test_secret")
	if err := os.WriteFile(testFile, []byte("secret"), constants.SecretFilePerm); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	t.Cleanup(func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Errorf("Failed to remove temp dir %s: %v", tmpDir, err)
		}
	})

	err = removeInstance(instanceDir, true)
	if err != nil {
		t.Fatalf("removeInstance failed: %v", err)
	}

	if _, err := os.Stat(instanceDir); !os.IsNotExist(err) {
		t.Errorf("Instance directory still exists after removal")
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
	if err != nil && !strings.Contains(err.Error(), expectedMsg) {
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
	if err := os.WriteFile(testFile, []byte("test"), constants.StackFilePerm); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	err = removeInstance(testFile, true)
	if err == nil {
		t.Error("Expected error when removing a file instead of directory, got nil")
	}

	expectedMsg := "not a directory"
	if err != nil && !strings.Contains(err.Error(), expectedMsg) {
		t.Errorf("Error should mention it's not a directory, got: %v", err)
	}
}

func TestRemoveInstance_RemovesNestedStructure(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "remove-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	instanceDir := filepath.Join(tmpDir, "complex-instance")

	// Create directories with appropriate permissions
	dirs := map[string]os.FileMode{
		filepath.Join(instanceDir, constants.SecretsDirName): constants.SecretsDirPerm,
		filepath.Join(instanceDir, constants.StackDirName):   constants.StackDirPerm,
		filepath.Join(instanceDir, "data", "postgres"):       constants.InstanceDirPerm,
		filepath.Join(instanceDir, "data", "redis"):          constants.InstanceDirPerm,
	}

	for dir, perm := range dirs {
		if err := os.MkdirAll(dir, perm); err != nil {
			t.Fatalf("Failed to create dir %s: %v", dir, err)
		}
	}

	// Create files with appropriate permissions
	files := map[string]os.FileMode{
		filepath.Join(instanceDir, constants.SecretsDirName, constants.PgPasswordFile):   constants.SecretFilePerm,
		filepath.Join(instanceDir, constants.SecretsDirName, constants.AdminSecretsFile): constants.SecretFilePerm,
		filepath.Join(instanceDir, constants.StackDirName, "deployment.yaml"):            constants.StackFilePerm,
		filepath.Join(instanceDir, "data", "postgres", "pg_data.db"):                     constants.StackFilePerm,
		filepath.Join(instanceDir, "data", "redis", "dump.rdb"):                          constants.StackFilePerm,
	}

	for file, perm := range files {
		if err := os.WriteFile(file, []byte("test data"), perm); err != nil {
			t.Fatalf("Failed to create file %s: %v", file, err)
		}
	}

	t.Cleanup(func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Errorf("Failed to remove temp dir %s: %v", tmpDir, err)
		}
	})

	err = removeInstance(instanceDir, true)
	if err != nil {
		t.Fatalf("removeInstance failed: %v", err)
	}

	if _, err := os.Stat(instanceDir); !os.IsNotExist(err) {
		t.Error("Instance directory still exists after removal")
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

	instanceDir := filepath.Join(tmpDir, "test-instance")
	if err := os.MkdirAll(instanceDir, constants.InstanceDirPerm); err != nil {
		t.Fatalf("Failed to create instance dir: %v", err)
	}

	t.Cleanup(func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Errorf("Failed to remove temp dir %s: %v", tmpDir, err)
		}
	})

	err = removeInstance(instanceDir, true)
	if err != nil {
		t.Fatalf("removeInstance with force=true failed: %v", err)
	}

	if _, err := os.Stat(instanceDir); !os.IsNotExist(err) {
		t.Error("Instance directory still exists after removal")
	}
}

func TestRemoveInstance_EmptyDirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "remove-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	instanceDir := filepath.Join(tmpDir, "empty-instance")
	if err := os.MkdirAll(instanceDir, constants.InstanceDirPerm); err != nil {
		t.Fatalf("Failed to create instance dir: %v", err)
	}

	t.Cleanup(func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Errorf("Failed to remove temp dir %s: %v", tmpDir, err)
		}
	})

	err = removeInstance(instanceDir, true)
	if err != nil {
		t.Fatalf("removeInstance failed on empty directory: %v", err)
	}

	if _, err := os.Stat(instanceDir); !os.IsNotExist(err) {
		t.Error("Empty directory still exists after removal")
	}
}

func TestRemoveInstance_WithSymlinks(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "remove-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	instanceDir := filepath.Join(tmpDir, "test-instance")
	if err := os.MkdirAll(instanceDir, constants.InstanceDirPerm); err != nil {
		t.Fatalf("Failed to create instance dir: %v", err)
	}

	targetFile := filepath.Join(tmpDir, "target.txt")
	if err := os.WriteFile(targetFile, []byte("target"), constants.StackFilePerm); err != nil {
		t.Fatalf("Failed to create target file: %v", err)
	}

	symlinkPath := filepath.Join(instanceDir, "link")
	if err := os.Symlink(targetFile, symlinkPath); err != nil {
		t.Fatalf("Failed to create symlink: %v", err)
	}

	t.Cleanup(func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Errorf("Failed to remove temp dir %s: %v", tmpDir, err)
		}
	})

	err = removeInstance(instanceDir, true)
	if err != nil {
		t.Fatalf("removeInstance failed: %v", err)
	}

	if _, err := os.Stat(instanceDir); !os.IsNotExist(err) {
		t.Error("Instance directory still exists after removal")
	}

	if _, err := os.Stat(targetFile); os.IsNotExist(err) {
		t.Error("Symlink target should not be deleted")
	}
}
