package audit

import (
	"testing"
	"time"

	"github.com/auth-platform/platform/resilience-service/internal/domain"
	"github.com/auth-platform/platform/resilience-service/internal/testutil"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// **Feature: resilience-microservice, Property 24: Audit Event Required Fields**
// **Validates: Requirements 9.3**
func TestProperty_AuditEventRequiredFields(t *testing.T) {
	params := testutil.DefaultTestParameters()
	props := gopter.NewProperties(params)

	props.Property("complete_event_has_required_fields", prop.ForAll(
		func(id, eventType, correlationID string) bool {
			event := domain.AuditEvent{
				ID:            id,
				Type:          eventType,
				Timestamp:     time.Now(),
				CorrelationID: correlationID,
				Action:        "test",
				Resource:      "test",
				Outcome:       "success",
			}

			// If all fields are non-empty, should pass validation
			if id != "" && eventType != "" && correlationID != "" {
				return HasRequiredFields(event)
			}

			// If any required field is empty, should fail validation
			return !HasRequiredFields(event)
		},
		gen.AlphaString(),
		gen.AlphaString(),
		gen.AlphaString(),
	))

	props.Property("missing_id_detected", prop.ForAll(
		func(eventType, correlationID string) bool {
			event := domain.AuditEvent{
				ID:            "", // Missing
				Type:          eventType,
				Timestamp:     time.Now(),
				CorrelationID: correlationID,
			}

			missing := ValidateEvent(event)
			for _, field := range missing {
				if field == "id" {
					return true
				}
			}
			return false
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
	))

	props.Property("missing_timestamp_detected", prop.ForAll(
		func(id, eventType, correlationID string) bool {
			event := domain.AuditEvent{
				ID:            id,
				Type:          eventType,
				Timestamp:     time.Time{}, // Zero value = missing
				CorrelationID: correlationID,
			}

			missing := ValidateEvent(event)
			for _, field := range missing {
				if field == "timestamp" {
					return true
				}
			}
			return false
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
	))

	props.Property("missing_correlation_id_detected", prop.ForAll(
		func(id, eventType string) bool {
			event := domain.AuditEvent{
				ID:            id,
				Type:          eventType,
				Timestamp:     time.Now(),
				CorrelationID: "", // Missing
			}

			missing := ValidateEvent(event)
			for _, field := range missing {
				if field == "correlation_id" {
					return true
				}
			}
			return false
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
	))

	props.Property("spiffe_id_optional", prop.ForAll(
		func(id, eventType, correlationID string) bool {
			// Event without SPIFFE ID should still be valid
			event := domain.AuditEvent{
				ID:            id,
				Type:          eventType,
				Timestamp:     time.Now(),
				CorrelationID: correlationID,
				SpiffeID:      "", // Optional
			}

			if id == "" || eventType == "" || correlationID == "" {
				return true // Skip invalid base events
			}

			return HasRequiredFields(event)
		},
		gen.AlphaString(),
		gen.AlphaString(),
		gen.AlphaString(),
	))

	props.TestingRun(t)
}
