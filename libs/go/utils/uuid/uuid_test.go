package uuid

import (
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/prop"
)

// **Feature: resilience-lib-extraction, Property 1: UUID v7 Format Compliance**
// **Validates: Requirements 1.1, 1.4**
func TestUUIDv7FormatCompliance(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("generated UUID v7 is RFC 9562 compliant", prop.ForAll(
		func() bool {
			id := GenerateEventID()

			// Check length is 36 characters
			if len(id) != 36 {
				return false
			}

			// Check hyphens at correct positions
			if id[8] != '-' || id[13] != '-' || id[18] != '-' || id[23] != '-' {
				return false
			}

			// Check version nibble is 7
			if id[14] != '7' {
				return false
			}

			// Check variant bits (position 19 should be 8, 9, a, or b)
			variant := id[19]
			if variant != '8' && variant != '9' && variant != 'a' && variant != 'b' {
				return false
			}

			// Validate using our function
			return IsValidUUIDv7(id)
		},
	))

	properties.TestingRun(t)
}

// **Feature: resilience-lib-extraction, Property 2: UUID v7 Timestamp Round-Trip**
// **Validates: Requirements 1.2, 1.5**
func TestUUIDv7TimestampRoundTrip(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("timestamp round-trip within 1ms tolerance", prop.ForAll(
		func() bool {
			before := time.Now()
			id := GenerateEventID()
			after := time.Now()

			parsed, err := ParseUUIDv7Timestamp(id)
			if err != nil {
				return false
			}

			// Parsed timestamp should be between before and after (with 1ms tolerance)
			beforeMs := before.UnixMilli()
			afterMs := after.UnixMilli()
			parsedMs := parsed.UnixMilli()

			return parsedMs >= beforeMs-1 && parsedMs <= afterMs+1
		},
	))

	properties.TestingRun(t)
}

// **Feature: resilience-lib-extraction, Property 3: UUID v7 Validation Consistency**
// **Validates: Requirements 1.3**
func TestUUIDv7ValidationConsistency(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("IsValidUUIDv7 returns true for generated UUIDs", prop.ForAll(
		func() bool {
			id := GenerateEventID()
			return IsValidUUIDv7(id)
		},
	))

	properties.TestingRun(t)
}

func TestIsValidUUIDv7_InvalidCases(t *testing.T) {
	invalidCases := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"too short", "12345678-1234-7123-8123-12345678901"},
		{"too long", "12345678-1234-7123-8123-1234567890123"},
		{"wrong version", "12345678-1234-4123-8123-123456789012"},
		{"wrong variant", "12345678-1234-7123-0123-123456789012"},
		{"missing hyphens", "123456781234712381231234567890123"},
		{"invalid hex", "12345678-1234-7123-8123-12345678901g"},
	}

	for _, tc := range invalidCases {
		t.Run(tc.name, func(t *testing.T) {
			if IsValidUUIDv7(tc.input) {
				t.Errorf("expected IsValidUUIDv7(%q) to be false", tc.input)
			}
		})
	}
}

func TestParseUUIDv7Timestamp_Errors(t *testing.T) {
	_, err := ParseUUIDv7Timestamp("invalid")
	if err == nil {
		t.Error("expected error for invalid UUID")
	}

	// Valid UUID v4 should fail
	_, err = ParseUUIDv7Timestamp("12345678-1234-4123-8123-123456789012")
	if err != ErrInvalidUUIDv7 {
		t.Errorf("expected ErrInvalidUUIDv7, got %v", err)
	}
}

func TestGenerateEventIDWithTime(t *testing.T) {
	testTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	id := GenerateEventIDWithTime(testTime)

	if !IsValidUUIDv7(id) {
		t.Errorf("generated UUID is not valid: %s", id)
	}

	parsed, err := ParseUUIDv7Timestamp(id)
	if err != nil {
		t.Fatalf("failed to parse timestamp: %v", err)
	}

	if parsed.UnixMilli() != testTime.UnixMilli() {
		t.Errorf("timestamp mismatch: got %v, want %v", parsed.UnixMilli(), testTime.UnixMilli())
	}
}
