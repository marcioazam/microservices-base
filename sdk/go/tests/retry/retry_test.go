// Package retry provides unit tests for retry logic.
package retry

import (
	"context"
	"testing"
	"time"

	"github.com/auth-platform/sdk-go/src/errors"
	"github.com/auth-platform/sdk-go/src/retry"
)

func TestDefaultPolicy(t *testing.T) {
	p := retry.DefaultPolicy()

	if p.MaxRetries != 3 {
		t.Errorf("MaxRetries = %d, want 3", p.MaxRetries)
	}
	if p.BaseDelay != time.Second {
		t.Errorf("BaseDelay = %v, want 1s", p.BaseDelay)
	}
	if p.MaxDelay != 30*time.Second {
		t.Errorf("MaxDelay = %v, want 30s", p.MaxDelay)
	}
	if p.Jitter != 0.2 {
		t.Errorf("Jitter = %v, want 0.2", p.Jitter)
	}
}

func TestNewPolicy_WithOptions(t *testing.T) {
	p := retry.NewPolicy(
		retry.WithMaxRetries(5),
		retry.WithBaseDelay(500*time.Millisecond),
		retry.WithMaxDelay(10*time.Second),
		retry.WithJitter(0.1),
	)

	if p.MaxRetries != 5 {
		t.Errorf("MaxRetries = %d, want 5", p.MaxRetries)
	}
	if p.BaseDelay != 500*time.Millisecond {
		t.Errorf("BaseDelay = %v, want 500ms", p.BaseDelay)
	}
	if p.MaxDelay != 10*time.Second {
		t.Errorf("MaxDelay = %v, want 10s", p.MaxDelay)
	}
	if p.Jitter != 0.1 {
		t.Errorf("Jitter = %v, want 0.1", p.Jitter)
	}
}

func TestPolicy_CalculateDelay(t *testing.T) {
	p := retry.NewPolicy(
		retry.WithBaseDelay(100*time.Millisecond),
		retry.WithMaxDelay(10*time.Second),
		retry.WithJitter(0),
	)

	tests := []struct {
		attempt int
		want    time.Duration
	}{
		{0, 100 * time.Millisecond},
		{1, 200 * time.Millisecond},
		{2, 400 * time.Millisecond},
		{3, 800 * time.Millisecond},
	}

	for _, tt := range tests {
		delay := p.CalculateDelay(tt.attempt)
		if delay != tt.want {
			t.Errorf("CalculateDelay(%d) = %v, want %v", tt.attempt, delay, tt.want)
		}
	}
}

func TestPolicy_CalculateDelay_MaxDelay(t *testing.T) {
	p := retry.NewPolicy(
		retry.WithBaseDelay(100*time.Millisecond),
		retry.WithMaxDelay(500*time.Millisecond),
		retry.WithJitter(0),
	)

	delay := p.CalculateDelay(10) // Would be 100ms * 2^10 = 102.4s without cap
	if delay != 500*time.Millisecond {
		t.Errorf("delay = %v, want 500ms (capped)", delay)
	}
}

func TestPolicy_IsRetryable(t *testing.T) {
	p := retry.DefaultPolicy()

	retryable := []int{429, 502, 503, 504}
	for _, code := range retryable {
		if !p.IsRetryable(code) {
			t.Errorf("status %d should be retryable", code)
		}
	}

	nonRetryable := []int{200, 400, 401, 403, 404, 500}
	for _, code := range nonRetryable {
		if p.IsRetryable(code) {
			t.Errorf("status %d should not be retryable", code)
		}
	}
}

func TestParseRetryAfter_Seconds(t *testing.T) {
	tests := []struct {
		header string
		want   time.Duration
		ok     bool
	}{
		{"120", 120 * time.Second, true},
		{"1", time.Second, true},
		{"0", 0, true},
		{"", 0, false},
		{"invalid", 0, false},
	}

	for _, tt := range tests {
		duration, ok := retry.ParseRetryAfter(tt.header)
		if ok != tt.ok {
			t.Errorf("ParseRetryAfter(%q) ok = %v, want %v", tt.header, ok, tt.ok)
		}
		if ok && duration != tt.want {
			t.Errorf("ParseRetryAfter(%q) = %v, want %v", tt.header, duration, tt.want)
		}
	}
}

func TestRetry_ImmediateSuccess(t *testing.T) {
	p := retry.DefaultPolicy()
	ctx := context.Background()

	callCount := 0
	result := retry.Retry(ctx, p, func(ctx context.Context) (int, error) {
		callCount++
		return 42, nil
	})

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if result.Value != 42 {
		t.Errorf("value = %d, want 42", result.Value)
	}
	if result.Attempts != 1 {
		t.Errorf("attempts = %d, want 1", result.Attempts)
	}
	if callCount != 1 {
		t.Errorf("callCount = %d, want 1", callCount)
	}
}

func TestRetry_EventualSuccess(t *testing.T) {
	p := retry.NewPolicy(
		retry.WithMaxRetries(3),
		retry.WithBaseDelay(time.Millisecond),
	)
	ctx := context.Background()

	callCount := 0
	result := retry.Retry(ctx, p, func(ctx context.Context) (int, error) {
		callCount++
		if callCount < 3 {
			return 0, errors.NewError(errors.ErrCodeNetwork, "temporary failure")
		}
		return 42, nil
	})

	if result.Err != nil {
		t.Fatalf("unexpected error: %v", result.Err)
	}
	if result.Value != 42 {
		t.Errorf("value = %d, want 42", result.Value)
	}
	if result.Attempts != 3 {
		t.Errorf("attempts = %d, want 3", result.Attempts)
	}
}

func TestRetry_Exhaustion(t *testing.T) {
	p := retry.NewPolicy(
		retry.WithMaxRetries(2),
		retry.WithBaseDelay(time.Millisecond),
	)
	ctx := context.Background()

	callCount := 0
	result := retry.Retry(ctx, p, func(ctx context.Context) (int, error) {
		callCount++
		return 0, errors.NewError(errors.ErrCodeNetwork, "always fails")
	})

	if result.Err == nil {
		t.Fatal("expected error after exhaustion")
	}
	if callCount != 3 { // initial + 2 retries
		t.Errorf("callCount = %d, want 3", callCount)
	}
}

func TestRetry_NonRetryableError(t *testing.T) {
	p := retry.DefaultPolicy()
	ctx := context.Background()

	callCount := 0
	result := retry.Retry(ctx, p, func(ctx context.Context) (int, error) {
		callCount++
		return 0, errors.NewError(errors.ErrCodeValidation, "validation error")
	})

	if result.Err == nil {
		t.Fatal("expected error")
	}
	if callCount != 1 {
		t.Errorf("callCount = %d, want 1 (no retry for non-retryable)", callCount)
	}
}

func TestRetry_ContextCancellation(t *testing.T) {
	p := retry.NewPolicy(
		retry.WithMaxRetries(10),
		retry.WithBaseDelay(time.Second),
	)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	result := retry.Retry(ctx, p, func(ctx context.Context) (int, error) {
		return 0, errors.NewError(errors.ErrCodeNetwork, "should not reach")
	})

	if result.Err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestWithRetry_Wrapper(t *testing.T) {
	p := retry.NewPolicy(
		retry.WithMaxRetries(2),
		retry.WithBaseDelay(time.Millisecond),
	)

	callCount := 0
	fn := retry.WithRetry(p, func(ctx context.Context) (string, error) {
		callCount++
		if callCount < 2 {
			return "", errors.NewError(errors.ErrCodeNetwork, "fail")
		}
		return "success", nil
	})

	result, err := fn(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "success" {
		t.Errorf("result = %s, want success", result)
	}
}
