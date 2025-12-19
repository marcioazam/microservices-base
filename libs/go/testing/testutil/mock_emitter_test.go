// Package testutil provides tests for mock emitter.
package testutil

import (
	"sync"
	"testing"
)

func TestMockEmitter_Emit(t *testing.T) {
	emitter := NewMockEmitter[string]()
	emitter.Emit("event1")
	emitter.Emit("event2")

	if emitter.Len() != 2 {
		t.Errorf("expected 2 events, got %d", emitter.Len())
	}
}

func TestMockEmitter_Events(t *testing.T) {
	emitter := NewMockEmitter[int]()
	emitter.Emit(1)
	emitter.Emit(2)
	emitter.Emit(3)

	events := emitter.Events()
	if len(events) != 3 {
		t.Errorf("expected 3 events, got %d", len(events))
	}

	// Verify events are copies
	events[0] = 999
	if emitter.Events()[0] == 999 {
		t.Error("Events() should return a copy")
	}
}

func TestMockEmitter_Filter(t *testing.T) {
	emitter := NewMockEmitter[int]()
	emitter.Emit(1)
	emitter.Emit(2)
	emitter.Emit(3)
	emitter.Emit(4)

	evens := emitter.Filter(func(n int) bool { return n%2 == 0 })
	if len(evens) != 2 {
		t.Errorf("expected 2 even events, got %d", len(evens))
	}
}

func TestMockEmitter_Clear(t *testing.T) {
	emitter := NewMockEmitter[string]()
	emitter.Emit("event1")
	emitter.Emit("event2")
	emitter.Clear()

	if emitter.Len() != 0 {
		t.Errorf("expected 0 events after clear, got %d", emitter.Len())
	}
}

func TestMockEmitter_Len(t *testing.T) {
	emitter := NewMockEmitter[int]()
	if emitter.Len() != 0 {
		t.Error("expected 0 for empty emitter")
	}

	emitter.Emit(1)
	if emitter.Len() != 1 {
		t.Error("expected 1 after emit")
	}
}

func TestMockEmitter_First(t *testing.T) {
	emitter := NewMockEmitter[string]()

	_, ok := emitter.First()
	if ok {
		t.Error("expected false for empty emitter")
	}

	emitter.Emit("first")
	emitter.Emit("second")

	first, ok := emitter.First()
	if !ok || first != "first" {
		t.Errorf("expected 'first', got '%s'", first)
	}
}

func TestMockEmitter_Last(t *testing.T) {
	emitter := NewMockEmitter[string]()

	_, ok := emitter.Last()
	if ok {
		t.Error("expected false for empty emitter")
	}

	emitter.Emit("first")
	emitter.Emit("last")

	last, ok := emitter.Last()
	if !ok || last != "last" {
		t.Errorf("expected 'last', got '%s'", last)
	}
}

func TestMockEmitter_At(t *testing.T) {
	emitter := NewMockEmitter[int]()
	emitter.Emit(10)
	emitter.Emit(20)
	emitter.Emit(30)

	val, ok := emitter.At(1)
	if !ok || val != 20 {
		t.Errorf("expected 20, got %d", val)
	}

	_, ok = emitter.At(-1)
	if ok {
		t.Error("expected false for negative index")
	}

	_, ok = emitter.At(10)
	if ok {
		t.Error("expected false for out of bounds index")
	}
}

func TestMockEmitter_Contains(t *testing.T) {
	emitter := NewMockEmitter[int]()
	emitter.Emit(1)
	emitter.Emit(2)
	emitter.Emit(3)

	if !emitter.Contains(func(n int) bool { return n == 2 }) {
		t.Error("expected to contain 2")
	}

	if emitter.Contains(func(n int) bool { return n == 5 }) {
		t.Error("expected not to contain 5")
	}
}

func TestMockEmitter_Count(t *testing.T) {
	emitter := NewMockEmitter[int]()
	emitter.Emit(1)
	emitter.Emit(2)
	emitter.Emit(2)
	emitter.Emit(3)

	count := emitter.Count(func(n int) bool { return n == 2 })
	if count != 2 {
		t.Errorf("expected count 2, got %d", count)
	}
}

func TestMockEmitter_ForEach(t *testing.T) {
	emitter := NewMockEmitter[int]()
	emitter.Emit(1)
	emitter.Emit(2)
	emitter.Emit(3)

	sum := 0
	emitter.ForEach(func(n int) { sum += n })

	if sum != 6 {
		t.Errorf("expected sum 6, got %d", sum)
	}
}

func TestMockEmitter_IsEmpty(t *testing.T) {
	emitter := NewMockEmitter[string]()

	if !emitter.IsEmpty() {
		t.Error("expected empty")
	}

	emitter.Emit("event")

	if emitter.IsEmpty() {
		t.Error("expected not empty")
	}
}

func TestMockEmitter_Concurrent(t *testing.T) {
	emitter := NewMockEmitter[int]()
	var wg sync.WaitGroup

	// Concurrent writes
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			emitter.Emit(n)
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = emitter.Len()
			_ = emitter.Events()
		}()
	}

	wg.Wait()

	if emitter.Len() != 100 {
		t.Errorf("expected 100 events, got %d", emitter.Len())
	}
}

func TestMockEmitter_WaitForEvents(t *testing.T) {
	emitter := NewMockEmitter[int]()
	emitter.Emit(1)
	emitter.Emit(2)

	// Should return immediately since we have 2 events
	if !emitter.WaitForEvents(2, nil) {
		t.Error("expected true for 2 events")
	}
}

type testEvent struct {
	Type string
	Data int
}

func TestMockEmitter_StructEvents(t *testing.T) {
	emitter := NewMockEmitter[testEvent]()
	emitter.Emit(testEvent{Type: "create", Data: 1})
	emitter.Emit(testEvent{Type: "update", Data: 2})
	emitter.Emit(testEvent{Type: "create", Data: 3})

	creates := emitter.Filter(func(e testEvent) bool { return e.Type == "create" })
	if len(creates) != 2 {
		t.Errorf("expected 2 create events, got %d", len(creates))
	}
}
