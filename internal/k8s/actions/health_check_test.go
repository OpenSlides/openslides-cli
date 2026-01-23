package actions

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestIsPodReady_Ready(t *testing.T) {
	pod := &corev1.Pod{
		Status: corev1.PodStatus{
			Conditions: []corev1.PodCondition{
				{Type: corev1.PodReady, Status: corev1.ConditionTrue},
			},
		},
	}
	if !isPodReady(pod) {
		t.Error("Expected pod to be ready")
	}
}

func TestIsPodReady_NotReady(t *testing.T) {
	pod := &corev1.Pod{
		Status: corev1.PodStatus{
			Conditions: []corev1.PodCondition{
				{Type: corev1.PodReady, Status: corev1.ConditionFalse},
			},
		},
	}
	if isPodReady(pod) {
		t.Error("Expected pod to not be ready")
	}
}

func TestIsPodReady_NoCondition(t *testing.T) {
	pod := &corev1.Pod{
		Status: corev1.PodStatus{Conditions: []corev1.PodCondition{}},
	}
	if isPodReady(pod) {
		t.Error("Expected pod to not be ready when no Ready condition exists")
	}
}

func TestIsPodReady_MultipleConditions(t *testing.T) {
	pod := &corev1.Pod{
		Status: corev1.PodStatus{
			Conditions: []corev1.PodCondition{
				{Type: corev1.PodScheduled, Status: corev1.ConditionTrue},
				{Type: corev1.PodReady, Status: corev1.ConditionTrue},
			},
		},
	}
	if !isPodReady(pod) {
		t.Error("Expected pod to be ready even with multiple conditions")
	}
}
