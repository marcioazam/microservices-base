// Package lens provides generic lens optics for immutable data access.
package lens

// Lens provides access to nested immutable structures.
type Lens[S, A any] struct {
	get func(S) A
	set func(S, A) S
}

// NewLens creates a lens from get and set functions.
func NewLens[S, A any](get func(S) A, set func(S, A) S) Lens[S, A] {
	return Lens[S, A]{get: get, set: set}
}

// Get retrieves the focused value.
func (l Lens[S, A]) Get(source S) A {
	return l.get(source)
}

// Set returns a new structure with the focused value replaced.
func (l Lens[S, A]) Set(source S, value A) S {
	return l.set(source, value)
}

// Modify applies a function to the focused value.
func (l Lens[S, A]) Modify(source S, fn func(A) A) S {
	return l.set(source, fn(l.get(source)))
}

// Compose creates a lens focusing deeper.
func Compose[S, A, B any](outer Lens[S, A], inner Lens[A, B]) Lens[S, B] {
	return Lens[S, B]{
		get: func(s S) B {
			return inner.get(outer.get(s))
		},
		set: func(s S, b B) S {
			return outer.set(s, inner.set(outer.get(s), b))
		},
	}
}

// Identity creates an identity lens.
func Identity[S any]() Lens[S, S] {
	return Lens[S, S]{
		get: func(s S) S { return s },
		set: func(_ S, s S) S { return s },
	}
}

// First creates a lens for the first element of a pair.
func First[A, B any]() Lens[struct{ First A; Second B }, A] {
	type Pair = struct{ First A; Second B }
	return Lens[Pair, A]{
		get: func(p Pair) A { return p.First },
		set: func(p Pair, a A) Pair { return Pair{First: a, Second: p.Second} },
	}
}

// Second creates a lens for the second element of a pair.
func Second[A, B any]() Lens[struct{ First A; Second B }, B] {
	type Pair = struct{ First A; Second B }
	return Lens[Pair, B]{
		get: func(p Pair) B { return p.Second },
		set: func(p Pair, b B) Pair { return Pair{First: p.First, Second: b} },
	}
}

// MapAt creates a lens for a map value at a specific key.
func MapAt[K comparable, V any](key K, defaultVal V) Lens[map[K]V, V] {
	return Lens[map[K]V, V]{
		get: func(m map[K]V) V {
			if v, ok := m[key]; ok {
				return v
			}
			return defaultVal
		},
		set: func(m map[K]V, v V) map[K]V {
			result := make(map[K]V, len(m))
			for k, val := range m {
				result[k] = val
			}
			result[key] = v
			return result
		},
	}
}

// SliceAt creates a lens for a slice element at a specific index.
func SliceAt[T any](index int, defaultVal T) Lens[[]T, T] {
	return Lens[[]T, T]{
		get: func(s []T) T {
			if index >= 0 && index < len(s) {
				return s[index]
			}
			return defaultVal
		},
		set: func(s []T, v T) []T {
			if index < 0 || index >= len(s) {
				return s
			}
			result := make([]T, len(s))
			copy(result, s)
			result[index] = v
			return result
		},
	}
}
