package grpc

import (
	"testing"

	"github.com/auth-platform/platform/resilience-service/internal/domain"
	"github.com/auth-platform/platform/resilience-service/internal/testutil"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// **Feature: resilience-microservice, Property 23: Error to gRPC Status Code Mapping**
// **Validates: Requirements 8.4**
func TestProperty_ErrorToGRPCStatusCodeMapping(t *testing.T) {
	params := testutil.DefaultTestParameters()
	props := gopter.NewProperties(params)

	props.Property("circuit_open_maps_to_unavailable", prop.ForAll(
		func(service string) bool {
			err := domain.NewCircuitOpenError(service)
			grpcErr := ToGRPCError(err)

			st, ok := status.FromError(grpcErr)
			if !ok {
				return false
			}

			return st.Code() == codes.Unavailable
		},
		gen.AlphaString(),
	))

	props.Property("rate_limit_maps_to_resource_exhausted", prop.ForAll(
		func(service string) bool {
			err := domain.NewRateLimitError(service, 0)
			grpcErr := ToGRPCError(err)

			st, ok := status.FromError(grpcErr)
			if !ok {
				return false
			}

			return st.Code() == codes.ResourceExhausted
		},
		gen.AlphaString(),
	))

	props.Property("timeout_maps_to_deadline_exceeded", prop.ForAll(
		func(service string) bool {
			err := domain.NewTimeoutError(service, 0)
			grpcErr := ToGRPCError(err)

			st, ok := status.FromError(grpcErr)
			if !ok {
				return false
			}

			return st.Code() == codes.DeadlineExceeded
		},
		gen.AlphaString(),
	))

	props.Property("bulkhead_full_maps_to_resource_exhausted", prop.ForAll(
		func(partition string) bool {
			err := domain.NewBulkheadFullError(partition)
			grpcErr := ToGRPCError(err)

			st, ok := status.FromError(grpcErr)
			if !ok {
				return false
			}

			return st.Code() == codes.ResourceExhausted
		},
		gen.AlphaString(),
	))

	props.Property("retry_exhausted_maps_to_unavailable", prop.ForAll(
		func(service string, attempts int) bool {
			err := domain.NewRetryExhaustedError(service, attempts, nil)
			grpcErr := ToGRPCError(err)

			st, ok := status.FromError(grpcErr)
			if !ok {
				return false
			}

			return st.Code() == codes.Unavailable
		},
		gen.AlphaString(),
		gen.IntRange(1, 10),
	))

	props.Property("invalid_policy_maps_to_invalid_argument", prop.ForAll(
		func(message string) bool {
			err := domain.NewInvalidPolicyError(message)
			grpcErr := ToGRPCError(err)

			st, ok := status.FromError(grpcErr)
			if !ok {
				return false
			}

			return st.Code() == codes.InvalidArgument
		},
		gen.AlphaString(),
	))

	props.Property("all_error_codes_have_mapping", prop.ForAll(
		func(codeIndex int) bool {
			errorCodes := []domain.ErrorCode{
				domain.ErrCircuitOpen,
				domain.ErrRateLimitExceeded,
				domain.ErrTimeout,
				domain.ErrBulkheadFull,
				domain.ErrRetryExhausted,
				domain.ErrInvalidPolicy,
				domain.ErrServiceUnavailable,
			}

			code := errorCodes[codeIndex%len(errorCodes)]
			grpcCode := ToGRPCCode(code)

			// Should not be Unknown (0)
			return grpcCode != codes.Unknown
		},
		gen.IntRange(0, 6),
	))

	props.TestingRun(t)
}

// **Feature: resilience-microservice, Property 22: Circuit State Retrieval Consistency**
// **Validates: Requirements 8.2**
func TestProperty_CircuitStateRetrievalConsistency(t *testing.T) {
	params := testutil.DefaultTestParameters()
	props := gopter.NewProperties(params)

	props.Property("state_retrieval_matches_internal_state", prop.ForAll(
		func(stateInt int) bool {
			// This tests the mapping consistency
			states := []domain.CircuitState{
				domain.StateClosed,
				domain.StateOpen,
				domain.StateHalfOpen,
			}

			state := states[stateInt%len(states)]

			// Verify state string conversion is consistent
			stateStr := state.String()

			switch state {
			case domain.StateClosed:
				return stateStr == "CLOSED"
			case domain.StateOpen:
				return stateStr == "OPEN"
			case domain.StateHalfOpen:
				return stateStr == "HALF_OPEN"
			default:
				return false
			}
		},
		gen.IntRange(0, 2),
	))

	props.TestingRun(t)
}
