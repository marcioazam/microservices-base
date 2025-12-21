package optics

import "github.com/authcorp/libs/go/src/functional"

// Prism provides access to sum types (variants).
type Prism[S, A any] struct {
	GetOption  func(S) functional.Option[A]
	ReverseGet func(A) S
}

// NewPrism creates a prism from getOption and reverseGet functions.
func NewPrism[S, A any](getOption func(S) functional.Option[A], reverseGet func(A) S) Prism[S, A] {
	return Prism[S, A]{GetOption: getOption, ReverseGet: reverseGet}
}

// Modify applies a function to the focused value if present.
func (p Prism[S, A]) Modify(source S, fn func(A) A) S {
	opt := p.GetOption(source)
	if opt.IsNone() {
		return source
	}
	return p.ReverseGet(fn(opt.Unwrap()))
}

// ModifyOption applies a function that may fail.
func (p Prism[S, A]) ModifyOption(source S, fn func(A) functional.Option[A]) functional.Option[S] {
	opt := p.GetOption(source)
	if opt.IsNone() {
		return functional.None[S]()
	}
	result := fn(opt.Unwrap())
	if result.IsNone() {
		return functional.None[S]()
	}
	return functional.Some(p.ReverseGet(result.Unwrap()))
}

// Set sets the focused value if the prism matches.
func (p Prism[S, A]) Set(source S, value A) S {
	if p.GetOption(source).IsNone() {
		return source
	}
	return p.ReverseGet(value)
}

// ComposePrism creates a prism focusing deeper.
func ComposePrism[S, A, B any](outer Prism[S, A], inner Prism[A, B]) Prism[S, B] {
	return Prism[S, B]{
		GetOption: func(s S) functional.Option[B] {
			outerOpt := outer.GetOption(s)
			if outerOpt.IsNone() {
				return functional.None[B]()
			}
			return inner.GetOption(outerOpt.Unwrap())
		},
		ReverseGet: func(b B) S {
			return outer.ReverseGet(inner.ReverseGet(b))
		},
	}
}

// SomePrism creates a prism for Option[T] that focuses on the Some case.
func SomePrism[T any]() Prism[functional.Option[T], T] {
	return Prism[functional.Option[T], T]{
		GetOption: func(o functional.Option[T]) functional.Option[T] {
			return o
		},
		ReverseGet: func(t T) functional.Option[T] {
			return functional.Some(t)
		},
	}
}

// Iso represents an isomorphism between two types.
type Iso[S, A any] struct {
	Get     func(S) A
	Reverse func(A) S
}

// NewIso creates a new isomorphism.
func NewIso[S, A any](get func(S) A, reverse func(A) S) Iso[S, A] {
	return Iso[S, A]{Get: get, Reverse: reverse}
}

// ToLens converts an Iso to a Lens.
func (i Iso[S, A]) ToLens() Lens[S, A] {
	return Lens[S, A]{
		Get: i.Get,
		Set: func(_ S, a A) S { return i.Reverse(a) },
	}
}

// ToPrism converts an Iso to a Prism.
func (i Iso[S, A]) ToPrism() Prism[S, A] {
	return Prism[S, A]{
		GetOption:  func(s S) functional.Option[A] { return functional.Some(i.Get(s)) },
		ReverseGet: i.Reverse,
	}
}

// ComposeIso composes two isomorphisms.
func ComposeIso[S, A, B any](outer Iso[S, A], inner Iso[A, B]) Iso[S, B] {
	return Iso[S, B]{
		Get:     func(s S) B { return inner.Get(outer.Get(s)) },
		Reverse: func(b B) S { return outer.Reverse(inner.Reverse(b)) },
	}
}

// StringToInt creates a prism from string to int.
func StringToInt() Prism[string, int] {
	return Prism[string, int]{
		GetOption: func(s string) functional.Option[int] {
			var n int
			for _, c := range s {
				if c < '0' || c > '9' {
					if c == '-' && n == 0 {
						continue
					}
					return functional.None[int]()
				}
				n = n*10 + int(c-'0')
			}
			if len(s) > 0 && s[0] == '-' {
				n = -n
			}
			if len(s) == 0 {
				return functional.None[int]()
			}
			return functional.Some(n)
		},
		ReverseGet: func(n int) string {
			if n == 0 {
				return "0"
			}
			neg := n < 0
			if neg {
				n = -n
			}
			digits := make([]byte, 0, 20)
			for n > 0 {
				digits = append(digits, byte('0'+n%10))
				n /= 10
			}
			if neg {
				digits = append(digits, '-')
			}
			for i, j := 0, len(digits)-1; i < j; i, j = i+1, j-1 {
				digits[i], digits[j] = digits[j], digits[i]
			}
			return string(digits)
		},
	}
}
