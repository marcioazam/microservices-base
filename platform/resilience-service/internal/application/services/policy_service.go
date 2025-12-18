// Package services provides policy management services.
package services

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/auth-platform/platform/resilience-service/internal/domain/entities"
	"github.com/auth-platform/platform/resilience-service/internal/domain/interfaces"
	"github.com/auth-platform/platform/resilience-service/internal/domain/valueobjects"
	"go.opentelemetry.io/otel/trace"
)

// PolicyService manages policy lifecycle operations.
type PolicyService struct {
	repository interfaces.PolicyRepository
	validator  interfaces.PolicyValidator
	emitter    interfaces.EventEmitter
	logger     *slog.Logger
	tracer     trace.Tracer
}

// NewPolicyService creates a new policy service.
func NewPolicyService(
	repository interfaces.PolicyRepository,
	validator interfaces.PolicyValidator,
	emitter interfaces.EventEmitter,
	logger *slog.Logger,
	tracer trace.Tracer,
) *PolicyService {
	return &PolicyService{
		repository: repository,
		validator:  validator,
		emitter:    emitter,
		logger:     logger,
		tracer:     tracer,
	}
}

// CreatePolicy creates a new resilience policy.
func (s *PolicyService) CreatePolicy(ctx context.Context, name string) (*entities.Policy, error) {
	ctx, span := s.tracer.Start(ctx, "policy.create")
	defer span.End()

	s.logger.InfoContext(ctx, "creating new policy", slog.String("policy_name", name))

	// Check if policy already exists
	existing, err := s.repository.Get(ctx, name)
	if err == nil && existing != nil {
		return nil, fmt.Errorf("policy '%s' already exists", name)
	}

	// Create new policy
	policy, err := entities.NewPolicy(name)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create policy entity",
			slog.String("policy_name", name),
			slog.String("error", err.Error()))
		span.RecordError(err)
		return nil, fmt.Errorf("failed to create policy: %w", err)
	}

	// Validate policy
	if err := s.validator.Validate(policy); err != nil {
		s.logger.ErrorContext(ctx, "policy validation failed",
			slog.String("policy_name", name),
			slog.String("error", err.Error()))
		span.RecordError(err)
		return nil, fmt.Errorf("policy validation failed: %w", err)
	}

	// Save policy
	if err := s.repository.Save(ctx, policy); err != nil {
		s.logger.ErrorContext(ctx, "failed to save policy",
			slog.String("policy_name", name),
			slog.String("error", err.Error()))
		span.RecordError(err)
		return nil, fmt.Errorf("failed to save policy: %w", err)
	}

	// Emit policy created event
	event := valueobjects.NewPolicyEvent(valueobjects.PolicyCreated, name, policy.Version())
	if err := s.emitter.Emit(ctx, event); err != nil {
		s.logger.WarnContext(ctx, "failed to emit policy created event",
			slog.String("policy_name", name),
			slog.String("error", err.Error()))
	}

	s.logger.InfoContext(ctx, "policy created successfully",
		slog.String("policy_name", name),
		slog.Int("version", policy.Version()))

	return policy, nil
}

// UpdatePolicy updates an existing resilience policy.
func (s *PolicyService) UpdatePolicy(ctx context.Context, policy *entities.Policy) error {
	ctx, span := s.tracer.Start(ctx, "policy.update")
	defer span.End()

	s.logger.InfoContext(ctx, "updating policy",
		slog.String("policy_name", policy.Name()),
		slog.Int("version", policy.Version()))

	// Validate policy
	if err := s.validator.Validate(policy); err != nil {
		s.logger.ErrorContext(ctx, "policy validation failed",
			slog.String("policy_name", policy.Name()),
			slog.String("error", err.Error()))
		span.RecordError(err)
		return fmt.Errorf("policy validation failed: %w", err)
	}

	// Increment version
	policy.IncrementVersion()

	// Save updated policy
	if err := s.repository.Save(ctx, policy); err != nil {
		s.logger.ErrorContext(ctx, "failed to save updated policy",
			slog.String("policy_name", policy.Name()),
			slog.String("error", err.Error()))
		span.RecordError(err)
		return fmt.Errorf("failed to save policy: %w", err)
	}

	// Emit policy updated event
	event := valueobjects.NewPolicyEvent(valueobjects.PolicyUpdated, policy.Name(), policy.Version())
	if err := s.emitter.Emit(ctx, event); err != nil {
		s.logger.WarnContext(ctx, "failed to emit policy updated event",
			slog.String("policy_name", policy.Name()),
			slog.String("error", err.Error()))
	}

	s.logger.InfoContext(ctx, "policy updated successfully",
		slog.String("policy_name", policy.Name()),
		slog.Int("version", policy.Version()))

	return nil
}

// GetPolicy retrieves a policy by name.
func (s *PolicyService) GetPolicy(ctx context.Context, name string) (*entities.Policy, error) {
	ctx, span := s.tracer.Start(ctx, "policy.get")
	defer span.End()

	s.logger.DebugContext(ctx, "retrieving policy", slog.String("policy_name", name))

	policy, err := s.repository.Get(ctx, name)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to retrieve policy",
			slog.String("policy_name", name),
			slog.String("error", err.Error()))
		span.RecordError(err)
		return nil, fmt.Errorf("failed to get policy: %w", err)
	}

	if policy == nil {
		return nil, fmt.Errorf("policy '%s' not found", name)
	}

	s.logger.DebugContext(ctx, "policy retrieved successfully",
		slog.String("policy_name", name),
		slog.Int("version", policy.Version()))

	return policy, nil
}

// DeletePolicy deletes a policy by name.
func (s *PolicyService) DeletePolicy(ctx context.Context, name string) error {
	ctx, span := s.tracer.Start(ctx, "policy.delete")
	defer span.End()

	s.logger.InfoContext(ctx, "deleting policy", slog.String("policy_name", name))

	// Get policy to get version for event
	policy, err := s.repository.Get(ctx, name)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to retrieve policy for deletion",
			slog.String("policy_name", name),
			slog.String("error", err.Error()))
		span.RecordError(err)
		return fmt.Errorf("failed to get policy for deletion: %w", err)
	}

	if policy == nil {
		return fmt.Errorf("policy '%s' not found", name)
	}

	// Delete policy
	if err := s.repository.Delete(ctx, name); err != nil {
		s.logger.ErrorContext(ctx, "failed to delete policy",
			slog.String("policy_name", name),
			slog.String("error", err.Error()))
		span.RecordError(err)
		return fmt.Errorf("failed to delete policy: %w", err)
	}

	// Emit policy deleted event
	event := valueobjects.NewPolicyEvent(valueobjects.PolicyDeleted, name, policy.Version())
	if err := s.emitter.Emit(ctx, event); err != nil {
		s.logger.WarnContext(ctx, "failed to emit policy deleted event",
			slog.String("policy_name", name),
			slog.String("error", err.Error()))
	}

	s.logger.InfoContext(ctx, "policy deleted successfully", slog.String("policy_name", name))

	return nil
}

// ListPolicies returns all policies.
func (s *PolicyService) ListPolicies(ctx context.Context) ([]*entities.Policy, error) {
	ctx, span := s.tracer.Start(ctx, "policy.list")
	defer span.End()

	s.logger.DebugContext(ctx, "listing all policies")

	policies, err := s.repository.List(ctx)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to list policies",
			slog.String("error", err.Error()))
		span.RecordError(err)
		return nil, fmt.Errorf("failed to list policies: %w", err)
	}

	s.logger.DebugContext(ctx, "policies listed successfully",
		slog.Int("count", len(policies)))

	return policies, nil
}

// WatchPolicies returns a channel for policy change events.
func (s *PolicyService) WatchPolicies(ctx context.Context) (<-chan valueobjects.PolicyEvent, error) {
	ctx, span := s.tracer.Start(ctx, "policy.watch")
	defer span.End()

	s.logger.InfoContext(ctx, "starting policy watch")

	eventCh, err := s.repository.Watch(ctx)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to start policy watch",
			slog.String("error", err.Error()))
		span.RecordError(err)
		return nil, fmt.Errorf("failed to watch policies: %w", err)
	}

	return eventCh, nil
}