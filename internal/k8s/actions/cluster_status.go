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
	ClusterStatusHelpExtra = `Checks the health of all nodes in the Kubernetes cluster.

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

		fmt.Printf("cluster_status: %d %d\n", status.TotalNodes, status.ReadyNodes)

		logger.Info("Total nodes: %d", status.TotalNodes)
		logger.Info("Ready nodes: %d", status.ReadyNodes)

		for _, node := range status.Nodes {
			if node.Ready {
				logger.Info("Node %s: Ready", node.Name)
			} else {
				logger.Info("Node %s: NotReady", node.Name)
				for _, cond := range node.Conditions {
					if cond.Status == corev1.ConditionTrue && cond.Type != corev1.NodeReady {
						logger.Debug("  - %s: %s (Reason: %s)", cond.Type, cond.Status, cond.Reason)
					}
				}
			}
		}

		if status.ReadyNodes != status.TotalNodes {
			return fmt.Errorf("cluster is not healthy: %d/%d nodes ready", status.ReadyNodes, status.TotalNodes)
		}

		logger.Info("Cluster is healthy")
		return nil
	}

	return cmd
}

// checkClusterStatus checks the overall cluster health
func checkClusterStatus(ctx context.Context, k8sClient *client.Client) (*ClusterStatus, error) {
	nodes, err := k8sClient.Clientset().CoreV1().Nodes().List(ctx, metav1.ListOptions{})
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

// isNodeReady checks if a node is ready
func isNodeReady(node *corev1.Node) bool {
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}

// IsNodeHealthy checks if a node is healthy (no pressure conditions)
func IsNodeHealthy(node *corev1.Node) bool {
	if !isNodeReady(node) {
		return false
	}

	pressureTypes := []corev1.NodeConditionType{
		corev1.NodeMemoryPressure,
		corev1.NodeDiskPressure,
		corev1.NodePIDPressure,
		corev1.NodeNetworkUnavailable,
	}

	for _, pressureType := range pressureTypes {
		condition := GetNodeCondition(node, pressureType)
		if condition != nil && condition.Status == corev1.ConditionTrue {
			return false
		}
	}

	return true
}

// GetNodeCondition retrieves a specific condition from a node
func GetNodeCondition(node *corev1.Node, conditionType corev1.NodeConditionType) *corev1.NodeCondition {
	for _, condition := range node.Status.Conditions {
		if condition.Type == conditionType {
			return &condition
		}
	}
	return nil
}
