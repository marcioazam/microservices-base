package optics

import "github.com/authcorp/libs/go/src/functional"

// Lens provides a way to focus on a part of a data structure.
type Lens[S, A any] struct {
	Get func(S) A
	Set func(S, A) S
}

// NewLens creates a new Lens.
func NewLens[S, A any](get func(S) A, set func(S, A) S) Lens[S, A] {
	return Lens[S, A]{Get: get, Set: set}
}

// Modify applies a function to the focused value.
func (l Lens[S, A]) Modify(s S, f func(A) A) S {
	return l.Set(s, f(l.Get(s)))
}

// Compose composes two lenses.
func Compose[S, A, B any](outer Lens[S, A], inner Lens[A, B]) Lens[S, B] {
	return Lens[S, B]{
		Get: func(s S) B {
			return inner.Get(outer.Get(s))
		},
		Set: func(s S, b B) S {
			return outer.Set(s, inner.Set(outer.Get(s), b))
		},
	}
}

// Optional is a lens that may not have a value.
type Optional[S, A any] struct {
	GetOption func(S) functional.Option[A]
	Set       func(S, A) S
}

// NewOptional creates a new Optional.
func NewOptional[S, A any](getOption func(S) functional.Option[A], set func(S, A) S) Optional[S, A] {
	return Optional[S, A]{GetOption: getOption, Set: set}
}

// Modify applies a function if value exists.
func (o Optional[S, A]) Modify(s S, f func(A) A) S {
	opt := o.GetOption(s)
	if opt.IsSome() {
		return o.Set(s, f(opt.Unwrap()))
	}
	return s
}

// ModifyOption applies a function that may fail.
func (o Optional[S, A]) ModifyOption(s S, f func(A) functional.Option[A]) S {
	opt := o.GetOption(s)
	if opt.IsSome() {
		newOpt := f(opt.Unwrap())
		if newOpt.IsSome() {
			return o.Set(s, newOpt.Unwrap())
		}
	}
	return s
}

// LensToOptional converts a Lens to an Optional.
func LensToOptional[S, A any](l Lens[S, A]) Optional[S, A] {
	return Optional[S, A]{
		GetOption: func(s S) functional.Option[A] {
			return functional.Some(l.Get(s))
		},
		Set: l.Set,
	}
}

// At creates a lens for map access.
func At[K comparable, V any](key K) Lens[map[K]V, functional.Option[V]] {
	return Lens[map[K]V, functional.Option[V]]{
		Get: func(m map[K]V) functional.Option[V] {
			if v, ok := m[key]; ok {
				return functional.Some(v)
			}
			return functional.None[V]()
		},
		Set: func(m map[K]V, opt functional.Option[V]) map[K]V {
			result := make(map[K]V, len(m))
			for k, v := range m {
				result[k] = v
			}
			if opt.IsSome() {
				result[key] = opt.Unwrap()
			} else {
				delete(result, key)
			}
			return result
		},
	}
}

// Index creates an optional for slice access.
func Index[T any](i int) Optional[[]T, T] {
	return Optional[[]T, T]{
		GetOption: func(s []T) functional.Option[T] {
			if i >= 0 && i < len(s) {
				return functional.Some(s[i])
			}
			return functional.None[T]()
		},
		Set: func(s []T, v T) []T {
			if i >= 0 && i < len(s) {
				result := make([]T, len(s))
				copy(result, s)
				result[i] = v
				return result
			}
			return s
		},
	}
}

// Identity creates an identity lens.
func Identity[S any]() Lens[S, S] {
	return Lens[S, S]{
		Get: func(s S) S { return s },
		Set: func(_ S, s S) S { return s },
	}
}

// First creates a lens for the first element of a pair.
func First[A, B any]() Lens[struct{ First A; Second B }, A] {
	type Pair = struct{ First A; Second B }
	return Lens[Pair, A]{
		Get: func(p Pair) A { return p.First },
		Set: func(p Pair, a A) Pair { return Pair{First: a, Second: p.Second} },
	}
}

// Second creates a lens for the second element of a pair.
func Second[A, B any]() Lens[struct{ First A; Second B }, B] {
	type Pair = struct{ First A; Second B }
	return Lens[Pair, B]{
		Get: func(p Pair) B { return p.Second },
		Set: func(p Pair, b B) Pair { return Pair{First: p.First, Second: b} },
	}
}

// MapAt creates a lens for a map value at a specific key with default.
func MapAt[K comparable, V any](key K, defaultVal V) Lens[map[K]V, V] {
	return Lens[map[K]V, V]{
		Get: func(m map[K]V) V {
			if v, ok := m[key]; ok {
				return v
			}
			return defaultVal
		},
		Set: func(m map[K]V, v V) map[K]V {
			result := make(map[K]V, len(m))
			for k, val := range m {
				result[k] = val
			}
			result[key] = v
			return result
		},
	}
}

// SliceAt creates a lens for a slice element at a specific index with default.
func SliceAt[T any](index int, defaultVal T) Lens[[]T, T] {
	return Lens[[]T, T]{
		Get: func(s []T) T {
			if index >= 0 && index < len(s) {
				return s[index]
			}
			return defaultVal
		},
		Set: func(s []T, v T) []T {
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
