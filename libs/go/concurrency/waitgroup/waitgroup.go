// Package waitgroup provides a generic WaitGroup with result collection.
package waitgroup

import "sync"

// WaitGroup collects results from goroutines.
type WaitGroup[T any] struct {
	wg      sync.WaitGroup
	mu      sync.Mutex
	results []T
	errors  []error
}

// New creates a new WaitGroup.
func New[T any]() *WaitGroup[T] {
	return &WaitGroup[T]{
		results: make([]T, 0),
		errors:  make([]error, 0),
	}
}

// Go starts a goroutine that returns a value.
func (wg *WaitGroup[T]) Go(fn func() T) {
	wg.wg.Add(1)
	go func() {
		defer wg.wg.Done()
		result := fn()
		wg.mu.Lock()
		wg.results = append(wg.results, result)
		wg.mu.Unlock()
	}()
}

// GoErr starts a goroutine that may return an error.
func (wg *WaitGroup[T]) GoErr(fn func() (T, error)) {
	wg.wg.Add(1)
	go func() {
		defer wg.wg.Done()
		result, err := fn()
		wg.mu.Lock()
		if err != nil {
			wg.errors = append(wg.errors, err)
		} else {
			wg.results = append(wg.results, result)
		}
		wg.mu.Unlock()
	}()
}

// Wait waits for all goroutines and returns results.
func (wg *WaitGroup[T]) Wait() []T {
	wg.wg.Wait()
	return wg.results
}

// WaitErr waits for all goroutines and returns results and errors.
func (wg *WaitGroup[T]) WaitErr() ([]T, []error) {
	wg.wg.Wait()
	return wg.results, wg.errors
}

// HasErrors returns true if any goroutine returned an error.
func (wg *WaitGroup[T]) HasErrors() bool {
	wg.wg.Wait()
	return len(wg.errors) > 0
}

// FirstError returns the first error, if any.
func (wg *WaitGroup[T]) FirstError() error {
	wg.wg.Wait()
	if len(wg.errors) > 0 {
		return wg.errors[0]
	}
	return nil
}

// IndexedWaitGroup collects results with preserved order.
type IndexedWaitGroup[T any] struct {
	wg      sync.WaitGroup
	mu      sync.Mutex
	results map[int]T
	errors  map[int]error
	count   int
}

// NewIndexed creates a new IndexedWaitGroup.
func NewIndexed[T any]() *IndexedWaitGroup[T] {
	return &IndexedWaitGroup[T]{
		results: make(map[int]T),
		errors:  make(map[int]error),
	}
}

// Go starts a goroutine with an index.
func (wg *IndexedWaitGroup[T]) Go(fn func() T) int {
	wg.mu.Lock()
	idx := wg.count
	wg.count++
	wg.mu.Unlock()

	wg.wg.Add(1)
	go func(index int) {
		defer wg.wg.Done()
		result := fn()
		wg.mu.Lock()
		wg.results[index] = result
		wg.mu.Unlock()
	}(idx)

	return idx
}

// GoErr starts a goroutine with an index that may return an error.
func (wg *IndexedWaitGroup[T]) GoErr(fn func() (T, error)) int {
	wg.mu.Lock()
	idx := wg.count
	wg.count++
	wg.mu.Unlock()

	wg.wg.Add(1)
	go func(index int) {
		defer wg.wg.Done()
		result, err := fn()
		wg.mu.Lock()
		if err != nil {
			wg.errors[index] = err
		} else {
			wg.results[index] = result
		}
		wg.mu.Unlock()
	}(idx)

	return idx
}

// Wait waits and returns results in order.
func (wg *IndexedWaitGroup[T]) Wait() []T {
	wg.wg.Wait()
	results := make([]T, wg.count)
	for i := 0; i < wg.count; i++ {
		if r, ok := wg.results[i]; ok {
			results[i] = r
		}
	}
	return results
}

// WaitErr waits and returns results and errors in order.
func (wg *IndexedWaitGroup[T]) WaitErr() ([]T, []error) {
	wg.wg.Wait()
	results := make([]T, wg.count)
	errors := make([]error, wg.count)
	for i := 0; i < wg.count; i++ {
		if r, ok := wg.results[i]; ok {
			results[i] = r
		}
		if e, ok := wg.errors[i]; ok {
			errors[i] = e
		}
	}
	return results, errors
}
