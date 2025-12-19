// Package sort provides tests for the sort library.
package sort

import (
	"reflect"
	"testing"
)

func TestSort(t *testing.T) {
	slice := []int{3, 1, 4, 1, 5, 9, 2, 6}
	Sort(slice, func(a, b int) bool { return a < b })
	expected := []int{1, 1, 2, 3, 4, 5, 6, 9}

	if !reflect.DeepEqual(slice, expected) {
		t.Errorf("expected %v, got %v", expected, slice)
	}
}

func TestSort_Empty(t *testing.T) {
	slice := []int{}
	Sort(slice, func(a, b int) bool { return a < b })

	if len(slice) != 0 {
		t.Errorf("expected empty slice")
	}
}

func TestSort_SingleElement(t *testing.T) {
	slice := []int{42}
	Sort(slice, func(a, b int) bool { return a < b })

	if slice[0] != 42 {
		t.Errorf("expected [42], got %v", slice)
	}
}

func TestSortStable(t *testing.T) {
	type item struct {
		key   int
		order int
	}

	slice := []item{
		{1, 1}, {2, 2}, {1, 3}, {2, 4},
	}

	SortStable(slice, func(a, b item) bool { return a.key < b.key })

	// Items with same key should maintain original order
	if slice[0].order != 1 || slice[1].order != 3 {
		t.Errorf("stable sort failed: %v", slice)
	}
}

func TestSorted(t *testing.T) {
	original := []int{3, 1, 2}
	sorted := Sorted(original, func(a, b int) bool { return a < b })

	// Original should be unchanged
	if original[0] != 3 {
		t.Error("original was modified")
	}

	expected := []int{1, 2, 3}
	if !reflect.DeepEqual(sorted, expected) {
		t.Errorf("expected %v, got %v", expected, sorted)
	}
}

func TestSortBy(t *testing.T) {
	type person struct {
		Name string
		Age  int
	}

	people := []person{
		{"Alice", 30},
		{"Bob", 25},
		{"Charlie", 35},
	}

	SortBy(people, func(p person) int { return p.Age })

	if people[0].Name != "Bob" || people[2].Name != "Charlie" {
		t.Errorf("sort by age failed: %v", people)
	}
}

func TestSortByDesc(t *testing.T) {
	slice := []int{1, 3, 2}
	SortByDesc(slice, func(x int) int { return x })
	expected := []int{3, 2, 1}

	if !reflect.DeepEqual(slice, expected) {
		t.Errorf("expected %v, got %v", expected, slice)
	}
}

func TestSortByMultiple(t *testing.T) {
	type person struct {
		Name string
		Age  int
	}

	people := []person{
		{"Alice", 30},
		{"Bob", 30},
		{"Charlie", 25},
	}

	SortByMultiple(people,
		CompareBy(func(p person) int { return p.Age }),
		CompareBy(func(p person) string { return p.Name }),
	)

	// Should be sorted by age, then by name
	if people[0].Name != "Charlie" {
		t.Errorf("expected Charlie first, got %v", people)
	}
	if people[1].Name != "Alice" || people[2].Name != "Bob" {
		t.Errorf("expected Alice before Bob at age 30, got %v", people)
	}
}

func TestIsSorted(t *testing.T) {
	sorted := []int{1, 2, 3, 4, 5}
	if !IsSorted(sorted, func(a, b int) bool { return a < b }) {
		t.Error("expected sorted")
	}

	unsorted := []int{1, 3, 2}
	if IsSorted(unsorted, func(a, b int) bool { return a < b }) {
		t.Error("expected unsorted")
	}
}

func TestIsSorted_Empty(t *testing.T) {
	empty := []int{}
	if !IsSorted(empty, func(a, b int) bool { return a < b }) {
		t.Error("empty slice should be sorted")
	}
}

func TestIsSortedBy(t *testing.T) {
	slice := []int{1, 2, 3}
	if !IsSortedBy(slice, func(x int) int { return x }) {
		t.Error("expected sorted")
	}
}

func TestReverse(t *testing.T) {
	slice := []int{1, 2, 3, 4, 5}
	Reverse(slice)
	expected := []int{5, 4, 3, 2, 1}

	if !reflect.DeepEqual(slice, expected) {
		t.Errorf("expected %v, got %v", expected, slice)
	}
}

func TestReverse_Empty(t *testing.T) {
	slice := []int{}
	Reverse(slice)
	if len(slice) != 0 {
		t.Error("expected empty slice")
	}
}

func TestReverse_SingleElement(t *testing.T) {
	slice := []int{42}
	Reverse(slice)
	if slice[0] != 42 {
		t.Errorf("expected [42], got %v", slice)
	}
}

func TestReversed(t *testing.T) {
	original := []int{1, 2, 3}
	reversed := Reversed(original)

	// Original should be unchanged
	if original[0] != 1 {
		t.Error("original was modified")
	}

	expected := []int{3, 2, 1}
	if !reflect.DeepEqual(reversed, expected) {
		t.Errorf("expected %v, got %v", expected, reversed)
	}
}

func TestMin(t *testing.T) {
	slice := []int{3, 1, 4, 1, 5}
	min, ok := Min(slice)

	if !ok || min != 1 {
		t.Errorf("expected 1, got %d", min)
	}
}

func TestMin_Empty(t *testing.T) {
	slice := []int{}
	_, ok := Min(slice)

	if ok {
		t.Error("expected false for empty slice")
	}
}

func TestMax(t *testing.T) {
	slice := []int{3, 1, 4, 1, 5}
	max, ok := Max(slice)

	if !ok || max != 5 {
		t.Errorf("expected 5, got %d", max)
	}
}

func TestMax_Empty(t *testing.T) {
	slice := []int{}
	_, ok := Max(slice)

	if ok {
		t.Error("expected false for empty slice")
	}
}

func TestMinBy(t *testing.T) {
	type item struct {
		Value int
	}

	slice := []item{{3}, {1}, {4}}
	min, ok := MinBy(slice, func(i item) int { return i.Value })

	if !ok || min.Value != 1 {
		t.Errorf("expected Value=1, got %v", min)
	}
}

func TestMaxBy(t *testing.T) {
	type item struct {
		Value int
	}

	slice := []item{{3}, {1}, {4}}
	max, ok := MaxBy(slice, func(i item) int { return i.Value })

	if !ok || max.Value != 4 {
		t.Errorf("expected Value=4, got %v", max)
	}
}

func TestTopN(t *testing.T) {
	slice := []int{3, 1, 4, 1, 5, 9, 2, 6}
	top3 := TopN(slice, 3, func(a, b int) bool { return a < b })

	if len(top3) != 3 {
		t.Errorf("expected 3 elements, got %d", len(top3))
	}

	// Should contain 9, 6, 5 (largest)
	expected := []int{9, 6, 5}
	if !reflect.DeepEqual(top3, expected) {
		t.Errorf("expected %v, got %v", expected, top3)
	}
}

func TestTopN_Zero(t *testing.T) {
	slice := []int{1, 2, 3}
	result := TopN(slice, 0, func(a, b int) bool { return a < b })

	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestBottomN(t *testing.T) {
	slice := []int{3, 1, 4, 1, 5, 9, 2, 6}
	bottom3 := BottomN(slice, 3, func(a, b int) bool { return a < b })

	if len(bottom3) != 3 {
		t.Errorf("expected 3 elements, got %d", len(bottom3))
	}

	// Should contain 1, 1, 2 (smallest)
	expected := []int{1, 1, 2}
	if !reflect.DeepEqual(bottom3, expected) {
		t.Errorf("expected %v, got %v", expected, bottom3)
	}
}

func TestCompare(t *testing.T) {
	cmp := Compare[int]()

	if cmp(1, 2) != -1 {
		t.Error("expected -1 for 1 < 2")
	}
	if cmp(2, 1) != 1 {
		t.Error("expected 1 for 2 > 1")
	}
	if cmp(1, 1) != 0 {
		t.Error("expected 0 for 1 == 1")
	}
}

func TestCompareDesc(t *testing.T) {
	cmp := CompareDesc[int]()

	if cmp(1, 2) != 1 {
		t.Error("expected 1 for 1 < 2 (desc)")
	}
	if cmp(2, 1) != -1 {
		t.Error("expected -1 for 2 > 1 (desc)")
	}
}

func TestShuffle(t *testing.T) {
	slice := []int{1, 2, 3, 4, 5}
	original := make([]int, len(slice))
	copy(original, slice)

	// Use deterministic "random" for testing
	i := 0
	randFn := func(n int) int {
		i++
		return i % n
	}

	Shuffle(slice, randFn)

	// Just verify it doesn't panic and has same elements
	Sort(slice, func(a, b int) bool { return a < b })
	if !reflect.DeepEqual(slice, original) {
		t.Error("shuffle changed elements")
	}
}

func TestBinarySearch(t *testing.T) {
	slice := []int{1, 2, 3, 4, 5}

	idx, found := BinarySearch(slice, 3)
	if !found || idx != 2 {
		t.Errorf("expected index 2, found=true; got %d, %v", idx, found)
	}

	idx, found = BinarySearch(slice, 6)
	if found {
		t.Error("expected not found for 6")
	}
	if idx != 5 {
		t.Errorf("expected insertion point 5, got %d", idx)
	}
}

func TestBinarySearch_Empty(t *testing.T) {
	slice := []int{}
	idx, found := BinarySearch(slice, 1)

	if found {
		t.Error("expected not found")
	}
	if idx != 0 {
		t.Errorf("expected insertion point 0, got %d", idx)
	}
}

func TestBinarySearchBy(t *testing.T) {
	type item struct {
		ID   int
		Name string
	}

	slice := []item{{1, "a"}, {2, "b"}, {3, "c"}}

	idx, found := BinarySearchBy(slice, 2, func(i item) int { return i.ID })
	if !found || idx != 1 {
		t.Errorf("expected index 1, found=true; got %d, %v", idx, found)
	}
}
