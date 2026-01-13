// Package service provides the authorization service for IAM Policy Service.
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/auth-platform/iam-policy-service/internal/caep"
	"github.com/auth-platform/iam-policy-service/internal/crypto"
	"github.com/auth-platform/iam-policy-service/internal/logging"
	"github.com/auth-platform/iam-policy-service/internal/policy"
	"github.com/auth-platform/iam-policy-service/internal/rbac"
	"github.com/google/uuid"
)

// AuthorizationRequest represents an authorization request.
type AuthorizationRequest struct {
	SubjectID   string
	SubjectType string
	ResourceID  string
	ResourceType string
	Action      string
	Context     map[string]interface{}
}

// AuthorizationResponse represents an authorization response.
type AuthorizationResponse struct {
	Allowed     bool
	Reason      string
	PolicyName  string
	EvaluatedAt time.Time
	FromCache   bool
	DecisionID  string
	Signature   []byte
	SigningKeyID crypto.KeyID
}

// BatchAuthorizationRequest represents a batch authorization request.
type BatchAuthorizationRequest struct {
	Requests []AuthorizationRequest
}

// BatchAuthorizationResponse represents a batch authorization response.
type BatchAuthorizationResponse struct {
	Responses []AuthorizationResponse
}

// AuthorizationService provides centralized authorization logic.
type AuthorizationService struct {
	engine    *policy.Engine
	hierarchy *rbac.RoleHierarchy
	emitter   *caep.Emitter
	signer    *crypto.DecisionSigner
	logger    *logging.Logger
}

// AuthorizationServiceConfig holds configuration for the authorization service.
type AuthorizationServiceConfig struct {
	Engine    *policy.Engine
	Hierarchy *rbac.RoleHierarchy
	Emitter   *caep.Emitter
	Signer    *crypto.DecisionSigner
	Logger    *logging.Logger
}

// NewAuthorizationService creates a new authorization service.
func NewAuthorizationService(cfg AuthorizationServiceConfig) *AuthorizationService {
	return &AuthorizationService{
		engine:    cfg.Engine,
		hierarchy: cfg.Hierarchy,
		emitter:   cfg.Emitter,
		signer:    cfg.Signer,
		logger:    cfg.Logger,
	}
}

// Authorize evaluates an authorization request.
func (s *AuthorizationService) Authorize(ctx context.Context, req AuthorizationRequest) (*AuthorizationResponse, error) {
	start := time.Now()
	decisionID := uuid.New().String()

	// Build policy input
	input := s.buildPolicyInput(req)

	// Evaluate policy
	result, err := s.engine.Evaluate(ctx, input)
	if err != nil {
		s.logAuthDecision(ctx, req, false, "error", err)
		return nil, fmt.Errorf("policy evaluation failed: %w", err)
	}

	response := &AuthorizationResponse{
		Allowed:     result.Allowed,
		PolicyName:  result.PolicyName,
		EvaluatedAt: start,
		FromCache:   result.FromCache,
		DecisionID:  decisionID,
	}

	if result.Allowed {
		response.Reason = "allowed by policy: " + result.PolicyName
	} else {
		response.Reason = "denied: no matching policy"
	}

	// Sign decision if signer is enabled
	if s.signer != nil && s.signer.IsEnabled() {
		signedDecision := crypto.NewSignedDecision(
			decisionID,
			req.SubjectID,
			req.ResourceID,
			req.Action,
			result.PolicyName,
			result.Allowed,
		)

		if err := s.signer.Sign(ctx, signedDecision); err != nil {
			s.logger.Warn(ctx, "failed to sign decision", logging.Error(err))
		} else {
			response.Signature = signedDecision.Signature
			response.SigningKeyID = signedDecision.KeyID
		}
	}

	s.logAuthDecision(ctx, req, result.Allowed, response.Reason, nil)

	return response, nil
}

// BatchAuthorize evaluates multiple authorization requests.
func (s *AuthorizationService) BatchAuthorize(ctx context.Context, req BatchAuthorizationRequest) (*BatchAuthorizationResponse, error) {
	responses := make([]AuthorizationResponse, len(req.Requests))

	for i, r := range req.Requests {
		resp, err := s.Authorize(ctx, r)
		if err != nil {
			responses[i] = AuthorizationResponse{
				Allowed:     false,
				Reason:      err.Error(),
				EvaluatedAt: time.Now(),
			}
			continue
		}
		responses[i] = *resp
	}

	return &BatchAuthorizationResponse{Responses: responses}, nil
}

// GetPermissions returns effective permissions for a subject.
func (s *AuthorizationService) GetPermissions(ctx context.Context, subjectID string, roles []string) ([]string, error) {
	permissionSet := make(map[string]bool)

	for _, roleID := range roles {
		perms := s.hierarchy.GetEffectivePermissions(roleID)
		for _, p := range perms {
			permissionSet[p] = true
		}
	}

	permissions := make([]string, 0, len(permissionSet))
	for p := range permissionSet {
		permissions = append(permissions, p)
	}

	if s.logger != nil {
		s.logger.Debug(ctx, "permissions retrieved",
			logging.String("subject_id", subjectID),
			logging.Int("permission_count", len(permissions)))
	}

	return permissions, nil
}

// GetRoles returns roles for a subject.
func (s *AuthorizationService) GetRoles(ctx context.Context, subjectID string) ([]string, error) {
	// In a real implementation, this would query a role store
	// For now, return empty slice
	return []string{}, nil
}

// ReloadPolicies reloads policies and invalidates cache.
func (s *AuthorizationService) ReloadPolicies(ctx context.Context) error {
	if s.logger != nil {
		s.logger.Info(ctx, "reloading policies")
	}
	return s.engine.ReloadPolicies(ctx)
}

func (s *AuthorizationService) buildPolicyInput(req AuthorizationRequest) map[string]interface{} {
	input := map[string]interface{}{
		"subject": map[string]interface{}{
			"id":   req.SubjectID,
			"type": req.SubjectType,
		},
		"resource": map[string]interface{}{
			"id":   req.ResourceID,
			"type": req.ResourceType,
		},
		"action": req.Action,
	}

	if req.Context != nil {
		input["context"] = req.Context
	}

	return input
}

func (s *AuthorizationService) logAuthDecision(ctx context.Context, req AuthorizationRequest, allowed bool, reason string, err error) {
	if s.logger == nil {
		return
	}

	fields := []logging.Field{
		logging.String("subject_id", req.SubjectID),
		logging.String("resource_type", req.ResourceType),
		logging.String("resource_id", req.ResourceID),
		logging.String("action", req.Action),
		logging.Bool("allowed", allowed),
		logging.String("reason", reason),
	}

	if err != nil {
		fields = append(fields, logging.Error(err))
		s.logger.Error(ctx, "authorization decision", fields...)
	} else {
		s.logger.Info(ctx, "authorization decision", fields...)
	}
}
