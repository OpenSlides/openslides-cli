package actions

import (
	"context"
	"fmt"
	"os"
	"strings"
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
	Healthy    bool
	Ready      int
	Total      int
	ActivePods int
	Pods       []corev1.Pod
}

// DeploymentStatus represents the rollout status of a deployment
type DeploymentStatus struct {
	Ready    int
	Desired  int
	Complete bool
}

// shouldCountPod returns true if the pod should be counted toward instance health
// excludes completed, failed, and terminating pods
func shouldCountPod(pod *corev1.Pod) bool {
	return pod.Status.Phase != corev1.PodSucceeded &&
		pod.Status.Phase != corev1.PodFailed &&
		pod.DeletionTimestamp == nil
}

// IsPodReady checks if a pod is ready based on pod status condition
func IsPodReady(pod *corev1.Pod) bool {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}

// GetHealthStatus returns instance pod health
func GetHealthStatus(ctx context.Context, k8sClient *client.Client, namespace string) (*HealthStatus, error) {
	pods, err := k8sClient.Clientset().CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing pods: %w", err)
	}

	deployments, err := k8sClient.Clientset().AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing deployments: %w", err)
	}

	desiredTotal := 0
	for _, d := range deployments.Items {
		if d.Spec.Replicas != nil {
			desiredTotal += int(*d.Spec.Replicas)
		}
	}

	var filteredPods []corev1.Pod
	for _, pod := range pods.Items {
		if shouldCountPod(&pod) {
			filteredPods = append(filteredPods, pod)
		}
	}

	ready := 0
	for _, pod := range filteredPods {
		if IsPodReady(&pod) {
			ready++
		}
	}

	total := desiredTotal
	if total == 0 {
		total = len(filteredPods)
	}

	return &HealthStatus{
		Healthy:    ready == total,
		Ready:      ready,
		Total:      total,
		ActivePods: len(filteredPods),
		Pods:       filteredPods,
	}, nil
}

// pollUntil runs fn on every interval tick until fn returns done=true, fn returns
// an error, or the timeout is exceeded.
func pollUntil(ctx context.Context, interval, timeout time.Duration, fn func() (done bool, err error)) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for {
		select {
		case <-ticker.C:
			done, err := fn()
			if err != nil {
				return err
			}
			if done {
				return nil
			}
		case <-timeoutCtx.Done():
			return fmt.Errorf("timeout: %w", timeoutCtx.Err())
		}
	}
}

// printHealthStatus prints pod-level health details to stdout.
func printHealthStatus(namespace string, status *HealthStatus) {
	if status.Total == 0 {
		fmt.Printf("No pods found in namespace %s\n", namespace)
		return
	}

	fmt.Printf("\nNamespace: %s\n", namespace)
	fmt.Printf("Ready: %d/%d pods (active: %d)\n\n", status.Ready, status.Total, status.ActivePods)
	fmt.Println("Pod Status:")
	for _, pod := range status.Pods {
		icon := constants.IconNotReady
		if IsPodReady(&pod) {
			icon = constants.IconReady
		}
		fmt.Printf("  %s %-50s %s\n", icon, pod.Name, pod.Status.Phase)
	}
	fmt.Println()
}

// getNotReadyNames returns the names of pods that are not ready.
func getNotReadyNames(pods []corev1.Pod) []string {
	var names []string
	for _, pod := range pods {
		if !IsPodReady(&pod) {
			names = append(names, pod.Name)
		}
	}
	return names
}

// WaitForInstanceHealthy waits for an instance to become healthy.
//
// When callback is non-nil (gRPC mode), it is called on every tick with the
// current status and no progress bar is rendered. When callback is nil (CLI
// mode), a progress bar is written to stdout.
func WaitForInstanceHealthy(
	ctx context.Context,
	k8sClient *client.Client,
	namespace string,
	timeout time.Duration,
	callback func(*HealthStatus) error,
) error {
	var bar *progressbar.ProgressBar
	if callback == nil {
		initial, err := GetHealthStatus(ctx, k8sClient, namespace)
		if err != nil {
			return fmt.Errorf("getting initial health status: %w", err)
		}
		if initial.Total > 0 {
			bar = createProgressBar(initial.Total, "Pods ready", constants.AddDetailLineBuffer)
		}
	}

	var lastStatus *HealthStatus

	err := pollUntil(ctx, constants.TickerDuration, timeout, func() (bool, error) {
		status, err := GetHealthStatus(ctx, k8sClient, namespace)
		if err != nil {
			logger.Debug("Error checking health: %v", err)
			return false, nil
		}
		lastStatus = status

		if callback != nil {
			if err := callback(status); err != nil {
				return false, err
			}
		} else {
			if bar == nil && status.Total > 0 {
				bar = createProgressBar(status.Total, "Pods ready", constants.AddDetailLineBuffer)
			}
			if bar != nil && !bar.IsFinished() {
				notReady := getNotReadyNames(status.Pods)
				detail := ""
				if len(notReady) > 0 {
					detail = fmt.Sprintf("%s Pending: %s", constants.IconNotReady, strings.Join(notReady, ", "))
				}
				if err := bar.AddDetail(detail); err != nil {
					return false, fmt.Errorf("updating progress bar detail: %w", err)
				}
				if err := bar.Set(status.Ready); err != nil {
					return false, fmt.Errorf("setting progress bar: %w", err)
				}
			}
		}

		if status.Healthy {
			if bar != nil && !bar.IsFinished() {
				if err := bar.Finish(); err != nil {
					return false, fmt.Errorf("finishing progress bar: %w", err)
				}
			}
			logger.Info("Instance is healthy: %d/%d pods ready", status.Ready, status.Total)
			return true, nil
		}
		return false, nil
	})

	if err != nil {
		if bar != nil && !bar.IsFinished() {
			_ = bar.Finish()
		}
		logger.Warn("Timeout reached. Current status:")
		if lastStatus != nil {
			printHealthStatus(namespace, lastStatus)
		}
		return fmt.Errorf("timeout waiting for instance to become healthy")
	}
	return nil
}

// waitForDeploymentReady waits for a specific deployment rollout to complete.
func waitForDeploymentReady(
	ctx context.Context,
	k8sClient *client.Client,
	namespace, deploymentName string,
	timeout time.Duration,
	callback func(*DeploymentStatus) error,
) error {
	logger.Debug("Waiting for deployment %s to be ready (timeout: %v)", deploymentName, timeout)

	deployment, err := k8sClient.Clientset().AppsV1().Deployments(namespace).Get(ctx, deploymentName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("getting deployment %s: %w", deploymentName, err)
	}
	desired := int(*deployment.Spec.Replicas)

	var bar *progressbar.ProgressBar
	if callback == nil && desired > 0 {
		bar = createProgressBar(desired, fmt.Sprintf("Waiting for %s rollout", deploymentName), 0)
	}

	var lastDeployment *appsv1.Deployment

	err = pollUntil(ctx, constants.TickerDuration, timeout, func() (bool, error) {
		d, err := k8sClient.Clientset().AppsV1().Deployments(namespace).Get(ctx, deploymentName, metav1.GetOptions{})
		if err != nil {
			logger.Debug("Error getting deployment: %v", err)
			return false, nil
		}
		lastDeployment = d

		desired := int(*d.Spec.Replicas)
		updated := int(d.Status.UpdatedReplicas)
		ready := int(d.Status.ReadyReplicas)
		available := int(d.Status.AvailableReplicas)
		total := int(d.Status.Replicas)

		status := &DeploymentStatus{
			Ready:   ready,
			Desired: desired,
		}

		complete := d.Status.ObservedGeneration >= d.Generation &&
			updated == desired &&
			available == desired &&
			ready == desired &&
			total == desired

		if callback != nil {
			status.Complete = complete
			if err := callback(status); err != nil {
				return false, err
			}
		} else {
			if bar != nil && !bar.IsFinished() {
				if err := bar.Set(ready); err != nil {
					return false, fmt.Errorf("setting progress bar: %w", err)
				}
			}
		}

		logger.Debug("Deployment %s: %d/%d updated, %d/%d ready, %d total (generation: %d/%d)",
			deploymentName, updated, desired, ready, desired, total,
			d.Status.ObservedGeneration, d.Generation)

		if complete {
			if bar != nil && !bar.IsFinished() {
				if err := bar.Finish(); err != nil {
					return false, fmt.Errorf("finishing progress bar: %w", err)
				}
			}
			logger.Info("Deployment %s is ready with %d replicas", deploymentName, desired)
			return true, nil
		}
		return false, nil
	})

	if err != nil {
		if bar != nil && !bar.IsFinished() {
			_ = bar.Finish()
		}
		logger.Warn("Timeout reached. Deployment status:")
		if lastDeployment != nil {
			printDeploymentStatus(namespace, deploymentName, lastDeployment)
		}
		return fmt.Errorf("timeout waiting for deployment %s rollout", deploymentName)
	}
	return nil
}

// waitForNamespaceDeletion waits for a namespace to be completely deleted.
func waitForNamespaceDeletion(ctx context.Context, k8sClient *client.Client, namespace string, timeout time.Duration) error {
	clientset := k8sClient.Clientset()
	bar := createProgressBar(-1, fmt.Sprintf("Stopping %s", namespace), 0)

	var lastErr error
	err := pollUntil(ctx, constants.TickerDuration, timeout, func() (bool, error) {
		_ = bar.Add(1)
		_, err := clientset.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				if err := bar.Finish(); err != nil {
					return false, fmt.Errorf("finishing progress bar: %w", err)
				}
				logger.Debug("Namespace %s successfully deleted", namespace)
				return true, nil
			}
			lastErr = err
			logger.Warn("Error checking namespace: %v", err)
		}
		logger.Debug("Namespace %s still terminating...", namespace)
		return false, nil
	})

	if err != nil {
		_ = bar.Finish()
		if lastErr != nil {
			return fmt.Errorf("timeout waiting for namespace %s to be deleted (last error: %w)", namespace, lastErr)
		}
		return fmt.Errorf("timeout waiting for namespace %s to be deleted", namespace)
	}
	return nil
}

// namespaceIsActive checks if a namespace exists and is active.
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

// printDeploymentStatus prints deployment rollout details to stdout.
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

func createProgressBar(max int, description string, maxDetailRow int) *progressbar.ProgressBar {
	opts := []progressbar.Option{
		progressbar.OptionSetDescription(description),
		progressbar.OptionSetWidth(constants.ProgressBarWidth),
		progressbar.OptionSetWriter(os.Stdout),
		progressbar.OptionSetMaxDetailRow(maxDetailRow),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        constants.Saucer,
			SaucerPadding: constants.SaucerPadding,
			BarStart:      constants.BarStart,
			BarEnd:        constants.BarEnd,
		}),
		progressbar.OptionThrottle(constants.ThrottleDuration),
		progressbar.OptionOnCompletion(func() {
			fmt.Println()
		}),
	}
	if max > 0 {
		opts = append(opts, progressbar.OptionShowCount())
	} else {
		opts = append(opts, progressbar.OptionSpinnerType(constants.SpinnerType))
	}
	return progressbar.NewOptions(max, opts...)
}
