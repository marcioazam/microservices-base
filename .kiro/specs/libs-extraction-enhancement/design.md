# Design Document: Library Extraction Enhancement

## Overview

This design document describes the architecture and implementation approach for enhancing the shared libraries (`libs/`) in the auth-platform monorepo. The enhancement focuses on:

1. **Auditing** existing libraries for gaps and documentation quality
2. **Extracting** reusable code from services to shared libraries
3. **Enhancing** existing libraries with state-of-the-art patterns
4. **Documenting** all libraries comprehensively with examples

The design follows functional programming principles where applicable, uses generics for maximum flexibility, and ensures all components are testable through property-based testing.

## Architecture

### High-Level Library Structure

```
libs/
├── go/                           # Go shared libraries
│   ├── collections/              # Data structures (existing)
│   ├── concurrency/              # Concurrency primitives (existing, enhanced)
│   ├── config/                   # Configuration management (NEW)
│   ├── context/                  # Context propagation (NEW)
│   ├── database/                 # Database utilities (NEW)
│   ├── domain/                   # Domain primitives (NEW)
│   ├── dto/                      # Request/Response DTOs (NEW)
│   ├── events/                   # Event handling (existing, enhanced)
│   ├── featureflags/             # Feature flags (NEW)
│   ├── functional/               # Functional types (existing)
│   ├── grpc/                     # gRPC utilities (existing, enhanced)
│   ├── http/                     # HTTP utilities (NEW)
│   ├── httpclient/               # HTTP client with resilience (NEW)
│   ├── idempotency/              # Idempotency utilities (NEW)
│   ├── lock/                     # Distributed locks (NEW)
│   ├── metrics/                  # Prometheus/OTEL metrics (NEW)
│   ├── optics/                   # Functional optics (existing)
│   ├── outbox/                   # Outbox pattern (NEW)
│   ├── pagination/               # Pagination utilities (NEW)
│   ├── patterns/                 # Design patterns (existing)
│   ├── resilience/               # Fault tolerance (existing, enhanced)
│   ├── security/                 # Security utilities (NEW)
│   ├── server/                   # Server utilities (existing, enhanced)
│   ├── testing/                  # Test utilities (existing, enhanced)
│   ├── utils/                    # General utilities (existing, enhanced)
│   ├── versioning/               # API versioning (NEW)
│   └── workerpool/               # Worker pool and job queue (NEW)
│
└── rust/                         # Rust shared libraries
    ├── caep/                     # CAEP implementation (existing)
    ├── common/                   # Common utilities (NEW)
    ├── domain/                   # Domain primitives (NEW)
    ├── error/                    # Error handling (NEW)
    ├── integration/              # Integration tests (existing)
    ├── linkerd/                  # Linkerd integration (existing)
    ├── observability/            # Observability (NEW)
    ├── pact/                     # Contract testing (existing)
    ├── resilience/               # Resilience patterns (NEW)
    ├── transport/                # HTTP/gRPC utilities (NEW)
    └── vault/                    # Vault integration (existing)
```

### Design Principles

1. **Zero Dependencies Where Possible**: Libraries should minimize external dependencies
2. **Generics First**: Use type parameters for maximum flexibility
3. **Functional Composition**: Support chaining and composition of operations
4. **Result Types**: Use Result/Either types instead of exceptions where applicable
5. **Immutability**: Prefer immutable data structures
6. **Thread Safety**: All shared state must be thread-safe
7. **Testability**: All components must be testable with property-based tests

## Components and Interfaces

### 1. Domain Primitives Library (`libs/go/domain/`)

Type-safe wrappers for common domain values that enforce validation at construction time.

```go
// libs/go/domain/email.go
package domain

import (
    "regexp"
    "github.com/auth-platform/libs/go/functional/result"
)

// Email represents a validated email address
type Email struct {
    value string
}

// NewEmail creates a new Email from a string, returning an error if invalid
func NewEmail(s string) result.Result[Email] {
    if !emailRegex.MatchString(s) {
        return result.Err[Email](ErrInvalidEmail)
    }
    return result.Ok(Email{value: s})
}

// String returns the email as a string
func (e Email) String() string { return e.value }

// MarshalJSON implements json.Marshaler
func (e Email) MarshalJSON() ([]byte, error) {
    return json.Marshal(e.value)
}

// UnmarshalJSON implements json.Unmarshaler
func (e *Email) UnmarshalJSON(data []byte) error {
    var s string
    if err := json.Unmarshal(data, &s); err != nil {
        return err
    }
    result := NewEmail(s)
    if result.IsErr() {
        return result.UnwrapErr()
    }
    *e = result.Unwrap()
    return nil
}
```

```go
// libs/go/domain/uuid.go
package domain

import (
    "crypto/rand"
    "encoding/hex"
)

// UUID represents a validated UUID v4
type UUID struct {
    bytes [16]byte
}

// NewUUID generates a new random UUID v4
func NewUUID() UUID {
    var uuid UUID
    rand.Read(uuid.bytes[:])
    uuid.bytes[6] = (uuid.bytes[6] & 0x0f) | 0x40 // Version 4
    uuid.bytes[8] = (uuid.bytes[8] & 0x3f) | 0x80 // Variant
    return uuid
}

// ParseUUID parses a UUID from string format
func ParseUUID(s string) result.Result[UUID] {
    // Implementation with validation
}

// String returns the UUID in standard format
func (u UUID) String() string {
    // Format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
}
```

```go
// libs/go/domain/money.go
package domain

import "math/big"

// Money represents a monetary value with currency
type Money struct {
    amount   *big.Int // Amount in smallest currency unit (cents)
    currency Currency
}

// Currency represents an ISO 4217 currency code
type Currency string

const (
    USD Currency = "USD"
    EUR Currency = "EUR"
    GBP Currency = "GBP"
)

// NewMoney creates a new Money value
func NewMoney(amount int64, currency Currency) Money {
    return Money{
        amount:   big.NewInt(amount),
        currency: currency,
    }
}

// Add adds two Money values (must be same currency)
func (m Money) Add(other Money) result.Result[Money] {
    if m.currency != other.currency {
        return result.Err[Money](ErrCurrencyMismatch)
    }
    sum := new(big.Int).Add(m.amount, other.amount)
    return result.Ok(Money{amount: sum, currency: m.currency})
}
```

### 2. Error Handling Library (`libs/go/utils/error/`)

Enhanced error handling with typed codes, wrapping, and HTTP/gRPC mapping.

```go
// libs/go/utils/error/error.go
package error

// ErrorCode represents a typed error code
type ErrorCode string

const (
    CodeValidation   ErrorCode = "VALIDATION_ERROR"
    CodeNotFound     ErrorCode = "NOT_FOUND"
    CodeUnauthorized ErrorCode = "UNAUTHORIZED"
    CodeForbidden    ErrorCode = "FORBIDDEN"
    CodeInternal     ErrorCode = "INTERNAL_ERROR"
    CodeTimeout      ErrorCode = "TIMEOUT"
    CodeRateLimit    ErrorCode = "RATE_LIMITED"
)

// AppError represents an application error with code and context
type AppError struct {
    Code    ErrorCode
    Message string
    Details map[string]interface{}
    Cause   error
}

// Error implements the error interface
func (e *AppError) Error() string {
    if e.Cause != nil {
        return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Cause)
    }
    return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying cause
func (e *AppError) Unwrap() error { return e.Cause }

// Wrap wraps an error with additional context
func Wrap(err error, code ErrorCode, message string) *AppError {
    return &AppError{
        Code:    code,
        Message: message,
        Cause:   err,
    }
}

// HTTPStatus returns the HTTP status code for this error
func (e *AppError) HTTPStatus() int {
    switch e.Code {
    case CodeValidation:
        return http.StatusBadRequest
    case CodeNotFound:
        return http.StatusNotFound
    case CodeUnauthorized:
        return http.StatusUnauthorized
    case CodeForbidden:
        return http.StatusForbidden
    case CodeRateLimit:
        return http.StatusTooManyRequests
    default:
        return http.StatusInternalServerError
    }
}

// GRPCCode returns the gRPC status code for this error
func (e *AppError) GRPCCode() codes.Code {
    switch e.Code {
    case CodeValidation:
        return codes.InvalidArgument
    case CodeNotFound:
        return codes.NotFound
    case CodeUnauthorized:
        return codes.Unauthenticated
    case CodeForbidden:
        return codes.PermissionDenied
    case CodeRateLimit:
        return codes.ResourceExhausted
    default:
        return codes.Internal
    }
}
```

### 3. Validation Library (`libs/go/utils/validator/`)

Composable validation with clear error messages.

```go
// libs/go/utils/validator/validator.go
package validator

// Validator is a function that validates a value
type Validator[T any] func(T) ValidationResult

// ValidationResult contains validation errors
type ValidationResult struct {
    Errors []ValidationError
}

// ValidationError represents a single validation error
type ValidationError struct {
    Field   string
    Message string
    Code    string
}

// IsValid returns true if there are no errors
func (r ValidationResult) IsValid() bool {
    return len(r.Errors) == 0
}

// And combines two validators (both must pass)
func And[T any](v1, v2 Validator[T]) Validator[T] {
    return func(value T) ValidationResult {
        r1 := v1(value)
        r2 := v2(value)
        return ValidationResult{
            Errors: append(r1.Errors, r2.Errors...),
        }
    }
}

// Or combines two validators (at least one must pass)
func Or[T any](v1, v2 Validator[T]) Validator[T] {
    return func(value T) ValidationResult {
        r1 := v1(value)
        if r1.IsValid() {
            return r1
        }
        return v2(value)
    }
}

// String validators
func MinLength(min int) Validator[string] {
    return func(s string) ValidationResult {
        if len(s) < min {
            return ValidationResult{
                Errors: []ValidationError{{
                    Message: fmt.Sprintf("must be at least %d characters", min),
                    Code:    "MIN_LENGTH",
                }},
            }
        }
        return ValidationResult{}
    }
}

func MaxLength(max int) Validator[string] {
    return func(s string) ValidationResult {
        if len(s) > max {
            return ValidationResult{
                Errors: []ValidationError{{
                    Message: fmt.Sprintf("must be at most %d characters", max),
                    Code:    "MAX_LENGTH",
                }},
            }
        }
        return ValidationResult{}
    }
}

func MatchesRegex(pattern *regexp.Regexp, message string) Validator[string] {
    return func(s string) ValidationResult {
        if !pattern.MatchString(s) {
            return ValidationResult{
                Errors: []ValidationError{{
                    Message: message,
                    Code:    "PATTERN_MISMATCH",
                }},
            }
        }
        return ValidationResult{}
    }
}
```

### 4. Observability Library (`libs/go/server/tracing/`)

Enhanced observability with structured logging, tracing, and metrics.

```go
// libs/go/server/tracing/context.go
package tracing

import (
    "context"
    "go.opentelemetry.io/otel/trace"
)

type contextKey string

const (
    correlationIDKey contextKey = "correlation_id"
    requestIDKey     contextKey = "request_id"
    userIDKey        contextKey = "user_id"
    tenantIDKey      contextKey = "tenant_id"
)

// WithCorrelationID adds a correlation ID to the context
func WithCorrelationID(ctx context.Context, id string) context.Context {
    return context.WithValue(ctx, correlationIDKey, id)
}

// CorrelationID extracts the correlation ID from context
func CorrelationID(ctx context.Context) string {
    if id, ok := ctx.Value(correlationIDKey).(string); ok {
        return id
    }
    return ""
}

// StartSpan creates a new span with correlation ID
func StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
    ctx, span := tracer.Start(ctx, name, opts...)
    if corrID := CorrelationID(ctx); corrID != "" {
        span.SetAttributes(attribute.String("correlation_id", corrID))
    }
    return ctx, span
}
```

```go
// libs/go/server/tracing/logger.go
package tracing

import (
    "encoding/json"
    "time"
)

// LogLevel represents log severity
type LogLevel string

const (
    DEBUG LogLevel = "DEBUG"
    INFO  LogLevel = "INFO"
    WARN  LogLevel = "WARN"
    ERROR LogLevel = "ERROR"
)

// LogEntry represents a structured log entry
type LogEntry struct {
    Timestamp     string                 `json:"timestamp"`
    Level         LogLevel               `json:"level"`
    Message       string                 `json:"message"`
    CorrelationID string                 `json:"correlation_id,omitempty"`
    TraceID       string                 `json:"trace_id,omitempty"`
    SpanID        string                 `json:"span_id,omitempty"`
    Fields        map[string]interface{} `json:"fields,omitempty"`
}

// Logger provides structured logging
type Logger struct {
    ctx    context.Context
    fields map[string]interface{}
}

// NewLogger creates a new logger with context
func NewLogger(ctx context.Context) *Logger {
    return &Logger{ctx: ctx, fields: make(map[string]interface{})}
}

// With adds fields to the logger
func (l *Logger) With(key string, value interface{}) *Logger {
    newFields := make(map[string]interface{})
    for k, v := range l.fields {
        newFields[k] = v
    }
    newFields[key] = RedactSensitive(key, value)
    return &Logger{ctx: l.ctx, fields: newFields}
}

// Info logs at INFO level
func (l *Logger) Info(message string) {
    l.log(INFO, message)
}

func (l *Logger) log(level LogLevel, message string) {
    entry := LogEntry{
        Timestamp:     time.Now().UTC().Format(time.RFC3339Nano),
        Level:         level,
        Message:       message,
        CorrelationID: CorrelationID(l.ctx),
        Fields:        l.fields,
    }
    
    if span := trace.SpanFromContext(l.ctx); span.SpanContext().IsValid() {
        entry.TraceID = span.SpanContext().TraceID().String()
        entry.SpanID = span.SpanContext().SpanID().String()
    }
    
    json.NewEncoder(os.Stdout).Encode(entry)
}

// RedactSensitive redacts sensitive field values
func RedactSensitive(key string, value interface{}) interface{} {
    sensitiveKeys := []string{"password", "token", "secret", "key", "authorization"}
    for _, k := range sensitiveKeys {
        if strings.Contains(strings.ToLower(key), k) {
            return "[REDACTED]"
        }
    }
    return value
}
```

### 5. Security Library (`libs/go/security/`)

Security utilities for common security patterns.

```go
// libs/go/security/compare.go
package security

import "crypto/subtle"

// ConstantTimeCompare compares two strings in constant time
func ConstantTimeCompare(a, b string) bool {
    return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// ConstantTimeCompareBytes compares two byte slices in constant time
func ConstantTimeCompareBytes(a, b []byte) bool {
    return subtle.ConstantTimeCompare(a, b) == 1
}
```

```go
// libs/go/security/random.go
package security

import (
    "crypto/rand"
    "encoding/base64"
    "encoding/hex"
)

// GenerateRandomBytes generates cryptographically secure random bytes
func GenerateRandomBytes(n int) ([]byte, error) {
    b := make([]byte, n)
    _, err := rand.Read(b)
    return b, err
}

// GenerateRandomHex generates a random hex string
func GenerateRandomHex(n int) (string, error) {
    b, err := GenerateRandomBytes(n)
    if err != nil {
        return "", err
    }
    return hex.EncodeToString(b), nil
}

// GenerateRandomBase64 generates a random URL-safe base64 string
func GenerateRandomBase64(n int) (string, error) {
    b, err := GenerateRandomBytes(n)
    if err != nil {
        return "", err
    }
    return base64.URLEncoding.EncodeToString(b), nil
}
```

```go
// libs/go/security/sanitize.go
package security

import (
    "html"
    "regexp"
    "strings"
)

// SanitizeHTML escapes HTML special characters
func SanitizeHTML(s string) string {
    return html.EscapeString(s)
}

// SanitizeSQL removes SQL injection patterns (use parameterized queries instead)
func SanitizeSQL(s string) string {
    // Remove common SQL injection patterns
    dangerous := []string{"--", ";", "'", "\"", "/*", "*/", "xp_", "sp_"}
    result := s
    for _, d := range dangerous {
        result = strings.ReplaceAll(result, d, "")
    }
    return result
}

// SanitizeShell removes shell injection patterns
func SanitizeShell(s string) string {
    // Remove shell metacharacters
    dangerous := regexp.MustCompile(`[;&|$\x60\\!><\(\)\[\]\{\}]`)
    return dangerous.ReplaceAllString(s, "")
}
```

### 6. HTTP/gRPC Transport Library

```go
// libs/go/http/middleware/logging.go
package middleware

import (
    "net/http"
    "time"
)

// LoggingMiddleware logs HTTP requests
func LoggingMiddleware(logger *tracing.Logger) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            start := time.Now()
            
            // Wrap response writer to capture status
            wrapped := &responseWriter{ResponseWriter: w, status: http.StatusOK}
            
            // Extract or generate correlation ID
            corrID := r.Header.Get("X-Correlation-ID")
            if corrID == "" {
                corrID = uuid.New().String()
            }
            ctx := tracing.WithCorrelationID(r.Context(), corrID)
            
            next.ServeHTTP(wrapped, r.WithContext(ctx))
            
            logger.With("method", r.Method).
                With("path", r.URL.Path).
                With("status", wrapped.status).
                With("duration_ms", time.Since(start).Milliseconds()).
                Info("HTTP request completed")
        })
    }
}
```

```go
// libs/go/http/health/health.go
package health

import (
    "encoding/json"
    "net/http"
    "sync"
)

// Status represents health check status
type Status string

const (
    StatusHealthy   Status = "healthy"
    StatusUnhealthy Status = "unhealthy"
    StatusDegraded  Status = "degraded"
)

// Check is a health check function
type Check func() Status

// Handler provides health check endpoints
type Handler struct {
    mu     sync.RWMutex
    checks map[string]Check
}

// NewHandler creates a new health handler
func NewHandler() *Handler {
    return &Handler{checks: make(map[string]Check)}
}

// Register adds a health check
func (h *Handler) Register(name string, check Check) {
    h.mu.Lock()
    defer h.mu.Unlock()
    h.checks[name] = check
}

// LivenessHandler returns the liveness probe handler
func (h *Handler) LivenessHandler() http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(map[string]string{"status": "alive"})
    }
}

// ReadinessHandler returns the readiness probe handler
func (h *Handler) ReadinessHandler() http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        h.mu.RLock()
        defer h.mu.RUnlock()
        
        results := make(map[string]Status)
        allHealthy := true
        
        for name, check := range h.checks {
            status := check()
            results[name] = status
            if status != StatusHealthy {
                allHealthy = false
            }
        }
        
        if allHealthy {
            w.WriteHeader(http.StatusOK)
        } else {
            w.WriteHeader(http.StatusServiceUnavailable)
        }
        
        json.NewEncoder(w).Encode(map[string]interface{}{
            "status": results,
        })
    }
}
```

### 7. Configuration Library (`libs/go/config/`)

```go
// libs/go/config/config.go
package config

import (
    "os"
    "gopkg.in/yaml.v3"
)

// Loader loads configuration from multiple sources
type Loader struct {
    defaults map[string]interface{}
    file     map[string]interface{}
    env      map[string]interface{}
}

// NewLoader creates a new configuration loader
func NewLoader() *Loader {
    return &Loader{
        defaults: make(map[string]interface{}),
        file:     make(map[string]interface{}),
        env:      make(map[string]interface{}),
    }
}

// WithDefaults sets default values
func (l *Loader) WithDefaults(defaults map[string]interface{}) *Loader {
    l.defaults = defaults
    return l
}

// LoadFile loads configuration from a YAML file
func (l *Loader) LoadFile(path string) error {
    data, err := os.ReadFile(path)
    if err != nil {
        return err
    }
    return yaml.Unmarshal(data, &l.file)
}

// LoadEnv loads configuration from environment variables with prefix
func (l *Loader) LoadEnv(prefix string) {
    for _, env := range os.Environ() {
        if strings.HasPrefix(env, prefix) {
            parts := strings.SplitN(env, "=", 2)
            key := strings.TrimPrefix(parts[0], prefix+"_")
            l.env[strings.ToLower(key)] = parts[1]
        }
    }
}

// Get retrieves a configuration value (env > file > defaults)
func (l *Loader) Get(key string) interface{} {
    if v, ok := l.env[key]; ok {
        return v
    }
    if v, ok := l.file[key]; ok {
        return v
    }
    return l.defaults[key]
}

// Validate validates configuration against required keys
func (l *Loader) Validate(required []string) error {
    var missing []string
    for _, key := range required {
        if l.Get(key) == nil {
            missing = append(missing, key)
        }
    }
    if len(missing) > 0 {
        return fmt.Errorf("missing required configuration: %v", missing)
    }
    return nil
}
```

### 8. Pagination Library (`libs/go/pagination/`)

```go
// libs/go/pagination/pagination.go
package pagination

import (
    "encoding/base64"
    "encoding/json"
)

// Page represents pagination parameters
type Page struct {
    Offset int
    Limit  int
}

// Cursor represents an opaque cursor for cursor-based pagination
type Cursor struct {
    ID        string `json:"id"`
    Timestamp int64  `json:"ts"`
}

// PageResult contains pagination metadata
type PageResult[T any] struct {
    Items       []T    `json:"items"`
    TotalCount  int64  `json:"total_count"`
    HasNext     bool   `json:"has_next"`
    HasPrevious bool   `json:"has_previous"`
    NextCursor  string `json:"next_cursor,omitempty"`
}

// EncodeCursor encodes a cursor to a string
func EncodeCursor(c Cursor) string {
    data, _ := json.Marshal(c)
    return base64.URLEncoding.EncodeToString(data)
}

// DecodeCursor decodes a cursor from a string
func DecodeCursor(s string) (Cursor, error) {
    data, err := base64.URLEncoding.DecodeString(s)
    if err != nil {
        return Cursor{}, err
    }
    var c Cursor
    err = json.Unmarshal(data, &c)
    return c, err
}

// NewPage creates a new Page with validation
func NewPage(offset, limit int) (Page, error) {
    if offset < 0 {
        return Page{}, errors.New("offset must be non-negative")
    }
    if limit <= 0 || limit > 1000 {
        return Page{}, errors.New("limit must be between 1 and 1000")
    }
    return Page{Offset: offset, Limit: limit}, nil
}
```

### 9. Cache Library Enhancement (`libs/go/utils/cache/`)

```go
// libs/go/utils/cache/lru.go
package cache

import (
    "container/list"
    "sync"
    "time"
)

// LRUCache is a thread-safe LRU cache with TTL support
type LRUCache[K comparable, V any] struct {
    mu       sync.RWMutex
    capacity int
    ttl      time.Duration
    items    map[K]*list.Element
    order    *list.List
    stats    Stats
}

// Stats contains cache statistics
type Stats struct {
    Hits      int64
    Misses    int64
    Evictions int64
}

type entry[K comparable, V any] struct {
    key       K
    value     V
    expiresAt time.Time
}

// NewLRUCache creates a new LRU cache
func NewLRUCache[K comparable, V any](capacity int, ttl time.Duration) *LRUCache[K, V] {
    return &LRUCache[K, V]{
        capacity: capacity,
        ttl:      ttl,
        items:    make(map[K]*list.Element),
        order:    list.New(),
    }
}

// Get retrieves a value from the cache
func (c *LRUCache[K, V]) Get(key K) (V, bool) {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    if elem, ok := c.items[key]; ok {
        e := elem.Value.(*entry[K, V])
        if time.Now().Before(e.expiresAt) {
            c.order.MoveToFront(elem)
            c.stats.Hits++
            return e.value, true
        }
        // Expired, remove it
        c.removeElement(elem)
    }
    
    c.stats.Misses++
    var zero V
    return zero, false
}

// Set adds or updates a value in the cache
func (c *LRUCache[K, V]) Set(key K, value V) {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    if elem, ok := c.items[key]; ok {
        c.order.MoveToFront(elem)
        e := elem.Value.(*entry[K, V])
        e.value = value
        e.expiresAt = time.Now().Add(c.ttl)
        return
    }
    
    // Evict if at capacity
    for c.order.Len() >= c.capacity {
        c.evictOldest()
    }
    
    e := &entry[K, V]{
        key:       key,
        value:     value,
        expiresAt: time.Now().Add(c.ttl),
    }
    elem := c.order.PushFront(e)
    c.items[key] = elem
}

// Stats returns cache statistics
func (c *LRUCache[K, V]) Stats() Stats {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return c.stats
}

func (c *LRUCache[K, V]) evictOldest() {
    elem := c.order.Back()
    if elem != nil {
        c.removeElement(elem)
        c.stats.Evictions++
    }
}

func (c *LRUCache[K, V]) removeElement(elem *list.Element) {
    c.order.Remove(elem)
    e := elem.Value.(*entry[K, V])
    delete(c.items, e.key)
}
```

### 10. Worker Pool Library (`libs/go/workerpool/`)

```go
package workerpool

// Job represents a unit of work with priority
type Job[T any] struct {
    ID       string
    Priority Priority // Low=0, Normal=1, High=2
    Payload  T
    Status   JobStatus // pending, running, completed, failed
    Retries  int
}

// WorkerPool manages concurrent job processing
type WorkerPool[T any] struct {
    workers    int
    queue      chan *Job[T]
    deadLetter chan *Job[T]
    handler    func(context.Context, T) error
    metrics    *PoolMetrics
}

func NewWorkerPool[T any](workers int, handler func(context.Context, T) error) *WorkerPool[T]
func (p *WorkerPool[T]) Submit(job *Job[T]) error
func (p *WorkerPool[T]) Shutdown(timeout time.Duration) error
```

### 11. Distributed Lock Library (`libs/go/lock/`)

```go
package lock

// Lock interface for distributed locking
type Lock interface {
    Acquire(ctx context.Context) error
    Release(ctx context.Context) error
    Renew(ctx context.Context) error
    Token() string // Fencing token
}

// RedisLock implements Lock with Redis backend
type RedisLock struct { /* TTL, retry config */ }

// EtcdLock implements Lock with etcd backend
type EtcdLock struct { /* lease-based locking */ }

func TryLock(ctx context.Context, lock Lock, timeout time.Duration) error
```

### 12. Feature Flags Library (`libs/go/featureflags/`)

```go
package featureflags

// Flag with targeting and rollout
type Flag struct {
    Key           string
    Enabled       bool
    Percentage    int      // 0-100 for gradual rollout
    TargetUsers   []string
    TargetTenants []string
}

// Client evaluates flags with caching and fallback
type Client struct { /* flags, defaults, source */ }

func (c *Client) IsEnabled(ctx context.Context, key string) bool
func (c *Client) WithOverride(key string, value bool) *Client // For testing
```

### 13. Metrics Library (`libs/go/metrics/`)

```go
package metrics

// Prometheus/OpenTelemetry compatible metrics
type Counter interface { Inc(); Add(float64); WithLabels(map[string]string) Counter }
type Gauge interface { Set(float64); Inc(); Dec(); WithLabels(map[string]string) Gauge }
type Histogram interface { Observe(float64); WithLabels(map[string]string) Histogram }

type Registry struct { /* thread-safe metric storage */ }

func (r *Registry) NewCounter(name, help string, labels []string) Counter
func (r *Registry) NewHistogram(name, help string, buckets []float64) Histogram
func (r *Registry) Handler() http.Handler // /metrics endpoint
```

### 14. HTTP Client Library (`libs/go/httpclient/`)

```go
package httpclient

// Client with built-in resilience and observability
type Client struct {
    httpClient     *http.Client
    timeout        time.Duration
    retryConfig    RetryConfig
    circuitBreaker CircuitBreaker
    interceptors   []Interceptor
}

func NewClient(opts ...Option) *Client
func (c *Client) Do(ctx context.Context, req *http.Request) (*http.Response, error)
// Auto: correlation ID, trace context, retry, circuit breaker
```

### 15. Database Utilities Library (`libs/go/database/`)

```go
package database

// DB with tracing and transaction management
type DB struct { *sql.DB; tracer Tracer }

func (db *DB) WithTransaction(ctx context.Context, fn func(*Tx) error) error
// Auto-rollback on error, auto-commit on success

// QueryBuilder with SQL injection prevention
func Select(columns ...string) *QueryBuilder
func (qb *QueryBuilder) Where(condition string, args ...interface{}) *QueryBuilder
```

### 16. Event Bus Library (`libs/go/events/`)

```go
package events

// Event with topic routing
type Event struct {
    ID, Topic string
    Payload   interface{}
    Timestamp time.Time
    Metadata  map[string]string
}

type Bus struct { /* handlers, deadLetter, retryConfig */ }

func (b *Bus) Subscribe(topic string, handler Handler)
func (b *Bus) Publish(ctx context.Context, event *Event) error
// Async delivery, retry with backoff, dead letter queue
```

### 17. Outbox Pattern Library (`libs/go/outbox/`)

```go
package outbox

// OutboxEntry for transactional messaging
type OutboxEntry struct {
    ID, AggregateID, AggregateType, EventType string
    Payload        []byte
    IdempotencyKey string
    CreatedAt      time.Time
    ProcessedAt    *time.Time
}

type Outbox struct { db *sql.DB; publisher Publisher }

func (o *Outbox) Store(ctx context.Context, tx *sql.Tx, entry *OutboxEntry) error
func (o *Outbox) ProcessPending(ctx context.Context) error // Background publisher
```

### 18. Idempotency Library (`libs/go/idempotency/`)

```go
package idempotency

// Store for idempotency keys (Redis backend)
type Store interface {
    Get(ctx context.Context, key string) (*Response, error)
    Set(ctx context.Context, key string, resp *Response, ttl time.Duration) error
    Lock(ctx context.Context, key string) (bool, error)
}

func Middleware(store Store, ttl time.Duration) func(http.Handler) http.Handler
// Extracts Idempotency-Key header, caches responses
```

### 19. API Versioning Library (`libs/go/versioning/`)

```go
package versioning

// Version with deprecation support
type Version struct {
    Major, Minor int
    Deprecated   bool
    SunsetDate   *time.Time
}

func Middleware(supported []Version) func(http.Handler) http.Handler
// Extracts version from URL (/v1/) or header (API-Version)
// Adds Deprecation and Sunset headers for deprecated versions
```

### 20. Structured Concurrency Library (`libs/go/concurrency/`)

```go
package concurrency

// TaskGroup with error propagation and panic recovery
type TaskGroup struct { ctx context.Context; cancel context.CancelFunc }

func NewTaskGroup(ctx context.Context) *TaskGroup
func (g *TaskGroup) Go(fn func(context.Context) error)
func (g *TaskGroup) Wait() error // Cancels all on first error

// FanOut with rate limiting
func FanOut[T, R any](ctx context.Context, items []T, concurrency int, 
    fn func(context.Context, T) (R, error)) ([]R, error)
```

## Data Models

### Error Types

```go
type ErrorCode string
const (
    ErrCodeValidation, ErrCodeNotFound, ErrCodeUnauthorized ErrorCode = "VALIDATION_ERROR", "NOT_FOUND", "UNAUTHORIZED"
    ErrCodeForbidden, ErrCodeInternal, ErrCodeTimeout       ErrorCode = "FORBIDDEN", "INTERNAL_ERROR", "TIMEOUT"
    ErrCodeRateLimit, ErrCodeCircuitOpen                    ErrorCode = "RATE_LIMITED", "CIRCUIT_OPEN"
)
```

### Domain Types

```go
type Email struct { value string }
type UUID struct { bytes [16]byte }
type Money struct { amount *big.Int; currency Currency }
type PhoneNumber struct { value string }
type Timestamp struct { value time.Time }
```



## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system—essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*

Based on the prework analysis, the following correctness properties have been identified for property-based testing:

### Property 1: Domain Primitive Validation Consistency

*For any* domain primitive type (Email, UUID, ULID, Money, PhoneNumber, URL, Timestamp, Duration) and *for any* input string, if the input is valid according to the type's specification, the constructor SHALL return a valid instance; if the input is invalid, the constructor SHALL return an error.

**Validates: Requirements 2.1, 2.2, 2.3, 2.4, 2.5, 2.6, 2.7, 2.8**

### Property 2: Domain Primitive Serialization Round-Trip

*For any* valid domain primitive value, serializing to JSON and then deserializing SHALL produce a value equivalent to the original.

**Validates: Requirements 2.9**

### Property 3: Error Chain Preservation

*For any* sequence of error wrappings (1 to N levels deep), unwrapping the error chain SHALL reveal all intermediate errors in order, with the root cause accessible at the end.

**Validates: Requirements 3.2, 3.3, 3.7**

### Property 4: Error Code to Status Mapping Consistency

*For any* error code, mapping to HTTP status code and then to gRPC status code SHALL produce consistent, non-conflicting status codes that correctly represent the error category.

**Validates: Requirements 3.4, 3.5**

### Property 5: Error Serialization Redaction

*For any* error containing internal details (stack traces, internal paths, database queries), serializing for API response SHALL NOT include these internal details while preserving the user-facing message.

**Validates: Requirements 3.8**

### Property 6: Validation Error Completeness

*For any* input with multiple validation failures, the validation result SHALL contain ALL validation errors, not just the first one encountered.

**Validates: Requirements 4.7**

### Property 7: Nested Validation Field Paths

*For any* nested structure with validation errors at various depths, each validation error SHALL include the complete field path from root to the invalid field.

**Validates: Requirements 4.8**

### Property 8: Codec Round-Trip Consistency

*For any* serializable type and *for any* valid instance of that type, encoding to JSON/YAML and then decoding SHALL produce a value equivalent to the original.

**Validates: Requirements 5.6**

### Property 9: Log Entry Timestamp Format

*For any* log entry created by the Observability library, the timestamp field SHALL be in ISO 8601 UTC format (RFC 3339).

**Validates: Requirements 6.7**

### Property 10: Trace Context Propagation

*For any* span created with a parent context, the child span SHALL inherit the trace ID and have the parent span ID set correctly.

**Validates: Requirements 6.8**

### Property 11: PII Redaction in Logs

*For any* log entry containing fields with sensitive key names (password, token, secret, key, authorization, ssn, credit_card), the values SHALL be redacted to "[REDACTED]".

**Validates: Requirements 6.9**

### Property 12: Random Token Uniqueness

*For any* batch of N generated random tokens (where N >= 100), all tokens SHALL be unique (no collisions).

**Validates: Requirements 7.8**

### Property 13: Request Timeout Enforcement

*For any* request with a timeout configured, if the operation exceeds the timeout duration, the context SHALL be cancelled and a timeout error SHALL be returned.

**Validates: Requirements 8.8**

### Property 14: Test Data Validity

*For any* domain object generated by the Testing library's generators, the generated object SHALL pass validation when validated by the corresponding validator.

**Validates: Requirements 9.8**

### Property 15: Configuration Error Completeness

*For any* configuration with multiple missing required values, the validation error SHALL list ALL missing values, not just the first one.

**Validates: Requirements 10.7, 10.8**

### Property 16: Circuit Breaker State Transitions

*For any* circuit breaker in the OPEN state, all execution attempts SHALL fail immediately with a CircuitOpenError without invoking the underlying operation.

**Validates: Requirements 11.8**

### Property 17: Rate Limiter Error Information

*For any* request that exceeds the rate limit, the returned RateLimitError SHALL include a non-zero retry-after duration.

**Validates: Requirements 11.9**

### Property 18: Resilience Configuration Validation

*For any* resilience component configuration with invalid parameters (negative thresholds, zero timeouts, etc.), the configuration validation SHALL reject it with a descriptive error.

**Validates: Requirements 11.10**

### Property 19: Pagination Parameter Validation

*For any* pagination parameters with invalid values (negative offset, zero or negative limit, limit > max), the pagination library SHALL return a validation error.

**Validates: Requirements 12.6**

### Property 20: Cursor Encoding Round-Trip

*For any* valid cursor value, encoding to string and then decoding SHALL produce a cursor equivalent to the original.

**Validates: Requirements 12.7**

### Property 21: Context Value Preservation

*For any* set of context values (correlation ID, user ID, tenant ID, roles), after propagation through HTTP headers or gRPC metadata, all values SHALL be retrievable and equal to the original values.

**Validates: Requirements 13.6**

### Property 22: Cache LRU Eviction Policy

*For any* LRU cache at capacity, inserting a new entry SHALL evict the least recently used entry, and the evicted entry SHALL NOT be retrievable.

**Validates: Requirements 14.7**

### Property 23: Cache TTL Expiration

*For any* cache entry with a TTL, after the TTL has expired, attempting to retrieve the entry SHALL return a cache miss (not found).

**Validates: Requirements 14.8**

### Property 24: Worker Pool Panic Recovery

*For any* worker pool, when a worker panics during job processing, the pool SHALL recover, continue processing other jobs, and move the failed job to the dead letter queue.

**Validates: Requirements 19.8**

### Property 25: Worker Pool Graceful Shutdown

*For any* worker pool with in-flight jobs, when shutdown is initiated, the pool SHALL complete all in-flight jobs before terminating (within timeout).

**Validates: Requirements 19.9**

### Property 26: Distributed Lock TTL Expiration

*For any* distributed lock with a TTL, after the TTL expires without renewal, another process SHALL be able to acquire the lock.

**Validates: Requirements 20.8, 20.9**

### Property 27: Feature Flag Default Fallback

*For any* feature flag key that does not exist in the configuration, the client SHALL return the configured default value.

**Validates: Requirements 21.8**

### Property 28: Feature Flag Remote Fallback

*For any* feature flag client, when remote configuration fails, the client SHALL fall back to cached values without error.

**Validates: Requirements 21.9**

### Property 29: Metrics Thread Safety

*For any* metric (counter, gauge, histogram), concurrent increments/observations from multiple goroutines SHALL produce correct aggregate values.

**Validates: Requirements 22.9**

### Property 30: HTTP Client Error Typing

*For any* failed HTTP request, the returned error SHALL include the HTTP status code and response body (if available).

**Validates: Requirements 23.9**

### Property 31: Database Transaction Auto-Rollback

*For any* database transaction where the operation function returns an error, the transaction SHALL be automatically rolled back.

**Validates: Requirements 24.8**

### Property 32: Event Bus Subscriber Recovery

*For any* event bus subscriber that panics, the bus SHALL recover and continue delivering events to other subscribers.

**Validates: Requirements 25.8**

### Property 33: Outbox At-Least-Once Delivery

*For any* outbox entry, the publisher SHALL retry with exponential backoff until successful delivery or max retries exceeded.

**Validates: Requirements 28.7**

### Property 34: Idempotency Duplicate Response

*For any* request with an idempotency key that was previously processed, the library SHALL return the cached response without re-executing the operation.

**Validates: Requirements 29.6**

### Property 35: API Version Deprecation Headers

*For any* request using a deprecated API version, the response SHALL include Deprecation and Sunset headers.

**Validates: Requirements 27.6**

### Property 36: Task Group Error Cancellation

*For any* task group where one task fails, all remaining tasks SHALL be cancelled via context cancellation.

**Validates: Requirements 30.7**

### Property 37: Task Group Panic Recovery

*For any* task group where a task panics, the panic SHALL be converted to an error and propagated to the Wait() caller.

**Validates: Requirements 30.3**

## Error Handling

### Error Categories

| Category | HTTP Status | gRPC Code | Description |
|----------|-------------|-----------|-------------|
| Validation | 400 | InvalidArgument | Input validation failed |
| NotFound | 404 | NotFound | Resource not found |
| Unauthorized | 401 | Unauthenticated | Authentication required |
| Forbidden | 403 | PermissionDenied | Insufficient permissions |
| RateLimit | 429 | ResourceExhausted | Rate limit exceeded |
| Timeout | 504 | DeadlineExceeded | Operation timed out |
| CircuitOpen | 503 | Unavailable | Circuit breaker open |
| Internal | 500 | Internal | Internal server error |

### Error Response Format

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid input provided",
    "details": [
      {
        "field": "email",
        "message": "Invalid email format",
        "code": "INVALID_FORMAT"
      }
    ],
    "request_id": "req-123-456"
  }
}
```

### Error Handling Patterns

1. **Wrap errors with context**: Always wrap errors with additional context when propagating up the call stack
2. **Use typed errors**: Use typed error codes instead of string matching
3. **Preserve error chains**: Never discard the original error when wrapping
4. **Redact sensitive data**: Never expose internal details in API responses
5. **Log with correlation**: Always include correlation ID in error logs

## Testing Strategy

### Dual Testing Approach

The testing strategy combines unit tests and property-based tests for comprehensive coverage:

1. **Unit Tests**: Verify specific examples, edge cases, and error conditions
2. **Property-Based Tests**: Verify universal properties across all valid inputs

### Property-Based Testing Configuration

- **Library**: Go: `github.com/leanovate/gopter`, Rust: `proptest`
- **Minimum iterations**: 100 per property test
- **Shrinking**: Enabled for minimal failing examples
- **Seed**: Configurable for reproducibility

### Test Organization

```
libs/go/
├── domain/
│   ├── email.go
│   ├── email_test.go          # Unit tests
│   └── email_property_test.go # Property tests
├── utils/
│   ├── error/
│   │   ├── error.go
│   │   ├── error_test.go
│   │   └── error_property_test.go
```

### Property Test Annotation Format

Each property test must be annotated with the design document property it validates:

```go
// Feature: libs-extraction-enhancement, Property 2: Domain Primitive Serialization Round-Trip
// Validates: Requirements 2.9
func TestEmailSerializationRoundTrip(t *testing.T) {
    properties := gopter.NewProperties(gopter.DefaultTestParameters())
    
    properties.Property("email serialization round-trip", prop.ForAll(
        func(email domain.Email) bool {
            data, err := json.Marshal(email)
            if err != nil {
                return false
            }
            var decoded domain.Email
            if err := json.Unmarshal(data, &decoded); err != nil {
                return false
            }
            return email.String() == decoded.String()
        },
        genValidEmail(),
    ))
    
    properties.TestingRun(t)
}
```

### Test Coverage Requirements

| Component | Unit Test Coverage | Property Tests |
|-----------|-------------------|----------------|
| Domain Primitives | 80%+ | All validation, round-trip |
| Error Handling | 80%+ | Chain preservation, mapping |
| Validation | 80%+ | Completeness, field paths |
| Codec | 80%+ | Round-trip for all formats |
| Observability | 70%+ | Timestamp format, redaction |
| Security | 80%+ | Token uniqueness |
| Resilience | 80%+ | State transitions, errors |
| Pagination | 80%+ | Cursor round-trip |
| Cache | 80%+ | LRU eviction, TTL |

### Test Generators

Custom generators for domain types:

```go
// genValidEmail generates valid email addresses
func genValidEmail() gopter.Gen {
    return gopter.CombineGens(
        gen.AlphaNumString().SuchThat(func(s string) bool { return len(s) > 0 }),
        gen.AlphaNumString().SuchThat(func(s string) bool { return len(s) > 0 }),
        gen.OneConstOf("com", "org", "net", "io"),
    ).Map(func(parts []interface{}) domain.Email {
        local := parts[0].(string)
        domain := parts[1].(string)
        tld := parts[2].(string)
        email, _ := domain.NewEmail(fmt.Sprintf("%s@%s.%s", local, domain, tld))
        return email.Unwrap()
    })
}

// genInvalidEmail generates invalid email addresses
func genInvalidEmail() gopter.Gen {
    return gen.OneGenOf(
        gen.Const(""),                    // Empty
        gen.Const("no-at-sign"),          // Missing @
        gen.Const("@no-local"),           // Missing local part
        gen.Const("no-domain@"),          // Missing domain
        gen.Const("spaces in@email.com"), // Spaces
    )
}
```

## Implementation Notes

### Go Library Dependencies

| Library | Purpose | Version |
|---------|---------|---------|
| `go.opentelemetry.io/otel` | Tracing | v1.21+ |
| `github.com/leanovate/gopter` | Property testing | v0.2+ |
| `gopkg.in/yaml.v3` | YAML parsing | v3.0+ |
| `golang.org/x/crypto` | Cryptography | latest |

### Rust Library Dependencies

| Crate | Purpose | Version |
|-------|---------|---------|
| `thiserror` | Error handling | 1.0+ |
| `serde` | Serialization | 1.0+ |
| `tracing` | Observability | 0.1+ |
| `proptest` | Property testing | 1.0+ |
| `tower` | HTTP middleware | 0.4+ |
| `tonic` | gRPC | 0.10+ |

### Migration Strategy

1. **Phase 1**: Create new library packages without breaking existing code
2. **Phase 2**: Add deprecation warnings to old packages
3. **Phase 3**: Update services to use new libraries
4. **Phase 4**: Remove deprecated packages

### Backward Compatibility

- All new libraries will be additive (no breaking changes to existing APIs)
- Deprecated packages will remain functional for at least 2 release cycles
- Migration guides will be provided in each library's README
