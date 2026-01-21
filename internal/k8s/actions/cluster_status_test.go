package actions

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestIsNodeReady_Ready(t *testing.T) {
	node := &corev1.Node{
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{
					Type:   corev1.NodeReady,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}

	if !isNodeReady(node) {
		t.Error("Expected node to be ready")
	}
}

func TestIsNodeReady_NotReady(t *testing.T) {
	node := &corev1.Node{
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{
					Type:   corev1.NodeReady,
					Status: corev1.ConditionFalse,
				},
			},
		},
	}

	if isNodeReady(node) {
		t.Error("Expected node to not be ready")
	}
}

func TestIsNodeReady_NoCondition(t *testing.T) {
	node := &corev1.Node{
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{},
		},
	}

	if isNodeReady(node) {
		t.Error("Expected node to not be ready when no Ready condition exists")
	}
}

func TestGetNodeCondition_Exists(t *testing.T) {
	node := &corev1.Node{
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{
					Type:   corev1.NodeReady,
					Status: corev1.ConditionTrue,
				},
				{
					Type:   corev1.NodeMemoryPressure,
					Status: corev1.ConditionFalse,
				},
			},
		},
	}

	condition := GetNodeCondition(node, corev1.NodeReady)
	if condition == nil {
		t.Fatal("Expected to find Ready condition")
	}

	if condition.Type != corev1.NodeReady {
		t.Errorf("Expected condition type %v, got %v", corev1.NodeReady, condition.Type)
	}

	if condition.Status != corev1.ConditionTrue {
		t.Errorf("Expected condition status %v, got %v", corev1.ConditionTrue, condition.Status)
	}
}

func TestGetNodeCondition_NotExists(t *testing.T) {
	node := &corev1.Node{
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{
					Type:   corev1.NodeReady,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}

	condition := GetNodeCondition(node, corev1.NodeDiskPressure)
	if condition != nil {
		t.Error("Expected condition to be nil when not found")
	}
}

func TestIsNodeHealthy_Healthy(t *testing.T) {
	node := &corev1.Node{
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{
					Type:   corev1.NodeReady,
					Status: corev1.ConditionTrue,
				},
				{
					Type:   corev1.NodeMemoryPressure,
					Status: corev1.ConditionFalse,
				},
				{
					Type:   corev1.NodeDiskPressure,
					Status: corev1.ConditionFalse,
				},
				{
					Type:   corev1.NodePIDPressure,
					Status: corev1.ConditionFalse,
				},
			},
		},
	}

	if !IsNodeHealthy(node) {
		t.Error("Expected node to be healthy")
	}
}

func TestIsNodeHealthy_NotReady(t *testing.T) {
	node := &corev1.Node{
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{
					Type:   corev1.NodeReady,
					Status: corev1.ConditionFalse,
				},
				{
					Type:   corev1.NodeMemoryPressure,
					Status: corev1.ConditionFalse,
				},
			},
		},
	}

	if IsNodeHealthy(node) {
		t.Error("Expected node to not be healthy when not ready")
	}
}

func TestIsNodeHealthy_MemoryPressure(t *testing.T) {
	node := &corev1.Node{
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{
					Type:   corev1.NodeReady,
					Status: corev1.ConditionTrue,
				},
				{
					Type:   corev1.NodeMemoryPressure,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}

	if IsNodeHealthy(node) {
		t.Error("Expected node to not be healthy when memory pressure is true")
	}
}

func TestIsNodeHealthy_DiskPressure(t *testing.T) {
	node := &corev1.Node{
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{
					Type:   corev1.NodeReady,
					Status: corev1.ConditionTrue,
				},
				{
					Type:   corev1.NodeDiskPressure,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}

	if IsNodeHealthy(node) {
		t.Error("Expected node to not be healthy when disk pressure is true")
	}
}

func TestCheckClusterStatus_AllNodesReady(t *testing.T) {
	nodes := []corev1.Node{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "node-1"},
			Status: corev1.NodeStatus{
				Conditions: []corev1.NodeCondition{
					{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "node-2"},
			Status: corev1.NodeStatus{
				Conditions: []corev1.NodeCondition{
					{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "node-3"},
			Status: corev1.NodeStatus{
				Conditions: []corev1.NodeCondition{
					{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
				},
			},
		},
	}

	fakeClient := fake.NewClientset(&nodes[0], &nodes[1], &nodes[2])

	ctx := context.Background()

	status, err := checkClusterStatusWithClientset(ctx, fakeClient)
	if err != nil {
		t.Fatalf("checkClusterStatus failed: %v", err)
	}

	if status.TotalNodes != 3 {
		t.Errorf("Expected 3 total nodes, got %d", status.TotalNodes)
	}

	if status.ReadyNodes != 3 {
		t.Errorf("Expected 3 ready nodes, got %d", status.ReadyNodes)
	}

	for i, nodeStatus := range status.Nodes {
		if !nodeStatus.Ready {
			t.Errorf("Expected node %s to be ready", nodes[i].Name)
		}
	}
}

func TestCheckClusterStatus_SomeNodesNotReady(t *testing.T) {
	nodes := []corev1.Node{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "node-1"},
			Status: corev1.NodeStatus{
				Conditions: []corev1.NodeCondition{
					{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "node-2"},
			Status: corev1.NodeStatus{
				Conditions: []corev1.NodeCondition{
					{Type: corev1.NodeReady, Status: corev1.ConditionFalse},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "node-3"},
			Status: corev1.NodeStatus{
				Conditions: []corev1.NodeCondition{
					{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
				},
			},
		},
	}

	fakeClient := fake.NewClientset(&nodes[0], &nodes[1], &nodes[2])

	ctx := context.Background()

	status, err := checkClusterStatusWithClientset(ctx, fakeClient)
	if err != nil {
		t.Fatalf("checkClusterStatus failed: %v", err)
	}

	if status.TotalNodes != 3 {
		t.Errorf("Expected 3 total nodes, got %d", status.TotalNodes)
	}

	if status.ReadyNodes != 2 {
		t.Errorf("Expected 2 ready nodes, got %d", status.ReadyNodes)
	}

	expectedReady := map[string]bool{
		"node-1": true,
		"node-2": false,
		"node-3": true,
	}

	for _, nodeStatus := range status.Nodes {
		expectedStatus, exists := expectedReady[nodeStatus.Name]
		if !exists {
			t.Errorf("Unexpected node: %s", nodeStatus.Name)
			continue
		}

		if nodeStatus.Ready != expectedStatus {
			t.Errorf("Node %s: expected ready=%v, got ready=%v", nodeStatus.Name, expectedStatus, nodeStatus.Ready)
		}
	}
}

func TestCheckClusterStatus_NoNodes(t *testing.T) {
	fakeClient := fake.NewClientset()

	ctx := context.Background()

	status, err := checkClusterStatusWithClientset(ctx, fakeClient)
	if err != nil {
		t.Fatalf("checkClusterStatus failed: %v", err)
	}

	if status.TotalNodes != 0 {
		t.Errorf("Expected 0 total nodes, got %d", status.TotalNodes)
	}

	if status.ReadyNodes != 0 {
		t.Errorf("Expected 0 ready nodes, got %d", status.ReadyNodes)
	}
}

// Helper function to test checkClusterStatus with a fake clientset
func checkClusterStatusWithClientset(ctx context.Context, clientset *fake.Clientset) (*ClusterStatus, error) {
	nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
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
