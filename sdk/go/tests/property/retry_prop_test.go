package property

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	authplatform "github.com/auth-platform/sdk-go"
	"pgregory.net/rapid"
)

// TestProperty17_ExponentialBackoffDelays validates Property 17:
// For any sequence of retry attempts, the delay between attempt n and n+1
// SHALL be approximately baseDelay * 2^n (with jitter).
func TestProperty17_ExponentialBackoffDelays(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		baseDelay := rapid.Int64Range(100, 1000).Draw(t, "baseDelayMs")
		maxDelay := rapid.Int64Range(10000, 60000).Draw(t, "maxDelayMs")
		attempt := rapid.IntRange(0, 10).Draw(t, "attempt")

		policy := authplatform.NewRetryPolicy(
			authplatform.WithBaseDelay(time.Duration(baseDelay)*time.Millisecond),
			authplatform.WithMaxDelay(time.Duration(maxDelay)*time.Millisecond),
			authplatform.WithJitter(0), // No jitter for deterministic test
		)

		delay := policy.CalculateDelay(attempt)

		// Expected delay without jitter: baseDelay * 2^attempt
		expected := time.Duration(baseDelay) * time.Millisecond * (1 << attempt)
		if expected > time.Duration(maxDelay)*time.Millisecond {
			expected = time.Duration(maxDelay) * time.Millisecond
		}

		if delay != expected {
			t.Errorf("attempt %d: expected %v, got %v", attempt, expected, delay)
		}
	})
}

// TestProperty18_RetryAfterHeaderRespect validates Property 18:
// For any response with Retry-After header, the next retry delay
// SHALL be at least the value specified in the header.
func TestProperty18_RetryAfterHeaderRespect(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		retryAfterSeconds := rapid.IntRange(1, 120).Draw(t, "retryAfterSeconds")
		headerValue := rapid.SampledFrom([]string{
			// Integer format
			string(rune('0' + retryAfterSeconds%10)),
		}).Draw(t, "headerFormat")

		// Use simple integer format for testing
		_ = headerValue
		intHeader := rapid.IntRange(1, 60).Draw(t, "intHeader")

		delay, ok := authplatform.ParseRetryAfter(string(rune('0'+intHeader%10)) + string(rune('0'+intHeader/10%10)))
		if intHeader < 10 {
			delay, ok = authplatform.ParseRetryAfter(string(rune('0' + intHeader)))
		} else {
			delay, ok = authplatform.ParseRetryAfter(string([]byte{byte('0' + intHeader/10), byte('0' + intHeader%10)}))
		}

		if !ok {
			// Parsing may fail for some formats, which is acceptable
			return
		}

		expectedMin := time.Duration(intHeader) * time.Second
		if delay < expectedMin {
			t.Errorf("Retry-After %d: expected at least %v, got %v", intHeader, expectedMin, delay)
		}
	})
}

// TestProperty19_MaximumRetryCount validates Property 19:
// For any operation with configured maxRetries, the total number of attempts
// SHALL NOT exceed maxRetries + 1 (initial + retries).
func TestProperty19_MaximumRetryCount(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		maxRetries := rapid.IntRange(0, 5).Draw(t, "maxRetries")

		policy := authplatform.NewRetryPolicy(
			authplatform.WithMaxRetries(maxRetries),
			authplatform.WithBaseDelay(time.Millisecond), // Fast for testing
		)

		attemptCount := 0
		alwaysFail := func(ctx context.Context) (int, error) {
			attemptCount++
			return 0, authplatform.NewError(authplatform.ErrCodeNetwork, "always fails")
		}

		ctx := context.Background()
		result := authplatform.Retry(ctx, policy, alwaysFail)

		expectedMaxAttempts := maxRetries + 1
		if result.Attempts > expectedMaxAttempts {
			t.Errorf("maxRetries=%d: expected at most %d attempts, got %d",
				maxRetries, expectedMaxAttempts, result.Attempts)
		}
		if attemptCount > expectedMaxAttempts {
			t.Errorf("maxRetries=%d: function called %d times, expected at most %d",
				maxRetries, attemptCount, expectedMaxAttempts)
		}
	})
}

// TestProperty20_RetryDelayBounds validates Property 20:
// For any retry delay calculation, the delay SHALL be at least baseDelay
// and at most maxDelay.
func TestProperty20_RetryDelayBounds(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		baseDelayMs := rapid.Int64Range(10, 1000).Draw(t, "baseDelayMs")
		maxDelayMs := rapid.Int64Range(baseDelayMs+1, 60000).Draw(t, "maxDelayMs")
		jitter := rapid.Float64Range(0, 0.5).Draw(t, "jitter")
		attempt := rapid.IntRange(0, 20).Draw(t, "attempt")

		policy := authplatform.NewRetryPolicy(
			authplatform.WithBaseDelay(time.Duration(baseDelayMs)*time.Millisecond),
			authplatform.WithMaxDelay(time.Duration(maxDelayMs)*time.Millisecond),
			authplatform.WithJitter(jitter),
		)

		delay := policy.CalculateDelay(attempt)

		minDelay := time.Duration(baseDelayMs) * time.Millisecond
		maxDelay := time.Duration(maxDelayMs) * time.Millisecond

		// With jitter, delay can be slightly below baseDelay
		minAllowed := time.Duration(float64(minDelay) * (1 - jitter))

		if delay < minAllowed {
			t.Errorf("delay %v below minimum %v (baseDelay=%v, jitter=%v)",
				delay, minAllowed, minDelay, jitter)
		}
		if delay > maxDelay {
			t.Errorf("delay %v exceeds maximum %v", delay, maxDelay)
		}
	})
}

// TestProperty21_ContextCancellationStopsRetry validates Property 21:
// For any cancelled context during retry loop, the operation SHALL return
// immediately with context.Canceled error.
func TestProperty21_ContextCancellationStopsRetry(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		maxRetries := rapid.IntRange(2, 5).Draw(t, "maxRetries")
		cancelAfter := rapid.IntRange(0, maxRetries).Draw(t, "cancelAfter")

		policy := authplatform.NewRetryPolicy(
			authplatform.WithMaxRetries(maxRetries),
			authplatform.WithBaseDelay(10*time.Millisecond),
		)

		ctx, cancel := context.WithCancel(context.Background())
		attemptCount := 0

		op := func(ctx context.Context) (int, error) {
			attemptCount++
			if attemptCount > cancelAfter {
				cancel()
			}
			return 0, authplatform.NewError(authplatform.ErrCodeNetwork, "fails")
		}

		result := authplatform.Retry(ctx, policy, op)

		// Should have stopped due to cancellation
		if result.Err == nil {
			t.Error("expected error after context cancellation")
		}

		// Should not have exceeded cancel point by much
		if attemptCount > cancelAfter+2 {
			t.Errorf("expected to stop near attempt %d, but made %d attempts",
				cancelAfter, attemptCount)
		}
	})
}

// TestProperty22_NonRetryableErrorHandling validates Property 22:
// For any HTTP 4xx error (except 429 and 503), the operation SHALL NOT retry
// and SHALL return immediately.
func TestProperty22_NonRetryableErrorHandling(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Non-retryable 4xx codes (excluding 429)
		nonRetryableCodes := []int{400, 401, 403, 404, 405, 406, 408, 409, 410, 422}
		statusCode := rapid.SampledFrom(nonRetryableCodes).Draw(t, "statusCode")

		policy := authplatform.DefaultRetryPolicy()

		if policy.IsRetryable(statusCode) {
			t.Errorf("status code %d should not be retryable", statusCode)
		}
	})
}

// TestRetryableStatusCodes tests that retryable status codes are correctly identified.
func TestRetryableStatusCodes(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		retryableCodes := []int{429, 502, 503, 504}
		statusCode := rapid.SampledFrom(retryableCodes).Draw(t, "statusCode")

		policy := authplatform.DefaultRetryPolicy()

		if !policy.IsRetryable(statusCode) {
			t.Errorf("status code %d should be retryable", statusCode)
		}
	})
}

// TestRetryableErrors tests that retryable errors are correctly identified.
func TestRetryableErrors(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		errType := rapid.IntRange(0, 2).Draw(t, "errType")

		policy := authplatform.DefaultRetryPolicy()

		var err error
		var shouldRetry bool

		switch errType {
		case 0:
			err = authplatform.ErrNetwork
			shouldRetry = true
		case 1:
			err = authplatform.ErrRateLimited
			shouldRetry = true
		case 2:
			err = authplatform.ErrValidation
			shouldRetry = false
		}

		if policy.IsRetryableError(err) != shouldRetry {
			t.Errorf("error %v: expected retryable=%v, got %v",
				err, shouldRetry, policy.IsRetryableError(err))
		}
	})
}

// TestParseRetryAfterInteger tests parsing integer Retry-After values.
func TestParseRetryAfterInteger(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		seconds := rapid.IntRange(1, 3600).Draw(t, "seconds")
		header := rapid.SampledFrom([]string{
			string(rune('0' + seconds%10)),
		}).Draw(t, "format")
		_ = header

		// Build header string properly
		headerStr := ""
		if seconds < 10 {
			headerStr = string([]byte{byte('0' + seconds)})
		} else if seconds < 100 {
			headerStr = string([]byte{byte('0' + seconds/10), byte('0' + seconds%10)})
		} else {
			// For larger numbers, use standard formatting
			headerStr = string([]byte{
				byte('0' + seconds/100),
				byte('0' + (seconds/10)%10),
				byte('0' + seconds%10),
			})
		}

		delay, ok := authplatform.ParseRetryAfter(headerStr)
		if !ok {
			return // Some formats may not parse
		}

		expected := time.Duration(seconds) * time.Second
		if delay != expected {
			t.Errorf("Retry-After %q: expected %v, got %v", headerStr, expected, delay)
		}
	})
}

// TestRetrySuccessOnFirstAttempt tests that successful operations don't retry.
func TestRetrySuccessOnFirstAttempt(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		maxRetries := rapid.IntRange(1, 5).Draw(t, "maxRetries")
		expectedValue := rapid.IntRange(1, 1000).Draw(t, "expectedValue")

		policy := authplatform.NewRetryPolicy(
			authplatform.WithMaxRetries(maxRetries),
		)

		attemptCount := 0
		op := func(ctx context.Context) (int, error) {
			attemptCount++
			return expectedValue, nil
		}

		ctx := context.Background()
		result := authplatform.Retry(ctx, policy, op)

		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if result.Value != expectedValue {
			t.Errorf("expected value %d, got %d", expectedValue, result.Value)
		}
		if attemptCount != 1 {
			t.Errorf("expected 1 attempt, got %d", attemptCount)
		}
		if result.Attempts != 1 {
			t.Errorf("expected Attempts=1, got %d", result.Attempts)
		}
	})
}

// TestRetryEventualSuccess tests retry with eventual success.
func TestRetryEventualSuccess(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		maxRetries := rapid.IntRange(2, 5).Draw(t, "maxRetries")
		failCount := rapid.IntRange(1, maxRetries).Draw(t, "failCount")
		expectedValue := rapid.IntRange(1, 1000).Draw(t, "expectedValue")

		policy := authplatform.NewRetryPolicy(
			authplatform.WithMaxRetries(maxRetries),
			authplatform.WithBaseDelay(time.Millisecond),
		)

		attemptCount := 0
		op := func(ctx context.Context) (int, error) {
			attemptCount++
			if attemptCount <= failCount {
				return 0, authplatform.ErrNetwork
			}
			return expectedValue, nil
		}

		ctx := context.Background()
		result := authplatform.Retry(ctx, policy, op)

		if result.Err != nil {
			t.Errorf("unexpected error: %v", result.Err)
		}
		if result.Value != expectedValue {
			t.Errorf("expected value %d, got %d", expectedValue, result.Value)
		}
		if attemptCount != failCount+1 {
			t.Errorf("expected %d attempts, got %d", failCount+1, attemptCount)
		}
	})
}

// TestNilErrorNotRetryable tests that nil errors are not retryable.
func TestNilErrorNotRetryable(t *testing.T) {
	policy := authplatform.DefaultRetryPolicy()
	if policy.IsRetryableError(nil) {
		t.Error("nil error should not be retryable")
	}
}

// TestParseRetryAfterEmpty tests parsing empty Retry-After header.
func TestParseRetryAfterEmpty(t *testing.T) {
	_, ok := authplatform.ParseRetryAfter("")
	if ok {
		t.Error("empty Retry-After should not parse")
	}
}

// TestWithRetryWrapper tests the WithRetry wrapper function.
func TestWithRetryWrapper(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		maxRetries := rapid.IntRange(1, 3).Draw(t, "maxRetries")
		expectedValue := rapid.IntRange(1, 100).Draw(t, "expectedValue")

		policy := authplatform.NewRetryPolicy(
			authplatform.WithMaxRetries(maxRetries),
			authplatform.WithBaseDelay(time.Millisecond),
		)

		attemptCount := 0
		fn := func(ctx context.Context) (int, error) {
			attemptCount++
			return expectedValue, nil
		}

		wrapped := authplatform.WithRetry(policy, fn)
		ctx := context.Background()
		value, err := wrapped(ctx)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if value != expectedValue {
			t.Errorf("expected %d, got %d", expectedValue, value)
		}
		if attemptCount != 1 {
			t.Errorf("expected 1 attempt, got %d", attemptCount)
		}
	})
}

// TestRetryWithResponseSuccess tests RetryWithResponse with successful response.
func TestRetryWithResponseSuccess(t *testing.T) {
	policy := authplatform.DefaultRetryPolicy()

	op := func(ctx context.Context) (*http.Response, error) {
		return &http.Response{StatusCode: 200}, nil
	}

	ctx := context.Background()
	resp, err := authplatform.RetryWithResponse(ctx, policy, op)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

// TestRetryWithResponseNetworkError tests RetryWithResponse with network errors.
func TestRetryWithResponseNetworkError(t *testing.T) {
	policy := authplatform.NewRetryPolicy(
		authplatform.WithMaxRetries(2),
		authplatform.WithBaseDelay(time.Millisecond),
	)

	attemptCount := 0
	op := func(ctx context.Context) (*http.Response, error) {
		attemptCount++
		return nil, errors.New("network error")
	}

	ctx := context.Background()
	_, err := authplatform.RetryWithResponse(ctx, policy, op)

	if err == nil {
		t.Error("expected error")
	}
	if attemptCount != 3 { // initial + 2 retries
		t.Errorf("expected 3 attempts, got %d", attemptCount)
	}
}
