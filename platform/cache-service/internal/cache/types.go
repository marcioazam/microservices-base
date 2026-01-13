package cache

import "time"

// Source indicates where the cached value was retrieved from.
type Source int

const (
	// SourceRedis indicates the value came from Redis.
	SourceRedis Source = iota
	// SourceLocal indicates the value came from local cache.
	SourceLocal
)

// String returns the string representation of Source.
func (s Source) String() string {
	switch s {
	case SourceRedis:
		return "redis"
	case SourceLocal:
		return "local"
	default:
		return "unknown"
	}
}

// Entry represents a cached value with metadata.
type Entry struct {
	Value     []byte
	Source    Source
	ExpiresAt time.Time
	Encrypted bool
}

// InvalidationEvent represents a cache invalidation message.
type InvalidationEvent struct {
	Namespace string   `json:"namespace"`
	Keys      []string `json:"keys"`
	Action    string   `json:"action"` // "delete" or "update"
	Timestamp int64    `json:"timestamp"`
}

// HealthStatus represents the health of the cache service.
type HealthStatus struct {
	Healthy      bool   `json:"healthy"`
	RedisStatus  string `json:"redis_status"`
	BrokerStatus string `json:"broker_status"`
	LocalCache   bool   `json:"local_cache_enabled"`
}

// SetOption configures Set operation behavior.
type SetOption func(*setOptions)

type setOptions struct {
	encrypt bool
}

// WithEncryption enables encryption for the cached value.
func WithEncryption() SetOption {
	return func(o *setOptions) {
		o.encrypt = true
	}
}

// ApplySetOptions applies all options and returns the configuration.
func ApplySetOptions(opts ...SetOption) setOptions {
	o := setOptions{}
	for _, opt := range opts {
		opt(&o)
	}
	return o
}

// InternalEntry represents a cache entry with metadata for local cache.
type InternalEntry struct {
	Value       []byte
	ExpiresAt   time.Time
	Encrypted   bool
	CreatedAt   time.Time
	AccessedAt  time.Time // For LRU tracking
	AccessCount int64     // For LFU tracking
}

// IsExpired checks if the entry has expired.
func (e *InternalEntry) IsExpired() bool {
	if e.ExpiresAt.IsZero() {
		return false
	}
	return time.Now().After(e.ExpiresAt)
}
