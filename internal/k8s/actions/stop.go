package actions

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/OpenSlides/openslides-cli/internal/k8s/client"
	"github.com/OpenSlides/openslides-cli/internal/logger"
	"github.com/spf13/cobra"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

const (
	StopHelp      = "Stop an OpenSlides instance"
	StopHelpExtra = `Stops an OpenSlides instance by deleting its namespace.
If a TLS certificate secret exists, it will be saved before deletion.

Examples:
  osmanage k8s stop --namespace openslides-prod --project-dir ./my-instance
  osmanage k8s stop -n openslides-test --project-dir ./test-instance`

	tlsCertSecret = "tls-letsencrypt"
)

func StopCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop",
		Short: StopHelp,
		Long:  StopHelp + "\n\n" + StopHelpExtra,
		Args:  cobra.NoArgs,
	}

	namespace := cmd.Flags().StringP("namespace", "n", "", "Kubernetes namespace to delete (required)")
	projectDir := cmd.Flags().StringP("project-dir", "d", "", "Project directory to save TLS secret (required)")
	kubeconfig := cmd.Flags().String("kubeconfig", "", "Path to kubeconfig file")

	_ = cmd.MarkFlagRequired("namespace")
	_ = cmd.MarkFlagRequired("project-dir")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		logger.Info("=== K8S STOP INSTANCE ===")
		logger.Debug("Namespace: %s", *namespace)
		logger.Debug("Project directory: %s", *projectDir)

		k8sClient, err := client.New(*kubeconfig)
		if err != nil {
			return fmt.Errorf("creating k8s client: %w", err)
		}

		ctx := context.Background()

		if err := saveTLSSecret(ctx, k8sClient, *namespace, *projectDir); err != nil {
			logger.Warn("Failed to save TLS secret: %v", err)
		}

		logger.Info("Stopping instance: %s", *namespace)
		if err := deleteNamespace(ctx, k8sClient, *namespace); err != nil {
			return fmt.Errorf("deleting namespace: %w", err)
		}

		logger.Info("Instance stopped successfully")
		return nil
	}

	return cmd
}

// saveTLSSecret saves the TLS certificate secret to a YAML file if it exists
func saveTLSSecret(ctx context.Context, k8sClient *client.Client, namespace, projectDir string) error {
	clientset := k8sClient.Clientset()

	secret, err := clientset.CoreV1().Secrets(namespace).Get(ctx, tlsCertSecret, metav1.GetOptions{})
	if err != nil {
		logger.Debug("TLS secret %s not found in namespace %s", tlsCertSecret, namespace)
		return nil
	}

	logger.Info("Found %s secret. Saving to %s", tlsCertSecret, tlsCertSecretYAML)

	secretYAML, err := yaml.Marshal(secret)
	if err != nil {
		return fmt.Errorf("marshaling secret to YAML: %w", err)
	}

	secretsDir := filepath.Join(projectDir, "secrets")
	if err := os.MkdirAll(secretsDir, 0755); err != nil {
		return fmt.Errorf("creating secrets directory: %w", err)
	}

	secretPath := filepath.Join(projectDir, tlsCertSecretYAML)
	if err := os.WriteFile(secretPath, secretYAML, 0600); err != nil {
		return fmt.Errorf("writing secret file: %w", err)
	}

	logger.Info("Saved TLS secret to: %s", secretPath)
	return nil
}

// deleteNamespace deletes a Kubernetes namespace
func deleteNamespace(ctx context.Context, k8sClient *client.Client, namespace string) error {
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
	return waitForNamespaceDeletion(ctx, k8sClient, namespace)
}

// waitForNamespaceDeletion waits for a namespace to be completely deleted
func waitForNamespaceDeletion(ctx context.Context, k8sClient *client.Client, namespace string) error {
	clientset := k8sClient.Clientset()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	timeout := time.After(5 * time.Minute)

	for {
		select {
		case <-ticker.C:
			_, err := clientset.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
			if err != nil {
				logger.Debug("Namespace %s successfully deleted", namespace)
				return nil
			}
			logger.Debug("Namespace %s still terminating...", namespace)

		case <-timeout:
			return fmt.Errorf("timeout waiting for namespace %s to be deleted", namespace)
		}
	}
}
