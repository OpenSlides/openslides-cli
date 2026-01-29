package create

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSecureSecretsDirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "create-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Errorf("Failed to remove temp dir %s: %v", tmpDir, err)
		}
	})

	// Create a secrets directory with some files
	secretsDir := filepath.Join(tmpDir, "secrets")
	if err := os.MkdirAll(secretsDir, 0755); err != nil {
		t.Fatalf("Failed to create secrets dir: %v", err)
	}

	// Create test secret files with open permissions
	testFiles := []string{"secret1", "secret2", "secret3"}
	for _, filename := range testFiles {
		path := filepath.Join(secretsDir, filename)
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}

	// Secure the directory
	err = secureSecretsDirectory(secretsDir)
	if err != nil {
		t.Fatalf("secureSecretsDirectory failed: %v", err)
	}

	// Verify directory permissions (700)
	dirInfo, err := os.Stat(secretsDir)
	if err != nil {
		t.Fatalf("Failed to stat secrets directory: %v", err)
	}

	expectedDirPerms := os.FileMode(secretDirPerm)
	if dirInfo.Mode().Perm() != expectedDirPerms {
		t.Errorf("Directory permissions = %v, want %v", dirInfo.Mode().Perm(), expectedDirPerms)
	}

	// Verify all file permissions (600)
	expectedFilePerms := os.FileMode(secretFilePerm)
	for _, filename := range testFiles {
		path := filepath.Join(secretsDir, filename)
		fileInfo, err := os.Stat(path)
		if err != nil {
			t.Fatalf("Failed to stat file %s: %v", filename, err)
		}

		if fileInfo.Mode().Perm() != expectedFilePerms {
			t.Errorf("File %s permissions = %v, want %v", filename, fileInfo.Mode().Perm(), expectedFilePerms)
		}
	}
}

func TestSecureSecretsDirectory_SkipsSubdirectories(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "create-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Errorf("Failed to remove temp dir %s: %v", tmpDir, err)
		}
	})

	// Create secrets directory
	secretsDir := filepath.Join(tmpDir, "secrets")
	if err := os.MkdirAll(secretsDir, 0755); err != nil {
		t.Fatalf("Failed to create secrets dir: %v", err)
	}

	// Create a subdirectory within secrets
	subDir := filepath.Join(secretsDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	// Create a file
	testFile := filepath.Join(secretsDir, "secret1")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Secure the directory (should skip subdirectory)
	err = secureSecretsDirectory(secretsDir)
	if err != nil {
		t.Fatalf("secureSecretsDirectory failed: %v", err)
	}

	// Verify subdirectory permissions were NOT changed
	subDirInfo, err := os.Stat(subDir)
	if err != nil {
		t.Fatalf("Failed to stat subdirectory: %v", err)
	}

	// Should still have original 0755 permissions
	if subDirInfo.Mode().Perm() == os.FileMode(secretFilePerm) {
		t.Error("Subdirectory permissions should not be changed to secretFilePerm")
	}

	// Verify file permissions WERE changed
	fileInfo, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}

	expectedPerms := os.FileMode(secretFilePerm)
	if fileInfo.Mode().Perm() != expectedPerms {
		t.Errorf("File permissions = %v, want %v", fileInfo.Mode().Perm(), expectedPerms)
	}
}

func TestCreateInstance(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "create-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Errorf("Failed to remove temp dir %s: %v", tmpDir, err)
		}
	})

	// Create secrets directory
	secretsDir := filepath.Join(tmpDir, "secrets")
	if err := os.MkdirAll(secretsDir, 0755); err != nil {
		t.Fatalf("Failed to create secrets dir: %v", err)
	}

	// Create some existing secret files (simulating 'setup' output)
	existingSecrets := map[string]string{
		"postgres_password":      "old-db-password",
		"superadmin":             "old-admin-password",
		"internal_auth_password": "some-auth-key",
	}

	for filename, content := range existingSecrets {
		path := filepath.Join(secretsDir, filename)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create existing secret %s: %v", filename, err)
		}
	}

	// Run createInstance
	dbPassword := "new-database-password"
	superadminPassword := "new-superadmin-password"

	err = createInstance(tmpDir, dbPassword, superadminPassword)
	if err != nil {
		t.Fatalf("createInstance failed: %v", err)
	}

	// Verify postgres_password was overwritten
	pgContent, err := os.ReadFile(filepath.Join(secretsDir, pgPasswordFile))
	if err != nil {
		t.Fatalf("Failed to read postgres_password: %v", err)
	}
	if string(pgContent) != dbPassword {
		t.Errorf("postgres_password = %q, want %q", string(pgContent), dbPassword)
	}

	// Verify superadmin was overwritten
	adminContent, err := os.ReadFile(filepath.Join(secretsDir, adminSecretsFile))
	if err != nil {
		t.Fatalf("Failed to read superadmin: %v", err)
	}
	if string(adminContent) != superadminPassword {
		t.Errorf("superadmin = %q, want %q", string(adminContent), superadminPassword)
	}

	// Verify other secrets were not touched
	authContent, err := os.ReadFile(filepath.Join(secretsDir, "internal_auth_password"))
	if err != nil {
		t.Fatalf("Failed to read internal_auth_password: %v", err)
	}
	if string(authContent) != existingSecrets["internal_auth_password"] {
		t.Errorf("internal_auth_password was unexpectedly changed")
	}

	// Verify all files have secretFilePerm permissions
	entries, err := os.ReadDir(secretsDir)
	if err != nil {
		t.Fatalf("Failed to read secrets directory: %v", err)
	}

	expectedPerms := os.FileMode(secretFilePerm)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		path := filepath.Join(secretsDir, entry.Name())
		fileInfo, err := os.Stat(path)
		if err != nil {
			t.Fatalf("Failed to stat %s: %v", entry.Name(), err)
		}

		if fileInfo.Mode().Perm() != expectedPerms {
			t.Errorf("File %s permissions = %v, want %v", entry.Name(), fileInfo.Mode().Perm(), expectedPerms)
		}
	}

	// Verify directory has secretDirPerm permissions
	dirInfo, err := os.Stat(secretsDir)
	if err != nil {
		t.Fatalf("Failed to stat secrets directory: %v", err)
	}

	expectedDirPerms := os.FileMode(secretDirPerm)
	if dirInfo.Mode().Perm() != expectedDirPerms {
		t.Errorf("Directory permissions = %v, want %v", dirInfo.Mode().Perm(), expectedDirPerms)
	}
}

func TestCreateInstance_SecretsDirectoryNotExist(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "create-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Errorf("Failed to remove temp dir %s: %v", tmpDir, err)
		}
	})

	// Don't create secrets directory - should fail
	err = createInstance(tmpDir, "password", "admin")
	if err == nil {
		t.Error("Expected error when secrets directory doesn't exist, got nil")
	}

	// Error message should mention running 'setup' first
	expectedMsg := "run 'setup' first"
	if err != nil && !strings.Contains(err.Error(), expectedMsg) {
		t.Errorf("Error should mention running 'setup', got: %v", err)
	}
}
