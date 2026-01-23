package actions

import (
	"context"
	"fmt"
	"time"

	"github.com/OpenSlides/openslides-cli/internal/k8s/client"
	"github.com/OpenSlides/openslides-cli/internal/logger"
	"github.com/spf13/cobra"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	HealthHelp      = "Check health status of an OpenSlides instance"
	HealthHelpExtra = `Checks if all pods in the instance namespace are ready and running.

Examples:
  osmanage k8s health ./my.instance.dir.org 
  osmanage k8s health ./my.instance.dir.org --wait --timeout 5m`
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
	timeout := cmd.Flags().Duration("timeout", 5*time.Minute, "Timeout for wait operation")

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

// checkHealth checks the current health status and prints details
func checkHealth(ctx context.Context, k8sClient *client.Client, namespace string) error {
	pods, err := k8sClient.Clientset().CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("listing pods: %w", err)
	}

	totalPods := len(pods.Items)
	if totalPods == 0 {
		return fmt.Errorf("no pods found in namespace %s", namespace)
	}

	readyPods := 0
	fmt.Printf("Namespace: %s\n", namespace)
	fmt.Println("Pod Status:")

	for _, pod := range pods.Items {
		ready := isPodReady(&pod)
		if ready {
			readyPods++
		}

		status := "✗"
		if ready {
			status = "✓"
		}
		fmt.Printf("  %s %-50s %s\n", status, pod.Name, pod.Status.Phase)
	}

	fmt.Printf("\nReady: %d/%d pods\n", readyPods, totalPods)

	if readyPods != totalPods {
		return fmt.Errorf("instance is not healthy")
	}

	logger.Info("Instance is healthy")
	return nil
}
