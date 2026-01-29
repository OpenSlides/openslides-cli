package actions

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/OpenSlides/openslides-cli/internal/logger"
	"github.com/spf13/cobra"
)

const (
	CreateHelp      = "Create an OpenSlides instance with custom passwords"
	CreateHelpExtra = `Creates an OpenSlides instance by setting up the secrets directory
with the provided database and superadmin passwords.

This command:
1. Creates/secures the secrets directory (700 permissions)
2. Sets all secret files to 600 permissions
3. Writes the database password to postgres_password
4. Writes the superadmin password to superadmin

The secrets directory must already exist (created by 'setup' command).

Examples:
  osmanage k8s create ./my.instance.dir.org --db-password "mydbpass" --superadmin-password "myadminpass"
  osmanage k8s create ./my.instance.dir.org --db-password "$(cat db.txt)" --superadmin-password "$(cat admin.txt)"`

	adminSecretsFile             = "superadmin"
	pgPasswordFile               = "postgres_password"
	secretDirPerm    os.FileMode = 0700
	secretFilePerm   os.FileMode = 0600
)

func CreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create <project-dir>",
		Short: CreateHelp,
		Long:  CreateHelp + "\n\n" + CreateHelpExtra,
		Args:  cobra.ExactArgs(1),
	}

	dbPassword := cmd.Flags().String("db-password", "", "PostgreSQL database password (required)")
	superadminPassword := cmd.Flags().String("superadmin-password", "", "Superadmin password (required)")

	_ = cmd.MarkFlagRequired("db-password")
	_ = cmd.MarkFlagRequired("superadmin-password")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if *dbPassword == "" {
			return fmt.Errorf("--db-password cannot be empty")
		}
		if *superadminPassword == "" {
			return fmt.Errorf("--superadmin-password cannot be empty")
		}

		logger.Info("=== K8S CREATE INSTANCE ===")
		projectDir := args[0]
		logger.Debug("Project directory: %s", projectDir)

		if err := createInstance(projectDir, *dbPassword, *superadminPassword); err != nil {
			return fmt.Errorf("creating instance: %w", err)
		}

		logger.Info("Instance created successfully")
		return nil
	}

	return cmd
}

// createInstance sets up the secrets directory with the provided passwords
func createInstance(projectDir, dbPassword, superadminPassword string) error {
	secretsDir := filepath.Join(projectDir, "secrets")

	if _, err := os.Stat(secretsDir); os.IsNotExist(err) {
		return fmt.Errorf("secrets directory does not exist: %s (run 'setup' first)", secretsDir)
	}

	logger.Info("Creating instance: %s", filepath.Base(projectDir))

	logger.Debug("Securing secrets directory: %s", secretsDir)
	if err := secureSecretsDirectory(secretsDir); err != nil {
		return fmt.Errorf("securing secrets directory: %w", err)
	}

	pgPasswordPath := filepath.Join(secretsDir, pgPasswordFile)
	logger.Debug("Writing PostgreSQL password to: %s", pgPasswordPath)
	if err := os.WriteFile(pgPasswordPath, []byte(dbPassword), secretFilePerm); err != nil {
		return fmt.Errorf("writing postgres password: %w", err)
	}

	superadminPath := filepath.Join(secretsDir, adminSecretsFile)
	logger.Debug("Writing superadmin password to: %s", superadminPath)
	if err := os.WriteFile(superadminPath, []byte(superadminPassword), secretFilePerm); err != nil {
		return fmt.Errorf("writing superadmin password: %w", err)
	}

	logger.Info("Passwords configured successfully")
	return nil
}

// secureSecretsDirectory sets restrictive permissions on the secrets directory and all files within
func secureSecretsDirectory(secretsDir string) error {
	if err := os.Chmod(secretsDir, secretDirPerm); err != nil {
		return fmt.Errorf("setting directory permissions: %w", err)
	}

	entries, err := os.ReadDir(secretsDir)
	if err != nil {
		return fmt.Errorf("reading secrets directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filePath := filepath.Join(secretsDir, entry.Name())
		if err := os.Chmod(filePath, secretFilePerm); err != nil {
			return fmt.Errorf("setting permissions for %s: %w", entry.Name(), err)
		}
	}

	return nil
}
