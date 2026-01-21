package actions

import (
	"context"
	"fmt"
	"time"

	"github.com/OpenSlides/openslides-cli/internal/k8s/client"
	"github.com/OpenSlides/openslides-cli/internal/logger"
	"github.com/spf13/cobra"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	HealthHelp      = "Check health status of an OpenSlides instance"
	HealthHelpExtra = `Checks if all pods in the instance namespace are ready and running.

Examples:
  osmanage k8s health --namespace openslides-prod
  osmanage k8s health --namespace openslides-test --wait --timeout 5m`
)

func HealthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "health",
		Short: HealthHelp,
		Long:  HealthHelp + "\n\n" + HealthHelpExtra,
		Args:  cobra.NoArgs,
	}

	namespace := cmd.Flags().StringP("namespace", "n", "", "Kubernetes namespace (required)")
	kubeconfig := cmd.Flags().String("kubeconfig", "", "Path to kubeconfig file")
	wait := cmd.Flags().Bool("wait", false, "Wait for instance to become healthy")
	timeout := cmd.Flags().Duration("timeout", 5*time.Minute, "Timeout for wait operation")

	_ = cmd.MarkFlagRequired("namespace")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		logger.Info("=== K8S HEALTH CHECK ===")
		logger.Debug("Namespace: %s", *namespace)

		k8sClient, err := client.New(*kubeconfig)
		if err != nil {
			return fmt.Errorf("creating k8s client: %w", err)
		}

		ctx := context.Background()

		if *wait {
			return waitForHealthy(ctx, k8sClient, *namespace, *timeout)
		}

		return checkHealth(ctx, k8sClient, *namespace)
	}

	return cmd
}

// checkHealth checks the current health status
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

// waitForHealthy waits for instance to become healthy
func waitForHealthy(ctx context.Context, k8sClient *client.Client, namespace string, timeout time.Duration) error {
	logger.Info("Waiting for instance to become healthy (timeout: %v)", timeout)

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for {
		select {
		case <-ticker.C:
			healthy, ready, total, err := getHealthStatus(ctx, k8sClient, namespace)
			if err != nil {
				logger.Debug("Error checking health: %v", err)
				continue
			}

			logger.Debug("Health check: %d/%d pods ready", ready, total)

			if healthy {
				logger.Info("Instance is healthy!")
				return nil
			}

		case <-timeoutCtx.Done():
			return fmt.Errorf("timeout waiting for instance to become healthy")
		}
	}
}

// getHealthStatus returns health metrics
func getHealthStatus(ctx context.Context, k8sClient *client.Client, namespace string) (healthy bool, ready, total int, err error) {
	pods, err := k8sClient.Clientset().CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return false, 0, 0, err
	}

	total = len(pods.Items)
	if total == 0 {
		return false, 0, 0, nil
	}

	ready = 0
	for _, pod := range pods.Items {
		if isPodReady(&pod) {
			ready++
		}
	}

	healthy = ready == total
	return healthy, ready, total, nil
}

// isPodReady checks if a pod is ready
func isPodReady(pod *corev1.Pod) bool {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}
