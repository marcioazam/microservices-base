package functional

// Pair represents a tuple of two values.
type Pair[A, B any] struct {
	First  A
	Second B
}

// NewPair creates a new Pair.
func NewPair[A, B any](first A, second B) Pair[A, B] {
	return Pair[A, B]{First: first, Second: second}
}

// Swap returns a new Pair with swapped elements.
func (p Pair[A, B]) Swap() Pair[B, A] {
	return Pair[B, A]{First: p.Second, Second: p.First}
}

// MapFirst applies a function to the first element.
func MapPairFirst[A, B, C any](p Pair[A, B], fn func(A) C) Pair[C, B] {
	return Pair[C, B]{First: fn(p.First), Second: p.Second}
}

// MapSecond applies a function to the second element.
func MapPairSecond[A, B, C any](p Pair[A, B], fn func(B) C) Pair[A, C] {
	return Pair[A, C]{First: p.First, Second: fn(p.Second)}
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

// ToPair converts Triple to Pair by dropping the third element.
func (t Triple[A, B, C]) ToPair() Pair[A, B] {
	return Pair[A, B]{First: t.First, Second: t.Second}
}

// Zip combines two slices into a slice of Pairs.
func Zip[A, B any](as []A, bs []B) []Pair[A, B] {
	minLen := len(as)
	if len(bs) < minLen {
		minLen = len(bs)
	}
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
