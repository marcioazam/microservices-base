package crypto

import (
	"context"
	"fmt"

	"github.com/auth-platform/iam-policy-service/internal/logging"
)

// DecisionSigner signs and verifies authorization decisions.
type DecisionSigner struct {
	cryptoClient *Client
	enabled      bool
	logger       *logging.Logger
}

// NewDecisionSigner creates a new decision signer.
func NewDecisionSigner(client *Client, logger *logging.Logger) *DecisionSigner {
	enabled := client != nil && client.IsDecisionSigningEnabled()

	return &DecisionSigner{
		cryptoClient: client,
		enabled:      enabled,
		logger:       logger,
	}
}

// IsEnabled returns true if decision signing is enabled.
func (s *DecisionSigner) IsEnabled() bool {
	return s.enabled
}

// Sign signs an authorization decision using ECDSA P-256.
func (s *DecisionSigner) Sign(ctx context.Context, decision *SignedDecision) error {
	if !s.enabled {
		return nil
	}

	if decision == nil {
		return NewCryptoError(ErrCodeInvalidInput, "decision cannot be nil", getCorrelationID(ctx), nil)
	}

	if !decision.HasAllRequiredFields() {
		return NewCryptoError(ErrCodeInvalidInput, "decision missing required fields", getCorrelationID(ctx), nil)
	}

	// Build canonical payload
	payload := decision.BuildSignaturePayload()

	// Sign via crypto-service
	result, err := s.cryptoClient.Sign(ctx, payload)
	if err != nil {
		if IsServiceUnavailable(err) {
			s.logWarn(ctx, "crypto service unavailable, skipping signature")
			return nil
		}
		return fmt.Errorf("signing failed: %w", err)
	}

	// Set signature and key ID on decision
	decision.Signature = result.Signature
	decision.KeyID = result.KeyID

	s.logDebug(ctx, "decision signed", decision.DecisionID)

	return nil
}

// Verify verifies a signed decision.
func (s *DecisionSigner) Verify(ctx context.Context, decision *SignedDecision) (bool, error) {
	if !s.enabled {
		return true, nil // If signing disabled, consider all valid
	}

	if decision == nil {
		return false, NewCryptoError(ErrCodeInvalidInput, "decision cannot be nil", getCorrelationID(ctx), nil)
	}

	if !decision.IsSigned() {
		return false, NewCryptoError(ErrCodeSignatureInvalid, "decision has no signature", getCorrelationID(ctx), nil)
	}

	// Build canonical payload
	payload := decision.BuildSignaturePayload()

	// Verify via crypto-service
	valid, err := s.cryptoClient.Verify(ctx, payload, decision.Signature, decision.KeyID)
	if err != nil {
		return false, fmt.Errorf("verification failed: %w", err)
	}

	if !valid {
		return false, NewCryptoError(ErrCodeSignatureInvalid, "signature verification failed", getCorrelationID(ctx), nil)
	}

	s.logDebug(ctx, "decision signature verified", decision.DecisionID)

	return true, nil
}

// SignIfEnabled signs the decision only if signing is enabled.
// Returns the decision unchanged if signing is disabled.
func (s *DecisionSigner) SignIfEnabled(ctx context.Context, decision *SignedDecision) (*SignedDecision, error) {
	if !s.enabled {
		return decision, nil
	}

	if err := s.Sign(ctx, decision); err != nil {
		return decision, err
	}

	return decision, nil
}

func (s *DecisionSigner) logDebug(ctx context.Context, msg, decisionID string) {
	if s.logger != nil {
		s.logger.Debug(ctx, msg, logging.String("decision_id", decisionID))
	}
}

func (s *DecisionSigner) logWarn(ctx context.Context, msg string) {
	if s.logger != nil {
		s.logger.Warn(ctx, msg)
	}
}
