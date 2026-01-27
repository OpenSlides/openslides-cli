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
	ScaleHelp      = "Scales an OpenSlides service deployment"
	ScaleHelpExtra = `Applies the deployment manifest for a specific service after replicas have been modified.

Note: You must edit the deployment file to change the replica count before running this command.

Examples:
  osmanage k8s scale ./my.instance.dir.org --service backendmanage
  osmanage k8s scale ./my.instance.dir.org --service autoupdate --skip-ready-check
  osmanage k8s scale ./my.instance.dir.org --service search --kubeconfig ~/.kube/config --timeout 30s`
)

func ScaleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scale <project-dir>",
		Short: ScaleHelp,
		Long:  ScaleHelp + "\n\n" + ScaleHelpExtra,
		Args:  cobra.ExactArgs(1),
	}

	service := cmd.Flags().String("service", "", "Service deployment to scale (required)")
	kubeconfig := cmd.Flags().String("kubeconfig", "", "Path to kubeconfig file")
	skipReadyCheck := cmd.Flags().Bool("skip-ready-check", false, "Skip waiting for deployment to become ready")
	timeout := cmd.Flags().Duration("timeout", 3*time.Minute, "Timeout for ready check")

	_ = cmd.MarkFlagRequired("service")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if *service == "" {
			return fmt.Errorf("--service cannot be empty")
		}

		logger.Info("=== K8S SCALE SERVICE ===")
		projectDir := args[0]
		logger.Debug("Project directory: %s", projectDir)
		logger.Info("Service: %s", *service)

		namespace := extractNamespace(projectDir)
		logger.Info("Namespace: %s", namespace)

		k8sClient, err := client.New(*kubeconfig)
		if err != nil {
			return fmt.Errorf("creating k8s client: %w", err)
		}

		ctx := context.Background()

		// Construct path to deployment file
		deploymentFile := fmt.Sprintf("%s-deployment.yaml", *service)
		deploymentPath := filepath.Join(projectDir, "stack", deploymentFile)

		logger.Info("Applying deployment manifest: %s", deploymentPath)
		if _, err := applyManifest(ctx, k8sClient, deploymentPath); err != nil {
			return fmt.Errorf("applying deployment: %w", err)
		}

		if *skipReadyCheck {
			logger.Info("Skipping ready check")
			return nil
		}

		logger.Info("Waiting for deployment to become ready...")
		// Wait for the specific deployment (service name is deployment name)
		if err := waitForDeploymentReady(ctx, k8sClient, namespace, *service, *timeout); err != nil {
			return fmt.Errorf("waiting for deployment ready: %w", err)
		}

		logger.Info("Service scaled successfully")
		return nil
	}

	return cmd
}
