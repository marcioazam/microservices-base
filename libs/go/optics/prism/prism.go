// Package prism provides generic prism optics for sum types.
package prism

import "github.com/auth-platform/libs/go/functional/option"

// Prism provides access to sum types (variants).
type Prism[S, A any] struct {
	getOption  func(S) option.Option[A]
	reverseGet func(A) S
}

// NewPrism creates a prism from getOption and reverseGet functions.
func NewPrism[S, A any](getOption func(S) option.Option[A], reverseGet func(A) S) Prism[S, A] {
	return Prism[S, A]{getOption: getOption, reverseGet: reverseGet}
}

// GetOption attempts to extract the focused value.
func (p Prism[S, A]) GetOption(source S) option.Option[A] {
	return p.getOption(source)
}

// ReverseGet constructs the source from the focused value.
func (p Prism[S, A]) ReverseGet(value A) S {
	return p.reverseGet(value)
}

// Modify applies a function to the focused value if present.
func (p Prism[S, A]) Modify(source S, fn func(A) A) S {
	opt := p.getOption(source)
	if opt.IsNone() {
		return source
	}
	return p.reverseGet(fn(opt.Unwrap()))
}

// ModifyOption applies a function that may fail.
func (p Prism[S, A]) ModifyOption(source S, fn func(A) option.Option[A]) option.Option[S] {
	opt := p.getOption(source)
	if opt.IsNone() {
		return option.None[S]()
	}
	result := fn(opt.Unwrap())
	if result.IsNone() {
		return option.None[S]()
	}
	return option.Some(p.reverseGet(result.Unwrap()))
}

// Set sets the focused value if the prism matches.
func (p Prism[S, A]) Set(source S, value A) S {
	if p.getOption(source).IsNone() {
		return source
	}
	return p.reverseGet(value)
}

// Compose creates a prism focusing deeper.
func Compose[S, A, B any](outer Prism[S, A], inner Prism[A, B]) Prism[S, B] {
	return Prism[S, B]{
		getOption: func(s S) option.Option[B] {
			outerOpt := outer.getOption(s)
			if outerOpt.IsNone() {
				return option.None[B]()
			}
			return inner.getOption(outerOpt.Unwrap())
		},
		reverseGet: func(b B) S {
			return outer.reverseGet(inner.reverseGet(b))
		},
	}
}

// Some creates a prism for Option[T] that focuses on the Some case.
func Some[T any]() Prism[option.Option[T], T] {
	return Prism[option.Option[T], T]{
		getOption: func(o option.Option[T]) option.Option[T] {
			return o
		},
		reverseGet: func(t T) option.Option[T] {
			return option.Some(t)
		},
	}
}

// StringToInt creates a prism from string to int.
func StringToInt() Prism[string, int] {
	return Prism[string, int]{
		getOption: func(s string) option.Option[int] {
			var n int
			for _, c := range s {
				if c < '0' || c > '9' {
					if c == '-' && n == 0 {
						continue
					}
					return option.None[int]()
				}
				n = n*10 + int(c-'0')
			}
			if len(s) > 0 && s[0] == '-' {
				n = -n
			}
			if len(s) == 0 {
				return option.None[int]()
			}
			return option.Some(n)
		},
		reverseGet: func(n int) string {
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
			// Reverse
			for i, j := 0, len(digits)-1; i < j; i, j = i+1, j-1 {
				digits[i], digits[j] = digits[j], digits[i]
			}
			return string(digits)
		},
	}
}
