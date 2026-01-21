package actions

import (
	"context"
	"fmt"
	"strings"

	"github.com/OpenSlides/openslides-cli/internal/k8s/client"
	"github.com/OpenSlides/openslides-cli/internal/logger"
	"github.com/spf13/cobra"
)

const (
	UpdateBackendmanageHelp      = "Updates an OpenSlides instance's backendmanage service."
	UpdateBackendmanageHelpExtra = `Updates the backendmanage service deployment image tag and registry to new version.

Examples:
  osmanage k8s update-backendmanage ./my-instance --kubeconfig ~/.kube/config`

	managementBackend = "backendmanage"
)

func UpdateBackendmanageCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-backendmanage <project-dir>",
		Short: StartHelp,
		Long:  StartHelp + "\n\n" + StartHelpExtra,
		Args:  cobra.ExactArgs(1),
	}

	kubeconfig := cmd.Flags().String("kubeconfig", "", "Path to kubeconfig file")
	tag := cmd.Flags().StringP("tag", "t", "", "OpenSlides backendmanage service image tag")
	containerRegistry := cmd.Flags().String("containerRegistry", "", "OpenSlides backendmanage image ContainerRegistry")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		projectDir := args[0]

		logger.Info("=== K8S UPDATE BACKENDMANAGE ===")

		k8sClient, err := client.New(*kubeconfig)
		if err != nil {
			return fmt.Errorf("creating k8s client: %w", err)
		}

		ctx := context.Background()

		namespace := strings.ReplaceAll(projectDir, ".", "")
		logger.Info("Create namespace string: %s", namespace)

		err := updateBackendmanage(ctx, k8sClient, namespace, tag, containerRegistry)
		if err != nil {
			return fmt.Errorf("updating backendmanage service: %w", err)
		}
	}

	return cmd
}

func updateBackendmanage(ctx context.Context, k8sClient *client.Client, namespace, tag, containerRegistry string) error {
	patch := fmt.Sprintf(`{"spec":{"template":{"spec":{"containers":[{"name":"%s","image":"%s"}]}}}}`, container, image)
    
    _, err := clientset.AppsV1().Deployments(namespace).Patch(
        ctx,
        deployment,
        types.StrategicMergePatchType,
        []byte(patch),
        metav1.PatchOptions{},
    )
    
    return err
}
