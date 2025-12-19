package slices

import (
	"reflect"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// **Feature: resilience-lib-extraction, Property 12: Slice Map Preserves Length**
// **Validates: Requirements 27.1**
func TestSliceMapPreservesLength(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("Map preserves slice length", prop.ForAll(
		func(slice []int) bool {
			fn := func(x int) int { return x * 2 }
			mapped := Map(slice, fn)
			return len(mapped) == len(slice)
		},
		gen.SliceOf(gen.Int()),
	))

	properties.Property("Map applies function to each element", prop.ForAll(
		func(slice []int) bool {
			fn := func(x int) int { return x * 2 }
			mapped := Map(slice, fn)
			for i, v := range slice {
				if mapped[i] != fn(v) {
					return false
				}
			}
			return true
		},
		gen.SliceOf(gen.Int()),
	))

	properties.TestingRun(t)
}

// **Feature: resilience-lib-extraction, Property 13: Slice Filter Subset Property**
// **Validates: Requirements 27.2**
func TestSliceFilterSubsetProperty(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("Filter returns subset where all satisfy predicate", prop.ForAll(
		func(slice []int) bool {
			predicate := func(x int) bool { return x > 0 }
			filtered := Filter(slice, predicate)
			for _, v := range filtered {
				if !predicate(v) {
					return false
				}
			}
			return true
		},
		gen.SliceOf(gen.Int()),
	))

	properties.Property("Filter includes all satisfying elements", prop.ForAll(
		func(slice []int) bool {
			predicate := func(x int) bool { return x > 0 }
			filtered := Filter(slice, predicate)
			expectedCount := 0
			for _, v := range slice {
				if predicate(v) {
					expectedCount++
				}
			}
			return len(filtered) == expectedCount
		},
		gen.SliceOf(gen.Int()),
	))

	properties.TestingRun(t)
}


// **Feature: resilience-lib-extraction, Property 14: Slice Chunk-Flatten Identity**
// **Validates: Requirements 27.9, 27.10**
func TestSliceChunkFlattenIdentity(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("Flatten(Chunk(slice, size)) equals original slice", prop.ForAll(
		func(slice []int, size int) bool {
			if size <= 0 {
				size = 1 // Ensure positive chunk size
			}
			chunked := Chunk(slice, size)
			flattened := Flatten(chunked)
			return reflect.DeepEqual(flattened, slice)
		},
		gen.SliceOf(gen.Int()),
		gen.IntRange(1, 10),
	))

	properties.TestingRun(t)
}

func TestReduceBasic(t *testing.T) {
	slice := []int{1, 2, 3, 4, 5}
	sum := Reduce(slice, 0, func(acc, v int) int { return acc + v })
	if sum != 15 {
		t.Errorf("expected 15, got %d", sum)
	}
}

func TestFindBasic(t *testing.T) {
	slice := []int{1, 2, 3, 4, 5}
	
	found := Find(slice, func(x int) bool { return x > 3 })
	if found.IsNone() || found.Unwrap() != 4 {
		t.Error("expected to find 4")
	}

	notFound := Find(slice, func(x int) bool { return x > 10 })
	if notFound.IsSome() {
		t.Error("expected None")
	}
}

func TestAnyAll(t *testing.T) {
	slice := []int{1, 2, 3, 4, 5}

	if !Any(slice, func(x int) bool { return x > 3 }) {
		t.Error("expected Any to return true")
	}

	if Any(slice, func(x int) bool { return x > 10 }) {
		t.Error("expected Any to return false")
	}

	if !All(slice, func(x int) bool { return x > 0 }) {
		t.Error("expected All to return true")
	}

	if All(slice, func(x int) bool { return x > 3 }) {
		t.Error("expected All to return false")
	}
}

func TestGroupBy(t *testing.T) {
	slice := []int{1, 2, 3, 4, 5, 6}
	grouped := GroupBy(slice, func(x int) string {
		if x%2 == 0 {
			return "even"
		}
		return "odd"
	})

	if len(grouped["even"]) != 3 {
		t.Errorf("expected 3 even numbers, got %d", len(grouped["even"]))
	}
	if len(grouped["odd"]) != 3 {
		t.Errorf("expected 3 odd numbers, got %d", len(grouped["odd"]))
	}
}

func TestPartition(t *testing.T) {
	slice := []int{1, 2, 3, 4, 5}
	positive, nonPositive := Partition(slice, func(x int) bool { return x > 2 })

	if len(positive) != 3 {
		t.Errorf("expected 3 positive, got %d", len(positive))
	}
	if len(nonPositive) != 2 {
		t.Errorf("expected 2 non-positive, got %d", len(nonPositive))
	}
}

func TestChunk(t *testing.T) {
	slice := []int{1, 2, 3, 4, 5}
	chunks := Chunk(slice, 2)

	if len(chunks) != 3 {
		t.Errorf("expected 3 chunks, got %d", len(chunks))
	}
	if len(chunks[0]) != 2 || len(chunks[1]) != 2 || len(chunks[2]) != 1 {
		t.Error("unexpected chunk sizes")
	}
}

func TestUnique(t *testing.T) {
	slice := []int{1, 2, 2, 3, 3, 3, 4}
	unique := Unique(slice)

	if len(unique) != 4 {
		t.Errorf("expected 4 unique elements, got %d", len(unique))
	}
}

func TestReverse(t *testing.T) {
	slice := []int{1, 2, 3, 4, 5}
	reversed := Reverse(slice)

	expected := []int{5, 4, 3, 2, 1}
	if !reflect.DeepEqual(reversed, expected) {
		t.Errorf("expected %v, got %v", expected, reversed)
	}
}

func TestTakeDrop(t *testing.T) {
	slice := []int{1, 2, 3, 4, 5}

	taken := Take(slice, 3)
	if !reflect.DeepEqual(taken, []int{1, 2, 3}) {
		t.Errorf("unexpected Take result: %v", taken)
	}

	dropped := Drop(slice, 2)
	if !reflect.DeepEqual(dropped, []int{3, 4, 5}) {
		t.Errorf("unexpected Drop result: %v", dropped)
	}
}
