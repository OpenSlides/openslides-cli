package remove

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
  osmanage remove ./my.instance.dir.org --force`
)

func Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove <instance-dir>",
		Short: RemoveHelp,
		Long:  RemoveHelp + "\n\n" + RemoveHelpExtra,
		Args:  cobra.ExactArgs(1),
	}

	force := cmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		logger.Info("=== K8S REMOVE INSTANCE ===")
		instanceDir := args[0]
		logger.Debug("Instance directory: %s", instanceDir)

		if err := removeInstance(instanceDir, *force); err != nil {
			return fmt.Errorf("removing instance: %w", err)
		}

		logger.Info("Instance removed successfully")
		return nil
	}

	return cmd
}

// removeInstance removes the entire instance directory
func removeInstance(instanceDir string, force bool) error {
	info, err := os.Stat(instanceDir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%s does not exist", instanceDir)
		}
		return fmt.Errorf("checking directory: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", instanceDir)
	}

	if !force {
		logger.Warn("This will permanently delete: %s", instanceDir)
		logger.Warn("All configuration files, secrets, and data will be lost!")

		fmt.Print("Are you sure you want to continue? [y/N]: ")
		var response string
		_, _ = fmt.Scanln(&response)

		if response != "y" && response != "Y" && response != "yes" && response != "YES" {
			logger.Info("Removal cancelled")
			return nil
		}
	}

	logger.Info("Removing instance directory: %s", instanceDir)

	if err := os.RemoveAll(instanceDir); err != nil {
		return fmt.Errorf("removing directory: %w", err)
	}

	return nil
}
