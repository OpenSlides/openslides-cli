package actions

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
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

func TestIsNodeReady_MultipleConditions(t *testing.T) {
	node := &corev1.Node{
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{Type: corev1.NodeMemoryPressure, Status: corev1.ConditionFalse},
				{Type: corev1.NodeDiskPressure, Status: corev1.ConditionFalse},
				{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
			},
		},
	}

	if !isNodeReady(node) {
		t.Error("Expected node to be ready even with multiple conditions")
	}
}

func TestIsNodeHealthy_Healthy(t *testing.T) {
	node := &corev1.Node{
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
				{Type: corev1.NodeMemoryPressure, Status: corev1.ConditionFalse},
				{Type: corev1.NodeDiskPressure, Status: corev1.ConditionFalse},
				{Type: corev1.NodePIDPressure, Status: corev1.ConditionFalse},
				{Type: corev1.NodeNetworkUnavailable, Status: corev1.ConditionFalse},
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
				{Type: corev1.NodeReady, Status: corev1.ConditionFalse},
				{Type: corev1.NodeMemoryPressure, Status: corev1.ConditionFalse},
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
				{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
				{Type: corev1.NodeMemoryPressure, Status: corev1.ConditionTrue},
			},
		},
	}

	if IsNodeHealthy(node) {
		t.Error("Expected node to not be healthy with memory pressure")
	}
}

func TestIsNodeHealthy_DiskPressure(t *testing.T) {
	node := &corev1.Node{
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
				{Type: corev1.NodeDiskPressure, Status: corev1.ConditionTrue},
			},
		},
	}

	if IsNodeHealthy(node) {
		t.Error("Expected node to not be healthy with disk pressure")
	}
}

func TestIsNodeHealthy_PIDPressure(t *testing.T) {
	node := &corev1.Node{
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
				{Type: corev1.NodePIDPressure, Status: corev1.ConditionTrue},
			},
		},
	}

	if IsNodeHealthy(node) {
		t.Error("Expected node to not be healthy with PID pressure")
	}
}

func TestIsNodeHealthy_NetworkUnavailable(t *testing.T) {
	node := &corev1.Node{
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
				{Type: corev1.NodeNetworkUnavailable, Status: corev1.ConditionTrue},
			},
		},
	}

	if IsNodeHealthy(node) {
		t.Error("Expected node to not be healthy with network unavailable")
	}
}

func TestGetNodeCondition_Exists(t *testing.T) {
	node := &corev1.Node{
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{
					Type:   corev1.NodeReady,
					Status: corev1.ConditionTrue,
					Reason: "KubeletReady",
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

	if condition.Reason != "KubeletReady" {
		t.Errorf("Expected reason 'KubeletReady', got %v", condition.Reason)
	}
}

func TestGetNodeCondition_NotExists(t *testing.T) {
	node := &corev1.Node{
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
			},
		},
	}

	condition := GetNodeCondition(node, corev1.NodeMemoryPressure)
	if condition != nil {
		t.Error("Expected nil when condition doesn't exist")
	}
}

func TestGetNodeCondition_EmptyConditions(t *testing.T) {
	node := &corev1.Node{
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{},
		},
	}

	condition := GetNodeCondition(node, corev1.NodeReady)
	if condition != nil {
		t.Error("Expected nil when no conditions exist")
	}
}

func TestGetNodeCondition_MultipleConditions(t *testing.T) {
	node := &corev1.Node{
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{Type: corev1.NodeMemoryPressure, Status: corev1.ConditionFalse},
				{Type: corev1.NodeDiskPressure, Status: corev1.ConditionTrue},
				{Type: corev1.NodePIDPressure, Status: corev1.ConditionFalse},
				{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
			},
		},
	}

	// Test finding DiskPressure condition
	condition := GetNodeCondition(node, corev1.NodeDiskPressure)
	if condition == nil {
		t.Fatal("Expected to find DiskPressure condition")
	}

	if condition.Type != corev1.NodeDiskPressure {
		t.Errorf("Expected DiskPressure, got %v", condition.Type)
	}

	if condition.Status != corev1.ConditionTrue {
		t.Errorf("Expected condition status True, got %v", condition.Status)
	}
}
