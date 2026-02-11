package actions

import (
	"context"
	"fmt"
	"time"

	"github.com/OpenSlides/openslides-cli/internal/constants"
	"github.com/OpenSlides/openslides-cli/internal/k8s/client"
	"github.com/OpenSlides/openslides-cli/internal/logger"
	"github.com/schollz/progressbar/v3"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// HealthStatus represents the health status of an instance
type HealthStatus struct {
	Healthy bool
	Ready   int
	Total   int
	Pods    []corev1.Pod
}

// getHealthStatus returns health metrics
func getHealthStatus(ctx context.Context, k8sClient *client.Client, namespace string) (*HealthStatus, error) {
	pods, err := k8sClient.Clientset().CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing pods: %w", err)
	}

	var filteredPods []corev1.Pod
	for _, pod := range pods.Items {
		if pod.Status.Phase == corev1.PodSucceeded {
			continue
		}
		if pod.DeletionTimestamp != nil {
			continue
		}
		filteredPods = append(filteredPods, pod)
	}

	total := len(filteredPods)
	if total == 0 {
		return &HealthStatus{
			Healthy: false,
			Ready:   0,
			Total:   0,
			Pods:    nil,
		}, nil
	}

	ready := 0
	for _, pod := range filteredPods {
		if isPodReady(&pod) {
			ready++
		}
	}

	return &HealthStatus{
		Healthy: ready == total,
		Ready:   ready,
		Total:   total,
		Pods:    filteredPods,
	}, nil
}

// Helper to print instance pod status
func printHealthStatus(namespace string, status *HealthStatus) {
	if status.Total == 0 {
		fmt.Printf("No pods found in namespace %s\n", namespace)
		return
	}

	fmt.Printf("\nNamespace: %s\n", namespace)
	fmt.Printf("Ready: %d/%d pods\n\n", status.Ready, status.Total)
	fmt.Println("Pod Status:")

	for _, pod := range status.Pods {
		ready := isPodReady(&pod)
		icon := constants.IconNotReady
		if ready {
			icon = constants.IconReady
		}
		fmt.Printf("  %s %-50s %s\n", icon, pod.Name, pod.Status.Phase)
	}
	fmt.Println()
}

// checkHealth checks the current health status and prints details
func checkHealth(ctx context.Context, k8sClient *client.Client, namespace string) error {
	status, err := getHealthStatus(ctx, k8sClient, namespace)
	if err != nil {
		return fmt.Errorf("getting health status: %w", err)
	}

	printHealthStatus(namespace, status)

	if !status.Healthy {
		return fmt.Errorf("instance is not healthy: %d/%d pods ready", status.Ready, status.Total)
	}

	logger.Info("Instance is healthy")
	return nil
}

// waitForInstanceHealthy waits for instance to become healthy
func waitForInstanceHealthy(ctx context.Context, k8sClient *client.Client, namespace string, timeout time.Duration) error {
	logger.Info("Waiting for instance to become healthy (timeout: %v)", timeout)

	ticker := time.NewTicker(constants.TickerDuration)
	defer ticker.Stop()

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var lastStatus *HealthStatus
	var bar *progressbar.ProgressBar

	for {
		select {
		case <-ticker.C:
			status, err := getHealthStatus(ctx, k8sClient, namespace)
			if err != nil {
				logger.Debug("Error checking health: %v", err)
				continue
			}
			lastStatus = status

			if bar == nil && status.Total > 0 {
				bar = createProgressBar(status.Total, "Pods ready", status.Total)
			} else if bar != nil {
				bar.ChangeMax(status.Total)
			}
			if bar != nil && !bar.IsFinished() {
				for _, pod := range status.Pods {
					icon := constants.IconReady
					if !isPodReady(&pod) {
						icon = constants.IconNotReady
					}
					if err := bar.AddDetail(fmt.Sprintf("%s %s", icon, pod.Name)); err != nil {
						return fmt.Errorf("adding detail on pending pods: %w", err)
					}
				}
				if err := bar.AddDetail(""); err != nil {
					return fmt.Errorf("adding trailing newline detail: %w", err)
				}
				if err := bar.Set(status.Ready); err != nil {
					return fmt.Errorf("setting progress bar: %w", err)
				}
			}

			if status.Healthy {
				if bar != nil && !bar.IsFinished() {
					if err := bar.Finish(); err != nil {
						return fmt.Errorf("finishing progress bar: %w", err)
					}
					fmt.Println()
				}
				logger.Info("Instance is healthy: %d/%d pods ready", status.Ready, status.Total)
				return nil
			}

		case <-timeoutCtx.Done():
			if bar != nil && !bar.IsFinished() {
				if err := bar.Finish(); err != nil {
					return fmt.Errorf("finishing progress bar: %w", err)
				}
				fmt.Println()
			}
			logger.Warn("Timeout reached. Current status:")
			if lastStatus != nil {
				printHealthStatus(namespace, lastStatus)
			}
			return fmt.Errorf("timeout waiting for instance to become healthy")
		}
	}
}

func createProgressBar(max int, description string, maxDetailRow int) *progressbar.ProgressBar {
	opts := []progressbar.Option{
		progressbar.OptionSetDescription(description),
		progressbar.OptionSetWidth(constants.ProgressBarWidth),
		progressbar.OptionSetMaxDetailRow(maxDetailRow + constants.AddDetailLineBuffer),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        constants.Saucer,
			SaucerPadding: constants.SaucerPadding,
			BarStart:      constants.BarStart,
			BarEnd:        constants.BarEnd,
		}),
		progressbar.OptionThrottle(constants.ThrottleDuration),
	}

	if max > 0 {
		opts = append(opts, progressbar.OptionShowCount())
	} else {
		opts = append(opts, progressbar.OptionSpinnerType(constants.SpinnerType))
	}

	return progressbar.NewOptions(max, opts...)
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

// namespaceIsActive checks if a namespace exists and is active
func namespaceIsActive(ctx context.Context, k8sClient *client.Client, namespace string) (bool, error) {
	ns, err := k8sClient.Clientset().CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("getting namespace: %w", err)
	}

	return ns.Status.Phase == corev1.NamespaceActive, nil
}

// Helper to print deployment status
func printDeploymentStatus(namespace, name string, deployment *appsv1.Deployment) {
	fmt.Printf("\nDeployment: %s (namespace: %s)\n", name, namespace)
	fmt.Printf("Generation: %d/%d (observed/current)\n",
		deployment.Status.ObservedGeneration,
		deployment.Generation)
	fmt.Printf("Replicas:\n")
	fmt.Printf("  Desired:   %d\n", *deployment.Spec.Replicas)
	fmt.Printf("  Current:   %d\n", deployment.Status.Replicas)
	fmt.Printf("  Ready:     %d\n", deployment.Status.ReadyReplicas)
	fmt.Printf("  Updated:   %d\n", deployment.Status.UpdatedReplicas)
	fmt.Printf("  Available: %d\n", deployment.Status.AvailableReplicas)

	if len(deployment.Status.Conditions) > 0 {
		fmt.Println("\nConditions:")
		for _, condition := range deployment.Status.Conditions {
			icon := constants.IconReady
			if condition.Status != corev1.ConditionTrue {
				icon = constants.IconNotReady
			}
			fmt.Printf("  %s %-20s %s\n", icon, condition.Type, condition.Message)
		}
	}
	fmt.Println()
}

// waitForDeploymentReady waits for a specific deployment to be ready
func waitForDeploymentReady(ctx context.Context, k8sClient *client.Client, namespace, deploymentName string, timeout time.Duration) error {
	logger.Debug("Waiting for deployment %s to be ready (timeout: %v)", deploymentName, timeout)

	ticker := time.NewTicker(constants.TickerDuration)
	defer ticker.Stop()

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var lastDeployment *appsv1.Deployment
	var bar *progressbar.ProgressBar

	for {
		select {
		case <-ticker.C:
			deployment, err := k8sClient.Clientset().AppsV1().Deployments(namespace).Get(timeoutCtx, deploymentName, metav1.GetOptions{})
			if err != nil {
				logger.Debug("Error getting deployment: %v", err)
				continue
			}

			lastDeployment = deployment

			desired := int(*deployment.Spec.Replicas)
			updated := int(deployment.Status.UpdatedReplicas)
			ready := int(deployment.Status.ReadyReplicas)
			available := int(deployment.Status.AvailableReplicas)
			total := int(deployment.Status.Replicas)
			observedGen := deployment.Status.ObservedGeneration
			gen := deployment.Generation

			if bar == nil && desired > 0 {
				bar = createProgressBar(-1, fmt.Sprintf("Waiting for %s deployment rollout", deploymentName), 0)
			}

			if bar != nil {
				_ = bar.Add(1)
			}

			if observedGen >= gen &&
				updated == desired &&
				available == desired &&
				ready == desired &&
				total == desired {
				if bar != nil {
					if err := bar.Finish(); err != nil {
						return fmt.Errorf("finishing progress bar: %w", err)
					}
					fmt.Println()
				}
				logger.Info("Deployment %s is ready with %d replicas", deploymentName, desired)
				return nil
			}

			logger.Debug("Deployment %s: %d/%d updated, %d/%d ready, %d total (generation: %d/%d)",
				deploymentName,
				updated, desired,
				ready, desired,
				total,
				observedGen, gen)

		case <-timeoutCtx.Done():
			if bar != nil {
				if err := bar.Finish(); err != nil {
					return fmt.Errorf("finishing progress bar: %w", err)
				}
				fmt.Println()
			}
			logger.Warn("Timeout reached. Deployment status:")
			if lastDeployment != nil {
				printDeploymentStatus(namespace, deploymentName, lastDeployment)
			}

			return fmt.Errorf("timeout waiting for deployment %s rollout", deploymentName)
		}
	}
}

// waitForNamespaceDeletion waits for a namespace to be completely deleted
func waitForNamespaceDeletion(ctx context.Context, k8sClient *client.Client, namespace string, timeout time.Duration) error {
	clientset := k8sClient.Clientset()
	ticker := time.NewTicker(constants.TickerDuration)
	defer ticker.Stop()

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	bar := createProgressBar(-1, fmt.Sprintf("Stopping %s", namespace), 0)

	for {
		select {
		case <-ticker.C:
			_ = bar.Add(1)
			_, err := clientset.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
			if err != nil {
				if !errors.IsNotFound(err) {
					logger.Warn("Error checking namespace: %v", err)
					continue
				}
				if err := bar.Finish(); err != nil {
					return fmt.Errorf("finishing progress bar: %w", err)
				}
				fmt.Println()
				logger.Debug("Namespace %s successfully deleted", namespace)
				return nil
			}
			logger.Debug("Namespace %s still terminating...", namespace)

		case <-timeoutCtx.Done():
			if err := bar.Finish(); err != nil {
				return fmt.Errorf("finishing progress bar: %w", err)
			}
			fmt.Println()
			return fmt.Errorf("timeout waiting for namespace %s to be deleted", namespace)
		}
	}
}
