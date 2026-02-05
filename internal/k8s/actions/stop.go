package actions

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/OpenSlides/openslides-cli/internal/constants"
	"github.com/OpenSlides/openslides-cli/internal/k8s/client"
	"github.com/OpenSlides/openslides-cli/internal/logger"
	"github.com/spf13/cobra"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

const (
	StopHelp      = "Stop an OpenSlides instance"
	StopHelpExtra = `Stops an OpenSlides instance by deleting its Kubernetes namespace.
If a TLS certificate secret exists, it will be saved before deletion.

Examples:
  osmanage k8s stop ./my.instance.dir.org --kubeconfig ~/.kube/config`
)

func StopCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop <instance-dir>",
		Short: StopHelp,
		Long:  StopHelp + "\n\n" + StopHelpExtra,
		Args:  cobra.ExactArgs(1),
	}

	kubeconfig := cmd.Flags().String("kubeconfig", "", "Path to kubeconfig file")
	timeout := cmd.Flags().Duration("timeout", constants.DefaultNamespaceTimeout, "timeout for namespace deletion")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		logger.Info("=== K8S STOP INSTANCE ===")
		instanceDir := args[0]

		logger.Debug("Instance directory: %s", instanceDir)

		k8sClient, err := client.New(*kubeconfig)
		if err != nil {
			return fmt.Errorf("creating k8s client: %w", err)
		}

		ctx := context.Background()

		namespace := extractNamespace(instanceDir)
		if err := saveTLSSecret(ctx, k8sClient, namespace, instanceDir); err != nil {
			logger.Warn("Failed to save TLS secret: %v", err)
		}

		logger.Info("Stopping instance: %s", namespace)
		if err := deleteNamespace(ctx, k8sClient, namespace, *timeout); err != nil {
			return fmt.Errorf("deleting namespace: %w", err)
		}

		logger.Info("Instance stopped successfully")
		return nil
	}

	return cmd
}

// saveTLSSecret saves the TLS certificate secret to a YAML file if it exists
func saveTLSSecret(ctx context.Context, k8sClient *client.Client, namespace, instanceDir string) error {
	clientset := k8sClient.Clientset()

	secret, err := clientset.CoreV1().Secrets(namespace).Get(ctx, constants.TlsCertSecret, metav1.GetOptions{})
	if err != nil {
		logger.Debug("TLS secret %s not found in namespace %s", constants.TlsCertSecret, namespace)
		return nil
	}

	secretYAML, err := yaml.Marshal(secret)
	if err != nil {
		return fmt.Errorf("marshaling secret to YAML: %w", err)
	}

	secretsDir := filepath.Join(instanceDir, constants.SecretsDirName)
	if err := os.MkdirAll(secretsDir, constants.SecretsDirPerm); err != nil {
		return fmt.Errorf("creating secrets directory: %w", err)
	}

	secretPath := filepath.Join(secretsDir, constants.TlsCertSecretYAML)
	if err := os.WriteFile(secretPath, secretYAML, constants.SecretFilePerm); err != nil {
		return fmt.Errorf("writing secret file: %w", err)
	}

	logger.Info("Saved TLS secret to: %s", secretPath)
	return nil
}

// deleteNamespace deletes a Kubernetes namespace
func deleteNamespace(ctx context.Context, k8sClient *client.Client, namespace string, timeout time.Duration) error {
	clientset := k8sClient.Clientset()

	logger.Debug("Deleting namespace: %s", namespace)

	deletePolicy := metav1.DeletePropagationForeground
	err := clientset.CoreV1().Namespaces().Delete(ctx, namespace, metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	})
	if err != nil {
		return fmt.Errorf("deleting namespace %s: %w", namespace, err)
	}

	logger.Info("Namespace %s deletion initiated", namespace)

	logger.Debug("Waiting for namespace to be fully deleted...")
	return waitForNamespaceDeletion(ctx, k8sClient, namespace, timeout)
}
