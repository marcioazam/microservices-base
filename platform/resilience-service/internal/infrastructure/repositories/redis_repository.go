// Package repositories provides Redis-based policy repository implementation.
package repositories

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"strings"

	"github.com/authcorp/libs/go/src/functional"
	"github.com/auth-platform/platform/resilience-service/internal/domain/entities"
	"github.com/auth-platform/platform/resilience-service/internal/domain/interfaces"
	"github.com/auth-platform/platform/resilience-service/internal/domain/valueobjects"
	"github.com/auth-platform/platform/resilience-service/internal/infrastructure/config"
	"github.com/redis/go-redis/v9"
)

// RedisRepository implements PolicyRepository using Redis.
type RedisRepository struct {
	client  *redis.Client
	codec   *PolicyCodec
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
	opts, err := redis.ParseURL(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	if cfg.TLSEnabled {
		opts.TLSConfig = &tls.Config{
			InsecureSkipVerify: cfg.TLSSkipVerify,
		}
	}

	opts.DB = cfg.DB
	opts.Password = cfg.Password
	opts.DialTimeout = cfg.ConnectTimeout
	opts.ReadTimeout = cfg.ReadTimeout
	opts.WriteTimeout = cfg.WriteTimeout
	opts.MaxRetries = cfg.MaxRetries
	opts.PoolSize = cfg.PoolSize

	client := redis.NewClient(opts)

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
		codec:   NewPolicyCodec(),
		logger:  logger,
		metrics: metrics,
		eventCh: make(chan valueobjects.PolicyEvent, 100),
	}, nil
}

// Get retrieves a policy by name from Redis.
func (r *RedisRepository) Get(ctx context.Context, name string) functional.Option[*entities.Policy] {
	key := r.policyKey(name)

	r.logger.DebugContext(ctx, "retrieving policy from Redis",
		slog.String("policy_name", name),
		slog.String("key", key))

	data, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return functional.None[*entities.Policy]()
		}
		r.logger.ErrorContext(ctx, "failed to get policy from Redis",
			slog.String("policy_name", name),
			slog.String("error", err.Error()))
		return functional.None[*entities.Policy]()
	}

	result := r.codec.Decode(data)
	if result.IsErr() {
		r.logger.ErrorContext(ctx, "failed to decode policy",
			slog.String("policy_name", name),
			slog.String("error", result.UnwrapErr().Error()))
		return functional.None[*entities.Policy]()
	}

	r.logger.DebugContext(ctx, "policy retrieved successfully",
		slog.String("policy_name", name),
		slog.Int("version", result.Unwrap().Version()))

	return functional.Some(result.Unwrap())
}

// Save stores a policy in Redis.
func (r *RedisRepository) Save(ctx context.Context, policy *entities.Policy) functional.Result[*entities.Policy] {
	key := r.policyKey(policy.Name())

	r.logger.DebugContext(ctx, "saving policy to Redis",
		slog.String("policy_name", policy.Name()),
		slog.String("key", key),
		slog.Int("version", policy.Version()))

	encodeResult := r.codec.Encode(policy)
	if encodeResult.IsErr() {
		r.logger.ErrorContext(ctx, "failed to encode policy",
			slog.String("policy_name", policy.Name()),
			slog.String("error", encodeResult.UnwrapErr().Error()))
		return functional.Err[*entities.Policy](encodeResult.UnwrapErr())
	}

	err := r.client.Set(ctx, key, encodeResult.Unwrap(), 0).Err()
	if err != nil {
		r.logger.ErrorContext(ctx, "failed to save policy to Redis",
			slog.String("policy_name", policy.Name()),
			slog.String("error", err.Error()))
		return functional.Err[*entities.Policy](fmt.Errorf("failed to save policy: %w", err))
	}

	listKey := r.policyListKey()
	if err := r.client.SAdd(ctx, listKey, policy.Name()).Err(); err != nil {
		r.logger.WarnContext(ctx, "failed to add policy to list",
			slog.String("policy_name", policy.Name()),
			slog.String("error", err.Error()))
	}

	r.logger.InfoContext(ctx, "policy saved successfully",
		slog.String("policy_name", policy.Name()),
		slog.Int("version", policy.Version()))

	return functional.Ok(policy)
}

// Delete removes a policy from Redis.
func (r *RedisRepository) Delete(ctx context.Context, name string) error {
	key := r.policyKey(name)

	r.logger.InfoContext(ctx, "deleting policy from Redis",
		slog.String("policy_name", name),
		slog.String("key", key))

	if err := r.client.Del(ctx, key).Err(); err != nil {
		r.logger.ErrorContext(ctx, "failed to delete policy from Redis",
			slog.String("policy_name", name),
			slog.String("error", err.Error()))
		return fmt.Errorf("failed to delete policy: %w", err)
	}

	listKey := r.policyListKey()
	if err := r.client.SRem(ctx, listKey, name).Err(); err != nil {
		r.logger.WarnContext(ctx, "failed to remove policy from list",
			slog.String("policy_name", name),
			slog.String("error", err.Error()))
	}

	r.logger.InfoContext(ctx, "policy deleted successfully",
		slog.String("policy_name", name))

	return nil
}

// List returns all policies from Redis.
func (r *RedisRepository) List(ctx context.Context) functional.Result[[]*entities.Policy] {
	listKey := r.policyListKey()

	r.logger.DebugContext(ctx, "listing all policies from Redis")

	names, err := r.client.SMembers(ctx, listKey).Result()
	if err != nil {
		r.logger.ErrorContext(ctx, "failed to get policy list from Redis",
			slog.String("error", err.Error()))
		return functional.Err[[]*entities.Policy](fmt.Errorf("failed to list policies: %w", err))
	}

	if len(names) == 0 {
		return functional.Ok([]*entities.Policy{})
	}

	pipe := r.client.Pipeline()
	cmds := make([]*redis.StringCmd, len(names))

	for i, name := range names {
		key := r.policyKey(name)
		cmds[i] = pipe.Get(ctx, key)
	}

	_, err = pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		r.logger.ErrorContext(ctx, "failed to execute pipeline",
			slog.String("error", err.Error()))
		return functional.Err[[]*entities.Policy](fmt.Errorf("failed to get policies: %w", err))
	}

	policies := make([]*entities.Policy, 0, len(names))
	for i, cmd := range cmds {
		data, err := cmd.Result()
		if err != nil {
			if err == redis.Nil {
				r.logger.WarnContext(ctx, "policy not found during list",
					slog.String("policy_name", names[i]))
				continue
			}
			r.logger.ErrorContext(ctx, "failed to get policy data",
				slog.String("policy_name", names[i]),
				slog.String("error", err.Error()))
			continue
		}

		result := r.codec.Decode(data)
		if result.IsErr() {
			r.logger.ErrorContext(ctx, "failed to decode policy during list",
				slog.String("policy_name", names[i]),
				slog.String("error", result.UnwrapErr().Error()))
			continue
		}

		policies = append(policies, result.Unwrap())
	}

	r.logger.DebugContext(ctx, "policies listed successfully",
		slog.Int("count", len(policies)))

	return functional.Ok(policies)
}

// Exists checks if a policy exists in Redis.
func (r *RedisRepository) Exists(ctx context.Context, name string) bool {
	key := r.policyKey(name)
	exists, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		r.logger.ErrorContext(ctx, "failed to check policy existence",
			slog.String("policy_name", name),
			slog.String("error", err.Error()))
		return false
	}
	return exists > 0
}

// Watch returns a channel for policy change events.
func (r *RedisRepository) Watch(ctx context.Context) (<-chan valueobjects.PolicyEvent, error) {
	r.logger.InfoContext(ctx, "starting policy watch")
	return r.eventCh, nil
}

// Close closes the Redis connection.
func (r *RedisRepository) Close() error {
	close(r.eventCh)
	return r.client.Close()
}

func (r *RedisRepository) policyKey(name string) string {
	return fmt.Sprintf("resilience:policy:%s", name)
}

func (r *RedisRepository) policyListKey() string {
	return "resilience:policies"
}

func sanitizeURL(url string) string {
	if strings.Contains(url, "@") {
		parts := strings.Split(url, "@")
		if len(parts) == 2 {
			return "redis://***@" + parts[1]
		}
	}
	return url
}

// Ensure RedisRepository implements PolicyRepository.
var _ interfaces.PolicyRepository = (*RedisRepository)(nil)
