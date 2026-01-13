// Package status provides status management for ResiliencePolicy resources.
package status

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	resiliencev1 "github.com/auth-platform/platform/resilience-operator/api/v1"
)

// Condition types
const (
	ConditionTypeReady       = "Ready"
	ConditionTypeProgressing = "Progressing"
	ConditionTypeDegraded    = "Degraded"
)

// Condition reasons
const (
	ReasonApplied        = "Applied"
	ReasonTargetNotFound = "TargetServiceNotFound"
	ReasonReconciling    = "Reconciling"
	ReasonFailed         = "Failed"
	ReasonDeleting       = "Deleting"
)

// Manager handles status updates for ResiliencePolicy resources.
type Manager struct {
	client client.Client
}

// NewManager creates a new status Manager.
func NewManager(c client.Client) *Manager {
	return &Manager{client: c}
}

// SetReady sets the policy status to Ready.
func (m *Manager) SetReady(ctx context.Context, policy *resiliencev1.ResiliencePolicy) error {
	return m.setCondition(ctx, policy, metav1.Condition{
		Type:    ConditionTypeReady,
		Status:  metav1.ConditionTrue,
		Reason:  ReasonApplied,
		Message: "Resilience policy successfully applied",
	})
}

// SetFailed sets the policy status to Failed.
func (m *Manager) SetFailed(ctx context.Context, policy *resiliencev1.ResiliencePolicy, reason, message string) error {
	return m.setCondition(ctx, policy, metav1.Condition{
		Type:    ConditionTypeReady,
		Status:  metav1.ConditionFalse,
		Reason:  reason,
		Message: message,
	})
}

// SetTargetNotFound sets the policy status to TargetServiceNotFound.
func (m *Manager) SetTargetNotFound(ctx context.Context, policy *resiliencev1.ResiliencePolicy, target string) error {
	return m.setCondition(ctx, policy, metav1.Condition{
		Type:    ConditionTypeReady,
		Status:  metav1.ConditionFalse,
		Reason:  ReasonTargetNotFound,
		Message: fmt.Sprintf("Target service %s not found", target),
	})
}

// SetReconciling sets the policy status to Reconciling.
func (m *Manager) SetReconciling(ctx context.Context, policy *resiliencev1.ResiliencePolicy) error {
	return m.setCondition(ctx, policy, metav1.Condition{
		Type:    ConditionTypeReady,
		Status:  metav1.ConditionUnknown,
		Reason:  ReasonReconciling,
		Message: "Reconciliation in progress",
	})
}

// setCondition updates a condition on the policy status.
func (m *Manager) setCondition(ctx context.Context, policy *resiliencev1.ResiliencePolicy, condition metav1.Condition) error {
	condition.LastTransitionTime = metav1.Now()
	condition.ObservedGeneration = policy.Generation

	found := false
	for i := range policy.Status.Conditions {
		if policy.Status.Conditions[i].Type == condition.Type {
			if policy.Status.Conditions[i].Status != condition.Status ||
				policy.Status.Conditions[i].Reason != condition.Reason {
				policy.Status.Conditions[i] = condition
			}
			found = true
			break
		}
	}
	if !found {
		policy.Status.Conditions = append(policy.Status.Conditions, condition)
	}

	policy.Status.ObservedGeneration = policy.Generation
	now := metav1.Now()
	policy.Status.LastUpdateTime = &now

	targetNamespace := policy.Spec.TargetRef.Namespace
	if targetNamespace == "" {
		targetNamespace = policy.Namespace
	}
	policy.Status.AppliedToServices = []string{
		fmt.Sprintf("%s/%s", targetNamespace, policy.Spec.TargetRef.Name),
	}

	return m.client.Status().Update(ctx, policy)
}

// GetCondition returns a condition by type.
func GetCondition(policy *resiliencev1.ResiliencePolicy, conditionType string) *metav1.Condition {
	for i := range policy.Status.Conditions {
		if policy.Status.Conditions[i].Type == conditionType {
			return &policy.Status.Conditions[i]
		}
	}
	return nil
}

// IsReady returns true if the policy is in Ready state.
func IsReady(policy *resiliencev1.ResiliencePolicy) bool {
	condition := GetCondition(policy, ConditionTypeReady)
	return condition != nil && condition.Status == metav1.ConditionTrue
}
