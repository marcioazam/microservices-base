package pubsub

import (
	"sync"
	"testing"

	"github.com/authcorp/libs/go/src/events"
)

func TestPubSub(t *testing.T) {
	t.Run("SubscribeTopic and PublishToTopic", func(t *testing.T) {
		ps := events.NewPubSub[string]()
		received := ""
		ps.SubscribeTopic("topic1", func(event string) {
			received = event
		})
		ps.PublishToTopic("topic1", "hello")
		if received != "hello" {
			t.Errorf("expected hello, got %s", received)
		}
	})

	t.Run("Multiple topics", func(t *testing.T) {
		ps := events.NewPubSub[string]()
		topic1Received := ""
		topic2Received := ""
		ps.SubscribeTopic("topic1", func(event string) { topic1Received = event })
		ps.SubscribeTopic("topic2", func(event string) { topic2Received = event })

		ps.PublishToTopic("topic1", "msg1")
		ps.PublishToTopic("topic2", "msg2")

		if topic1Received != "msg1" {
			t.Error("topic1 not received")
		}
		if topic2Received != "msg2" {
			t.Error("topic2 not received")
		}
	})

	t.Run("Unsubscribe", func(t *testing.T) {
		ps := events.NewPubSub[int]()
		count := 0
		sub := ps.SubscribeTopic("topic", func(int) { count++ })
		ps.PublishToTopic("topic", 1)
		sub.Unsubscribe()
		ps.PublishToTopic("topic", 2)
		if count != 1 {
			t.Errorf("expected 1, got %d", count)
		}
	})

	t.Run("SubscribePattern", func(t *testing.T) {
		ps := events.NewPubSub[string]()
		received := make([]string, 0)
		var mu sync.Mutex
		ps.SubscribePattern("user.*", func(topic string, event string) {
			mu.Lock()
			received = append(received, topic+":"+event)
			mu.Unlock()
		})

		ps.PublishToTopic("user.created", "alice")
		ps.PublishToTopic("user.deleted", "bob")
		ps.PublishToTopic("order.created", "order1") // Should not match

		if len(received) != 2 {
			t.Errorf("expected 2, got %d", len(received))
		}
	})

	t.Run("Topics", func(t *testing.T) {
		ps := events.NewPubSub[int]()
		ps.SubscribeTopic("topic1", func(int) {})
		ps.SubscribeTopic("topic2", func(int) {})
		ps.SubscribeTopic("topic1", func(int) {}) // Duplicate topic

		topics := ps.Topics()
		if len(topics) != 2 {
			t.Errorf("expected 2 topics, got %d", len(topics))
		}
	})

	t.Run("TopicSubscriberCount", func(t *testing.T) {
		ps := events.NewPubSub[int]()
		ps.SubscribeTopic("topic", func(int) {})
		ps.SubscribeTopic("topic", func(int) {})

		if ps.TopicSubscriberCount("topic") != 2 {
			t.Errorf("expected 2, got %d", ps.TopicSubscriberCount("topic"))
		}
		if ps.TopicSubscriberCount("nonexistent") != 0 {
			t.Error("expected 0 for nonexistent topic")
		}
	})

	t.Run("ClosePubSub", func(t *testing.T) {
		ps := events.NewPubSub[int]()
		ps.SubscribeTopic("topic", func(int) {})
		ps.ClosePubSub()

		if !ps.IsPubSubClosed() {
			t.Error("expected closed")
		}
		if len(ps.Topics()) != 0 {
			t.Error("expected no topics after close")
		}
	})

	t.Run("PublishToTopicAsync", func(t *testing.T) {
		ps := events.NewPubSub[int]()
		var wg sync.WaitGroup
		wg.Add(2)
		ps.SubscribeTopic("topic", func(int) { wg.Done() })
		ps.SubscribeTopic("topic", func(int) { wg.Done() })
		ps.PublishToTopicAsync("topic", 1)
		wg.Wait()
	})
}
