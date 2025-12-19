package bulkhead

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

func TestBulkhead_AllowsUpToMaxConcurrent(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("Bulkhead allows up to max concurrent", prop.ForAll(
		func(maxConcurrent int) bool {
			if maxConcurrent < 1 {
				maxConcurrent = 1
			}
			if maxConcurrent > 20 {
				maxConcurrent = 20
			}

			b := New(Config{
				Name:          "test",
				MaxConcurrent: maxConcurrent,
				MaxQueue:      0, // No queue
				QueueTimeout:  0,
			})

			// Should allow exactly maxConcurrent acquires
			for i := 0; i < maxConcurrent; i++ {
				err := b.Acquire(context.Background())
				if err != nil {
					t.Logf("Acquire %d should succeed", i)
					return false
				}
			}

			// Next acquire should fail
			err := b.Acquire(context.Background())
			if err == nil {
				t.Log("Acquire after max should fail")
				return false
			}

			return true
		},
		gen.IntRange(1, 20),
	))

	properties.TestingRun(t)
}

func TestBulkhead_QueueWorks(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("Bulkhead queue allows waiting", prop.ForAll(
		func(maxQueue int) bool {
			if maxQueue < 1 {
				maxQueue = 1
			}
			if maxQueue > 10 {
				maxQueue = 10
			}

			b := New(Config{
				Name:          "test",
				MaxConcurrent: 1,
				MaxQueue:      maxQueue,
				QueueTimeout:  100 * time.Millisecond,
			})

			// Acquire the only slot
			b.Acquire(context.Background())

			// Queue should accept maxQueue requests
			var wg sync.WaitGroup
			errors := make(chan error, maxQueue+1)

			for i := 0; i < maxQueue; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					err := b.Acquire(context.Background())
					errors <- err
				}()
			}

			// Give goroutines time to enter queue
			time.Sleep(10 * time.Millisecond)

			// One more should fail immediately (queue full)
			err := b.Acquire(context.Background())
			if err == nil {
				t.Log("Acquire with full queue should fail")
				return false
			}

			// Release to let queued requests through
			b.Release()

			wg.Wait()
			close(errors)

			// At least one queued request should succeed
			successCount := 0
			for err := range errors {
				if err == nil {
					successCount++
				}
			}

			return successCount >= 1
		},
		gen.IntRange(1, 10),
	))

	properties.TestingRun(t)
}

func TestBulkhead_EmitsEventsOnRejection(t *testing.T) {
	emitter := newMockEmitter()
	builder := domain.NewEventBuilder(emitter, "test-service", nil)

	b := New(Config{
		Name:          "test",
		MaxConcurrent: 1,
		MaxQueue:      0,
		EventBuilder:  builder,
	})

	// Acquire the only slot
	b.Acquire(context.Background())

	// Next acquire should fail and emit event
	b.Acquire(context.Background())

	events := emitter.GetEvents()
	if len(events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(events))
	}

	if len(events) > 0 && events[0].Type != domain.EventBulkheadRejection {
		t.Errorf("Expected EventBulkheadRejection, got %s", events[0].Type)
	}
}

func TestBulkhead_Release(t *testing.T) {
	b := New(Config{
		Name:          "test",
		MaxConcurrent: 1,
		MaxQueue:      0,
	})

	// Acquire
	err := b.Acquire(context.Background())
	if err != nil {
		t.Errorf("First acquire should succeed: %v", err)
	}

	// Second acquire should fail
	err = b.Acquire(context.Background())
	if err == nil {
		t.Error("Second acquire should fail")
	}

	// Release
	b.Release()

	// Now acquire should succeed
	err = b.Acquire(context.Background())
	if err != nil {
		t.Errorf("Acquire after release should succeed: %v", err)
	}
}

func TestBulkhead_Metrics(t *testing.T) {
	b := New(Config{
		Name:          "test",
		MaxConcurrent: 2,
		MaxQueue:      0,
	})

	// Initial metrics
	metrics := b.GetMetrics()
	if metrics.ActiveCount != 0 {
		t.Errorf("Expected 0 active, got %d", metrics.ActiveCount)
	}

	// Acquire one
	b.Acquire(context.Background())
	metrics = b.GetMetrics()
	if metrics.ActiveCount != 1 {
		t.Errorf("Expected 1 active, got %d", metrics.ActiveCount)
	}

	// Acquire another
	b.Acquire(context.Background())
	metrics = b.GetMetrics()
	if metrics.ActiveCount != 2 {
		t.Errorf("Expected 2 active, got %d", metrics.ActiveCount)
	}

	// Try to acquire (should fail and increment rejected)
	b.Acquire(context.Background())
	metrics = b.GetMetrics()
	if metrics.RejectedCount != 1 {
		t.Errorf("Expected 1 rejected, got %d", metrics.RejectedCount)
	}
}

func TestBulkhead_NilEventBuilder(t *testing.T) {
	b := New(Config{
		Name:          "test",
		MaxConcurrent: 1,
		MaxQueue:      0,
		EventBuilder:  nil,
	})

	// Acquire the only slot
	b.Acquire(context.Background())

	// Should not panic with nil event builder
	b.Acquire(context.Background())
}

func TestManager_CreatesBulkheads(t *testing.T) {
	manager := NewManager(domain.BulkheadConfig{
		MaxConcurrent: 5,
		MaxQueue:      2,
		QueueTimeout:  time.Second,
	}, nil)

	b1 := manager.GetBulkhead("partition1")
	b2 := manager.GetBulkhead("partition2")
	b1Again := manager.GetBulkhead("partition1")

	if b1 == b2 {
		t.Error("Different partitions should have different bulkheads")
	}

	if b1 != b1Again {
		t.Error("Same partition should return same bulkhead")
	}
}

func TestManager_GetAllMetrics(t *testing.T) {
	manager := NewManager(domain.BulkheadConfig{
		MaxConcurrent: 5,
		MaxQueue:      2,
		QueueTimeout:  time.Second,
	}, nil)

	// Create some bulkheads
	manager.GetBulkhead("partition1")
	manager.GetBulkhead("partition2")

	metrics := manager.GetAllMetrics()
	if len(metrics) != 2 {
		t.Errorf("Expected 2 partitions, got %d", len(metrics))
	}

	if _, ok := metrics["partition1"]; !ok {
		t.Error("Missing partition1 metrics")
	}

	if _, ok := metrics["partition2"]; !ok {
		t.Error("Missing partition2 metrics")
	}
}
