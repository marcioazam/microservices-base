package retry

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"

	"github.com/auth-platform/platform/resilience-service/internal/domain"
)

// mockEmitter is a test implementation of EventEmitter.
type mockEmitter struct {
	mu     sync.Mutex
	events []domain.ResilienceEvent
}

func newMockEmitter() *mockEmitter {
	return &mockEmitter{events: make([]domain.ResilienceEvent, 0)}
}

func (m *mockEmitter) Emit(event domain.ResilienceEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, event)
}

func (m *mockEmitter) EmitAudit(event domain.AuditEvent) {}

func (m *mockEmitter) GetEvents() []domain.ResilienceEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]domain.ResilienceEvent, len(m.events))
	copy(result, m.events)
	return result
}

func TestRetryHandler_RetriesOnFailure(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("Retry handler retries up to max attempts", prop.ForAll(
		func(maxAttempts int) bool {
			if maxAttempts < 1 {
				maxAttempts = 1
			}
			if maxAttempts > 5 {
				maxAttempts = 5
			}

			attempts := 0
			handler := New(Config{
				ServiceName: "test",
				Config: domain.RetryConfig{
					MaxAttempts:   maxAttempts,
					BaseDelay:     time.Millisecond,
					MaxDelay:      10 * time.Millisecond,
					Multiplier:    1.5,
					JitterPercent: 0.1,
				},
			})

			err := handler.Execute(context.Background(), func() error {
				attempts++
				return errors.New("always fail")
			})

			if err == nil {
				t.Log("Expected error")
				return false
			}

			if attempts != maxAttempts {
				t.Logf("Expected %d attempts, got %d", maxAttempts, attempts)
				return false
			}

			return true
		},
		gen.IntRange(1, 5),
	))

	properties.TestingRun(t)
}

func TestRetryHandler_SucceedsOnFirstAttempt(t *testing.T) {
	handler := New(Config{
		ServiceName: "test",
		Config: domain.RetryConfig{
			MaxAttempts:   3,
			BaseDelay:     time.Millisecond,
			MaxDelay:      10 * time.Millisecond,
			Multiplier:    2.0,
			JitterPercent: 0.1,
		},
	})

	attempts := 0
	err := handler.Execute(context.Background(), func() error {
		attempts++
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if attempts != 1 {
		t.Errorf("Expected 1 attempt, got %d", attempts)
	}
}

func TestRetryHandler_SucceedsAfterRetries(t *testing.T) {
	handler := New(Config{
		ServiceName: "test",
		Config: domain.RetryConfig{
			MaxAttempts:   5,
			BaseDelay:     time.Millisecond,
			MaxDelay:      10 * time.Millisecond,
			Multiplier:    1.5,
			JitterPercent: 0.1,
		},
	})

	attempts := 0
	err := handler.Execute(context.Background(), func() error {
		attempts++
		if attempts < 3 {
			return errors.New("fail")
		}
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

func TestRetryHandler_EmitsEventsOnRetry(t *testing.T) {
	emitter := newMockEmitter()
	builder := domain.NewEventBuilder(emitter, "test-service", nil)

	handler := New(Config{
		ServiceName:  "test",
		EventBuilder: builder,
		Config: domain.RetryConfig{
			MaxAttempts:   3,
			BaseDelay:     time.Millisecond,
			MaxDelay:      10 * time.Millisecond,
			Multiplier:    1.5,
			JitterPercent: 0.1,
		},
	})

	handler.Execute(context.Background(), func() error {
		return errors.New("fail")
	})

	events := emitter.GetEvents()
	// Should emit events for attempts 1 and 2 (not the last one)
	if len(events) != 2 {
		t.Errorf("Expected 2 events, got %d", len(events))
	}

	for _, event := range events {
		if event.Type != domain.EventRetryAttempt {
			t.Errorf("Expected EventRetryAttempt, got %s", event.Type)
		}
	}
}

func TestRetryHandler_RespectsContextCancellation(t *testing.T) {
	handler := New(Config{
		ServiceName: "test",
		Config: domain.RetryConfig{
			MaxAttempts:   10,
			BaseDelay:     100 * time.Millisecond,
			MaxDelay:      time.Second,
			Multiplier:    2.0,
			JitterPercent: 0.1,
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	attempts := 0

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := handler.Execute(ctx, func() error {
		attempts++
		return errors.New("fail")
	})

	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
}

func TestRetryHandler_CalculateDelay(t *testing.T) {
	handler := New(Config{
		ServiceName: "test",
		Config: domain.RetryConfig{
			MaxAttempts:   5,
			BaseDelay:     100 * time.Millisecond,
			MaxDelay:      time.Second,
			Multiplier:    2.0,
			JitterPercent: 0.1,
		},
	})

	// Test exponential backoff
	delay0 := handler.CalculateDelay(0)
	delay1 := handler.CalculateDelay(1)
	delay2 := handler.CalculateDelay(2)

	// With jitter, delays should be approximately:
	// attempt 0: ~100ms
	// attempt 1: ~200ms
	// attempt 2: ~400ms

	if delay0 < 80*time.Millisecond || delay0 > 120*time.Millisecond {
		t.Errorf("Delay 0 out of expected range: %v", delay0)
	}

	if delay1 < 160*time.Millisecond || delay1 > 240*time.Millisecond {
		t.Errorf("Delay 1 out of expected range: %v", delay1)
	}

	if delay2 < 320*time.Millisecond || delay2 > 480*time.Millisecond {
		t.Errorf("Delay 2 out of expected range: %v", delay2)
	}
}

func TestRetryHandler_NilEventBuilder(t *testing.T) {
	handler := New(Config{
		ServiceName:  "test",
		EventBuilder: nil,
		Config: domain.RetryConfig{
			MaxAttempts:   2,
			BaseDelay:     time.Millisecond,
			MaxDelay:      10 * time.Millisecond,
			Multiplier:    1.5,
			JitterPercent: 0.1,
		},
	})

	// Should not panic with nil event builder
	err := handler.Execute(context.Background(), func() error {
		return errors.New("fail")
	})

	if err == nil {
		t.Error("Expected error")
	}
}
