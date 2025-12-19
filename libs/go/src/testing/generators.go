package testing

import (
	"github.com/authcorp/libs/go/src/functional"
	"pgregory.net/rapid"
)

// OptionGen generates Option[T] values.
func OptionGen[T any](valueGen *rapid.Generator[T]) *rapid.Generator[functional.Option[T]] {
	return rapid.Custom(func(t *rapid.T) functional.Option[T] {
		if rapid.Bool().Draw(t, "isSome") {
			return functional.Some(valueGen.Draw(t, "value"))
		}
		return functional.None[T]()
	})
}

// SomeGen generates Some[T] values only.
func SomeGen[T any](valueGen *rapid.Generator[T]) *rapid.Generator[functional.Option[T]] {
	return rapid.Custom(func(t *rapid.T) functional.Option[T] {
		return functional.Some(valueGen.Draw(t, "value"))
	})
}

// NoneGen generates None[T] values only.
func NoneGen[T any]() *rapid.Generator[functional.Option[T]] {
	return rapid.Just(functional.None[T]())
}

// ResultGen generates Result[T] values.
func ResultGen[T any](valueGen *rapid.Generator[T], errGen *rapid.Generator[error]) *rapid.Generator[functional.Result[T]] {
	return rapid.Custom(func(t *rapid.T) functional.Result[T] {
		if rapid.Bool().Draw(t, "isOk") {
			return functional.Ok(valueGen.Draw(t, "value"))
		}
		return functional.Err[T](errGen.Draw(t, "error"))
	})
}

// OkGen generates Ok[T] values only.
func OkGen[T any](valueGen *rapid.Generator[T]) *rapid.Generator[functional.Result[T]] {
	return rapid.Custom(func(t *rapid.T) functional.Result[T] {
		return functional.Ok(valueGen.Draw(t, "value"))
	})
}

// ErrGen generates Err[T] values only.
func ErrGen[T any](errGen *rapid.Generator[error]) *rapid.Generator[functional.Result[T]] {
	return rapid.Custom(func(t *rapid.T) functional.Result[T] {
		return functional.Err[T](errGen.Draw(t, "error"))
	})
}

// EitherGen generates Either[L, R] values.
func EitherGen[L, R any](leftGen *rapid.Generator[L], rightGen *rapid.Generator[R]) *rapid.Generator[functional.Either[L, R]] {
	return rapid.Custom(func(t *rapid.T) functional.Either[L, R] {
		if rapid.Bool().Draw(t, "isRight") {
			return functional.Right[L](rightGen.Draw(t, "right"))
		}
		return functional.Left[L, R](leftGen.Draw(t, "left"))
	})
}

// LeftGen generates Left[L, R] values only.
func LeftGen[L, R any](leftGen *rapid.Generator[L]) *rapid.Generator[functional.Either[L, R]] {
	return rapid.Custom(func(t *rapid.T) functional.Either[L, R] {
		return functional.Left[L, R](leftGen.Draw(t, "left"))
	})
}

// RightGen generates Right[L, R] values only.
func RightGen[L, R any](rightGen *rapid.Generator[R]) *rapid.Generator[functional.Either[L, R]] {
	return rapid.Custom(func(t *rapid.T) functional.Either[L, R] {
		return functional.Right[L](rightGen.Draw(t, "right"))
	})
}

// PairGen generates Pair[A, B] values.
func PairGen[A, B any](firstGen *rapid.Generator[A], secondGen *rapid.Generator[B]) *rapid.Generator[functional.Pair[A, B]] {
	return rapid.Custom(func(t *rapid.T) functional.Pair[A, B] {
		return functional.NewPair(
			firstGen.Draw(t, "first"),
			secondGen.Draw(t, "second"),
		)
	})
}

// ErrorGen generates error values.
func ErrorGen() *rapid.Generator[error] {
	return rapid.Custom(func(t *rapid.T) error {
		msg := rapid.String().Draw(t, "errorMsg")
		return functional.NewError(msg)
	})
}

// NonEmptyStringGen generates non-empty strings.
func NonEmptyStringGen() *rapid.Generator[string] {
	return rapid.StringMatching(`.+`)
}

// AlphanumericGen generates alphanumeric strings.
func AlphanumericGen() *rapid.Generator[string] {
	return rapid.StringMatching(`[a-zA-Z0-9]+`)
}

// PositiveIntGen generates positive integers.
func PositiveIntGen() *rapid.Generator[int] {
	return rapid.IntRange(1, 1000000)
}

// NonNegativeIntGen generates non-negative integers.
func NonNegativeIntGen() *rapid.Generator[int] {
	return rapid.IntRange(0, 1000000)
}

// SliceOfGen generates slices with specified size range.
func SliceOfGen[T any](elemGen *rapid.Generator[T], minSize, maxSize int) *rapid.Generator[[]T] {
	return rapid.SliceOfN(elemGen, minSize, maxSize)
}

// MapOfGen generates maps with specified size range.
func MapOfGen[K comparable, V any](keyGen *rapid.Generator[K], valueGen *rapid.Generator[V], minSize, maxSize int) *rapid.Generator[map[K]V] {
	return rapid.MapOfN(keyGen, valueGen, minSize, maxSize)
}
