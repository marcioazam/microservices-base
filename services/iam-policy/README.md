# IAM Policy Service

Policy decision point (PDP) service implementing RBAC and ABAC using Open Policy Agent (OPA).

## Overview

The IAM Policy Service provides:

- **RBAC**: Role-Based Access Control with hierarchical roles
- **ABAC**: Attribute-Based Access Control via Rego policies
- **Hot Reload**: Policy updates without service restart
- **gRPC Interface**: High-performance authorization decisions
- **Decision Caching**: Distributed caching via platform cache-service with local fallback
- **Platform Integration**: Centralized logging and observability

## Tech Stack

- **Language**: Go 1.24+
- **Policy Engine**: Open Policy Agent (OPA) v1.0+
- **RPC**: gRPC v1.70+
- **Observability**: OpenTelemetry 1.35+
- **Testing**: pgregory.net/rapid (property-based)

## Project Structure

```
services/iam-policy/
├── cmd/server/          # Entry point with dependency injection
├── internal/
│   ├── cache/           # Decision caching with local fallback and encryption
│   ├── caep/            # CAEP event emission
│   ├── config/          # Centralized configuration
│   ├── crypto/          # Crypto-service client for encryption and signing
│   ├── errors/          # Consistent error handling
│   ├── grpc/            # gRPC handlers and interceptors
│   ├── health/          # Health checks
│   ├── logging/         # Logging integration
│   ├── observability/   # Metrics and tracing
│   ├── policy/          # OPA policy engine
│   ├── ratelimit/       # Rate limiting
│   ├── rbac/            # Role hierarchy
│   ├── server/          # Graceful shutdown management
│   ├── service/         # Authorization service
│   └── validation/      # Input validation
├── policies/            # Rego policy files
└── tests/
    ├── unit/            # Unit tests
    ├── property/        # Property-based tests
    ├── integration/     # Integration tests
    └── testutil/        # Test utilities and generators
```

## Resilience

This service is protected by Service Mesh (Linkerd) via `ResiliencePolicy`:

- **Circuit Breaker**: Threshold 10, consecutive failures
- **Retry**: 2 attempts for 502/503/504
- **Timeout**: Request 30s, Response 15s

See `deploy/kubernetes/service-mesh/iam-policy-service/resilience-policy.yaml` for configuration.

> **Note**: Internal circuit breaker code was removed - Service Mesh handles all resilience patterns at the infrastructure level.

## Configuration

Environment variables with prefix `IAM_POLICY_`:

| Variable | Description | Default |
|----------|-------------|---------|
| `IAM_POLICY_SERVER_GRPC_PORT` | gRPC server port | `50054` |
| `IAM_POLICY_SERVER_HEALTH_PORT` | Health check port | `8080` |
| `IAM_POLICY_SERVER_METRICS_PORT` | Metrics port | `9090` |
| `IAM_POLICY_SERVER_SHUTDOWN_TIMEOUT` | Graceful shutdown timeout | `30s` |
| `IAM_POLICY_POLICY_PATH` | Path to Rego policies | `./policies` |
| `IAM_POLICY_CACHE_ADDRESS` | Cache service address | `localhost:50051` |
| `IAM_POLICY_CACHE_NAMESPACE` | Cache key namespace | `iam-policy` |
| `IAM_POLICY_LOGGING_ADDRESS` | Logging service address | `localhost:50052` |
| `IAM_POLICY_CAEP_ENABLED` | Enable CAEP event emission | `false` |
| `IAM_POLICY_CAEP_TRANSMITTER` | CAEP transmitter URL | `` |
| `IAM_POLICY_CAEP_SERVICE_TOKEN` | Service token for CAEP | `` |
| `IAM_POLICY_CAEP_ISSUER` | CAEP event issuer | `iam-policy-service` |

### Crypto Integration

| Variable | Description | Default |
|----------|-------------|---------|
| `IAM_POLICY_CRYPTO_ENABLED` | Enable crypto-service integration | `false` |
| `IAM_POLICY_CRYPTO_ADDRESS` | Crypto service gRPC address | `localhost:50055` |
| `IAM_POLICY_CRYPTO_TIMEOUT` | Connection timeout | `5s` |
| `IAM_POLICY_CRYPTO_ENCRYPTION_KEY_ID` | Key ID for cache encryption (format: `namespace/id/version`) | `iam-policy/cache-encryption/1` |
| `IAM_POLICY_CRYPTO_SIGNING_KEY_ID` | Key ID for decision signing (format: `namespace/id/version`) | `iam-policy/decision-signing/1` |
| `IAM_POLICY_CRYPTO_KEY_CACHE_TTL` | TTL for key metadata cache | `5m` |
| `IAM_POLICY_CRYPTO_CACHE_ENCRYPTION` | Enable encrypted decision cache | `false` |
| `IAM_POLICY_CRYPTO_DECISION_SIGNING` | Enable decision signing | `false` |

## Running

```bash
# Development
go run cmd/server/main.go

# Production
go build -o iam-policy-service cmd/server/main.go
./iam-policy-service
```

## Testing

```bash
# Run all tests
go test ./...

# Run property-based tests only
go test ./tests/property/...

# Run with verbose output
go test -v ./tests/property/...

# Run specific property test
go test -v -run TestCacheNamespaceIsolation ./tests/property/...
```

### Property-Based Tests

The service uses property-based testing with `pgregory.net/rapid` to validate correctness properties:

- **Authorization Request-Response Consistency**: Valid response structure with non-empty reason
- **Batch Authorization Consistency**: Response count matches request count
- **Authorization Determinism**: Same request produces identical results
- **Permission Retrieval Completeness**: GetPermissions returns valid permission sets
- **Cache Namespace Isolation**: All cache keys prefixed with service namespace
- **Decision Cache Round-Trip**: Cached decisions retrievable with identical values
- **Cache Key Determinism**: Same input produces same cache key
- **Log Entry Enrichment**: Context values (correlation_id, trace_id, span_id) preserved
- **Configuration Consistency**: Deterministic configuration loading
- **Permission Inheritance Completeness**: Child roles inherit all parent permissions
- **Circular Dependency Detection**: Role cycles prevented at add time
- **CAEP Event Structure Completeness**: Events contain required fields (event_type, subject, timestamp)
- **CAEP Assurance Level Change**: Assurance level events include previous/current levels
- **CAEP Token Claims Change**: Token claims events include changed claims list
- **CAEP Subject Format**: Subject uses iss_sub format with valid issuer and subject
- **Input Validation and Sanitization**: Invalid/malicious inputs rejected or sanitized before processing
- **Rate Limiting Enforcement**: Clients exceeding rate limits are rejected with appropriate errors

## API

See `proto/iam_policy.proto` for the complete gRPC service definition.

### Key Endpoints

- `Authorize`: Evaluates authorization request against policies
- `BatchAuthorize`: Batch authorization requests
- `GetUserPermissions`: Returns permissions for a subject
- `GetUserRoles`: Returns roles for a subject
- `ReloadPolicies`: Triggers policy hot-reload

## RBAC Role Hierarchy

The service supports hierarchical roles with permission inheritance:

- **Role Inheritance**: Child roles inherit all permissions from parent roles
- **Circular Dependency Detection**: Prevents cycles when adding roles with parents
- **Permission Caching**: Effective permissions cached per role, invalidated on changes
- **Hierarchy Traversal**: Get ancestors and descendants of any role

```go
// Example: Creating a role hierarchy
hierarchy := rbac.NewRoleHierarchy()

// Add parent role
hierarchy.AddRole(&rbac.Role{
    ID:          "editor",
    Permissions: []string{"read", "write"},
})

// Add child role (inherits editor permissions)
hierarchy.AddRole(&rbac.Role{
    ID:          "admin",
    ParentID:    "editor",
    Permissions: []string{"delete"},
})

// Get effective permissions (returns: read, write, delete)
perms := hierarchy.GetEffectivePermissions("admin")
```

## Policy Hot Reload

The service watches the policy directory for changes and automatically reloads policies without restart. This enables:

- Zero-downtime policy updates
- A/B testing of policies
- Gradual policy rollouts

## Health Endpoints

The service exposes health check endpoints on the HTTP server (default port 8080):

| Endpoint | Description |
|----------|-------------|
| `/health/live` | Liveness probe - returns 200 if service is running |
| `/health/ready` | Readiness probe - returns 200 if service is ready to accept traffic |

The readiness check validates:
- Cache connectivity (degraded if unavailable, uses local fallback)
- Policy engine initialization
- gRPC server status

## Graceful Shutdown

The service implements graceful shutdown with configurable timeout:

1. **Health Check Update**: Marks service as not ready (readiness probe fails)
2. **gRPC Server Stop**: Stops accepting new connections, completes in-flight requests
3. **Log Flush**: Ensures all buffered logs are written
4. **Cache Close**: Closes cache connections cleanly

Shutdown is triggered by `SIGINT` or `SIGTERM` signals.

## Crypto Integration

The service integrates with the centralized `crypto-service` for:

### Cache Encryption

When enabled (`IAM_POLICY_CRYPTO_CACHE_ENCRYPTION=true`), authorization decisions stored in the distributed cache are encrypted using AES-256-GCM:

- **Encryption**: Decisions encrypted before cache storage
- **AAD Binding**: Subject ID and Resource ID used as Additional Authenticated Data
- **Graceful Fallback**: Falls back to plaintext storage if crypto-service unavailable

### Decision Signing

When enabled (`IAM_POLICY_CRYPTO_DECISION_SIGNING=true`), authorization decisions can be digitally signed using ECDSA P-256:

- **Integrity**: Signatures verify decision hasn't been tampered
- **Audit Trail**: Signed decisions support compliance requirements
- **Key Rotation**: Supports verification with previous key versions

### Fallback Behavior

When crypto-service is unavailable:
- Cache stores decisions in plaintext (JSON)
- Signatures omitted from responses
- Health check returns `DEGRADED` (not `UNHEALTHY`)
- Metrics increment `iam_crypto_fallback_total`

### Key ID Format

Key IDs follow the format `namespace/id/version`:
```
iam-policy/cache-encryption/1
iam-policy/decision-signing/1
```

## Metrics

The service exposes Prometheus-compatible metrics at `/metrics` (default port 9090).

### Available Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `iam_auth_decisions_total` | Counter | Total authorization decisions |
| `iam_auth_allowed_total` | Counter | Total allowed decisions |
| `iam_auth_denied_total` | Counter | Total denied decisions |
| `iam_auth_errors_total` | Counter | Total authorization errors |
| `iam_cache_hits_total` | Counter | Total cache hits |
| `iam_cache_misses_total` | Counter | Total cache misses |
| `iam_policy_evaluations_total` | Counter | Total policy evaluations |
| `iam_policy_count` | Gauge | Current loaded policy count |

### Crypto Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `iam_crypto_encrypt_total` | Counter | `status` | Total encryption operations |
| `iam_crypto_decrypt_total` | Counter | `status` | Total decryption operations |
| `iam_crypto_sign_total` | Counter | `status` | Total signing operations |
| `iam_crypto_verify_total` | Counter | `status` | Total verification operations |
| `iam_crypto_latency_seconds` | Histogram | `operation` | Crypto operation latency |
| `iam_crypto_errors_total` | Counter | `code` | Crypto errors by error code |
| `iam_crypto_fallback_total` | Counter | - | Fallback to unencrypted mode |

### gRPC Method Metrics

Per-method metrics are tracked internally for:
- Request count
- Error count
- Average duration

## Error Handling

The service uses a consistent error handling pattern with typed errors that map to gRPC status codes.

### Error Codes

| Code | gRPC Status | Description |
|------|-------------|-------------|
| `INVALID_INPUT` | `InvalidArgument` | Invalid request parameters |
| `UNAUTHORIZED` | `Unauthenticated` | Missing or invalid authentication |
| `FORBIDDEN` | `PermissionDenied` | Authorization denied |
| `NOT_FOUND` | `NotFound` | Resource not found |
| `CONFLICT` | `AlreadyExists` | Resource conflict |
| `UNAVAILABLE` | `Unavailable` | Service temporarily unavailable |
| `TIMEOUT` | `DeadlineExceeded` | Operation timed out |
| `RATE_LIMITED` | `ResourceExhausted` | Rate limit exceeded |
| `INTERNAL` | `Internal` | Internal server error |

### Usage

```go
import "github.com/auth-platform/iam-policy-service/internal/errors"

// Create typed errors
err := errors.InvalidInput("subject_id is required")
err := errors.Forbidden("access denied to resource")

// Wrap underlying errors
err := errors.Wrap(dbErr, errors.CodeInternal, "database query failed")

// Add correlation ID for tracing
err := errors.InvalidInput("invalid action").WithCorrelationID(correlationID)

// Convert to gRPC status
grpcErr := errors.ToGRPCError(err)
```
