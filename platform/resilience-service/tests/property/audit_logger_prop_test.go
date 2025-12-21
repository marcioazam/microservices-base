package property

import (
	"testing"
	"time"

	"github.com/auth-platform/platform/resilience-service/internal/domain"
	"github.com/auth-platform/platform/resilience-service/internal/infrastructure/observability"
	"pgregory.net/rapid"
)

// **Feature: resilience-microservice, Property 24: Audit Event Required Fields**
// **Validates: Requirements 9.3**
func TestProperty_AuditEventRequiredFields(t *testing.T) {
	t.Run("complete_event_has_required_fields", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			id := rapid.StringMatching(`[a-zA-Z0-9]{0,20}`).Draw(t, "id")
			eventType := rapid.StringMatching(`[a-zA-Z0-9]{0,20}`).Draw(t, "eventType")
			correlationID := rapid.StringMatching(`[a-zA-Z0-9]{0,20}`).Draw(t, "correlationID")

			event := domain.AuditEvent{
				ID:            id,
				Type:          eventType,
				Timestamp:     time.Now(),
				CorrelationID: correlationID,
				Action:        "test",
				Resource:      "test",
				Outcome:       "success",
			}

			hasRequired := observability.HasRequiredAuditFields(event)
			expectedHasRequired := id != "" && eventType != "" && correlationID != ""

			if hasRequired != expectedHasRequired {
				t.Fatalf("HasRequiredAuditFields mismatch: got %v, expected %v", hasRequired, expectedHasRequired)
			}
		})
	})

	t.Run("missing_id_detected", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			eventType := rapid.StringMatching(`[a-zA-Z][a-zA-Z0-9]{0,19}`).Draw(t, "eventType")
			correlationID := rapid.StringMatching(`[a-zA-Z][a-zA-Z0-9]{0,19}`).Draw(t, "correlationID")

			event := domain.AuditEvent{
				ID:            "",
				Type:          eventType,
				Timestamp:     time.Now(),
				CorrelationID: correlationID,
			}

			missing := observability.ValidateAuditEvent(event)
			foundID := false
			for _, field := range missing {
				if field == "id" {
					foundID = true
					break
				}
			}

			if !foundID {
				t.Fatal("expected 'id' to be in missing fields")
			}
		})
	})
}
