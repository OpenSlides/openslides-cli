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

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"
)

const (
	StartHelp      = "Start an OpenSlides instance"
	StartHelpExtra = `Applies Kubernetes manifests to start an OpenSlides instance.

Examples:
  osmanage k8s start ./my-instance
  osmanage k8s start ./my-instance --skip-ready-check
  osmanage k8s start ./my-instance --kubeconfig ~/.kube/config`

	tlsCertSecretYAML = "secrets/tls-letsencrypt-secret.yaml"
)

func StartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start <project-dir>",
		Short: StartHelp,
		Long:  StartHelp + "\n\n" + StartHelpExtra,
		Args:  cobra.ExactArgs(1),
	}

	kubeconfig := cmd.Flags().String("kubeconfig", "", "Path to kubeconfig file")
	skipReadyCheck := cmd.Flags().Bool("skip-ready-check", false, "Skip waiting for instance to become ready")
	timeout := cmd.Flags().Duration("timeout", 5*time.Minute, "Timeout for ready check")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		projectDir := args[0]

		logger.Info("=== K8S START INSTANCE ===")
		logger.Debug("Project directory: %s", projectDir)

		k8sClient, err := client.New(*kubeconfig)
		if err != nil {
			return fmt.Errorf("creating k8s client: %w", err)
		}

		ctx := context.Background()

		namespacePath := filepath.Join(projectDir, "namespace.yaml")
		namespace, err := applyManifest(ctx, k8sClient, namespacePath)
		if err != nil {
			return fmt.Errorf("applying namespace: %w", err)
		}
		logger.Info("Applied namespace: %s", namespace)

		tlsSecretPath := filepath.Join(projectDir, tlsCertSecretYAML)
		if fileExists(tlsSecretPath) {
			logger.Info("Found and applying %s", tlsCertSecretYAML)
			if _, err := applyManifest(ctx, k8sClient, tlsSecretPath); err != nil {
				return fmt.Errorf("applying TLS secret: %w", err)
			}
		}

		stackDir := filepath.Join(projectDir, "stack")
		logger.Info("Applying stack manifests from: %s", stackDir)
		if err := applyDirectory(ctx, k8sClient, stackDir); err != nil {
			return fmt.Errorf("applying stack: %w", err)
		}

		if *skipReadyCheck {
			logger.Info("Skipping ready check")
			return nil
		}

		logger.Info("Waiting for instance to become ready...")
		if err := waitForHealthy(ctx, k8sClient, namespace, *timeout); err != nil {
			return fmt.Errorf("waiting for ready: %w", err)
		}

		logger.Info("Instance started successfully")
		return nil
	}

	return cmd
}

// applyManifest applies a single YAML manifest file using RESTMapper
func applyManifest(ctx context.Context, k8sClient *client.Client, manifestPath string) (string, error) {
	logger.Debug("Applying manifest: %s", manifestPath)

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return "", fmt.Errorf("reading manifest: %w", err)
	}

	var obj unstructured.Unstructured
	if err := yaml.Unmarshal(data, &obj); err != nil {
		return "", fmt.Errorf("parsing YAML: %w", err)
	}

	namespace := obj.GetNamespace()
	if namespace == "" && obj.GetKind() == "Namespace" {
		namespace = obj.GetName()
	}

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(k8sClient.Config())
	if err != nil {
		return "", fmt.Errorf("creating discovery client: %w", err)
	}

	apiGroupResources, err := restmapper.GetAPIGroupResources(discoveryClient)
	if err != nil {
		return "", fmt.Errorf("getting API group resources: %w", err)
	}

	mapper := restmapper.NewDiscoveryRESTMapper(apiGroupResources)

	gvk := obj.GroupVersionKind()

	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return "", fmt.Errorf("getting REST mapping for %s: %w", gvk.String(), err)
	}

	dynamicClient, err := dynamic.NewForConfig(k8sClient.Config())
	if err != nil {
		return "", fmt.Errorf("creating dynamic client: %w", err)
	}

	var result *unstructured.Unstructured
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		if namespace == "" {
			namespace = "default"
		}
		result, err = dynamicClient.Resource(mapping.Resource).Namespace(namespace).Apply(
			ctx,
			obj.GetName(),
			&obj,
			metav1.ApplyOptions{
				FieldManager: "osmanage",
				Force:        true,
			},
		)
	} else {
		result, err = dynamicClient.Resource(mapping.Resource).Apply(
			ctx,
			obj.GetName(),
			&obj,
			metav1.ApplyOptions{
				FieldManager: "osmanage",
				Force:        true,
			},
		)
	}

	if err != nil {
		return namespace, fmt.Errorf("applying %s/%s: %w", obj.GetKind(), obj.GetName(), err)
	}

	logger.Info("Applied %s: %s", result.GetKind(), result.GetName())
	return namespace, nil
}

// applyDirectory applies all YAML files in a directory
func applyDirectory(ctx context.Context, k8sClient *client.Client, dirPath string) error {
	files, err := os.ReadDir(dirPath)
	if err != nil {
		return fmt.Errorf("reading directory: %w", err)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		if !isYAMLFile(file.Name()) {
			logger.Debug("Skipping non-YAML file: %s", file.Name())
			continue
		}

		manifestPath := filepath.Join(dirPath, file.Name())
		if _, err := applyManifest(ctx, k8sClient, manifestPath); err != nil {
			logger.Warn("Failed to apply %s: %v", file.Name(), err)
			continue
		}
	}

	return nil
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// isYAMLFile checks if filename has YAML extension
func isYAMLFile(filename string) bool {
	ext := filepath.Ext(filename)
	return ext == ".yaml" || ext == ".yml"
}
