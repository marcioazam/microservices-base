package observability

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"

	"github.com/auth-platform/cache-service/internal/loggingclient"
)

// Provider holds all observability components.
type Provider struct {
	Logger         *loggingclient.Client
	Metrics        *Metrics
	Tracer         *Tracer
	tracerProvider *sdktrace.TracerProvider
	config         Config
}

// Config holds observability configuration.
type Config struct {
	ServiceName     string
	ServiceVersion  string
	Environment     string
	LoggingConfig   loggingclient.Config
	MetricsEnabled  bool
	TracingEnabled  bool
	TracingEndpoint string
}

// DefaultConfig returns the default observability configuration.
func DefaultConfig() Config {
	return Config{
		ServiceName:     "cache-service",
		ServiceVersion:  "1.0.0",
		Environment:     "development",
		LoggingConfig:   loggingclient.DefaultConfig(),
		MetricsEnabled:  true,
		TracingEnabled:  false,
		TracingEndpoint: "",
	}
}

// New creates a new observability provider.
func New(cfg Config) (*Provider, error) {
	p := &Provider{
		config:  cfg,
		Metrics: NewMetrics(cfg.ServiceName),
		Tracer:  NewTracer(cfg.ServiceName),
	}

	// Initialize logging client
	logger, err := loggingclient.New(cfg.LoggingConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create logging client: %w", err)
	}
	p.Logger = logger

	// Initialize tracing if enabled
	if cfg.TracingEnabled && cfg.TracingEndpoint != "" {
		if err := p.initTracing(context.Background()); err != nil {
			logger.Warn(context.Background(), "failed to initialize tracing",
				loggingclient.Error(err))
		}
	}

	return p, nil
}

func (p *Provider) initTracing(ctx context.Context) error {
	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(p.config.TracingEndpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return fmt.Errorf("failed to create trace exporter: %w", err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(p.config.ServiceName),
			semconv.ServiceVersion(p.config.ServiceVersion),
			semconv.DeploymentEnvironment(p.config.Environment),
		),
	)
	if err != nil {
		return fmt.Errorf("failed to create resource: %w", err)
	}

	p.tracerProvider = sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	otel.SetTracerProvider(p.tracerProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return nil
}

// Shutdown gracefully shuts down all observability components.
func (p *Provider) Shutdown(ctx context.Context) error {
	var errs []error

	if p.Logger != nil {
		if err := p.Logger.Close(); err != nil {
			errs = append(errs, fmt.Errorf("logger shutdown: %w", err))
		}
	}

	if p.tracerProvider != nil {
		if err := p.tracerProvider.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("tracer shutdown: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("shutdown errors: %v", errs)
	}
	return nil
}

// Info logs at info level using the provider's logger.
func (p *Provider) Info(ctx context.Context, msg string, fields ...loggingclient.Field) {
	p.Logger.Info(ctx, msg, fields...)
}

// Error logs at error level using the provider's logger.
func (p *Provider) Error(ctx context.Context, msg string, fields ...loggingclient.Field) {
	p.Logger.Error(ctx, msg, fields...)
}

// Warn logs at warn level using the provider's logger.
func (p *Provider) Warn(ctx context.Context, msg string, fields ...loggingclient.Field) {
	p.Logger.Warn(ctx, msg, fields...)
}

// Debug logs at debug level using the provider's logger.
func (p *Provider) Debug(ctx context.Context, msg string, fields ...loggingclient.Field) {
	p.Logger.Debug(ctx, msg, fields...)
}
