package actions

import (
	"context"
	"fmt"

	"github.com/OpenSlides/openslides-cli/internal/k8s/client"
	"github.com/OpenSlides/openslides-cli/internal/utils"
	"github.com/spf13/cobra"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	GetServiceAddressHelp      = "Get service ClusterIP:Port address"
	GetServiceAddressHelpExtra = `Returns the service ClusterIP:Port for the given instance and service name.

Examples:
  osmanage k8s get-service-address ./my.instance.dir.org backendmanage
  osmanage k8s get-service-address ./my.instance.dir.org backendmanage --kubeconfig ~/.kube/config`
)

// GetServiceAddressCmd creates the Cobra CLI command
func GetServiceAddressCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-service-address <instance-dir> <service-name>",
		Short: GetServiceAddressHelp,
		Long:  GetServiceAddressHelp + "\n\n" + GetServiceAddressHelpExtra,
		Args:  cobra.ExactArgs(2),
	}

	kubeconfig := cmd.Flags().String("kubeconfig", "", "Path to kubeconfig file")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		instanceDir := args[0]
		serviceName := args[1]

		namespace := utils.ExtractNamespace(instanceDir)

		k8sClient, err := client.New(*kubeconfig)
		if err != nil {
			return fmt.Errorf("creating k8s client: %w", err)
		}

		ctx := context.Background()
		address, err := GetServiceAddress(ctx, k8sClient.Clientset(), namespace, serviceName)
		if err != nil {
			return err
		}

		fmt.Println(address)
		return nil
	}

	return cmd
}

// GetServiceAddress retrieves the ClusterIP:Port address for a service in a namespace.
// Returns error if service doesn't exist or has invalid configuration.
func GetServiceAddress(ctx context.Context, k8sClient kubernetes.Interface, namespace, serviceName string) (string, error) {
	svc, err := k8sClient.CoreV1().Services(namespace).Get(ctx, serviceName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return "", fmt.Errorf("service %s not found in namespace %s", serviceName, namespace)
		}
		return "", fmt.Errorf("getting service: %w", err)
	}

	if svc.Spec.ClusterIP == "" || svc.Spec.ClusterIP == "None" {
		return "", fmt.Errorf("service %s has no ClusterIP", serviceName)
	}

	if len(svc.Spec.Ports) == 0 {
		return "", fmt.Errorf("service %s has no ports", serviceName)
	}

	port := svc.Spec.Ports[0].Port
	address := fmt.Sprintf("%s:%d", svc.Spec.ClusterIP, port)

	return address, nil
}
