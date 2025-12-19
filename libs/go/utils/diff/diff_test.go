// Package diff provides tests for the diff library.
package diff

import (
	"testing"
)

func TestDiff_EmptySlices(t *testing.T) {
	old := []int{}
	new := []int{}
	changes := Diff(old, new)

	if len(changes) != 0 {
		t.Errorf("expected 0 changes, got %d", len(changes))
	}
}

func TestDiff_EmptyToNonEmpty(t *testing.T) {
	old := []int{}
	new := []int{1, 2, 3}
	changes := Diff(old, new)

	insertCount := 0
	for _, c := range changes {
		if c.Op == OpInsert {
			insertCount++
		}
	}

	if insertCount != 3 {
		t.Errorf("expected 3 inserts, got %d", insertCount)
	}
}

func TestDiff_NonEmptyToEmpty(t *testing.T) {
	old := []int{1, 2, 3}
	new := []int{}
	changes := Diff(old, new)

	deleteCount := 0
	for _, c := range changes {
		if c.Op == OpDelete {
			deleteCount++
		}
	}

	if deleteCount != 3 {
		t.Errorf("expected 3 deletes, got %d", deleteCount)
	}
}

func TestDiff_EqualSlices(t *testing.T) {
	old := []int{1, 2, 3}
	new := []int{1, 2, 3}
	changes := Diff(old, new)

	for _, c := range changes {
		if c.Op != OpEqual {
			t.Errorf("expected all OpEqual, got %v", c.Op)
		}
	}
}

func TestDiff_SingleInsertion(t *testing.T) {
	old := []int{1, 3}
	new := []int{1, 2, 3}
	changes := Diff(old, new)

	hasInsert := false
	for _, c := range changes {
		if c.Op == OpInsert && c.NewValue == 2 {
			hasInsert = true
		}
	}

	if !hasInsert {
		t.Error("expected insert of value 2")
	}
}

func TestDiff_SingleDeletion(t *testing.T) {
	old := []int{1, 2, 3}
	new := []int{1, 3}
	changes := Diff(old, new)

	hasDelete := false
	for _, c := range changes {
		if c.Op == OpDelete && c.OldValue == 2 {
			hasDelete = true
		}
	}

	if !hasDelete {
		t.Error("expected delete of value 2")
	}
}

func TestDiff_Strings(t *testing.T) {
	old := []string{"a", "b", "c"}
	new := []string{"a", "x", "c"}
	changes := Diff(old, new)

	if !HasChanges(changes) {
		t.Error("expected changes")
	}
}

func TestPatch_EmptyChanges(t *testing.T) {
	old := []int{1, 2, 3}
	changes := []Change[int]{}
	result := Patch(old, changes)

	if len(result) != len(old) {
		t.Errorf("expected length %d, got %d", len(old), len(result))
	}
}

func TestPatch_Insertions(t *testing.T) {
	old := []int{1, 3}
	changes := []Change[int]{
		{Op: OpEqual, Index: 0, OldValue: 1, NewValue: 1},
		{Op: OpInsert, Index: 1, NewValue: 2},
		{Op: OpEqual, Index: 2, OldValue: 3, NewValue: 3},
	}
	result := Patch(old, changes)

	expected := []int{1, 2, 3}
	if len(result) != len(expected) {
		t.Errorf("expected length %d, got %d", len(expected), len(result))
	}
	for i, v := range expected {
		if result[i] != v {
			t.Errorf("at index %d: expected %d, got %d", i, v, result[i])
		}
	}
}

func TestPatch_Deletions(t *testing.T) {
	old := []int{1, 2, 3}
	changes := []Change[int]{
		{Op: OpEqual, Index: 0, OldValue: 1, NewValue: 1},
		{Op: OpDelete, Index: 1, OldValue: 2},
		{Op: OpEqual, Index: 2, OldValue: 3, NewValue: 3},
	}
	result := Patch(old, changes)

	expected := []int{1, 3}
	if len(result) != len(expected) {
		t.Errorf("expected length %d, got %d", len(expected), len(result))
	}
}

// **Feature: resilience-lib-extraction, Property 22: Diff-Patch Round-Trip**
// **Validates: Requirements 99.1, 99.2**
func TestDiffPatch_RoundTrip(t *testing.T) {
	testCases := []struct {
		name string
		old  []int
		new  []int
	}{
		{"empty to empty", []int{}, []int{}},
		{"empty to non-empty", []int{}, []int{1, 2, 3}},
		{"non-empty to empty", []int{1, 2, 3}, []int{}},
		{"equal slices", []int{1, 2, 3}, []int{1, 2, 3}},
		{"single insert", []int{1, 3}, []int{1, 2, 3}},
		{"single delete", []int{1, 2, 3}, []int{1, 3}},
		{"multiple changes", []int{1, 2, 3, 4}, []int{1, 5, 3, 6}},
		{"complete replacement", []int{1, 2, 3}, []int{4, 5, 6}},
		{"reorder", []int{1, 2, 3}, []int{3, 2, 1}},
		{"duplicates", []int{1, 1, 2, 2}, []int{1, 2, 2, 2}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			changes := Diff(tc.old, tc.new)
			result := Patch(tc.old, changes)

			if len(result) != len(tc.new) {
				t.Errorf("length mismatch: expected %d, got %d", len(tc.new), len(result))
				return
			}

			for i := range tc.new {
				if result[i] != tc.new[i] {
					t.Errorf("at index %d: expected %d, got %d", i, tc.new[i], result[i])
				}
			}
		})
	}
}

func TestDiffPatch_RoundTrip_Strings(t *testing.T) {
	old := []string{"hello", "world", "foo"}
	new := []string{"hello", "bar", "world", "baz"}

	changes := Diff(old, new)
	result := Patch(old, changes)

	if len(result) != len(new) {
		t.Errorf("length mismatch: expected %d, got %d", len(new), len(result))
	}

	for i := range new {
		if result[i] != new[i] {
			t.Errorf("at index %d: expected %s, got %s", i, new[i], result[i])
		}
	}
}

func TestDiffWithEqual(t *testing.T) {
	type item struct {
		ID   int
		Name string
	}

	old := []item{{1, "a"}, {2, "b"}}
	new := []item{{1, "a"}, {3, "c"}}

	equal := func(a, b item) bool {
		return a.ID == b.ID
	}

	changes := DiffWithEqual(old, new, equal)

	if !HasChanges(changes) {
		t.Error("expected changes")
	}
}

func TestDiffObjects(t *testing.T) {
	type Person struct {
		Name string
		Age  int
	}

	old := Person{Name: "Alice", Age: 30}
	new := Person{Name: "Alice", Age: 31}

	changes := DiffObjects(old, new)

	if len(changes) != 1 {
		t.Errorf("expected 1 change, got %d", len(changes))
	}

	if changes[0].Path != "Age" {
		t.Errorf("expected path 'Age', got '%s'", changes[0].Path)
	}
}

func TestDiffObjects_NoChanges(t *testing.T) {
	type Person struct {
		Name string
		Age  int
	}

	old := Person{Name: "Alice", Age: 30}
	new := Person{Name: "Alice", Age: 30}

	changes := DiffObjects(old, new)

	if len(changes) != 0 {
		t.Errorf("expected 0 changes, got %d", len(changes))
	}
}

func TestDiffObjects_Pointers(t *testing.T) {
	type Person struct {
		Name string
		Age  int
	}

	old := &Person{Name: "Alice", Age: 30}
	new := &Person{Name: "Bob", Age: 30}

	changes := DiffObjects(old, new)

	if len(changes) != 1 {
		t.Errorf("expected 1 change, got %d", len(changes))
	}
}

func TestDiffMaps(t *testing.T) {
	old := map[string]int{"a": 1, "b": 2}
	new := map[string]int{"a": 1, "c": 3}

	equal := func(a, b int) bool { return a == b }
	changes := DiffMaps(old, new, equal)

	// Should have: delete "b", insert "c"
	deleteCount := 0
	insertCount := 0
	for _, c := range changes {
		if c.Op == OpDelete {
			deleteCount++
		}
		if c.Op == OpInsert {
			insertCount++
		}
	}

	if deleteCount != 1 {
		t.Errorf("expected 1 delete, got %d", deleteCount)
	}
	if insertCount != 1 {
		t.Errorf("expected 1 insert, got %d", insertCount)
	}
}

func TestHasChanges(t *testing.T) {
	noChanges := []Change[int]{
		{Op: OpEqual, Index: 0},
		{Op: OpEqual, Index: 1},
	}

	if HasChanges(noChanges) {
		t.Error("expected no changes")
	}

	withChanges := []Change[int]{
		{Op: OpEqual, Index: 0},
		{Op: OpInsert, Index: 1},
	}

	if !HasChanges(withChanges) {
		t.Error("expected changes")
	}
}

func TestFilterChanges(t *testing.T) {
	changes := []Change[int]{
		{Op: OpEqual, Index: 0},
		{Op: OpInsert, Index: 1},
		{Op: OpDelete, Index: 2},
		{Op: OpInsert, Index: 3},
	}

	inserts := FilterChanges(changes, OpInsert)
	if len(inserts) != 2 {
		t.Errorf("expected 2 inserts, got %d", len(inserts))
	}

	deletes := FilterChanges(changes, OpDelete)
	if len(deletes) != 1 {
		t.Errorf("expected 1 delete, got %d", len(deletes))
	}
}

func TestCountChanges(t *testing.T) {
	changes := []Change[int]{
		{Op: OpEqual, Index: 0},
		{Op: OpInsert, Index: 1},
		{Op: OpDelete, Index: 2},
		{Op: OpInsert, Index: 3},
	}

	counts := CountChanges(changes)

	if counts[OpEqual] != 1 {
		t.Errorf("expected 1 equal, got %d", counts[OpEqual])
	}
	if counts[OpInsert] != 2 {
		t.Errorf("expected 2 inserts, got %d", counts[OpInsert])
	}
	if counts[OpDelete] != 1 {
		t.Errorf("expected 1 delete, got %d", counts[OpDelete])
	}
}

func TestOperation_String(t *testing.T) {
	tests := []struct {
		op       Operation
		expected string
	}{
		{OpEqual, "equal"},
		{OpInsert, "insert"},
		{OpDelete, "delete"},
		{OpReplace, "replace"},
		{Operation(99), "unknown"},
	}

	for _, tc := range tests {
		if tc.op.String() != tc.expected {
			t.Errorf("expected %s, got %s", tc.expected, tc.op.String())
		}
	}
}
