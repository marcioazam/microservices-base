// Package caep provides CAEP event emission for IAM Policy Service.
package caep

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/auth-platform/iam-policy-service/internal/logging"
	"github.com/google/uuid"
)

// Emitter handles CAEP event emission.
type Emitter struct {
	enabled      bool
	transmitter  string
	serviceToken string
	issuer       string
	httpClient   *http.Client
	logger       *logging.Logger
}

// EmitterConfig holds configuration for the CAEP emitter.
type EmitterConfig struct {
	Enabled      bool
	Transmitter  string
	ServiceToken string
	Issuer       string
	Timeout      time.Duration
	Logger       *logging.Logger
}

// NewEmitter creates a new CAEP emitter.
func NewEmitter(cfg EmitterConfig) *Emitter {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	return &Emitter{
		enabled:      cfg.Enabled,
		transmitter:  cfg.Transmitter,
		serviceToken: cfg.ServiceToken,
		issuer:       cfg.Issuer,
		httpClient:   &http.Client{Timeout: timeout},
		logger:       cfg.Logger,
	}
}

// Event represents a CAEP event.
type Event struct {
	EventType      string                 `json:"event_type"`
	Subject        Subject                `json:"subject"`
	EventTimestamp int64                  `json:"event_timestamp"`
	ReasonAdmin    map[string]string      `json:"reason_admin,omitempty"`
	Extra          map[string]interface{} `json:"extra,omitempty"`
}

// Subject represents a CAEP subject identifier.
type Subject struct {
	Format string `json:"format"`
	Iss    string `json:"iss"`
	Sub    string `json:"sub"`
}

// EmitResult is the result of emitting an event.
type EmitResult struct {
	EventID string
	Error   error
}

// EmitAssuranceLevelChange emits an assurance-level-change event.
func (e *Emitter) EmitAssuranceLevelChange(ctx context.Context, userID, previousLevel, currentLevel, reason string) EmitResult {
	event := Event{
		EventType: "assurance-level-change",
		Subject: Subject{
			Format: "iss_sub",
			Iss:    e.issuer,
			Sub:    userID,
		},
		EventTimestamp: time.Now().Unix(),
		Extra: map[string]interface{}{
			"previous_level": previousLevel,
			"current_level":  currentLevel,
			"reason":         reason,
		},
	}

	return e.emit(ctx, event)
}

// EmitTokenClaimsChange emits a token-claims-change event.
func (e *Emitter) EmitTokenClaimsChange(ctx context.Context, userID string, changedClaims []string, reason string) EmitResult {
	event := Event{
		EventType: "token-claims-change",
		Subject: Subject{
			Format: "iss_sub",
			Iss:    e.issuer,
			Sub:    userID,
		},
		EventTimestamp: time.Now().Unix(),
		Extra: map[string]interface{}{
			"changed_claims": changedClaims,
			"reason":         reason,
		},
	}

	return e.emit(ctx, event)
}

// EmitRoleChange emits a token-claims-change event for role updates.
func (e *Emitter) EmitRoleChange(ctx context.Context, userID, previousRole, newRole string) EmitResult {
	return e.EmitTokenClaimsChange(ctx, userID, []string{"roles"}, fmt.Sprintf("Role changed from %s to %s", previousRole, newRole))
}

// EmitPermissionChange emits a token-claims-change event for permission updates.
func (e *Emitter) EmitPermissionChange(ctx context.Context, userID string, addedPermissions, removedPermissions []string) EmitResult {
	reason := fmt.Sprintf("Permissions updated: added=%v, removed=%v", addedPermissions, removedPermissions)
	return e.EmitTokenClaimsChange(ctx, userID, []string{"permissions"}, reason)
}

func (e *Emitter) emit(ctx context.Context, event Event) EmitResult {
	eventID := uuid.New().String()

	if !e.enabled {
		e.logDebug(ctx, "CAEP disabled, skipping event emission", event)
		return EmitResult{EventID: eventID}
	}

	body, err := json.Marshal(event)
	if err != nil {
		e.logError(ctx, "failed to marshal event", event, err)
		return EmitResult{Error: fmt.Errorf("failed to marshal event: %w", err)}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.transmitter+"/caep/emit", bytes.NewReader(body))
	if err != nil {
		e.logError(ctx, "failed to create request", event, err)
		return EmitResult{Error: fmt.Errorf("failed to create request: %w", err)}
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e.serviceToken)

	resp, err := e.httpClient.Do(req)
	if err != nil {
		e.logError(ctx, "failed to send event", event, err)
		return EmitResult{Error: fmt.Errorf("failed to send event: %w", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		err := fmt.Errorf("transmitter returned status %d", resp.StatusCode)
		e.logError(ctx, "CAEP transmitter returned error", event, err)
		return EmitResult{Error: err}
	}

	var result struct {
		EventID string `json:"event_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		result.EventID = eventID
	}

	e.logInfo(ctx, "CAEP event emitted", event, result.EventID)

	return EmitResult{EventID: result.EventID}
}

func (e *Emitter) logDebug(ctx context.Context, msg string, event Event) {
	if e.logger != nil {
		e.logger.Debug(ctx, msg,
			logging.String("event_type", event.EventType),
			logging.String("user_id", event.Subject.Sub))
	}
}

func (e *Emitter) logInfo(ctx context.Context, msg string, event Event, eventID string) {
	if e.logger != nil {
		e.logger.Info(ctx, msg,
			logging.String("event_type", event.EventType),
			logging.String("user_id", event.Subject.Sub),
			logging.String("event_id", eventID))
	}
}

func (e *Emitter) logError(ctx context.Context, msg string, event Event, err error) {
	if e.logger != nil {
		e.logger.Error(ctx, msg,
			logging.String("event_type", event.EventType),
			logging.String("user_id", event.Subject.Sub),
			logging.Error(err))
	}
}

// IsEnabled returns whether CAEP is enabled.
func (e *Emitter) IsEnabled() bool {
	return e.enabled
}
