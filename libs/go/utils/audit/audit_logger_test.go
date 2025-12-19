package audit

import (
	"testing"
	"time"

	"pgregory.net/rapid"
)

// **Feature: auth-microservices-platform, Property 22: Audit Log Completeness**
// **Validates: Requirements 8.1, 8.2**
func TestAuditLogCompleteness(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random audit event data
		userID := rapid.String().Draw(t, "userID")
		action := rapid.SampledFrom([]string{"login", "logout", "token_issue", "mfa_verify", "authorize"}).Draw(t, "action")
		result := rapid.SampledFrom([]string{"success", "failure"}).Draw(t, "result")
		correlationID := rapid.StringMatching(`[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}`).Draw(t, "correlationID")

		event := &AuditEvent{
			Timestamp:     time.Now().UTC(),
			CorrelationID: correlationID,
			Action:        action,
			Result:        result,
			UserID:        userID,
			ServiceName:   "test-service",
		}

		// Verify all required fields are present
		if event.Timestamp.IsZero() {
			t.Fatal("Timestamp must be present")
		}
		if event.CorrelationID == "" {
			t.Fatal("CorrelationID must be present")
		}
		if event.Action == "" {
			t.Fatal("Action must be present")
		}
		if event.Result == "" {
			t.Fatal("Result must be present")
		}

		// Validate event
		if !ValidateEvent(event) {
			t.Fatal("Event validation failed")
		}
	})
}

// **Feature: auth-microservices-platform, Property 23: Audit Log Query Filtering**
// **Validates: Requirements 8.4**
func TestAuditLogQueryFiltering(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a set of events
		numEvents := rapid.IntRange(1, 100).Draw(t, "numEvents")
		events := make([]AuditEvent, numEvents)

		targetUserID := "target-user-123"
		targetAction := "login"

		for i := 0; i < numEvents; i++ {
			userID := rapid.SampledFrom([]string{targetUserID, "other-user-1", "other-user-2"}).Draw(t, "userID")
			action := rapid.SampledFrom([]string{targetAction, "logout", "token_issue"}).Draw(t, "action")

			events[i] = AuditEvent{
				EventID:       rapid.String().Draw(t, "eventID"),
				Timestamp:     time.Now().UTC().Add(time.Duration(-i) * time.Hour),
				CorrelationID: rapid.String().Draw(t, "correlationID"),
				Action:        action,
				Result:        "success",
				UserID:        userID,
				ServiceName:   "test-service",
			}
		}

		// Filter by user ID
		userFilter := EventFilter{UserID: targetUserID}
		filteredByUser := FilterEvents(events, userFilter)

		for _, event := range filteredByUser {
			if event.UserID != targetUserID {
				t.Fatalf("Filter by user failed: expected %s, got %s", targetUserID, event.UserID)
			}
		}

		// Filter by action
		actionFilter := EventFilter{Action: targetAction}
		filteredByAction := FilterEvents(events, actionFilter)

		for _, event := range filteredByAction {
			if event.Action != targetAction {
				t.Fatalf("Filter by action failed: expected %s, got %s", targetAction, event.Action)
			}
		}

		// Filter by both
		combinedFilter := EventFilter{UserID: targetUserID, Action: targetAction}
		filteredByCombined := FilterEvents(events, combinedFilter)

		for _, event := range filteredByCombined {
			if event.UserID != targetUserID || event.Action != targetAction {
				t.Fatal("Combined filter failed")
			}
		}
	})
}

func TestValidateEvent(t *testing.T) {
	tests := []struct {
		name     string
		event    *AuditEvent
		expected bool
	}{
		{
			name: "valid event",
			event: &AuditEvent{
				Timestamp:     time.Now(),
				CorrelationID: "test-correlation",
				Action:        "login",
				Result:        "success",
				ServiceName:   "test-service",
			},
			expected: true,
		},
		{
			name: "missing timestamp",
			event: &AuditEvent{
				CorrelationID: "test-correlation",
				Action:        "login",
				Result:        "success",
				ServiceName:   "test-service",
			},
			expected: false,
		},
		{
			name: "missing correlation ID",
			event: &AuditEvent{
				Timestamp:   time.Now(),
				Action:      "login",
				Result:      "success",
				ServiceName: "test-service",
			},
			expected: false,
		},
		{
			name: "missing action",
			event: &AuditEvent{
				Timestamp:     time.Now(),
				CorrelationID: "test-correlation",
				Result:        "success",
				ServiceName:   "test-service",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateEvent(tt.event)
			if result != tt.expected {
				t.Errorf("ValidateEvent() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestFilterEvents_TimeRange(t *testing.T) {
	now := time.Now().UTC()
	events := []AuditEvent{
		{Timestamp: now.Add(-24 * time.Hour), Action: "login", Result: "success", CorrelationID: "1", ServiceName: "test"},
		{Timestamp: now.Add(-12 * time.Hour), Action: "login", Result: "success", CorrelationID: "2", ServiceName: "test"},
		{Timestamp: now.Add(-1 * time.Hour), Action: "login", Result: "success", CorrelationID: "3", ServiceName: "test"},
	}

	startTime := now.Add(-18 * time.Hour)
	endTime := now.Add(-6 * time.Hour)

	filter := EventFilter{
		StartTime: &startTime,
		EndTime:   &endTime,
	}

	filtered := FilterEvents(events, filter)

	if len(filtered) != 1 {
		t.Errorf("Expected 1 event in time range, got %d", len(filtered))
	}
}
