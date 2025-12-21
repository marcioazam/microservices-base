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

// Stage is a processing stage that may fail.
type Stage[T any] func(T) (T, error)

// StagedPipeline is a pipeline with error handling stages.
type StagedPipeline[T any] struct {
	stages []Stage[T]
}

// NewStagedPipeline creates a new staged pipeline.
func NewStagedPipeline[T any]() *StagedPipeline[T] {
	return &StagedPipeline[T]{stages: make([]Stage[T], 0)}
}

// Use adds a stage that cannot fail.
func (p *StagedPipeline[T]) Use(stage func(T) T) *StagedPipeline[T] {
	p.stages = append(p.stages, func(t T) (T, error) {
		return stage(t), nil
	})
	return p
}

// UseWithError adds a stage that may fail.
func (p *StagedPipeline[T]) UseWithError(stage func(T) (T, error)) *StagedPipeline[T] {
	p.stages = append(p.stages, stage)
	return p
}

// UseIf adds a conditional stage.
func (p *StagedPipeline[T]) UseIf(predicate func(T) bool, stage func(T) T) *StagedPipeline[T] {
	p.stages = append(p.stages, func(t T) (T, error) {
		if predicate(t) {
			return stage(t), nil
		}
		return t, nil
	})
	return p
}

// UseIfWithError adds a conditional stage that may fail.
func (p *StagedPipeline[T]) UseIfWithError(predicate func(T) bool, stage func(T) (T, error)) *StagedPipeline[T] {
	p.stages = append(p.stages, func(t T) (T, error) {
		if predicate(t) {
			return stage(t)
		}
		return t, nil
	})
	return p
}

// Execute runs all stages in order.
func (p *StagedPipeline[T]) Execute(input T) (T, error) {
	current := input
	for _, stage := range p.stages {
		result, err := stage(current)
		if err != nil {
			return current, err
		}
		current = result
	}
	return current, nil
}

// Compose merges another pipeline's stages.
func (p *StagedPipeline[T]) Compose(other *StagedPipeline[T]) *StagedPipeline[T] {
	p.stages = append(p.stages, other.stages...)
	return p
}

// Clone creates a copy of the pipeline.
func (p *StagedPipeline[T]) Clone() *StagedPipeline[T] {
	stages := make([]Stage[T], len(p.stages))
	copy(stages, p.stages)
	return &StagedPipeline[T]{stages: stages}
}

// Len returns the number of stages.
func (p *StagedPipeline[T]) Len() int {
	return len(p.stages)
}

// Clear removes all stages.
func (p *StagedPipeline[T]) Clear() *StagedPipeline[T] {
	p.stages = make([]Stage[T], 0)
	return p
}

// ThenStaged creates a function that runs first pipeline then transforms and runs second.
func ThenStaged[T, U any](first *StagedPipeline[T], transform func(T) U, second *StagedPipeline[U]) func(T) (U, error) {
	return func(input T) (U, error) {
		result, err := first.Execute(input)
		if err != nil {
			var zero U
			return zero, err
		}
		return second.Execute(transform(result))
	}
}

// PipelineMap creates a function that maps slice values.
func PipelineMap[T, U any](fn func(T) U) func([]T) []U {
	return func(items []T) []U {
		result := make([]U, len(items))
		for i, item := range items {
			result[i] = fn(item)
		}
		return result
	}
}

// PipelineFilter creates a function that filters slice values.
func PipelineFilter[T any](predicate func(T) bool) func([]T) []T {
	return func(items []T) []T {
		result := make([]T, 0)
		for _, item := range items {
			if predicate(item) {
				result = append(result, item)
			}
		}
		return result
	}
}

// PipelineReduce creates a function that reduces slice values.
func PipelineReduce[T, U any](initial U, fn func(U, T) U) func([]T) U {
	return func(items []T) U {
		result := initial
		for _, item := range items {
			result = fn(result, item)
		}
		return result
	}
}
