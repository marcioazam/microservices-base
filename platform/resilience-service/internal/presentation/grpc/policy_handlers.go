// Package grpc provides policy gRPC handlers with Option/Result type handling.
package grpc

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/auth-platform/platform/resilience-service/internal/application/services"
	"github.com/auth-platform/platform/resilience-service/internal/domain/entities"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// PolicyRequest represents a policy request.
type PolicyRequest struct {
	Name string
}

// PolicyResponse represents a policy response.
type PolicyResponse struct {
	Name           string
	Version        int32
	CircuitBreaker *CircuitBreakerConfigResponse
	Retry          *RetryConfigResponse
	Timeout        *TimeoutConfigResponse
	RateLimit      *RateLimitConfigResponse
	Bulkhead       *BulkheadConfigResponse
	CreatedAt      *timestamppb.Timestamp
	UpdatedAt      *timestamppb.Timestamp
}

// CircuitBreakerConfigResponse represents circuit breaker config in response.
type CircuitBreakerConfigResponse struct {
	FailureThreshold int32
	SuccessThreshold int32
	Timeout          *durationpb.Duration
	ProbeCount       int32
}

// RetryConfigResponse represents retry config in response.
type RetryConfigResponse struct {
	MaxAttempts   int32
	BaseDelay     *durationpb.Duration
	MaxDelay      *durationpb.Duration
	Multiplier    float64
	JitterPercent float64
}

// TimeoutConfigResponse represents timeout config in response.
type TimeoutConfigResponse struct {
	Default *durationpb.Duration
	Max     *durationpb.Duration
}

// RateLimitConfigResponse represents rate limit config in response.
type RateLimitConfigResponse struct {
	Algorithm string
	Limit     int32
	Window    *durationpb.Duration
	BurstSize int32
}

// BulkheadConfigResponse represents bulkhead config in response.
type BulkheadConfigResponse struct {
	MaxConcurrent int32
	MaxQueue      int32
	QueueTimeout  *durationpb.Duration
}

// PolicyHandlers handles policy gRPC requests.
type PolicyHandlers struct {
	policyService *services.PolicyService
	logger        *slog.Logger
}

// NewPolicyHandlers creates new policy handlers.
func NewPolicyHandlers(policyService *services.PolicyService, logger *slog.Logger) *PolicyHandlers {
	return &PolicyHandlers{
		policyService: policyService,
		logger:        logger,
	}
}

// GetPolicy handles GetPolicy requests with Option type handling.
func (h *PolicyHandlers) GetPolicy(ctx context.Context, req *PolicyRequest) (*PolicyResponse, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "policy name is required")
	}

	h.logger.DebugContext(ctx, "handling GetPolicy request",
		slog.String("policy_name", req.Name))

	// Get policy - returns Option[*Policy]
	opt := h.policyService.GetPolicy(ctx, req.Name)

	// Handle Option type - if None, return NotFound
	if !opt.IsSome() {
		return nil, status.Errorf(codes.NotFound, "policy '%s' not found", req.Name)
	}

	policy := opt.Unwrap()
	return h.policyToResponse(policy), nil
}

// CreatePolicy handles CreatePolicy requests with Result type handling.
func (h *PolicyHandlers) CreatePolicy(ctx context.Context, req *PolicyRequest) (*PolicyResponse, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "policy name is required")
	}

	h.logger.InfoContext(ctx, "handling CreatePolicy request",
		slog.String("policy_name", req.Name))

	// Create policy - returns Result[*Policy]
	result := h.policyService.CreatePolicy(ctx, req.Name)

	// Handle Result type - map errors to gRPC status codes
	if result.IsErr() {
		err := result.UnwrapErr()
		return nil, h.mapErrorToStatus(err)
	}

	policy := result.Unwrap()
	return h.policyToResponse(policy), nil
}

// UpdatePolicy handles UpdatePolicy requests with Result type handling.
func (h *PolicyHandlers) UpdatePolicy(ctx context.Context, policy *entities.Policy) (*PolicyResponse, error) {
	if policy == nil {
		return nil, status.Error(codes.InvalidArgument, "policy is required")
	}

	h.logger.InfoContext(ctx, "handling UpdatePolicy request",
		slog.String("policy_name", policy.Name()))

	// Update policy - returns Result[*Policy]
	result := h.policyService.UpdatePolicy(ctx, policy)

	// Handle Result type
	if result.IsErr() {
		err := result.UnwrapErr()
		return nil, h.mapErrorToStatus(err)
	}

	updated := result.Unwrap()
	return h.policyToResponse(updated), nil
}

// DeletePolicy handles DeletePolicy requests.
func (h *PolicyHandlers) DeletePolicy(ctx context.Context, req *PolicyRequest) error {
	if req.Name == "" {
		return status.Error(codes.InvalidArgument, "policy name is required")
	}

	h.logger.InfoContext(ctx, "handling DeletePolicy request",
		slog.String("policy_name", req.Name))

	err := h.policyService.DeletePolicy(ctx, req.Name)
	if err != nil {
		return h.mapErrorToStatus(err)
	}

	return nil
}

// ListPolicies handles ListPolicies requests with Result type handling.
func (h *PolicyHandlers) ListPolicies(ctx context.Context) ([]*PolicyResponse, error) {
	h.logger.DebugContext(ctx, "handling ListPolicies request")

	// List policies - returns Result[[]*Policy]
	result := h.policyService.ListPolicies(ctx)

	// Handle Result type
	if result.IsErr() {
		err := result.UnwrapErr()
		return nil, h.mapErrorToStatus(err)
	}

	policies := result.Unwrap()
	responses := make([]*PolicyResponse, len(policies))
	for i, policy := range policies {
		responses[i] = h.policyToResponse(policy)
	}

	return responses, nil
}

// policyToResponse converts a domain policy to a response.
func (h *PolicyHandlers) policyToResponse(policy *entities.Policy) *PolicyResponse {
	resp := &PolicyResponse{
		Name:      policy.Name(),
		Version:   int32(policy.Version()),
		CreatedAt: timestamppb.New(policy.CreatedAt()),
		UpdatedAt: timestamppb.New(policy.UpdatedAt()),
	}

	// Handle Option types for configs
	if policy.CircuitBreaker().IsSome() {
		cb := policy.CircuitBreaker().Unwrap()
		resp.CircuitBreaker = &CircuitBreakerConfigResponse{
			FailureThreshold: int32(cb.FailureThreshold),
			SuccessThreshold: int32(cb.SuccessThreshold),
			Timeout:          durationpb.New(cb.Timeout),
			ProbeCount:       int32(cb.ProbeCount),
		}
	}

	if policy.Retry().IsSome() {
		r := policy.Retry().Unwrap()
		resp.Retry = &RetryConfigResponse{
			MaxAttempts:   int32(r.MaxAttempts),
			BaseDelay:     durationpb.New(r.BaseDelay),
			MaxDelay:      durationpb.New(r.MaxDelay),
			Multiplier:    r.Multiplier,
			JitterPercent: r.JitterPercent,
		}
	}

	if policy.Timeout().IsSome() {
		t := policy.Timeout().Unwrap()
		resp.Timeout = &TimeoutConfigResponse{
			Default: durationpb.New(t.Default),
			Max:     durationpb.New(t.Max),
		}
	}

	if policy.RateLimit().IsSome() {
		rl := policy.RateLimit().Unwrap()
		resp.RateLimit = &RateLimitConfigResponse{
			Algorithm: rl.Algorithm,
			Limit:     int32(rl.Limit),
			Window:    durationpb.New(rl.Window),
			BurstSize: int32(rl.BurstSize),
		}
	}

	if policy.Bulkhead().IsSome() {
		bh := policy.Bulkhead().Unwrap()
		resp.Bulkhead = &BulkheadConfigResponse{
			MaxConcurrent: int32(bh.MaxConcurrent),
			MaxQueue:      int32(bh.MaxQueue),
			QueueTimeout:  durationpb.New(bh.QueueTimeout),
		}
	}

	return resp
}

// mapErrorToStatus maps domain errors to gRPC status codes.
func (h *PolicyHandlers) mapErrorToStatus(err error) error {
	errStr := err.Error()

	// Map common error patterns to gRPC codes
	switch {
	case contains(errStr, "not found"):
		return status.Error(codes.NotFound, err.Error())
	case contains(errStr, "already exists"):
		return status.Error(codes.AlreadyExists, err.Error())
	case contains(errStr, "validation failed"):
		return status.Error(codes.InvalidArgument, err.Error())
	case contains(errStr, "invalid"):
		return status.Error(codes.InvalidArgument, err.Error())
	case contains(errStr, "unauthorized"):
		return status.Error(codes.Unauthenticated, err.Error())
	case contains(errStr, "permission denied"):
		return status.Error(codes.PermissionDenied, err.Error())
	case contains(errStr, "timeout"):
		return status.Error(codes.DeadlineExceeded, err.Error())
	case contains(errStr, "unavailable"):
		return status.Error(codes.Unavailable, err.Error())
	default:
		return status.Error(codes.Internal, fmt.Sprintf("internal error: %v", err))
	}
}

// contains checks if s contains substr (case-insensitive).
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && containsLower(s, substr)))
}

func containsLower(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if matchLower(s[i:i+len(substr)], substr) {
			return true
		}
	}
	return false
}

func matchLower(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}
