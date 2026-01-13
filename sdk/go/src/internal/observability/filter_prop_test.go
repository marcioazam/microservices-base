// Package observability provides property-based tests for sensitive data filtering.
package observability

import (
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// Property 25: Sensitive Data Filtering
func TestProperty_SensitiveDataFiltering(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		sensitivePatterns := []string{"Bearer ", "DPoP ", "password=", "secret="}
		pattern := rapid.SampledFrom(sensitivePatterns).Draw(t, "pattern")
		prefix := rapid.StringMatching(`[a-z ]{0,20}`).Draw(t, "prefix")
		suffix := rapid.StringMatching(`[a-z0-9]{5,20}`).Draw(t, "suffix")

		input := prefix + pattern + suffix
		filtered := FilterSensitiveData(input)

		// The filtered output should contain [REDACTED]
		if !strings.Contains(filtered, "[REDACTED]") {
			t.Fatalf("filtered output should contain [REDACTED] for input with %s", pattern)
		}
	})
}

// Property: Non-sensitive data is preserved
func TestProperty_NonSensitiveDataPreserved(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate strings that don't match sensitive patterns
		input := rapid.StringMatching(`[a-z]{10,30}`).Draw(t, "input")

		filtered := FilterSensitiveData(input)

		if filtered != input {
			t.Fatalf("non-sensitive data should be preserved: got %s, want %s", filtered, input)
		}
	})
}

// Property: JWT tokens are always redacted
func TestProperty_JWTTokensRedacted(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate JWT-like structure
		header := rapid.StringMatching(`eyJ[A-Za-z0-9_-]{10,20}`).Draw(t, "header")
		payload := rapid.StringMatching(`eyJ[A-Za-z0-9_-]{10,30}`).Draw(t, "payload")
		signature := rapid.StringMatching(`[A-Za-z0-9_-]{20,40}`).Draw(t, "signature")

		jwt := header + "." + payload + "." + signature
		filtered := FilterSensitiveData(jwt)

		if filtered == jwt {
			t.Fatal("JWT should be redacted")
		}
	})
}

// Property: Sensitive headers are identified correctly
func TestProperty_SensitiveHeadersIdentified(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		sensitiveHeaders := []string{
			"authorization", "Authorization", "AUTHORIZATION",
			"x-api-key", "X-Api-Key", "X-API-KEY",
			"cookie", "Cookie", "COOKIE",
			"dpop", "DPoP", "DPOP",
		}
		header := rapid.SampledFrom(sensitiveHeaders).Draw(t, "header")

		if !IsSensitiveHeader(header) {
			t.Fatalf("header %s should be identified as sensitive", header)
		}
	})
}

// Property: Non-sensitive headers are not flagged
func TestProperty_NonSensitiveHeadersNotFlagged(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		nonSensitiveHeaders := []string{
			"content-type", "Content-Type",
			"accept", "Accept",
			"x-request-id", "X-Request-Id",
			"user-agent", "User-Agent",
		}
		header := rapid.SampledFrom(nonSensitiveHeaders).Draw(t, "header")

		if IsSensitiveHeader(header) {
			t.Fatalf("header %s should not be identified as sensitive", header)
		}
	})
}

// Property: FilterSensitiveHeaders preserves non-sensitive headers
func TestProperty_FilterHeadersPreservesNonSensitive(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		headerName := rapid.SampledFrom([]string{"Content-Type", "Accept", "X-Request-Id"}).Draw(t, "name")
		headerValue := rapid.StringMatching(`[a-z/]{5,20}`).Draw(t, "value")

		headers := map[string][]string{
			headerName: {headerValue},
		}

		filtered := FilterSensitiveHeaders(headers)

		if filtered[headerName][0] != headerValue {
			t.Fatalf("non-sensitive header %s should be preserved", headerName)
		}
	})
}

// Property: FilterSensitiveHeaders redacts sensitive headers
func TestProperty_FilterHeadersRedactsSensitive(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		headerName := rapid.SampledFrom([]string{"Authorization", "X-Api-Key", "Cookie"}).Draw(t, "name")
		headerValue := rapid.StringMatching(`[A-Za-z0-9]{10,30}`).Draw(t, "value")

		headers := map[string][]string{
			headerName: {headerValue},
		}

		filtered := FilterSensitiveHeaders(headers)

		if filtered[headerName][0] != "[REDACTED]" {
			t.Fatalf("sensitive header %s should be redacted", headerName)
		}
	})
}

// Property: RedactValue handles all lengths correctly
func TestProperty_RedactValueHandlesAllLengths(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		length := rapid.IntRange(1, 50).Draw(t, "length")
		value := rapid.StringN(length, length, length).Draw(t, "value")

		result := RedactValue(value)

		if length <= 4 {
			if result != "[REDACTED]" {
				t.Fatalf("short value should be fully redacted")
			}
		} else {
			if !strings.Contains(result, "...") {
				t.Fatalf("longer value should contain ellipsis")
			}
		}
	})
}

// Property: SafeMap redacts specified keys
func TestProperty_SafeMapRedactsSpecifiedKeys(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		key := rapid.StringMatching(`[a-z]{3,10}`).Draw(t, "key")
		value := rapid.StringMatching(`[a-z0-9]{5,20}`).Draw(t, "value")

		m := map[string]string{key: value}
		result := SafeMap(m, key)

		if result[key] != "[REDACTED]" {
			t.Fatalf("specified key %s should be redacted", key)
		}
	})
}

// Property: SafeMap preserves non-specified keys
func TestProperty_SafeMapPreservesNonSpecified(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		key := rapid.StringMatching(`[a-z]{3,10}`).Draw(t, "key")
		value := rapid.StringMatching(`[a-z0-9]{5,20}`).Draw(t, "value")
		sensitiveKey := rapid.StringMatching(`[A-Z]{3,10}`).Draw(t, "sensitiveKey")

		m := map[string]string{key: value}
		result := SafeMap(m, sensitiveKey)

		if result[key] != value {
			t.Fatalf("non-specified key %s should be preserved", key)
		}
	})
}

// Property: ContainsSensitiveData is consistent with FilterSensitiveData
func TestProperty_ContainsSensitiveConsistentWithFilter(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		input := rapid.StringMatching(`[a-z ]{10,50}`).Draw(t, "input")

		containsSensitive := ContainsSensitiveData(input)
		filtered := FilterSensitiveData(input)

		if containsSensitive && filtered == input {
			t.Fatal("if ContainsSensitiveData returns true, FilterSensitiveData should modify input")
		}
	})
}
