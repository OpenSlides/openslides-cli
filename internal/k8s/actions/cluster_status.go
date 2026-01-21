package actions

import (
	"context"
	"fmt"

	"github.com/OpenSlides/openslides-cli/internal/k8s/client"
	"github.com/OpenSlides/openslides-cli/internal/logger"
	"github.com/spf13/cobra"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ClusterStatusHelp      = "Check Kubernetes cluster status"
	ClusterStatusHelpExtra = `Checks the status of the Kubernetes cluster by querying node conditions.
Reports the total number of nodes and how many are in Ready state.

Examples:
  osmanage k8s cluster-status
  osmanage k8s cluster-status --kubeconfig ~/.kube/config`
)

type NodeStatus struct {
	Name       string
	Ready      bool
	Conditions []corev1.NodeCondition
}

type ClusterStatus struct {
	TotalNodes int
	ReadyNodes int
	Nodes      []NodeStatus
}

func ClusterStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cluster-status",
		Short: ClusterStatusHelp,
		Long:  ClusterStatusHelp + "\n\n" + ClusterStatusHelpExtra,
		Args:  cobra.NoArgs,
	}

	kubeconfig := cmd.Flags().String("kubeconfig", "", "Path to kubeconfig file")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		logger.Info("=== K8S CLUSTER STATUS ===")

		k8sClient, err := client.New(*kubeconfig)
		if err != nil {
			return fmt.Errorf("creating k8s client: %w", err)
		}

		ctx := context.Background()

		status, err := checkClusterStatus(ctx, k8sClient)
		if err != nil {
			return fmt.Errorf("checking cluster status: %w", err)
		}

		logger.Info("Total nodes: %d", status.TotalNodes)
		logger.Info("Ready nodes: %d", status.ReadyNodes)

		for _, node := range status.Nodes {
			statusStr := "NotReady"
			if node.Ready {
				statusStr = "Ready"
			}
			logger.Info("Node %s: %s", node.Name, statusStr)

			if !node.Ready {
				for _, condition := range node.Conditions {
					if condition.Status == corev1.ConditionTrue && condition.Type != corev1.NodeReady {
						logger.Debug("  - %s: %s (Reason: %s)", condition.Type, condition.Message, condition.Reason)
					}
				}
			}
		}

		if status.ReadyNodes < status.TotalNodes {
			return fmt.Errorf("cluster is not healthy: %d/%d nodes ready", status.ReadyNodes, status.TotalNodes)
		}

		logger.Info("Cluster is healthy ✓")
		return nil
	}

	return cmd
}

// checkClusterStatus retrieves and analyzes the cluster status
func checkClusterStatus(ctx context.Context, k8sClient *client.Client) (*ClusterStatus, error) {
	clientset := k8sClient.Clientset()

	nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing nodes: %w", err)
	}

	status := &ClusterStatus{
		TotalNodes: len(nodes.Items),
		Nodes:      make([]NodeStatus, 0, len(nodes.Items)),
	}

	for _, node := range nodes.Items {
		nodeStatus := NodeStatus{
			Name:       node.Name,
			Ready:      isNodeReady(&node),
			Conditions: node.Status.Conditions,
		}

		if nodeStatus.Ready {
			status.ReadyNodes++
		}

		status.Nodes = append(status.Nodes, nodeStatus)
	}

	return status, nil
}

// isNodeReady checks if a node is in Ready state
func isNodeReady(node *corev1.Node) bool {
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}

// GetNodeCondition retrieves a specific condition from a node
func GetNodeCondition(node *corev1.Node, conditionType corev1.NodeConditionType) *corev1.NodeCondition {
	for i := range node.Status.Conditions {
		if node.Status.Conditions[i].Type == conditionType {
			return &node.Status.Conditions[i]
		}
	}
	return nil
}

// IsNodeHealthy checks if a node has any problematic conditions
func IsNodeHealthy(node *corev1.Node) bool {
	readyCondition := GetNodeCondition(node, corev1.NodeReady)
	if readyCondition == nil || readyCondition.Status != corev1.ConditionTrue {
		return false
	}

	negativeConditions := []corev1.NodeConditionType{
		corev1.NodeMemoryPressure,
		corev1.NodeDiskPressure,
		corev1.NodePIDPressure,
		corev1.NodeNetworkUnavailable,
	}

	for _, condType := range negativeConditions {
		condition := GetNodeCondition(node, condType)
		if condition != nil && condition.Status == corev1.ConditionTrue {
			return false
		}
	}

	return true
}
