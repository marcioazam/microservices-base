// Package errors provides RFC 7807 Problem Details error responses.
package errors

import (
	"encoding/json"
	"net/http"
)

// ProblemDetails represents an RFC 7807 Problem Details response.
type ProblemDetails struct {
	Type          string         `json:"type"`
	Title         string         `json:"title"`
	Status        int            `json:"status"`
	Detail        string         `json:"detail,omitempty"`
	Instance      string         `json:"instance,omitempty"`
	CorrelationID string         `json:"correlation_id,omitempty"`
	Extensions    map[string]any `json:"extensions,omitempty"`
}

// ErrorCode represents application error codes.
type ErrorCode string

// Error codes.
const (
	ErrInvalidFileType    ErrorCode = "INVALID_FILE_TYPE"
	ErrFileTooLarge       ErrorCode = "FILE_TOO_LARGE"
	ErrExtensionMismatch  ErrorCode = "EXTENSION_MISMATCH"
	ErrInvalidChunk       ErrorCode = "INVALID_CHUNK"
	ErrDuplicateChunk     ErrorCode = "DUPLICATE_CHUNK"
	ErrChecksumMismatch   ErrorCode = "CHECKSUM_MISMATCH"
	ErrMissingToken       ErrorCode = "MISSING_TOKEN"
	ErrInvalidToken       ErrorCode = "INVALID_TOKEN"
	ErrTokenExpired       ErrorCode = "TOKEN_EXPIRED"
	ErrAccessDenied       ErrorCode = "ACCESS_DENIED"
	ErrFileNotFound       ErrorCode = "FILE_NOT_FOUND"
	ErrSessionNotFound    ErrorCode = "SESSION_NOT_FOUND"
	ErrSessionExpired     ErrorCode = "SESSION_EXPIRED"
	ErrRateLimitExceeded  ErrorCode = "RATE_LIMIT_EXCEEDED"
	ErrStorageError       ErrorCode = "STORAGE_ERROR"
	ErrDatabaseError      ErrorCode = "DATABASE_ERROR"
	ErrScannerError       ErrorCode = "SCANNER_ERROR"
	ErrInternalError      ErrorCode = "INTERNAL_ERROR"
)

// errorMapping maps error codes to HTTP status and titles.
var errorMapping = map[ErrorCode]struct {
	Status int
	Title  string
}{
	ErrInvalidFileType:   {http.StatusBadRequest, "Invalid File Type"},
	ErrFileTooLarge:      {http.StatusBadRequest, "File Too Large"},
	ErrExtensionMismatch: {http.StatusBadRequest, "Extension Mismatch"},
	ErrInvalidChunk:      {http.StatusBadRequest, "Invalid Chunk"},
	ErrDuplicateChunk:    {http.StatusBadRequest, "Duplicate Chunk"},
	ErrChecksumMismatch:  {http.StatusBadRequest, "Checksum Mismatch"},
	ErrMissingToken:      {http.StatusUnauthorized, "Missing Token"},
	ErrInvalidToken:      {http.StatusUnauthorized, "Invalid Token"},
	ErrTokenExpired:      {http.StatusUnauthorized, "Token Expired"},
	ErrAccessDenied:      {http.StatusForbidden, "Access Denied"},
	ErrFileNotFound:      {http.StatusNotFound, "File Not Found"},
	ErrSessionNotFound:   {http.StatusNotFound, "Session Not Found"},
	ErrSessionExpired:    {http.StatusGone, "Session Expired"},
	ErrRateLimitExceeded: {http.StatusTooManyRequests, "Rate Limit Exceeded"},
	ErrStorageError:      {http.StatusInternalServerError, "Storage Error"},
	ErrDatabaseError:     {http.StatusInternalServerError, "Database Error"},
	ErrScannerError:      {http.StatusInternalServerError, "Scanner Error"},
	ErrInternalError:     {http.StatusInternalServerError, "Internal Error"},
}

// NewProblemDetails creates a new ProblemDetails from an error code.
func NewProblemDetails(code ErrorCode, detail, instance, correlationID string) *ProblemDetails {
	mapping, ok := errorMapping[code]
	if !ok {
		mapping = errorMapping[ErrInternalError]
	}

	return &ProblemDetails{
		Type:          "https://api.example.com/errors/" + string(code),
		Title:         mapping.Title,
		Status:        mapping.Status,
		Detail:        detail,
		Instance:      instance,
		CorrelationID: correlationID,
	}
}

// WithExtension adds an extension to the problem details.
func (p *ProblemDetails) WithExtension(key string, value any) *ProblemDetails {
	if p.Extensions == nil {
		p.Extensions = make(map[string]any)
	}
	p.Extensions[key] = value
	return p
}

// Write writes the problem details to the response.
func (p *ProblemDetails) Write(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.Header().Set("X-Correlation-ID", p.CorrelationID)
	w.WriteHeader(p.Status)
	json.NewEncoder(w).Encode(p)
}

// WriteError writes an error response.
func WriteError(w http.ResponseWriter, code ErrorCode, detail, instance, correlationID string) {
	problem := NewProblemDetails(code, detail, instance, correlationID)
	problem.Write(w)
}

// WriteErrorWithExtensions writes an error response with extensions.
func WriteErrorWithExtensions(w http.ResponseWriter, code ErrorCode, detail, instance, correlationID string, extensions map[string]any) {
	problem := NewProblemDetails(code, detail, instance, correlationID)
	problem.Extensions = extensions
	problem.Write(w)
}

// GetHTTPStatus returns the HTTP status for an error code.
func GetHTTPStatus(code ErrorCode) int {
	if mapping, ok := errorMapping[code]; ok {
		return mapping.Status
	}
	return http.StatusInternalServerError
}
