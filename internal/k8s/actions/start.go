package actions

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/OpenSlides/openslides-cli/internal/k8s/client"
	"github.com/OpenSlides/openslides-cli/internal/logger"
	"github.com/spf13/cobra"
)

const (
	StartHelp      = "Start an OpenSlides instance"
	StartHelpExtra = `Applies Kubernetes manifests to start an OpenSlides instance.

Examples:
  osmanage k8s start ./my-instance
  osmanage k8s start ./my-instance --skip-ready-check
  osmanage k8s start ./my-instance --kubeconfig ~/.kube/config`

	tlsCertSecretYAML = "secrets/tls-letsencrypt-secret.yaml"
)

func StartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start <project-dir>",
		Short: StartHelp,
		Long:  StartHelp + "\n\n" + StartHelpExtra,
		Args:  cobra.ExactArgs(1),
	}

	kubeconfig := cmd.Flags().String("kubeconfig", "", "Path to kubeconfig file")
	skipReadyCheck := cmd.Flags().Bool("skip-ready-check", false, "Skip waiting for instance to become ready")
	timeout := cmd.Flags().Duration("timeout", 5*time.Minute, "Timeout for ready check")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		projectDir := args[0]

		logger.Info("=== K8S START INSTANCE ===")
		logger.Debug("Project directory: %s", projectDir)

		k8sClient, err := client.New(*kubeconfig)
		if err != nil {
			return fmt.Errorf("creating k8s client: %w", err)
		}

		ctx := context.Background()

		namespacePath := filepath.Join(projectDir, "namespace.yaml")
		namespace, err := applyManifest(ctx, k8sClient, namespacePath)
		if err != nil {
			return fmt.Errorf("applying namespace: %w", err)
		}
		logger.Info("Applied namespace: %s", namespace)

		tlsSecretPath := filepath.Join(projectDir, tlsCertSecretYAML)
		if fileExists(tlsSecretPath) {
			logger.Info("Found and applying %s", tlsCertSecretYAML)
			if _, err := applyManifest(ctx, k8sClient, tlsSecretPath); err != nil {
				return fmt.Errorf("applying TLS secret: %w", err)
			}
		}

		stackDir := filepath.Join(projectDir, "stack")
		logger.Info("Applying stack manifests from: %s", stackDir)
		if err := applyDirectory(ctx, k8sClient, stackDir); err != nil {
			return fmt.Errorf("applying stack: %w", err)
		}

		if *skipReadyCheck {
			logger.Info("Skipping ready check")
			return nil
		}

		logger.Info("Waiting for instance to become ready...")
		if err := waitForHealthy(ctx, k8sClient, namespace, *timeout); err != nil {
			return fmt.Errorf("waiting for ready: %w", err)
		}

		logger.Info("✓ Instance started successfully")
		return nil
	}

	return cmd
}
