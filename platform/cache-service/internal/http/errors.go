package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/auth-platform/cache-service/internal/cache"
	"github.com/auth-platform/cache-service/internal/observability"
)

// ErrorResponse represents a JSON error response.
type ErrorResponse struct {
	Error         string `json:"error"`
	Code          string `json:"code,omitempty"`
	Message       string `json:"message,omitempty"`
	CorrelationID string `json:"correlation_id,omitempty"`
}

// WriteError writes a standardized error response.
func WriteError(w http.ResponseWriter, r *http.Request, err error) {
	correlationID := observability.GetCorrelationID(r.Context())

	var cacheErr *cache.Error
	if errors.As(err, &cacheErr) {
		writeJSONError(w, cacheErr.ToHTTPStatus(), ErrorResponse{
			Error:         cacheErr.Code.String(),
			Code:          cacheErr.Code.String(),
			Message:       cacheErr.Message,
			CorrelationID: correlationID,
		})
		return
	}

	writeJSONError(w, http.StatusInternalServerError, ErrorResponse{
		Error:         "internal_error",
		Message:       "An internal error occurred",
		CorrelationID: correlationID,
	})
}

// WriteErrorWithStatus writes an error response with a specific status code.
func WriteErrorWithStatus(w http.ResponseWriter, r *http.Request, status int, message string) {
	correlationID := observability.GetCorrelationID(r.Context())
	writeJSONError(w, status, ErrorResponse{
		Error:         http.StatusText(status),
		Message:       message,
		CorrelationID: correlationID,
	})
}

// WriteBadRequest writes a 400 Bad Request error response.
func WriteBadRequest(w http.ResponseWriter, r *http.Request, message string) {
	WriteErrorWithStatus(w, r, http.StatusBadRequest, message)
}

// WriteUnauthorized writes a 401 Unauthorized error response.
func WriteUnauthorized(w http.ResponseWriter, r *http.Request, message string) {
	WriteErrorWithStatus(w, r, http.StatusUnauthorized, message)
}

// WriteForbidden writes a 403 Forbidden error response.
func WriteForbidden(w http.ResponseWriter, r *http.Request, message string) {
	WriteErrorWithStatus(w, r, http.StatusForbidden, message)
}

// WriteNotFound writes a 404 Not Found error response.
func WriteNotFound(w http.ResponseWriter, r *http.Request, message string) {
	WriteErrorWithStatus(w, r, http.StatusNotFound, message)
}

// WriteServiceUnavailable writes a 503 Service Unavailable error response.
func WriteServiceUnavailable(w http.ResponseWriter, r *http.Request, message string) {
	WriteErrorWithStatus(w, r, http.StatusServiceUnavailable, message)
}

// WriteInternalError writes a 500 Internal Server Error response.
func WriteInternalError(w http.ResponseWriter, r *http.Request, message string) {
	WriteErrorWithStatus(w, r, http.StatusInternalServerError, message)
}

func writeJSONError(w http.ResponseWriter, status int, response ErrorResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response) // Error intentionally ignored - response already committed
}
