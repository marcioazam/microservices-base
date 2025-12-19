package events_test

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/authcorp/libs/go/src/events"
	"pgregory.net/rapid"
)

// TestEvent implements Event interface for testing.
type TestEvent struct {
	EventType string
	Data      string
}

func (e TestEvent) Type() string {
	return e.EventType
}

// Property 22: Event Sync Delivery
// Sync events are delivered to all subscribers before Publish returns.
func TestProperty_EventSyncDelivery(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		eventType := rapid.StringMatching(`[a-z]+`).Draw(t, "eventType")
		data := rapid.String().Draw(t, "data")
		subscriberCount := rapid.IntRange(1, 5).Draw(t, "subscriberCount")

		bus := events.NewEventBus[TestEvent]()
		var deliveryCount atomic.Int32

		for i := 0; i < subscriberCount; i++ {
			bus.Subscribe(eventType, func(ctx context.Context, e TestEvent) error {
				deliveryCount.Add(1)
				return nil
			})
		}

		event := TestEvent{EventType: eventType, Data: data}
		err := bus.Publish(context.Background(), event)

		if err != nil {
			t.Fatalf("Publish should not error: %v", err)
		}

		// All handlers should have been called synchronously
		if int(deliveryCount.Load()) != subscriberCount {
			t.Fatalf("Expected %d deliveries, got %d", subscriberCount, deliveryCount.Load())
		}
	})
}

// Property 23: Event Filtering
// Filtered events are only delivered to matching handlers.
func TestProperty_EventFiltering(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		eventType := rapid.StringMatching(`[a-z]+`).Draw(t, "eventType")
		matchData := rapid.StringMatching(`match-[a-z]+`).Draw(t, "matchData")
		noMatchData := rapid.StringMatching(`nomatch-[a-z]+`).Draw(t, "noMatchData")

		bus := events.NewEventBus[TestEvent]()
		var matchCount, noMatchCount atomic.Int32

		// Handler with filter
		bus.SubscribeWithFilter(eventType, func(ctx context.Context, e TestEvent) error {
			matchCount.Add(1)
			return nil
		}, func(e TestEvent) bool {
			return len(e.Data) > 0 && e.Data[0] == 'm'
		})

		// Handler without filter
		bus.Subscribe(eventType, func(ctx context.Context, e TestEvent) error {
			noMatchCount.Add(1)
			return nil
		})

		// Publish matching event
		bus.Publish(context.Background(), TestEvent{EventType: eventType, Data: matchData})

		if matchCount.Load() != 1 {
			t.Fatalf("Filtered handler should receive matching event")
		}
		if noMatchCount.Load() != 1 {
			t.Fatalf("Unfiltered handler should receive all events")
		}

		// Publish non-matching event
		bus.Publish(context.Background(), TestEvent{EventType: eventType, Data: noMatchData})

		if matchCount.Load() != 1 {
			t.Fatalf("Filtered handler should not receive non-matching event")
		}
		if noMatchCount.Load() != 2 {
			t.Fatalf("Unfiltered handler should receive all events")
		}
	})
}

// Property: Subscription cancellation
func TestProperty_SubscriptionCancellation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		eventType := rapid.StringMatching(`[a-z]+`).Draw(t, "eventType")

		bus := events.NewEventBus[TestEvent]()
		var count atomic.Int32

		sub := bus.Subscribe(eventType, func(ctx context.Context, e TestEvent) error {
			count.Add(1)
			return nil
		})

		// First publish should work
		bus.Publish(context.Background(), TestEvent{EventType: eventType})
		if count.Load() != 1 {
			t.Fatalf("Handler should be called before cancel")
		}

		// Cancel subscription
		sub.Cancel()

		// Second publish should not call handler
		bus.Publish(context.Background(), TestEvent{EventType: eventType})
		if count.Load() != 1 {
			t.Fatalf("Handler should not be called after cancel")
		}
	})
}

// Property: HasSubscribers accuracy
func TestProperty_HasSubscribers(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		eventType := rapid.StringMatching(`[a-z]+`).Draw(t, "eventType")

		bus := events.NewEventBus[TestEvent]()

		if bus.HasSubscribers(eventType) {
			t.Fatalf("Should have no subscribers initially")
		}

		sub := bus.Subscribe(eventType, func(ctx context.Context, e TestEvent) error {
			return nil
		})

		if !bus.HasSubscribers(eventType) {
			t.Fatalf("Should have subscribers after Subscribe")
		}

		sub.Cancel()

		if bus.HasSubscribers(eventType) {
			t.Fatalf("Should have no subscribers after Cancel")
		}
	})
}
