package error

import (
	"encoding/json"
	"net/http"
	"testing"

	"pgregory.net/rapid"
)

// **Feature: auth-microservices-platform, Property 26: Error Response Structure**
// **Validates: Requirements 10.1, 10.2, 10.3, 10.4, 10.5**
func TestErrorResponseStructure(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random error data
		code := rapid.SampledFrom([]string{
			"AUTH_TOKEN_MISSING",
			"AUTH_TOKEN_INVALID",
			"AUTH_TOKEN_EXPIRED",
			"AUTHZ_DENIED",
			"VALIDATION_ERROR",
			"SERVICE_UNAVAILABLE",
			"INTERNAL_ERROR",
		}).Draw(t, "code")

		message := rapid.String().Draw(t, "message")
		correlationID := rapid.StringMatching(`[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}`).Draw(t, "correlationID")

		resp := NewErrorResponse(code, message, correlationID)

		// Verify all required fields are present
		if resp.Code == "" {
			t.Fatal("Error code must be present")
		}
		if resp.Message == "" {
			t.Fatal("Error message must be present")
		}
		if resp.CorrelationID == "" {
			t.Fatal("Correlation ID must be present")
		}

		// Verify JSON serialization
		jsonData, err := resp.ToJSON()
		if err != nil {
			t.Fatalf("Failed to serialize error response: %v", err)
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal(jsonData, &parsed); err != nil {
			t.Fatalf("Failed to parse JSON: %v", err)
		}

		errorObj, ok := parsed["error"].(map[string]interface{})
		if !ok {
			t.Fatal("JSON should have 'error' wrapper")
		}

		if errorObj["code"] != code {
			t.Fatalf("Code mismatch: expected %s, got %v", code, errorObj["code"])
		}
		if errorObj["correlation_id"] != correlationID {
			t.Fatalf("Correlation ID mismatch: expected %s, got %v", correlationID, errorObj["correlation_id"])
		}
	})
}

func TestHTTPStatusMapping(t *testing.T) {
	tests := []struct {
		code           string
		expectedStatus int
	}{
		{"AUTH_TOKEN_MISSING", http.StatusUnauthorized},
		{"AUTH_TOKEN_INVALID", http.StatusUnauthorized},
		{"AUTH_TOKEN_EXPIRED", http.StatusUnauthorized},
		{"AUTH_CREDENTIALS_INVALID", http.StatusUnauthorized},
		{"AUTHZ_DENIED", http.StatusForbidden},
		{"VALIDATION_ERROR", http.StatusBadRequest},
		{"OAUTH_PKCE_REQUIRED", http.StatusBadRequest},
		{"OAUTH_PKCE_INVALID", http.StatusBadRequest},
		{"SERVICE_UNAVAILABLE", http.StatusServiceUnavailable},
		{"NOT_FOUND", http.StatusNotFound},
		{"UNKNOWN_ERROR", http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			resp := NewErrorResponse(tt.code, "test message", "test-correlation")
			status := resp.HTTPStatus()

			if status != tt.expectedStatus {
				t.Errorf("HTTPStatus() for %s = %d, want %d", tt.code, status, tt.expectedStatus)
			}
		})
	}
}

func TestErrorResponseWithDetails(t *testing.T) {
	resp := NewErrorResponse("VALIDATION_ERROR", "Invalid input", "test-correlation").
		WithDetails(map[string]string{
			"field":   "email",
			"message": "Invalid email format",
		})

	if resp.Details == nil {
		t.Fatal("Details should not be nil")
	}

	if resp.Details["field"] != "email" {
		t.Errorf("Expected field 'email', got %s", resp.Details["field"])
	}
}

func TestValidateErrorResponse(t *testing.T) {
	tests := []struct {
		name     string
		resp     *ErrorResponse
		expected bool
	}{
		{
			name: "valid response",
			resp: &ErrorResponse{
				Code:          "AUTH_TOKEN_INVALID",
				Message:       "Token is invalid",
				CorrelationID: "test-123",
			},
			expected: true,
		},
		{
			name: "missing code",
			resp: &ErrorResponse{
				Message:       "Token is invalid",
				CorrelationID: "test-123",
			},
			expected: false,
		},
		{
			name: "missing message",
			resp: &ErrorResponse{
				Code:          "AUTH_TOKEN_INVALID",
				CorrelationID: "test-123",
			},
			expected: false,
		},
		{
			name: "missing correlation ID",
			resp: &ErrorResponse{
				Code:    "AUTH_TOKEN_INVALID",
				Message: "Token is invalid",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateErrorResponse(tt.resp)
			if result != tt.expected {
				t.Errorf("ValidateErrorResponse() = %v, want %v", result, tt.expected)
			}
		})
	}
}
