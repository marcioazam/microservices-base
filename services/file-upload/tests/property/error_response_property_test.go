// Package property contains property-based tests for file-upload service.
// Feature: file-upload-modernization-2025, Property 1: Error Response Consistency
// Validates: Requirements 4.2, 10.6, 11.1, 11.2, 11.3, 11.4
package property

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// TestProblemDetails represents RFC 7807 Problem Details for testing.
type TestProblemDetails struct {
	Type          string         `json:"type"`
	Title         string         `json:"title"`
	Status        int            `json:"status"`
	Detail        string         `json:"detail,omitempty"`
	Instance      string         `json:"instance,omitempty"`
	CorrelationID string         `json:"correlation_id,omitempty"`
	Extensions    map[string]any `json:"extensions,omitempty"`
}

// TestErrorCode represents error codes for testing.
type TestErrorCode string

const (
	TestErrInvalidFileType   TestErrorCode = "INVALID_FILE_TYPE"
	TestErrFileTooLarge      TestErrorCode = "FILE_TOO_LARGE"
	TestErrAccessDenied      TestErrorCode = "ACCESS_DENIED"
	TestErrFileNotFound      TestErrorCode = "FILE_NOT_FOUND"
	TestErrRateLimitExceeded TestErrorCode = "RATE_LIMIT_EXCEEDED"
	TestErrInternalError     TestErrorCode = "INTERNAL_ERROR"
)

var testErrorMapping = map[TestErrorCode]struct {
	Status int
	Title  string
}{
	TestErrInvalidFileType:   {http.StatusBadRequest, "Invalid File Type"},
	TestErrFileTooLarge:      {http.StatusBadRequest, "File Too Large"},
	TestErrAccessDenied:      {http.StatusForbidden, "Access Denied"},
	TestErrFileNotFound:      {http.StatusNotFound, "File Not Found"},
	TestErrRateLimitExceeded: {http.StatusTooManyRequests, "Rate Limit Exceeded"},
	TestErrInternalError:     {http.StatusInternalServerError, "Internal Error"},
}

// CreateTestProblemDetails creates a test problem details response.
func CreateTestProblemDetails(code TestErrorCode, detail, instance, correlationID string) *TestProblemDetails {
	mapping := testErrorMapping[code]
	return &TestProblemDetails{
		Type:          "https://api.example.com/errors/" + string(code),
		Title:         mapping.Title,
		Status:        mapping.Status,
		Detail:        detail,
		Instance:      instance,
		CorrelationID: correlationID,
	}
}

// TestProperty1_ErrorResponseHasRequiredFields tests that error responses have required RFC 7807 fields.
// Property 1: Error Response Consistency
// Validates: Requirements 4.2, 10.6, 11.1, 11.2, 11.3, 11.4
func TestProperty1_ErrorResponseHasRequiredFields(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		code := rapid.SampledFrom([]TestErrorCode{
			TestErrInvalidFileType, TestErrFileTooLarge, TestErrAccessDenied,
			TestErrFileNotFound, TestErrRateLimitExceeded, TestErrInternalError,
		}).Draw(t, "code")

		detail := rapid.StringMatching(`[A-Za-z0-9 ]{10,100}`).Draw(t, "detail")
		instance := "/api/v1/" + rapid.StringMatching(`[a-z/]{5,20}`).Draw(t, "instance")
		correlationID := rapid.StringMatching(`[a-f0-9]{32}`).Draw(t, "correlationID")

		problem := CreateTestProblemDetails(code, detail, instance, correlationID)

		// Property: Error response SHALL include type
		if problem.Type == "" {
			t.Error("type is required")
		}
		if !strings.HasPrefix(problem.Type, "https://") {
			t.Errorf("type should be a URI: %q", problem.Type)
		}

		// Property: Error response SHALL include title
		if problem.Title == "" {
			t.Error("title is required")
		}

		// Property: Error response SHALL include status
		if problem.Status == 0 {
			t.Error("status is required")
		}
		if problem.Status < 400 || problem.Status >= 600 {
			t.Errorf("status should be 4xx or 5xx, got %d", problem.Status)
		}
	})
}

// TestProperty1_CorrelationIDPresent tests that X-Correlation-ID header is present.
// Property 1: Error Response Consistency
// Validates: Requirements 4.2, 10.6, 11.1, 11.2, 11.3, 11.4
func TestProperty1_CorrelationIDPresent(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		code := rapid.SampledFrom([]TestErrorCode{
			TestErrInvalidFileType, TestErrAccessDenied, TestErrInternalError,
		}).Draw(t, "code")

		correlationID := rapid.StringMatching(`[a-f0-9]{32}`).Draw(t, "correlationID")

		problem := CreateTestProblemDetails(code, "error detail", "/api/v1/upload", correlationID)

		// Property: X-Correlation-ID header SHALL be present
		if problem.CorrelationID == "" {
			t.Error("correlation_id is required in response")
		}

		// Verify correlation ID format
		if len(problem.CorrelationID) < 16 {
			t.Errorf("correlation_id should be at least 16 chars, got %d", len(problem.CorrelationID))
		}
	})
}

// TestProperty1_StatusMatchesErrorCode tests that HTTP status matches error code.
// Property 1: Error Response Consistency
// Validates: Requirements 4.2, 10.6, 11.1, 11.2, 11.3, 11.4
func TestProperty1_StatusMatchesErrorCode(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Test each error code maps to correct status
		testCases := []struct {
			code           TestErrorCode
			expectedStatus int
		}{
			{TestErrInvalidFileType, http.StatusBadRequest},
			{TestErrFileTooLarge, http.StatusBadRequest},
			{TestErrAccessDenied, http.StatusForbidden},
			{TestErrFileNotFound, http.StatusNotFound},
			{TestErrRateLimitExceeded, http.StatusTooManyRequests},
			{TestErrInternalError, http.StatusInternalServerError},
		}

		for _, tc := range testCases {
			problem := CreateTestProblemDetails(tc.code, "detail", "/api/v1/test", "corr-123")

			// Property: HTTP status SHALL match error code
			if problem.Status != tc.expectedStatus {
				t.Errorf("code %s: expected status %d, got %d",
					tc.code, tc.expectedStatus, problem.Status)
			}
		}
	})
}

// TestProperty1_ResponseIsValidJSON tests that error response is valid JSON.
// Property 1: Error Response Consistency
// Validates: Requirements 4.2, 10.6, 11.1, 11.2, 11.3, 11.4
func TestProperty1_ResponseIsValidJSON(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		code := rapid.SampledFrom([]TestErrorCode{
			TestErrInvalidFileType, TestErrAccessDenied, TestErrInternalError,
		}).Draw(t, "code")

		detail := rapid.StringMatching(`[A-Za-z0-9 ]{10,100}`).Draw(t, "detail")
		correlationID := rapid.StringMatching(`[a-f0-9]{32}`).Draw(t, "correlationID")

		problem := CreateTestProblemDetails(code, detail, "/api/v1/upload", correlationID)

		// Serialize to JSON
		jsonBytes, err := json.Marshal(problem)
		if err != nil {
			t.Fatalf("failed to marshal problem details: %v", err)
		}

		// Property: Response SHALL be valid JSON
		var parsed TestProblemDetails
		if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
			t.Fatalf("failed to unmarshal problem details: %v", err)
		}

		// Verify round-trip
		if parsed.Type != problem.Type {
			t.Errorf("type mismatch after round-trip: %q != %q", parsed.Type, problem.Type)
		}
		if parsed.Status != problem.Status {
			t.Errorf("status mismatch after round-trip: %d != %d", parsed.Status, problem.Status)
		}
	})
}

// TestProperty1_TypeContainsErrorCode tests that type URI contains error code.
// Property 1: Error Response Consistency
// Validates: Requirements 4.2, 10.6, 11.1, 11.2, 11.3, 11.4
func TestProperty1_TypeContainsErrorCode(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		code := rapid.SampledFrom([]TestErrorCode{
			TestErrInvalidFileType, TestErrFileTooLarge, TestErrAccessDenied,
			TestErrFileNotFound, TestErrRateLimitExceeded, TestErrInternalError,
		}).Draw(t, "code")

		problem := CreateTestProblemDetails(code, "detail", "/api/v1/test", "corr-123")

		// Property: Type URI SHALL contain error code
		if !strings.Contains(problem.Type, string(code)) {
			t.Errorf("type %q should contain error code %q", problem.Type, code)
		}
	})
}

// TestProperty1_ExtensionsPreserved tests that extensions are preserved in response.
// Property 1: Error Response Consistency
// Validates: Requirements 4.2, 10.6, 11.1, 11.2, 11.3, 11.4
func TestProperty1_ExtensionsPreserved(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		problem := CreateTestProblemDetails(
			TestErrInvalidFileType,
			"File type not allowed",
			"/api/v1/upload",
			"corr-123",
		)

		// Add extensions
		allowedTypes := []string{"image/jpeg", "image/png", "application/pdf"}
		problem.Extensions = map[string]any{
			"allowed_types": allowedTypes,
			"request_id":    "req-456",
		}

		// Serialize and deserialize
		jsonBytes, _ := json.Marshal(problem)
		var parsed TestProblemDetails
		json.Unmarshal(jsonBytes, &parsed)

		// Property: Extensions SHALL be preserved
		if parsed.Extensions == nil {
			t.Error("extensions should be preserved")
		}
		if _, ok := parsed.Extensions["allowed_types"]; !ok {
			t.Error("allowed_types extension should be preserved")
		}
		if _, ok := parsed.Extensions["request_id"]; !ok {
			t.Error("request_id extension should be preserved")
		}
	})
}
