package actions

import (
	"context"
	"fmt"
	"time"

	"github.com/OpenSlides/openslides-cli/internal/k8s/client"
	"github.com/OpenSlides/openslides-cli/internal/logger"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	UpdateBackendmanageHelp      = "Updates an OpenSlides instance's backend."
	UpdateBackendmanageHelpExtra = `Updates the backendmanage service deployment image tag and registry to new version.

Examples:
  osmanage k8s update-backendmanage ./my.instance.dir.org --kubeconfig ~/.kube/config --tag 4.2.23 --containerRegistry myRegistry`
)

func UpdateBackendmanageCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-backendmanage <project-dir>",
		Short: UpdateBackendmanageHelp,
		Long:  UpdateBackendmanageHelp + "\n\n" + UpdateBackendmanageHelpExtra,
		Args:  cobra.ExactArgs(1),
	}

	kubeconfig := cmd.Flags().String("kubeconfig", "", "Path to kubeconfig file")
	revert := cmd.Flags().Bool("revert", false, "Changes image back with given tag and registry")
	tag := cmd.Flags().StringP("tag", "t", "", "Image tag (required)")
	containerRegistry := cmd.Flags().String("containerRegistry", "", "Container registry (required)")

	_ = cmd.MarkFlagRequired("tag")
	_ = cmd.MarkFlagRequired("containerRegistry")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		logger.Info("=== K8S UPDATE/REVERT BACKENDMANAGE ===")
		projectDir := args[0]
		namespace := extractNamespace(projectDir)

		logger.Info("Namespace: %s", namespace)

		k8sClient, err := client.New(*kubeconfig)
		if err != nil {
			return fmt.Errorf("creating k8s client: %w", err)
		}

		ctx := context.Background()

		if *revert {
			if err := revertBackendmanage(ctx, k8sClient, namespace, *tag, *containerRegistry); err != nil {
				return err
			}

			logger.Info("Successfully reverted backendmanage")
		} else {
			if err := updateBackendmanage(ctx, k8sClient, namespace, *tag, *containerRegistry); err != nil {
				return err
			}

			logger.Info("Successfully updated backendmanage")
		}
		return nil
	}

	return cmd
}

func updateBackendmanage(ctx context.Context, k8sClient *client.Client, namespace, tag, containerRegistry string) error {
	image := fmt.Sprintf("%s/openslides-backend:%s", containerRegistry, tag)

	logger.Info("Updating deployment to image: %s", image)

	patch := []byte(fmt.Sprintf(`{"spec":{"template":{"spec":{"containers":[{"name":"backendmanage","image":"%s"}]}}}}`, image))

	updated, err := k8sClient.Clientset().AppsV1().Deployments(namespace).Patch(
		ctx,
		"backendmanage",
		types.StrategicMergePatchType,
		patch,
		metav1.PatchOptions{},
	)
	if err != nil {
		return fmt.Errorf("patching deployment: %w", err)
	}

	logger.Info("Patch applied (generation: %d)", updated.Generation)

	logger.Info("Waiting for rollout to complete...")
	if err := waitForRollout(ctx, k8sClient, namespace, "backendmanage", 5*time.Minute); err != nil {
		return fmt.Errorf("rollout failed: %w", err)
	}

	return nil
}

func revertBackendmanage(ctx context.Context, k8sClient *client.Client, namespace, tag, containerRegistry string) error {
	image := fmt.Sprintf("%s/openslides-backend:%s", containerRegistry, tag)

	logger.Info("Reverting deployment to image: %s", image)

	patch := []byte(fmt.Sprintf(`{"spec":{"template":{"spec":{"containers":[{"name":"backendmanage","image":"%s"}]}}}}`, image))

	updated, err := k8sClient.Clientset().AppsV1().Deployments(namespace).Patch(
		ctx,
		"backendmanage",
		types.StrategicMergePatchType,
		patch,
		metav1.PatchOptions{},
	)
	if err != nil {
		return fmt.Errorf("patching deployment: %w", err)
	}

	logger.Info("Patch applied (generation: %d)", updated.Generation)

	logger.Info("Waiting for rollout to complete...")
	if err := waitForRollout(ctx, k8sClient, namespace, "backendmanage", 5*time.Minute); err != nil {
		return fmt.Errorf("rollout failed: %w", err)
	}

	return nil
}

func waitForRollout(ctx context.Context, k8sClient *client.Client, namespace, deploymentName string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout after %v", timeout)

		case <-ticker.C:
			d, err := k8sClient.Clientset().AppsV1().Deployments(namespace).Get(ctx, deploymentName, metav1.GetOptions{})
			if err != nil {
				return err
			}

			if d.Status.ObservedGeneration >= d.Generation &&
				d.Status.UpdatedReplicas == *d.Spec.Replicas &&
				d.Status.AvailableReplicas == *d.Spec.Replicas {
				return nil
			}

			logger.Info("  %d/%d replicas ready", d.Status.ReadyReplicas, *d.Spec.Replicas)
		}
	}
}
