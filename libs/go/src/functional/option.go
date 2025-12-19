package functional

// Option represents an optional value that may or may not be present.
// It provides a type-safe alternative to nil pointers.
type Option[T any] struct {
	value   T
	present bool
}

// Some creates an Option containing a value.
func Some[T any](value T) Option[T] {
	return Option[T]{value: value, present: true}
}

// None creates an empty Option.
func None[T any]() Option[T] {
	return Option[T]{present: false}
}

// IsSome returns true if the Option contains a value.
func (o Option[T]) IsSome() bool {
	return o.present
}

// IsNone returns true if the Option is empty.
func (o Option[T]) IsNone() bool {
	return !o.present
}

// Unwrap returns the contained value or panics if empty.
func (o Option[T]) Unwrap() T {
	if !o.present {
		panic("called Unwrap on None")
	}
	return o.value
}

// UnwrapOr returns the contained value or a default.
func (o Option[T]) UnwrapOr(defaultValue T) T {
	if o.present {
		return o.value
	}
	return defaultValue
}

// UnwrapOrElse returns the contained value or computes a default.
func (o Option[T]) UnwrapOrElse(fn func() T) T {
	if o.present {
		return o.value
	}
	return fn()
}

// Map applies a function to the contained value if present.
func (o Option[T]) Map(fn func(T) T) Functor[T] {
	if o.present {
		return Some(fn(o.value))
	}
	return None[T]()
}

// MapOption applies a transformation function to Option.
func MapOption[T, U any](o Option[T], fn func(T) U) Option[U] {
	if o.present {
		return Some(fn(o.value))
	}
	return None[U]()
}

// FlatMap applies a function that returns an Option.
func FlatMapOption[T, U any](o Option[T], fn func(T) Option[U]) Option[U] {
	if o.present {
		return fn(o.value)
	}
	return None[U]()
}

// Match executes one of two functions based on Option state.
func (o Option[T]) Match(onSome func(T), onNone func()) {
	if o.present {
		onSome(o.value)
	} else {
		onNone()
	}
}

// MatchReturn executes one of two functions and returns the result.
func MatchOption[T, U any](o Option[T], onSome func(T) U, onNone func() U) U {
	if o.present {
		return onSome(o.value)
	}
	return onNone()
}

// Filter returns None if predicate returns false.
func (o Option[T]) Filter(predicate func(T) bool) Option[T] {
	if o.present && predicate(o.value) {
		return o
	}
	return None[T]()
}

// ToSlice converts Option to a slice (empty or single element).
func (o Option[T]) ToSlice() []T {
	if o.present {
		return []T{o.value}
	}
	return []T{}
}

// FromPtr creates an Option from a pointer.
func FromPtr[T any](ptr *T) Option[T] {
	if ptr == nil {
		return None[T]()
	}
	return Some(*ptr)
}

// ToPtr converts Option to a pointer.
func (o Option[T]) ToPtr() *T {
	if o.present {
		return &o.value
	}
	return nil
}
