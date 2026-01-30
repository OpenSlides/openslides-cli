package actions

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/OpenSlides/openslides-cli/internal/k8s/client"
	"github.com/OpenSlides/openslides-cli/internal/logger"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
)

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

	mapper, err := k8sClient.RESTMapper()
	if err != nil {
		return "", fmt.Errorf("getting REST mapper: %w", err)
	}

	gvk := obj.GroupVersionKind()

	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return "", fmt.Errorf("getting REST mapping for %s: %w", gvk.String(), err)
	}

	dynamicClient, err := k8sClient.Dynamic()
	if err != nil {
		return "", fmt.Errorf("getting dynamic client: %w", err)
	}

	var result *unstructured.Unstructured
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		if namespace == "" {
			return "", fmt.Errorf("resource %s/%s is namespaced but has no namespace specified",
				obj.GetKind(), obj.GetName())
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
		// Cluster-scoped resource (Namespace, ClusterRole, etc.)
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
