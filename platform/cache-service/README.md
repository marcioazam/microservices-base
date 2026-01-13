# Cache Microservice

A distributed cache microservice built with Go, providing centralized cache management using Redis as the primary storage backend with optional local in-memory fallback.

## Features

- **Redis Backend**: Primary cache storage with support for standalone and cluster modes
- **Dual API**: gRPC for high-performance inter-service communication, REST for simpler integrations
- **Event-Driven Invalidation**: RabbitMQ/Kafka for async cache invalidation across services
- **Circuit Breaker**: Resilience pattern to handle Redis failures gracefully
- **Local Cache Fallback**: In-memory LRU cache for degraded mode operation
- **JWT Authentication**: Secure access with namespace-based isolation
- **AES Encryption**: Optional encryption for sensitive cached values
- **Prometheus Metrics**: Comprehensive observability with hit/miss rates, latency histograms

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Cache Microservice                        │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐  │
│  │  gRPC API   │  │  REST API   │  │  Message Broker     │  │
│  │  :50051     │  │  :8080      │  │  (RabbitMQ/Kafka)   │  │
│  └──────┬──────┘  └──────┬──────┘  └──────────┬──────────┘  │
│         │                │                     │             │
│         └────────────────┼─────────────────────┘             │
│                          │                                   │
│                   ┌──────▼──────┐                            │
│                   │ Auth Layer  │                            │
│                   │ (JWT)       │                            │
│                   └──────┬──────┘                            │
│                          │                                   │
│                   ┌──────▼──────┐                            │
│                   │Cache Service│                            │
│                   └──────┬──────┘                            │
│                          │                                   │
│         ┌────────────────┼────────────────┐                  │
│         │                │                │                  │
│  ┌──────▼──────┐  ┌──────▼──────┐  ┌──────▼──────┐          │
│  │Circuit      │  │ Local Cache │  │ Encryptor   │          │
│  │Breaker      │  │ (LRU)       │  │ (AES-GCM)   │          │
│  └──────┬──────┘  └─────────────┘  └─────────────┘          │
│         │                                                    │
│  ┌──────▼──────┐                                            │
│  │   Redis     │                                            │
│  │   Client    │                                            │
│  └─────────────┘                                            │
└─────────────────────────────────────────────────────────────┘
```

## Quick Start

### Prerequisites

- Go 1.24+
- Redis 6.0+
- RabbitMQ or Kafka (optional, for cache invalidation)
- Logging Service (optional, for centralized logging)

### Running Locally

```bash
# Navigate to service directory
cd platform/cache-service

# Download dependencies
go mod download

# Run the service
go run cmd/cache-service/main.go
```

### Environment Variables

#### Server Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `SERVER_GRPC_PORT` | gRPC server port | `50051` |
| `SERVER_HTTP_PORT` | HTTP server port | `8080` |
| `SERVER_GRACEFUL_TIMEOUT` | Graceful shutdown timeout | `30s` |

#### Redis Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `REDIS_ADDRESSES` | Redis addresses (comma-separated) | `localhost:6379` |
| `REDIS_PASSWORD` | Redis password | `` |
| `REDIS_DB` | Redis database number | `0` |
| `REDIS_POOL_SIZE` | Connection pool size | `10` |
| `REDIS_CLUSTER_MODE` | Enable cluster mode | `false` |

#### Cache Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `CACHE_DEFAULT_TTL` | Default TTL for cache entries | `1h` |
| `CACHE_LOCAL_CACHE_ENABLED` | Enable local cache fallback | `true` |
| `CACHE_LOCAL_CACHE_SIZE` | Max entries in local cache | `10000` |
| `CACHE_EVICTION_POLICY` | Eviction policy (lru/lfu) | `lru` |

#### Message Broker Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `BROKER_TYPE` | Message broker type (rabbitmq/kafka) | `rabbitmq` |
| `BROKER_URL` | Message broker URL | `` |
| `BROKER_TOPIC` | Invalidation topic | `cache-invalidation` |

#### Authentication Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `AUTH_JWT_SECRET` | JWT signing secret | `` |
| `AUTH_JWT_ISSUER` | JWT issuer | `cache-service` |
| `AUTH_ENCRYPTION_KEY` | AES encryption key (base64) | `` |

#### Logging Configuration (New in 2025)

| Variable | Description | Default |
|----------|-------------|---------|
| `LOGGING_SERVICE_ADDRESS` | Logging service gRPC address | `localhost:50052` |
| `LOGGING_BATCH_SIZE` | Log batch size before flush | `100` |
| `LOGGING_FLUSH_INTERVAL` | Max time before log flush | `5s` |
| `LOGGING_BUFFER_SIZE` | Max log buffer size | `10000` |
| `LOGGING_ENABLED` | Enable centralized logging | `true` |
| `LOGGING_SERVICE_ID` | Service identifier for logs | `cache-service` |
| `LOGGING_CIRCUIT_BREAKER_THRESHOLD` | Failures before circuit opens | `5` |
| `LOGGING_CIRCUIT_BREAKER_TIMEOUT` | Time before circuit half-opens | `30s` |

#### Observability Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `METRICS_ENABLED` | Enable Prometheus metrics | `true` |
| `METRICS_PATH` | Metrics endpoint path | `/metrics` |
| `TRACING_ENABLED` | Enable OpenTelemetry tracing | `false` |
| `TRACING_ENDPOINT` | OTLP tracing endpoint | `` |

## API Reference

### gRPC API (port 50051)

```protobuf
service CacheService {
  rpc Get(GetRequest) returns (GetResponse);
  rpc Set(SetRequest) returns (SetResponse);
  rpc Delete(DeleteRequest) returns (DeleteResponse);
  rpc BatchGet(BatchGetRequest) returns (BatchGetResponse);
  rpc BatchSet(BatchSetRequest) returns (BatchSetResponse);
  rpc Health(HealthRequest) returns (HealthResponse);
}
```

### REST API (port 8080)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/cache/{namespace}/{key}` | Get cached value |
| PUT | `/api/v1/cache/{namespace}/{key}` | Set cached value |
| DELETE | `/api/v1/cache/{namespace}/{key}` | Delete cached value |
| POST | `/api/v1/cache/{namespace}/batch/get` | Batch get values |
| POST | `/api/v1/cache/{namespace}/batch/set` | Batch set values |
| GET | `/health` | Health check |
| GET | `/ready` | Readiness check |
| GET | `/metrics` | Prometheus metrics |

### Example Requests

```bash
# Set a value
curl -X PUT http://localhost:8080/api/v1/cache/myapp/user:123 \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"value": "eyJ1c2VyIjoiam9obiJ9", "ttl_seconds": 3600}'

# Get a value
curl http://localhost:8080/api/v1/cache/myapp/user:123 \
  -H "Authorization: Bearer <token>"

# Delete a value
curl -X DELETE http://localhost:8080/api/v1/cache/myapp/user:123 \
  -H "Authorization: Bearer <token>"

# Health check
curl http://localhost:8080/health
```

## Project Structure

```
platform/cache-service/
├── api/
│   └── proto/cache/v1/       # Protocol Buffer definitions
├── cmd/
│   └── cache-service/        # Application entry point
├── configs/                  # Configuration files
├── internal/
│   ├── auth/                 # JWT authentication
│   ├── broker/               # Message broker (RabbitMQ/Kafka)
│   ├── cache/                # Core cache service
│   ├── config/               # Configuration loading
│   ├── crypto/               # AES encryption
│   ├── grpc/                 # gRPC server
│   ├── http/                 # REST API handlers
│   ├── localcache/           # In-memory LRU cache
│   ├── loggingclient/        # gRPC client for logging-service
│   ├── observability/        # Unified metrics, tracing, context
│   └── redis/                # Redis client with circuit breaker
└── tests/
    ├── integration/          # Integration tests
    ├── property/             # Property-based tests
    └── testutil/             # Test utilities and mocks
```

## Resilience

This service has two layers of resilience:

1. **Internal Circuit Breaker** (Redis): Uses `libs/go/src/fault` for Redis connection protection with local cache fallback
2. **Service Mesh** (Linkerd): External callers are protected via `ResiliencePolicy` CRD

See `deploy/kubernetes/service-mesh/cache-service/resilience-policy.yaml` for the Linkerd configuration.

> **Note**: This service uses shared libraries from `libs/go/src/`:
> - `fault` - Circuit breaker and retry patterns for resilience
> - `server` - Graceful shutdown handling
> - `grpc` - Consistent gRPC error code mapping
> - `http` - HTTP middleware (timeout, health handlers)
> - `observability` - Context propagation (correlation ID, trace context)

## Testing

```bash
# Run all tests
go test ./...

# Run property-based tests (100+ iterations)
go test ./tests/property/... -v

# Run integration tests
go test ./tests/integration/... -v

# Run with coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Metrics

The service exposes Prometheus metrics at `/metrics`:

- `cache_hits_total` - Total cache hits (by namespace, source)
- `cache_misses_total` - Total cache misses (by namespace)
- `cache_operations_total` - Total operations (by operation, status)
- `cache_operation_duration_seconds` - Operation latency histogram
- `cache_local_cache_size` - Current local cache size
- `cache_circuit_breaker_state` - Circuit breaker state (0=closed, 1=open, 2=half-open)

## Circuit Breaker

The circuit breaker protects against Redis failures:

- **Closed**: Normal operation, requests go to Redis
- **Open**: After N failures, requests fail fast or use local cache
- **Half-Open**: After timeout, allows limited requests to test recovery

Configuration:
- `MaxFailures`: 5 (failures before opening)
- `Timeout`: 30s (time before half-open)
- `HalfOpenMaxReqs`: 3 (requests allowed in half-open)
- `SuccessThreshold`: 2 (successes to close from half-open)

## Cache Invalidation

The service supports async cache invalidation via message broker:

```json
{
  "namespace": "myapp",
  "keys": ["user:123", "user:456"],
  "action": "delete",
  "timestamp": 1703123456
}
```

Actions:
- `delete`: Remove specified keys from cache
- `update`: Force cache refresh on next access

## Security

- **JWT Authentication**: All API requests require valid JWT token
- **Namespace Isolation**: Services can only access their own namespace
- **AES-GCM Encryption**: Optional encryption for sensitive values
- **TLS Support**: Redis TLS for production deployments

## License

Proprietary - Auth Platform Team
