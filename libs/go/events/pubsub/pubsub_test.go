package pubsub

import (
	"sync"
	"testing"
)

func TestPubSub(t *testing.T) {
	t.Run("Subscribe and Publish", func(t *testing.T) {
		ps := New[string]()
		received := ""
		ps.Subscribe("topic1", func(event string) {
			received = event
		})
		ps.Publish("topic1", "hello")
		if received != "hello" {
			t.Errorf("expected hello, got %s", received)
		}
	})

	t.Run("Multiple topics", func(t *testing.T) {
		ps := New[string]()
		topic1Received := ""
		topic2Received := ""
		ps.Subscribe("topic1", func(event string) { topic1Received = event })
		ps.Subscribe("topic2", func(event string) { topic2Received = event })

		ps.Publish("topic1", "msg1")
		ps.Publish("topic2", "msg2")

		if topic1Received != "msg1" {
			t.Error("topic1 not received")
		}
		if topic2Received != "msg2" {
			t.Error("topic2 not received")
		}
	})

	t.Run("Unsubscribe", func(t *testing.T) {
		ps := New[int]()
		count := 0
		sub := ps.Subscribe("topic", func(int) { count++ })
		ps.Publish("topic", 1)
		sub.Unsubscribe()
		ps.Publish("topic", 2)
		if count != 1 {
			t.Errorf("expected 1, got %d", count)
		}
	})

	t.Run("SubscribePattern", func(t *testing.T) {
		ps := New[string]()
		received := make([]string, 0)
		var mu sync.Mutex
		ps.SubscribePattern("user.*", func(topic string, event string) {
			mu.Lock()
			received = append(received, topic+":"+event)
			mu.Unlock()
		})

		ps.Publish("user.created", "alice")
		ps.Publish("user.deleted", "bob")
		ps.Publish("order.created", "order1") // Should not match

		if len(received) != 2 {
			t.Errorf("expected 2, got %d", len(received))
		}
	})

	t.Run("Topics", func(t *testing.T) {
		ps := New[int]()
		ps.Subscribe("topic1", func(int) {})
		ps.Subscribe("topic2", func(int) {})
		ps.Subscribe("topic1", func(int) {}) // Duplicate topic

		topics := ps.Topics()
		if len(topics) != 2 {
			t.Errorf("expected 2 topics, got %d", len(topics))
		}
	})

	t.Run("SubscriberCount", func(t *testing.T) {
		ps := New[int]()
		ps.Subscribe("topic", func(int) {})
		ps.Subscribe("topic", func(int) {})

		if ps.SubscriberCount("topic") != 2 {
			t.Errorf("expected 2, got %d", ps.SubscriberCount("topic"))
		}
		if ps.SubscriberCount("nonexistent") != 0 {
			t.Error("expected 0 for nonexistent topic")
		}
	})

	t.Run("Close", func(t *testing.T) {
		ps := New[int]()
		ps.Subscribe("topic", func(int) {})
		ps.Close()

		if !ps.IsClosed() {
			t.Error("expected closed")
		}
		if len(ps.Topics()) != 0 {
			t.Error("expected no topics after close")
		}
	})

	t.Run("PublishAsync", func(t *testing.T) {
		ps := New[int]()
		var wg sync.WaitGroup
		wg.Add(2)
		ps.Subscribe("topic", func(int) { wg.Done() })
		ps.Subscribe("topic", func(int) { wg.Done() })
		ps.PublishAsync("topic", 1)
		wg.Wait()
	})
}

func TestMatchPattern(t *testing.T) {
	tests := []struct {
		pattern string
		topic   string
		match   bool
	}{
		{"user.*", "user.created", true},
		{"user.*", "user.deleted", true},
		{"user.*", "order.created", false},
		{"*.*", "user.created", true},
		{"user.created", "user.created", true},
		{"user.created", "user.deleted", false},
	}

	for _, tt := range tests {
		result := matchPattern(tt.pattern, tt.topic)
		if result != tt.match {
			t.Errorf("matchPattern(%s, %s) = %v, want %v", tt.pattern, tt.topic, result, tt.match)
		}
	}
}
