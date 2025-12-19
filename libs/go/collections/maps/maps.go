// Package maps provides generic map utility functions.
package maps

import "github.com/auth-platform/libs/go/functional/option"

// Keys returns all keys from the map.
func Keys[K comparable, V any](m map[K]V) []K {
	result := make([]K, 0, len(m))
	for k := range m {
		result = append(result, k)
	}
	return result
}

// Values returns all values from the map.
func Values[K comparable, V any](m map[K]V) []V {
	result := make([]V, 0, len(m))
	for _, v := range m {
		result = append(result, v)
	}
	return result
}

// Merge combines two maps, with values from the second map taking precedence.
func Merge[K comparable, V any](m1, m2 map[K]V) map[K]V {
	result := make(map[K]V, len(m1)+len(m2))
	for k, v := range m1 {
		result[k] = v
	}
	for k, v := range m2 {
		result[k] = v
	}
	return result
}

// Filter returns a new map containing only entries that satisfy the predicate.
func Filter[K comparable, V any](m map[K]V, predicate func(K, V) bool) map[K]V {
	result := make(map[K]V)
	for k, v := range m {
		if predicate(k, v) {
			result[k] = v
		}
	}
	return result
}

// MapValues applies fn to each value, returning a new map.
func MapValues[K comparable, V, U any](m map[K]V, fn func(V) U) map[K]U {
	result := make(map[K]U, len(m))
	for k, v := range m {
		result[k] = fn(v)
	}
	return result
}

// MapKeys applies fn to each key, returning a new map.
func MapKeys[K1, K2 comparable, V any](m map[K1]V, fn func(K1) K2) map[K2]V {
	result := make(map[K2]V, len(m))
	for k, v := range m {
		result[fn(k)] = v
	}
	return result
}


// Invert swaps keys and values. Requires values to be comparable.
func Invert[K, V comparable](m map[K]V) map[V]K {
	result := make(map[V]K, len(m))
	for k, v := range m {
		result[v] = k
	}
	return result
}

// Get returns the value for a key as an Option.
func Get[K comparable, V any](m map[K]V, key K) option.Option[V] {
	if v, ok := m[key]; ok {
		return option.Some(v)
	}
	return option.None[V]()
}

// GetOrDefault returns the value for a key, or the default if not found.
func GetOrDefault[K comparable, V any](m map[K]V, key K, defaultVal V) V {
	if v, ok := m[key]; ok {
		return v
	}
	return defaultVal
}

// Contains returns true if the map contains the key.
func Contains[K comparable, V any](m map[K]V, key K) bool {
	_, ok := m[key]
	return ok
}

// Clone creates a shallow copy of the map.
func Clone[K comparable, V any](m map[K]V) map[K]V {
	result := make(map[K]V, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}

// ForEach applies fn to each key-value pair.
func ForEach[K comparable, V any](m map[K]V, fn func(K, V)) {
	for k, v := range m {
		fn(k, v)
	}
}

// Entries returns all key-value pairs as a slice of structs.
func Entries[K comparable, V any](m map[K]V) []struct{ Key K; Value V } {
	result := make([]struct{ Key K; Value V }, 0, len(m))
	for k, v := range m {
		result = append(result, struct{ Key K; Value V }{k, v})
	}
	return result
}

// FromEntries creates a map from a slice of key-value pairs.
func FromEntries[K comparable, V any](entries []struct{ Key K; Value V }) map[K]V {
	result := make(map[K]V, len(entries))
	for _, e := range entries {
		result[e.Key] = e.Value
	}
	return result
}

// Pick returns a new map with only the specified keys.
func Pick[K comparable, V any](m map[K]V, keys []K) map[K]V {
	result := make(map[K]V)
	for _, k := range keys {
		if v, ok := m[k]; ok {
			result[k] = v
		}
	}
	return result
}

// Omit returns a new map without the specified keys.
func Omit[K comparable, V any](m map[K]V, keys []K) map[K]V {
	keySet := make(map[K]struct{}, len(keys))
	for _, k := range keys {
		keySet[k] = struct{}{}
	}
	result := make(map[K]V)
	for k, v := range m {
		if _, ok := keySet[k]; !ok {
			result[k] = v
		}
	}
	return result
}

// MergeWith combines two maps using a merge function for conflicting keys.
func MergeWith[K comparable, V any](m1, m2 map[K]V, mergeFn func(V, V) V) map[K]V {
	result := make(map[K]V, len(m1)+len(m2))
	for k, v := range m1 {
		result[k] = v
	}
	for k, v := range m2 {
		if existing, ok := result[k]; ok {
			result[k] = mergeFn(existing, v)
		} else {
			result[k] = v
		}
	}
	return result
}
