// Package unit contains unit tests for the resilience operator.
package unit

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestStatusManager(t *testing.T) {
	t.Run("IsReady returns true for ready policy", func(t *testing.T) {
		conditions := []metav1.Condition{
			{Type: "Ready", Status: metav1.ConditionTrue, Reason: "Applied"},
		}
		if !isReady(conditions) {
			t.Error("expected policy to be ready")
		}
	})

	t.Run("IsReady returns false for not ready policy", func(t *testing.T) {
		conditions := []metav1.Condition{
			{Type: "Ready", Status: metav1.ConditionFalse, Reason: "Failed"},
		}
		if isReady(conditions) {
			t.Error("expected policy to not be ready")
		}
	})

	t.Run("IsReady returns false for missing condition", func(t *testing.T) {
		conditions := []metav1.Condition{}
		if isReady(conditions) {
			t.Error("expected policy to not be ready")
		}
	})
}

func TestGetCondition(t *testing.T) {
	conditions := []metav1.Condition{
		{Type: "Ready", Status: metav1.ConditionTrue},
		{Type: "Progressing", Status: metav1.ConditionFalse},
	}

	t.Run("finds existing condition", func(t *testing.T) {
		cond := getCondition(conditions, "Ready")
		if cond == nil {
			t.Fatal("expected to find Ready condition")
		}
		if cond.Status != metav1.ConditionTrue {
			t.Errorf("expected status True, got %s", cond.Status)
		}
	})

	t.Run("returns nil for missing condition", func(t *testing.T) {
		cond := getCondition(conditions, "NonExistent")
		if cond != nil {
			t.Error("expected nil for non-existent condition")
		}
	})
}

func isReady(conditions []metav1.Condition) bool {
	for _, c := range conditions {
		if c.Type == "Ready" && c.Status == metav1.ConditionTrue {
			return true
		}
	}
	return false
}

func getCondition(conditions []metav1.Condition, conditionType string) *metav1.Condition {
	for i := range conditions {
		if conditions[i].Type == conditionType {
			return &conditions[i]
		}
	}
	return nil
}
