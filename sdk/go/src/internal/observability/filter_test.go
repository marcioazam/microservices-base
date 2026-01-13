// Package observability provides unit tests for sensitive data filtering.
package observability

import (
	"testing"
)

func TestFilterSensitiveData_BearerToken(t *testing.T) {
	input := "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"
	result := FilterSensitiveData(input)

	if result == input {
		t.Error("Bearer token should be redacted")
	}
	if result != "[REDACTED]" && !containsStr(result, "[REDACTED]") {
		t.Errorf("result should contain [REDACTED], got %s", result)
	}
}

func TestFilterSensitiveData_JWT(t *testing.T) {
	input := "token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U"
	result := FilterSensitiveData(input)

	if result == input {
		t.Error("JWT should be redacted")
	}
}

func TestFilterSensitiveData_Password(t *testing.T) {
	tests := []string{
		"password=secret123",
		"password: mysecret",
		"PASSWORD=test",
	}

	for _, input := range tests {
		result := FilterSensitiveData(input)
		if result == input {
			t.Errorf("password should be redacted in: %s", input)
		}
	}
}

func TestFilterSensitiveData_NoSensitiveData(t *testing.T) {
	input := "This is a normal log message without sensitive data"
	result := FilterSensitiveData(input)

	if result != input {
		t.Errorf("non-sensitive data should not be modified, got %s", result)
	}
}

func TestFilterSensitiveHeaders(t *testing.T) {
	headers := map[string][]string{
		"Authorization": {"Bearer token123"},
		"Content-Type":  {"application/json"},
		"X-Api-Key":     {"secret-key"},
		"Accept":        {"*/*"},
	}

	filtered := FilterSensitiveHeaders(headers)

	if filtered["Authorization"][0] != "[REDACTED]" {
		t.Error("Authorization header should be redacted")
	}
	if filtered["X-Api-Key"][0] != "[REDACTED]" {
		t.Error("X-Api-Key header should be redacted")
	}
	if filtered["Content-Type"][0] != "application/json" {
		t.Error("Content-Type should not be redacted")
	}
	if filtered["Accept"][0] != "*/*" {
		t.Error("Accept should not be redacted")
	}
}

func TestIsSensitiveHeader(t *testing.T) {
	tests := []struct {
		header    string
		sensitive bool
	}{
		{"Authorization", true},
		{"authorization", true},
		{"AUTHORIZATION", true},
		{"X-Api-Key", true},
		{"x-api-key", true},
		{"Cookie", true},
		{"DPoP", true},
		{"Content-Type", false},
		{"Accept", false},
		{"X-Request-Id", false},
	}

	for _, tt := range tests {
		if got := IsSensitiveHeader(tt.header); got != tt.sensitive {
			t.Errorf("IsSensitiveHeader(%s) = %v, want %v", tt.header, got, tt.sensitive)
		}
	}
}

func TestContainsSensitiveData(t *testing.T) {
	tests := []struct {
		input     string
		sensitive bool
	}{
		{"Bearer token123", true},
		{"eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxIn0.sig", true},
		{"password=secret", true},
		{"api_key=abc123", true},
		{"normal text", false},
		{"user@example.com", false},
	}

	for _, tt := range tests {
		if got := ContainsSensitiveData(tt.input); got != tt.sensitive {
			t.Errorf("ContainsSensitiveData(%s) = %v, want %v", tt.input, got, tt.sensitive)
		}
	}
}

func TestRedactValue(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"ab", "[REDACTED]"},
		{"abc", "[REDACTED]"},
		{"abcd", "[REDACTED]"},
		{"abcde", "ab...de"},
		{"secrettoken", "se...en"},
	}

	for _, tt := range tests {
		if got := RedactValue(tt.input); got != tt.expected {
			t.Errorf("RedactValue(%s) = %s, want %s", tt.input, got, tt.expected)
		}
	}
}

func TestSafeMap(t *testing.T) {
	m := map[string]string{
		"username": "john",
		"password": "secret123",
		"email":    "john@example.com",
		"token":    "abc123",
	}

	result := SafeMap(m, "password", "token")

	if result["username"] != "john" {
		t.Error("username should not be redacted")
	}
	if result["password"] != "[REDACTED]" {
		t.Error("password should be redacted")
	}
	if result["email"] != "john@example.com" {
		t.Error("email should not be redacted")
	}
	if result["token"] != "[REDACTED]" {
		t.Error("token should be redacted")
	}
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
