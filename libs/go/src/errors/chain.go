package errors

import (
	"errors"
	"fmt"
)

// Wrap wraps an error with additional context.
func Wrap(err error, message string) *AppError {
	if err == nil {
		return nil
	}
	// If already an AppError, preserve the code
	if appErr, ok := err.(*AppError); ok {
		return &AppError{
			Code:          appErr.Code,
			Message:       message,
			Details:       appErr.Details,
			CorrelationID: appErr.CorrelationID,
			Timestamp:     appErr.Timestamp,
			cause:         err,
		}
	}
	return Internal(message).WithCause(err)
}

// Wrapf wraps an error with a formatted message.
func Wrapf(err error, format string, args ...any) *AppError {
	return Wrap(err, fmt.Sprintf(format, args...))
}

// RootCause traverses the error chain to find the root cause.
func RootCause(err error) error {
	for {
		unwrapped := errors.Unwrap(err)
		if unwrapped == nil {
			return err
		}
		err = unwrapped
	}
}

// Chain returns all errors in the chain.
func Chain(err error) []error {
	var chain []error
	for err != nil {
		chain = append(chain, err)
		err = errors.Unwrap(err)
	}
	return chain
}

// Is checks if any error in the chain matches the target.
func Is(err, target error) bool {
	return errors.Is(err, target)
}

// As finds the first error in the chain that matches target.
func As(err error, target any) bool {
	return errors.As(err, target)
}

// IsCode checks if the error has the specified error code.
func IsCode(err error, code ErrorCode) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code == code
	}
	return false
}

// GetCode extracts the error code from an error.
func GetCode(err error) ErrorCode {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code
	}
	return ErrCodeInternal
}

// GetCorrelationID extracts the correlation ID from an error chain.
func GetCorrelationID(err error) string {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.CorrelationID
	}
	return ""
}
