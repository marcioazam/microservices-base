package error

import (
	"encoding/json"
	"net/http"
)

// ErrorResponse represents a structured error response
type ErrorResponse struct {
	Code          string            `json:"code"`
	Message       string            `json:"message"`
	CorrelationID string            `json:"correlation_id"`
	Details       map[string]string `json:"details,omitempty"`
}

// Error implements the error interface
func (e *ErrorResponse) Error() string {
	return e.Message
}

// ToJSON converts the error response to JSON
func (e *ErrorResponse) ToJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"error": e,
	})
}

// HTTPStatus returns the appropriate HTTP status code for the error
func (e *ErrorResponse) HTTPStatus() int {
	switch e.Code {
	case "AUTH_TOKEN_MISSING", "AUTH_TOKEN_INVALID", "AUTH_TOKEN_EXPIRED", "AUTH_CREDENTIALS_INVALID":
		return http.StatusUnauthorized
	case "AUTHZ_DENIED":
		return http.StatusForbidden
	case "VALIDATION_ERROR", "OAUTH_PKCE_REQUIRED", "OAUTH_PKCE_INVALID":
		return http.StatusBadRequest
	case "SERVICE_UNAVAILABLE":
		return http.StatusServiceUnavailable
	case "NOT_FOUND":
		return http.StatusNotFound
	default:
		return http.StatusInternalServerError
	}
}

// NewErrorResponse creates a new error response
func NewErrorResponse(code, message, correlationID string) *ErrorResponse {
	return &ErrorResponse{
		Code:          code,
		Message:       message,
		CorrelationID: correlationID,
	}
}

// WithDetails adds details to the error response
func (e *ErrorResponse) WithDetails(details map[string]string) *ErrorResponse {
	e.Details = details
	return e
}

// Common error constructors

func TokenMissing(correlationID string) *ErrorResponse {
	return NewErrorResponse("AUTH_TOKEN_MISSING", "Authorization token is required", correlationID)
}

func TokenInvalid(correlationID string) *ErrorResponse {
	return NewErrorResponse("AUTH_TOKEN_INVALID", "Token signature is invalid", correlationID)
}

func TokenExpired(correlationID string) *ErrorResponse {
	return NewErrorResponse("AUTH_TOKEN_EXPIRED", "Token has expired", correlationID)
}

func AuthorizationDenied(correlationID, reason string) *ErrorResponse {
	return NewErrorResponse("AUTHZ_DENIED", "Authorization denied", correlationID).
		WithDetails(map[string]string{"reason": reason})
}

func ValidationError(correlationID, field, message string) *ErrorResponse {
	return NewErrorResponse("VALIDATION_ERROR", message, correlationID).
		WithDetails(map[string]string{"field": field})
}

func ServiceUnavailable(correlationID, service string, retryAfter int) *ErrorResponse {
	return NewErrorResponse("SERVICE_UNAVAILABLE", "Service temporarily unavailable", correlationID).
		WithDetails(map[string]string{
			"service":     service,
			"retry_after": string(rune(retryAfter)),
		})
}

// ValidateErrorResponse checks if an error response has all required fields
func ValidateErrorResponse(resp *ErrorResponse) bool {
	return resp.Code != "" && resp.Message != "" && resp.CorrelationID != ""
}
