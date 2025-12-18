// Package repositories provides infrastructure implementations for data persistence.
package repositories

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/auth-platform/platform/resilience-service/internal/domain/entities"
	"github.com/auth-platform/platform/resilience-service/internal/domain/interfaces"
	"github.com/auth-platform/platform/resilience-service/internal/domain/valueobjects"
	"github.com/auth-platform/platform/resilience-service/internal/infrastructure/config"
	"github.com/redis/go-redis/v9"
)

// RedisRepository implements PolicyRepository using Redis.
type RedisRepository struct {
	client  *redis.Client
	logger  *slog.Logger
	metrics interfaces.MetricsRecorder
	eventCh chan valueobjects.PolicyEvent
}

// NewRedisRepository creates a new Redis-based policy repository.
func NewRedisRepository(
	cfg *config.RedisConfig,
	logger *slog.Logger,
	metrics interfaces.MetricsRecorder,
) (*RedisRepository, error) {
	// Parse Redis URL and configure TLS
	opts, err := redis.ParseURL(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	// Configure TLS if enabled
	if cfg.TLSEnabled {
		opts.TLSConfig = &tls.Config{
			InsecureSkipVerify: cfg.TLSSkipVerify,
		}
	}

	// Apply additional configuration
	opts.DB = cfg.DB
	opts.Password = cfg.Password
	opts.DialTimeout = cfg.ConnectTimeout
	opts.ReadTimeout = cfg.ReadTimeout
	opts.WriteTimeout = cfg.WriteTimeout
	opts.MaxRetries = cfg.MaxRetries
	opts.PoolSize = cfg.PoolSize

	client := redis.NewClient(opts)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), cfg.ConnectTimeout)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logger.Info("connected to Redis",
		slog.String("url", sanitizeURL(cfg.URL)),
		slog.Int("db", cfg.DB),
		slog.Bool("tls_enabled", cfg.TLSEnabled))

	return &RedisRepository{
		client:  client,
		logger:  logger,
		metrics: metrics,
		eventCh: make(chan valueobjects.PolicyEvent, 100),
	}, nil
}

// Get retrieves a policy by name from Redis.
func (r *RedisRepository) Get(ctx context.Context, name string) (*entities.Policy, error) {
	key := r.policyKey(name)
	
	r.logger.DebugContext(ctx, "retrieving policy from Redis",
		slog.String("policy_name", name),
		slog.String("key", key))

	data, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Policy not found
		}
		r.logger.ErrorContext(ctx, "failed to get policy from Redis",
			slog.String("policy_name", name),
			slog.String("error", err.Error()))
		return nil, fmt.Errorf("failed to get policy from Redis: %w", err)
	}

	policy, err := r.deserializePolicy(data)
	if err != nil {
		r.logger.ErrorContext(ctx, "failed to deserialize policy",
			slog.String("policy_name", name),
			slog.String("error", err.Error()))
		return nil, fmt.Errorf("failed to deserialize policy: %w", err)
	}

	r.logger.DebugContext(ctx, "policy retrieved successfully",
		slog.String("policy_name", name),
		slog.Int("version", policy.Version()))

	return policy, nil
}

// Save stores a policy in Redis.
func (r *RedisRepository) Save(ctx context.Context, policy *entities.Policy) error {
	key := r.policyKey(policy.Name())
	
	r.logger.DebugContext(ctx, "saving policy to Redis",
		slog.String("policy_name", policy.Name()),
		slog.String("key", key),
		slog.Int("version", policy.Version()))

	data, err := r.serializePolicy(policy)
	if err != nil {
		r.logger.ErrorContext(ctx, "failed to serialize policy",
			slog.String("policy_name", policy.Name()),
			slog.String("error", err.Error()))
		return fmt.Errorf("failed to serialize policy: %w", err)
	}

	// Use SET with expiration (optional, for cache-like behavior)
	err = r.client.Set(ctx, key, data, 0).Err() // 0 means no expiration
	if err != nil {
		r.logger.ErrorContext(ctx, "failed to save policy to Redis",
			slog.String("policy_name", policy.Name()),
			slog.String("error", err.Error()))
		return fmt.Errorf("failed to save policy to Redis: %w", err)
	}

	// Add to policy list
	listKey := r.policyListKey()
	err = r.client.SAdd(ctx, listKey, policy.Name()).Err()
	if err != nil {
		r.logger.WarnContext(ctx, "failed to add policy to list",
			slog.String("policy_name", policy.Name()),
			slog.String("error", err.Error()))
	}

	r.logger.InfoContext(ctx, "policy saved successfully",
		slog.String("policy_name", policy.Name()),
		slog.Int("version", policy.Version()))

	return nil
}

// Delete removes a policy from Redis.
func (r *RedisRepository) Delete(ctx context.Context, name string) error {
	key := r.policyKey(name)
	
	r.logger.InfoContext(ctx, "deleting policy from Redis",
		slog.String("policy_name", name),
		slog.String("key", key))

	// Delete the policy
	err := r.client.Del(ctx, key).Err()
	if err != nil {
		r.logger.ErrorContext(ctx, "failed to delete policy from Redis",
			slog.String("policy_name", name),
			slog.String("error", err.Error()))
		return fmt.Errorf("failed to delete policy from Redis: %w", err)
	}

	// Remove from policy list
	listKey := r.policyListKey()
	err = r.client.SRem(ctx, listKey, name).Err()
	if err != nil {
		r.logger.WarnContext(ctx, "failed to remove policy from list",
			slog.String("policy_name", name),
			slog.String("error", err.Error()))
	}

	r.logger.InfoContext(ctx, "policy deleted successfully",
		slog.String("policy_name", name))

	return nil
}

// List returns all policies from Redis.
func (r *RedisRepository) List(ctx context.Context) ([]*entities.Policy, error) {
	listKey := r.policyListKey()
	
	r.logger.DebugContext(ctx, "listing all policies from Redis")

	// Get all policy names
	names, err := r.client.SMembers(ctx, listKey).Result()
	if err != nil {
		r.logger.ErrorContext(ctx, "failed to get policy list from Redis",
			slog.String("error", err.Error()))
		return nil, fmt.Errorf("failed to get policy list from Redis: %w", err)
	}

	if len(names) == 0 {
		return []*entities.Policy{}, nil
	}

	// Get all policies in parallel
	pipe := r.client.Pipeline()
	cmds := make([]*redis.StringCmd, len(names))
	
	for i, name := range names {
		key := r.policyKey(name)
		cmds[i] = pipe.Get(ctx, key)
	}

	_, err = pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		r.logger.ErrorContext(ctx, "failed to execute pipeline for policy list",
			slog.String("error", err.Error()))
		return nil, fmt.Errorf("failed to get policies from Redis: %w", err)
	}

	// Deserialize policies
	policies := make([]*entities.Policy, 0, len(names))
	for i, cmd := range cmds {
		data, err := cmd.Result()
		if err != nil {
			if err == redis.Nil {
				// Policy was deleted between listing and retrieval
				r.logger.WarnContext(ctx, "policy not found during list operation",
					slog.String("policy_name", names[i]))
				continue
			}
			r.logger.ErrorContext(ctx, "failed to get policy data",
				slog.String("policy_name", names[i]),
				slog.String("error", err.Error()))
			continue
		}

		policy, err := r.deserializePolicy(data)
		if err != nil {
			r.logger.ErrorContext(ctx, "failed to deserialize policy during list",
				slog.String("policy_name", names[i]),
				slog.String("error", err.Error()))
			continue
		}

		policies = append(policies, policy)
	}

	r.logger.DebugContext(ctx, "policies listed successfully",
		slog.Int("count", len(policies)))

	return policies, nil
}

// Watch returns a channel for policy change events.
func (r *RedisRepository) Watch(ctx context.Context) (<-chan valueobjects.PolicyEvent, error) {
	r.logger.InfoContext(ctx, "starting policy watch")
	
	// For now, return the event channel
	// In a full implementation, this would set up Redis pub/sub
	return r.eventCh, nil
}

// Close closes the Redis connection.
func (r *RedisRepository) Close() error {
	close(r.eventCh)
	return r.client.Close()
}

// Helper methods

func (r *RedisRepository) policyKey(name string) string {
	return fmt.Sprintf("resilience:policy:%s", name)
}

func (r *RedisRepository) policyListKey() string {
	return "resilience:policies"
}

func (r *RedisRepository) serializePolicy(policy *entities.Policy) (string, error) {
	// Create a serializable representation
	data := map[string]any{
		"name":       policy.Name(),
		"version":    policy.Version(),
		"created_at": policy.CreatedAt(),
		"updated_at": policy.UpdatedAt(),
	}

	if cb := policy.CircuitBreaker(); cb != nil {
		data["circuit_breaker"] = map[string]any{
			"failure_threshold": cb.FailureThreshold,
			"success_threshold": cb.SuccessThreshold,
			"timeout":           cb.Timeout,
			"probe_count":       cb.ProbeCount,
		}
	}

	if retry := policy.Retry(); retry != nil {
		data["retry"] = map[string]any{
			"max_attempts":   retry.MaxAttempts,
			"base_delay":     retry.BaseDelay,
			"max_delay":      retry.MaxDelay,
			"multiplier":     retry.Multiplier,
			"jitter_percent": retry.JitterPercent,
		}
	}

	if timeout := policy.Timeout(); timeout != nil {
		data["timeout"] = map[string]any{
			"default": timeout.Default,
			"max":     timeout.Max,
		}
	}

	if rl := policy.RateLimit(); rl != nil {
		data["rate_limit"] = map[string]any{
			"algorithm":  rl.Algorithm,
			"limit":      rl.Limit,
			"window":     rl.Window,
			"burst_size": rl.BurstSize,
		}
	}

	if bh := policy.Bulkhead(); bh != nil {
		data["bulkhead"] = map[string]any{
			"max_concurrent": bh.MaxConcurrent,
			"max_queue":      bh.MaxQueue,
			"queue_timeout":  bh.QueueTimeout,
		}
	}

	bytes, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

func (r *RedisRepository) deserializePolicy(data string) (*entities.Policy, error) {
	var raw map[string]any
	if err := json.Unmarshal([]byte(data), &raw); err != nil {
		return nil, err
	}

	name, ok := raw["name"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid policy name")
	}

	policy, err := entities.NewPolicy(name)
	if err != nil {
		return nil, err
	}

	// Deserialize configurations
	if cbData, ok := raw["circuit_breaker"].(map[string]any); ok {
		cb, err := r.deserializeCircuitBreaker(cbData)
		if err != nil {
			return nil, fmt.Errorf("failed to deserialize circuit breaker: %w", err)
		}
		policy.SetCircuitBreaker(cb)
	}

	if retryData, ok := raw["retry"].(map[string]any); ok {
		retry, err := r.deserializeRetry(retryData)
		if err != nil {
			return nil, fmt.Errorf("failed to deserialize retry: %w", err)
		}
		policy.SetRetry(retry)
	}

	// Add other configuration deserializations as needed...

	return policy, nil
}

func (r *RedisRepository) deserializeCircuitBreaker(data map[string]any) (*entities.CircuitBreakerConfig, error) {
	failureThreshold := int(data["failure_threshold"].(float64))
	successThreshold := int(data["success_threshold"].(float64))
	timeoutStr := data["timeout"].(string)
	timeout, err := time.ParseDuration(timeoutStr)
	if err != nil {
		return nil, err
	}
	probeCount := int(data["probe_count"].(float64))

	return entities.NewCircuitBreakerConfig(failureThreshold, successThreshold, timeout, probeCount)
}

func (r *RedisRepository) deserializeRetry(data map[string]any) (*entities.RetryConfig, error) {
	maxAttempts := int(data["max_attempts"].(float64))
	baseDelayStr := data["base_delay"].(string)
	baseDelay, err := time.ParseDuration(baseDelayStr)
	if err != nil {
		return nil, err
	}
	maxDelayStr := data["max_delay"].(string)
	maxDelay, err := time.ParseDuration(maxDelayStr)
	if err != nil {
		return nil, err
	}
	multiplier := data["multiplier"].(float64)
	jitterPercent := data["jitter_percent"].(float64)

	return entities.NewRetryConfig(maxAttempts, baseDelay, maxDelay, multiplier, jitterPercent)
}

// sanitizeURL removes sensitive information from URL for logging.
func sanitizeURL(url string) string {
	if strings.Contains(url, "@") {
		parts := strings.Split(url, "@")
		if len(parts) == 2 {
			return "redis://***@" + parts[1]
		}
	}
	return url
}