package fault

import "errors"

// IsCircuitOpen checks if error is CircuitOpenError.
func IsCircuitOpen(err error) bool {
	var circuitErr *CircuitOpenError
	return errors.As(err, &circuitErr)
}

// IsRateLimited checks if error is RateLimitError.
func IsRateLimited(err error) bool {
	var rateLimitErr *RateLimitError
	return errors.As(err, &rateLimitErr)
}

// IsTimeout checks if error is TimeoutError.
func IsTimeout(err error) bool {
	var timeoutErr *TimeoutError
	return errors.As(err, &timeoutErr)
}

// IsBulkheadFull checks if error is BulkheadFullError.
func IsBulkheadFull(err error) bool {
	var bulkheadErr *BulkheadFullError
	return errors.As(err, &bulkheadErr)
}

// IsRetryExhausted checks if error is RetryExhaustedError.
func IsRetryExhausted(err error) bool {
	var retryErr *RetryExhaustedError
	return errors.As(err, &retryErr)
}

// IsInvalidPolicy checks if error is InvalidPolicyError.
func IsInvalidPolicy(err error) bool {
	var policyErr *InvalidPolicyError
	return errors.As(err, &policyErr)
}

// IsResilienceError checks if error is any ResilienceError.
func IsResilienceError(err error) bool {
	var resErr *ResilienceError
	return errors.As(err, &resErr)
}

// GetErrorCode extracts error code from ResilienceError.
func GetErrorCode(err error) (ErrorCode, bool) {
	var resErr *ResilienceError
	if errors.As(err, &resErr) {
		return resErr.Code, true
	}
	// Check specific error types
	if IsCircuitOpen(err) {
		return ErrCodeCircuitOpen, true
	}
	if IsRateLimited(err) {
		return ErrCodeRateLimited, true
	}
	if IsTimeout(err) {
		return ErrCodeTimeout, true
	}
	if IsBulkheadFull(err) {
		return ErrCodeBulkheadFull, true
	}
	if IsRetryExhausted(err) {
		return ErrCodeRetryExhausted, true
	}
	if IsInvalidPolicy(err) {
		return ErrCodeInvalidPolicy, true
	}
	return "", false
}

// AsCircuitOpenError extracts CircuitOpenError from error chain.
func AsCircuitOpenError(err error) (*CircuitOpenError, bool) {
	var circuitErr *CircuitOpenError
	if errors.As(err, &circuitErr) {
		return circuitErr, true
	}
	return nil, false
}

// AsRateLimitError extracts RateLimitError from error chain.
func AsRateLimitError(err error) (*RateLimitError, bool) {
	var rateLimitErr *RateLimitError
	if errors.As(err, &rateLimitErr) {
		return rateLimitErr, true
	}
	return nil, false
}

// AsTimeoutError extracts TimeoutError from error chain.
func AsTimeoutError(err error) (*TimeoutError, bool) {
	var timeoutErr *TimeoutError
	if errors.As(err, &timeoutErr) {
		return timeoutErr, true
	}
	return nil, false
}

// AsBulkheadFullError extracts BulkheadFullError from error chain.
func AsBulkheadFullError(err error) (*BulkheadFullError, bool) {
	var bulkheadErr *BulkheadFullError
	if errors.As(err, &bulkheadErr) {
		return bulkheadErr, true
	}
	return nil, false
}

// AsRetryExhaustedError extracts RetryExhaustedError from error chain.
func AsRetryExhaustedError(err error) (*RetryExhaustedError, bool) {
	var retryErr *RetryExhaustedError
	if errors.As(err, &retryErr) {
		return retryErr, true
	}
	return nil, false
}

// AsInvalidPolicyError extracts InvalidPolicyError from error chain.
func AsInvalidPolicyError(err error) (*InvalidPolicyError, bool) {
	var policyErr *InvalidPolicyError
	if errors.As(err, &policyErr) {
		return policyErr, true
	}
	return nil, false
}
