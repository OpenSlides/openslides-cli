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
  osmanage k8s update-backendmanage ./my.instance.dir.org --kubeconfig ~/.kube/config --tag 4.2.23 --container-registry myRegistry
  osmanage k8s update-backendmanage ./my.instance.dir.org --tag 4.2.23 --container-registry myRegistry --timeout 30s
  osmanage k8s update-backendmanage ./my.instance.dir.org --tag 4.2.23 --container-registry myRegistry --revert --timeout 30s`

	backendmanageDeployment = "backendmanage"
	backendmanageContainer  = "backendmanage"
)

func UpdateBackendmanageCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-backendmanage <project-dir>",
		Short: UpdateBackendmanageHelp,
		Long:  UpdateBackendmanageHelp + "\n\n" + UpdateBackendmanageHelpExtra,
		Args:  cobra.ExactArgs(1),
	}

	tag := cmd.Flags().StringP("tag", "t", "", "Image tag (required)")
	containerRegistry := cmd.Flags().String("container-registry", "", "Container registry (required)")
	kubeconfig := cmd.Flags().String("kubeconfig", "", "Path to kubeconfig file")
	revert := cmd.Flags().Bool("revert", false, "Changes image back with given tag and registry")
	timeout := cmd.Flags().Duration("timeout", 3*time.Minute, "Timeout for deployment readiness check")

	_ = cmd.MarkFlagRequired("tag")
	_ = cmd.MarkFlagRequired("container-registry")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if *tag == "" {
			return fmt.Errorf("--tag cannot be empty")
		}
		if *containerRegistry == "" {
			return fmt.Errorf("--container-registry cannot be empty")
		}

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
			if err := revertBackendmanage(ctx, k8sClient, namespace, *tag, *containerRegistry, *timeout); err != nil {
				return err
			}

			logger.Info("Successfully reverted backendmanage")
		} else {
			if err := updateBackendmanage(ctx, k8sClient, namespace, *tag, *containerRegistry, *timeout); err != nil {
				return err
			}

			logger.Info("Successfully updated backendmanage")
		}
		return nil
	}

	return cmd
}

func updateBackendmanage(ctx context.Context, k8sClient *client.Client, namespace, tag, containerRegistry string, timeout time.Duration) error {
	image := fmt.Sprintf("%s/openslides-backend:%s", containerRegistry, tag)

	logger.Info("Updating deployment to image: %s", image)

	patch := []byte(fmt.Sprintf(`{"spec":{"template":{"spec":{"containers":[{"name":"%s","image":"%s"}]}}}}`, backendmanageContainer, image))

	updated, err := k8sClient.Clientset().AppsV1().Deployments(namespace).Patch(
		ctx,
		backendmanageDeployment,
		types.StrategicMergePatchType,
		patch,
		metav1.PatchOptions{},
	)
	if err != nil {
		return fmt.Errorf("patching deployment: %w", err)
	}

	logger.Info("Patch applied (generation: %d)", updated.Generation)

	logger.Info("Waiting for rollout to complete...")
	if err := waitForDeploymentReady(ctx, k8sClient, namespace, backendmanageDeployment, timeout); err != nil {
		return fmt.Errorf("rollout failed: %w", err)
	}

	return nil
}

func revertBackendmanage(ctx context.Context, k8sClient *client.Client, namespace, tag, containerRegistry string, timeout time.Duration) error {
	image := fmt.Sprintf("%s/openslides-backend:%s", containerRegistry, tag)

	logger.Info("Reverting deployment to image: %s", image)

	patch := []byte(fmt.Sprintf(`{"spec":{"template":{"spec":{"containers":[{"name":"%s","image":"%s"}]}}}}`, backendmanageContainer, image))

	updated, err := k8sClient.Clientset().AppsV1().Deployments(namespace).Patch(
		ctx,
		backendmanageDeployment,
		types.StrategicMergePatchType,
		patch,
		metav1.PatchOptions{},
	)
	if err != nil {
		return fmt.Errorf("patching deployment: %w", err)
	}

	logger.Info("Patch applied (generation: %d)", updated.Generation)

	logger.Info("Waiting for rollout to complete...")
	if err := waitForDeploymentReady(ctx, k8sClient, namespace, backendmanageDeployment, timeout); err != nil {
		return fmt.Errorf("rollout failed: %w", err)
	}

	return nil
}
