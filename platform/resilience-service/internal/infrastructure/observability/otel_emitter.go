// Package observability provides OpenTelemetry-based observability implementations.
package observability

import (
	"context"
	"log/slog"

	"github.com/auth-platform/platform/resilience-service/internal/domain/valueobjects"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// OTelEmitter implements EventEmitter using OpenTelemetry.
type OTelEmitter struct {
	tracer  trace.Tracer
	meter   metric.Meter
	logger  *slog.Logger
	
	// Metrics
	eventCounter    metric.Int64Counter
	executionTimer  metric.Float64Histogram
	circuitCounter  metric.Int64Counter
	retryCounter    metric.Int64Counter
}

// NewOTelEmitter creates a new OpenTelemetry-based event emitter.
func NewOTelEmitter(
	tracer trace.Tracer,
	meter metric.Meter,
	logger *slog.Logger,
) (*OTelEmitter, error) {
	// Create metrics instruments
	eventCounter, err := meter.Int64Counter(
		"resilience_events_total",
		metric.WithDescription("Total number of resilience events emitted"),
	)
	if err != nil {
		return nil, err
	}

	executionTimer, err := meter.Float64Histogram(
		"resilience_execution_duration_seconds",
		metric.WithDescription("Duration of resilience policy executions"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	circuitCounter, err := meter.Int64Counter(
		"resilience_circuit_state_changes_total",
		metric.WithDescription("Total number of circuit breaker state changes"),
	)
	if err != nil {
		return nil, err
	}

	retryCounter, err := meter.Int64Counter(
		"resilience_retry_attempts_total",
		metric.WithDescription("Total number of retry attempts"),
	)
	if err != nil {
		return nil, err
	}

	return &OTelEmitter{
		tracer:         tracer,
		meter:          meter,
		logger:         logger,
		eventCounter:   eventCounter,
		executionTimer: executionTimer,
		circuitCounter: circuitCounter,
		retryCounter:   retryCounter,
	}, nil
}

// Emit emits a domain event using OpenTelemetry.
func (o *OTelEmitter) Emit(ctx context.Context, event valueobjects.DomainEvent) error {
	ctx, span := o.tracer.Start(ctx, "event.emit")
	defer span.End()

	// Add event attributes to span
	span.SetAttributes(
		attribute.String("event.id", event.EventID()),
		attribute.String("event.type", event.EventType()),
		attribute.String("event.aggregate_id", event.AggregateID()),
		attribute.String("event.timestamp", event.Timestamp().Format("2006-01-02T15:04:05.000Z")),
	)

	// Record event metric
	o.eventCounter.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("event_type", event.EventType()),
			attribute.String("aggregate_id", event.AggregateID()),
		),
	)

	// Log the event
	o.logger.InfoContext(ctx, "domain event emitted",
		slog.String("event_id", event.EventID()),
		slog.String("event_type", event.EventType()),
		slog.String("aggregate_id", event.AggregateID()),
		slog.Time("timestamp", event.Timestamp()))

	// Handle specific event types
	if policyEvent, ok := event.(valueobjects.PolicyEvent); ok {
		o.emitPolicyEvent(ctx, policyEvent)
	}

	return nil
}

// emitPolicyEvent handles policy-specific events.
func (o *OTelEmitter) emitPolicyEvent(ctx context.Context, event valueobjects.PolicyEvent) {
	ctx, span := o.tracer.Start(ctx, "event.emit_policy")
	defer span.End()

	// Add policy-specific attributes
	span.SetAttributes(
		attribute.String("policy.name", event.PolicyName),
		attribute.Int("policy.version", event.Version),
		attribute.String("policy.event_type", string(event.Type)),
	)

	// Record policy event metric
	o.eventCounter.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("event_type", "policy"),
			attribute.String("policy_event_type", string(event.Type)),
			attribute.String("policy_name", event.PolicyName),
		),
	)

	o.logger.InfoContext(ctx, "policy event emitted",
		slog.String("policy_name", event.PolicyName),
		slog.String("policy_event_type", string(event.Type)),
		slog.Int("version", event.Version))
}

// RecordExecution records execution metrics.
func (o *OTelEmitter) RecordExecution(ctx context.Context, metrics valueobjects.ExecutionMetrics) {
	ctx, span := o.tracer.Start(ctx, "metrics.record_execution")
	defer span.End()

	// Add execution attributes to span
	span.SetAttributes(
		attribute.String("policy.name", metrics.PolicyName),
		attribute.Float64("execution.duration_seconds", metrics.ExecutionTime.Seconds()),
		attribute.Bool("execution.success", metrics.Success),
	)

	// Record execution duration
	o.executionTimer.Record(ctx, metrics.ExecutionTime.Seconds(),
		metric.WithAttributes(
			attribute.String("policy_name", metrics.PolicyName),
			attribute.Bool("success", metrics.Success),
		),
	)

	// Record circuit breaker state if present
	if metrics.CircuitState != "" {
		span.SetAttributes(attribute.String("circuit.state", metrics.CircuitState))
		o.circuitCounter.Add(ctx, 1,
			metric.WithAttributes(
				attribute.String("policy_name", metrics.PolicyName),
				attribute.String("circuit_state", metrics.CircuitState),
			),
		)
	}

	// Record retry attempts if present
	if metrics.RetryAttempts > 0 {
		span.SetAttributes(attribute.Int("retry.attempts", metrics.RetryAttempts))
		o.retryCounter.Add(ctx, int64(metrics.RetryAttempts),
			metric.WithAttributes(
				attribute.String("policy_name", metrics.PolicyName),
			),
		)
	}

	// Record rate limiting if present
	if metrics.RateLimited {
		span.SetAttributes(attribute.Bool("rate_limit.limited", true))
	}

	// Record bulkhead queuing if present
	if metrics.BulkheadQueued {
		span.SetAttributes(attribute.Bool("bulkhead.queued", true))
	}

	o.logger.DebugContext(ctx, "execution metrics recorded",
		slog.String("policy_name", metrics.PolicyName),
		slog.Duration("duration", metrics.ExecutionTime),
		slog.Bool("success", metrics.Success))
}

// RecordCircuitState records circuit breaker state changes.
func (o *OTelEmitter) RecordCircuitState(ctx context.Context, policyName string, state string) {
	ctx, span := o.tracer.Start(ctx, "metrics.record_circuit_state")
	defer span.End()

	span.SetAttributes(
		attribute.String("policy.name", policyName),
		attribute.String("circuit.state", state),
	)

	o.circuitCounter.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("policy_name", policyName),
			attribute.String("circuit_state", state),
		),
	)

	o.logger.InfoContext(ctx, "circuit breaker state recorded",
		slog.String("policy_name", policyName),
		slog.String("state", state))
}

// RecordRetryAttempt records retry attempts.
func (o *OTelEmitter) RecordRetryAttempt(ctx context.Context, policyName string, attempt int) {
	ctx, span := o.tracer.Start(ctx, "metrics.record_retry")
	defer span.End()

	span.SetAttributes(
		attribute.String("policy.name", policyName),
		attribute.Int("retry.attempt", attempt),
	)

	o.retryCounter.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("policy_name", policyName),
		),
	)

	o.logger.DebugContext(ctx, "retry attempt recorded",
		slog.String("policy_name", policyName),
		slog.Int("attempt", attempt))
}

// RecordRateLimit records rate limiting events.
func (o *OTelEmitter) RecordRateLimit(ctx context.Context, policyName string, limited bool) {
	ctx, span := o.tracer.Start(ctx, "metrics.record_rate_limit")
	defer span.End()

	span.SetAttributes(
		attribute.String("policy.name", policyName),
		attribute.Bool("rate_limit.limited", limited),
	)

	if limited {
		o.eventCounter.Add(ctx, 1,
			metric.WithAttributes(
				attribute.String("event_type", "rate_limit"),
				attribute.String("policy_name", policyName),
			),
		)
	}

	o.logger.DebugContext(ctx, "rate limit recorded",
		slog.String("policy_name", policyName),
		slog.Bool("limited", limited))
}

// RecordBulkheadQueue records bulkhead queuing events.
func (o *OTelEmitter) RecordBulkheadQueue(ctx context.Context, policyName string, queued bool) {
	ctx, span := o.tracer.Start(ctx, "metrics.record_bulkhead")
	defer span.End()

	span.SetAttributes(
		attribute.String("policy.name", policyName),
		attribute.Bool("bulkhead.queued", queued),
	)

	if queued {
		o.eventCounter.Add(ctx, 1,
			metric.WithAttributes(
				attribute.String("event_type", "bulkhead_queue"),
				attribute.String("policy_name", policyName),
			),
		)
	}

	o.logger.DebugContext(ctx, "bulkhead queue recorded",
		slog.String("policy_name", policyName),
		slog.Bool("queued", queued))
}