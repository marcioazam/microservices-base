package timeout

import (
	"context"
	"testing"
	"time"

	"github.com/auth-platform/libs/go/resilience"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// **Feature: resilience-lib-extraction, Property: Timeout Enforcement**
// **Validates: Requirements 1.5**
func TestTimeoutEnforcement(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 50 // Reduced for time-sensitive tests

	properties := gopter.NewProperties(parameters)

	properties.Property("operations exceeding timeout return timeout error", prop.ForAll(
		func(timeoutMs int) bool {
			if timeoutMs < 10 {
				timeoutMs = 10
			}
			if timeoutMs > 100 {
				timeoutMs = 100
			}

			m := New(Config{
				ServiceName: "test",
				Config: resilience.TimeoutConfig{
					Default: time.Duration(timeoutMs) * time.Millisecond,
				},
			})

			ctx := context.Background()
			err := m.Execute(ctx, "test-op", func(ctx context.Context) error {
				// Sleep longer than timeout
				time.Sleep(time.Duration(timeoutMs*2) * time.Millisecond)
				return nil
			})

			// Should return timeout error
			return err != nil
		},
		gen.IntRange(10, 100),
	))

	properties.TestingRun(t)
}

func TestTimeoutGetTimeout(t *testing.T) {
	m := New(Config{
		ServiceName: "test",
		Config: resilience.TimeoutConfig{
			Default: 30 * time.Second,
			Max:     60 * time.Second,
			PerOp: map[string]time.Duration{
				"fast-op": 5 * time.Second,
				"slow-op": 45 * time.Second,
			},
		},
	})

	tests := []struct {
		operation string
		expected  time.Duration
	}{
		{"fast-op", 5 * time.Second},
		{"slow-op", 45 * time.Second},
		{"unknown-op", 30 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.operation, func(t *testing.T) {
			got := m.GetTimeout(tt.operation)
			if got != tt.expected {
				t.Errorf("GetTimeout(%s) = %v, want %v", tt.operation, got, tt.expected)
			}
		})
	}
}

func TestTimeoutWithTimeout(t *testing.T) {
	m := New(Config{
		ServiceName: "test",
		Config: resilience.TimeoutConfig{
			Default: 100 * time.Millisecond,
		},
	})

	ctx := context.Background()
	timeoutCtx, cancel := m.WithTimeout(ctx, "test-op")
	defer cancel()

	deadline, ok := timeoutCtx.Deadline()
	if !ok {
		t.Error("expected context to have deadline")
	}

	// Deadline should be approximately 100ms from now
	remaining := time.Until(deadline)
	if remaining < 50*time.Millisecond || remaining > 150*time.Millisecond {
		t.Errorf("unexpected deadline: %v remaining", remaining)
	}
}

func TestTimeoutExecuteSuccess(t *testing.T) {
	m := New(Config{
		ServiceName: "test",
		Config: resilience.TimeoutConfig{
			Default: time.Second,
		},
	})

	ctx := context.Background()
	err := m.Execute(ctx, "test-op", func(ctx context.Context) error {
		// Fast operation
		return nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}
