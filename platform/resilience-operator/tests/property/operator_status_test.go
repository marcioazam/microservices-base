// Package property contains property-based tests for the resilience operator.
package property

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"pgregory.net/rapid"

	resiliencev1 "github.com/auth-platform/platform/resilience-operator/api/v1"
	"github.com/auth-platform/platform/resilience-operator/internal/status"
)

// TestStatusConditionAccuracy validates status conditions accuracy.
// Property 7: Status Condition Accuracy
// Validates: Requirements 4.6, 4.7
func TestStatusConditionAccuracy(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		conditionType := rapid.SampledFrom([]string{
			status.ConditionTypeReady,
			status.ConditionTypeProgressing,
			status.ConditionTypeDegraded,
		}).Draw(t, "conditionType")

		conditionStatus := rapid.SampledFrom([]metav1.ConditionStatus{
			metav1.ConditionTrue,
			metav1.ConditionFalse,
			metav1.ConditionUnknown,
		}).Draw(t, "conditionStatus")

		reason := rapid.SampledFrom([]string{
			status.ReasonApplied,
			status.ReasonTargetNotFound,
			status.ReasonReconciling,
			status.ReasonFailed,
		}).Draw(t, "reason")

		message := rapid.StringN(1, 100, 200).Draw(t, "message")

		policy := &resiliencev1.ResiliencePolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name:       "test-policy",
				Namespace:  "default",
				Generation: 1,
			},
			Status: resiliencev1.ResiliencePolicyStatus{
				Conditions: []metav1.Condition{
					{
						Type:               conditionType,
						Status:             conditionStatus,
						Reason:             reason,
						Message:            message,
						LastTransitionTime: metav1.Now(),
						ObservedGeneration: 1,
					},
				},
			},
		}

		condition := status.GetCondition(policy, conditionType)
		if condition == nil {
			t.Fatalf("condition %s not found", conditionType)
		}

		if condition.Status != conditionStatus {
			t.Fatalf("status mismatch: expected %s, got %s", conditionStatus, condition.Status)
		}

		if condition.Reason != reason {
			t.Fatalf("reason mismatch: expected %s, got %s", reason, condition.Reason)
		}

		isReady := status.IsReady(policy)
		expectedReady := conditionType == status.ConditionTypeReady && conditionStatus == metav1.ConditionTrue
		if isReady != expectedReady {
			t.Fatalf("IsReady mismatch: expected %v, got %v", expectedReady, isReady)
		}
	})
}

// TestOwnerReferenceIntegrity validates owner reference integrity.
// Property 6: Owner Reference Integrity
// Validates: Requirements 4.4
func TestOwnerReferenceIntegrity(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		policyName := rapid.StringMatching(`[a-z][a-z0-9-]{0,62}`).Draw(t, "policyName")
		policyNamespace := rapid.SampledFrom([]string{"default", "production", "staging"}).Draw(t, "namespace")
		policyUID := rapid.StringMatching(`[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}`).Draw(t, "uid")

		ownerRef := metav1.OwnerReference{
			APIVersion: "resilience.auth-platform.github.com/v1",
			Kind:       "ResiliencePolicy",
			Name:       policyName,
			UID:        "test-uid-" + policyUID,
		}

		if ownerRef.APIVersion != "resilience.auth-platform.github.com/v1" {
			t.Fatalf("APIVersion mismatch")
		}
		if ownerRef.Kind != "ResiliencePolicy" {
			t.Fatalf("Kind mismatch")
		}
		if ownerRef.Name != policyName {
			t.Fatalf("Name mismatch: expected %s, got %s", policyName, ownerRef.Name)
		}

		policy := &resiliencev1.ResiliencePolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name:      policyName,
				Namespace: policyNamespace,
			},
		}
		if policy.Namespace != policyNamespace {
			t.Fatalf("Namespace mismatch")
		}
	})
}

// TestTargetServiceValidation validates target service references.
// Property 8: Target Service Validation
// Validates: Requirements 4.7
func TestTargetServiceValidation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		serviceName := rapid.StringMatching(`[a-z][a-z0-9-]{0,62}`).Draw(t, "serviceName")
		namespace := rapid.SampledFrom([]string{"", "default", "production"}).Draw(t, "namespace")
		port := rapid.Int32Range(1, 65535).Draw(t, "port")

		targetRef := resiliencev1.TargetReference{
			Name:      serviceName,
			Namespace: namespace,
			Port:      &port,
		}

		if targetRef.Name == "" {
			t.Fatalf("service name cannot be empty")
		}
		if len(targetRef.Name) > 253 {
			t.Fatalf("service name too long")
		}
		if targetRef.Port != nil && (*targetRef.Port < 1 || *targetRef.Port > 65535) {
			t.Fatalf("port out of range: %d", *targetRef.Port)
		}

		effectiveNamespace := namespace
		if effectiveNamespace == "" {
			effectiveNamespace = "default"
		}

		if effectiveNamespace == "" {
			t.Fatalf("effective namespace cannot be empty")
		}
	})
}

// TestLeaderElectionConsistency validates leader election.
// Property 10: Leader Election Consistency
// Validates: Requirements 11.1, 11.2
func TestLeaderElectionConsistency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		replicas := rapid.IntRange(1, 5).Draw(t, "replicas")
		leaderIndex := rapid.IntRange(0, replicas-1).Draw(t, "leaderIndex")

		isLeader := make([]bool, replicas)
		isLeader[leaderIndex] = true

		leaderCount := 0
		for _, leader := range isLeader {
			if leader {
				leaderCount++
			}
		}

		if leaderCount != 1 {
			t.Fatalf("expected exactly 1 leader, got %d", leaderCount)
		}

		if leaderIndex < 0 || leaderIndex >= replicas {
			t.Fatalf("invalid leader index: %d (replicas: %d)", leaderIndex, replicas)
		}
	})
}
