package functional

// Pipeline represents a composable sequence of transformations.
type Pipeline[T any] struct {
	value T
}

// NewPipeline creates a new pipeline with an initial value.
func NewPipeline[T any](value T) Pipeline[T] {
	return Pipeline[T]{value: value}
}

// Then applies a transformation and returns a new pipeline.
func Then[T, U any](p Pipeline[T], fn func(T) U) Pipeline[U] {
	return Pipeline[U]{value: fn(p.value)}
}

// Value returns the current pipeline value.
func (p Pipeline[T]) Value() T {
	return p.value
}

// Pipe applies a function to the pipeline value.
func Pipe[T any](value T, fns ...func(T) T) T {
	result := value
	for _, fn := range fns {
		result = fn(result)
	}
	return result
}

// Compose creates a function that applies functions right-to-left.
func Compose[T any](fns ...func(T) T) func(T) T {
	return func(value T) T {
		result := value
		for i := len(fns) - 1; i >= 0; i-- {
			result = fns[i](result)
		}
		return result
	}
}

// AndThen creates a function that applies functions left-to-right.
func AndThen[T any](fns ...func(T) T) func(T) T {
	return func(value T) T {
		result := value
		for _, fn := range fns {
			result = fn(result)
		}
		return result
	}
}

// Identity returns its input unchanged.
func Identity[T any](value T) T {
	return value
}

// Const returns a function that always returns the given value.
func Const[T, U any](value T) func(U) T {
	return func(_ U) T {
		return value
	}
}

// Flip swaps the arguments of a two-argument function.
func Flip[A, B, C any](fn func(A, B) C) func(B, A) C {
	return func(b B, a A) C {
		return fn(a, b)
	}
}

// Curry converts a two-argument function to curried form.
func Curry[A, B, C any](fn func(A, B) C) func(A) func(B) C {
	return func(a A) func(B) C {
		return func(b B) C {
			return fn(a, b)
		}
	}
}

// Uncurry converts a curried function to two-argument form.
func Uncurry[A, B, C any](fn func(A) func(B) C) func(A, B) C {
	return func(a A, b B) C {
		return fn(a)(b)
	}
}
