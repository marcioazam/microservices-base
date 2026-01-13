package errors_test

import (
	"errors"
	"testing"

	sdkerrors "github.com/auth-platform/sdk-go/src/errors"
	"pgregory.net/rapid"
)

// Feature: go-sdk-state-of-art-2025, Property 1: Error Structure Completeness
func TestProperty_ErrorStructureCompleteness(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		codes := sdkerrors.AllErrorCodes()
		code := rapid.SampledFrom(codes).Draw(t, "code")
		msg := rapid.StringN(1, 100, 200).Draw(t, "message")

		err := sdkerrors.NewError(code, msg)

		if err.Code == "" {
			t.Error("Code should not be empty")
		}
		if err.Message == "" {
			t.Error("Message should not be empty")
		}
	})
}

// Feature: go-sdk-state-of-art-2025, Property 2: Error Helper Functions Correctness
func TestProperty_ErrorHelperCorrectness(t *testing.T) {
	type helperTest struct {
		code    sdkerrors.ErrorCode
		checker func(error) bool
	}

	helpers := []helperTest{
		{sdkerrors.ErrCodeTokenExpired, sdkerrors.IsTokenExpired},
		{sdkerrors.ErrCodeTokenInvalid, sdkerrors.IsTokenInvalid},
		{sdkerrors.ErrCodeTokenMissing, sdkerrors.IsTokenMissing},
		{sdkerrors.ErrCodeRateLimited, sdkerrors.IsRateLimited},
		{sdkerrors.ErrCodeNetwork, sdkerrors.IsNetwork},
		{sdkerrors.ErrCodeValidation, sdkerrors.IsValidation},
		{sdkerrors.ErrCodeInvalidConfig, sdkerrors.IsInvalidConfig},
		{sdkerrors.ErrCodeUnauthorized, sdkerrors.IsUnauthorized},
		{sdkerrors.ErrCodeDPoPRequired, sdkerrors.IsDPoPRequired},
		{sdkerrors.ErrCodeDPoPInvalid, sdkerrors.IsDPoPInvalid},
		{sdkerrors.ErrCodePKCEInvalid, sdkerrors.IsPKCEInvalid},
	}

	rapid.Check(t, func(t *rapid.T) {
		idx := rapid.IntRange(0, len(helpers)-1).Draw(t, "helperIndex")
		msg := rapid.StringN(1, 50, 100).Draw(t, "message")

		helper := helpers[idx]
		err := sdkerrors.NewError(helper.code, msg)

		// The matching helper should return true
		if !helper.checker(err) {
			t.Errorf("helper for %s should return true", helper.code)
		}

		// All other helpers should return false
		for i, other := range helpers {
			if i != idx && other.checker(err) {
				t.Errorf("helper for %s should return false for error with code %s", other.code, helper.code)
			}
		}
	})
}

// Feature: go-sdk-state-of-art-2025, Property 3: Error Message Sanitization
func TestProperty_ErrorSanitization(t *testing.T) {
	sensitivePatterns := []string{"Bearer ", "DPoP ", "secret", "password", "eyJ"}

	rapid.Check(t, func(t *rapid.T) {
		pattern := rapid.SampledFrom(sensitivePatterns).Draw(t, "pattern")
		prefix := rapid.StringN(0, 20, 50).Draw(t, "prefix")
		suffix := rapid.StringN(0, 20, 50).Draw(t, "suffix")

		input := prefix + pattern + suffix
		if sdkerrors.ContainsSensitiveData(input) {
			sanitized := sdkerrors.SanitizeError(errors.New(input))
			sanitizedMsg := sanitized.Error()
			// Sanitized message should not contain the original sensitive pattern
			if sdkerrors.ContainsSensitiveData(sanitizedMsg) && sanitizedMsg != "authplatform: error occurred (details redacted)" {
				t.Errorf("sanitized message still contains sensitive data: %s", sanitizedMsg)
			}
		}
	})
}

// Feature: go-sdk-state-of-art-2025, Property 4: Error Chain Preservation
func TestProperty_ErrorChainPreservation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		codes := sdkerrors.AllErrorCodes()
		code := rapid.SampledFrom(codes).Draw(t, "code")
		msg := rapid.StringN(1, 50, 100).Draw(t, "message")
		causeMsg := rapid.StringN(1, 50, 100).Draw(t, "causeMessage")

		cause := errors.New(causeMsg)
		wrapped := sdkerrors.WrapError(code, msg, cause)

		// Unwrap should return the original cause
		unwrapped := errors.Unwrap(wrapped)
		if unwrapped != cause {
			t.Error("Unwrap should return the original cause")
		}

		// errors.Is should correctly identify the wrapped error
		if !errors.Is(wrapped, cause) {
			t.Error("errors.Is should find the cause in the chain")
		}

		// GetCode should return the correct code
		if sdkerrors.GetCode(wrapped) != code {
			t.Errorf("GetCode should return %s, got %s", code, sdkerrors.GetCode(wrapped))
		}
	})
}
