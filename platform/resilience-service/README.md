# Resilience Service

Centralized resilience layer for the Auth Platform, providing advanced resilience patterns including circuit breaker, retry with exponential backoff, timeout management, rate limiting, and bulkhead isolation.

## Features

- **Circuit Breaker**: Prevents cascading failures with configurable thresholds
- **Retry Handler**: Exponential backoff with jitter for transient failures
- **Timeout Manager**: Context-based timeout enforcement with per-operation configuration
- **Rate Limiter**: Token bucket and sliding window algorithms
- **Bulkhead**: Semaphore-based concurrency limiting with partition isolation
- **Health Aggregator**: Aggregated health status with CAEP event emission
- **Policy Engine**: Hot-reloadable resilience policies

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    gRPC Service Layer                           │
│  ExecuteWithResilience │ GetCircuitState │ UpdatePolicy │ Health│
├─────────────────────────────────────────────────────────────────┤
│                      Policy Engine                              │
│  Parser │ Validator │ Store │ Hot-Reload                        │
├─────────────────────────────────────────────────────────────────┤
│                  Resilience Patterns Layer                      │
│  Circuit Breaker │ Retry │ Timeout │ Rate Limiter │ Bulkhead    │
├─────────────────────────────────────────────────────────────────┤
│                  Infrastructure Clients                         │
│  Redis │ OpenTelemetry │ Audit Logger                           │
└─────────────────────────────────────────────────────────────────┘
```

## API

See [api/proto/infra/resilience.proto](../../api/proto/infra/resilience.proto) for the gRPC API definition.

### RPCs

- `ExecuteWithResilience`: Execute request with resilience policy applied
- `GetCircuitState`: Get current circuit breaker state
- `UpdatePolicy`: Update or create resilience policy
- `GetPolicy`: Retrieve policy by name
- `GetHealth`: Get aggregated health status
- `WatchCircuitState`: Stream circuit breaker state changes

## Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `RESILIENCE_HOST` | Service bind address | `0.0.0.0` |
| `RESILIENCE_PORT` | gRPC port | `50056` |
| `REDIS_URL` | Redis connection URL | `redis://localhost:6379` |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | OpenTelemetry endpoint | `http://localhost:4317` |
| `LOG_LEVEL` | Logging level | `info` |
| `POLICY_CONFIG_PATH` | Path to policy configuration | `/etc/resilience/policies.yaml` |

## Development

```bash
# Build
go build ./...

# Run tests
go test ./...

# Run with coverage
go test ./... -cover
```

## Deployment

```bash
# Kubernetes
helm install resilience-service deploy/kubernetes/helm/resilience-service

# Docker
docker build -f deploy/docker/resilience-service/Dockerfile -t resilience-service .
```

## Property-Based Tests

The service includes 25 property-based tests using gopter:

1. Circuit Breaker State Machine Correctness
2. Circuit Breaker State Serialization Round-Trip
3. Circuit State Change Event Emission
4. Retry Delay with Exponential Backoff and Jitter
5. Retry Exhaustion Returns Final Error
6. Open Circuit Blocks Retry Attempts
7. Retry Policy Configuration Round-Trip
8. Timeout Enforcement
9. Operation-Specific Timeout Precedence
10. Timeout Configuration Validation
11. Rate Limit Enforcement
12. Token Bucket Invariants
13. Sliding Window Request Counting
14. Rate Limit Response Headers
15. Bulkhead Concurrent Request Enforcement
16. Bulkhead Partition Isolation
17. Bulkhead Metrics Accuracy
18. Health Aggregation Logic
19. Health Change Event Emission
20. Policy Validation Rejects Invalid Configurations
21. Policy Definition Round-Trip
22. Circuit State Retrieval Consistency
23. Error to gRPC Status Code Mapping
24. Audit Event Required Fields
25. Graceful Shutdown Request Draining
