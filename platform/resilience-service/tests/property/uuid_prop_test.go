// Package property contains property-based tests for the resilience service.
package property

import (
	"strings"
	"testing"
	"time"

	"github.com/auth-platform/platform/resilience-service/internal/domain"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// **Feature: resilience-service-modernization, Property 1: UUID v7 Format Compliance**
// **Validates: Requirements 3.1, 3.2, 3.3, 3.5**
func TestUUIDv7FormatCompliance(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("UUID v7 has correct format", prop.ForAll(
		func(_ int) bool {
			id := domain.GenerateEventID()

			// Test length is 36 characters
			if len(id) != 36 {
				t.Logf("Expected 36 chars, got %d", len(id))
				return false
			}

			// Test hyphens at correct positions (8, 13, 18, 23)
			if id[8] != '-' || id[13] != '-' || id[18] != '-' || id[23] != '-' {
				t.Logf("Hyphens not at correct positions: %s", id)
				return false
			}

			// Test version nibble is '7' (position 14)
			if id[14] != '7' {
				t.Logf("Version nibble is not '7': %c", id[14])
				return false
			}

			// Test variant bits (position 19) are '8', '9', 'a', or 'b'
			variant := id[19]
			if variant != '8' && variant != '9' && variant != 'a' && variant != 'b' {
				t.Logf("Invalid variant: %c", variant)
				return false
			}

			return true
		},
		gen.Int(),
	))

	properties.Property("UUID v7 timestamp is accurate", prop.ForAll(
		func(_ int) bool {
			before := time.Now()
			id := domain.GenerateEventID()
			after := time.Now()

			ts, err := domain.ParseUUIDv7Timestamp(id)
			if err != nil {
				t.Logf("Failed to parse timestamp: %v", err)
				return false
			}

			// Timestamp should be within 1 second of generation time
			if ts.Before(before.Add(-time.Second)) || ts.After(after.Add(time.Second)) {
				t.Logf("Timestamp out of range: %v (expected between %v and %v)", ts, before, after)
				return false
			}

			return true
		},
		gen.Int(),
	))

	properties.Property("UUID v7 is valid", prop.ForAll(
		func(_ int) bool {
			id := domain.GenerateEventID()
			return domain.IsValidUUIDv7(id)
		},
		gen.Int(),
	))

	properties.TestingRun(t)
}

// **Feature: resilience-service-modernization, Property 2: UUID v7 Chronological Ordering**
// **Validates: Requirements 3.4**
func TestUUIDv7ChronologicalOrdering(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("UUID v7 lexicographic order matches chronological order", prop.ForAll(
		func(_ int) bool {
			// Generate first UUID
			id1 := domain.GenerateEventID()

			// Wait at least 1ms to ensure different timestamp
			time.Sleep(2 * time.Millisecond)

			// Generate second UUID
			id2 := domain.GenerateEventID()

			// Lexicographic comparison should show id1 < id2
			if strings.Compare(id1, id2) >= 0 {
				t.Logf("Lexicographic order incorrect: %s >= %s", id1, id2)
				return false
			}

			return true
		},
		gen.Int(),
	))

	properties.TestingRun(t)
}

func TestGenerateEventID_Basic(t *testing.T) {
	id := domain.GenerateEventID()

	if len(id) != 36 {
		t.Errorf("Expected 36 characters, got %d", len(id))
	}

	if !domain.IsValidUUIDv7(id) {
		t.Errorf("Generated ID is not a valid UUID v7: %s", id)
	}
}

func TestParseUUIDv7Timestamp_InvalidInputs(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"too short", "123"},
		{"no hyphens", "12345678123412341234123456789012"},
		{"wrong hyphen positions", "1234567-81234-1234-1234-123456789012"},
		{"invalid hex", "gggggggg-gggg-7ggg-8ggg-gggggggggggg"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := domain.ParseUUIDv7Timestamp(tt.input)
			if err == nil {
				t.Errorf("Expected error for input %q", tt.input)
			}
		})
	}
}

func TestIsValidUUIDv7_InvalidVersion(t *testing.T) {
	// UUID v4 format (version 4, not 7)
	uuidV4 := "550e8400-e29b-41d4-a716-446655440000"
	if domain.IsValidUUIDv7(uuidV4) {
		t.Error("UUID v4 should not be valid as UUID v7")
	}
}
