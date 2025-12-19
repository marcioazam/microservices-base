package maps

import (
	"reflect"
	"sort"
	"testing"
)

func TestKeys(t *testing.T) {
	m := map[string]int{"a": 1, "b": 2, "c": 3}
	keys := Keys(m)
	sort.Strings(keys)
	expected := []string{"a", "b", "c"}
	if !reflect.DeepEqual(keys, expected) {
		t.Errorf("expected %v, got %v", expected, keys)
	}
}

func TestValues(t *testing.T) {
	m := map[string]int{"a": 1, "b": 2, "c": 3}
	values := Values(m)
	sort.Ints(values)
	expected := []int{1, 2, 3}
	if !reflect.DeepEqual(values, expected) {
		t.Errorf("expected %v, got %v", expected, values)
	}
}

func TestMerge(t *testing.T) {
	m1 := map[string]int{"a": 1, "b": 2}
	m2 := map[string]int{"b": 3, "c": 4}
	merged := Merge(m1, m2)

	if merged["a"] != 1 || merged["b"] != 3 || merged["c"] != 4 {
		t.Errorf("unexpected merge result: %v", merged)
	}
}

func TestFilter(t *testing.T) {
	m := map[string]int{"a": 1, "b": 2, "c": 3}
	filtered := Filter(m, func(k string, v int) bool { return v > 1 })

	if len(filtered) != 2 || filtered["a"] != 0 {
		t.Errorf("unexpected filter result: %v", filtered)
	}
}

func TestMapValues(t *testing.T) {
	m := map[string]int{"a": 1, "b": 2}
	mapped := MapValues(m, func(v int) int { return v * 2 })

	if mapped["a"] != 2 || mapped["b"] != 4 {
		t.Errorf("unexpected map result: %v", mapped)
	}
}

func TestInvert(t *testing.T) {
	m := map[string]int{"a": 1, "b": 2}
	inverted := Invert(m)

	if inverted[1] != "a" || inverted[2] != "b" {
		t.Errorf("unexpected invert result: %v", inverted)
	}
}

func TestGet(t *testing.T) {
	m := map[string]int{"a": 1}

	found := Get(m, "a")
	if found.IsNone() || found.Unwrap() != 1 {
		t.Error("expected to find value")
	}

	notFound := Get(m, "b")
	if notFound.IsSome() {
		t.Error("expected None")
	}
}

func TestGetOrDefault(t *testing.T) {
	m := map[string]int{"a": 1}

	if GetOrDefault(m, "a", 0) != 1 {
		t.Error("expected actual value")
	}
	if GetOrDefault(m, "b", 99) != 99 {
		t.Error("expected default value")
	}
}

func TestClone(t *testing.T) {
	m := map[string]int{"a": 1, "b": 2}
	cloned := Clone(m)

	if !reflect.DeepEqual(m, cloned) {
		t.Error("clone should be equal")
	}

	cloned["c"] = 3
	if _, ok := m["c"]; ok {
		t.Error("original should not be modified")
	}
}

func TestPickOmit(t *testing.T) {
	m := map[string]int{"a": 1, "b": 2, "c": 3}

	picked := Pick(m, []string{"a", "c"})
	if len(picked) != 2 || picked["b"] != 0 {
		t.Errorf("unexpected pick result: %v", picked)
	}

	omitted := Omit(m, []string{"b"})
	if len(omitted) != 2 || omitted["b"] != 0 {
		t.Errorf("unexpected omit result: %v", omitted)
	}
}

func TestMergeWith(t *testing.T) {
	m1 := map[string]int{"a": 1, "b": 2}
	m2 := map[string]int{"b": 3, "c": 4}
	merged := MergeWith(m1, m2, func(v1, v2 int) int { return v1 + v2 })

	if merged["a"] != 1 || merged["b"] != 5 || merged["c"] != 4 {
		t.Errorf("unexpected merge result: %v", merged)
	}
}
