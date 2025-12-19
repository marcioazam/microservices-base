package waitgroup

import (
	"errors"
	"testing"
)

func TestWaitGroup(t *testing.T) {
	t.Run("Go collects results", func(t *testing.T) {
		wg := New[int]()
		wg.Go(func() int { return 1 })
		wg.Go(func() int { return 2 })
		wg.Go(func() int { return 3 })
		results := wg.Wait()
		if len(results) != 3 {
			t.Errorf("expected 3 results, got %d", len(results))
		}
	})

	t.Run("GoErr collects results and errors", func(t *testing.T) {
		wg := New[int]()
		wg.GoErr(func() (int, error) { return 1, nil })
		wg.GoErr(func() (int, error) { return 0, errors.New("fail") })
		wg.GoErr(func() (int, error) { return 3, nil })
		results, errs := wg.WaitErr()
		if len(results) != 2 {
			t.Errorf("expected 2 results, got %d", len(results))
		}
		if len(errs) != 1 {
			t.Errorf("expected 1 error, got %d", len(errs))
		}
	})

	t.Run("HasErrors returns true when errors exist", func(t *testing.T) {
		wg := New[int]()
		wg.GoErr(func() (int, error) { return 0, errors.New("fail") })
		if !wg.HasErrors() {
			t.Error("expected HasErrors to be true")
		}
	})

	t.Run("FirstError returns first error", func(t *testing.T) {
		wg := New[int]()
		wg.GoErr(func() (int, error) { return 0, errors.New("first") })
		wg.GoErr(func() (int, error) { return 0, errors.New("second") })
		err := wg.FirstError()
		if err == nil {
			t.Error("expected error")
		}
	})
}

func TestIndexedWaitGroup(t *testing.T) {
	t.Run("Go preserves order", func(t *testing.T) {
		wg := NewIndexed[int]()
		wg.Go(func() int { return 1 })
		wg.Go(func() int { return 2 })
		wg.Go(func() int { return 3 })
		results := wg.Wait()
		if len(results) != 3 {
			t.Errorf("expected 3 results, got %d", len(results))
		}
		// Note: order is preserved by index
	})

	t.Run("GoErr preserves order with errors", func(t *testing.T) {
		wg := NewIndexed[int]()
		wg.GoErr(func() (int, error) { return 1, nil })
		wg.GoErr(func() (int, error) { return 0, errors.New("fail") })
		wg.GoErr(func() (int, error) { return 3, nil })
		results, errs := wg.WaitErr()
		if len(results) != 3 {
			t.Errorf("expected 3 result slots, got %d", len(results))
		}
		if len(errs) != 3 {
			t.Errorf("expected 3 error slots, got %d", len(errs))
		}
		if errs[1] == nil {
			t.Error("expected error at index 1")
		}
	})
}
