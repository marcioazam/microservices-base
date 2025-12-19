package eventbus

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestEventBus(t *testing.T) {
	t.Run("Subscribe and Publish", func(t *testing.T) {
		eb := New[int]()
		received := 0
		eb.Subscribe(func(event int) {
			received = event
		})
		eb.Publish(42)
		if received != 42 {
			t.Errorf("expected 42, got %d", received)
		}
	})

	t.Run("Multiple subscribers", func(t *testing.T) {
		eb := New[int]()
		count := 0
		eb.Subscribe(func(event int) { count++ })
		eb.Subscribe(func(event int) { count++ })
		eb.Subscribe(func(event int) { count++ })
		eb.Publish(1)
		if count != 3 {
			t.Errorf("expected 3, got %d", count)
		}
	})

	t.Run("Unsubscribe", func(t *testing.T) {
		eb := New[int]()
		count := 0
		sub := eb.Subscribe(func(event int) { count++ })
		eb.Publish(1)
		sub.Unsubscribe()
		eb.Publish(2)
		if count != 1 {
			t.Errorf("expected 1, got %d", count)
		}
	})

	t.Run("SubscribeFiltered", func(t *testing.T) {
		eb := New[int]()
		received := make([]int, 0)
		eb.SubscribeFiltered(
			func(e int) bool { return e%2 == 0 },
			func(e int) { received = append(received, e) },
		)
		eb.Publish(1)
		eb.Publish(2)
		eb.Publish(3)
		eb.Publish(4)
		if len(received) != 2 || received[0] != 2 || received[1] != 4 {
			t.Error("unexpected filtered events")
		}
	})

	t.Run("PublishAsync", func(t *testing.T) {
		eb := New[int]()
		var count int32
		var wg sync.WaitGroup
		wg.Add(3)
		for i := 0; i < 3; i++ {
			eb.Subscribe(func(event int) {
				atomic.AddInt32(&count, 1)
				wg.Done()
			})
		}
		eb.PublishAsync(1)
		wg.Wait()
		if atomic.LoadInt32(&count) != 3 {
			t.Errorf("expected 3, got %d", count)
		}
	})

	t.Run("PublishAsyncWait", func(t *testing.T) {
		eb := New[int]()
		var count int32
		for i := 0; i < 3; i++ {
			eb.Subscribe(func(event int) {
				time.Sleep(10 * time.Millisecond)
				atomic.AddInt32(&count, 1)
			})
		}
		eb.PublishAsyncWait(1)
		if atomic.LoadInt32(&count) != 3 {
			t.Errorf("expected 3, got %d", count)
		}
	})

	t.Run("SubscriberCount", func(t *testing.T) {
		eb := New[int]()
		eb.Subscribe(func(int) {})
		eb.Subscribe(func(int) {})
		if eb.SubscriberCount() != 2 {
			t.Errorf("expected 2, got %d", eb.SubscriberCount())
		}
	})

	t.Run("Close", func(t *testing.T) {
		eb := New[int]()
		eb.Subscribe(func(int) {})
		eb.Close()
		if !eb.IsClosed() {
			t.Error("expected closed")
		}
		if eb.SubscriberCount() != 0 {
			t.Error("expected no subscribers after close")
		}
	})

	t.Run("SubscribeOnce", func(t *testing.T) {
		eb := New[int]()
		count := 0
		eb.SubscribeOnce(func(int) { count++ })
		eb.Publish(1)
		eb.Publish(2)
		eb.Publish(3)
		if count != 1 {
			t.Errorf("expected 1, got %d", count)
		}
	})

	t.Run("SubscribeChannel", func(t *testing.T) {
		eb := New[int]()
		ch, sub := eb.SubscribeChannel(10)
		defer sub.Unsubscribe()

		eb.Publish(1)
		eb.Publish(2)

		if v := <-ch; v != 1 {
			t.Errorf("expected 1, got %d", v)
		}
		if v := <-ch; v != 2 {
			t.Errorf("expected 2, got %d", v)
		}
	})
}
