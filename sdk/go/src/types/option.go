package types

// Option represents an optional value that may or may not be present.
type Option[T any] struct {
	value T
	some  bool
}

// Some creates an Option with a value.
func Some[T any](value T) Option[T] {
	return Option[T]{value: value, some: true}
}

// None creates an empty Option.
func None[T any]() Option[T] {
	return Option[T]{}
}

// IsSome returns true if the Option contains a value.
func (o Option[T]) IsSome() bool {
	return o.some
}

// IsNone returns true if the Option is empty.
func (o Option[T]) IsNone() bool {
	return !o.some
}

// Unwrap returns the value if present, panics otherwise.
func (o Option[T]) Unwrap() T {
	if !o.some {
		panic("called Unwrap on a None Option")
	}
	return o.value
}

// UnwrapOr returns the value if present, or the default value otherwise.
func (o Option[T]) UnwrapOr(defaultValue T) T {
	if !o.some {
		return defaultValue
	}
	return o.value
}

// Value returns the value and a boolean indicating presence.
func (o Option[T]) Value() (T, bool) {
	return o.value, o.some
}

// Match applies the appropriate function based on presence.
func (o Option[T]) Match(onSome func(T), onNone func()) {
	if o.some {
		onSome(o.value)
	} else {
		onNone()
	}
}

// MapOption transforms the value if present.
func MapOption[T, U any](o Option[T], fn func(T) U) Option[U] {
	if !o.some {
		return None[U]()
	}
	return Some(fn(o.value))
}

// FlatMapOption chains operations that may return empty.
func FlatMapOption[T, U any](o Option[T], fn func(T) Option[U]) Option[U] {
	if !o.some {
		return None[U]()
	}
	return fn(o.value)
}

// Filter returns None if the predicate returns false.
func Filter[T any](o Option[T], predicate func(T) bool) Option[T] {
	if !o.some || !predicate(o.value) {
		return None[T]()
	}
	return o
}

// OkOr converts an Option to a Result, using the given error if None.
func OkOr[T any](o Option[T], err error) Result[T] {
	if o.some {
		return Ok(o.value)
	}
	return Err[T](err)
}

// ToOption converts a Result to an Option, discarding the error.
func ToOption[T any](r Result[T]) Option[T] {
	if r.ok {
		return Some(r.value)
	}
	return None[T]()
}
