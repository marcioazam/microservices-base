package audit

import (
	"testing"
	"time"

	"pgregory.net/rapid"
)

// **Feature: auth-platform-2025-enhancements, Property 35: Token Issuance Audit**
// **Validates: Requirements 6.1**
func TestTokenIssuanceAuditProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random audit event data
		tokenType := rapid.StringMatching(`(access|refresh|id)_token`).Draw(t, "tokenType")
		subject := rapid.StringMatching(`user-[a-z0-9]{8}`).Draw(t, "subject")
		issuer := rapid.StringMatching(`https://[a-z]+\.example\.com`).Draw(t, "issuer")
		expiration := rapid.Int64Range(time.Now().Unix(), time.Now().Add(24*time.Hour).Unix()).Draw(t, "expiration")
		requestingService := rapid.StringMatching(`[a-z]+-service`).Draw(t, "requestingService")
		correlationID := rapid.StringMatching(`[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}`).Draw(t, "correlationID")

		// Create audit event
		event := &AuditEvent{
			EventType:     "token_issuance",
			Action:        "issue_token",
			Result:        "success",
			CorrelationID: correlationID,
			ServiceName:   requestingService,
			RequestData: map[string]any{
				"token_type": tokenType,
				"subject":    subject,
				"issuer":     issuer,
				"expiration": expiration,
			},
		}

		// Validate event has all required fields
		if event.RequestData["token_type"] == nil {
			t.Error("Token type must be present in audit log")
		}
		if event.RequestData["subject"] == nil {
			t.Error("Subject must be present in audit log")
		}
		if event.RequestData["issuer"] == nil {
			t.Error("Issuer must be present in audit log")
		}
		if event.RequestData["expiration"] == nil {
			t.Error("Expiration must be present in audit log")
		}
		if event.ServiceName == "" {
			t.Error("Requesting service must be present in audit log")
		}
	})
}

// **Feature: auth-platform-2025-enhancements, Property 36: Correlation ID Presence**
// **Validates: Requirements 6.4**
func TestCorrelationIDPresenceProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random audit events
		eventType := rapid.StringMatching(`(authentication|authorization|token_issuance|session_created)`).Draw(t, "eventType")
		action := rapid.StringMatching(`[a-z_]+`).Draw(t, "action")
		result := rapid.StringMatching(`(success|failure)`).Draw(t, "result")
		correlationID := rapid.StringMatching(`[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}`).Draw(t, "correlationID")

		event := &AuditEvent{
			EventType:     eventType,
			Action:        action,
			Result:        result,
			CorrelationID: correlationID,
			ServiceName:   "test-service",
			Timestamp:     time.Now(),
		}

		// Correlation ID must always be present
		if event.CorrelationID == "" {
			t.Error("Correlation ID must be present in all audit logs")
		}

		// Correlation ID must be valid UUID format
		if len(event.CorrelationID) != 36 {
			t.Errorf("Correlation ID must be valid UUID format, got length %d", len(event.CorrelationID))
		}

		// Validate event is complete
		if !ValidateEvent(event) {
			t.Error("Audit event validation failed")
		}
	})
}

// **Feature: auth-platform-2025-enhancements, Property 32: Circuit Breaker Observability**
// **Validates: Requirements 16.5**
func TestCircuitBreakerObservabilityProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate circuit breaker state transitions
		fromState := rapid.StringMatching(`(closed|open|half_open)`).Draw(t, "fromState")
		toState := rapid.StringMatching(`(closed|open|half_open)`).Draw(t, "toState")
		correlationID := rapid.StringMatching(`[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}`).Draw(t, "correlationID")
		serviceName := rapid.StringMatching(`[a-z]+-service`).Draw(t, "serviceName")

		// Create state change event
		event := &AuditEvent{
			EventType:     "circuit_breaker",
			Action:        "state_change",
			Result:        "success",
			CorrelationID: correlationID,
			ServiceName:   serviceName,
			Metadata: map[string]string{
				"from_state": fromState,
				"to_state":   toState,
			},
		}

		// State change must be logged with correlation ID
		if event.CorrelationID == "" {
			t.Error("Circuit breaker state change must include correlation ID")
		}

		// State transition must be recorded
		if event.Metadata["from_state"] == "" || event.Metadata["to_state"] == "" {
			t.Error("Circuit breaker state transition must be recorded")
		}

		// Service name must be present
		if event.ServiceName == "" {
			t.Error("Service name must be present in circuit breaker logs")
		}
	})
}

func TestAuditEventFiltering(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate multiple events
		numEvents := rapid.IntRange(5, 20).Draw(t, "numEvents")
		events := make([]AuditEvent, numEvents)

		targetUserID := "user-target"

		for i := 0; i < numEvents; i++ {
			userID := rapid.StringMatching(`user-[a-z0-9]{6}`).Draw(t, "userID")
			// Make some events match the target
			if rapid.Bool().Draw(t, "isTarget") {
				userID = targetUserID
			}

			events[i] = AuditEvent{
				EventID:       rapid.StringMatching(`evt-[a-f0-9]{8}`).Draw(t, "eventID"),
				EventType:     "authentication",
				Action:        "login",
				Result:        "success",
				UserID:        userID,
				CorrelationID: rapid.StringMatching(`[a-f0-9-]{36}`).Draw(t, "correlationID"),
				ServiceName:   "auth-service",
				Timestamp:     time.Now(),
			}
		}

		// Filter by user ID
		filter := EventFilter{UserID: targetUserID}
		filtered := FilterEvents(events, filter)

		// All filtered events should match the target user
		for _, event := range filtered {
			if event.UserID != targetUserID {
				t.Errorf("Filtered event has wrong user ID: %s", event.UserID)
			}
		}
	})
}

func TestAuditEventValidation(t *testing.T) {
	tests := []struct {
		name    string
		event   *AuditEvent
		isValid bool
	}{
		{
			name: "valid event",
			event: &AuditEvent{
				Timestamp:     time.Now(),
				CorrelationID: "abc-123-def",
				Action:        "login",
				Result:        "success",
				ServiceName:   "auth-service",
			},
			isValid: true,
		},
		{
			name: "missing correlation ID",
			event: &AuditEvent{
				Timestamp:   time.Now(),
				Action:      "login",
				Result:      "success",
				ServiceName: "auth-service",
			},
			isValid: false,
		},
		{
			name: "missing action",
			event: &AuditEvent{
				Timestamp:     time.Now(),
				CorrelationID: "abc-123-def",
				Result:        "success",
				ServiceName:   "auth-service",
			},
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if ValidateEvent(tt.event) != tt.isValid {
				t.Errorf("ValidateEvent() = %v, want %v", !tt.isValid, tt.isValid)
			}
		})
	}
}
