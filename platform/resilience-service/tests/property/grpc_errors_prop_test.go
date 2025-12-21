package property

import (
	"testing"

	liberror "github.com/auth-platform/libs/go/error"
	"github.com/auth-platform/libs/go/resilience"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"pgregory.net/rapid"
)

// **Feature: resilience-microservice, Property 23: Error to gRPC Status Code Mapping**
// **Validates: Requirements 8.4**
func TestProperty_ErrorToGRPCStatusCodeMapping(t *testing.T) {
	t.Run("circuit_open_maps_to_unavailable", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			service := rapid.StringMatching(`[a-zA-Z][a-zA-Z0-9]{0,29}`).Draw(t, "service")

			err := liberror.NewCircuitOpenError(service)
			grpcErr := liberror.ToGRPCError(err)
			st, ok := status.FromError(grpcErr)
			if !ok {
				t.Fatal("expected gRPC status error")
			}
			if st.Code() != codes.Unavailable {
				t.Fatalf("expected Unavailable, got %v", st.Code())
			}
		})
	})

	t.Run("rate_limit_maps_to_resource_exhausted", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			service := rapid.StringMatching(`[a-zA-Z][a-zA-Z0-9]{0,29}`).Draw(t, "service")

			err := liberror.NewRateLimitError(service, 0)
			grpcErr := liberror.ToGRPCError(err)
			st, ok := status.FromError(grpcErr)
			if !ok {
				t.Fatal("expected gRPC status error")
			}
			if st.Code() != codes.ResourceExhausted {
				t.Fatalf("expected ResourceExhausted, got %v", st.Code())
			}
		})
	})

	t.Run("timeout_maps_to_deadline_exceeded", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			service := rapid.StringMatching(`[a-zA-Z][a-zA-Z0-9]{0,29}`).Draw(t, "service")

			err := liberror.NewTimeoutError(service, 0)
			grpcErr := liberror.ToGRPCError(err)
			st, ok := status.FromError(grpcErr)
			if !ok {
				t.Fatal("expected gRPC status error")
			}
			if st.Code() != codes.DeadlineExceeded {
				t.Fatalf("expected DeadlineExceeded, got %v", st.Code())
			}
		})
	})

	t.Run("bulkhead_full_maps_to_resource_exhausted", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			partition := rapid.StringMatching(`[a-zA-Z][a-zA-Z0-9]{0,29}`).Draw(t, "partition")

			err := liberror.NewBulkheadFullError(partition)
			grpcErr := liberror.ToGRPCError(err)
			st, ok := status.FromError(grpcErr)
			if !ok {
				t.Fatal("expected gRPC status error")
			}
			if st.Code() != codes.ResourceExhausted {
				t.Fatalf("expected ResourceExhausted, got %v", st.Code())
			}
		})
	})

	t.Run("all_error_codes_have_mapping", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			codeIndex := rapid.IntRange(0, 6).Draw(t, "codeIndex")

			errorCodes := []liberror.ResilienceErrorCode{
				liberror.ErrCircuitOpen,
				liberror.ErrRateLimitExceeded,
				liberror.ErrTimeout,
				liberror.ErrBulkheadFull,
				liberror.ErrRetryExhausted,
				liberror.ErrInvalidPolicy,
				liberror.ErrServiceUnavailable,
			}
			code := errorCodes[codeIndex%len(errorCodes)]
			grpcCode := liberror.ToGRPCCode(code)
			if grpcCode == codes.Unknown {
				t.Fatalf("error code %v should have a gRPC mapping", code)
			}
		})
	})
}

// **Feature: resilience-microservice, Property 22: Circuit State Retrieval Consistency**
// **Validates: Requirements 8.2**
func TestProperty_CircuitStateRetrievalConsistency(t *testing.T) {
	t.Run("state_retrieval_matches_internal_state", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			stateInt := rapid.IntRange(0, 2).Draw(t, "stateInt")

			states := []resilience.CircuitState{
				resilience.StateClosed,
				resilience.StateOpen,
				resilience.StateHalfOpen,
			}
			state := states[stateInt%len(states)]
			stateStr := state.String()

			switch state {
			case resilience.StateClosed:
				if stateStr != "CLOSED" {
					t.Fatalf("expected CLOSED, got %s", stateStr)
				}
			case resilience.StateOpen:
				if stateStr != "OPEN" {
					t.Fatalf("expected OPEN, got %s", stateStr)
				}
			case resilience.StateHalfOpen:
				if stateStr != "HALF_OPEN" {
					t.Fatalf("expected HALF_OPEN, got %s", stateStr)
				}
			}
		})
	})
}
