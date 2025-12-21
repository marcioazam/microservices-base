package property_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	authplatform "github.com/auth-platform/sdk-go"
	"pgregory.net/rapid"
)

// TestErrorStructureCompleteness tests Property 2: Error Structure Completeness
// **Feature: go-sdk-modernization, Property 2: Error Structure Completeness**
// **Validates: Requirements 4.1**
func TestErrorStructureCompleteness(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		code := rapid.SampledFrom([]authplatform.ErrorCode{
			authplatform.ErrCodeInvalidConfig,
			authplatform.ErrCodeTokenExpired,
			authplatform.ErrCodeTokenInvalid,
			authplatform.ErrCodeTokenRefresh,
			authplatform.ErrCodeNetwork,
			authplatform.ErrCodeRateLimited,
			authplatform.ErrCodeValidation,
			authplatform.ErrCodeUnauthorized,
			authplatform.ErrCodeDPoPRequired,
			authplatform.ErrCodeDPoPInvalid,
			authplatform.ErrCodePKCEInvalid,
		}).Draw(t, "code")

		message := rapid.StringMatching(`[a-zA-Z0-9 ]+`).Draw(t, "message")
		if message == "" {
			message = "test error"
		}

		err := authplatform.NewError(code, message)

		// Property: Code must not be empty
		if err.Code == "" {
			t.Fatal("error code should not be empty")
		}

		// Property: Message must not be empty
		if err.Message == "" {
			t.Fatal("error message should not be empty")
		}

		// Property: Error() must return non-empty string
		if err.Error() == "" {
			t.Fatal("Error() should return non-empty string")
		}

		// Property: Error() must contain "authplatform" prefix
		if !strings.HasPrefix(err.Error(), "authplatform:") {
			t.Fatalf("Error() should start with 'authplatform:': %s", err.Error())
		}
	})
}

// TestErrorTypeExtractionWithAs tests Property 3: Error Type Extraction with AsType
// **Feature: go-sdk-modernization, Property 3: Error Type Extraction with AsType**
// **Validates: Requirements 4.2**
func TestErrorTypeExtractionWithAs(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		code := rapid.SampledFrom([]authplatform.ErrorCode{
			authplatform.ErrCodeInvalidConfig,
			authplatform.ErrCodeTokenExpired,
			authplatform.ErrCodeNetwork,
		}).Draw(t, "code")

		message := rapid.StringMatching(`[a-zA-Z0-9 ]+`).Draw(t, "message")
		if message == "" {
			message = "test"
		}

		original := authplatform.NewError(code, message)

		// Wrap the error
		wrapCount := rapid.IntRange(1, 5).Draw(t, "wrapCount")
		var wrapped error = original
		for i := 0; i < wrapCount; i++ {
			wrapped = fmt.Errorf("wrap %d: %w", i, wrapped)
		}

		// Property: errors.As must successfully extract SDKError
		var extracted *authplatform.SDKError
		if !errors.As(wrapped, &extracted) {
			t.Fatal("errors.As should extract SDKError from wrapped error")
		}

		// Property: Extracted error must have same code
		if extracted.Code != code {
			t.Fatalf("extracted code %s != original code %s", extracted.Code, code)
		}
	})
}

// TestIsHelperFunctionsCorrectness tests Property 4: Is Helper Functions Correctness
// **Feature: go-sdk-modernization, Property 4: Is Helper Functions Correctness**
// **Validates: Requirements 4.3**
func TestIsHelperFunctionsCorrectness(t *testing.T) {
	type helperTest struct {
		err     error
		checkFn func(error) bool
		name    string
	}

	helpers := []helperTest{
		{authplatform.ErrTokenExpired, authplatform.IsTokenExpired, "IsTokenExpired"},
		{authplatform.ErrRateLimited, authplatform.IsRateLimited, "IsRateLimited"},
		{authplatform.ErrNetwork, authplatform.IsNetwork, "IsNetwork"},
		{authplatform.ErrValidation, authplatform.IsValidation, "IsValidation"},
		{authplatform.ErrInvalidConfig, authplatform.IsInvalidConfig, "IsInvalidConfig"},
		{authplatform.ErrUnauthorized, authplatform.IsUnauthorized, "IsUnauthorized"},
		{authplatform.ErrDPoPRequired, authplatform.IsDPoPRequired, "IsDPoPRequired"},
		{authplatform.ErrDPoPInvalid, authplatform.IsDPoPInvalid, "IsDPoPInvalid"},
		{authplatform.ErrPKCEInvalid, authplatform.IsPKCEInvalid, "IsPKCEInvalid"},
	}

	rapid.Check(t, func(t *rapid.T) {
		idx := rapid.IntRange(0, len(helpers)-1).Draw(t, "helperIndex")
		helper := helpers[idx]

		// Property: Helper returns true for its sentinel error
		if !helper.checkFn(helper.err) {
			t.Fatalf("%s should return true for its sentinel error", helper.name)
		}

		// Property: Helper returns true for wrapped sentinel error
		wrapped := fmt.Errorf("context: %w", helper.err)
		if !helper.checkFn(wrapped) {
			t.Fatalf("%s should return true for wrapped sentinel error", helper.name)
		}

		// Property: Helper returns false for other sentinel errors
		for j, other := range helpers {
			if j != idx && helper.checkFn(other.err) {
				t.Fatalf("%s should return false for %s", helper.name, other.name)
			}
		}

		// Property: Helper returns false for generic error
		genericErr := errors.New("generic error")
		if helper.checkFn(genericErr) {
			t.Fatalf("%s should return false for generic error", helper.name)
		}
	})
}

// TestErrorChainPreservation tests Property 5: Error Chain Preservation
// **Feature: go-sdk-modernization, Property 5: Error Chain Preservation**
// **Validates: Requirements 4.4**
func TestErrorChainPreservation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Create a chain of errors
		chainLength := rapid.IntRange(1, 10).Draw(t, "chainLength")

		baseErr := authplatform.ErrTokenExpired
		var chain error = baseErr

		for i := 0; i < chainLength; i++ {
			chain = fmt.Errorf("level %d: %w", i, chain)
		}

		// Property: errors.Is must find the base error
		if !errors.Is(chain, baseErr) {
			t.Fatal("errors.Is should find base error in chain")
		}

		// Property: Unwrap chain must eventually reach base error
		current := chain
		found := false
		for i := 0; i <= chainLength+1; i++ {
			if current == baseErr {
				found = true
				break
			}
			unwrapper, ok := current.(interface{ Unwrap() error })
			if !ok {
				break
			}
			current = unwrapper.Unwrap()
		}

		if !found {
			t.Fatal("unwrap chain should eventually reach base error")
		}
	})
}

// TestNoSensitiveDataInErrors tests Property 6: No Sensitive Data in Errors
// **Feature: go-sdk-modernization, Property 6: No Sensitive Data in Errors**
// **Validates: Requirements 4.6**
func TestNoSensitiveDataInErrors(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		code := rapid.SampledFrom([]authplatform.ErrorCode{
			authplatform.ErrCodeInvalidConfig,
			authplatform.ErrCodeTokenExpired,
			authplatform.ErrCodeNetwork,
		}).Draw(t, "code")

		// Generate safe message (no sensitive patterns)
		safeMessage := rapid.StringMatching(`[a-zA-Z0-9 ]+`).Draw(t, "safeMessage")
		if safeMessage == "" {
			safeMessage = "operation failed"
		}

		err := authplatform.NewError(code, safeMessage)
		errMsg := err.Error()

		// Property: Error message should not contain sensitive patterns
		sensitivePatterns := []string{"Bearer ", "DPoP ", "secret", "password", "credential", "eyJ"}
		for _, pattern := range sensitivePatterns {
			if strings.Contains(strings.ToLower(errMsg), strings.ToLower(pattern)) {
				t.Fatalf("error message should not contain sensitive pattern '%s': %s", pattern, errMsg)
			}
		}
	})
}

// TestSDKErrorWrapping tests SDKError with cause wrapping
func TestSDKErrorWrapping(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		code := rapid.SampledFrom([]authplatform.ErrorCode{
			authplatform.ErrCodeNetwork,
			authplatform.ErrCodeValidation,
		}).Draw(t, "code")

		message := rapid.StringMatching(`[a-zA-Z0-9 ]+`).Draw(t, "message")
		if message == "" {
			message = "test"
		}

		cause := errors.New("underlying cause")
		err := authplatform.WrapError(code, message, cause)

		// Property: Unwrap returns the cause
		if err.Unwrap() != cause {
			t.Fatal("Unwrap should return the cause")
		}

		// Property: errors.Is finds the cause
		if !errors.Is(err, cause) {
			t.Fatal("errors.Is should find the cause")
		}

		// Property: Error() includes cause message
		if !strings.Contains(err.Error(), cause.Error()) {
			t.Fatal("Error() should include cause message")
		}
	})
}
