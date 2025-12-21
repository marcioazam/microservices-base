// Package grpc provides resilience gRPC handlers with Option/Result type handling.
package grpc

import (
	"context"
	"fmt"

	"github.com/auth-platform/platform/resilience-service/internal/application/services"
	"github.com/auth-platform/platform/resilience-service/internal/domain/entities"
	"github.com/auth-platform/platform/resilience-service/internal/domain/valueobjects"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ResilienceHandler implements the gRPC ResilienceService.
type ResilienceHandler struct {
	policyService     *services.PolicyService
	resilienceService *services.ResilienceService
}

// NewResilienceHandler creates a new resilience handler.
func NewResilienceHandler(ps *services.PolicyService, rs *services.ResilienceService) *ResilienceHandler {
	return &ResilienceHandler{policyService: ps, resilienceService: rs}
}

// CreatePolicy creates a new resilience policy.
func (h *ResilienceHandler) CreatePolicy(ctx context.Context, req *CreatePolicyRequest) (*CreatePolicyResponse, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "policy name is required")
	}
	result := h.policyService.CreatePolicy(ctx, req.Name)
	if result.IsErr() {
		return nil, status.Error(codes.Internal, result.UnwrapErr().Error())
	}
	policy := result.Unwrap()
	if err := applyConfigs(policy, req.CircuitBreaker, req.Retry, req.Timeout, req.RateLimit, req.Bulkhead); err != nil {
		return nil, err
	}
	updateResult := h.policyService.UpdatePolicy(ctx, policy)
	if updateResult.IsErr() {
		return nil, status.Error(codes.Internal, updateResult.UnwrapErr().Error())
	}
	return &CreatePolicyResponse{Policy: policyToProto(updateResult.Unwrap())}, nil
}

// GetPolicy retrieves a policy by name.
func (h *ResilienceHandler) GetPolicy(ctx context.Context, req *GetPolicyRequest) (*GetPolicyResponse, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "policy name is required")
	}
	opt := h.policyService.GetPolicy(ctx, req.Name)
	if !opt.IsSome() {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("policy '%s' not found", req.Name))
	}
	return &GetPolicyResponse{Policy: policyToProto(opt.Unwrap())}, nil
}


// UpdatePolicy updates an existing policy.
func (h *ResilienceHandler) UpdatePolicy(ctx context.Context, req *UpdatePolicyRequest) (*UpdatePolicyResponse, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "policy name is required")
	}
	opt := h.policyService.GetPolicy(ctx, req.Name)
	if !opt.IsSome() {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("policy '%s' not found", req.Name))
	}
	policy := opt.Unwrap()
	if err := applyConfigs(policy, req.CircuitBreaker, req.Retry, req.Timeout, req.RateLimit, req.Bulkhead); err != nil {
		return nil, err
	}
	result := h.policyService.UpdatePolicy(ctx, policy)
	if result.IsErr() {
		return nil, status.Error(codes.Internal, result.UnwrapErr().Error())
	}
	return &UpdatePolicyResponse{Policy: policyToProto(result.Unwrap())}, nil
}

// DeletePolicy deletes a policy by name.
func (h *ResilienceHandler) DeletePolicy(ctx context.Context, req *DeletePolicyRequest) (*DeletePolicyResponse, error) {
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "policy name is required")
	}
	if err := h.policyService.DeletePolicy(ctx, req.Name); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &DeletePolicyResponse{Success: true}, nil
}

// ListPolicies returns all policies.
func (h *ResilienceHandler) ListPolicies(ctx context.Context, req *ListPoliciesRequest) (*ListPoliciesResponse, error) {
	result := h.policyService.ListPolicies(ctx)
	if result.IsErr() {
		return nil, status.Error(codes.Internal, result.UnwrapErr().Error())
	}
	policies := result.Unwrap()
	protoPolicies := make([]*Policy, len(policies))
	for i, p := range policies {
		protoPolicies[i] = policyToProto(p)
	}
	return &ListPoliciesResponse{Policies: protoPolicies, TotalCount: int32(len(policies))}, nil
}

// WatchPolicies streams policy change events.
func (h *ResilienceHandler) WatchPolicies(ctx context.Context, req *WatchPoliciesRequest, stream ResilienceService_WatchPoliciesServer) error {
	eventCh, err := h.policyService.WatchPolicies(ctx)
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event, ok := <-eventCh:
			if !ok {
				return nil
			}
			if len(req.PolicyNames) > 0 && !sliceContains(req.PolicyNames, event.PolicyName) {
				continue
			}
			if err := stream.Send(policyEventToProto(event)); err != nil {
				return err
			}
		}
	}
}


func applyConfigs(p *entities.Policy, cb *CircuitBreakerConfig, r *RetryConfig, t *TimeoutConfig, rl *RateLimitConfig, bh *BulkheadConfig) error {
	if cb != nil {
		if res := p.SetCircuitBreaker(protoToCB(cb)); res.IsErr() {
			return status.Error(codes.InvalidArgument, res.UnwrapErr().Error())
		}
	}
	if r != nil {
		if res := p.SetRetry(protoToRetry(r)); res.IsErr() {
			return status.Error(codes.InvalidArgument, res.UnwrapErr().Error())
		}
	}
	if t != nil {
		if res := p.SetTimeout(protoToTimeout(t)); res.IsErr() {
			return status.Error(codes.InvalidArgument, res.UnwrapErr().Error())
		}
	}
	if rl != nil {
		if res := p.SetRateLimit(protoToRL(rl)); res.IsErr() {
			return status.Error(codes.InvalidArgument, res.UnwrapErr().Error())
		}
	}
	if bh != nil {
		if res := p.SetBulkhead(protoToBH(bh)); res.IsErr() {
			return status.Error(codes.InvalidArgument, res.UnwrapErr().Error())
		}
	}
	return nil
}

func policyToProto(p *entities.Policy) *Policy {
	proto := &Policy{Name: p.Name(), Version: int32(p.Version()), CreatedAt: timestamppb.New(p.CreatedAt()), UpdatedAt: timestamppb.New(p.UpdatedAt())}
	if p.CircuitBreaker().IsSome() {
		proto.CircuitBreaker = cbToProto(p.CircuitBreaker().Unwrap())
	}
	if p.Retry().IsSome() {
		proto.Retry = retryToProto(p.Retry().Unwrap())
	}
	if p.Timeout().IsSome() {
		proto.Timeout = timeoutToProto(p.Timeout().Unwrap())
	}
	if p.RateLimit().IsSome() {
		proto.RateLimit = rlToProto(p.RateLimit().Unwrap())
	}
	if p.Bulkhead().IsSome() {
		proto.Bulkhead = bhToProto(p.Bulkhead().Unwrap())
	}
	return proto
}

func cbToProto(c *entities.CircuitBreakerConfig) *CircuitBreakerConfig {
	return &CircuitBreakerConfig{FailureThreshold: int32(c.FailureThreshold), SuccessThreshold: int32(c.SuccessThreshold), Timeout: durationpb.New(c.Timeout), ProbeCount: int32(c.ProbeCount)}
}

func retryToProto(c *entities.RetryConfig) *RetryConfig {
	return &RetryConfig{MaxAttempts: int32(c.MaxAttempts), BaseDelay: durationpb.New(c.BaseDelay), MaxDelay: durationpb.New(c.MaxDelay), Multiplier: c.Multiplier, JitterPercent: c.JitterPercent}
}

func timeoutToProto(c *entities.TimeoutConfig) *TimeoutConfig {
	return &TimeoutConfig{DefaultTimeout: durationpb.New(c.Default), MaxTimeout: durationpb.New(c.Max)}
}

func rlToProto(c *entities.RateLimitConfig) *RateLimitConfig {
	return &RateLimitConfig{Algorithm: c.Algorithm, Limit: int32(c.Limit), Window: durationpb.New(c.Window), BurstSize: int32(c.BurstSize)}
}

func bhToProto(c *entities.BulkheadConfig) *BulkheadConfig {
	return &BulkheadConfig{MaxConcurrent: int32(c.MaxConcurrent), MaxQueue: int32(c.MaxQueue), QueueTimeout: durationpb.New(c.QueueTimeout)}
}


func protoToCB(p *CircuitBreakerConfig) *entities.CircuitBreakerConfig {
	return &entities.CircuitBreakerConfig{FailureThreshold: int(p.FailureThreshold), SuccessThreshold: int(p.SuccessThreshold), Timeout: p.Timeout.AsDuration(), ProbeCount: int(p.ProbeCount)}
}

func protoToRetry(p *RetryConfig) *entities.RetryConfig {
	return &entities.RetryConfig{MaxAttempts: int(p.MaxAttempts), BaseDelay: p.BaseDelay.AsDuration(), MaxDelay: p.MaxDelay.AsDuration(), Multiplier: p.Multiplier, JitterPercent: p.JitterPercent}
}

func protoToTimeout(p *TimeoutConfig) *entities.TimeoutConfig {
	return &entities.TimeoutConfig{Default: p.DefaultTimeout.AsDuration(), Max: p.MaxTimeout.AsDuration()}
}

func protoToRL(p *RateLimitConfig) *entities.RateLimitConfig {
	return &entities.RateLimitConfig{Algorithm: p.Algorithm, Limit: int(p.Limit), Window: p.Window.AsDuration(), BurstSize: int(p.BurstSize)}
}

func protoToBH(p *BulkheadConfig) *entities.BulkheadConfig {
	return &entities.BulkheadConfig{MaxConcurrent: int(p.MaxConcurrent), MaxQueue: int(p.MaxQueue), QueueTimeout: p.QueueTimeout.AsDuration()}
}

func policyEventToProto(e valueobjects.PolicyEvent) *PolicyEvent {
	eventType := PolicyEventType_POLICY_EVENT_TYPE_UNSPECIFIED
	switch e.Type {
	case valueobjects.PolicyCreated:
		eventType = PolicyEventType_POLICY_EVENT_TYPE_CREATED
	case valueobjects.PolicyUpdated:
		eventType = PolicyEventType_POLICY_EVENT_TYPE_UPDATED
	case valueobjects.PolicyDeleted:
		eventType = PolicyEventType_POLICY_EVENT_TYPE_DELETED
	}
	return &PolicyEvent{EventId: e.ID, Type: eventType, PolicyName: e.PolicyName, Version: int32(e.Version), Timestamp: timestamppb.New(e.OccurredAt)}
}

// sliceContains checks if a string slice contains an item.
func sliceContains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
