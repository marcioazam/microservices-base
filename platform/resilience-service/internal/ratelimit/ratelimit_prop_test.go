package ratelimit

import (
	"context"
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

func TestTokenBucket_AllowsUpToCapacity(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("Token bucket allows requests up to capacity", prop.ForAll(
		func(capacity int) bool {
			if capacity < 1 {
				capacity = 1
			}
			if capacity > 100 {
				capacity = 100
			}

			bucket := NewTokenBucket(TokenBucketConfig{
				Capacity:   capacity,
				RefillRate: capacity,
				Window:     time.Second,
			})

			// Should allow exactly capacity requests
			for i := 0; i < capacity; i++ {
				decision, _ := bucket.Allow(context.Background(), "test")
				if !decision.Allowed {
					t.Logf("Request %d should be allowed", i)
					return false
				}
			}

			// Next request should be denied
			decision, _ := bucket.Allow(context.Background(), "test")
			if decision.Allowed {
				t.Log("Request after capacity should be denied")
				return false
			}

			return true
		},
		gen.IntRange(1, 100),
	))

	properties.TestingRun(t)
}

func TestTokenBucket_RefillsOverTime(t *testing.T) {
	bucket := NewTokenBucket(TokenBucketConfig{
		Capacity:   10,
		RefillRate: 100, // 100 tokens per second
		Window:     time.Second,
	})

	// Drain all tokens
	for i := 0; i < 10; i++ {
		bucket.Allow(context.Background(), "test")
	}

	// Wait for refill
	time.Sleep(100 * time.Millisecond)

	// Should have some tokens now
	tokens := bucket.GetTokenCount()
	if tokens < 5 {
		t.Errorf("Expected at least 5 tokens after 100ms, got %f", tokens)
	}
}

func TestTokenBucket_EmitsEventsOnDenial(t *testing.T) {
	emitter := newMockEmitter()
	builder := domain.NewEventBuilder(emitter, "test-service", nil)

	bucket := NewTokenBucket(TokenBucketConfig{
		Capacity:     1,
		RefillRate:   1,
		Window:       time.Second,
		EventBuilder: builder,
	})

	// First request allowed
	bucket.Allow(context.Background(), "test")

	// Second request denied - should emit event
	bucket.Allow(context.Background(), "test")

	events := emitter.GetEvents()
	if len(events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(events))
	}

	if len(events) > 0 && events[0].Type != domain.EventRateLimitHit {
		t.Errorf("Expected EventRateLimitHit, got %s", events[0].Type)
	}
}

func TestSlidingWindow_AllowsUpToLimit(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("Sliding window allows requests up to limit", prop.ForAll(
		func(limit int) bool {
			if limit < 1 {
				limit = 1
			}
			if limit > 50 {
				limit = 50
			}

			sw := NewSlidingWindow(SlidingWindowConfig{
				Limit:  limit,
				Window: time.Second,
			})

			// Should allow exactly limit requests
			for i := 0; i < limit; i++ {
				decision, _ := sw.Allow(context.Background(), "test")
				if !decision.Allowed {
					t.Logf("Request %d should be allowed", i)
					return false
				}
			}

			// Next request should be denied
			decision, _ := sw.Allow(context.Background(), "test")
			if decision.Allowed {
				t.Log("Request after limit should be denied")
				return false
			}

			return true
		},
		gen.IntRange(1, 50),
	))

	properties.TestingRun(t)
}

func TestSlidingWindow_ExpiresOldRequests(t *testing.T) {
	sw := NewSlidingWindow(SlidingWindowConfig{
		Limit:  5,
		Window: 50 * time.Millisecond,
	})

	// Make 5 requests
	for i := 0; i < 5; i++ {
		sw.Allow(context.Background(), "test")
	}

	// Should be at limit
	decision, _ := sw.Allow(context.Background(), "test")
	if decision.Allowed {
		t.Error("Should be at limit")
	}

	// Wait for window to expire
	time.Sleep(60 * time.Millisecond)

	// Should allow again
	decision, _ = sw.Allow(context.Background(), "test")
	if !decision.Allowed {
		t.Error("Should allow after window expires")
	}
}

func TestSlidingWindow_EmitsEventsOnDenial(t *testing.T) {
	emitter := newMockEmitter()
	builder := domain.NewEventBuilder(emitter, "test-service", nil)

	sw := NewSlidingWindow(SlidingWindowConfig{
		Limit:        1,
		Window:       time.Second,
		EventBuilder: builder,
	})

	// First request allowed
	sw.Allow(context.Background(), "test")

	// Second request denied - should emit event
	sw.Allow(context.Background(), "test")

	events := emitter.GetEvents()
	if len(events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(events))
	}

	if len(events) > 0 && events[0].Type != domain.EventRateLimitHit {
		t.Errorf("Expected EventRateLimitHit, got %s", events[0].Type)
	}
}

func TestRateLimiter_NilEventBuilder(t *testing.T) {
	// Token bucket with nil builder
	bucket := NewTokenBucket(TokenBucketConfig{
		Capacity:     1,
		RefillRate:   1,
		Window:       time.Second,
		EventBuilder: nil,
	})

	bucket.Allow(context.Background(), "test")
	bucket.Allow(context.Background(), "test") // Should not panic

	// Sliding window with nil builder
	sw := NewSlidingWindow(SlidingWindowConfig{
		Limit:        1,
		Window:       time.Second,
		EventBuilder: nil,
	})

	sw.Allow(context.Background(), "test")
	sw.Allow(context.Background(), "test") // Should not panic
}
