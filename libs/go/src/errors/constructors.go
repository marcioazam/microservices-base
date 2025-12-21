package errors

import "time"

// New creates a new AppError with the given code and message.
func New(code ErrorCode, message string) *AppError {
	return &AppError{
		Code:      code,
		Message:   message,
		Timestamp: time.Now(),
	}
}

// Validation creates a validation error.
func Validation(message string) *AppError {
	return New(ErrCodeValidation, message)
}

// NotFound creates a not found error.
func NotFound(resource string) *AppError {
	return New(ErrCodeNotFound, resource+" not found")
}

// Unauthorized creates an unauthorized error.
func Unauthorized(message string) *AppError {
	return New(ErrCodeUnauthorized, message)
}

// Forbidden creates a forbidden error.
func Forbidden(message string) *AppError {
	return New(ErrCodeForbidden, message)
}

// Conflict creates a conflict error.
func Conflict(message string) *AppError {
	return New(ErrCodeConflict, message)
}

// BadRequest creates a bad request error.
func BadRequest(message string) *AppError {
	return New(ErrCodeBadRequest, message)
}

// TooManyRequests creates a rate limit error.
func TooManyRequests(message string) *AppError {
	return New(ErrCodeTooManyReqs, message)
}

// Internal creates an internal server error.
func Internal(message string) *AppError {
	return New(ErrCodeInternal, message)
}

// Unavailable creates a service unavailable error.
func Unavailable(message string) *AppError {
	return New(ErrCodeUnavailable, message)
}

// Timeout creates a timeout error.
func Timeout(message string) *AppError {
	return New(ErrCodeTimeout, message)
}

// NotImplemented creates a not implemented error.
func NotImplemented(message string) *AppError {
	return New(ErrCodeNotImplemented, message)
}

// Dependency creates a dependency error.
func Dependency(service, message string) *AppError {
	return New(ErrCodeDependency, message).WithDetail("service", service)
}

// BusinessRule creates a business rule violation error.
func BusinessRule(rule, message string) *AppError {
	return New(ErrCodeBusinessRule, message).WithDetail("rule", rule)
}

// InvalidState creates an invalid state error.
func InvalidState(expected, actual string) *AppError {
	return New(ErrCodeInvalidState, "invalid state transition").
		WithDetail("expected", expected).
		WithDetail("actual", actual)
}
