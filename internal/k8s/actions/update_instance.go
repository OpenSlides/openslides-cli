package actions

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/OpenSlides/openslides-cli/internal/constants"
	"github.com/OpenSlides/openslides-cli/internal/k8s/client"
	"github.com/OpenSlides/openslides-cli/internal/logger"
	"github.com/OpenSlides/openslides-cli/internal/utils"
	"github.com/spf13/cobra"
)

const (
	UpdateInstanceHelp      = "Updates an OpenSlides instance."
	UpdateInstanceHelpExtra = `Updates the instance by applying new manifest files from the instance directory.

Examples:
  osmanage k8s update-instance ./my.instance.dir.org
  osmanage k8s update-instance ./my.instance.dir.org --skip-ready-check
  osmanage k8s update-instance ./my.instance.dir.org --kubeconfig ~/.kube/config`
)

func UpdateInstanceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-instance <instance-dir>",
		Short: UpdateInstanceHelp,
		Long:  UpdateInstanceHelp + "\n\n" + UpdateInstanceHelpExtra,
		Args:  cobra.ExactArgs(1),
	}

	kubeconfig := cmd.Flags().String("kubeconfig", "", "Path to kubeconfig file")
	skipReadyCheck := cmd.Flags().Bool("skip-ready-check", false, "Skip waiting for instance to become ready")
	timeout := cmd.Flags().Duration("timeout", constants.DefaultInstanceTimeout, "Timeout for instance health check")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		logger.Info("=== K8S UPDATE INSTANCE ===")
		instanceDir := args[0]

		logger.Debug("Instance directory: %s", instanceDir)

		namespace := utils.ExtractNamespace(instanceDir)
		logger.Info("Namespace: %s", namespace)

		k8sClient, err := client.New(*kubeconfig)
		if err != nil {
			return fmt.Errorf("creating k8s client: %w", err)
		}

		ctx := context.Background()

		isActive, err := namespaceIsActive(ctx, k8sClient, namespace)
		if err != nil {
			return fmt.Errorf("checking namespace: %w", err)
		}

		if !isActive {
			logger.Info("%s is not running.", namespace)
			logger.Info("The configuration has been updated and the instance will be upgraded upon its next start.")
			logger.Info("Note that the next start might take a long time due to pending migrations.")
			logger.Info("Consider starting the instance and running migrations now.")
			logger.Info("Alternatively, downgrade for now and run migrations in the background once the instance is started.")
			return nil
		}

		logger.Info("Updating OpenSlides services.")

		stackDir := filepath.Join(instanceDir, constants.StackDirName)
		if err := applyDirectory(ctx, k8sClient, stackDir); err != nil {
			return fmt.Errorf("applying stack: %w", err)
		}

		if *skipReadyCheck {
			logger.Info("Skip ready check.")
			return nil
		}

		logger.Info("Waiting for instance to become ready...")
		if err := waitForInstanceHealthy(ctx, k8sClient, namespace, *timeout); err != nil {
			return fmt.Errorf("waiting for instance health: %w", err)
		}

		logger.Info("Instance updated successfully")
		return nil
	}

	return cmd
}
