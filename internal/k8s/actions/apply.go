package actions

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/OpenSlides/openslides-cli/internal/constants"
	"github.com/OpenSlides/openslides-cli/internal/k8s/client"
	"github.com/OpenSlides/openslides-cli/internal/logger"
	"github.com/OpenSlides/openslides-cli/internal/utils"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"
)

const (
	// fieldManager identifies this client in Server-Side Apply operations
	fieldManager string = "osmanage"

	// forceConflicts takes ownership of fields from other managers when conflicts occur
	forceConflicts bool = true
)

// resourceKey uniquely identifies a Kubernetes resource by GVR and name
type resourceKey struct {
	gvr  schema.GroupVersionResource
	name string
}

// applyManifest applies a single YAML manifest file using RESTMapper and returns
// the applied resourceKey and namespace. Returns nil key if the manifest is skipped.
func applyManifest(ctx context.Context, k8sClient *client.Client, manifestPath string) (*resourceKey, string, error) {
	logger.Debug("Applying manifest: %s", manifestPath)

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, "", fmt.Errorf("reading manifest: %w", err)
	}

	var obj unstructured.Unstructured
	if err := yaml.Unmarshal(data, &obj); err != nil {
		return nil, "", fmt.Errorf("parsing YAML: %w", err)
	}

	if obj.GetKind() == "" {
		logger.Info("Skipping manifest with no kind: %s", manifestPath)
		return nil, "", nil
	}

	namespace := obj.GetNamespace()
	if namespace == "" && obj.GetKind() == "Namespace" {
		namespace = obj.GetName()
	}

	mapper, err := k8sClient.RESTMapper()
	if err != nil {
		return nil, "", fmt.Errorf("getting REST mapper: %w", err)
	}

	gvk := obj.GroupVersionKind()

	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return nil, "", fmt.Errorf("getting REST mapping for %s: %w", gvk.String(), err)
	}

	dynamicClient, err := k8sClient.Dynamic()
	if err != nil {
		return nil, "", fmt.Errorf("getting dynamic client: %w", err)
	}

	var result *unstructured.Unstructured
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		if namespace == "" {
			return nil, "", fmt.Errorf("resource %s/%s is namespaced but has no namespace specified",
				obj.GetKind(), obj.GetName())
		}
		result, err = dynamicClient.Resource(mapping.Resource).Namespace(namespace).Apply(
			ctx,
			obj.GetName(),
			&obj,
			metav1.ApplyOptions{
				FieldManager: fieldManager,
				Force:        forceConflicts,
			},
		)
	} else {
		// Cluster-scoped resource (Namespace, ClusterRole, etc.)
		result, err = dynamicClient.Resource(mapping.Resource).Apply(
			ctx,
			obj.GetName(),
			&obj,
			metav1.ApplyOptions{
				FieldManager: fieldManager,
				Force:        forceConflicts,
			},
		)
	}

	if err != nil {
		return nil, namespace, fmt.Errorf("applying %s/%s: %w", obj.GetKind(), obj.GetName(), err)
	}

	logger.Info("Applied %s: %s", result.GetKind(), result.GetName())
	return &resourceKey{gvr: mapping.Resource, name: obj.GetName()}, namespace, nil
}

// applyDirectory applies all YAML files in a directory and returns the set of applied resources.
func applyDirectory(ctx context.Context, k8sClient *client.Client, dirPath string) ([]resourceKey, error) {
	files, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("reading directory: %w", err)
	}

	var yamlFiles []os.DirEntry
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		if !utils.IsYAMLFile(file.Name()) {
			logger.Debug("Skipping non-YAML file: %s", file.Name())
			continue
		}
		yamlFiles = append(yamlFiles, file)
	}

	sort.Slice(yamlFiles, func(i, j int) bool {
		kindI := getKindFromFile(filepath.Join(dirPath, yamlFiles[i].Name()))
		kindJ := getKindFromFile(filepath.Join(dirPath, yamlFiles[j].Name()))
		return constants.GetKindPriority(kindI) < constants.GetKindPriority(kindJ)
	})

	var applied []resourceKey
	for _, file := range yamlFiles {
		manifestPath := filepath.Join(dirPath, file.Name())
		key, _, err := applyManifest(ctx, k8sClient, manifestPath)
		if err != nil {
			logger.Warn("Failed to apply %s: %v", file.Name(), err)
			continue
		}
		if key != nil {
			applied = append(applied, *key)
		}
	}

	return applied, nil
}

// pruneOrphans deletes namespaced resources in the given namespace that are owned
// by osmanage but are no longer present in the applied set.
func pruneOrphans(ctx context.Context, k8sClient *client.Client, namespace string, applied []resourceKey) error {
	desired := make(map[resourceKey]bool, len(applied))
	for _, k := range applied {
		desired[k] = true
	}

	dynamicClient, err := k8sClient.Dynamic()
	if err != nil {
		return fmt.Errorf("getting dynamic client: %w", err)
	}

	groupResources, err := k8sClient.APIGroupResources()
	if err != nil {
		return fmt.Errorf("getting API group resources: %w", err)
	}

	for _, group := range groupResources {
		for _, version := range group.Group.Versions {
			for _, resource := range group.VersionedResources[version.Version] {
				if !resource.Namespaced {
					continue
				}

				gvr := schema.GroupVersionResource{
					Group:    group.Group.Name,
					Version:  version.Version,
					Resource: resource.Name,
				}

				list, err := dynamicClient.Resource(gvr).Namespace(namespace).List(ctx, metav1.ListOptions{})
				if err != nil {
					logger.Debug("Skipping %s: %v", gvr.Resource, err)
					continue
				}

				for _, item := range list.Items {
					if desired[resourceKey{gvr: gvr, name: item.GetName()}] {
						continue
					}
					for _, mf := range item.GetManagedFields() {
						if mf.Manager != fieldManager {
							continue
						}
						logger.Info("Pruning orphaned %s: %s", item.GetKind(), item.GetName())
						if err := dynamicClient.Resource(gvr).Namespace(namespace).Delete(
							ctx, item.GetName(), metav1.DeleteOptions{},
						); err != nil {
							logger.Warn("Failed to prune %s/%s: %v", item.GetKind(), item.GetName(), err)
						}
						break
					}
				}
			}
		}
	}

	return nil
}

// getKindFromFile reads the Kind field from a YAML file
func getKindFromFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	var obj unstructured.Unstructured
	if err := yaml.Unmarshal(data, &obj); err != nil {
		return ""
	}

	return obj.GetKind()
}
