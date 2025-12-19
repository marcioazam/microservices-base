// Package pipeline provides a generic processing pipeline.
package pipeline

// Stage is a processing stage.
type Stage[T any] func(T) (T, error)

// Pipeline is a generic processing pipeline.
type Pipeline[T any] struct {
	stages []Stage[T]
}

// New creates a new Pipeline.
func New[T any]() *Pipeline[T] {
	return &Pipeline[T]{stages: make([]Stage[T], 0)}
}

// Use adds a stage that cannot fail.
func (p *Pipeline[T]) Use(stage func(T) T) *Pipeline[T] {
	p.stages = append(p.stages, func(t T) (T, error) {
		return stage(t), nil
	})
	return p
}

// UseWithError adds a stage that may fail.
func (p *Pipeline[T]) UseWithError(stage func(T) (T, error)) *Pipeline[T] {
	p.stages = append(p.stages, stage)
	return p
}

// UseIf adds a conditional stage.
func (p *Pipeline[T]) UseIf(predicate func(T) bool, stage func(T) T) *Pipeline[T] {
	p.stages = append(p.stages, func(t T) (T, error) {
		if predicate(t) {
			return stage(t), nil
		}
		return t, nil
	})
	return p
}

// UseIfWithError adds a conditional stage that may fail.
func (p *Pipeline[T]) UseIfWithError(predicate func(T) bool, stage func(T) (T, error)) *Pipeline[T] {
	p.stages = append(p.stages, func(t T) (T, error) {
		if predicate(t) {
			return stage(t)
		}
		return t, nil
	})
	return p
}

// Execute runs all stages in order.
func (p *Pipeline[T]) Execute(input T) (T, error) {
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
func (p *Pipeline[T]) Compose(other *Pipeline[T]) *Pipeline[T] {
	p.stages = append(p.stages, other.stages...)
	return p
}

// Clone creates a copy of the pipeline.
func (p *Pipeline[T]) Clone() *Pipeline[T] {
	stages := make([]Stage[T], len(p.stages))
	copy(stages, p.stages)
	return &Pipeline[T]{stages: stages}
}

// Len returns the number of stages.
func (p *Pipeline[T]) Len() int {
	return len(p.stages)
}

// Clear removes all stages.
func (p *Pipeline[T]) Clear() *Pipeline[T] {
	p.stages = make([]Stage[T], 0)
	return p
}

// Then creates a new pipeline that runs this pipeline then another.
func Then[T, U any](first *Pipeline[T], transform func(T) U, second *Pipeline[U]) func(T) (U, error) {
	return func(input T) (U, error) {
		result, err := first.Execute(input)
		if err != nil {
			var zero U
			return zero, err
		}
		return second.Execute(transform(result))
	}
}

// Map creates a pipeline that maps values.
func Map[T, U any](fn func(T) U) func([]T) []U {
	return func(items []T) []U {
		result := make([]U, len(items))
		for i, item := range items {
			result[i] = fn(item)
		}
		return result
	}
}

// Filter creates a pipeline that filters values.
func Filter[T any](predicate func(T) bool) func([]T) []T {
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

// Reduce creates a pipeline that reduces values.
func Reduce[T, U any](initial U, fn func(U, T) U) func([]T) U {
	return func(items []T) U {
		result := initial
		for _, item := range items {
			result = fn(result, item)
		}
		return result
	}
}
