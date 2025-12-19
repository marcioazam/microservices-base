package channels

import (
	"testing"
	"time"
)

func TestMap(t *testing.T) {
	in := FromSlice([]int{1, 2, 3})
	out := Map(in, func(x int) int { return x * 2 })
	result := ToSlice(out)
	if len(result) != 3 || result[0] != 2 || result[1] != 4 || result[2] != 6 {
		t.Error("unexpected values")
	}
}

func TestFilter(t *testing.T) {
	in := FromSlice([]int{1, 2, 3, 4, 5})
	out := Filter(in, func(x int) bool { return x%2 == 0 })
	result := ToSlice(out)
	if len(result) != 2 || result[0] != 2 || result[1] != 4 {
		t.Error("unexpected values")
	}
}

func TestMerge(t *testing.T) {
	ch1 := FromSlice([]int{1, 2})
	ch2 := FromSlice([]int{3, 4})
	merged := Merge(ch1, ch2)
	result := ToSlice(merged)
	if len(result) != 4 {
		t.Errorf("expected 4 values, got %d", len(result))
	}
}

func TestFanOut(t *testing.T) {
	in := FromSlice([]int{1, 2, 3, 4})
	outs := FanOut(in, 2)
	if len(outs) != 2 {
		t.Errorf("expected 2 channels, got %d", len(outs))
	}
	result1 := ToSlice(outs[0])
	result2 := ToSlice(outs[1])
	if len(result1)+len(result2) != 4 {
		t.Error("not all values distributed")
	}
}

func TestBuffer(t *testing.T) {
	in := FromSlice([]int{1, 2, 3})
	buffered := Buffer(in, 10)
	result := ToSlice(buffered)
	if len(result) != 3 {
		t.Errorf("expected 3 values, got %d", len(result))
	}
}

func TestTake(t *testing.T) {
	in := FromSlice([]int{1, 2, 3, 4, 5})
	taken := Take(in, 3)
	result := ToSlice(taken)
	if len(result) != 3 || result[0] != 1 || result[2] != 3 {
		t.Error("unexpected values")
	}
}

func TestSkip(t *testing.T) {
	in := FromSlice([]int{1, 2, 3, 4, 5})
	skipped := Skip(in, 2)
	result := ToSlice(skipped)
	if len(result) != 3 || result[0] != 3 {
		t.Error("unexpected values")
	}
}

func TestDistinct(t *testing.T) {
	in := FromSlice([]int{1, 1, 2, 2, 2, 3, 1})
	distinct := Distinct(in)
	result := ToSlice(distinct)
	if len(result) != 4 || result[0] != 1 || result[1] != 2 || result[2] != 3 || result[3] != 1 {
		t.Errorf("unexpected values: %v", result)
	}
}

func TestBatch(t *testing.T) {
	in := make(chan int)
	go func() {
		for i := 1; i <= 5; i++ {
			in <- i
		}
		close(in)
	}()

	batched := Batch(in, 2, time.Second)
	result := ToSlice(batched)
	if len(result) != 3 { // [1,2], [3,4], [5]
		t.Errorf("expected 3 batches, got %d", len(result))
	}
}

func TestGenerate(t *testing.T) {
	ch := Generate(func(yield func(int)) {
		for i := 1; i <= 3; i++ {
			yield(i)
		}
	})
	result := ToSlice(ch)
	if len(result) != 3 || result[0] != 1 || result[2] != 3 {
		t.Error("unexpected values")
	}
}

func TestFromSliceToSlice(t *testing.T) {
	items := []int{1, 2, 3, 4, 5}
	ch := FromSlice(items)
	result := ToSlice(ch)
	if len(result) != len(items) {
		t.Errorf("expected %d items, got %d", len(items), len(result))
	}
	for i, v := range result {
		if v != items[i] {
			t.Errorf("expected %d, got %d", items[i], v)
		}
	}
}

func TestTee(t *testing.T) {
	in := FromSlice([]int{1, 2, 3})
	out1, out2 := Tee(in)

	// Consume both in parallel
	done := make(chan []int, 2)
	go func() { done <- ToSlice(out1) }()
	go func() { done <- ToSlice(out2) }()

	result1 := <-done
	result2 := <-done

	if len(result1) != 3 || len(result2) != 3 {
		t.Error("both outputs should have all values")
	}
}
