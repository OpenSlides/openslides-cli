package actions

import (
	"fmt"
	"os"

	"github.com/OpenSlides/openslides-cli/internal/logger"
	"github.com/spf13/cobra"
)

const (
	RemoveHelp      = "Remove an OpenSlides instance directory"
	RemoveHelpExtra = `Removes the entire OpenSlides instance directory and all its contents.

WARNING: This operation is irreversible! All configuration files, secrets,
and instance data in the directory will be permanently deleted.

Examples:
  osmanage k8s remove --project-dir ./my-instance
  osmanage k8s remove -d ./old-instance`
)

func RemoveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove",
		Short: RemoveHelp,
		Long:  RemoveHelp + "\n\n" + RemoveHelpExtra,
		Args:  cobra.NoArgs,
	}

	projectDir := cmd.Flags().StringP("project-dir", "d", "", "Project directory to remove (required)")
	force := cmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")

	_ = cmd.MarkFlagRequired("project-dir")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		logger.Info("=== K8S REMOVE INSTANCE ===")
		logger.Debug("Project directory: %s", *projectDir)

		if err := removeInstance(*projectDir, *force); err != nil {
			return fmt.Errorf("removing instance: %w", err)
		}

		logger.Info("Instance removed successfully")
		return nil
	}

	return cmd
}

// removeInstance removes the entire project directory
func removeInstance(projectDir string, force bool) error {
	info, err := os.Stat(projectDir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%s does not exist", projectDir)
		}
		return fmt.Errorf("checking directory: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", projectDir)
	}

	if !force {
		logger.Warn("This will permanently delete: %s", projectDir)
		logger.Warn("All configuration files, secrets, and data will be lost!")

		fmt.Print("Are you sure you want to continue? [y/N]: ")
		var response string
		fmt.Scanln(&response)

		if response != "y" && response != "Y" && response != "yes" && response != "YES" {
			logger.Info("Removal cancelled")
			return nil
		}
	}

	logger.Info("Removing instance directory: %s", projectDir)

	if err := os.RemoveAll(projectDir); err != nil {
		return fmt.Errorf("removing directory: %w", err)
	}

	return nil
}
