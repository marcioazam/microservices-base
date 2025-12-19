package functional

import "sync"

// Stream provides lazy, potentially infinite sequences with memoization.
type Stream[T any] struct {
	head     T
	tail     func() *Stream[T]
	tailOnce sync.Once
	tailVal  *Stream[T]
	empty    bool
}

// EmptyStream returns an empty stream.
func EmptyStream[T any]() *Stream[T] {
	return &Stream[T]{empty: true}
}

// ConsStream creates a stream with head and lazy tail.
func ConsStream[T any](head T, tail func() *Stream[T]) *Stream[T] {
	return &Stream[T]{head: head, tail: tail}
}

// StreamFromSlice creates a stream from a slice.
func StreamFromSlice[T any](slice []T) *Stream[T] {
	if len(slice) == 0 {
		return EmptyStream[T]()
	}
	return ConsStream(slice[0], func() *Stream[T] {
		return StreamFromSlice(slice[1:])
	})
}

// IsEmpty returns true if stream is empty.
func (s *Stream[T]) IsEmpty() bool {
	return s == nil || s.empty
}

// Head returns the first element.
func (s *Stream[T]) Head() Option[T] {
	if s.IsEmpty() {
		return None[T]()
	}
	return Some(s.head)
}

// Tail returns the rest of the stream (memoized).
func (s *Stream[T]) Tail() *Stream[T] {
	if s.IsEmpty() || s.tail == nil {
		return EmptyStream[T]()
	}
	s.tailOnce.Do(func() {
		s.tailVal = s.tail()
	})
	return s.tailVal
}

// MapStream transforms stream elements lazily.
func MapStream[T, U any](s *Stream[T], fn func(T) U) *Stream[U] {
	if s.IsEmpty() {
		return EmptyStream[U]()
	}
	return ConsStream(fn(s.head), func() *Stream[U] {
		return MapStream(s.Tail(), fn)
	})
}

// FilterStream keeps elements matching predicate.
func FilterStream[T any](s *Stream[T], pred func(T) bool) *Stream[T] {
	if s.IsEmpty() {
		return EmptyStream[T]()
	}
	if pred(s.head) {
		return ConsStream(s.head, func() *Stream[T] {
			return FilterStream(s.Tail(), pred)
		})
	}
	return FilterStream(s.Tail(), pred)
}

// TakeStream takes first n elements.
func TakeStream[T any](s *Stream[T], n int) *Stream[T] {
	if s.IsEmpty() || n <= 0 {
		return EmptyStream[T]()
	}
	return ConsStream(s.head, func() *Stream[T] {
		return TakeStream(s.Tail(), n-1)
	})
}

// DropStream drops first n elements.
func DropStream[T any](s *Stream[T], n int) *Stream[T] {
	if s.IsEmpty() || n <= 0 {
		return s
	}
	return DropStream(s.Tail(), n-1)
}

// FoldStream reduces stream to single value.
func FoldStream[T, U any](s *Stream[T], initial U, fn func(U, T) U) U {
	if s.IsEmpty() {
		return initial
	}
	return FoldStream(s.Tail(), fn(initial, s.head), fn)
}

// CollectStream materializes stream to slice.
func CollectStream[T any](s *Stream[T]) []T {
	var result []T
	for !s.IsEmpty() {
		result = append(result, s.head)
		s = s.Tail()
	}
	return result
}

// Iterate creates infinite stream from seed and function.
func Iterate[T any](seed T, fn func(T) T) *Stream[T] {
	return ConsStream(seed, func() *Stream[T] {
		return Iterate(fn(seed), fn)
	})
}

// Generate creates infinite stream from generator function.
func Generate[T any](gen func() T) *Stream[T] {
	return ConsStream(gen(), func() *Stream[T] {
		return Generate(gen)
	})
}

// ZipStream combines two streams element-wise.
func ZipStream[T, U any](s1 *Stream[T], s2 *Stream[U]) *Stream[Pair[T, U]] {
	if s1.IsEmpty() || s2.IsEmpty() {
		return EmptyStream[Pair[T, U]]()
	}
	return ConsStream(NewPair(s1.head, s2.head), func() *Stream[Pair[T, U]] {
		return ZipStream(s1.Tail(), s2.Tail())
	})
}

// FlatMapStream maps and flattens streams.
func FlatMapStream[T, U any](s *Stream[T], fn func(T) *Stream[U]) *Stream[U] {
	if s.IsEmpty() {
		return EmptyStream[U]()
	}
	return appendStream(fn(s.head), func() *Stream[U] {
		return FlatMapStream(s.Tail(), fn)
	})
}

// appendStream concatenates stream with lazy continuation.
func appendStream[T any](s *Stream[T], cont func() *Stream[T]) *Stream[T] {
	if s.IsEmpty() {
		return cont()
	}
	return ConsStream(s.head, func() *Stream[T] {
		return appendStream(s.Tail(), cont)
	})
}
