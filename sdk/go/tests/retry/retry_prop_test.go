// Package retry provides property-based tests for retry logic.
package retry

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/auth-platform/sdk-go/src/errors"
	"github.com/auth-platform/sdk-go/src/retry"
	"pgregory.net/rapid"
)

// Property 10: Retry Delay Exponential Backoff
func TestProperty_RetryDelayExponentialBackoff(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		attempt := rapid.IntRange(0, 10).Draw(t, "attempt")
		baseMs := rapid.IntRange(100, 1000).Draw(t, "baseMs")
		baseDelay := time.Duration(baseMs) * time.Millisecond
		maxDelay := baseDelay * 100
		jitter := 0.1

		p := retry.NewPolicy(
			retry.WithBaseDelay(baseDelay),
			retry.WithMaxDelay(maxDelay),
			retry.WithJitter(jitter),
		)

		delay := p.CalculateDelay(attempt)

		// Calculate expected base (without jitter)
		expectedBase := baseDelay * time.Duration(1<<attempt)
		if expectedBase > maxDelay {
			expectedBase = maxDelay
		}

		// Delay should be within jitter range
		minDelay := time.Duration(float64(expectedBase) * (1 - jitter))
		maxDelayWithJitter := time.Duration(float64(expectedBase) * (1 + jitter))

		if delay < minDelay {
			t.Fatalf("delay %v < minDelay %v", delay, minDelay)
		}
		if delay > maxDelayWithJitter {
			t.Fatalf("delay %v > maxDelayWithJitter %v", delay, maxDelayWithJitter)
		}
	})
}

// Property 11: Retry-After Header Parsing
func TestProperty_RetryAfterParsing(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		seconds := rapid.IntRange(1, 3600).Draw(t, "seconds")
		header := strconv.Itoa(seconds)

		duration, ok := retry.ParseRetryAfter(header)
		if !ok {
			t.Fatal("ParseRetryAfter should succeed for valid seconds")
		}
		if duration != time.Duration(seconds)*time.Second {
			t.Fatalf("duration = %v, want %v", duration, time.Duration(seconds)*time.Second)
		}
	})
}

// Property 12: Retry Success Behavior - successful operation returns immediately
func TestProperty_RetrySuccessBehavior(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		maxRetries := rapid.IntRange(1, 5).Draw(t, "maxRetries")
		expectedValue := rapid.Int().Draw(t, "expectedValue")

		p := retry.NewPolicy(
			retry.WithMaxRetries(maxRetries),
			retry.WithBaseDelay(time.Millisecond),
		)

		callCount := 0
		result := retry.Retry(context.Background(), p, func(ctx context.Context) (int, error) {
			callCount++
			return expectedValue, nil
		})

		if result.Err != nil {
			t.Fatalf("unexpected error: %v", result.Err)
		}
		if result.Value != expectedValue {
			t.Fatalf("value = %d, want %d", result.Value, expectedValue)
		}
		if callCount != 1 {
			t.Fatalf("callCount = %d, want 1", callCount)
		}
		if result.Attempts != 1 {
			t.Fatalf("attempts = %d, want 1", result.Attempts)
		}
	})
}

// Property 13: Retry Exhaustion Behavior
func TestProperty_RetryExhaustionBehavior(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		maxRetries := rapid.IntRange(0, 5).Draw(t, "maxRetries")

		p := retry.NewPolicy(
			retry.WithMaxRetries(maxRetries),
			retry.WithBaseDelay(time.Microsecond),
		)

		callCount := 0
		result := retry.Retry(context.Background(), p, func(ctx context.Context) (int, error) {
			callCount++
			return 0, errors.NewError(errors.ErrCodeNetwork, "network error")
		})

		expectedCalls := maxRetries + 1 // initial + retries
		if callCount != expectedCalls {
			t.Fatalf("callCount = %d, want %d", callCount, expectedCalls)
		}
		if result.Err == nil {
			t.Fatal("expected error after exhaustion")
		}
	})
}

// networkError implements a retryable error for testing
type networkError struct{}

func (e *networkError) Error() string { return "network error" }

// Property: Delay never exceeds MaxDelay
func TestProperty_DelayNeverExceedsMax(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		baseMs := rapid.IntRange(10, 100).Draw(t, "baseMs")
		maxMs := rapid.IntRange(100, 1000).Draw(t, "maxMs")
		attempt := rapid.IntRange(0, 20).Draw(t, "attempt")

		p := retry.NewPolicy(
			retry.WithBaseDelay(time.Duration(baseMs)*time.Millisecond),
			retry.WithMaxDelay(time.Duration(maxMs)*time.Millisecond),
			retry.WithJitter(0.2),
		)

		delay := p.CalculateDelay(attempt)
		maxWithJitter := time.Duration(float64(maxMs)*1.2) * time.Millisecond

		if delay > maxWithJitter {
			t.Fatalf("delay %v exceeds max with jitter %v", delay, maxWithJitter)
		}
	})
}

// Property: Delay never below BaseDelay (accounting for jitter)
func TestProperty_DelayNeverBelowBase(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		baseMs := rapid.IntRange(100, 1000).Draw(t, "baseMs")
		jitter := rapid.Float64Range(0, 0.5).Draw(t, "jitter")

		p := retry.NewPolicy(
			retry.WithBaseDelay(time.Duration(baseMs)*time.Millisecond),
			retry.WithMaxDelay(time.Hour),
			retry.WithJitter(jitter),
		)

		delay := p.CalculateDelay(0)
		minWithJitter := time.Duration(float64(baseMs)*(1-jitter)) * time.Millisecond

		if delay < minWithJitter {
			t.Fatalf("delay %v below min with jitter %v", delay, minWithJitter)
		}
	})
}

// Property: Zero jitter produces deterministic delays
func TestProperty_ZeroJitterDeterministic(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		baseMs := rapid.IntRange(10, 100).Draw(t, "baseMs")
		attempt := rapid.IntRange(0, 5).Draw(t, "attempt")

		p := retry.NewPolicy(
			retry.WithBaseDelay(time.Duration(baseMs)*time.Millisecond),
			retry.WithMaxDelay(time.Hour),
			retry.WithJitter(0),
		)

		delay1 := p.CalculateDelay(attempt)
		delay2 := p.CalculateDelay(attempt)

		if delay1 != delay2 {
			t.Fatalf("zero jitter should be deterministic: %v != %v", delay1, delay2)
		}
	})
}

// Property: IsRetryable is consistent
func TestProperty_IsRetryableConsistent(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		codes := []int{429, 502, 503, 504}
		code := rapid.SampledFrom(codes).Draw(t, "code")

		p := retry.DefaultPolicy()

		// Should always return true for default retryable codes
		if !p.IsRetryable(code) {
			t.Fatalf("code %d should be retryable", code)
		}
	})
}

// Property: Non-retryable codes are not retried
func TestProperty_NonRetryableCodesNotRetried(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		nonRetryable := []int{200, 201, 400, 401, 403, 404, 500}
		code := rapid.SampledFrom(nonRetryable).Draw(t, "code")

		p := retry.DefaultPolicy()

		if p.IsRetryable(code) {
			t.Fatalf("code %d should not be retryable", code)
		}
	})
}

// Property: Custom retryable codes work
func TestProperty_CustomRetryableCodes(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		customCodes := []int{
			rapid.IntRange(400, 599).Draw(t, "code1"),
			rapid.IntRange(400, 599).Draw(t, "code2"),
		}

		p := retry.NewPolicy(retry.WithRetryableStatusCodes(customCodes...))

		for _, code := range customCodes {
			if !p.IsRetryable(code) {
				t.Fatalf("custom code %d should be retryable", code)
			}
		}
	})
}
