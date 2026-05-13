package actions

import (
	"context"
	"fmt"
	"strings"

	"os"
	"text/tabwriter"

	"github.com/OpenSlides/openslides-cli/internal/k8s/client"
	"github.com/OpenSlides/openslides-cli/internal/logger"
	"github.com/OpenSlides/openslides-cli/internal/utils"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	StatusHelp      = "Get current status of an OpenSlides instance"
	StatusHelpExtra = `Prints pod and container status for a running OpenSlides instance.

Examples:
  osmanage k8s status ./my.instance.dir.org
  osmanage k8s status ./my.instance.dir.org --kubeconfig ~/.kube/config`
)

func GetInstanceStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status <instance-dir>",
		Short: StatusHelp,
		Long:  StatusHelp + "\n\n" + StatusHelpExtra,
		Args:  cobra.ExactArgs(1),
	}

	kubeconfig := cmd.Flags().String("kubeconfig", "", "Path to kubeconfig file")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		logger.Info("=== K8S INSTANCE STATUS ===")
		instanceDir := args[0]
		namespace := utils.ExtractNamespace(instanceDir)

		k8sClient, err := client.New(*kubeconfig)
		if err != nil {
			return fmt.Errorf("creating k8s client: %w", err)
		}

		status, err := GetInstanceStatus(context.Background(), k8sClient, namespace)
		if err != nil {
			return fmt.Errorf("getting instance status: %w", err)
		}

		if !status.NamespaceExists {
			fmt.Println("Unavailable")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		if _, err := fmt.Fprintln(w, "NAMESPACE\tSERVICE\tPOD\tREGISTRY\tTAG\tREADY\tSTARTED\tNODE"); err != nil {
			return fmt.Errorf("writing header: %w", err)
		}
		for _, pod := range status.Pods {
			for _, container := range pod.Containers {
				registry := container.ContainerRegistry
				if idx := strings.LastIndex(registry, "/"); idx != -1 {
					registry = registry[:idx]
				}
				if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%v\t%v\t%s\n",
					namespace,
					pod.Service,
					pod.Name,
					registry,
					container.Tag,
					container.Ready,
					container.Started,
					pod.Node,
				); err != nil {
					return fmt.Errorf("writing row: %w", err)
				}
			}
		}
		if err := w.Flush(); err != nil {
			return fmt.Errorf("flushing output: %w", err)
		}
		return nil
	}

	return cmd
}

type ContainerStatus struct {
	Name              string
	Tag               string
	ContainerRegistry string
	Ready             bool
	Started           bool
	EnvVars           map[string]string
}

type PodStatus struct {
	Name       string
	Service    string
	Node       string
	Containers []ContainerStatus
}

type InstanceStatus struct {
	NamespaceExists bool
	Pods            []PodStatus
	ServiceCounts   map[string]int32
}

func GetInstanceStatus(ctx context.Context, k8sClient *client.Client, namespace string) (*InstanceStatus, error) {
	clientset := k8sClient.Clientset()

	// check namespace exists
	_, err := clientset.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		return &InstanceStatus{NamespaceExists: false}, nil
	}

	podList, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing pods: %w", err)
	}

	var pods []PodStatus
	serviceCounts := make(map[string]int32)

	for _, pod := range podList.Items {
		// skip terminating pods
		if pod.DeletionTimestamp != nil {
			continue
		}

		service := pod.Labels["osinstance/service"]
		serviceCounts[service]++

		var containers []ContainerStatus
		for _, cs := range pod.Status.ContainerStatuses {
			if cs.Name == "redis" {
				continue
			}
			registryService, tag, _ := strings.Cut(cs.Image, ":")
			registry := registryService[:strings.LastIndex(registryService, "/")]

			envVars := make(map[string]string)
			for _, spec := range pod.Spec.Containers {
				if spec.Name != cs.Name {
					continue
				}
				for _, env := range spec.Env {
					envVars[env.Name] = env.Value
				}
			}

			started := cs.Started != nil && *cs.Started
			containers = append(containers, ContainerStatus{
				Name:              cs.Name,
				Tag:               tag,
				ContainerRegistry: registry,
				Ready:             cs.Ready,
				Started:           started,
				EnvVars:           envVars,
			})
		}

		pods = append(pods, PodStatus{
			Name:       pod.Name,
			Service:    service,
			Node:       pod.Spec.NodeName,
			Containers: containers,
		})
	}

	return &InstanceStatus{
		NamespaceExists: true,
		Pods:            pods,
		ServiceCounts:   serviceCounts,
	}, nil
}
