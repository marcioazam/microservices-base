// Package property contains property-based tests for IAM Policy Service.
package property

import (
	"testing"
	"time"

	"github.com/auth-platform/iam-policy-service/tests/testutil"
	"pgregory.net/rapid"
)

// TestCAEPEventStructureCompleteness validates Property 9.
// CAEP events must have complete structure.
func TestCAEPEventStructureCompleteness(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		userID := testutil.NonEmptyStringGen().Draw(t, "userID")
		eventType := rapid.SampledFrom([]string{
			"assurance-level-change",
			"token-claims-change",
		}).Draw(t, "eventType")

		event := testutil.NewMockCAEPEvent(eventType, userID)

		// Property: event_type must be non-empty
		if event.EventType == "" {
			t.Error("event_type should not be empty")
		}

		// Property: subject must have required fields
		if event.Subject.Format == "" {
			t.Error("subject.format should not be empty")
		}
		if event.Subject.Sub == "" {
			t.Error("subject.sub should not be empty")
		}

		// Property: timestamp must be valid
		if event.EventTimestamp <= 0 {
			t.Error("event_timestamp should be positive")
		}

		// Property: timestamp should be recent (within last hour)
		now := time.Now().Unix()
		if event.EventTimestamp > now+60 {
			t.Error("event_timestamp should not be in the future")
		}
	})
}

// TestCAEPAssuranceLevelChangeEvent validates assurance level change events.
func TestCAEPAssuranceLevelChangeEvent(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		userID := testutil.NonEmptyStringGen().Draw(t, "userID")
		previousLevel := rapid.SampledFrom([]string{"low", "medium", "high"}).Draw(t, "previousLevel")
		currentLevel := rapid.SampledFrom([]string{"low", "medium", "high"}).Draw(t, "currentLevel")

		event := testutil.NewMockAssuranceLevelChangeEvent(userID, previousLevel, currentLevel)

		// Property: event type must be correct
		if event.EventType != "assurance-level-change" {
			t.Errorf("expected event_type 'assurance-level-change', got %s", event.EventType)
		}

		// Property: extra must contain level information
		if event.Extra["previous_level"] != previousLevel {
			t.Errorf("previous_level mismatch: expected %s, got %v", previousLevel, event.Extra["previous_level"])
		}
		if event.Extra["current_level"] != currentLevel {
			t.Errorf("current_level mismatch: expected %s, got %v", currentLevel, event.Extra["current_level"])
		}
	})
}

// TestCAEPTokenClaimsChangeEvent validates token claims change events.
func TestCAEPTokenClaimsChangeEvent(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		userID := testutil.NonEmptyStringGen().Draw(t, "userID")
		changedClaims := rapid.SliceOfN(
			rapid.SampledFrom([]string{"roles", "permissions", "groups"}),
			1, 3,
		).Draw(t, "changedClaims")

		event := testutil.NewMockTokenClaimsChangeEvent(userID, changedClaims)

		// Property: event type must be correct
		if event.EventType != "token-claims-change" {
			t.Errorf("expected event_type 'token-claims-change', got %s", event.EventType)
		}

		// Property: extra must contain changed claims
		claims, ok := event.Extra["changed_claims"].([]string)
		if !ok {
			t.Error("changed_claims should be []string")
			return
		}

		if len(claims) != len(changedClaims) {
			t.Errorf("changed_claims length mismatch: expected %d, got %d", len(changedClaims), len(claims))
		}
	})
}

// TestCAEPSubjectFormat validates subject format.
func TestCAEPSubjectFormat(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		userID := testutil.NonEmptyStringGen().Draw(t, "userID")
		issuer := "https://auth.example.com"

		event := testutil.NewMockCAEPEventWithIssuer("test-event", userID, issuer)

		// Property: format must be iss_sub
		if event.Subject.Format != "iss_sub" {
			t.Errorf("expected format 'iss_sub', got %s", event.Subject.Format)
		}

		// Property: issuer must match
		if event.Subject.Iss != issuer {
			t.Errorf("issuer mismatch: expected %s, got %s", issuer, event.Subject.Iss)
		}

		// Property: subject must match user ID
		if event.Subject.Sub != userID {
			t.Errorf("subject mismatch: expected %s, got %s", userID, event.Subject.Sub)
		}
	})
}
