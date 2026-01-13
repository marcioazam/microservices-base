# Design Document: IAM Policy Service Modernization

## Overview

This design document describes the modernization of the IAM Policy Service to state-of-the-art December 2025 standards. The service provides a Policy Decision Point (PDP) implementing RBAC and ABAC using Open Policy Agent (OPA) v1.0+. The modernization eliminates redundancy, integrates with platform services (logging-service, cache-service), adopts Go 1.24+ features, and achieves production-ready architecture.

### Key Modernization Goals

1. **Zero Redundancy**: Eliminate all duplicated logic, centralize cross-cutting concerns
2. **Platform Integration**: Use shared libs and platform services (cache, logging)
3. **State-of-the-Art Stack**: Go 1.24+, OPA v1.0+, gRPC v1.70+, OpenTelemetry 1.35+
4. **Production Ready**: Health checks, graceful shutdown, circuit breakers, observability
5. **Test Separation**: Source code in `internal/`, tests in `tests/`

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         IAM Policy Service (Go 1.24+)                       │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                        Transport Layer                               │   │
│  │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────────┐  │   │
│  │  │  gRPC Server    │  │  Health HTTP    │  │  Metrics HTTP       │  │   │
│  │  │  :50054         │  │  :8080          │  │  :9090              │  │   │
│  │  │  - Authorize    │  │  - /health/live │  │  - /metrics         │  │   │
│  │  │  - Batch        │  │  - /health/ready│  │                     │  │   │
│  │  │  - Permissions  │  │                 │  │                     │  │   │
│  │  └────────┬────────┘  └─────────────────┘  └─────────────────────┘  │   │
│  └───────────┼──────────────────────────────────────────────────────────┘   │
│              │                                                              │
│  ┌───────────▼──────────────────────────────────────────────────────────┐   │
│  │                      Interceptor Chain                                │   │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐   │   │
│  │  │ Recovery │→│ Logging  │→│ Tracing  │→│ Metrics  │→│ Error    │   │   │
│  │  │          │ │(libs/go) │ │(OTel)    │ │(Prom)    │ │(libs/go) │   │   │
│  │  └──────────┘ └──────────┘ └──────────┘ └──────────┘ └──────────┘   │   │
│  └───────────┬──────────────────────────────────────────────────────────┘   │
│              │                                                              │
│  ┌───────────▼──────────────────────────────────────────────────────────┐   │
│  │                        Service Layer                                  │   │
│  │  ┌─────────────────────────────────────────────────────────────────┐ │   │
│  │  │                    AuthorizationService                          │ │   │
│  │  │  - Authorize(ctx, request) → Decision                           │ │   │
│  │  │  - BatchAuthorize(ctx, requests) → []Decision                   │ │   │
│  │  │  - GetPermissions(ctx, subjectID) → []Permission                │ │   │
│  │  │  - GetRoles(ctx, subjectID) → []Role                            │ │   │
│  │  └─────────────────────────────────────────────────────────────────┘ │   │
│  └───────────┬──────────────────────────────────────────────────────────┘   │
│              │                                                              │
│  ┌───────────▼──────────────────────────────────────────────────────────┐   │
│  │                        Domain Layer                                   │   │
│  │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────────┐   │   │
│  │  │  PolicyEngine   │  │  RoleHierarchy  │  │  DecisionCache      │   │   │
│  │  │  - Evaluate()   │  │  - GetEffective │  │  - Get/Set/Delete   │   │   │
│  │  │  - Reload()     │  │  - AddRole()    │  │  - Invalidate()     │   │   │
│  │  │  - Watch()      │  │  - GetAncestors │  │                     │   │   │
│  │  └─────────────────┘  └─────────────────┘  └─────────────────────┘   │   │
│  └───────────┬──────────────────────────────────────────────────────────┘   │
│              │                                                              │
│  ┌───────────▼──────────────────────────────────────────────────────────┐   │
│  │                     Infrastructure Layer                              │   │
│  │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────────┐   │   │
│  │  │  CacheClient    │  │  LoggingClient  │  │  CAEPEmitter        │   │   │
│  │  │  (libs/go/cache)│  │  (libs/go/log)  │  │  - EmitRoleChange   │   │   │
│  │  │  - Get/Set/Del  │  │  - Info/Error   │  │  - EmitPermChange   │   │   │
│  │  │  - BatchOps     │  │  - WithContext  │  │                     │   │   │
│  │  └─────────────────┘  └─────────────────┘  └─────────────────────┘   │   │
│  └──────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                    ┌───────────────┼───────────────┐
                    ▼               ▼               ▼
            ┌───────────────┐ ┌───────────┐ ┌───────────────┐
            │ cache-service │ │ logging-  │ │ CAEP          │
            │ (platform)    │ │ service   │ │ Transmitter   │
            └───────────────┘ └───────────┘ └───────────────┘
```

## Components and Interfaces

### 1. Configuration (internal/config)

```go
// Config holds all service configuration.
type Config struct {
    Server    ServerConfig
    Policy    PolicyConfig
    Cache     CacheConfig
    Logging   LoggingConfig
    CAEP      CAEPConfig
    Metrics   MetricsConfig
    Tracing   TracingConfig
}

type ServerConfig struct {
    GRPCPort         int           `env:"IAM_POLICY_GRPC_PORT" default:"50054"`
    HealthPort       int           `env:"IAM_POLICY_HEALTH_PORT" default:"8080"`
    MetricsPort      int           `env:"IAM_POLICY_METRICS_PORT" default:"9090"`
    ShutdownTimeout  time.Duration `env:"IAM_POLICY_SHUTDOWN_TIMEOUT" default:"30s"`
}

type PolicyConfig struct {
    Path           string        `env:"IAM_POLICY_PATH" default:"./policies"`
    CacheTTL       time.Duration `env:"IAM_POLICY_CACHE_TTL" default:"5m"`
    WatchEnabled   bool          `env:"IAM_POLICY_WATCH_ENABLED" default:"true"`
}

type CacheConfig struct {
    Address        string        `env:"IAM_POLICY_CACHE_ADDRESS" default:"localhost:50051"`
    Namespace      string        `env:"IAM_POLICY_CACHE_NAMESPACE" default:"iam-policy"`
    LocalFallback  bool          `env:"IAM_POLICY_CACHE_LOCAL_FALLBACK" default:"true"`
    LocalCacheSize int           `env:"IAM_POLICY_CACHE_LOCAL_SIZE" default:"10000"`
    Timeout        time.Duration `env:"IAM_POLICY_CACHE_TIMEOUT" default:"100ms"`
}

type LoggingConfig struct {
    Address       string        `env:"IAM_POLICY_LOGGING_ADDRESS" default:"localhost:50052"`
    ServiceName   string        `env:"IAM_POLICY_SERVICE_NAME" default:"iam-policy-service"`
    MinLevel      string        `env:"IAM_POLICY_LOG_LEVEL" default:"info"`
    LocalFallback bool          `env:"IAM_POLICY_LOG_LOCAL_FALLBACK" default:"true"`
    BufferSize    int           `env:"IAM_POLICY_LOG_BUFFER_SIZE" default:"1000"`
    FlushInterval time.Duration `env:"IAM_POLICY_LOG_FLUSH_INTERVAL" default:"5s"`
}

type CAEPConfig struct {
    Enabled      bool   `env:"IAM_POLICY_CAEP_ENABLED" default:"false"`
    Transmitter  string `env:"IAM_POLICY_CAEP_TRANSMITTER" default:""`
    ServiceToken string `env:"IAM_POLICY_CAEP_SERVICE_TOKEN" default:""`
    Issuer       string `env:"IAM_POLICY_CAEP_ISSUER" default:"iam-policy-service"`
}

// Load loads configuration using libs/go/src/config.
func Load() (*Config, error)
```

### 2. Policy Engine (internal/policy)

```go
// Engine evaluates authorization policies using OPA.
type Engine struct {
    mu           sync.RWMutex
    queries      map[string]*rego.PreparedEvalQuery
    policies     map[string]string
    cache        DecisionCache
    logger       *logging.Client
    metrics      *Metrics
}

// EvaluationResult contains the policy evaluation outcome.
type EvaluationResult struct {
    Allowed      bool
    PolicyID     string
    MatchedRules []string
    Cached       bool
    Duration     time.Duration
}

// NewEngine creates a new policy engine.
func NewEngine(cfg PolicyConfig, cache DecisionCache, logger *logging.Client) (*Engine, error)

// Evaluate evaluates an authorization request against loaded policies.
func (e *Engine) Evaluate(ctx context.Context, input AuthorizationInput) (EvaluationResult, error)

// Reload reloads all policies from disk.
func (e *Engine) Reload(ctx context.Context) error

// Watch starts watching for policy file changes.
func (e *Engine) Watch(ctx context.Context) error

// PolicyCount returns the number of loaded policies.
func (e *Engine) PolicyCount() int
```

### 3. Decision Cache (internal/cache)

```go
// DecisionCache caches authorization decisions.
type DecisionCache interface {
    Get(ctx context.Context, key string) (Decision, bool)
    Set(ctx context.Context, key string, decision Decision, ttl time.Duration) error
    Delete(ctx context.Context, key string) error
    Invalidate(ctx context.Context) error
}

// Decision represents a cached authorization decision.
type Decision struct {
    Allowed      bool      `json:"allowed"`
    PolicyID     string    `json:"policy_id"`
    MatchedRules []string  `json:"matched_rules"`
    CachedAt     time.Time `json:"cached_at"`
}

// CacheKey generates a deterministic cache key from authorization input.
func CacheKey(input AuthorizationInput) string

// NewDecisionCache creates a decision cache backed by cache-service.
func NewDecisionCache(client *cache.Client, ttl time.Duration) DecisionCache
```

### 4. Role Hierarchy (internal/rbac)

```go
// Role represents a role with optional parent for hierarchy.
type Role struct {
    ID          string
    Name        string
    Description string
    ParentID    string
    Permissions []string
}

// RoleHierarchy manages hierarchical roles.
type RoleHierarchy struct {
    mu    sync.RWMutex
    roles map[string]*Role
}

// NewRoleHierarchy creates a new role hierarchy manager.
func NewRoleHierarchy() *RoleHierarchy

// AddRole adds a role to the hierarchy.
func (h *RoleHierarchy) AddRole(role *Role) error

// GetRole retrieves a role by ID.
func (h *RoleHierarchy) GetRole(id string) (*Role, bool)

// GetEffectivePermissions returns all permissions including inherited.
func (h *RoleHierarchy) GetEffectivePermissions(roleID string) []string

// GetAncestors returns all ancestor role IDs.
func (h *RoleHierarchy) GetAncestors(roleID string) []string

// HasCircularDependency checks if adding a parent would create a cycle.
func (h *RoleHierarchy) HasCircularDependency(roleID, parentID string) bool
```

### 5. Authorization Service (internal/service)

```go
// AuthorizationService provides authorization operations.
type AuthorizationService struct {
    engine    *policy.Engine
    hierarchy *rbac.RoleHierarchy
    caep      *caep.Emitter
    logger    *logging.Client
    metrics   *Metrics
}

// AuthorizationInput represents an authorization request.
type AuthorizationInput struct {
    SubjectID         string                 `json:"subject_id"`
    SubjectAttributes map[string]interface{} `json:"subject_attributes"`
    ResourceType      string                 `json:"resource_type"`
    ResourceID        string                 `json:"resource_id"`
    ResourceAttributes map[string]interface{} `json:"resource_attributes"`
    Action            string                 `json:"action"`
    Environment       map[string]interface{} `json:"environment"`
}

// AuthorizationDecision represents the authorization outcome.
type AuthorizationDecision struct {
    Allowed      bool
    PolicyID     string
    Reason       string
    MatchedRules []string
    Cached       bool
}

// NewAuthorizationService creates a new authorization service.
func NewAuthorizationService(
    engine *policy.Engine,
    hierarchy *rbac.RoleHierarchy,
    caep *caep.Emitter,
    logger *logging.Client,
) *AuthorizationService

// Authorize evaluates a single authorization request.
func (s *AuthorizationService) Authorize(ctx context.Context, input AuthorizationInput) (AuthorizationDecision, error)

// BatchAuthorize evaluates multiple authorization requests.
func (s *AuthorizationService) BatchAuthorize(ctx context.Context, inputs []AuthorizationInput) ([]AuthorizationDecision, error)

// GetPermissions retrieves permissions for a subject.
func (s *AuthorizationService) GetPermissions(ctx context.Context, subjectID string) ([]string, error)

// GetRoles retrieves roles for a subject.
func (s *AuthorizationService) GetRoles(ctx context.Context, subjectID string) ([]Role, error)
```

### 6. gRPC Handler (internal/grpc)

```go
// Handler implements the IAMPolicyService gRPC interface.
type Handler struct {
    pb.UnimplementedIAMPolicyServiceServer
    service *service.AuthorizationService
    engine  *policy.Engine
    logger  *logging.Client
    config  *config.Config
}

// NewHandler creates a new gRPC handler.
func NewHandler(
    service *service.AuthorizationService,
    engine *policy.Engine,
    logger *logging.Client,
    config *config.Config,
) *Handler

// Authorize handles single authorization requests.
func (h *Handler) Authorize(ctx context.Context, req *pb.AuthorizeRequest) (*pb.AuthorizeResponse, error)

// BatchAuthorize handles batch authorization requests.
func (h *Handler) BatchAuthorize(ctx context.Context, req *pb.BatchAuthorizeRequest) (*pb.BatchAuthorizeResponse, error)

// GetUserPermissions retrieves user permissions.
func (h *Handler) GetUserPermissions(ctx context.Context, req *pb.GetPermissionsRequest) (*pb.PermissionsResponse, error)

// GetUserRoles retrieves user roles.
func (h *Handler) GetUserRoles(ctx context.Context, req *pb.GetRolesRequest) (*pb.RolesResponse, error)

// ReloadPolicies triggers policy hot-reload.
func (h *Handler) ReloadPolicies(ctx context.Context, req *pb.ReloadRequest) (*pb.ReloadResponse, error)
```

### 7. CAEP Emitter (internal/caep)

```go
// Emitter handles CAEP event emission.
type Emitter struct {
    enabled      bool
    transmitter  string
    serviceToken string
    issuer       string
    httpClient   *http.Client
    logger       *logging.Client
}

// Event represents a CAEP event.
type Event struct {
    EventType      string                 `json:"event_type"`
    Subject        Subject                `json:"subject"`
    EventTimestamp int64                  `json:"event_timestamp"`
    ReasonAdmin    map[string]string      `json:"reason_admin,omitempty"`
    Extra          map[string]interface{} `json:"extra,omitempty"`
}

// NewEmitter creates a new CAEP emitter.
func NewEmitter(cfg CAEPConfig, logger *logging.Client) *Emitter

// EmitRoleChange emits a role change event.
func (e *Emitter) EmitRoleChange(ctx context.Context, userID, previousRole, newRole string) error

// EmitPermissionChange emits a permission change event.
func (e *Emitter) EmitPermissionChange(ctx context.Context, userID string, added, removed []string) error

// EmitAssuranceLevelChange emits an assurance level change event.
func (e *Emitter) EmitAssuranceLevelChange(ctx context.Context, userID, previous, current, reason string) error
```

### 8. Health Manager (internal/health)

```go
// Manager handles health and readiness checks.
type Manager struct {
    mu       sync.RWMutex
    ready    bool
    checks   map[string]HealthCheck
    logger   *logging.Client
}

// HealthCheck is a function that returns health status.
type HealthCheck func(ctx context.Context) error

// NewManager creates a new health manager.
func NewManager(logger *logging.Client) *Manager

// RegisterCheck registers a named health check.
func (m *Manager) RegisterCheck(name string, check HealthCheck)

// SetReady sets the readiness status.
func (m *Manager) SetReady(ready bool)

// LivenessHandler returns HTTP handler for liveness probe.
func (m *Manager) LivenessHandler() http.HandlerFunc

// ReadinessHandler returns HTTP handler for readiness probe.
func (m *Manager) ReadinessHandler() http.HandlerFunc
```

## Data Models

### Authorization Input

```go
type AuthorizationInput struct {
    SubjectID          string                 `json:"subject_id"`
    SubjectAttributes  map[string]interface{} `json:"subject_attributes"`
    ResourceType       string                 `json:"resource_type"`
    ResourceID         string                 `json:"resource_id"`
    ResourceAttributes map[string]interface{} `json:"resource_attributes"`
    Action             string                 `json:"action"`
    Environment        map[string]interface{} `json:"environment"`
}
```

### Authorization Decision

```go
type AuthorizationDecision struct {
    Allowed      bool     `json:"allowed"`
    PolicyID     string   `json:"policy_id"`
    Reason       string   `json:"reason"`
    MatchedRules []string `json:"matched_rules"`
    Cached       bool     `json:"cached"`
}
```

### Cache Key Format

```
iam-policy:decision:{sha256(subject_id:resource_type:resource_id:action)}
```

### Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `iam_policy_authorization_total` | Counter | `allowed`, `cached`, `policy_id` | Total authorization requests |
| `iam_policy_authorization_duration_seconds` | Histogram | `cached` | Authorization latency |
| `iam_policy_cache_hits_total` | Counter | - | Cache hits |
| `iam_policy_cache_misses_total` | Counter | - | Cache misses |
| `iam_policy_policies_loaded` | Gauge | - | Number of loaded policies |
| `iam_policy_policy_reload_total` | Counter | `status` | Policy reload attempts |



## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system—essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*

Based on the prework analysis of acceptance criteria, the following correctness properties have been identified for property-based testing:

### Property 1: Configuration Loading Consistency

*For any* valid environment variable with prefix `IAM_POLICY_`, loading configuration SHALL correctly parse and store the value, and reloading SHALL produce the same configuration state.

**Validates: Requirements 1.1, 1.2, 1.3**

### Property 2: Cache Namespace Isolation

*For any* cache operation performed by the IAM Policy Service, the cache key SHALL be prefixed with the configured namespace `iam-policy:`, ensuring isolation from other services.

**Validates: Requirements 2.5**

### Property 3: Log Entry Enrichment

*For any* log entry produced by the IAM Policy Service with a context containing correlation_id, trace_id, or span_id, the log entry SHALL include all present context values.

**Validates: Requirements 2.6**

### Property 4: Policy Evaluation Determinism

*For any* valid authorization input and loaded policy set, evaluating the same input multiple times SHALL produce identical decisions (allow/deny, policy_id, matched_rules).

**Validates: Requirements 3.4, 3.5**

### Property 5: Decision Cache Round-Trip

*For any* authorization decision that is cached, retrieving it before TTL expiration SHALL return the exact same decision (allowed, policy_id, matched_rules), and the cached flag SHALL be true.

**Validates: Requirements 3.6, 3.7**

### Property 6: Permission Inheritance Completeness

*For any* role hierarchy where role R has parent P, the effective permissions of R SHALL be a superset of the effective permissions of P (R.permissions ⊇ P.permissions).

**Validates: Requirements 4.2, 4.3**

### Property 7: Circular Dependency Detection

*For any* role hierarchy, attempting to set role A as parent of role B when B is already an ancestor of A SHALL be rejected, preventing circular dependencies.

**Validates: Requirements 4.4**

### Property 8: gRPC Request-Response Consistency

*For any* valid authorization request sent via gRPC, the response SHALL contain a valid decision (allowed boolean, non-empty reason), and batch requests SHALL return exactly one response per input request.

**Validates: Requirements 5.1, 5.2, 5.8**

### Property 9: CAEP Event Structure Completeness

*For any* CAEP event emitted (role change, permission change, assurance level change), the event SHALL contain event_type, subject (with format, iss, sub), event_timestamp, and relevant extra fields.

**Validates: Requirements 6.1, 6.2, 6.4**

### Property 10: Trace Context Propagation

*For any* gRPC call with incoming trace context, the trace_id and span_id SHALL be propagated to all downstream operations and included in log entries.

**Validates: Requirements 7.2**

### Property 11: Metrics Recording Accuracy

*For any* authorization request processed, the metrics SHALL be updated: authorization_total counter incremented, duration histogram observed, and cache hit/miss counter incremented appropriately.

**Validates: Requirements 7.4, 7.5, 7.6**

### Property 12: Circuit Breaker State Transitions

*For any* sequence of cache-service failures exceeding the threshold, the circuit breaker SHALL transition to open state, and subsequent requests SHALL use local fallback until the circuit half-opens.

**Validates: Requirements 9.3**

### Property 13: Error Response Consistency

*For any* error returned by the gRPC handler, the response SHALL contain an appropriate gRPC status code, a correlation_id, and SHALL NOT expose internal implementation details.

**Validates: Requirements 9.5, 9.6**

### Property 14: Input Validation and Sanitization

*For any* authorization request with invalid or malicious input (empty subject_id, injection attempts in attributes), the service SHALL reject the request with a validation error OR sanitize the input before processing.

**Validates: Requirements 10.1, 10.2, 10.3**

### Property 15: Rate Limiting Enforcement

*For any* client exceeding the configured rate limit, subsequent requests within the rate limit window SHALL be rejected with a rate limit exceeded error.

**Validates: Requirements 10.5**

## Error Handling

### Error Categories

| Category | gRPC Code | Description |
|----------|-----------|-------------|
| Validation | `INVALID_ARGUMENT` | Invalid input parameters |
| NotFound | `NOT_FOUND` | Resource not found |
| PermissionDenied | `PERMISSION_DENIED` | Authorization denied |
| RateLimited | `RESOURCE_EXHAUSTED` | Rate limit exceeded |
| Internal | `INTERNAL` | Internal server error |
| Unavailable | `UNAVAILABLE` | Service temporarily unavailable |

### Error Response Format

```go
type ErrorResponse struct {
    Code          string `json:"code"`
    Message       string `json:"message"`
    CorrelationID string `json:"correlation_id"`
    Details       []any  `json:"details,omitempty"`
}
```

### Circuit Breaker Configuration

```go
type CircuitBreakerConfig struct {
    MaxFailures      int           // Failures before opening (default: 5)
    Timeout          time.Duration // Time before half-open (default: 30s)
    HalfOpenMaxReqs  int           // Requests in half-open (default: 3)
    SuccessThreshold int           // Successes to close (default: 2)
}
```

## Testing Strategy

### Dual Testing Approach

The service uses both unit tests and property-based tests for comprehensive coverage:

- **Unit tests**: Verify specific examples, edge cases, and error conditions
- **Property tests**: Verify universal properties across all valid inputs

### Property-Based Testing Configuration

- **Library**: `pgregory.net/rapid` (Go property-based testing)
- **Minimum iterations**: 100 per property test
- **Tag format**: `**Feature: iam-policy-service-modernization, Property {N}: {title}**`
- **Location**: `tests/property/`

### Test Organization

```
services/iam-policy/
├── internal/           # Source code only
│   ├── cache/
│   ├── caep/
│   ├── config/
│   ├── grpc/
│   ├── health/
│   ├── policy/
│   ├── rbac/
│   └── service/
└── tests/              # All tests
    ├── unit/           # Unit tests
    │   ├── cache/
    │   ├── config/
    │   ├── policy/
    │   ├── rbac/
    │   └── service/
    ├── property/       # Property-based tests
    │   ├── cache_test.go
    │   ├── policy_test.go
    │   ├── rbac_test.go
    │   └── service_test.go
    ├── integration/    # Integration tests
    │   ├── grpc_test.go
    │   └── platform_test.go
    └── testutil/       # Test utilities
        ├── generators.go
        └── mocks.go
```

### Test Coverage Requirements

| Component | Minimum Coverage |
|-----------|-----------------|
| Policy Engine | 90% |
| RBAC Module | 90% |
| Authorization Service | 85% |
| gRPC Handler | 80% |
| Config | 80% |
| Overall | 80% |

## Project Structure

```
services/iam-policy/
├── cmd/
│   └── server/
│       └── main.go              # Entry point
├── internal/
│   ├── cache/
│   │   └── decision_cache.go    # Decision caching
│   ├── caep/
│   │   └── emitter.go           # CAEP event emission
│   ├── config/
│   │   └── config.go            # Configuration management
│   ├── grpc/
│   │   ├── handler.go           # gRPC handlers
│   │   └── interceptors.go      # Custom interceptors
│   ├── health/
│   │   └── manager.go           # Health checks
│   ├── policy/
│   │   └── engine.go            # OPA policy engine
│   ├── rbac/
│   │   └── hierarchy.go         # Role hierarchy
│   └── service/
│       └── authorization.go     # Authorization service
├── policies/
│   ├── rbac.rego                # RBAC policies
│   └── abac.rego                # ABAC policies
├── tests/
│   ├── unit/
│   ├── property/
│   ├── integration/
│   └── testutil/
├── go.mod
├── go.sum
├── Dockerfile
├── Makefile
└── README.md
```

## Dependencies

### Go Modules

```go
module github.com/auth-platform/iam-policy-service

go 1.24

require (
    // Platform libs
    github.com/authcorp/libs/go/src/cache v0.0.0
    github.com/authcorp/libs/go/src/config v0.0.0
    github.com/authcorp/libs/go/src/errors v0.0.0
    github.com/authcorp/libs/go/src/fault v0.0.0
    github.com/authcorp/libs/go/src/grpc v0.0.0
    github.com/authcorp/libs/go/src/logging v0.0.0
    github.com/authcorp/libs/go/src/observability v0.0.0
    github.com/authcorp/libs/go/src/server v0.0.0
    github.com/authcorp/libs/go/src/testing v0.0.0
    
    // External
    github.com/fsnotify/fsnotify v1.8.0
    github.com/open-policy-agent/opa v1.0.0
    github.com/prometheus/client_golang v1.20.0
    go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.58.0
    go.opentelemetry.io/otel v1.35.0
    google.golang.org/grpc v1.70.0
    google.golang.org/protobuf v1.36.0
    pgregory.net/rapid v1.2.0
)
```
