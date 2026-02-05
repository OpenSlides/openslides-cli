package setup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/OpenSlides/openslides-cli/internal/constants"
)

func TestRandomSecret(t *testing.T) {
	secret1, err := randomSecret()
	if err != nil {
		t.Errorf("randomSecret() error = %v", err)
	}
	if len(secret1) == 0 {
		t.Error("Expected non-empty secret")
	}

	secret2, err := randomSecret()
	if err != nil {
		t.Errorf("randomSecret() error = %v", err)
	}

	// Secrets should be different
	if string(secret1) == string(secret2) {
		t.Error("Expected different random secrets")
	}

	// Should be base64 encoded (44 chars for 32 bytes)
	if len(secret1) != 44 {
		t.Errorf("Expected 44 char base64 string, got %d", len(secret1))
	}

	// Should end with padding character
	if secret1[len(secret1)-1] != '=' {
		t.Error("Expected this base64 string to end with padding '='")
	}
}

func TestRandomString(t *testing.T) {
	t.Run("generates correct length", func(t *testing.T) {
		lengths := []int{16, 32, 64, 128}
		for _, length := range lengths {
			str, err := randomString(length)
			if err != nil {
				t.Errorf("randomString(%d) error = %v", length, err)
			}
			if len(str) != length {
				t.Errorf("Expected length %d, got %d", length, len(str))
			}
		}
	})

	t.Run("generates unique strings", func(t *testing.T) {
		str1, err := randomString(32)
		if err != nil {
			t.Errorf("randomString() error = %v", err)
		}
		str2, err := randomString(32)
		if err != nil {
			t.Errorf("randomString() error = %v", err)
		}

		if string(str1) == string(str2) {
			t.Error("Expected different random strings")
		}
	})

	t.Run("contains only allowed characters", func(t *testing.T) {
		allowedChars := constants.PasswordCharset

		str, err := randomString(100)
		if err != nil {
			t.Errorf("randomString() error = %v", err)
		}

		for i, ch := range str {
			if !strings.ContainsRune(allowedChars, rune(ch)) {
				t.Errorf("Character at position %d (%c) is not in allowed charset", i, ch)
			}
		}
	})

	t.Run("invalid length", func(t *testing.T) {
		invalidLengths := []int{0, -1, -10}
		for _, length := range invalidLengths {
			_, err := randomString(length)
			if err == nil {
				t.Errorf("Expected error for length %d", length)
			}
		}
	})

	t.Run("suitable for postgres password", func(t *testing.T) {
		// Test that generated strings work as postgres passwords
		// Postgres passwords can contain most characters except null bytes
		str, err := randomString(100)
		if err != nil {
			t.Errorf("randomString() error = %v", err)
		}

		// Check no null bytes
		for i, ch := range str {
			if ch == 0 {
				t.Errorf("Found null byte at position %d", i)
			}
		}

		// Check it's printable ASCII range plus allowed special chars
		for i, ch := range str {
			if ch < 33 || ch > 126 {
				t.Errorf("Character at position %d (%d) outside printable ASCII range", i, ch)
			}
		}
	})
}

func TestCreateSecrets(t *testing.T) {
	tmpdir := t.TempDir()

	specs := []SecretSpec{
		{"test_secret1", func() ([]byte, error) { return []byte("secret1"), nil }},
		{"test_secret2", func() ([]byte, error) { return []byte("secret2"), nil }},
	}

	err := createSecrets(tmpdir, false, specs)
	if err != nil {
		t.Errorf("createSecrets() error = %v", err)
	}

	// Check files were created
	data1, err := os.ReadFile(filepath.Join(tmpdir, "test_secret1"))
	if err != nil {
		t.Error("Expected test_secret1 to be created")
	}
	if string(data1) != "secret1" {
		t.Errorf("Expected 'secret1', got %s", string(data1))
	}

	data2, err := os.ReadFile(filepath.Join(tmpdir, "test_secret2"))
	if err != nil {
		t.Error("Expected test_secret2 to be created")
	}
	if string(data2) != "secret2" {
		t.Errorf("Expected 'secret2', got %s", string(data2))
	}

	// Verify permissions
	info1, err := os.Stat(filepath.Join(tmpdir, "test_secret1"))
	if err != nil {
		t.Fatalf("failed to stat secret file: %v", err)
	}
	if info1.Mode().Perm() != constants.SecretFilePerm {
		t.Errorf("Secret file permissions = %v, want %v", info1.Mode().Perm(), constants.SecretFilePerm)
	}
}

func TestCreateSecrets_NoOverwrite(t *testing.T) {
	tmpdir := t.TempDir()

	// Create initial secret
	if err := os.WriteFile(filepath.Join(tmpdir, "existing"), []byte("original"), constants.SecretFilePerm); err != nil {
		t.Fatalf("failed to write initial secret: %v", err)
	}

	specs := []SecretSpec{
		{"existing", func() ([]byte, error) { return []byte("new"), nil }},
	}

	// Without force, should not overwrite
	err := createSecrets(tmpdir, false, specs)
	if err != nil {
		t.Errorf("createSecrets() error = %v", err)
	}

	data, _ := os.ReadFile(filepath.Join(tmpdir, "existing"))
	if string(data) != "original" {
		t.Error("Secret was overwritten without force flag")
	}

	// With force, should overwrite
	err = createSecrets(tmpdir, true, specs)
	if err != nil {
		t.Errorf("createSecrets() error = %v", err)
	}

	data, _ = os.ReadFile(filepath.Join(tmpdir, "existing"))
	if string(data) != "new" {
		t.Error("Secret was not overwritten with force flag")
	}
}

func TestCreateCerts(t *testing.T) {
	tmpdir := t.TempDir()

	err := createCerts(tmpdir, false)
	if err != nil {
		t.Errorf("createCerts() error = %v", err)
	}

	// Check cert file
	certPath := filepath.Join(tmpdir, constants.CertCertName)
	certData, err := os.ReadFile(certPath)
	if err != nil {
		t.Error("Expected cert_crt to be created")
	}
	if !strings.Contains(string(certData), "BEGIN CERTIFICATE") {
		t.Error("Expected PEM encoded certificate")
	}

	// Verify cert permissions
	certInfo, err := os.Stat(certPath)
	if err != nil {
		t.Fatalf("failed to stat cert file: %v", err)
	}
	if certInfo.Mode().Perm() != constants.SecretFilePerm {
		t.Errorf("Cert file permissions = %v, want %v", certInfo.Mode().Perm(), constants.SecretFilePerm)
	}

	// Check key file
	keyPath := filepath.Join(tmpdir, constants.CertKeyName)
	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		t.Error("Expected cert_key to be created")
	}
	if !strings.Contains(string(keyData), "BEGIN PRIVATE KEY") {
		t.Error("Expected PEM encoded private key")
	}

	// Verify key permissions
	keyInfo, err := os.Stat(keyPath)
	if err != nil {
		t.Fatalf("failed to stat key file: %v", err)
	}
	if keyInfo.Mode().Perm() != constants.SecretFilePerm {
		t.Errorf("Key file permissions = %v, want %v", keyInfo.Mode().Perm(), constants.SecretFilePerm)
	}
}

func TestDefaultSecrets(t *testing.T) {
	// Verify default secrets are defined with correct names from constants
	expectedSecrets := []string{
		constants.AuthTokenKey,
		constants.AuthCookieKey,
		constants.InternalAuthPassword,
		constants.PgPasswordFile,
		constants.AdminSecretsFile,
	}

	if len(defaultSecrets) != len(expectedSecrets) {
		t.Errorf("Expected %d default secrets, got %d", len(expectedSecrets), len(defaultSecrets))
	}

	for i, expected := range expectedSecrets {
		if defaultSecrets[i].Name != expected {
			t.Errorf("Expected secret %s at position %d, got %s", expected, i, defaultSecrets[i].Name)
		}
	}

	// Test that postgres_password generates proper string
	for _, spec := range defaultSecrets {
		if spec.Name == constants.PgPasswordFile {
			pwd, err := spec.Generator()
			if err != nil {
				t.Errorf("postgres_password generator error = %v", err)
			}
			if len(pwd) != constants.DefaultPostgresPasswordLength {
				t.Errorf("Expected length %d, got %d", constants.DefaultPostgresPasswordLength, len(pwd))
			}
		}
	}

	// Test superadmin password generator
	for _, spec := range defaultSecrets {
		if spec.Name == constants.AdminSecretsFile {
			pwd, err := spec.Generator()
			if err != nil {
				t.Errorf("Superadmin generator error = %v", err)
			}
			if len(pwd) != constants.DefaultSuperadminPasswordLength {
				t.Errorf("Expected length %d, got %d", constants.DefaultSuperadminPasswordLength, len(pwd))
			}
		}
	}

	// Test that base64-encoded secrets still work as expected
	for _, spec := range defaultSecrets {
		if spec.Name == constants.AuthTokenKey || spec.Name == constants.AuthCookieKey || spec.Name == constants.InternalAuthPassword {
			secret, err := spec.Generator()
			if err != nil {
				t.Errorf("%s generator error = %v", spec.Name, err)
			}
			// These should be base64 encoded (44 chars)
			if len(secret) != 44 {
				t.Errorf("Expected %s to be 44 chars (base64), got %d", spec.Name, len(secret))
			}
		}
	}
}

// Integration-style test (without actual backend)
func TestSetupFlow(t *testing.T) {
	tmpdir := t.TempDir()

	// Create a minimal config file with camelCase keys
	configFile := filepath.Join(tmpdir, "config.yml")
	configContent := `---
url: test.example.com
stackName: test-stack
filename: docker-compose.yml
host: 0.0.0.0
port: 8000
disablePostgres: false
disableDependsOn: false
enableLocalHTTPS: false
defaults:
  containerRegistry: registry.example.com
  tag: 4.2.21
`
	if err := os.WriteFile(configFile, []byte(configContent), constants.StackFilePerm); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Create a minimal template using camelCase config keys
	templateFile := filepath.Join(tmpdir, "template.yml")
	templateContent := `---
# Test template with camelCase config keys
url: {{ .url }}
stackName: {{ .stackName }}
host: {{ .host }}
port: {{ .port }}
disablePostgres: {{ .disablePostgres }}
registry: {{ .defaults.containerRegistry }}
tag: {{ .defaults.tag }}
`
	if err := os.WriteFile(templateFile, []byte(templateContent), constants.StackFilePerm); err != nil {
		t.Fatalf("failed to write template: %v", err)
	}

	// This would be what the setup command does
	outDir := filepath.Join(tmpdir, "output")

	t.Run("full setup flow", func(t *testing.T) {
		// 1. Create secrets directory with correct permissions
		secretsDir := filepath.Join(outDir, constants.SecretsDirName)
		if err := os.MkdirAll(secretsDir, constants.SecretsDirPerm); err != nil {
			t.Fatal(err)
		}

		// Verify secrets directory permissions
		dirInfo, err := os.Stat(secretsDir)
		if err != nil {
			t.Fatalf("failed to stat secrets directory: %v", err)
		}
		if dirInfo.Mode().Perm() != constants.SecretsDirPerm {
			t.Errorf("Secrets directory permissions = %v, want %v", dirInfo.Mode().Perm(), constants.SecretsDirPerm)
		}

		// 2. Create secrets
		if err := createSecrets(secretsDir, false, defaultSecrets); err != nil {
			t.Errorf("createSecrets() error = %v", err)
		}

		// 3. Verify secrets were created
		for _, spec := range defaultSecrets {
			path := filepath.Join(secretsDir, spec.Name)
			if _, err := os.Stat(path); err != nil {
				t.Errorf("Expected secret %s to exist", spec.Name)
			}
		}

		// 4. Verify superadmin password length
		superadminPath := filepath.Join(secretsDir, constants.AdminSecretsFile)
		pwd, _ := os.ReadFile(superadminPath)
		if len(pwd) != constants.DefaultSuperadminPasswordLength {
			t.Errorf("Expected superadmin password length %d, got %d", constants.DefaultSuperadminPasswordLength, len(pwd))
		}

		// 5. Check postgres_password has correct length (not base64)
		postgresPath := filepath.Join(secretsDir, constants.PgPasswordFile)
		postgresPwd, _ := os.ReadFile(postgresPath)
		if len(postgresPwd) != constants.DefaultPostgresPasswordLength {
			t.Errorf("Expected postgres password length %d, got %d", constants.DefaultPostgresPasswordLength, len(postgresPwd))
		}
	})

	t.Run("template processing with camelCase", func(t *testing.T) {
		// This tests that templates using camelCase keys work correctly
		// In real usage, config.CreateDirAndFiles would be called

		// Just verify the template has correct camelCase references
		templateContent, _ := os.ReadFile(templateFile)
		content := string(templateContent)

		// Verify camelCase usage in template
		if !strings.Contains(content, "{{ .url }}") {
			t.Error("Template should use camelCase .url")
		}
		if !strings.Contains(content, "{{ .stackName }}") {
			t.Error("Template should use camelCase .stackName")
		}
		if !strings.Contains(content, "{{ .defaults.containerRegistry }}") {
			t.Error("Template should use camelCase .defaults.containerRegistry")
		}

		// Verify no PascalCase
		if strings.Contains(content, "{{ .Url }}") || strings.Contains(content, "{{ .StackName }}") {
			t.Error("Template should not use PascalCase")
		}
	})
}

func TestSetupFlow_WithHTTPS(t *testing.T) {
	tmpdir := t.TempDir()

	// Config with HTTPS enabled
	configFile := filepath.Join(tmpdir, "config.yml")
	configContent := `---
url: test.example.com
stackName: test-stack
filename: docker-compose.yml
enableLocalHTTPS: true
defaults:
  containerRegistry: registry.example.com
  tag: 4.2.21
`
	if err := os.WriteFile(configFile, []byte(configContent), constants.StackFilePerm); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	outDir := filepath.Join(tmpdir, "output")
	secretsDir := filepath.Join(outDir, constants.SecretsDirName)

	t.Run("creates HTTPS certificates when enabled", func(t *testing.T) {
		// Create secrets directory with correct permissions
		if err := os.MkdirAll(secretsDir, constants.SecretsDirPerm); err != nil {
			t.Fatal(err)
		}

		// Create regular secrets
		if err := createSecrets(secretsDir, false, defaultSecrets); err != nil {
			t.Errorf("createSecrets() error = %v", err)
		}

		// Create certificates (this would be called by setup command when enableLocalHTTPS is true)
		if err := createCerts(secretsDir, false); err != nil {
			t.Errorf("createCerts() error = %v", err)
		}

		// Verify certificates exist
		certPath := filepath.Join(secretsDir, constants.CertCertName)
		keyPath := filepath.Join(secretsDir, constants.CertKeyName)

		if _, err := os.Stat(certPath); err != nil {
			t.Error("Expected cert_crt to be created when enableLocalHTTPS is true")
		}

		if _, err := os.Stat(keyPath); err != nil {
			t.Error("Expected cert_key to be created when enableLocalHTTPS is true")
		}
	})
}

func TestSecretSpec_Pattern(t *testing.T) {
	t.Run("demonstrates SecretSpec pattern usage", func(t *testing.T) {
		// This test documents the SecretSpec pattern for future developers

		// Define custom secrets with different generators
		customSecrets := []SecretSpec{
			{
				Name: "api_key",
				Generator: func() ([]byte, error) {
					return []byte("custom-api-key"), nil
				},
			},
			{
				Name:      "webhook_secret",
				Generator: randomSecret, // Reuse existing generator
			},
			{
				Name: "short_password",
				Generator: func() ([]byte, error) {
					return randomString(12) // Custom length
				},
			},
		}

		tmpdir := t.TempDir()

		// Create all secrets
		if err := createSecrets(tmpdir, false, customSecrets); err != nil {
			t.Errorf("createSecrets() error = %v", err)
		}

		// Verify each was created with correct content/length
		apiKey, _ := os.ReadFile(filepath.Join(tmpdir, "api_key"))
		if string(apiKey) != "custom-api-key" {
			t.Error("Custom generator didn't work")
		}

		webhookSecret, _ := os.ReadFile(filepath.Join(tmpdir, "webhook_secret"))
		if len(webhookSecret) != 44 { // base64 of 32 bytes
			t.Error("Reused generator didn't work")
		}

		shortPwd, _ := os.ReadFile(filepath.Join(tmpdir, "short_password"))
		if len(shortPwd) != 12 {
			t.Error("Custom length generator didn't work")
		}
	})
}
