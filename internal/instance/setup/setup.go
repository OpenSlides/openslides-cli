package setup

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"time"

	"github.com/OpenSlides/openslides-cli/internal/constants"
	"github.com/OpenSlides/openslides-cli/internal/instance/config"
	"github.com/OpenSlides/openslides-cli/internal/logger"
	"github.com/OpenSlides/openslides-cli/internal/utils"

	"github.com/spf13/cobra"
)

const (
	SetupHelp      = "Creates the required files for deployment"
	SetupHelpExtra = `Creates deployment configuration files and generates secrets for an OpenSlides instance.

This command:
1. Creates secrets directory with secure permissions
2. Generates authentication tokens and passwords
3. Creates SSL certificates (if enableLocalHTTPS: true)
4. Generates deployment files from templates

Examples:
  osmanage setup ./my.instance.dir.org
  osmanage setup ./my.instance.dir.org --force
  osmanage setup ./my.instance.dir.org --template ./custom --config ./config.yaml
  osmanage setup ./my.instance.dir.org --config ./base.yaml --config ./override.yaml`
)

type SecretSpec struct {
	Name      string
	Generator func() ([]byte, error)
}

var defaultSecrets = []SecretSpec{
	{constants.AuthTokenKey, randomSecret},
	{constants.AuthCookieKey, randomSecret},
	{constants.InternalAuthPassword, randomSecret},
	{constants.PgPasswordFile, func() ([]byte, error) { return randomString(constants.DefaultPostgresPasswordLength) }},
	{constants.AdminSecretsFile, func() ([]byte, error) { return randomString(constants.DefaultSuperadminPasswordLength) }},
}

func Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "setup <instance-dir>",
		Short: SetupHelp,
		Long:  SetupHelp + "\n\n" + SetupHelpExtra,
		Args:  cobra.ExactArgs(1),
	}

	force := cmd.Flags().BoolP("force", "f", false, "overwrite existing files")
	customTemplate := cmd.Flags().StringP("template", "t", "", "custom template file or directory")
	configFiles := cmd.Flags().StringArrayP("config", "c", nil, "custom YAML config file (can be used multiple times)")
	cmd.MarkFlagsRequiredTogether("template", "config")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		logger.Info("=== SETUP ===")

		baseDir := args[0]
		logger.Debug("Base directory: %s", baseDir)
		logger.Debug("Force: %v, Custom: %s", *force, *customTemplate)

		// Parse configuration
		cfg, err := config.NewConfig(*configFiles)
		if err != nil {
			return fmt.Errorf("parsing configuration: %w", err)
		}

		// Create secrets directory
		secretsDir := filepath.Join(baseDir, constants.SecretsDirName)
		logger.Debug("Creating secrets directory: %s", secretsDir)
		if err := os.MkdirAll(secretsDir, constants.SecretsDirPerm); err != nil {
			return fmt.Errorf("creating secrets directory: %w", err)
		}

		// Create secrets
		logger.Info("Creating secrets...")
		if err := createSecrets(secretsDir, *force, defaultSecrets); err != nil {
			return fmt.Errorf("creating secrets: %w", err)
		}

		// Create certificates if HTTPS is enabled
		if enableLocalHTTPS, ok := cfg["enableLocalHTTPS"].(bool); ok && enableLocalHTTPS {
			logger.Info("Creating SSL certificates...")
			if err := createCerts(secretsDir, *force); err != nil {
				return fmt.Errorf("creating certificates: %w", err)
			}
		}

		// Create deployment files
		logger.Info("Creating deployment files...")
		if err := config.CreateDirAndFiles(baseDir, *force, *customTemplate, cfg); err != nil {
			return fmt.Errorf("creating deployment files: %w", err)
		}

		logger.Info("Setup completed successfully")
		fmt.Printf("Setup completed in: %s\n", baseDir)
		return nil
	}

	return cmd
}

func createSecrets(dir string, force bool, secrets []SecretSpec) error {
	for _, spec := range secrets {
		logger.Debug("Generating secret: %s", spec.Name)
		data, err := spec.Generator()
		if err != nil {
			return fmt.Errorf("generating secret %q: %w", spec.Name, err)
		}
		if err := utils.CreateFile(dir, force, spec.Name, data, constants.SecretFilePerm); err != nil {
			return fmt.Errorf("creating secret file %q: %w", spec.Name, err)
		}
	}
	return nil
}

func randomSecret() ([]byte, error) {
	buf := new(bytes.Buffer)
	b64e := base64.NewEncoder(base64.StdEncoding, buf)

	if _, err := io.Copy(b64e, io.LimitReader(rand.Reader, constants.DefaultSecretBytesLength)); err != nil {
		if err := b64e.Close(); err != nil {
			return nil, fmt.Errorf("closing base64 encoder: %w", err)
		}
		return nil, fmt.Errorf("generating random secret: %w", err)
	}
	if err := b64e.Close(); err != nil {
		return nil, fmt.Errorf("closing base64 encoder: %w", err)
	}

	return buf.Bytes(), nil
}

func randomString(length int) ([]byte, error) {
	if length <= 0 {
		return nil, fmt.Errorf("length must be positive, got %d", length)
	}

	result := make([]byte, length)

	charset := constants.PasswordCharset
	maxIndex := len(charset)

	randomBytes := make([]byte, length)
	if _, err := rand.Read(randomBytes); err != nil {
		return nil, fmt.Errorf("generating random bytes: %w", err)
	}

	for i := range length {
		result[i] = charset[int(randomBytes[i])%maxIndex]
	}

	return result, nil
}

func createCerts(dir string, force bool) error {
	logger.Debug("Generating ECDSA key pair")
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("generating key: %w", err)
	}

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return fmt.Errorf("generating serial number: %w", err)
	}

	templ := x509.Certificate{
		SerialNumber:          serialNumber,
		Subject:               pkix.Name{Organization: []string{"OpenSlides"}},
		DNSNames:              []string{"localhost"},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(30, 0, 0),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	certData, err := x509.CreateCertificate(rand.Reader, &templ, &templ, &key.PublicKey, key)
	if err != nil {
		return fmt.Errorf("creating certificate: %w", err)
	}

	// Encode and save certificate
	buf1 := new(bytes.Buffer)
	if err := pem.Encode(buf1, &pem.Block{Type: "CERTIFICATE", Bytes: certData}); err != nil {
		return fmt.Errorf("encoding certificate: %w", err)
	}
	if err := utils.CreateFile(dir, force, constants.CertCertName, buf1.Bytes(), constants.SecretFilePerm); err != nil {
		return fmt.Errorf("creating certificate file: %w", err)
	}

	// Encode and save private key
	keyData, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return fmt.Errorf("marshalling key: %w", err)
	}
	buf2 := new(bytes.Buffer)
	if err := pem.Encode(buf2, &pem.Block{Type: "PRIVATE KEY", Bytes: keyData}); err != nil {
		return fmt.Errorf("encoding key: %w", err)
	}
	if err := utils.CreateFile(dir, force, constants.CertKeyName, buf2.Bytes(), constants.SecretFilePerm); err != nil {
		return fmt.Errorf("creating key file: %w", err)
	}

	logger.Debug("SSL certificates created successfully")
	return nil
}
