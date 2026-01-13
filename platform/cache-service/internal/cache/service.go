package cache

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/auth-platform/cache-service/internal/config"
	"github.com/auth-platform/cache-service/internal/localcache"
	"github.com/auth-platform/cache-service/internal/loggingclient"
	"github.com/authcorp/libs/go/src/fault"
)

// RedisOperations defines the Redis operations interface.
type RedisOperations interface {
	Get(ctx context.Context, key string) ([]byte, Source, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Del(ctx context.Context, keys ...string) (int64, error)
	MGet(ctx context.Context, keys ...string) ([]interface{}, Source, error)
	SetWithExpire(ctx context.Context, entries map[string][]byte, ttl time.Duration) error
	Ping(ctx context.Context) error
	Close() error
	CircuitState() fault.State
}

// CacheServiceImpl implements the Service interface.
type CacheServiceImpl struct {
	redis        RedisOperations
	local        *localcache.Cache
	encryptor    Encryptor
	ttlConfig    TTLConfig
	localEnabled bool
	logger       *loggingclient.Client
}

// ServiceConfig holds cache service configuration.
type ServiceConfig struct {
	TTLConfig    TTLConfig
	LocalEnabled bool
	LocalConfig  localcache.Config
	Logger       *loggingclient.Client
}

// NewService creates a new cache service.
func NewService(redis RedisOperations, encryptor Encryptor, cfg ServiceConfig) *CacheServiceImpl {
	var local *localcache.Cache
	if cfg.LocalEnabled {
		local = localcache.New(cfg.LocalConfig)
	}

	logger := cfg.Logger
	if logger == nil {
		logger = loggingclient.NewNoop()
	}

	return &CacheServiceImpl{
		redis:        redis,
		local:        local,
		encryptor:    encryptor,
		ttlConfig:    cfg.TTLConfig,
		localEnabled: cfg.LocalEnabled,
		logger:       logger,
	}
}

// NewServiceFromConfig creates a new cache service from config.
func NewServiceFromConfig(redis RedisOperations, encryptor Encryptor, cfg config.CacheConfig, logger *loggingclient.Client) *CacheServiceImpl {
	return NewService(redis, encryptor, ServiceConfig{
		TTLConfig: TTLConfig{
			DefaultTTL: cfg.DefaultTTL,
			MinTTL:     time.Second,
			MaxTTL:     24 * time.Hour * 30,
		},
		LocalEnabled: cfg.LocalCacheEnabled,
		LocalConfig: localcache.Config{
			MaxSize:     cfg.LocalCacheSize,
			DefaultTTL:  cfg.DefaultTTL,
			CleanupTick: time.Minute,
		},
		Logger: logger,
	})
}

// Get retrieves a value from cache.
func (s *CacheServiceImpl) Get(ctx context.Context, namespace, key string) (*Entry, error) {
	if err := validateNamespace(namespace); err != nil {
		return nil, err
	}
	if key == "" {
		return nil, ErrInvalidKeyError
	}

	fullKey := buildKey(namespace, key)

	value, source, err := s.redis.Get(ctx, fullKey)
	if err != nil {
		s.logger.Debug(ctx, "cache get failed",
			loggingclient.String("namespace", namespace),
			loggingclient.String("key", key),
			loggingclient.Error(err),
		)
		return nil, err
	}

	s.logger.Debug(ctx, "cache get success",
		loggingclient.String("namespace", namespace),
		loggingclient.String("key", key),
		loggingclient.String("source", source.String()),
	)

	return &Entry{
		Value:  value,
		Source: source,
	}, nil
}

// Set stores a value in cache.
func (s *CacheServiceImpl) Set(ctx context.Context, namespace, key string, value []byte, ttl time.Duration, opts ...SetOption) error {
	if err := validateNamespace(namespace); err != nil {
		return err
	}
	if key == "" {
		return ErrInvalidKeyError
	}
	if len(value) == 0 {
		return ErrInvalidValueError
	}

	options := ApplySetOptions(opts...)

	normalizedTTL, err := s.ttlConfig.ValidateTTL(ttl)
	if err != nil {
		return err
	}

	dataToStore := value
	if options.encrypt && s.encryptor != nil {
		encrypted, err := s.encryptor.Encrypt(value)
		if err != nil {
			s.logger.Error(ctx, "encryption failed",
				loggingclient.String("namespace", namespace),
				loggingclient.String("key", key),
				loggingclient.Error(err),
			)
			return WrapError(ErrEncryptFailed, "failed to encrypt value", err)
		}
		dataToStore = encrypted
	}

	fullKey := buildKey(namespace, key)
	if err := s.redis.Set(ctx, fullKey, dataToStore, normalizedTTL); err != nil {
		s.logger.Error(ctx, "cache set failed",
			loggingclient.String("namespace", namespace),
			loggingclient.String("key", key),
			loggingclient.Error(err),
		)
		return err
	}

	s.logger.Debug(ctx, "cache set success",
		loggingclient.String("namespace", namespace),
		loggingclient.String("key", key),
		loggingclient.Int64("ttl_seconds", int64(normalizedTTL.Seconds())),
	)
	return nil
}

// Delete removes a value from cache.
func (s *CacheServiceImpl) Delete(ctx context.Context, namespace, key string) (bool, error) {
	if err := validateNamespace(namespace); err != nil {
		return false, err
	}
	if key == "" {
		return false, ErrInvalidKeyError
	}

	fullKey := buildKey(namespace, key)
	count, err := s.redis.Del(ctx, fullKey)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// BatchGet retrieves multiple values from cache.
func (s *CacheServiceImpl) BatchGet(ctx context.Context, namespace string, keys []string) (map[string][]byte, []string, error) {
	if err := validateNamespace(namespace); err != nil {
		return nil, nil, err
	}
	if len(keys) == 0 {
		return nil, nil, nil
	}

	// Build full keys
	fullKeys := make([]string, len(keys))
	for i, key := range keys {
		if key == "" {
			return nil, nil, ErrInvalidKeyError
		}
		fullKeys[i] = buildKey(namespace, key)
	}

	results, _, err := s.redis.MGet(ctx, fullKeys...)
	if err != nil {
		return nil, nil, err
	}

	found := make(map[string][]byte)
	var missing []string

	for i, result := range results {
		if result == nil {
			missing = append(missing, keys[i])
			continue
		}

		switch v := result.(type) {
		case string:
			found[keys[i]] = []byte(v)
		case []byte:
			found[keys[i]] = v
		default:
			missing = append(missing, keys[i])
		}
	}

	return found, missing, nil
}

// BatchSet stores multiple key-value pairs in cache.
func (s *CacheServiceImpl) BatchSet(ctx context.Context, namespace string, entries map[string][]byte, ttl time.Duration) (int, error) {
	if err := validateNamespace(namespace); err != nil {
		return 0, err
	}
	if len(entries) == 0 {
		return 0, nil
	}

	// Validate and normalize TTL
	normalizedTTL, err := s.ttlConfig.ValidateTTL(ttl)
	if err != nil {
		return 0, err
	}

	// Build full keys
	fullEntries := make(map[string][]byte, len(entries))
	for key, value := range entries {
		if key == "" {
			return 0, ErrInvalidKeyError
		}
		if len(value) == 0 {
			return 0, ErrInvalidValueError
		}
		fullEntries[buildKey(namespace, key)] = value
	}

	err = s.redis.SetWithExpire(ctx, fullEntries, normalizedTTL)
	if err != nil {
		return 0, err
	}

	return len(entries), nil
}

// Health returns the current health status.
func (s *CacheServiceImpl) Health(ctx context.Context) (*HealthStatus, error) {
	status := &HealthStatus{
		Healthy:    true,
		LocalCache: s.localEnabled,
	}

	if err := s.redis.Ping(ctx); err != nil {
		status.RedisStatus = "unavailable"
		status.Healthy = false
		s.logger.Warn(ctx, "redis health check failed", loggingclient.Error(err))
	} else {
		status.RedisStatus = "healthy"
	}

	cbState := s.redis.CircuitState()
	if cbState == fault.StateOpen {
		status.RedisStatus = fmt.Sprintf("circuit_open (%s)", status.RedisStatus)
		s.logger.Warn(ctx, "circuit breaker open")
	}

	return status, nil
}

// Close closes the cache service.
func (s *CacheServiceImpl) Close() error {
	if s.local != nil {
		s.local.Close()
	}
	return s.redis.Close()
}

func validateNamespace(namespace string) error {
	if namespace == "" {
		return ErrInvalidNamespace
	}
	if strings.ContainsAny(namespace, ":/\\") {
		return WrapError(ErrInvalidNamespace, "namespace contains invalid characters", nil)
	}
	return nil
}

func buildKey(namespace, key string) string {
	return fmt.Sprintf("%s:%s", namespace, key)
}
