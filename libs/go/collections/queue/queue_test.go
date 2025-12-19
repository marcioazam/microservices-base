package queue

import (
	"sync"
	"testing"
)

func TestQueue(t *testing.T) {
	t.Run("Enqueue and Dequeue", func(t *testing.T) {
		q := NewQueue[int]()
		q.Enqueue(1)
		q.Enqueue(2)
		q.Enqueue(3)

		if q.Len() != 3 {
			t.Errorf("expected 3, got %d", q.Len())
		}

		v := q.Dequeue()
		if v.IsNone() || v.Unwrap() != 1 {
			t.Error("expected 1")
		}

		v = q.Dequeue()
		if v.IsNone() || v.Unwrap() != 2 {
			t.Error("expected 2")
		}
	})

	t.Run("Dequeue empty returns None", func(t *testing.T) {
		q := NewQueue[int]()
		if q.Dequeue().IsSome() {
			t.Error("expected None")
		}
	})

	t.Run("Peek returns front without removing", func(t *testing.T) {
		q := NewQueue[int]()
		q.Enqueue(1)
		q.Enqueue(2)

		v := q.Peek()
		if v.IsNone() || v.Unwrap() != 1 {
			t.Error("expected 1")
		}
		if q.Len() != 2 {
			t.Error("peek should not remove")
		}
	})

	t.Run("IsEmpty", func(t *testing.T) {
		q := NewQueue[int]()
		if !q.IsEmpty() {
			t.Error("expected empty")
		}
		q.Enqueue(1)
		if q.IsEmpty() {
			t.Error("expected not empty")
		}
	})

	t.Run("Clear", func(t *testing.T) {
		q := NewQueue[int]()
		q.Enqueue(1)
		q.Enqueue(2)
		q.Clear()
		if !q.IsEmpty() {
			t.Error("expected empty after clear")
		}
	})

	t.Run("ToSlice", func(t *testing.T) {
		q := NewQueue[int]()
		q.Enqueue(1)
		q.Enqueue(2)
		q.Enqueue(3)
		s := q.ToSlice()
		if len(s) != 3 || s[0] != 1 || s[2] != 3 {
			t.Error("unexpected slice")
		}
	})
}

func TestStack(t *testing.T) {
	t.Run("Push and Pop", func(t *testing.T) {
		s := NewStack[int]()
		s.Push(1)
		s.Push(2)
		s.Push(3)

		if s.Len() != 3 {
			t.Errorf("expected 3, got %d", s.Len())
		}

		v := s.Pop()
		if v.IsNone() || v.Unwrap() != 3 {
			t.Error("expected 3")
		}

		v = s.Pop()
		if v.IsNone() || v.Unwrap() != 2 {
			t.Error("expected 2")
		}
	})

	t.Run("Pop empty returns None", func(t *testing.T) {
		s := NewStack[int]()
		if s.Pop().IsSome() {
			t.Error("expected None")
		}
	})

	t.Run("Peek returns top without removing", func(t *testing.T) {
		s := NewStack[int]()
		s.Push(1)
		s.Push(2)

		v := s.Peek()
		if v.IsNone() || v.Unwrap() != 2 {
			t.Error("expected 2")
		}
		if s.Len() != 2 {
			t.Error("peek should not remove")
		}
	})

	t.Run("IsEmpty", func(t *testing.T) {
		s := NewStack[int]()
		if !s.IsEmpty() {
			t.Error("expected empty")
		}
		s.Push(1)
		if s.IsEmpty() {
			t.Error("expected not empty")
		}
	})
}

func TestConcurrentQueue(t *testing.T) {
	t.Run("Concurrent operations", func(t *testing.T) {
		q := NewConcurrentQueue[int]()
		var wg sync.WaitGroup

		// Enqueue concurrently
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(v int) {
				defer wg.Done()
				q.Enqueue(v)
			}(i)
		}
		wg.Wait()

		if q.Len() != 100 {
			t.Errorf("expected 100, got %d", q.Len())
		}

		// Dequeue concurrently
		count := 0
		var mu sync.Mutex
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if q.Dequeue().IsSome() {
					mu.Lock()
					count++
					mu.Unlock()
				}
			}()
		}
		wg.Wait()

		if count != 100 {
			t.Errorf("expected 100 dequeues, got %d", count)
		}
	})
}

func TestConcurrentStack(t *testing.T) {
	t.Run("Concurrent operations", func(t *testing.T) {
		s := NewConcurrentStack[int]()
		var wg sync.WaitGroup

		// Push concurrently
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(v int) {
				defer wg.Done()
				s.Push(v)
			}(i)
		}
		wg.Wait()

		if s.Len() != 100 {
			t.Errorf("expected 100, got %d", s.Len())
		}

		// Pop concurrently
		count := 0
		var mu sync.Mutex
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if s.Pop().IsSome() {
					mu.Lock()
					count++
					mu.Unlock()
				}
			}()
		}
		wg.Wait()

		if count != 100 {
			t.Errorf("expected 100 pops, got %d", count)
		}
	})
}
