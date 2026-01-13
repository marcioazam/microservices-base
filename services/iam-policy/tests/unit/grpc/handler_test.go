// Package grpc contains unit tests for gRPC handlers.
package grpc

import (
	"context"
	"testing"

	"github.com/auth-platform/iam-policy-service/tests/testutil"
)

func TestAuthorize_ValidRequest(t *testing.T) {
	handler := testutil.NewMockGRPCHandler()
	ctx := context.Background()

	req := &testutil.AuthorizeRequest{
		SubjectID:    "user123",
		ResourceType: "document",
		ResourceID:   "doc456",
		Action:       "read",
	}

	resp, err := handler.Authorize(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp == nil {
		t.Fatal("response should not be nil")
	}
}

func TestAuthorize_MissingSubjectID(t *testing.T) {
	handler := testutil.NewMockGRPCHandler()
	ctx := context.Background()

	req := &testutil.AuthorizeRequest{
		ResourceType: "document",
		Action:       "read",
	}

	_, err := handler.Authorize(ctx, req)
	if err == nil {
		t.Error("expected error for missing subject_id")
	}
}

func TestAuthorize_MissingAction(t *testing.T) {
	handler := testutil.NewMockGRPCHandler()
	ctx := context.Background()

	req := &testutil.AuthorizeRequest{
		SubjectID:    "user123",
		ResourceType: "document",
	}

	_, err := handler.Authorize(ctx, req)
	if err == nil {
		t.Error("expected error for missing action")
	}
}

func TestBatchAuthorize_ValidRequests(t *testing.T) {
	handler := testutil.NewMockGRPCHandler()
	ctx := context.Background()

	requests := []*testutil.AuthorizeRequest{
		{SubjectID: "user1", ResourceType: "doc", Action: "read"},
		{SubjectID: "user2", ResourceType: "doc", Action: "write"},
	}

	responses, err := handler.BatchAuthorize(ctx, requests)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(responses) != len(requests) {
		t.Errorf("expected %d responses, got %d", len(requests), len(responses))
	}
}

func TestBatchAuthorize_EmptyRequests(t *testing.T) {
	handler := testutil.NewMockGRPCHandler()
	ctx := context.Background()

	_, err := handler.BatchAuthorize(ctx, []*testutil.AuthorizeRequest{})
	if err == nil {
		t.Error("expected error for empty requests")
	}
}

func TestBatchAuthorize_TooManyRequests(t *testing.T) {
	handler := testutil.NewMockGRPCHandler()
	ctx := context.Background()

	requests := make([]*testutil.AuthorizeRequest, 101)
	for i := range requests {
		requests[i] = &testutil.AuthorizeRequest{
			SubjectID:    "user",
			ResourceType: "doc",
			Action:       "read",
		}
	}

	_, err := handler.BatchAuthorize(ctx, requests)
	if err == nil {
		t.Error("expected error for too many requests")
	}
}

func TestGetUserPermissions_ValidRequest(t *testing.T) {
	handler := testutil.NewMockGRPCHandler()
	ctx := context.Background()

	permissions, err := handler.GetUserPermissions(ctx, "user123", []string{"admin"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if permissions == nil {
		t.Error("permissions should not be nil")
	}
}

func TestGetUserPermissions_MissingUserID(t *testing.T) {
	handler := testutil.NewMockGRPCHandler()
	ctx := context.Background()

	_, err := handler.GetUserPermissions(ctx, "", []string{"admin"})
	if err == nil {
		t.Error("expected error for missing user_id")
	}
}

func TestReloadPolicies(t *testing.T) {
	handler := testutil.NewMockGRPCHandler()
	ctx := context.Background()

	success, err := handler.ReloadPolicies(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !success {
		t.Error("reload should succeed")
	}
}
