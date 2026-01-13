// Package property contains property-based tests for IAM Policy Service.
package property

import (
	"context"
	"testing"

	"github.com/auth-platform/iam-policy-service/tests/testutil"
	"pgregory.net/rapid"
)

// TestAuthorizationRequestResponseConsistency validates Property 8.
// gRPC request-response must be consistent.
func TestAuthorizationRequestResponseConsistency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		subjectID := testutil.NonEmptyStringGen().Draw(t, "subjectID")
		resourceID := testutil.NonEmptyStringGen().Draw(t, "resourceID")
		resourceType := rapid.SampledFrom([]string{"document", "user", "project"}).Draw(t, "resourceType")
		action := rapid.SampledFrom([]string{"read", "write", "delete"}).Draw(t, "action")

		// Create mock authorization service
		authService := testutil.NewMockAuthorizationService()
		ctx := context.Background()

		req := testutil.AuthorizationRequest{
			SubjectID:    subjectID,
			ResourceID:   resourceID,
			ResourceType: resourceType,
			Action:       action,
		}

		// Authorize
		resp, err := authService.Authorize(ctx, req)
		if err != nil {
			t.Fatalf("authorization failed: %v", err)
		}

		// Property: response must have valid structure
		if resp.EvaluatedAt.IsZero() {
			t.Error("EvaluatedAt should not be zero")
		}

		// Property: reason must be non-empty
		if resp.Reason == "" {
			t.Error("Reason should not be empty")
		}
	})
}

// TestBatchAuthorizationConsistency validates batch authorization.
func TestBatchAuthorizationConsistency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		numRequests := rapid.IntRange(1, 10).Draw(t, "numRequests")

		authService := testutil.NewMockAuthorizationService()
		ctx := context.Background()

		requests := make([]testutil.AuthorizationRequest, numRequests)
		for i := 0; i < numRequests; i++ {
			requests[i] = testutil.AuthorizationRequest{
				SubjectID:    testutil.NonEmptyStringGen().Draw(t, "subjectID"),
				ResourceID:   testutil.NonEmptyStringGen().Draw(t, "resourceID"),
				ResourceType: "document",
				Action:       rapid.SampledFrom([]string{"read", "write"}).Draw(t, "action"),
			}
		}

		// Batch authorize
		responses, err := authService.BatchAuthorize(ctx, requests)
		if err != nil {
			t.Fatalf("batch authorization failed: %v", err)
		}

		// Property: response count must match request count
		if len(responses) != len(requests) {
			t.Errorf("response count mismatch: expected %d, got %d", len(requests), len(responses))
		}

		// Property: each response must be valid
		for i, resp := range responses {
			if resp.EvaluatedAt.IsZero() {
				t.Errorf("response[%d].EvaluatedAt should not be zero", i)
			}
		}
	})
}

// TestAuthorizationDeterminism validates that same request produces same result.
func TestAuthorizationDeterminism(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		req := testutil.AuthorizationRequest{
			SubjectID:    testutil.NonEmptyStringGen().Draw(t, "subjectID"),
			ResourceID:   testutil.NonEmptyStringGen().Draw(t, "resourceID"),
			ResourceType: "document",
			Action:       "read",
		}

		authService := testutil.NewMockAuthorizationService()
		ctx := context.Background()

		// Authorize multiple times
		results := make([]bool, 5)
		for i := 0; i < 5; i++ {
			resp, _ := authService.Authorize(ctx, req)
			results[i] = resp.Allowed
		}

		// Property: all results must be identical
		for i := 1; i < len(results); i++ {
			if results[i] != results[0] {
				t.Errorf("non-deterministic: result[0]=%v, result[%d]=%v", results[0], i, results[i])
			}
		}
	})
}

// TestGetPermissionsCompleteness validates permission retrieval.
func TestGetPermissionsCompleteness(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		roles := rapid.SliceOfN(
			rapid.SampledFrom([]string{"admin", "editor", "viewer"}),
			1, 3,
		).Draw(t, "roles")

		authService := testutil.NewMockAuthorizationService()
		ctx := context.Background()

		permissions, err := authService.GetPermissions(ctx, "user1", roles)
		if err != nil {
			t.Fatalf("GetPermissions failed: %v", err)
		}

		// Property: permissions should be returned (may be empty for unknown roles)
		if permissions == nil {
			t.Error("permissions should not be nil")
		}
	})
}
