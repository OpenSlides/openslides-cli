package actions

import (
	"context"
	"fmt"
	"time"

	"github.com/OpenSlides/openslides-cli/internal/k8s/client"
	"github.com/OpenSlides/openslides-cli/internal/logger"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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

// waitForDeploymentReady waits for a specific deployment to be ready
func waitForDeploymentReady(ctx context.Context, k8sClient *client.Client, namespace, deploymentName string, timeout time.Duration) error {
	logger.Debug("Waiting for deployment %s to be ready (timeout: %v)", deploymentName, timeout)

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for {
		select {
		case <-ticker.C:
			deployment, err := k8sClient.Clientset().AppsV1().Deployments(namespace).Get(timeoutCtx, deploymentName, metav1.GetOptions{})
			if err != nil {
				logger.Debug("Error getting deployment: %v", err)
				continue
			}

			if deployment.Status.ObservedGeneration >= deployment.Generation &&
				deployment.Status.UpdatedReplicas == *deployment.Spec.Replicas &&
				deployment.Status.AvailableReplicas == *deployment.Spec.Replicas &&
				deployment.Status.ReadyReplicas == *deployment.Spec.Replicas {

				logger.Info("Deployment %s is ready with %d replicas", deploymentName, *deployment.Spec.Replicas)
				return nil
			}

			logger.Debug("Deployment %s: %d/%d replicas ready (generation: %d/%d)",
				deploymentName,
				deployment.Status.ReadyReplicas,
				*deployment.Spec.Replicas,
				deployment.Status.ObservedGeneration,
				deployment.Generation)

		case <-timeoutCtx.Done():
			return fmt.Errorf("timeout waiting for deployment %s to become ready", deploymentName)
		}
	}
}
