package pipeline

import (
	"errors"
	"testing"
)

func TestPipeline(t *testing.T) {
	t.Run("Use adds stage", func(t *testing.T) {
		p := New[int]().
			Use(func(x int) int { return x + 1 }).
			Use(func(x int) int { return x * 2 })

		result, err := p.Execute(5)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if result != 12 { // (5+1)*2
			t.Errorf("expected 12, got %d", result)
		}
	})

	t.Run("UseWithError handles errors", func(t *testing.T) {
		p := New[int]().
			Use(func(x int) int { return x + 1 }).
			UseWithError(func(x int) (int, error) {
				if x > 5 {
					return 0, errors.New("too big")
				}
				return x, nil
			})

		_, err := p.Execute(5)
		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("UseIf applies conditionally", func(t *testing.T) {
		p := New[int]().
			UseIf(func(x int) bool { return x > 0 }, func(x int) int { return x * 2 })

		result, _ := p.Execute(5)
		if result != 10 {
			t.Errorf("expected 10, got %d", result)
		}

		result, _ = p.Execute(-5)
		if result != -5 {
			t.Errorf("expected -5, got %d", result)
		}
	})

	t.Run("Compose merges pipelines", func(t *testing.T) {
		p1 := New[int]().Use(func(x int) int { return x + 1 })
		p2 := New[int]().Use(func(x int) int { return x * 2 })

		p1.Compose(p2)
		result, _ := p1.Execute(5)
		if result != 12 { // (5+1)*2
			t.Errorf("expected 12, got %d", result)
		}
	})

	t.Run("Clone creates independent copy", func(t *testing.T) {
		p1 := New[int]().Use(func(x int) int { return x + 1 })
		p2 := p1.Clone()
		p2.Use(func(x int) int { return x * 2 })

		r1, _ := p1.Execute(5)
		r2, _ := p2.Execute(5)

		if r1 != 6 {
			t.Errorf("expected 6, got %d", r1)
		}
		if r2 != 12 {
			t.Errorf("expected 12, got %d", r2)
		}
	})

	t.Run("Len returns stage count", func(t *testing.T) {
		p := New[int]().
			Use(func(x int) int { return x }).
			Use(func(x int) int { return x })

		if p.Len() != 2 {
			t.Errorf("expected 2, got %d", p.Len())
		}
	})

	t.Run("Clear removes all stages", func(t *testing.T) {
		p := New[int]().
			Use(func(x int) int { return x + 1 }).
			Clear()

		if p.Len() != 0 {
			t.Error("expected empty pipeline")
		}
	})
}

func TestPipelineHelpers(t *testing.T) {
	t.Run("Map transforms slice", func(t *testing.T) {
		mapper := Map(func(x int) int { return x * 2 })
		result := mapper([]int{1, 2, 3})
		if result[0] != 2 || result[1] != 4 || result[2] != 6 {
			t.Error("unexpected values")
		}
	})

	t.Run("Filter keeps matching", func(t *testing.T) {
		filter := Filter(func(x int) bool { return x%2 == 0 })
		result := filter([]int{1, 2, 3, 4, 5})
		if len(result) != 2 || result[0] != 2 || result[1] != 4 {
			t.Error("unexpected values")
		}
	})

	t.Run("Reduce folds values", func(t *testing.T) {
		reducer := Reduce(0, func(acc, x int) int { return acc + x })
		result := reducer([]int{1, 2, 3, 4, 5})
		if result != 15 {
			t.Errorf("expected 15, got %d", result)
		}
	})
}
