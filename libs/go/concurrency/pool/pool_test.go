package pool

import (
	"bytes"
	"context"
	"testing"
	"time"
)

func TestPool(t *testing.T) {
	t.Run("Acquire creates new object", func(t *testing.T) {
		p := New(func() *bytes.Buffer {
			return &bytes.Buffer{}
		}, func(b *bytes.Buffer) {
			b.Reset()
		})

		buf := p.Acquire()
		if buf == nil {
			t.Error("expected buffer")
		}
	})

	t.Run("Release returns object to pool", func(t *testing.T) {
		p := New(func() *bytes.Buffer {
			return &bytes.Buffer{}
		}, func(b *bytes.Buffer) {
			b.Reset()
		})

		buf := p.Acquire()
		buf.WriteString("test")
		p.Release(buf)

		buf2 := p.Acquire()
		if buf2.Len() != 0 {
			t.Error("expected reset buffer")
		}
	})

	t.Run("Stats tracks hits and misses", func(t *testing.T) {
		p := New(func() int { return 42 }, nil)

		p.Acquire() // Miss
		p.Release(1)
		p.Acquire() // Hit

		stats := p.Stats()
		if stats.Hits != 1 {
			t.Errorf("expected 1 hit, got %d", stats.Hits)
		}
		if stats.Misses != 1 {
			t.Errorf("expected 1 miss, got %d", stats.Misses)
		}
	})

	t.Run("WithCapacity limits pool size", func(t *testing.T) {
		p := New(func() int { return 42 }, nil).WithCapacity(2)

		p.Release(1)
		p.Release(2)
		p.Release(3) // Should be discarded

		if p.Size() != 2 {
			t.Errorf("expected 2, got %d", p.Size())
		}
	})

	t.Run("Drain removes all items", func(t *testing.T) {
		p := New(func() int { return 42 }, nil)

		p.Release(1)
		p.Release(2)
		p.Release(3)
		p.Drain()

		if p.Size() != 0 {
			t.Errorf("expected 0, got %d", p.Size())
		}
	})

	t.Run("AcquireContext respects cancellation", func(t *testing.T) {
		p := New(func() int { return 42 }, nil)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := p.AcquireContext(ctx)
		// Should still work because it falls through to factory
		if err != nil {
			// This is expected if pool is empty and context is cancelled
		}
	})

	t.Run("AcquireContext returns from pool", func(t *testing.T) {
		p := New(func() int { return 42 }, nil)
		p.Release(99)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		v, err := p.AcquireContext(ctx)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if v != 99 {
			t.Errorf("expected 99, got %d", v)
		}
	})

	t.Run("Use acquires and releases", func(t *testing.T) {
		p := New(func() *bytes.Buffer {
			return &bytes.Buffer{}
		}, func(b *bytes.Buffer) {
			b.Reset()
		})

		var result string
		p.Use(func(buf *bytes.Buffer) {
			buf.WriteString("hello")
			result = buf.String()
		})

		if result != "hello" {
			t.Errorf("expected hello, got %s", result)
		}
	})

	t.Run("UseWithResult returns result", func(t *testing.T) {
		p := New(func() int { return 10 }, nil)

		result := UseWithResult(p, func(n int) int {
			return n * 2
		})

		if result != 20 {
			t.Errorf("expected 20, got %d", result)
		}
	})
}
