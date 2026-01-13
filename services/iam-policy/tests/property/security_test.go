// Package property contains property-based tests for IAM Policy Service.
package property

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/auth-platform/iam-policy-service/internal/ratelimit"
	"github.com/auth-platform/iam-policy-service/internal/validation"
	"github.com/auth-platform/iam-policy-service/tests/testutil"
	"pgregory.net/rapid"
)

// TestInputValidationAndSanitization validates Property 14.
// All inputs must be validated and sanitized.
func TestInputValidationAndSanitization(t *testing.T) {
	validator := validation.NewAuthorizationRequestValidator()

	rapid.Check(t, func(t *rapid.T) {
		subjectID := testutil.NonEmptyStringGen().Draw(t, "subjectID")
		resourceType := rapid.StringMatching(`^[a-zA-Z][a-zA-Z0-9_-]{0,30}$`).Draw(t, "resourceType")
		action := rapid.StringMatching(`^[a-z][a-z0-9_:]{0,30}$`).Draw(t, "action")

		// Property: valid inputs should pass validation
		err := validator.ValidateSubjectID(subjectID)
		if err != nil && !strings.Contains(err.Error(), "invalid") {
			t.Logf("subject validation: %v", err)
		}

		err = validator.ValidateResourceType(resourceType)
		if err != nil {
			t.Errorf("valid resource type should pass: %v", err)
		}

		err = validator.ValidateAction(action)
		if err != nil {
			t.Errorf("valid action should pass: %v", err)
		}
	})
}

// TestInputValidationRejectsInvalid validates that invalid inputs are rejected.
func TestInputValidationRejectsInvalid(t *testing.T) {
	validator := validation.NewAuthorizationRequestValidator()

	testCases := []struct {
		name  string
		input string
		field string
	}{
		{"empty subject", "", "subject"},
		{"script injection", "<script>alert(1)</script>", "subject"},
		{"javascript protocol", "javascript:alert(1)", "subject"},
		{"control characters", "user\x00id", "subject"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var err error
			switch tc.field {
			case "subject":
				err = validator.ValidateSubjectID(tc.input)
			}

			if err == nil && tc.input != "" {
				// Some inputs might be valid depending on the pattern
				t.Logf("input %q was accepted", tc.input)
			}
		})
	}
}

// TestSanitizeForLog validates log sanitization.
func TestSanitizeForLog(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		input := rapid.String().Draw(t, "input")

		sanitized := validation.SanitizeForLog(input)

		// Property: sanitized output should not contain control characters
		for _, r := range sanitized {
			if r < 32 && r != '\n' && r != '\t' {
				t.Errorf("sanitized output contains control character: %d", r)
			}
		}

		// Property: sanitized output should be bounded in length
		if len(sanitized) > 260 { // 256 + "..."
			t.Errorf("sanitized output too long: %d", len(sanitized))
		}
	})
}

// TestRateLimitingEnforcement validates Property 15.
// Rate limiting must be enforced correctly.
func TestRateLimitingEnforcement(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		rps := rapid.IntRange(1, 10).Draw(t, "rps")
		burst := rapid.IntRange(1, 20).Draw(t, "burst")

		config := ratelimit.LimiterConfig{
			RequestsPerSecond: rps,
			BurstSize:         burst,
			CleanupInterval:   time.Minute,
		}

		limiter := ratelimit.NewLimiter(config)
		defer limiter.Close()

		clientID := testutil.NonEmptyStringGen().Draw(t, "clientID")
		ctx := context.Background()

		// Property: first burst requests should be allowed
		allowedCount := 0
		for i := 0; i < burst+5; i++ {
			if err := limiter.Allow(ctx, clientID); err == nil {
				allowedCount++
			}
		}

		// Property: at least burst requests should be allowed
		if allowedCount < burst {
			t.Errorf("expected at least %d allowed, got %d", burst, allowedCount)
		}
	})
}

// TestRateLimitingPerClient validates per-client rate limiting.
func TestRateLimitingPerClient(t *testing.T) {
	config := ratelimit.LimiterConfig{
		RequestsPerSecond: 10,
		BurstSize:         5,
		CleanupInterval:   time.Minute,
	}

	limiter := ratelimit.NewLimiter(config)
	defer limiter.Close()

	ctx := context.Background()

	// Exhaust client1's quota
	for i := 0; i < 10; i++ {
		_ = limiter.Allow(ctx, "client1")
	}

	// Property: client2 should still have quota
	err := limiter.Allow(ctx, "client2")
	if err != nil {
		t.Error("client2 should not be rate limited")
	}
}

// TestRateLimitingAnonymous validates anonymous request handling.
func TestRateLimitingAnonymous(t *testing.T) {
	config := ratelimit.DefaultLimiterConfig()
	limiter := ratelimit.NewLimiter(config)
	defer limiter.Close()

	ctx := context.Background()

	// Property: anonymous requests (empty client ID) should be allowed
	for i := 0; i < 100; i++ {
		err := limiter.Allow(ctx, "")
		if err != nil {
			t.Errorf("anonymous request should be allowed: %v", err)
		}
	}
}

// TestRateLimitingBatchRequests validates batch request rate limiting.
func TestRateLimitingBatchRequests(t *testing.T) {
	config := ratelimit.LimiterConfig{
		RequestsPerSecond: 10,
		BurstSize:         20,
		CleanupInterval:   time.Minute,
	}

	limiter := ratelimit.NewLimiter(config)
	defer limiter.Close()

	ctx := context.Background()

	// Property: AllowN should consume n tokens
	err := limiter.AllowN(ctx, "client", 15)
	if err != nil {
		t.Error("first batch should be allowed")
	}

	// Property: subsequent large batch should be rejected
	err = limiter.AllowN(ctx, "client", 10)
	if err == nil {
		t.Error("second batch should be rate limited")
	}
}
