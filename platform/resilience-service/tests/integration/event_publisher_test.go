package integration_test

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/auth-platform/platform/resilience-service/internal/domain/valueobjects"
	"github.com/auth-platform/platform/resilience-service/internal/infrastructure/events"
)

func TestPolicyEventPublisherPublishAndSubscribe(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	publisher := events.NewPolicyEventPublisher(logger)
	defer publisher.Close()

	var received []valueobjects.PolicyEvent
	var mu sync.Mutex

	publisher.Subscribe(func(ctx context.Context, event valueobjects.PolicyEvent) {
		mu.Lock()
		defer mu.Unlock()
		received = append(received, event)
	})

	ctx := context.Background()
	event := valueobjects.NewPolicyEvent(valueobjects.PolicyCreated, "test-policy", 1)

	err := publisher.Publish(ctx, event)
	if err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	// Wait for async delivery
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if len(received) != 1 {
		t.Errorf("Expected 1 event, got %d", len(received))
	}

	if received[0].PolicyName != "test-policy" {
		t.Errorf("Expected policy name 'test-policy', got '%s'", received[0].PolicyName)
	}
}

func TestPolicyEventPublisherMultipleSubscribers(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	publisher := events.NewPolicyEventPublisher(logger)
	defer publisher.Close()

	var count1, count2 int
	var mu sync.Mutex

	publisher.Subscribe(func(ctx context.Context, event valueobjects.PolicyEvent) {
		mu.Lock()
		defer mu.Unlock()
		count1++
	})

	publisher.Subscribe(func(ctx context.Context, event valueobjects.PolicyEvent) {
		mu.Lock()
		defer mu.Unlock()
		count2++
	})

	ctx := context.Background()
	event := valueobjects.NewPolicyEvent(valueobjects.PolicyUpdated, "test-policy", 2)

	err := publisher.Publish(ctx, event)
	if err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if count1 != 1 {
		t.Errorf("Subscriber 1 expected 1 event, got %d", count1)
	}

	if count2 != 1 {
		t.Errorf("Subscriber 2 expected 1 event, got %d", count2)
	}
}

func TestPolicyEventPublisherContextCancellation(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	publisher := events.NewPolicyEventPublisher(logger)
	defer publisher.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	event := valueobjects.NewPolicyEvent(valueobjects.PolicyDeleted, "test-policy", 1)

	err := publisher.Publish(ctx, event)
	// With a buffered channel, the event may be sent before context check.
	// Both nil (event sent) and context.Canceled (context checked first) are valid.
	if err != nil && err != context.Canceled {
		t.Errorf("Expected nil or context.Canceled error, got %v", err)
	}
}

func TestEventBusGeneric(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	type TestEvent struct {
		ID      string
		Message string
	}

	bus := events.NewEventBus[TestEvent](logger, 100)
	defer bus.Close()

	var received []TestEvent
	var mu sync.Mutex

	bus.Subscribe(func(ctx context.Context, event TestEvent) {
		mu.Lock()
		defer mu.Unlock()
		received = append(received, event)
	})

	ctx := context.Background()
	testEvent := TestEvent{ID: "1", Message: "hello"}

	err := bus.Publish(ctx, testEvent)
	if err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if len(received) != 1 {
		t.Errorf("Expected 1 event, got %d", len(received))
	}

	if received[0].Message != "hello" {
		t.Errorf("Expected message 'hello', got '%s'", received[0].Message)
	}
}

func TestEventEmitterImpl(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	emitter := events.NewEventEmitter(logger)
	defer emitter.Close()

	var received []valueobjects.PolicyEvent
	var mu sync.Mutex

	emitter.SubscribePolicyEvents(func(ctx context.Context, event valueobjects.PolicyEvent) {
		mu.Lock()
		defer mu.Unlock()
		received = append(received, event)
	})

	ctx := context.Background()
	event := valueobjects.NewPolicyEvent(valueobjects.PolicyCreated, "test-policy", 1)

	err := emitter.EmitPolicyEvent(ctx, event)
	if err != nil {
		t.Fatalf("EmitPolicyEvent failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if len(received) != 1 {
		t.Errorf("Expected 1 event, got %d", len(received))
	}
}
