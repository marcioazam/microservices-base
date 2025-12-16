package authplatform

import (
	"errors"
	"fmt"
	"testing"
)

// TestErrorTypeSafety tests Property 13: SDK Error Type Safety
// **Feature: auth-platform-q2-2025-evolution, Property 13: SDK Error Type Safety**
// **Validates: Requirements 8.5, 10.4**
//
// For any error returned by SDK methods, the error SHALL be an instance of
// a typed error class with error code and actionable message.
func TestErrorTypeSafety(t *testing.T) {
	testCases := []struct {
		name     string
		err      error
		checkFn  func(error) bool
		expected bool
	}{
		{
			name:     "ErrTokenExpired is identifiable",
			err:      ErrTokenExpired,
			checkFn:  IsTokenExpired,
			expected: true,
		},
		{
			name:     "ErrRateLimited is identifiable",
			err:      ErrRateLimited,
			checkFn:  IsRateLimited,
			expected: true,
		},
		{
			name:     "ErrNetwork is identifiable",
			err:      ErrNetwork,
			checkFn:  IsNetwork,
			expected: true,
		},
		{
			name:     "ErrValidation is identifiable",
			err:      ErrValidation,
			checkFn:  IsValidation,
			expected: true,
		},
		{
			name:     "Wrapped ErrTokenExpired is identifiable",
			err:      fmt.Errorf("context: %w", ErrTokenExpired),
			checkFn:  IsTokenExpired,
			expected: true,
		},
		{
			name:     "Wrapped ErrRateLimited is identifiable",
			err:      fmt.Errorf("context: %w", ErrRateLimited),
			checkFn:  IsRateLimited,
			expected: true,
		},
		{
			name:     "Wrapped ErrNetwork is identifiable",
			err:      fmt.Errorf("context: %w", ErrNetwork),
			checkFn:  IsNetwork,
			expected: true,
		},
		{
			name:     "Wrapped ErrValidation is identifiable",
			err:      fmt.Errorf("context: %w", ErrValidation),
			checkFn:  IsValidation,
			expected: true,
		},
		{
			name:     "Generic error is not ErrTokenExpired",
			err:      errors.New("some error"),
			checkFn:  IsTokenExpired,
			expected: false,
		},
		{
			name:     "Generic error is not ErrRateLimited",
			err:      errors.New("some error"),
			checkFn:  IsRateLimited,
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.checkFn(tc.err)
			if result != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, result)
			}
		})
	}
}

// TestErrorWrapping tests that errors can be wrapped and unwrapped correctly.
func TestErrorWrapping(t *testing.T) {
	// Property: Wrapped errors must be unwrappable
	wrapped := fmt.Errorf("operation failed: %w", ErrTokenExpired)

	if !errors.Is(wrapped, ErrTokenExpired) {
		t.Error("wrapped error should be identifiable as ErrTokenExpired")
	}

	// Property: Double-wrapped errors must still be identifiable
	doubleWrapped := fmt.Errorf("outer: %w", wrapped)

	if !errors.Is(doubleWrapped, ErrTokenExpired) {
		t.Error("double-wrapped error should be identifiable as ErrTokenExpired")
	}
}

// TestSentinelErrorsAreDistinct tests that sentinel errors are distinct.
func TestSentinelErrorsAreDistinct(t *testing.T) {
	sentinels := []error{
		ErrInvalidConfig,
		ErrTokenExpired,
		ErrTokenRefresh,
		ErrNetwork,
		ErrRateLimited,
		ErrValidation,
		ErrUnauthorized,
	}

	// Property: Each sentinel error must be distinct
	for i, err1 := range sentinels {
		for j, err2 := range sentinels {
			if i != j && errors.Is(err1, err2) {
				t.Errorf("sentinel errors %v and %v should be distinct", err1, err2)
			}
		}
	}
}

// TestErrorMessages tests that errors have meaningful messages.
func TestErrorMessages(t *testing.T) {
	sentinels := []error{
		ErrInvalidConfig,
		ErrTokenExpired,
		ErrTokenRefresh,
		ErrNetwork,
		ErrRateLimited,
		ErrValidation,
		ErrUnauthorized,
	}

	for _, err := range sentinels {
		t.Run(err.Error(), func(t *testing.T) {
			msg := err.Error()

			// Property: Error message must not be empty
			if msg == "" {
				t.Error("error message should not be empty")
			}

			// Property: Error message must contain "authplatform" prefix
			if len(msg) < 12 || msg[:12] != "authplatform" {
				t.Errorf("error message should start with 'authplatform': %s", msg)
			}
		})
	}
}
