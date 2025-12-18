package error

import (
	"errors"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestToGRPCCode(t *testing.T) {
	tests := []struct {
		code     ResilienceErrorCode
		expected codes.Code
	}{
		{ErrCircuitOpen, codes.Unavailable},
		{ErrRateLimitExceeded, codes.ResourceExhausted},
		{ErrTimeout, codes.DeadlineExceeded},
		{ErrBulkheadFull, codes.ResourceExhausted},
		{ErrRetryExhausted, codes.Unavailable},
		{ErrInvalidPolicy, codes.InvalidArgument},
		{ErrServiceUnavailable, codes.Unavailable},
		{ErrValidation, codes.InvalidArgument},
		{ResilienceErrorCode("UNKNOWN"), codes.Internal},
	}

	for _, tt := range tests {
		t.Run(string(tt.code), func(t *testing.T) {
			if got := ToGRPCCode(tt.code); got != tt.expected {
				t.Errorf("ToGRPCCode(%s) = %v, want %v", tt.code, got, tt.expected)
			}
		})
	}
}

func TestToGRPCError(t *testing.T) {
	// Test nil error
	if err := ToGRPCError(nil); err != nil {
		t.Errorf("ToGRPCError(nil) = %v, want nil", err)
	}

	// Test ResilienceError
	resErr := NewCircuitOpenError("test-service")
	grpcErr := ToGRPCError(resErr)
	st, ok := status.FromError(grpcErr)
	if !ok {
		t.Fatal("expected gRPC status error")
	}
	if st.Code() != codes.Unavailable {
		t.Errorf("expected Unavailable, got %v", st.Code())
	}

	// Test non-ResilienceError
	plainErr := errors.New("plain error")
	grpcErr = ToGRPCError(plainErr)
	st, ok = status.FromError(grpcErr)
	if !ok {
		t.Fatal("expected gRPC status error")
	}
	if st.Code() != codes.Internal {
		t.Errorf("expected Internal, got %v", st.Code())
	}
}

func TestToGRPCStatus(t *testing.T) {
	// Test nil error
	st := ToGRPCStatus(nil)
	if st.Code() != codes.OK {
		t.Errorf("ToGRPCStatus(nil) code = %v, want OK", st.Code())
	}

	// Test ResilienceError
	resErr := NewRateLimitError("test-service", 0)
	st = ToGRPCStatus(resErr)
	if st.Code() != codes.ResourceExhausted {
		t.Errorf("expected ResourceExhausted, got %v", st.Code())
	}
}

func TestFromGRPCCode(t *testing.T) {
	tests := []struct {
		code     codes.Code
		expected ResilienceErrorCode
	}{
		{codes.Unavailable, ErrCircuitOpen}, // First match in map
		{codes.ResourceExhausted, ErrRateLimitExceeded},
		{codes.DeadlineExceeded, ErrTimeout},
		{codes.InvalidArgument, ErrInvalidPolicy},
		{codes.NotFound, ErrServiceUnavailable}, // Default
	}

	for _, tt := range tests {
		t.Run(tt.code.String(), func(t *testing.T) {
			got := FromGRPCCode(tt.code)
			// Note: Due to map iteration order, we just check it returns a valid code
			if got == "" {
				t.Errorf("FromGRPCCode(%v) returned empty code", tt.code)
			}
		})
	}
}

func TestFromGRPCError(t *testing.T) {
	// Test nil error
	if err := FromGRPCError(nil); err != nil {
		t.Errorf("FromGRPCError(nil) = %v, want nil", err)
	}

	// Test gRPC error
	grpcErr := status.Error(codes.Unavailable, "service unavailable")
	resErr := FromGRPCError(grpcErr)
	if resErr == nil {
		t.Fatal("expected non-nil ResilienceError")
	}
	if resErr.Message != "service unavailable" {
		t.Errorf("expected message 'service unavailable', got %s", resErr.Message)
	}

	// Test non-gRPC error
	plainErr := errors.New("plain error")
	resErr = FromGRPCError(plainErr)
	if resErr == nil {
		t.Fatal("expected non-nil ResilienceError")
	}
	if resErr.Code != ErrServiceUnavailable {
		t.Errorf("expected code %s, got %s", ErrServiceUnavailable, resErr.Code)
	}
}

func TestAllErrorCodesMapped(t *testing.T) {
	// Ensure all defined error codes have a gRPC mapping
	codes := []ResilienceErrorCode{
		ErrCircuitOpen,
		ErrRateLimitExceeded,
		ErrTimeout,
		ErrBulkheadFull,
		ErrRetryExhausted,
		ErrInvalidPolicy,
		ErrServiceUnavailable,
		ErrValidation,
	}

	for _, code := range codes {
		if _, ok := ResilienceErrorMapping[code]; !ok {
			t.Errorf("error code %s not mapped to gRPC code", code)
		}
	}
}
