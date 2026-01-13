# Requirements Document

## Introduction

This document defines the requirements for modernizing the cache-service microservice to December 2025 state-of-the-art standards. The modernization focuses on integrating with the centralized logging-service via gRPC, eliminating code redundancies, upgrading deprecated dependencies, and achieving production-ready architecture with zero legacy patterns.

## Glossary

- **Cache_Service**: The distributed cache microservice being modernized
- **Logging_Service**: The centralized logging microservice that Cache_Service will integrate with for all logging operations
- **Logging_Client**: gRPC client for communicating with Logging_Service
- **Log_Entry**: Structured log message sent to Logging_Service via gRPC
- **Correlation_ID**: Unique identifier for tracing requests across services
- **Circuit_Breaker**: Resilience pattern preventing cascading failures
- **Protected_Client**: Redis client wrapper with circuit breaker and local cache fallback
- **TTL_Config**: Centralized TTL validation and normalization configuration
- **Message_Broker**: RabbitMQ or Kafka for async cache invalidation events

## Requirements

### Requirement 1: Centralized Logging Integration

**User Story:** As a platform engineer, I want the cache-service to send all logs to the centralized logging-service via gRPC, so that logs are unified across all microservices and can be queried centrally.

#### Acceptance Criteria

1. WHEN the Cache_Service starts, THE Logging_Client SHALL establish a gRPC connection to the Logging_Service
2. WHEN any log event occurs, THE Cache_Service SHALL send a Log_Entry to the Logging_Service via gRPC IngestLog RPC
3. WHEN multiple log events occur in rapid succession, THE Cache_Service SHALL batch them using IngestLogBatch RPC for efficiency
4. WHEN the Logging_Service is unavailable, THE Cache_Service SHALL buffer logs locally and retry with exponential backoff
5. WHEN buffering logs locally, THE Cache_Service SHALL limit buffer size to prevent memory exhaustion
6. THE Log_Entry SHALL include service_id, correlation_id, trace_id, span_id, level, message, and metadata fields
7. THE Cache_Service SHALL remove the local zap logger implementation and use only the Logging_Client for production logging

### Requirement 2: Dependency Modernization

**User Story:** As a developer, I want all dependencies upgraded to December 2025 stable versions, so that the service benefits from latest security patches and performance improvements.

#### Acceptance Criteria

1. THE Cache_Service SHALL upgrade Go version from 1.22 to 1.24 or 1.25
2. THE Cache_Service SHALL replace streadway/amqp with rabbitmq/amqp091-go for RabbitMQ communication
3. THE Cache_Service SHALL upgrade go-redis/v9 to latest stable version with built-in circuit breaker support
4. THE Cache_Service SHALL upgrade OpenTelemetry SDK to latest stable version (1.36+)
5. THE Cache_Service SHALL upgrade all transitive dependencies to resolve security vulnerabilities
6. THE Cache_Service SHALL use go.mod toolchain directive for reproducible builds

### Requirement 3: Code Deduplication and Centralization

**User Story:** As a maintainer, I want all duplicated logic eliminated and centralized, so that the codebase is minimal, cohesive, and easy to maintain.

#### Acceptance Criteria

1. THE Cache_Service SHALL have exactly one error handling implementation in internal/cache/errors.go
2. THE Cache_Service SHALL have exactly one TTL validation implementation in internal/cache/ttl.go
3. THE Cache_Service SHALL have exactly one circuit breaker state type definition (remove duplicate in cache/types.go)
4. THE Cache_Service SHALL centralize all gRPC error mapping in a single location
5. THE Cache_Service SHALL centralize all HTTP error response handling in a single location
6. THE Cache_Service SHALL remove redundant Logger interface in broker/invalidation.go and use Logging_Client

### Requirement 4: Architecture Restructuring

**User Story:** As an architect, I want the codebase restructured following Go 2025 best practices, so that it has clear separation of concerns and minimal coupling.

#### Acceptance Criteria

1. THE Cache_Service SHALL separate source code from tests (src/ vs tests/ directories)
2. THE Cache_Service SHALL have a single entry point in cmd/cache-service/main.go
3. THE Cache_Service SHALL organize internal packages by domain: cache, redis, broker, auth, observability
4. THE Cache_Service SHALL merge logging, metrics, and tracing into a single observability package
5. THE Cache_Service SHALL remove all .gitkeep files from non-empty directories
6. WHEN a package has a single file, THE Cache_Service SHALL evaluate merging with related package

### Requirement 5: Logging Client Implementation

**User Story:** As a developer, I want a robust gRPC client for the logging-service, so that all logs are reliably delivered to the centralized logging infrastructure.

#### Acceptance Criteria

1. THE Logging_Client SHALL implement connection pooling for gRPC connections
2. THE Logging_Client SHALL implement automatic reconnection on connection loss
3. THE Logging_Client SHALL support configurable batch size and flush interval
4. THE Logging_Client SHALL implement circuit breaker pattern for logging-service calls
5. WHEN circuit breaker is open, THE Logging_Client SHALL write to stderr as fallback
6. THE Logging_Client SHALL support all log levels: Debug, Info, Warn, Error, Fatal
7. THE Logging_Client SHALL automatically extract trace context from context.Context

### Requirement 6: Configuration Modernization

**User Story:** As a DevOps engineer, I want configuration to follow 12-factor app principles with validation, so that deployment is reliable across environments.

#### Acceptance Criteria

1. THE Cache_Service SHALL add LOGGING_SERVICE_ADDRESS environment variable for logging-service gRPC endpoint
2. THE Cache_Service SHALL add LOGGING_BATCH_SIZE environment variable (default: 100)
3. THE Cache_Service SHALL add LOGGING_FLUSH_INTERVAL environment variable (default: 5s)
4. THE Cache_Service SHALL validate all configuration at startup and fail fast on invalid values
5. THE Cache_Service SHALL use caarlos0/env/v11 for environment variable parsing (upgrade from v10)
6. THE Cache_Service SHALL log configuration summary (excluding secrets) on startup

### Requirement 7: Test Modernization

**User Story:** As a QA engineer, I want tests organized separately from source code with comprehensive coverage, so that testing is maintainable and thorough.

#### Acceptance Criteria

1. THE Cache_Service SHALL move all test files to tests/ directory mirroring src/ structure
2. THE Cache_Service SHALL maintain property-based tests using leanovate/gopter
3. THE Cache_Service SHALL achieve minimum 80% code coverage on core packages
4. THE Cache_Service SHALL have integration tests for logging-service gRPC communication
5. THE Cache_Service SHALL have property tests for logging client batching behavior
6. WHEN running tests, THE Cache_Service SHALL use testcontainers for Redis and RabbitMQ

### Requirement 8: Observability Consolidation

**User Story:** As a platform engineer, I want observability concerns (logging, metrics, tracing) unified in a single package, so that instrumentation is consistent and maintainable.

#### Acceptance Criteria

1. THE Cache_Service SHALL merge internal/logging, internal/metrics, internal/tracing into internal/observability
2. THE Observability package SHALL provide a single initialization function for all telemetry
3. THE Observability package SHALL use OpenTelemetry for both tracing and metrics
4. THE Observability package SHALL integrate with Logging_Client for structured logging
5. WHEN creating spans, THE Observability package SHALL automatically add correlation_id attribute
6. THE Cache_Service SHALL remove duplicate context key definitions (consolidate in observability)

### Requirement 9: gRPC Server Modernization

**User Story:** As a developer, I want the gRPC server to use generated protobuf code and modern interceptors, so that the API is type-safe and well-instrumented.

#### Acceptance Criteria

1. THE Cache_Service SHALL generate gRPC code from api/proto/cache/v1/cache.proto
2. THE Cache_Service SHALL remove manually defined proto message types from internal/grpc/server.go
3. THE Cache_Service SHALL use otelgrpc interceptors for automatic tracing
4. THE Cache_Service SHALL use grpc-ecosystem/go-grpc-middleware/v2 for interceptor chaining
5. THE Cache_Service SHALL implement proper gRPC health checking using grpc-health-probe compatible service

### Requirement 10: HTTP Router Modernization

**User Story:** As a developer, I want the HTTP router to use modern middleware patterns, so that request handling is consistent and well-instrumented.

#### Acceptance Criteria

1. THE Cache_Service SHALL continue using go-chi/chi/v5 as HTTP router
2. THE Cache_Service SHALL use otelhttp middleware for automatic tracing
3. THE Cache_Service SHALL implement request ID middleware that generates or extracts correlation_id
4. THE Cache_Service SHALL implement structured error responses with consistent JSON format
5. THE Cache_Service SHALL remove duplicate error response functions (centralize in http package)

### Requirement 11: Redis Client Modernization

**User Story:** As a developer, I want the Redis client to use go-redis v9 built-in features, so that custom implementations are minimized.

#### Acceptance Criteria

1. THE Cache_Service SHALL evaluate using go-redis v9 built-in circuit breaker (maintnotifications package)
2. THE Cache_Service SHALL use go-redis v9 connection pooling configuration
3. THE Cache_Service SHALL implement Redis Cluster support using go-redis ClusterClient
4. THE Cache_Service SHALL remove redundant NewClient function that accepts simple Config struct
5. THE Cache_Service SHALL use context-aware Redis operations throughout

### Requirement 12: Message Broker Modernization

**User Story:** As a developer, I want message broker implementations to use current stable libraries, so that the service is secure and maintainable.

#### Acceptance Criteria

1. THE Cache_Service SHALL replace streadway/amqp with rabbitmq/amqp091-go
2. THE Cache_Service SHALL implement connection recovery for RabbitMQ using amqp091-go features
3. THE Cache_Service SHALL use segmentio/kafka-go with latest stable version
4. THE Cache_Service SHALL implement graceful shutdown for broker connections
5. THE Cache_Service SHALL remove duplicate broker interface definitions (consolidate in broker package)

