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
	StartHelp      = "Start an OpenSlides instance"
	StartHelpExtra = `Applies Kubernetes manifests to start an OpenSlides instance.

Examples:
  osmanage k8s start ./my.instance.dir.org
  osmanage k8s start ./my.instance.dir.org --skip-ready-check
  osmanage k8s start ./my.instance.dir.org --kubeconfig ~/.kube/config --timeout 30s`
)

func StartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start <instance-dir>",
		Short: StartHelp,
		Long:  StartHelp + "\n\n" + StartHelpExtra,
		Args:  cobra.ExactArgs(1),
	}

	kubeconfig := cmd.Flags().String("kubeconfig", "", "Path to kubeconfig file")
	skipReadyCheck := cmd.Flags().Bool("skip-ready-check", false, "Skip waiting for instance to become ready")
	timeout := cmd.Flags().Duration("timeout", constants.DefaultInstanceTimeout, "Timeout for instance health check")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		logger.Info("=== K8S START INSTANCE ===")
		instanceDir := args[0]
		logger.Debug("Instance directory: %s", instanceDir)

		k8sClient, err := client.New(*kubeconfig)
		if err != nil {
			return fmt.Errorf("creating k8s client: %w", err)
		}

		ctx := context.Background()

		namespacePath := filepath.Join(instanceDir, constants.NamespaceYAML)
		namespace, err := applyManifest(ctx, k8sClient, namespacePath)
		if err != nil {
			return fmt.Errorf("applying namespace: %w", err)
		}
		logger.Info("Applied namespace: %s", namespace)

		tlsSecretPath := filepath.Join(instanceDir, constants.SecretsDirName, constants.TlsCertSecretYAML)
		tlsExists, err := utils.FileExists(tlsSecretPath)
		if err != nil {
			return fmt.Errorf("checking tls secret path %s: %w", tlsSecretPath, err)
		}
		if tlsExists {
			logger.Info("Found and applying %s", tlsSecretPath)
			if _, err := applyManifest(ctx, k8sClient, tlsSecretPath); err != nil {
				return fmt.Errorf("applying TLS secret: %w", err)
			}
		}

		stackDir := filepath.Join(instanceDir, constants.StackDirName)
		logger.Info("Applying stack manifests from: %s", stackDir)
		if err := applyDirectory(ctx, k8sClient, stackDir); err != nil {
			return fmt.Errorf("applying stack: %w", err)
		}

		if *skipReadyCheck {
			logger.Info("Skipping ready check")
			return nil
		}

		if err := waitForInstanceHealthy(ctx, k8sClient, namespace, *timeout); err != nil {
			return fmt.Errorf("waiting for ready: %w", err)
		}

		logger.Info("Instance started successfully")
		return nil
	}

	return cmd
}
