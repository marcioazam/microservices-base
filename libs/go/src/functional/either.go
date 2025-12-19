package functional

// Either represents a value of one of two possible types.
// By convention, Left is used for errors and Right for success values.
type Either[L, R any] struct {
	left    L
	right   R
	isRight bool
}

// Left creates an Either with a left value.
func Left[L, R any](value L) Either[L, R] {
	return Either[L, R]{left: value, isRight: false}
}

// Right creates an Either with a right value.
func Right[L, R any](value R) Either[L, R] {
	return Either[L, R]{right: value, isRight: true}
}

// IsLeft returns true if Either contains a left value.
func (e Either[L, R]) IsLeft() bool {
	return !e.isRight
}

// IsRight returns true if Either contains a right value.
func (e Either[L, R]) IsRight() bool {
	return e.isRight
}

// LeftValue returns the left value or panics.
func (e Either[L, R]) LeftValue() L {
	if e.isRight {
		panic("called LeftValue on Right")
	}
	return e.left
}

// RightValue returns the right value or panics.
func (e Either[L, R]) RightValue() R {
	if !e.isRight {
		panic("called RightValue on Left")
	}
	return e.right
}

// LeftOr returns the left value or a default.
func (e Either[L, R]) LeftOr(defaultValue L) L {
	if !e.isRight {
		return e.left
	}
	return defaultValue
}

// RightOr returns the right value or a default.
func (e Either[L, R]) RightOr(defaultValue R) R {
	if e.isRight {
		return e.right
	}
	return defaultValue
}

// MapRight applies a function to the right value.
func MapEitherRight[L, R, U any](e Either[L, R], fn func(R) U) Either[L, U] {
	if e.isRight {
		return Right[L, U](fn(e.right))
	}
	return Left[L, U](e.left)
}

// MapLeft applies a function to the left value.
func MapEitherLeft[L, R, U any](e Either[L, R], fn func(L) U) Either[U, R] {
	if !e.isRight {
		return Left[U, R](fn(e.left))
	}
	return Right[U, R](e.right)
}

// FlatMapRight applies a function that returns an Either.
func FlatMapEitherRight[L, R, U any](e Either[L, R], fn func(R) Either[L, U]) Either[L, U] {
	if e.isRight {
		return fn(e.right)
	}
	return Left[L, U](e.left)
}

// Match executes one of two functions based on Either state.
func (e Either[L, R]) Match(onLeft func(L), onRight func(R)) {
	if e.isRight {
		onRight(e.right)
	} else {
		onLeft(e.left)
	}
}

// MatchReturn executes one of two functions and returns the result.
func MatchEither[L, R, U any](e Either[L, R], onLeft func(L) U, onRight func(R) U) U {
	if e.isRight {
		return onRight(e.right)
	}
	return onLeft(e.left)
}

// Swap exchanges left and right values.
func (e Either[L, R]) Swap() Either[R, L] {
	if e.isRight {
		return Left[R, L](e.right)
	}
	return Right[R, L](e.left)
}

// EitherToResult converts Either[error, T] to Result[T].
func EitherToResult[T any](e Either[error, T]) Result[T] {
	if e.IsRight() {
		return Ok(e.RightValue())
	}
	return Err[T](e.LeftValue())
}

// ResultToEither converts Result[T] to Either[error, T].
func ResultToEither[T any](r Result[T]) Either[error, T] {
	if r.IsOk() {
		return Right[error, T](r.Unwrap())
	}
	return Left[error, T](r.UnwrapErr())
}

// Note: ToResult as a method is not possible in Go due to type parameter constraints.
// Use the standalone function EitherToResult[T](e Either[error, T]) instead.
