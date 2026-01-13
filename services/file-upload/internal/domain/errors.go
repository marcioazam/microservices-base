package domain

import (
	"errors"
	"fmt"
)

// Error codes
const (
	ErrCodeInvalidFileType    = "INVALID_FILE_TYPE"
	ErrCodeFileTooLarge       = "FILE_TOO_LARGE"
	ErrCodeExtensionMismatch  = "EXTENSION_MISMATCH"
	ErrCodeMalwareDetected    = "MALWARE_DETECTED"
	ErrCodeInvalidToken       = "INVALID_TOKEN"
	ErrCodeMissingToken       = "MISSING_TOKEN"
	ErrCodeTokenExpired       = "TOKEN_EXPIRED"
	ErrCodeAccessDenied       = "ACCESS_DENIED"
	ErrCodeFileNotFound       = "FILE_NOT_FOUND"
	ErrCodeSessionNotFound    = "SESSION_NOT_FOUND"
	ErrCodeSessionExpired     = "SESSION_EXPIRED"
	ErrCodeRateLimitExceeded  = "RATE_LIMIT_EXCEEDED"
	ErrCodeStorageError       = "STORAGE_ERROR"
	ErrCodeDatabaseError      = "DATABASE_ERROR"
	ErrCodeScannerError       = "SCANNER_ERROR"
	ErrCodeInvalidChunk       = "INVALID_CHUNK"
	ErrCodeDuplicateChunk     = "DUPLICATE_CHUNK"
	ErrCodeChecksumMismatch   = "CHECKSUM_MISMATCH"
	ErrCodeInternalError      = "INTERNAL_ERROR"
)

// DomainError represents a domain-specific error
type DomainError struct {
	Code    string
	Message string
	Err     error
}

func (e *DomainError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *DomainError) Unwrap() error {
	return e.Err
}

// Is implements errors.Is interface
func (e *DomainError) Is(target error) bool {
	t, ok := target.(*DomainError)
	if !ok {
		return false
	}
	return e.Code == t.Code
}

// NewDomainError creates a new domain error
func NewDomainError(code, message string, err error) *DomainError {
	return &DomainError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// Predefined errors
var (
	ErrInvalidFileType = &DomainError{
		Code:    ErrCodeInvalidFileType,
		Message: "file type is not allowed",
	}
	ErrFileTooLarge = &DomainError{
		Code:    ErrCodeFileTooLarge,
		Message: "file exceeds maximum allowed size",
	}
	ErrExtensionMismatch = &DomainError{
		Code:    ErrCodeExtensionMismatch,
		Message: "file extension does not match content type",
	}
	ErrMalwareDetected = &DomainError{
		Code:    ErrCodeMalwareDetected,
		Message: "malware detected in file",
	}
	ErrInvalidToken = &DomainError{
		Code:    ErrCodeInvalidToken,
		Message: "invalid authentication token",
	}
	ErrMissingToken = &DomainError{
		Code:    ErrCodeMissingToken,
		Message: "authentication token is required",
	}
	ErrTokenExpired = &DomainError{
		Code:    ErrCodeTokenExpired,
		Message: "authentication token has expired",
	}
	ErrAccessDenied = &DomainError{
		Code:    ErrCodeAccessDenied,
		Message: "access denied to this resource",
	}
	ErrFileNotFound = &DomainError{
		Code:    ErrCodeFileNotFound,
		Message: "file not found",
	}
	ErrSessionNotFound = &DomainError{
		Code:    ErrCodeSessionNotFound,
		Message: "upload session not found",
	}
	ErrSessionExpired = &DomainError{
		Code:    ErrCodeSessionExpired,
		Message: "upload session has expired",
	}
	ErrRateLimitExceeded = &DomainError{
		Code:    ErrCodeRateLimitExceeded,
		Message: "rate limit exceeded",
	}
	ErrStorageError = &DomainError{
		Code:    ErrCodeStorageError,
		Message: "storage operation failed",
	}
	ErrDatabaseError = &DomainError{
		Code:    ErrCodeDatabaseError,
		Message: "database operation failed",
	}
	ErrScannerError = &DomainError{
		Code:    ErrCodeScannerError,
		Message: "malware scanner unavailable",
	}
	ErrInvalidChunk = &DomainError{
		Code:    ErrCodeInvalidChunk,
		Message: "invalid chunk index",
	}
	ErrDuplicateChunk = &DomainError{
		Code:    ErrCodeDuplicateChunk,
		Message: "chunk already uploaded",
	}
	ErrChecksumMismatch = &DomainError{
		Code:    ErrCodeChecksumMismatch,
		Message: "chunk checksum does not match",
	}
)

// IsValidationError returns true if the error is a validation error
func IsValidationError(err error) bool {
	var domainErr *DomainError
	if !errors.As(err, &domainErr) {
		return false
	}
	switch domainErr.Code {
	case ErrCodeInvalidFileType, ErrCodeFileTooLarge, ErrCodeExtensionMismatch,
		ErrCodeInvalidChunk, ErrCodeDuplicateChunk, ErrCodeChecksumMismatch:
		return true
	}
	return false
}

// IsAuthError returns true if the error is an authentication error
func IsAuthError(err error) bool {
	var domainErr *DomainError
	if !errors.As(err, &domainErr) {
		return false
	}
	switch domainErr.Code {
	case ErrCodeInvalidToken, ErrCodeMissingToken, ErrCodeTokenExpired:
		return true
	}
	return false
}

// IsAuthorizationError returns true if the error is an authorization error
func IsAuthorizationError(err error) bool {
	var domainErr *DomainError
	if !errors.As(err, &domainErr) {
		return false
	}
	return domainErr.Code == ErrCodeAccessDenied
}

// IsNotFoundError returns true if the error is a not found error
func IsNotFoundError(err error) bool {
	var domainErr *DomainError
	if !errors.As(err, &domainErr) {
		return false
	}
	switch domainErr.Code {
	case ErrCodeFileNotFound, ErrCodeSessionNotFound:
		return true
	}
	return false
}

// IsRateLimitError returns true if the error is a rate limit error
func IsRateLimitError(err error) bool {
	var domainErr *DomainError
	if !errors.As(err, &domainErr) {
		return false
	}
	return domainErr.Code == ErrCodeRateLimitExceeded
}

// GetErrorCode extracts the error code from a domain error
func GetErrorCode(err error) string {
	var domainErr *DomainError
	if errors.As(err, &domainErr) {
		return domainErr.Code
	}
	return ErrCodeInternalError
}
