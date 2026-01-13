# Implementation Tasks: Cache Service Modernization 2025

## Phase 1: Foundation

### Task 1.1: Update go.mod Dependencies
- [x] Upgrade Go version from 1.22 to 1.24
- [x] Replace `github.com/streadway/amqp` with `github.com/rabbitmq/amqp091-go`
- [x] Upgrade `github.com/caarlos0/env/v10` to `github.com/caarlos0/env/v11`
- [x] Upgrade OpenTelemetry packages to latest stable (1.36+)
- [x] Add `go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc`
- [x] Add `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp`
- [x] Add `github.com/testcontainers/testcontainers-go` for integration tests
- [x] Run `go mod tidy` to clean up dependencies

### Task 1.2: Create Logging Client Package
- [x] Create `internal/loggingclient/client.go` with gRPC client implementation
- [x] Create `internal/loggingclient/config.go` with configuration struct
- [x] Create `internal/loggingclient/level.go` with log level types
- [x] Create `internal/loggingclient/field.go` with structured field types
- [x] Implement batch buffering with configurable size and flush interval
- [x] Implement circuit breaker for logging-service calls
- [x] Implement stderr fallback when circuit is open
- [x] Implement graceful shutdown with buffer flush

### Task 1.3: Create Observability Package
- [x] Create `internal/observability/provider.go` with unified Provider struct
- [x] Create `internal/observability/context.go` with centralized context keys
- [x] Create `internal/observability/metrics.go` with Prometheus metrics
- [x] Create `internal/observability/tracing.go` with OpenTelemetry tracer
- [x] Migrate context key definitions from logging/logger.go
- [x] Migrate context key definitions from tracing package
- [x] Implement single initialization function for all telemetry

### Task 1.4: Generate Protobuf Code
- [x] Copy logging.proto to `api/proto/logging/v1/logging.proto`
- [x] Create buf.gen.yaml for protobuf generation
- [x] Generate Go code for logging-service client
- [x] Verify generated code compiles correctly

## Phase 2: Integration

### Task 2.1: Update Configuration
- [x] Add LoggingConfig struct to `internal/config/config.go`
- [x] Add LOGGING_SERVICE_ADDRESS environment variable
- [x] Add LOGGING_BATCH_SIZE environment variable
- [x] Add LOGGING_FLUSH_INTERVAL environment variable
- [x] Add LOGGING_BUFFER_SIZE environment variable
- [x] Add LOGGING_ENABLED environment variable
- [x] Update config validation to include logging config
- [x] Update config loading to use env/v11

### Task 2.2: Enhance Error Handling
- [x] Add ToHTTPStatus() method to cache.Error
- [x] Add ToGRPCStatus() method to cache.Error
- [x] Create `internal/http/errors.go` with centralized WriteError function
- [x] Create `internal/grpc/errors.go` with centralized error conversion
- [x] Update HTTP handlers to use centralized error handling
- [x] Update gRPC server to use centralized error handling

### Task 2.3: Update Broker Package
- [x] Replace streadway/amqp import with rabbitmq/amqp091-go
- [x] Update RabbitMQBroker to use amqp091-go types
- [x] Implement connection recovery using amqp091-go features
- [x] Remove duplicate Logger interface from broker/invalidation.go
- [x] Update broker to use loggingclient.Client for logging
- [x] Test broker reconnection behavior

### Task 2.4: Update Main Application
- [x] Initialize loggingclient in main.go
- [x] Initialize observability provider in main.go
- [x] Pass loggingclient to all components
- [x] Update graceful shutdown to close loggingclient
- [x] Remove zap logger initialization

## Phase 3: Cleanup

### Task 3.1: Remove Duplicate Code
- [x] Remove CircuitState from cache/types.go (use circuitbreaker.State)
- [x] Remove Logger interface from broker/invalidation.go
- [x] Remove duplicate context keys from logging/logger.go
- [x] Remove duplicate writeError functions from http/handlers.go
- [x] Update all references to use centralized implementations

### Task 3.2: Remove Legacy Packages
- [x] Remove `internal/logging/` package entirely
- [x] Merge `internal/metrics/` into `internal/observability/`
- [x] Merge `internal/tracing/` into `internal/observability/`
- [x] Update all imports throughout codebase
- [x] Verify no broken imports

### Task 3.3: Update All Components to Use New Logging
- [x] Update cache/service.go to use loggingclient
- [x] Update redis/client.go to use loggingclient
- [x] Update redis/protected_client.go to use loggingclient
- [x] Update broker/rabbitmq.go to use loggingclient
- [x] Update broker/kafka.go to use loggingclient
- [x] Update http/handlers.go to use loggingclient
- [x] Update grpc/server.go to use loggingclient
- [x] Update auth/jwt.go to use loggingclient

## Phase 4: Testing

### Task 4.1: Create Test Infrastructure
- [x] Create `tests/testutil/containers.go` with testcontainers helpers
- [x] Create `tests/testutil/mocks.go` with common mocks
- [x] Create `tests/testutil/fixtures.go` with test fixtures
- [x] Create mock logging-service for integration tests

### Task 4.2: Create Unit Tests
- [x] Create `tests/unit/loggingclient/client_test.go`
- [x] Create `tests/unit/loggingclient/batch_test.go`
- [x] Create `tests/unit/observability/context_test.go`
- [x] Create `tests/unit/cache/errors_test.go` (enhance existing)
- [x] Verify 80%+ coverage on new packages

### Task 4.3: Create Property-Based Tests
- [x] Create `tests/property/logging_properties_test.go`
  - [x] Property 1: Delivery guarantee
  - [x] Property 2: Batch flush invariant
  - [x] Property 8: Buffer overflow protection
- [x] Create `tests/property/circuit_breaker_properties_test.go`
  - [x] Property 3: State transitions
- [x] Create `tests/property/error_mapping_properties_test.go`
  - [x] Property 4: Error code mapping consistency
- [x] Create `tests/property/ttl_properties_test.go`
  - [x] Property 5: TTL normalization idempotence
  - [x] Property 6: Cache entry expiration monotonicity
- [x] Create `tests/property/context_properties_test.go`
  - [x] Property 7: Correlation ID propagation

### Task 4.4: Create Integration Tests
- [x] Create `tests/integration/loggingclient_test.go`
  - [x] Property 9: Graceful shutdown completeness
- [x] Create `tests/integration/redis_test.go`
  - [x] Property 10: Redis operation context propagation
  - [x] Property 15: Local cache fallback activation
- [x] Create `tests/integration/broker_test.go`
  - [x] Property 11: Message broker reconnection resilience
- [x] Create `tests/integration/grpc_test.go`
  - [x] Property 13: gRPC interceptor chain order
- [x] Create `tests/integration/http_test.go`
  - [x] Property 14: HTTP middleware chain order

### Task 4.5: Migrate Existing Tests
- [x] Move existing property tests to tests/property/
- [x] Move existing integration tests to tests/integration/
- [x] Update test imports for new package structure
- [x] Verify all existing tests pass

## Phase 5: Validation

### Task 5.1: Run Full Test Suite
- [x] Run `go test -v -race ./...`
- [x] Verify all tests pass
- [x] Generate coverage report
- [x] Verify 80%+ overall coverage

### Task 5.2: Security Scan
- [x] Run `gosec ./...`
- [x] Fix any high/critical findings
- [x] Document any accepted risks

### Task 5.3: Lint Check
- [x] Run `golangci-lint run` (used `go vet` as alternative)
- [x] Fix any linting errors
- [x] Verify code style compliance

### Task 5.4: Final Verification
- [x] Verify no deprecated dependencies remain
- [x] Verify no duplicate code remains
- [x] Verify all imports are correct
- [x] Test service startup with logging-service
- [x] Test service startup without logging-service (fallback)
- [x] Update README.md with new configuration options
- [x] Update CHANGELOG.md with modernization changes
