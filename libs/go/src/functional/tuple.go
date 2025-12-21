package functional

import "iter"

// Pair represents a tuple of two values.
type Pair[A, B any] struct {
	First  A
	Second B
}

// NewPair creates a new Pair.
func NewPair[A, B any](first A, second B) Pair[A, B] {
	return Pair[A, B]{First: first, Second: second}
}

// Unpack returns the pair's values.
func (p Pair[A, B]) Unpack() (A, B) {
	return p.First, p.Second
}

// Swap returns a new Pair with swapped elements.
func (p Pair[A, B]) Swap() Pair[B, A] {
	return Pair[B, A]{First: p.Second, Second: p.First}
}

// MapPairFirst applies a function to the first element.
func MapPairFirst[A, B, C any](p Pair[A, B], fn func(A) C) Pair[C, B] {
	return Pair[C, B]{First: fn(p.First), Second: p.Second}
}

// MapPairSecond applies a function to the second element.
func MapPairSecond[A, B, C any](p Pair[A, B], fn func(B) C) Pair[A, C] {
	return Pair[A, C]{First: p.First, Second: fn(p.Second)}
}

// MapPairBoth applies functions to both values.
func MapPairBoth[A, B, C, D any](p Pair[A, B], fnA func(A) C, fnB func(B) D) Pair[C, D] {
	return Pair[C, D]{First: fnA(p.First), Second: fnB(p.Second)}
}

// Triple represents a tuple of three values.
type Triple[A, B, C any] struct {
	First  A
	Second B
	Third  C
}

// NewTriple creates a new Triple.
func NewTriple[A, B, C any](first A, second B, third C) Triple[A, B, C] {
	return Triple[A, B, C]{First: first, Second: second, Third: third}
}

// Unpack returns the triple's values.
func (t Triple[A, B, C]) Unpack() (A, B, C) {
	return t.First, t.Second, t.Third
}

// ToPair converts Triple to Pair by dropping the third element.
func (t Triple[A, B, C]) ToPair() Pair[A, B] {
	return Pair[A, B]{First: t.First, Second: t.Second}
}

// Quad represents a tuple of four values.
type Quad[A, B, C, D any] struct {
	First  A
	Second B
	Third  C
	Fourth D
}

// NewQuad creates a new Quad.
func NewQuad[A, B, C, D any](first A, second B, third C, fourth D) Quad[A, B, C, D] {
	return Quad[A, B, C, D]{First: first, Second: second, Third: third, Fourth: fourth}
}

// Unpack returns the quad's values.
func (q Quad[A, B, C, D]) Unpack() (A, B, C, D) {
	return q.First, q.Second, q.Third, q.Fourth
}

// Zip combines two slices into a slice of Pairs.
func Zip[A, B any](as []A, bs []B) []Pair[A, B] {
	minLen := min(len(as), len(bs))
	result := make([]Pair[A, B], minLen)
	for i := 0; i < minLen; i++ {
		result[i] = NewPair(as[i], bs[i])
	}
	return result
}

// Unzip splits a slice of Pairs into two slices.
func Unzip[A, B any](pairs []Pair[A, B]) ([]A, []B) {
	as := make([]A, len(pairs))
	bs := make([]B, len(pairs))
	for i, p := range pairs {
		as[i] = p.First
		bs[i] = p.Second
	}
	return as, bs
}

// ZipWith combines two slices using a function.
func ZipWith[A, B, C any](as []A, bs []B, fn func(A, B) C) []C {
	minLen := min(len(as), len(bs))
	result := make([]C, minLen)
	for i := 0; i < minLen; i++ {
		result[i] = fn(as[i], bs[i])
	}
	return result
}

// Zip3 combines three slices into a slice of Triples.
func Zip3[A, B, C any](as []A, bs []B, cs []C) []Triple[A, B, C] {
	minLen := min(len(as), len(bs), len(cs))
	result := make([]Triple[A, B, C], minLen)
	for i := 0; i < minLen; i++ {
		result[i] = NewTriple(as[i], bs[i], cs[i])
	}
	return result
}

// Unzip3 splits a slice of Triples into three slices.
func Unzip3[A, B, C any](triples []Triple[A, B, C]) ([]A, []B, []C) {
	as := make([]A, len(triples))
	bs := make([]B, len(triples))
	cs := make([]C, len(triples))
	for i, t := range triples {
		as[i] = t.First
		bs[i] = t.Second
		cs[i] = t.Third
	}
	return as, bs, cs
}

// EnumerateSlice returns pairs of (index, value) for a slice.
func EnumerateSlice[T any](items []T) []Pair[int, T] {
	result := make([]Pair[int, T], len(items))
	for i, item := range items {
		result[i] = NewPair(i, item)
	}
	return result
}

// ZipIter returns a Go 1.23+ iterator over pairs from two slices.
func ZipIter[A, B any](as []A, bs []B) iter.Seq[Pair[A, B]] {
	return func(yield func(Pair[A, B]) bool) {
		minLen := min(len(as), len(bs))
		for i := 0; i < minLen; i++ {
			if !yield(NewPair(as[i], bs[i])) {
				return
			}
		}
	}
}

// EnumerateIter returns a Go 1.23+ iterator with index-value pairs.
func EnumerateIter[T any](items []T) iter.Seq2[int, T] {
	return func(yield func(int, T) bool) {
		for i, item := range items {
			if !yield(i, item) {
				return
			}
		}
	}
}
