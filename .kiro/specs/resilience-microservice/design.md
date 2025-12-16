# Design Document: Resilience Microservice

## Overview

The Resilience Microservice is a state-of-the-art (2025) centralized resilience layer for the Auth Platform, providing advanced resilience patterns including circuit breaker, retry with hedging, timeout, adaptive rate limiting (GCRA), bulkhead isolation, and priority-based load shedding. Built in Go using **failsafe-go** library for battle-tested resilience primitives, it integrates with the existing Rust and Elixir services through gRPC, maintaining consistency with the platform's Zero Trust architecture.

### State of the Art 2025 Features

Based on industry research and best practices from Netflix, Google, Uber, and AWS:

1. **Adaptive Circuit Breaker** - Dynamic threshold adjustment based on error rate trends
2. **Request Hedging** - Parallel requests to reduce tail latency (Google's "The Tail at Scale")
3. **GCRA Rate Limiting** - Generic Cell Rate Algorithm for precise, memory-efficient rate limiting
4. **Priority-Based Load Shedding** - Netflix-style service-level prioritization during overload
5. **Adaptive Throttling** - Automatic adjustment based on downstream service health
6. **Chaos Engineering Ready** - Built-in fault injection for Litmus/Gremlin integration

### Design Goals

- **Sub-50ms latency overhead** for resilience operations
- **Horizontal scalability** with distributed state coordination
- **Hot-reloadable policies** without service restart
- **Comprehensive observability** with OpenTelemetry integration
- **Zero Trust security** leveraging existing Linkerd service mesh for mTLS

### Service Mesh Integration

The Auth Platform already has **Linkerd** configured as the service mesh, providing:
- Automatic mTLS between all services (no application-level TLS needed)
- Golden metrics (latency, success rate, request volume)
- W3C Trace Context propagation
- Certificate rotation via cert-manager

The Resilience Microservice will integrate with Linkerd by:
1. Being injected with the Linkerd proxy sidecar
2. Leveraging mesh-level mTLS instead of implementing application-level mTLS
3. Complementing Linkerd metrics with resilience-specific metrics
4. Propagating trace context through resilience operations

### Technology Stack (2025 State of the Art)

| Component | Technology | Rationale |
|-----------|------------|-----------|
| Language | Go 1.22+ | Excellent concurrency, low latency, strong ecosystem |
| Resilience Library | failsafe-go | Production-proven, supports all patterns including hedging |
| Rate Limiting | GCRA (Generic Cell Rate Algorithm) | Memory-efficient, precise, used by Let's Encrypt |
| Distributed State | Redis 7+ with Lua scripts | Atomic operations, cluster support |
| Observability | OpenTelemetry 1.x | Vendor-neutral, W3C Trace Context |
| gRPC | gRPC-Go 1.60+ | High performance, streaming support |
| Configuration | YAML + Hot-reload | Dapr-style declarative policies |
| Chaos Testing | Litmus ChaosHub integration | CNCF-hosted, Kubernetes-native |

## Architecture

```
                              ┌─────────────────────────────────────┐
                              │         External Clients            │
                              └─────────────────┬───────────────────┘
                                                │
                              ┌─────────────────▼───────────────────┐
                              │     Envoy Gateway (Gateway API)     │
                              │         North-South Traffic         │
                              │  • TLS Termination • Rate Limiting  │
                              │  • Authentication  • Routing        │
                              └─────────────────┬───────────────────┘
                                                │
┌───────────────────────────────────────────────▼───────────────────────────────────────────────┐
│                                    Service Mesh (Linkerd)                                      │
│                                     East-West Traffic + mTLS                                   │
├───────────────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                                │
│  ┌─────────────────────────────────────────────────────────────────────────────────────────┐  │
│  │                           Resilience Microservice (Go)                                   │  │
│  ├─────────────────────────────────────────────────────────────────────────────────────────┤  │
│  │                                                                                          │  │
│  │  ┌────────────────────────────────────────────────────────────────────────────────────┐ │  │
│  │  │                            gRPC Service Layer                                       │ │  │
│  │  │    ExecuteWithResilience │ GetCircuitState │ UpdatePolicy │ HealthCheck            │ │  │
│  │  └────────────────────────────────────────────────────────────────────────────────────┘ │  │
│  │                                          │                                              │  │
│  │  ┌───────────────────────────────────────▼────────────────────────────────────────────┐ │  │
│  │  │                          Policy Engine                                              │ │  │
│  │  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐                │ │  │
│  │  │  │   Parser    │  │  Validator  │  │   Store     │  │  Hot-Reload │                │ │  │
│  │  │  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘                │ │  │
│  │  └────────────────────────────────────────────────────────────────────────────────────┘ │  │
│  │                                          │                                              │  │
│  │  ┌───────────────────────────────────────▼────────────────────────────────────────────┐ │  │
│  │  │                      Resilience Patterns Layer                                      │ │  │
│  │  │  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐ ┌──────────────┐               │ │  │
│  │  │  │   Circuit    │ │    Retry     │ │   Timeout    │ │    Rate      │               │ │  │
│  │  │  │   Breaker    │ │   Handler    │ │   Manager    │ │   Limiter    │               │ │  │
│  │  │  │  • State FSM │ │ • Exp Backoff│ │ • Context    │ │ • Token Bucket│              │ │  │
│  │  │  │  • Probes    │ │ • Jitter     │ │ • Deadline   │ │ • Sliding Win │              │ │  │
│  │  │  └──────────────┘ └──────────────┘ └──────────────┘ └──────────────┘               │ │  │
│  │  │  ┌──────────────┐ ┌──────────────┐                                                 │ │  │
│  │  │  │   Bulkhead   │ │   Health     │                                                 │ │  │
│  │  │  │   Isolator   │ │  Aggregator  │                                                 │ │  │
│  │  │  │  • Semaphore │ │ • Probes     │                                                 │ │  │
│  │  │  │  • Partitions│ │ • CAEP Emit  │                                                 │ │  │
│  │  │  └──────────────┘ └──────────────┘                                                 │ │  │
│  │  └────────────────────────────────────────────────────────────────────────────────────┘ │  │
│  │                                          │                                              │  │
│  │  ┌───────────────────────────────────────▼────────────────────────────────────────────┐ │  │
│  │  │                        Infrastructure Clients                                       │ │  │
│  │  │  ┌──────────┐  ┌──────────┐  ┌──────────────┐  ┌──────────┐                        │ │  │
│  │  │  │  Redis   │  │  Vault   │  │ OpenTelemetry│  │   CAEP   │                        │ │  │
│  │  │  │  Client  │  │  Client  │  │   Provider   │  │  Emitter │                        │ │  │
│  │  │  └──────────┘  └──────────┘  └──────────────┘  └──────────┘                        │ │  │
│  │  └────────────────────────────────────────────────────────────────────────────────────┘ │  │
│  └─────────────────────────────────────────────────────────────────────────────────────────┘  │
│                                              │                                                │
│              ┌───────────────────────────────┼───────────────────────────────┐                │
│              │                               │                               │                │
│              ▼                               ▼                               ▼                │
│  ┌───────────────────┐          ┌───────────────────┐          ┌───────────────────┐         │
│  │   Auth Edge       │          │   Token Service   │          │ Session Identity  │         │
│  │   Service (Rust)  │          │   (Rust)          │          │ Core (Elixir)     │         │
│  │   Port: 50051     │          │   Port: 50052     │          │ Port: 50053       │         │
│  └───────────────────┘          └───────────────────┘          └───────────────────┘         │
│              │                               │                               │                │
│              ▼                               ▼                               ▼                │
│  ┌───────────────────┐          ┌───────────────────┐          ┌───────────────────┐         │
│  │   IAM Policy      │          │   MFA Service     │          │   Shared Libs     │         │
│  │   Service (Go)    │          │   (Elixir)        │          │   (Go)            │         │
│  │   Port: 50054     │          │   Port: 50055     │          │                   │         │
│  └───────────────────┘          └───────────────────┘          └───────────────────┘         │
│                                                                                               │
└───────────────────────────────────────────────────────────────────────────────────────────────┘
                              │                               │
                              ▼                               ▼
                    ┌─────────────────┐             ┌─────────────────┐
                    │      Redis      │             │  HashiCorp Vault│
                    │  (State Store)  │             │    (Secrets)    │
                    └─────────────────┘             └─────────────────┘
```

### Traffic Flow

1. **North-South**: External clients → Envoy Gateway → Services
2. **East-West**: Service-to-Service via Linkerd mesh with mTLS
3. **Resilience**: Services call Resilience Microservice for protected downstream calls

## Components and Interfaces

### 1. gRPC Service Layer

```protobuf
service ResilienceService {
  // Execute request with resilience policy applied
  rpc ExecuteWithResilience(ExecuteRequest) returns (ExecuteResponse);
  
  // Get current circuit breaker state
  rpc GetCircuitState(CircuitStateRequest) returns (CircuitStateResponse);
  
  // Update resilience policy
  rpc UpdatePolicy(UpdatePolicyRequest) returns (UpdatePolicyResponse);
  
  // Aggregated health check
  rpc HealthCheck(HealthCheckRequest) returns (HealthCheckResponse);
  
  // Stream circuit breaker state changes
  rpc WatchCircuitState(WatchRequest) returns (stream CircuitStateEvent);
}
```

### 2. Circuit Breaker Component

```go
// CircuitBreaker manages state transitions for a protected service
type CircuitBreaker interface {
    // Execute runs the operation with circuit breaker protection
    Execute(ctx context.Context, operation func() error) error
    
    // GetState returns current circuit state
    GetState() CircuitState
    
    // RecordSuccess records a successful operation
    RecordSuccess()
    
    // RecordFailure records a failed operation
    RecordFailure()
    
    // Reset forces circuit to closed state
    Reset()
}

// CircuitState represents the circuit breaker state
type CircuitState int

const (
    StateClosed CircuitState = iota
    StateOpen
    StateHalfOpen
)

// CircuitBreakerConfig defines circuit breaker behavior
type CircuitBreakerConfig struct {
    FailureThreshold  int           `json:"failure_threshold"`
    SuccessThreshold  int           `json:"success_threshold"`
    Timeout           time.Duration `json:"timeout"`
    ProbeCount        int           `json:"probe_count"`
}
```

### 3. Retry Handler Component

```go
// RetryHandler manages retry logic with backoff
type RetryHandler interface {
    // Execute runs operation with retry policy
    Execute(ctx context.Context, operation func() error) error
    
    // CalculateDelay returns next retry delay
    CalculateDelay(attempt int) time.Duration
}

// RetryConfig defines retry behavior
type RetryConfig struct {
    MaxAttempts     int           `json:"max_attempts"`
    BaseDelay       time.Duration `json:"base_delay"`
    MaxDelay        time.Duration `json:"max_delay"`
    Multiplier      float64       `json:"multiplier"`
    JitterPercent   float64       `json:"jitter_percent"`
    RetryableErrors []string      `json:"retryable_errors"`
}

// BackoffStrategy calculates delay between retries
type BackoffStrategy interface {
    NextDelay(attempt int, baseDelay time.Duration) time.Duration
}
```

### 4. Rate Limiter Component

```go
// RateLimiter controls request throughput
type RateLimiter interface {
    // Allow checks if request should be allowed
    Allow(ctx context.Context, key string) (RateLimitDecision, error)
    
    // GetInfo returns current rate limit info
    GetInfo(ctx context.Context, key string) (RateLimitInfo, error)
}

// RateLimitDecision represents allow/deny decision
type RateLimitDecision struct {
    Allowed    bool
    Remaining  int
    ResetAt    time.Time
    RetryAfter time.Duration
}

// RateLimitConfig defines rate limiting behavior
type RateLimitConfig struct {
    Algorithm   RateLimitAlgorithm `json:"algorithm"`
    Limit       int                `json:"limit"`
    Window      time.Duration      `json:"window"`
    BurstSize   int                `json:"burst_size"`
}

type RateLimitAlgorithm string

const (
    TokenBucket   RateLimitAlgorithm = "token_bucket"
    SlidingWindow RateLimitAlgorithm = "sliding_window"
)
```

### 5. Bulkhead Component

```go
// Bulkhead provides isolation through concurrency limits
type Bulkhead interface {
    // Acquire attempts to acquire a permit
    Acquire(ctx context.Context) error
    
    // Release returns a permit
    Release()
    
    // GetMetrics returns current utilization
    GetMetrics() BulkheadMetrics
}

// BulkheadConfig defines bulkhead behavior
type BulkheadConfig struct {
    MaxConcurrent int           `json:"max_concurrent"`
    MaxQueue      int           `json:"max_queue"`
    QueueTimeout  time.Duration `json:"queue_timeout"`
}

// BulkheadMetrics reports bulkhead utilization
type BulkheadMetrics struct {
    ActiveCount   int
    QueuedCount   int
    RejectedCount int64
}
```

### 6. Policy Engine Component

```go
// PolicyEngine manages resilience policies
type PolicyEngine interface {
    // GetPolicy retrieves policy by name
    GetPolicy(name string) (*ResiliencePolicy, error)
    
    // UpdatePolicy updates or creates a policy
    UpdatePolicy(policy *ResiliencePolicy) error
    
    // DeletePolicy removes a policy
    DeletePolicy(name string) error
    
    // WatchPolicies streams policy changes
    WatchPolicies(ctx context.Context) (<-chan PolicyEvent, error)
}

// ResiliencePolicy combines all resilience settings
type ResiliencePolicy struct {
    Name           string               `json:"name"`
    Version        int                  `json:"version"`
    CircuitBreaker *CircuitBreakerConfig `json:"circuit_breaker,omitempty"`
    Retry          *RetryConfig          `json:"retry,omitempty"`
    Timeout        *TimeoutConfig        `json:"timeout,omitempty"`
    RateLimit      *RateLimitConfig      `json:"rate_limit,omitempty"`
    Bulkhead       *BulkheadConfig       `json:"bulkhead,omitempty"`
}
```

### 7. Health Aggregator Component

```go
// HealthAggregator collects and aggregates service health
type HealthAggregator interface {
    // GetAggregatedHealth returns overall system health
    GetAggregatedHealth(ctx context.Context) (*AggregatedHealth, error)
    
    // RegisterService adds a service to monitor
    RegisterService(name string, endpoint string) error
    
    // UnregisterService removes a service from monitoring
    UnregisterService(name string) error
}

// AggregatedHealth represents system-wide health status
type AggregatedHealth struct {
    Status    HealthStatus           `json:"status"`
    Services  map[string]ServiceHealth `json:"services"`
    Timestamp time.Time              `json:"timestamp"`
}

type HealthStatus string

const (
    HealthHealthy   HealthStatus = "healthy"
    HealthDegraded  HealthStatus = "degraded"
    HealthUnhealthy HealthStatus = "unhealthy"
)
```

## Data Models

### Circuit Breaker State

```go
// CircuitBreakerState represents persistent circuit state
type CircuitBreakerState struct {
    ServiceName     string        `json:"service_name"`
    State           CircuitState  `json:"state"`
    FailureCount    int           `json:"failure_count"`
    SuccessCount    int           `json:"success_count"`
    LastFailureTime *time.Time    `json:"last_failure_time,omitempty"`
    LastStateChange time.Time     `json:"last_state_change"`
    Version         int64         `json:"version"`
}
```

### Retry Policy Definition

```go
// RetryPolicyDefinition is the serializable retry configuration
type RetryPolicyDefinition struct {
    MaxAttempts   int      `json:"max_attempts" validate:"min=1,max=10"`
    BaseDelayMs   int      `json:"base_delay_ms" validate:"min=10,max=60000"`
    MaxDelayMs    int      `json:"max_delay_ms" validate:"min=100,max=300000"`
    Multiplier    float64  `json:"multiplier" validate:"min=1.0,max=5.0"`
    JitterPercent float64  `json:"jitter_percent" validate:"min=0,max=0.5"`
    RetryOn       []string `json:"retry_on"`
}
```

### Rate Limit State

```go
// RateLimitState represents distributed rate limit state
type RateLimitState struct {
    Key          string    `json:"key"`
    Tokens       float64   `json:"tokens"`
    LastRefill   time.Time `json:"last_refill"`
    RequestCount int64     `json:"request_count"`
    WindowStart  time.Time `json:"window_start"`
}
```

### Resilience Event

```go
// ResilienceEvent represents an event for observability
type ResilienceEvent struct {
    ID            string                 `json:"id"`
    Type          ResilienceEventType    `json:"type"`
    ServiceName   string                 `json:"service_name"`
    Timestamp     time.Time              `json:"timestamp"`
    CorrelationID string                 `json:"correlation_id"`
    TraceID       string                 `json:"trace_id"`
    SpanID        string                 `json:"span_id"`
    Metadata      map[string]interface{} `json:"metadata"`
}

type ResilienceEventType string

const (
    EventCircuitStateChange ResilienceEventType = "circuit_state_change"
    EventRetryAttempt       ResilienceEventType = "retry_attempt"
    EventTimeout            ResilienceEventType = "timeout"
    EventRateLimitHit       ResilienceEventType = "rate_limit_hit"
    EventBulkheadRejection  ResilienceEventType = "bulkhead_rejection"
)
```



## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system-essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*

Based on the prework analysis, the following correctness properties have been identified. Properties testing similar aspects have been consolidated to eliminate redundancy.

### Property 1: Circuit Breaker State Machine Correctness

*For any* sequence of success/failure events and any valid circuit breaker configuration, the circuit breaker state transitions SHALL follow the correct state machine:
- Closed → Open: when failure count >= failure threshold
- Open → HalfOpen: when timeout elapses
- HalfOpen → Closed: when success count >= success threshold
- HalfOpen → Open: when any failure occurs

**Validates: Requirements 1.1, 1.2, 1.3, 1.4**

### Property 2: Circuit Breaker State Serialization Round-Trip

*For any* valid CircuitBreakerState, serializing to JSON and deserializing back SHALL produce an equivalent state object with all fields preserved.

**Validates: Requirements 1.6**

### Property 3: Circuit State Change Event Emission

*For any* circuit breaker state transition, the system SHALL emit exactly one event containing the service name, previous state, new state, and a valid correlation ID.

**Validates: Requirements 1.5**

### Property 4: Retry Delay with Exponential Backoff and Jitter

*For any* retry attempt number n, base delay b, multiplier m, max delay M, and jitter percentage j, the calculated delay d SHALL satisfy:
- base_delay = min(b * m^n, M)
- d ∈ [base_delay * (1-j), base_delay * (1+j)]

**Validates: Requirements 2.2, 2.3**

### Property 5: Retry Exhaustion Returns Final Error

*For any* operation that fails all retry attempts, the returned error SHALL include retry exhaustion metadata indicating the total attempts made and the final error.

**Validates: Requirements 2.4**

### Property 6: Open Circuit Blocks Retry Attempts

*For any* operation targeting a service with an Open circuit breaker, the retry handler SHALL make zero retry attempts and return a circuit open error immediately.

**Validates: Requirements 2.5**

### Property 7: Retry Policy Configuration Round-Trip

*For any* valid RetryPolicyDefinition, parsing from JSON, pretty-printing, and parsing again SHALL produce an equivalent policy configuration.

**Validates: Requirements 2.6**

### Property 8: Timeout Enforcement

*For any* operation with configured timeout T, if the operation duration exceeds T, the operation SHALL be cancelled and a timeout error returned within T + epsilon (where epsilon accounts for scheduling overhead).

**Validates: Requirements 3.1**

### Property 9: Operation-Specific Timeout Precedence

*For any* operation with both global timeout G and operation-specific timeout O configured, the effective timeout SHALL equal O (operation-specific takes precedence).

**Validates: Requirements 3.2**

### Property 10: Timeout Configuration Validation

*For any* timeout configuration value, the validator SHALL reject values that are non-positive or exceed the maximum allowed duration (e.g., > 5 minutes).

**Validates: Requirements 3.4**

### Property 11: Rate Limit Enforcement

*For any* client that has made L requests where L >= configured limit within the window, the next request SHALL be rejected with a 429 status and a positive Retry-After value.

**Validates: Requirements 4.1**

### Property 12: Token Bucket Invariants

*For any* token bucket rate limiter with capacity C and refill rate R, the token count SHALL:
- Never exceed C (capacity invariant)
- Increase by R tokens per second when below capacity (refill invariant)
- Decrease by 1 for each allowed request (consumption invariant)

**Validates: Requirements 4.2**

### Property 13: Sliding Window Request Counting

*For any* sliding window rate limiter with window W, the request count SHALL only include requests with timestamps within [now - W, now].

**Validates: Requirements 4.3**

### Property 14: Rate Limit Response Headers

*For any* rate limit decision (allow or deny), the response SHALL include all three headers: X-RateLimit-Limit, X-RateLimit-Remaining, and X-RateLimit-Reset with valid values.

**Validates: Requirements 4.5**

### Property 15: Bulkhead Concurrent Request Enforcement

*For any* bulkhead with max concurrent C and max queue Q, the number of active + queued requests SHALL never exceed C + Q, and requests beyond this limit SHALL be rejected with BulkheadFull error.

**Validates: Requirements 5.1, 5.2**

### Property 16: Bulkhead Partition Isolation

*For any* two distinct partition keys, their bulkhead limits SHALL be enforced independently - exhausting one partition's limit SHALL NOT affect the other partition's availability.

**Validates: Requirements 5.3**

### Property 17: Bulkhead Metrics Accuracy

*For any* bulkhead state, the reported metrics (active count, queued count, rejected count) SHALL accurately reflect the actual state of the bulkhead.

**Validates: Requirements 5.4**

### Property 18: Health Aggregation Logic

*For any* set of service health statuses, the aggregated health SHALL be:
- Healthy: if all services are healthy
- Degraded: if any service is degraded but none unhealthy
- Unhealthy: if any service is unhealthy

**Validates: Requirements 6.1**

### Property 19: Health Change Event Emission

*For any* change in a protected service's health status, exactly one CAEP event SHALL be emitted containing the service name, previous status, and new status.

**Validates: Requirements 6.2**

### Property 20: Policy Validation Rejects Invalid Configurations

*For any* policy configuration with invalid values (negative thresholds, zero timeouts, etc.), the validator SHALL reject the policy and return a descriptive error message.

**Validates: Requirements 7.1**

### Property 21: Policy Definition Round-Trip

*For any* valid ResiliencePolicy, serializing to JSON, deserializing, and serializing again SHALL produce identical JSON output.

**Validates: Requirements 7.4**

### Property 22: Circuit State Retrieval Consistency

*For any* GetCircuitState request, the returned state SHALL match the actual internal circuit breaker state at the time of the request.

**Validates: Requirements 8.2**

### Property 23: Error to gRPC Status Code Mapping

*For any* internal error type, the error SHALL map to the correct gRPC status code:
- CircuitOpen → UNAVAILABLE
- RateLimitExceeded → RESOURCE_EXHAUSTED
- Timeout → DEADLINE_EXCEEDED
- BulkheadFull → RESOURCE_EXHAUSTED
- InvalidPolicy → INVALID_ARGUMENT

**Validates: Requirements 8.4**

### Property 24: Audit Event Required Fields

*For any* audit event emitted, the event SHALL contain: event ID, timestamp, correlation ID, event type, and SPIFFE identity (when available).

**Validates: Requirements 9.3**

### Property 25: Graceful Shutdown Request Draining

*For any* graceful shutdown with N in-flight requests, all N requests SHALL complete (success or timeout) before the service terminates.

**Validates: Requirements 10.4**

## Error Handling

### Error Types

```go
// ResilienceError represents errors from resilience operations
type ResilienceError struct {
    Code       ErrorCode              `json:"code"`
    Message    string                 `json:"message"`
    Service    string                 `json:"service,omitempty"`
    RetryAfter time.Duration          `json:"retry_after,omitempty"`
    Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

type ErrorCode string

const (
    ErrCircuitOpen       ErrorCode = "CIRCUIT_OPEN"
    ErrRateLimitExceeded ErrorCode = "RATE_LIMIT_EXCEEDED"
    ErrTimeout           ErrorCode = "TIMEOUT"
    ErrBulkheadFull      ErrorCode = "BULKHEAD_FULL"
    ErrRetryExhausted    ErrorCode = "RETRY_EXHAUSTED"
    ErrInvalidPolicy     ErrorCode = "INVALID_POLICY"
    ErrServiceUnavailable ErrorCode = "SERVICE_UNAVAILABLE"
)
```

### Error to gRPC Status Mapping

| Error Code | gRPC Status | HTTP Status |
|------------|-------------|-------------|
| CIRCUIT_OPEN | UNAVAILABLE | 503 |
| RATE_LIMIT_EXCEEDED | RESOURCE_EXHAUSTED | 429 |
| TIMEOUT | DEADLINE_EXCEEDED | 504 |
| BULKHEAD_FULL | RESOURCE_EXHAUSTED | 503 |
| RETRY_EXHAUSTED | UNAVAILABLE | 503 |
| INVALID_POLICY | INVALID_ARGUMENT | 400 |

### Error Sanitization

All errors returned to clients are sanitized to remove:
- Internal stack traces
- Database connection strings
- Secret values
- Internal service addresses

## Testing Strategy

### Property-Based Testing Framework

The project uses **gopter** (Go Property Testing) for property-based testing with a minimum of 100 iterations per property.

```go
import "github.com/leanovate/gopter"
import "github.com/leanovate/gopter/gen"
import "github.com/leanovate/gopter/prop"
```

### Test Organization

```
resilience-service/
├── internal/
│   ├── circuitbreaker/
│   │   ├── breaker.go
│   │   ├── breaker_test.go        # Unit tests
│   │   └── breaker_prop_test.go   # Property tests
│   ├── retry/
│   │   ├── handler.go
│   │   ├── handler_test.go
│   │   └── handler_prop_test.go
│   ├── ratelimit/
│   │   ├── limiter.go
│   │   ├── limiter_test.go
│   │   └── limiter_prop_test.go
│   ├── bulkhead/
│   │   ├── bulkhead.go
│   │   ├── bulkhead_test.go
│   │   └── bulkhead_prop_test.go
│   └── policy/
│       ├── engine.go
│       ├── engine_test.go
│       └── engine_prop_test.go
└── tests/
    ├── integration/
    └── contract/
```

### Unit Testing Requirements

- Test specific examples and edge cases
- Test error conditions and boundary values
- Use table-driven tests for multiple scenarios
- Mock external dependencies (Redis, Vault)

### Property-Based Testing Requirements

- Each correctness property maps to one property-based test
- Minimum 100 iterations per property test
- Use generators for random valid inputs
- Tag each test with property reference: `// **Feature: resilience-microservice, Property N: description**`

### Test Coverage Targets

| Component | Target Coverage |
|-----------|-----------------|
| Circuit Breaker | 90% |
| Retry Handler | 90% |
| Rate Limiter | 85% |
| Bulkhead | 85% |
| Policy Engine | 90% |
| Health Aggregator | 80% |

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `RESILIENCE_HOST` | Service bind address | `0.0.0.0` |
| `RESILIENCE_PORT` | gRPC port | `50056` |
| `REDIS_URL` | Redis connection URL | `redis://localhost:6379` |
| `VAULT_ADDR` | Vault server address | `http://localhost:8200` |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | OpenTelemetry endpoint | `http://localhost:4317` |
| `LOG_LEVEL` | Logging level | `info` |
| `POLICY_CONFIG_PATH` | Path to policy configuration | `/etc/resilience/policies.yaml` |

### Default Policy Configuration

```yaml
policies:
  default:
    circuit_breaker:
      failure_threshold: 5
      success_threshold: 3
      timeout: 30s
    retry:
      max_attempts: 3
      base_delay: 100ms
      max_delay: 10s
      multiplier: 2.0
      jitter_percent: 0.1
    timeout:
      default: 5s
    rate_limit:
      algorithm: token_bucket
      limit: 1000
      window: 1m
      burst_size: 100
    bulkhead:
      max_concurrent: 100
      max_queue: 50
      queue_timeout: 5s
```
