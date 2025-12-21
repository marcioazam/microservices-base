// Package diff provides generic diff and patch utilities for slices and objects.
package diff

import (
	"reflect"
)

// Operation represents a diff operation type.
type Operation int

const (
	// OpEqual indicates elements are equal.
	OpEqual Operation = iota
	// OpInsert indicates an element was inserted.
	OpInsert
	// OpDelete indicates an element was deleted.
	OpDelete
	// OpReplace indicates an element was replaced.
	OpReplace
)

// String returns the string representation of the operation.
func (o Operation) String() string {
	switch o {
	case OpEqual:
		return "equal"
	case OpInsert:
		return "insert"
	case OpDelete:
		return "delete"
	case OpReplace:
		return "replace"
	default:
		return "unknown"
	}
}

// Change represents a single change in a diff.
type Change[T any] struct {
	Op       Operation
	Index    int
	OldValue T
	NewValue T
}

// ObjectChange represents a change in an object field.
type ObjectChange struct {
	Op       Operation
	Path     string
	OldValue interface{}
	NewValue interface{}
}

// Diff computes the differences between two slices.
// Returns a list of changes needed to transform old into new.
func Diff[T comparable](old, new []T) []Change[T] {
	var changes []Change[T]

	// Use LCS-based diff algorithm
	lcs := longestCommonSubsequence(old, new)

	oldIdx, newIdx, lcsIdx := 0, 0, 0

	for oldIdx < len(old) || newIdx < len(new) {
		if lcsIdx < len(lcs) && oldIdx < len(old) && newIdx < len(new) {
			if old[oldIdx] == lcs[lcsIdx] && new[newIdx] == lcs[lcsIdx] {
				// Equal element
				changes = append(changes, Change[T]{
					Op:       OpEqual,
					Index:    newIdx,
					OldValue: old[oldIdx],
					NewValue: new[newIdx],
				})
				oldIdx++
				newIdx++
				lcsIdx++
			} else if old[oldIdx] != lcs[lcsIdx] {
				// Delete from old
				changes = append(changes, Change[T]{
					Op:       OpDelete,
					Index:    oldIdx,
					OldValue: old[oldIdx],
				})
				oldIdx++
			} else {
				// Insert into new
				changes = append(changes, Change[T]{
					Op:       OpInsert,
					Index:    newIdx,
					NewValue: new[newIdx],
				})
				newIdx++
			}
		} else if oldIdx < len(old) {
			// Remaining deletions
			changes = append(changes, Change[T]{
				Op:       OpDelete,
				Index:    oldIdx,
				OldValue: old[oldIdx],
			})
			oldIdx++
		} else if newIdx < len(new) {
			// Remaining insertions
			changes = append(changes, Change[T]{
				Op:       OpInsert,
				Index:    newIdx,
				NewValue: new[newIdx],
			})
			newIdx++
		}
	}

	return changes
}

// longestCommonSubsequence computes the LCS of two slices.
func longestCommonSubsequence[T comparable](a, b []T) []T {
	m, n := len(a), len(b)
	if m == 0 || n == 0 {
		return nil
	}

	// Build LCS table
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}

	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if a[i-1] == b[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else {
				dp[i][j] = max(dp[i-1][j], dp[i][j-1])
			}
		}
	}

	// Backtrack to find LCS
	lcs := make([]T, 0, dp[m][n])
	i, j := m, n
	for i > 0 && j > 0 {
		if a[i-1] == b[j-1] {
			lcs = append([]T{a[i-1]}, lcs...)
			i--
			j--
		} else if dp[i-1][j] > dp[i][j-1] {
			i--
		} else {
			j--
		}
	}

	return lcs
}

// Patch applies a list of changes to a slice.
// Returns the resulting slice after applying all changes.
func Patch[T comparable](old []T, changes []Change[T]) []T {
	result := make([]T, 0, len(old))
	oldIdx := 0

	for _, change := range changes {
		switch change.Op {
		case OpEqual:
			if oldIdx < len(old) {
				result = append(result, old[oldIdx])
				oldIdx++
			}
		case OpInsert:
			result = append(result, change.NewValue)
		case OpDelete:
			oldIdx++
		case OpReplace:
			result = append(result, change.NewValue)
			oldIdx++
		}
	}

	// Append remaining elements
	for oldIdx < len(old) {
		result = append(result, old[oldIdx])
		oldIdx++
	}

	return result
}

// DiffWithEqual computes differences using a custom equality function.
func DiffWithEqual[T any](old, new []T, equal func(a, b T) bool) []Change[T] {
	var changes []Change[T]

	// Simple O(n*m) diff for custom equality
	oldIdx, newIdx := 0, 0

	for oldIdx < len(old) && newIdx < len(new) {
		if equal(old[oldIdx], new[newIdx]) {
			changes = append(changes, Change[T]{
				Op:       OpEqual,
				Index:    newIdx,
				OldValue: old[oldIdx],
				NewValue: new[newIdx],
			})
			oldIdx++
			newIdx++
		} else {
			// Check if old element exists later in new
			foundInNew := false
			for k := newIdx + 1; k < len(new); k++ {
				if equal(old[oldIdx], new[k]) {
					foundInNew = true
					break
				}
			}

			if foundInNew {
				// Insert new element
				changes = append(changes, Change[T]{
					Op:       OpInsert,
					Index:    newIdx,
					NewValue: new[newIdx],
				})
				newIdx++
			} else {
				// Delete old element
				changes = append(changes, Change[T]{
					Op:       OpDelete,
					Index:    oldIdx,
					OldValue: old[oldIdx],
				})
				oldIdx++
			}
		}
	}

	// Handle remaining elements
	for oldIdx < len(old) {
		changes = append(changes, Change[T]{
			Op:       OpDelete,
			Index:    oldIdx,
			OldValue: old[oldIdx],
		})
		oldIdx++
	}

	for newIdx < len(new) {
		changes = append(changes, Change[T]{
			Op:       OpInsert,
			Index:    newIdx,
			NewValue: new[newIdx],
		})
		newIdx++
	}

	return changes
}

// DiffObjects computes differences between two struct objects.
// Returns a list of field changes.
func DiffObjects(old, new interface{}) []ObjectChange {
	var changes []ObjectChange

	oldVal := reflect.ValueOf(old)
	newVal := reflect.ValueOf(new)

	// Handle pointers
	if oldVal.Kind() == reflect.Ptr {
		oldVal = oldVal.Elem()
	}
	if newVal.Kind() == reflect.Ptr {
		newVal = newVal.Elem()
	}

	if oldVal.Kind() != reflect.Struct || newVal.Kind() != reflect.Struct {
		return changes
	}

	oldType := oldVal.Type()
	for i := 0; i < oldVal.NumField(); i++ {
		field := oldType.Field(i)
		if !field.IsExported() {
			continue
		}

		oldField := oldVal.Field(i)
		newField := newVal.FieldByName(field.Name)

		if !newField.IsValid() {
			changes = append(changes, ObjectChange{
				Op:       OpDelete,
				Path:     field.Name,
				OldValue: oldField.Interface(),
			})
			continue
		}

		if !reflect.DeepEqual(oldField.Interface(), newField.Interface()) {
			changes = append(changes, ObjectChange{
				Op:       OpReplace,
				Path:     field.Name,
				OldValue: oldField.Interface(),
				NewValue: newField.Interface(),
			})
		}
	}

	// Check for new fields
	newType := newVal.Type()
	for i := 0; i < newVal.NumField(); i++ {
		field := newType.Field(i)
		if !field.IsExported() {
			continue
		}

		oldField := oldVal.FieldByName(field.Name)
		if !oldField.IsValid() {
			changes = append(changes, ObjectChange{
				Op:       OpInsert,
				Path:     field.Name,
				NewValue: newVal.Field(i).Interface(),
			})
		}
	}

	return changes
}

// DiffMaps computes differences between two maps.
func DiffMaps[K comparable, V any](old, new map[K]V, equal func(a, b V) bool) []ObjectChange {
	var changes []ObjectChange

	// Check for deleted and modified keys
	for k, oldV := range old {
		if newV, exists := new[k]; exists {
			if !equal(oldV, newV) {
				changes = append(changes, ObjectChange{
					Op:       OpReplace,
					Path:     formatKey(k),
					OldValue: oldV,
					NewValue: newV,
				})
			}
		} else {
			changes = append(changes, ObjectChange{
				Op:       OpDelete,
				Path:     formatKey(k),
				OldValue: oldV,
			})
		}
	}

	// Check for inserted keys
	for k, newV := range new {
		if _, exists := old[k]; !exists {
			changes = append(changes, ObjectChange{
				Op:       OpInsert,
				Path:     formatKey(k),
				NewValue: newV,
			})
		}
	}

	return changes
}

// formatKey formats a map key as a string path.
func formatKey[K any](k K) string {
	return reflect.ValueOf(k).String()
}

// HasChanges returns true if there are any non-equal changes.
func HasChanges[T any](changes []Change[T]) bool {
	for _, c := range changes {
		if c.Op != OpEqual {
			return true
		}
	}
	return false
}

// FilterChanges returns only changes matching the given operation.
func FilterChanges[T any](changes []Change[T], op Operation) []Change[T] {
	var filtered []Change[T]
	for _, c := range changes {
		if c.Op == op {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

// CountChanges returns the count of each operation type.
func CountChanges[T any](changes []Change[T]) map[Operation]int {
	counts := make(map[Operation]int)
	for _, c := range changes {
		counts[c.Op]++
	}
	return counts
}
