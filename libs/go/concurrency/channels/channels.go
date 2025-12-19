// Package channels provides generic channel utilities.
package channels

import (
	"sync"
	"time"
)

// Map transforms channel values.
func Map[T, U any](in <-chan T, fn func(T) U) <-chan U {
	out := make(chan U)
	go func() {
		defer close(out)
		for v := range in {
			out <- fn(v)
		}
	}()
	return out
}

// Filter keeps values matching predicate.
func Filter[T any](in <-chan T, predicate func(T) bool) <-chan T {
	out := make(chan T)
	go func() {
		defer close(out)
		for v := range in {
			if predicate(v) {
				out <- v
			}
		}
	}()
	return out
}

// Merge combines multiple channels into one.
func Merge[T any](channels ...<-chan T) <-chan T {
	out := make(chan T)
	var wg sync.WaitGroup
	wg.Add(len(channels))

	for _, ch := range channels {
		go func(c <-chan T) {
			defer wg.Done()
			for v := range c {
				out <- v
			}
		}(ch)
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}

// FanOut distributes values to multiple channels.
func FanOut[T any](in <-chan T, n int) []<-chan T {
	outs := make([]chan T, n)
	for i := range outs {
		outs[i] = make(chan T)
	}

	go func() {
		defer func() {
			for _, ch := range outs {
				close(ch)
			}
		}()
		i := 0
		for v := range in {
			outs[i] <- v
			i = (i + 1) % n
		}
	}()

	result := make([]<-chan T, n)
	for i, ch := range outs {
		result[i] = ch
	}
	return result
}

// FanIn combines multiple channels into one (alias for Merge).
func FanIn[T any](channels ...<-chan T) <-chan T {
	return Merge(channels...)
}

// Buffer creates a buffered channel from an unbuffered one.
func Buffer[T any](in <-chan T, size int) <-chan T {
	out := make(chan T, size)
	go func() {
		defer close(out)
		for v := range in {
			out <- v
		}
	}()
	return out
}

// Batch collects values into batches.
func Batch[T any](in <-chan T, size int, timeout time.Duration) <-chan []T {
	out := make(chan []T)
	go func() {
		defer close(out)
		batch := make([]T, 0, size)
		timer := time.NewTimer(timeout)
		defer timer.Stop()

		for {
			select {
			case v, ok := <-in:
				if !ok {
					if len(batch) > 0 {
						out <- batch
					}
					return
				}
				batch = append(batch, v)
				if len(batch) >= size {
					out <- batch
					batch = make([]T, 0, size)
					timer.Reset(timeout)
				}
			case <-timer.C:
				if len(batch) > 0 {
					out <- batch
					batch = make([]T, 0, size)
				}
				timer.Reset(timeout)
			}
		}
	}()
	return out
}

// Take takes first n values from channel.
func Take[T any](in <-chan T, n int) <-chan T {
	out := make(chan T)
	go func() {
		defer close(out)
		count := 0
		for v := range in {
			if count >= n {
				return
			}
			out <- v
			count++
		}
	}()
	return out
}

// Skip skips first n values from channel.
func Skip[T any](in <-chan T, n int) <-chan T {
	out := make(chan T)
	go func() {
		defer close(out)
		count := 0
		for v := range in {
			if count >= n {
				out <- v
			}
			count++
		}
	}()
	return out
}

// Distinct removes consecutive duplicates.
func Distinct[T comparable](in <-chan T) <-chan T {
	out := make(chan T)
	go func() {
		defer close(out)
		var last T
		first := true
		for v := range in {
			if first || v != last {
				out <- v
				last = v
				first = false
			}
		}
	}()
	return out
}

// Debounce emits value only after quiet period.
func Debounce[T any](in <-chan T, duration time.Duration) <-chan T {
	out := make(chan T)
	go func() {
		defer close(out)
		var timer *time.Timer
		var lastValue T
		hasValue := false

		for {
			select {
			case v, ok := <-in:
				if !ok {
					if hasValue {
						out <- lastValue
					}
					return
				}
				lastValue = v
				hasValue = true
				if timer == nil {
					timer = time.NewTimer(duration)
				} else {
					timer.Reset(duration)
				}
			case <-func() <-chan time.Time {
				if timer != nil {
					return timer.C
				}
				return nil
			}():
				if hasValue {
					out <- lastValue
					hasValue = false
				}
			}
		}
	}()
	return out
}

// Throttle limits throughput to one value per duration.
func Throttle[T any](in <-chan T, duration time.Duration) <-chan T {
	out := make(chan T)
	go func() {
		defer close(out)
		ticker := time.NewTicker(duration)
		defer ticker.Stop()

		for v := range in {
			out <- v
			<-ticker.C
		}
	}()
	return out
}

// Generate creates a channel from a generator function.
func Generate[T any](fn func(yield func(T))) <-chan T {
	out := make(chan T)
	go func() {
		defer close(out)
		fn(func(v T) {
			out <- v
		})
	}()
	return out
}

// FromSlice creates a channel from a slice.
func FromSlice[T any](items []T) <-chan T {
	out := make(chan T)
	go func() {
		defer close(out)
		for _, item := range items {
			out <- item
		}
	}()
	return out
}

// ToSlice collects channel values into a slice.
func ToSlice[T any](in <-chan T) []T {
	result := make([]T, 0)
	for v := range in {
		result = append(result, v)
	}
	return result
}

// Tee duplicates channel to two outputs.
func Tee[T any](in <-chan T) (<-chan T, <-chan T) {
	out1 := make(chan T)
	out2 := make(chan T)
	go func() {
		defer close(out1)
		defer close(out2)
		for v := range in {
			out1 <- v
			out2 <- v
		}
	}()
	return out1, out2
}
