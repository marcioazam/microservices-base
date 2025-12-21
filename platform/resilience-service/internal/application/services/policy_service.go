// Package services provides policy management services with functional error handling.
package services

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/authcorp/libs/go/src/functional"
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

// CreatePolicy creates a new resilience policy using functional composition.
func (s *PolicyService) CreatePolicy(ctx context.Context, name string) functional.Result[*entities.Policy] {
	ctx, span := s.tracer.Start(ctx, "policy.create")
	defer span.End()

	s.logger.InfoContext(ctx, "creating new policy", slog.String("policy_name", name))

	// Check if policy already exists
	if s.repository.Get(ctx, name).IsSome() {
		err := fmt.Errorf("policy '%s' already exists", name)
		span.RecordError(err)
		return functional.Err[*entities.Policy](err)
	}

	// Create policy using functional composition with FlatMapResult
	createResult := functional.TryFunc(entities.NewPolicy(name))

	// Chain: create -> validate -> save using FlatMapResult
	result := functional.FlatMapResult(createResult, func(policy *entities.Policy) functional.Result[*entities.Policy] {
		// Validate
		validationResult := s.validator.Validate(policy)
		if validationResult.IsErr() {
			s.logger.ErrorContext(ctx, "policy validation failed",
				slog.String("policy_name", name),
				slog.String("error", validationResult.UnwrapErr().Error()))
			span.RecordError(validationResult.UnwrapErr())
			return functional.Err[*entities.Policy](fmt.Errorf("validation failed: %w", validationResult.UnwrapErr()))
		}

		// Save using FlatMapResult
		return functional.FlatMapResult(s.repository.Save(ctx, policy), func(saved *entities.Policy) functional.Result[*entities.Policy] {
			// Emit event (non-blocking)
			event := valueobjects.NewPolicyEvent(valueobjects.PolicyCreated, name, saved.Version())
			if err := s.emitter.EmitPolicyEvent(ctx, event); err != nil {
				s.logger.WarnContext(ctx, "failed to emit policy created event",
					slog.String("policy_name", name),
					slog.String("error", err.Error()))
			}

			s.logger.InfoContext(ctx, "policy created successfully",
				slog.String("policy_name", name),
				slog.Int("version", saved.Version()))

			return functional.Ok(saved)
		})
	})

	if result.IsErr() {
		span.RecordError(result.UnwrapErr())
	}

	return result
}

// UpdatePolicy updates an existing resilience policy using functional composition.
func (s *PolicyService) UpdatePolicy(ctx context.Context, policy *entities.Policy) functional.Result[*entities.Policy] {
	ctx, span := s.tracer.Start(ctx, "policy.update")
	defer span.End()

	s.logger.InfoContext(ctx, "updating policy",
		slog.String("policy_name", policy.Name()),
		slog.Int("version", policy.Version()))

	// Chain: validate -> increment version -> save using FlatMapResult
	result := functional.FlatMapResult(s.validator.Validate(policy), func(validated *entities.Policy) functional.Result[*entities.Policy] {
		// Increment version
		validated.IncrementVersion()

		// Save and emit event
		return functional.FlatMapResult(s.repository.Save(ctx, validated), func(saved *entities.Policy) functional.Result[*entities.Policy] {
			// Emit event
			event := valueobjects.NewPolicyEvent(valueobjects.PolicyUpdated, saved.Name(), saved.Version())
			if err := s.emitter.EmitPolicyEvent(ctx, event); err != nil {
				s.logger.WarnContext(ctx, "failed to emit policy updated event",
					slog.String("policy_name", saved.Name()),
					slog.String("error", err.Error()))
			}

			s.logger.InfoContext(ctx, "policy updated successfully",
				slog.String("policy_name", saved.Name()),
				slog.Int("version", saved.Version()))

			return functional.Ok(saved)
		})
	})

	if result.IsErr() {
		s.logger.ErrorContext(ctx, "policy update failed",
			slog.String("policy_name", policy.Name()),
			slog.String("error", result.UnwrapErr().Error()))
		span.RecordError(result.UnwrapErr())
	}

	return result
}

// GetPolicy retrieves a policy by name.
func (s *PolicyService) GetPolicy(ctx context.Context, name string) functional.Option[*entities.Policy] {
	ctx, span := s.tracer.Start(ctx, "policy.get")
	defer span.End()

	s.logger.DebugContext(ctx, "retrieving policy", slog.String("policy_name", name))

	opt := s.repository.Get(ctx, name)
	if opt.IsSome() {
		s.logger.DebugContext(ctx, "policy retrieved successfully",
			slog.String("policy_name", name),
			slog.Int("version", opt.Unwrap().Version()))
	}

	return opt
}

// DeletePolicy deletes a policy by name.
func (s *PolicyService) DeletePolicy(ctx context.Context, name string) error {
	ctx, span := s.tracer.Start(ctx, "policy.delete")
	defer span.End()

	s.logger.InfoContext(ctx, "deleting policy", slog.String("policy_name", name))

	// Get policy to get version for event
	opt := s.repository.Get(ctx, name)
	if !opt.IsSome() {
		err := fmt.Errorf("policy '%s' not found", name)
		span.RecordError(err)
		return err
	}

	policy := opt.Unwrap()

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
	if err := s.emitter.EmitPolicyEvent(ctx, event); err != nil {
		s.logger.WarnContext(ctx, "failed to emit policy deleted event",
			slog.String("policy_name", name),
			slog.String("error", err.Error()))
	}

	s.logger.InfoContext(ctx, "policy deleted successfully", slog.String("policy_name", name))

	return nil
}

// ListPolicies returns all policies.
func (s *PolicyService) ListPolicies(ctx context.Context) functional.Result[[]*entities.Policy] {
	ctx, span := s.tracer.Start(ctx, "policy.list")
	defer span.End()

	s.logger.DebugContext(ctx, "listing all policies")

	result := s.repository.List(ctx)
	if result.IsOk() {
		s.logger.DebugContext(ctx, "policies listed successfully",
			slog.Int("count", len(result.Unwrap())))
	}

	return result
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
