package errors

import (
	"testing"
	"time"

	resilienceerrors "github.com/auth-platform/libs/go/resilience/errors"

	"google.golang.org/grpc/codes"
)

func TestToGRPCCode(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected codes.Code
	}{
		{
			name:     "nil error returns OK",
			err:      nil,
			expected: codes.OK,
		},
		{
			name:     "circuit open returns Unavailable",
			err:      resilienceerrors.NewCircuitOpenError("test"),
			expected: codes.Unavailable,
		},
		{
			name:     "rate limit returns ResourceExhausted",
			err:      resilienceerrors.NewRateLimitError("test", time.Second),
			expected: codes.ResourceExhausted,
		},
		{
			name:     "timeout returns DeadlineExceeded",
			err:      resilienceerrors.NewTimeoutError("test", time.Second),
			expected: codes.DeadlineExceeded,
		},
		{
			name:     "bulkhead full returns ResourceExhausted",
			err:      resilienceerrors.NewBulkheadFullError("test", "partition"),
			expected: codes.ResourceExhausted,
		},
		{
			name:     "invalid policy returns InvalidArgument",
			err:      resilienceerrors.NewInvalidPolicyError("test", "policy", "field", "reason"),
			expected: codes.InvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := ToGRPCCode(tt.err)
			if code != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, code)
			}
		})
	}
}

func TestToGRPCError(t *testing.T) {
	t.Run("nil error returns nil", func(t *testing.T) {
		if ToGRPCError(nil) != nil {
			t.Error("expected nil")
		}
	})

	t.Run("converts to gRPC error", func(t *testing.T) {
		err := resilienceerrors.NewCircuitOpenError("test")
		grpcErr := ToGRPCError(err)
		if grpcErr == nil {
			t.Error("expected gRPC error")
		}
	})
}

func TestToGRPCStatus(t *testing.T) {
	t.Run("nil error returns OK status", func(t *testing.T) {
		st := ToGRPCStatus(nil)
		if st.Code() != codes.OK {
			t.Errorf("expected OK, got %v", st.Code())
		}
	})

	t.Run("converts to gRPC status", func(t *testing.T) {
		err := resilienceerrors.NewTimeoutError("test", time.Second)
		st := ToGRPCStatus(err)
		if st.Code() != codes.DeadlineExceeded {
			t.Errorf("expected DeadlineExceeded, got %v", st.Code())
		}
	})
}

func TestFromGRPCCode(t *testing.T) {
	tests := []struct {
		code     codes.Code
		expected resilienceerrors.ErrorCode
	}{
		{codes.Unavailable, resilienceerrors.ErrCircuitOpen},
		{codes.ResourceExhausted, resilienceerrors.ErrRateLimitExceeded},
		{codes.DeadlineExceeded, resilienceerrors.ErrTimeout},
		{codes.InvalidArgument, resilienceerrors.ErrInvalidPolicy},
		{codes.Internal, ""},
	}

	for _, tt := range tests {
		t.Run(tt.code.String(), func(t *testing.T) {
			result := FromGRPCCode(tt.code)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestIsHelpers(t *testing.T) {
	t.Run("IsUnavailable", func(t *testing.T) {
		err := resilienceerrors.NewCircuitOpenError("test")
		if !IsUnavailable(err) {
			t.Error("expected true")
		}
	})

	t.Run("IsResourceExhausted", func(t *testing.T) {
		err := resilienceerrors.NewRateLimitError("test", time.Second)
		if !IsResourceExhausted(err) {
			t.Error("expected true")
		}
	})

	t.Run("IsDeadlineExceeded", func(t *testing.T) {
		err := resilienceerrors.NewTimeoutError("test", time.Second)
		if !IsDeadlineExceeded(err) {
			t.Error("expected true")
		}
	})

	t.Run("IsInvalidArgument", func(t *testing.T) {
		err := resilienceerrors.NewInvalidPolicyError("test", "policy", "field", "reason")
		if !IsInvalidArgument(err) {
			t.Error("expected true")
		}
	})
}
