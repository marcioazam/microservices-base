package authplatform_test

import (
	"errors"
	"fmt"
	"testing"

	authplatform "github.com/auth-platform/sdk-go"
)

// TestErrorTypeSafety tests Property 13: SDK Error Type Safety
// **Feature: go-sdk-modernization, Property 4: Is Helper Functions Correctness**
// **Validates: Requirements 4.3**
func TestErrorTypeSafety(t *testing.T) {
	testCases := []struct {
		name     string
		err      error
		checkFn  func(error) bool
		expected bool
	}{
		{
			name:     "ErrTokenExpired is identifiable",
			err:      authplatform.ErrTokenExpired,
			checkFn:  authplatform.IsTokenExpired,
			expected: true,
		},
		{
			name:     "ErrRateLimited is identifiable",
			err:      authplatform.ErrRateLimited,
			checkFn:  authplatform.IsRateLimited,
			expected: true,
		},
		{
			name:     "ErrNetwork is identifiable",
			err:      authplatform.ErrNetwork,
			checkFn:  authplatform.IsNetwork,
			expected: true,
		},
		{
			name:     "ErrValidation is identifiable",
			err:      authplatform.ErrValidation,
			checkFn:  authplatform.IsValidation,
			expected: true,
		},
		{
			name:     "Wrapped ErrTokenExpired is identifiable",
			err:      fmt.Errorf("context: %w", authplatform.ErrTokenExpired),
			checkFn:  authplatform.IsTokenExpired,
			expected: true,
		},
		{
			name:     "Wrapped ErrRateLimited is identifiable",
			err:      fmt.Errorf("context: %w", authplatform.ErrRateLimited),
			checkFn:  authplatform.IsRateLimited,
			expected: true,
		},
		{
			name:     "Wrapped ErrNetwork is identifiable",
			err:      fmt.Errorf("context: %w", authplatform.ErrNetwork),
			checkFn:  authplatform.IsNetwork,
			expected: true,
		},
		{
			name:     "Wrapped ErrValidation is identifiable",
			err:      fmt.Errorf("context: %w", authplatform.ErrValidation),
			checkFn:  authplatform.IsValidation,
			expected: true,
		},
		{
			name:     "Generic error is not ErrTokenExpired",
			err:      errors.New("some error"),
			checkFn:  authplatform.IsTokenExpired,
			expected: false,
		},
		{
			name:     "Generic error is not ErrRateLimited",
			err:      errors.New("some error"),
			checkFn:  authplatform.IsRateLimited,
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

// TestErrorWrapping tests Property 5: Error Chain Preservation
// **Feature: go-sdk-modernization, Property 5: Error Chain Preservation**
// **Validates: Requirements 4.4**
func TestErrorWrapping(t *testing.T) {
	wrapped := fmt.Errorf("operation failed: %w", authplatform.ErrTokenExpired)

	if !errors.Is(wrapped, authplatform.ErrTokenExpired) {
		t.Error("wrapped error should be identifiable as ErrTokenExpired")
	}

	doubleWrapped := fmt.Errorf("outer: %w", wrapped)

	if !errors.Is(doubleWrapped, authplatform.ErrTokenExpired) {
		t.Error("double-wrapped error should be identifiable as ErrTokenExpired")
	}
}

// TestSentinelErrorsAreDistinct tests that sentinel errors are distinct.
func TestSentinelErrorsAreDistinct(t *testing.T) {
	sentinels := []error{
		authplatform.ErrInvalidConfig,
		authplatform.ErrTokenExpired,
		authplatform.ErrTokenRefresh,
		authplatform.ErrNetwork,
		authplatform.ErrRateLimited,
		authplatform.ErrValidation,
		authplatform.ErrUnauthorized,
	}

	for i, err1 := range sentinels {
		for j, err2 := range sentinels {
			if i != j && errors.Is(err1, err2) {
				t.Errorf("sentinel errors %v and %v should be distinct", err1, err2)
			}
		}
	}
}

// TestErrorMessages tests Property 2: Error Structure Completeness
// **Feature: go-sdk-modernization, Property 2: Error Structure Completeness**
// **Validates: Requirements 4.1**
func TestErrorMessages(t *testing.T) {
	sentinels := []error{
		authplatform.ErrInvalidConfig,
		authplatform.ErrTokenExpired,
		authplatform.ErrTokenRefresh,
		authplatform.ErrNetwork,
		authplatform.ErrRateLimited,
		authplatform.ErrValidation,
		authplatform.ErrUnauthorized,
	}

	for _, err := range sentinels {
		t.Run(err.Error(), func(t *testing.T) {
			msg := err.Error()

			if msg == "" {
				t.Error("error message should not be empty")
			}

			if len(msg) < 12 || msg[:12] != "authplatform" {
				t.Errorf("error message should start with 'authplatform': %s", msg)
			}
		})
	}
}
