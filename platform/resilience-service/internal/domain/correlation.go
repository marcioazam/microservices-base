package domain

// DefaultCorrelationFn returns a no-op correlation function that returns empty string.
func DefaultCorrelationFn() func() string {
	return func() string { return "" }
}

// NewCorrelationFn creates a correlation function with fallback to default.
// If fn is nil, returns the default correlation function.
func NewCorrelationFn(fn func() string) func() string {
	if fn == nil {
		return DefaultCorrelationFn()
	}
	return fn
}
