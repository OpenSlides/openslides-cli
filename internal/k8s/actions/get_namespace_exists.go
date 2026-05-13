package actions

import (
	"context"
	"fmt"
	"strings"

	"github.com/OpenSlides/openslides-cli/internal/k8s/client"
	"github.com/spf13/cobra"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	GetNamespaceExistsHelp      = "Returns true if namespace for given instance URL exists on cluster."
	GetNamespaceExistsHelpExtra = `Will look for namespace with string derived by removing dots 
	from instance url string (i.e.: my.instance.url.org -> myinstanceurlorg) on the cluster.

Examples:
  osmanage k8s get-namespace-exists my.instance.url.org --kubeconfig ~/.kube/config`
)

// GetCmd creates the Cobra CLI command
func GetNamespaceExistsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-namespace-exists <instance-url>",
		Short: GetNamespaceExistsHelp,
		Long:  GetNamespaceExistsHelp + "\n\n" + GetNamespaceExistsHelpExtra,
		Args:  cobra.ExactArgs(1),
	}

	kubeconfig := cmd.Flags().String("kubeconfig", "", "Path to kubeconfig file")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		instanceUrl := args[0]

		namespace := strings.ReplaceAll(instanceUrl, ".", "")

		k8sClient, err := client.New(*kubeconfig)
		if err != nil {
			return fmt.Errorf("creating k8s client: %w", err)
		}

		ctx := context.Background()
		exists, err := GetNamespaceExists(ctx, k8sClient.Clientset(), namespace)
		if err != nil {
			return err
		}

		fmt.Println(exists)
		return nil
	}

	return cmd
}

// GetNamespaceExists checks whether the given namespace exists on cluster.
// Returns error on failed client call.
func GetNamespaceExists(ctx context.Context, k8sClient kubernetes.Interface, namespace string) (bool, error) {
	_, err := k8sClient.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("getting namespace: %w", err)
	}
	return true, nil
}
