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
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"
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
