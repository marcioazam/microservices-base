// Package merge provides tests for the merge library.
package merge

import (
	"reflect"
	"testing"
)

func TestSlices(t *testing.T) {
	a := []int{1, 2}
	b := []int{3, 4}
	c := []int{5}

	result := Slices(a, b, c)
	expected := []int{1, 2, 3, 4, 5}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestSlices_Empty(t *testing.T) {
	result := Slices[int]()
	if len(result) != 0 {
		t.Errorf("expected empty slice, got %v", result)
	}
}

func TestSlices_SingleSlice(t *testing.T) {
	a := []int{1, 2, 3}
	result := Slices(a)

	if !reflect.DeepEqual(result, a) {
		t.Errorf("expected %v, got %v", a, result)
	}
}

func TestSlicesUnique(t *testing.T) {
	a := []int{1, 2, 3}
	b := []int{2, 3, 4}
	c := []int{4, 5}

	result := SlicesUnique(a, b, c)
	expected := []int{1, 2, 3, 4, 5}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestSlicesUnique_AllDuplicates(t *testing.T) {
	a := []int{1, 1, 1}
	b := []int{1, 1}

	result := SlicesUnique(a, b)
	expected := []int{1}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestSlicesUniqueBy(t *testing.T) {
	type item struct {
		ID   int
		Name string
	}

	a := []item{{1, "a"}, {2, "b"}}
	b := []item{{2, "c"}, {3, "d"}}

	result := SlicesUniqueBy([][]item{a, b}, func(i item) int { return i.ID })

	if len(result) != 3 {
		t.Errorf("expected 3 items, got %d", len(result))
	}
}

func TestMaps(t *testing.T) {
	a := map[string]int{"a": 1, "b": 2}
	b := map[string]int{"b": 3, "c": 4}

	result := Maps(a, b)

	if result["a"] != 1 {
		t.Errorf("expected a=1, got %d", result["a"])
	}
	if result["b"] != 3 { // b should be overwritten
		t.Errorf("expected b=3, got %d", result["b"])
	}
	if result["c"] != 4 {
		t.Errorf("expected c=4, got %d", result["c"])
	}
}

func TestMaps_Empty(t *testing.T) {
	result := Maps[string, int]()
	if len(result) != 0 {
		t.Errorf("expected empty map, got %v", result)
	}
}

func TestMapsWithStrategy_Overwrite(t *testing.T) {
	a := map[string]int{"a": 1}
	b := map[string]int{"a": 2}

	result := MapsWithStrategy(StrategyOverwrite, a, b)

	if result["a"] != 2 {
		t.Errorf("expected a=2, got %d", result["a"])
	}
}

func TestMapsWithStrategy_Keep(t *testing.T) {
	a := map[string]int{"a": 1}
	b := map[string]int{"a": 2}

	result := MapsWithStrategy(StrategyKeep, a, b)

	if result["a"] != 1 {
		t.Errorf("expected a=1, got %d", result["a"])
	}
}

func TestMapsWithResolver(t *testing.T) {
	a := map[string]int{"a": 1}
	b := map[string]int{"a": 2}

	// Sum resolver
	resolver := func(key string, old, new int) int {
		return old + new
	}

	result := MapsWithResolver(resolver, a, b)

	if result["a"] != 3 {
		t.Errorf("expected a=3, got %d", result["a"])
	}
}

func TestDeepMerge_Maps(t *testing.T) {
	a := map[string]interface{}{
		"name": "Alice",
		"x":    1,
	}
	b := map[string]interface{}{
		"age": 30,
		"y":   2,
	}

	result := DeepMerge(a, b).(map[string]interface{})

	if result["name"] != "Alice" {
		t.Errorf("expected name=Alice, got %v", result["name"])
	}
	if result["age"] != 30 {
		t.Errorf("expected age=30, got %v", result["age"])
	}
	if result["x"] != 1 {
		t.Errorf("expected x=1, got %v", result["x"])
	}
	if result["y"] != 2 {
		t.Errorf("expected y=2, got %v", result["y"])
	}
}

func TestDeepMerge_Slices(t *testing.T) {
	a := []int{1, 2}
	b := []int{3, 4}

	result := DeepMerge(a, b).([]int)
	expected := []int{1, 2, 3, 4}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestMergeSliceAt(t *testing.T) {
	slice := []int{1, 2, 5}
	result := MergeSliceAt(slice, 2, 3, 4)
	expected := []int{1, 2, 3, 4, 5}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestMergeSliceAt_Beginning(t *testing.T) {
	slice := []int{3, 4, 5}
	result := MergeSliceAt(slice, 0, 1, 2)
	expected := []int{1, 2, 3, 4, 5}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestMergeSliceAt_End(t *testing.T) {
	slice := []int{1, 2, 3}
	result := MergeSliceAt(slice, 3, 4, 5)
	expected := []int{1, 2, 3, 4, 5}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestMergeSliceAt_NegativeIndex(t *testing.T) {
	slice := []int{3, 4, 5}
	result := MergeSliceAt(slice, -1, 1, 2)
	expected := []int{1, 2, 3, 4, 5}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestMergeSliceOrdered(t *testing.T) {
	a := []int{1, 4, 7}
	b := []int{2, 5, 8}
	c := []int{3, 6, 9}

	less := func(a, b int) bool { return a < b }
	result := MergeSliceOrdered(less, a, b, c)
	expected := []int{1, 2, 3, 4, 5, 6, 7, 8, 9}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestMergeMapSliceValues(t *testing.T) {
	a := map[string][]int{"x": {1, 2}}
	b := map[string][]int{"x": {3}, "y": {4}}

	result := MergeMapSliceValues(a, b)

	if !reflect.DeepEqual(result["x"], []int{1, 2, 3}) {
		t.Errorf("expected x=[1,2,3], got %v", result["x"])
	}
	if !reflect.DeepEqual(result["y"], []int{4}) {
		t.Errorf("expected y=[4], got %v", result["y"])
	}
}

func TestInterleave(t *testing.T) {
	a := []int{1, 3, 5}
	b := []int{2, 4, 6}

	result := Interleave(a, b)
	expected := []int{1, 2, 3, 4, 5, 6}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestInterleave_UnequalLengths(t *testing.T) {
	a := []int{1, 3}
	b := []int{2, 4, 5, 6}

	result := Interleave(a, b)
	expected := []int{1, 2, 3, 4, 5, 6}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestInterleave_Empty(t *testing.T) {
	result := Interleave[int]()
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestZip(t *testing.T) {
	a := []int{1, 2, 3}
	b := []string{"a", "b", "c"}

	result := Zip(a, b)

	if len(result) != 3 {
		t.Errorf("expected 3 pairs, got %d", len(result))
	}
	if result[0][0] != 1 || result[0][1] != "a" {
		t.Errorf("expected [1, a], got %v", result[0])
	}
}

func TestZip_UnequalLengths(t *testing.T) {
	a := []int{1, 2}
	b := []string{"a", "b", "c"}

	result := Zip(a, b)

	if len(result) != 2 {
		t.Errorf("expected 2 pairs, got %d", len(result))
	}
}

func TestZipWith(t *testing.T) {
	a := []int{1, 2, 3}
	b := []int{10, 20, 30}

	result := ZipWith(a, b, func(x, y int) int { return x + y })
	expected := []int{11, 22, 33}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestZipWith_Strings(t *testing.T) {
	a := []string{"hello", "world"}
	b := []string{" ", "!"}

	result := ZipWith(a, b, func(x, y string) string { return x + y })
	expected := []string{"hello ", "world!"}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}
