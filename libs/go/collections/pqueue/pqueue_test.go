package pqueue

import "testing"

func TestPriorityQueue(t *testing.T) {
	t.Run("MinHeap returns smallest first", func(t *testing.T) {
		pq := MinHeap[int]()
		pq.Push(3)
		pq.Push(1)
		pq.Push(2)

		v := pq.Pop()
		if v.IsNone() || v.Unwrap() != 1 {
			t.Error("expected 1")
		}

		v = pq.Pop()
		if v.IsNone() || v.Unwrap() != 2 {
			t.Error("expected 2")
		}

		v = pq.Pop()
		if v.IsNone() || v.Unwrap() != 3 {
			t.Error("expected 3")
		}
	})

	t.Run("MaxHeap returns largest first", func(t *testing.T) {
		pq := MaxHeap[int]()
		pq.Push(1)
		pq.Push(3)
		pq.Push(2)

		v := pq.Pop()
		if v.IsNone() || v.Unwrap() != 3 {
			t.Error("expected 3")
		}

		v = pq.Pop()
		if v.IsNone() || v.Unwrap() != 2 {
			t.Error("expected 2")
		}
	})

	t.Run("Pop empty returns None", func(t *testing.T) {
		pq := MinHeap[int]()
		if pq.Pop().IsSome() {
			t.Error("expected None")
		}
	})

	t.Run("Peek returns without removing", func(t *testing.T) {
		pq := MinHeap[int]()
		pq.Push(1)
		pq.Push(2)

		v := pq.Peek()
		if v.IsNone() || v.Unwrap() != 1 {
			t.Error("expected 1")
		}
		if pq.Len() != 2 {
			t.Error("peek should not remove")
		}
	})

	t.Run("Len and IsEmpty", func(t *testing.T) {
		pq := MinHeap[int]()
		if !pq.IsEmpty() {
			t.Error("expected empty")
		}
		pq.Push(1)
		if pq.IsEmpty() {
			t.Error("expected not empty")
		}
		if pq.Len() != 1 {
			t.Errorf("expected 1, got %d", pq.Len())
		}
	})

	t.Run("Clear", func(t *testing.T) {
		pq := MinHeap[int]()
		pq.Push(1)
		pq.Push(2)
		pq.Clear()
		if !pq.IsEmpty() {
			t.Error("expected empty after clear")
		}
	})
}

func TestPriorityQueueWithPriority(t *testing.T) {
	t.Run("Returns highest priority first", func(t *testing.T) {
		pq := NewWithPriority[string]()
		pq.Push(PriorityItem[string]{Value: "low", Priority: 1})
		pq.Push(PriorityItem[string]{Value: "high", Priority: 10})
		pq.Push(PriorityItem[string]{Value: "medium", Priority: 5})

		v := pq.Pop()
		if v.IsNone() || v.Unwrap().Value != "high" {
			t.Error("expected high")
		}

		v = pq.Pop()
		if v.IsNone() || v.Unwrap().Value != "medium" {
			t.Error("expected medium")
		}

		v = pq.Pop()
		if v.IsNone() || v.Unwrap().Value != "low" {
			t.Error("expected low")
		}
	})
}

func TestCustomPriorityQueue(t *testing.T) {
	type Task struct {
		Name     string
		Priority int
	}

	pq := New(func(a, b Task) bool {
		return a.Priority > b.Priority
	})

	pq.Push(Task{Name: "task1", Priority: 1})
	pq.Push(Task{Name: "task3", Priority: 3})
	pq.Push(Task{Name: "task2", Priority: 2})

	v := pq.Pop()
	if v.IsNone() || v.Unwrap().Name != "task3" {
		t.Error("expected task3")
	}
}
