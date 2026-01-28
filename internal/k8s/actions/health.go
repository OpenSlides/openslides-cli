package actions

import (
	"context"
	"fmt"
	"time"

	"github.com/OpenSlides/openslides-cli/internal/k8s/client"
	"github.com/OpenSlides/openslides-cli/internal/logger"
	"github.com/spf13/cobra"
)

const (
	HealthHelp      = "Check health status of an OpenSlides instance"
	HealthHelpExtra = `Checks if all pods in the instance namespace are ready and running.

Examples:
  osmanage k8s health ./my.instance.dir.org 
  osmanage k8s health ./my.instance.dir.org --wait --timeout 30s`
)

func HealthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "health",
		Short: HealthHelp,
		Long:  HealthHelp + "\n\n" + HealthHelpExtra,
		Args:  cobra.ExactArgs(1),
	}

	kubeconfig := cmd.Flags().String("kubeconfig", "", "Path to kubeconfig file")
	wait := cmd.Flags().Bool("wait", false, "Wait for instance to become healthy")
	timeout := cmd.Flags().Duration("timeout", 3*time.Minute, "Timeout for wait operation")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		logger.Info("=== K8S HEALTH CHECK ===")
		projectDir := args[0]
		namespace := extractNamespace(projectDir)
		logger.Debug("Namespace: %s", namespace)

		k8sClient, err := client.New(*kubeconfig)
		if err != nil {
			return fmt.Errorf("creating k8s client: %w", err)
		}

		ctx := context.Background()

		if *wait {
			return waitForHealthy(ctx, k8sClient, namespace, *timeout)
		}

		return checkHealth(ctx, k8sClient, namespace)
	}

	return cmd
}
