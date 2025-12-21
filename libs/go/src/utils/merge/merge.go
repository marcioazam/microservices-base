// Package merge provides generic merge utilities for slices and maps.
package merge

import (
	"reflect"
)

// Strategy defines how to handle conflicts during merge.
type Strategy int

const (
	// StrategyOverwrite replaces old values with new values.
	StrategyOverwrite Strategy = iota
	// StrategyKeep keeps old values, ignoring new values.
	StrategyKeep
	// StrategyAppend appends new values (for slices).
	StrategyAppend
)

// Slices merges multiple slices into one.
// Elements are appended in order.
func Slices[T any](slices ...[]T) []T {
	totalLen := 0
	for _, s := range slices {
		totalLen += len(s)
	}

	result := make([]T, 0, totalLen)
	for _, s := range slices {
		result = append(result, s...)
	}
	return result
}

// SlicesUnique merges slices keeping only unique elements.
// Uses the first occurrence of each element.
func SlicesUnique[T comparable](slices ...[]T) []T {
	seen := make(map[T]bool)
	var result []T

	for _, s := range slices {
		for _, v := range s {
			if !seen[v] {
				seen[v] = true
				result = append(result, v)
			}
		}
	}
	return result
}

// SlicesUniqueBy merges slices keeping unique elements by key function.
func SlicesUniqueBy[T any, K comparable](slices [][]T, keyFn func(T) K) []T {
	seen := make(map[K]bool)
	var result []T

	for _, s := range slices {
		for _, v := range s {
			key := keyFn(v)
			if !seen[key] {
				seen[key] = true
				result = append(result, v)
			}
		}
	}
	return result
}

// Maps merges multiple maps into one.
// Later maps overwrite earlier maps for duplicate keys.
func Maps[K comparable, V any](maps ...map[K]V) map[K]V {
	result := make(map[K]V)
	for _, m := range maps {
		for k, v := range m {
			result[k] = v
		}
	}
	return result
}

// MapsWithStrategy merges maps using the specified strategy.
func MapsWithStrategy[K comparable, V any](strategy Strategy, maps ...map[K]V) map[K]V {
	result := make(map[K]V)

	for _, m := range maps {
		for k, v := range m {
			if _, exists := result[k]; exists {
				switch strategy {
				case StrategyOverwrite:
					result[k] = v
				case StrategyKeep:
					// Keep existing value
				}
			} else {
				result[k] = v
			}
		}
	}
	return result
}

// MapsWithResolver merges maps using a custom resolver for conflicts.
func MapsWithResolver[K comparable, V any](resolver func(key K, old, new V) V, maps ...map[K]V) map[K]V {
	result := make(map[K]V)

	for _, m := range maps {
		for k, v := range m {
			if existing, exists := result[k]; exists {
				result[k] = resolver(k, existing, v)
			} else {
				result[k] = v
			}
		}
	}
	return result
}

// DeepMerge performs a deep merge of two values.
// Supports maps, slices, and structs.
func DeepMerge(dst, src interface{}) interface{} {
	return deepMerge(reflect.ValueOf(dst), reflect.ValueOf(src)).Interface()
}

func deepMerge(dst, src reflect.Value) reflect.Value {
	// Handle nil values
	if !src.IsValid() {
		return dst
	}
	if !dst.IsValid() {
		return src
	}

	// Handle pointers
	if dst.Kind() == reflect.Ptr {
		if dst.IsNil() {
			return src
		}
		dst = dst.Elem()
	}
	if src.Kind() == reflect.Ptr {
		if src.IsNil() {
			return dst
		}
		src = src.Elem()
	}

	// Type mismatch - return src
	if dst.Kind() != src.Kind() {
		return src
	}

	switch dst.Kind() {
	case reflect.Map:
		return mergeMapValues(dst, src)
	case reflect.Slice:
		return mergeSliceValues(dst, src)
	case reflect.Struct:
		return mergeStructValues(dst, src)
	default:
		// For primitive types, return src (overwrite)
		return src
	}
}

func mergeMapValues(dst, src reflect.Value) reflect.Value {
	if dst.IsNil() {
		return src
	}
	if src.IsNil() {
		return dst
	}

	result := reflect.MakeMap(dst.Type())

	// Copy dst values
	for _, key := range dst.MapKeys() {
		result.SetMapIndex(key, dst.MapIndex(key))
	}

	// Merge src values
	for _, key := range src.MapKeys() {
		srcVal := src.MapIndex(key)
		if dstVal := result.MapIndex(key); dstVal.IsValid() {
			// Deep merge nested values
			merged := deepMerge(dstVal, srcVal)
			result.SetMapIndex(key, merged)
		} else {
			result.SetMapIndex(key, srcVal)
		}
	}

	return result
}

func mergeSliceValues(dst, src reflect.Value) reflect.Value {
	if dst.IsNil() {
		return src
	}
	if src.IsNil() {
		return dst
	}

	// Append slices
	result := reflect.MakeSlice(dst.Type(), 0, dst.Len()+src.Len())
	result = reflect.AppendSlice(result, dst)
	result = reflect.AppendSlice(result, src)
	return result
}

func mergeStructValues(dst, src reflect.Value) reflect.Value {
	result := reflect.New(dst.Type()).Elem()

	for i := 0; i < dst.NumField(); i++ {
		dstField := dst.Field(i)
		srcField := src.Field(i)

		if !result.Field(i).CanSet() {
			continue
		}

		// Deep merge field values
		merged := deepMerge(dstField, srcField)
		if merged.IsValid() && merged.Type().AssignableTo(result.Field(i).Type()) {
			result.Field(i).Set(merged)
		}
	}

	return result
}

// MergeSliceAt inserts elements at a specific index.
func MergeSliceAt[T any](slice []T, index int, elements ...T) []T {
	if index < 0 {
		index = 0
	}
	if index > len(slice) {
		index = len(slice)
	}

	result := make([]T, 0, len(slice)+len(elements))
	result = append(result, slice[:index]...)
	result = append(result, elements...)
	result = append(result, slice[index:]...)
	return result
}

// MergeSliceOrdered merges slices maintaining sorted order.
// Requires a less function for comparison.
func MergeSliceOrdered[T any](less func(a, b T) bool, slices ...[]T) []T {
	merged := Slices(slices...)
	
	// Simple insertion sort for stability
	for i := 1; i < len(merged); i++ {
		j := i
		for j > 0 && less(merged[j], merged[j-1]) {
			merged[j], merged[j-1] = merged[j-1], merged[j]
			j--
		}
	}
	
	return merged
}

// MergeMapSliceValues merges maps where values are slices.
func MergeMapSliceValues[K comparable, V any](maps ...map[K][]V) map[K][]V {
	result := make(map[K][]V)

	for _, m := range maps {
		for k, v := range m {
			result[k] = append(result[k], v...)
		}
	}
	return result
}

// Interleave merges slices by alternating elements.
func Interleave[T any](slices ...[]T) []T {
	if len(slices) == 0 {
		return nil
	}

	maxLen := 0
	for _, s := range slices {
		if len(s) > maxLen {
			maxLen = len(s)
		}
	}

	var result []T
	for i := 0; i < maxLen; i++ {
		for _, s := range slices {
			if i < len(s) {
				result = append(result, s[i])
			}
		}
	}
	return result
}

// Zip merges two slices into pairs.
func Zip[T, U any](a []T, b []U) [][2]interface{} {
	minLen := len(a)
	if len(b) < minLen {
		minLen = len(b)
	}

	result := make([][2]interface{}, minLen)
	for i := 0; i < minLen; i++ {
		result[i] = [2]interface{}{a[i], b[i]}
	}
	return result
}

// ZipWith merges two slices using a combiner function.
func ZipWith[T, U, R any](a []T, b []U, combine func(T, U) R) []R {
	minLen := len(a)
	if len(b) < minLen {
		minLen = len(b)
	}

	result := make([]R, minLen)
	for i := 0; i < minLen; i++ {
		result[i] = combine(a[i], b[i])
	}
	return result
}
