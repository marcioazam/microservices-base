package errors_test

import (
	"errors"
	"testing"

	sdkerrors "github.com/auth-platform/sdk-go/src/errors"
)

func TestNewError(t *testing.T) {
	err := sdkerrors.NewError(sdkerrors.ErrCodeTokenExpired, "token has expired")
	if err.Code != sdkerrors.ErrCodeTokenExpired {
		t.Errorf("expected code %s, got %s", sdkerrors.ErrCodeTokenExpired, err.Code)
	}
	if err.Message != "token has expired" {
		t.Errorf("expected message 'token has expired', got '%s'", err.Message)
	}
}

func TestWrapError(t *testing.T) {
	cause := errors.New("underlying error")
	err := sdkerrors.WrapError(sdkerrors.ErrCodeNetwork, "network failed", cause)
	if err.Cause != cause {
		t.Error("expected cause to be preserved")
	}
	if !errors.Is(err, cause) {
		t.Error("errors.Is should find the cause")
	}
}

func TestSDKErrorUnwrap(t *testing.T) {
	cause := errors.New("root cause")
	err := sdkerrors.WrapError(sdkerrors.ErrCodeValidation, "validation failed", cause)
	unwrapped := errors.Unwrap(err)
	if unwrapped != cause {
		t.Error("Unwrap should return the cause")
	}
}

func TestSDKErrorIs(t *testing.T) {
	err1 := sdkerrors.NewError(sdkerrors.ErrCodeTokenExpired, "expired")
	err2 := sdkerrors.NewError(sdkerrors.ErrCodeTokenExpired, "also expired")
	err3 := sdkerrors.NewError(sdkerrors.ErrCodeNetwork, "network")

	if !err1.Is(err2) {
		t.Error("errors with same code should match")
	}
	if err1.Is(err3) {
		t.Error("errors with different codes should not match")
	}
}

func TestIsHelperFunctions(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		checker  func(error) bool
		expected bool
	}{
		{"TokenExpired", sdkerrors.NewError(sdkerrors.ErrCodeTokenExpired, ""), sdkerrors.IsTokenExpired, true},
		{"TokenInvalid", sdkerrors.NewError(sdkerrors.ErrCodeTokenInvalid, ""), sdkerrors.IsTokenInvalid, true},
		{"TokenMissing", sdkerrors.NewError(sdkerrors.ErrCodeTokenMissing, ""), sdkerrors.IsTokenMissing, true},
		{"RateLimited", sdkerrors.NewError(sdkerrors.ErrCodeRateLimited, ""), sdkerrors.IsRateLimited, true},
		{"Network", sdkerrors.NewError(sdkerrors.ErrCodeNetwork, ""), sdkerrors.IsNetwork, true},
		{"Validation", sdkerrors.NewError(sdkerrors.ErrCodeValidation, ""), sdkerrors.IsValidation, true},
		{"InvalidConfig", sdkerrors.NewError(sdkerrors.ErrCodeInvalidConfig, ""), sdkerrors.IsInvalidConfig, true},
		{"Unauthorized", sdkerrors.NewError(sdkerrors.ErrCodeUnauthorized, ""), sdkerrors.IsUnauthorized, true},
		{"DPoPRequired", sdkerrors.NewError(sdkerrors.ErrCodeDPoPRequired, ""), sdkerrors.IsDPoPRequired, true},
		{"DPoPInvalid", sdkerrors.NewError(sdkerrors.ErrCodeDPoPInvalid, ""), sdkerrors.IsDPoPInvalid, true},
		{"PKCEInvalid", sdkerrors.NewError(sdkerrors.ErrCodePKCEInvalid, ""), sdkerrors.IsPKCEInvalid, true},
		{"WrongChecker", sdkerrors.NewError(sdkerrors.ErrCodeNetwork, ""), sdkerrors.IsTokenExpired, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.checker(tt.err); got != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, got)
			}
		})
	}
}

func TestGetCode(t *testing.T) {
	err := sdkerrors.NewError(sdkerrors.ErrCodeTokenExpired, "expired")
	if code := sdkerrors.GetCode(err); code != sdkerrors.ErrCodeTokenExpired {
		t.Errorf("expected %s, got %s", sdkerrors.ErrCodeTokenExpired, code)
	}

	plainErr := errors.New("plain error")
	if code := sdkerrors.GetCode(plainErr); code != "" {
		t.Errorf("expected empty code for plain error, got %s", code)
	}
}

func TestErrorString(t *testing.T) {
	err := sdkerrors.NewError(sdkerrors.ErrCodeTokenExpired, "token expired")
	expected := "authplatform: TOKEN_EXPIRED: token expired"
	if err.Error() != expected {
		t.Errorf("expected '%s', got '%s'", expected, err.Error())
	}

	cause := errors.New("cause")
	wrapped := sdkerrors.WrapError(sdkerrors.ErrCodeNetwork, "failed", cause)
	if !errors.Is(wrapped, cause) {
		t.Error("wrapped error should contain cause")
	}
}
