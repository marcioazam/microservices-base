# Requirements Document

## Introduction

This document specifies the requirements for modernizing the `platform/logging-service` to state-of-the-art December 2025 standards. The modernization includes upgrading to .NET 9 (latest LTS), replacing deprecated libraries (NEST → Elastic.Clients.Elasticsearch 8.x, RabbitMQ.Client 6.x → 7.x), eliminating code redundancy, centralizing shared logic, and ensuring 100% production readiness with comprehensive test coverage.

## Glossary

- **Logging_Service**: The centralized logging microservice for the authentication platform
- **Log_Entry**: A structured log record containing timestamp, correlation ID, service ID, level, message, and metadata
- **Log_Queue**: Asynchronous queue for buffering log entries before persistence
- **Log_Repository**: Storage abstraction for persisting and querying log entries
- **PII_Masker**: Component responsible for detecting and masking personally identifiable information
- **Retention_Service**: Component managing log lifecycle and archival policies
- **Elastic_Client**: The Elasticsearch client for .NET (Elastic.Clients.Elasticsearch 8.x)
- **RabbitMQ_Client**: The RabbitMQ messaging client for .NET (version 7.x)

## Requirements

### Requirement 1: .NET 9 Platform Upgrade

**User Story:** As a platform engineer, I want the logging service upgraded to .NET 9, so that I can benefit from the latest performance improvements, security patches, and language features.

#### Acceptance Criteria

1. THE Logging_Service SHALL target .NET 9.0 framework across all projects
2. THE Logging_Service SHALL use C# 13 language features where beneficial
3. THE Logging_Service SHALL update all NuGet packages to .NET 9 compatible versions
4. THE Logging_Service SHALL maintain backward compatibility with existing API contracts

### Requirement 2: Elasticsearch Client Modernization

**User Story:** As a developer, I want the deprecated NEST client replaced with Elastic.Clients.Elasticsearch 8.x, so that I can use the officially supported modern client with better performance.

#### Acceptance Criteria

1. WHEN the Log_Repository saves a log entry, THE Elastic_Client SHALL use the Elastic.Clients.Elasticsearch 8.x API
2. WHEN the Log_Repository queries logs, THE Elastic_Client SHALL use the modern fluent query builder
3. WHEN the Log_Repository performs bulk operations, THE Elastic_Client SHALL use the BulkAll helper for efficient batching
4. THE Elastic_Client SHALL support connection pooling and automatic node discovery
5. THE Elastic_Client SHALL serialize LogEntry objects using System.Text.Json source generators

### Requirement 3: RabbitMQ Client Modernization

**User Story:** As a developer, I want the RabbitMQ client upgraded to version 7.x, so that I can use the modern async-first API with better resource management.

#### Acceptance Criteria

1. WHEN the Log_Queue enqueues entries, THE RabbitMQ_Client SHALL use the async channel API
2. WHEN the Log_Queue dequeues entries, THE RabbitMQ_Client SHALL use async consumers
3. THE RabbitMQ_Client SHALL use IConnection and IChannel interfaces with proper async disposal
4. THE RabbitMQ_Client SHALL support connection recovery and automatic reconnection

### Requirement 4: Code Redundancy Elimination

**User Story:** As a maintainer, I want all duplicated code eliminated and centralized, so that the codebase is easier to maintain and less prone to inconsistencies.

#### Acceptance Criteria

1. WHEN metrics are defined, THE Logging_Service SHALL have a single centralized metrics registry
2. WHEN validation is performed, THE Logging_Service SHALL use a single validation pipeline
3. WHEN configuration is loaded, THE Logging_Service SHALL use centralized configuration binding
4. THE Logging_Service SHALL eliminate duplicate ErrorResponse definitions between Core and Api layers
5. THE Logging_Service SHALL centralize all Prometheus metric definitions in a single location

### Requirement 5: Architecture Consolidation

**User Story:** As an architect, I want the service architecture consolidated and simplified, so that the codebase follows clean architecture principles with minimal layers.

#### Acceptance Criteria

1. THE Logging_Service SHALL maintain clear separation between Api, Core, Infrastructure, and Worker layers
2. THE Logging_Service SHALL use dependency injection for all service dependencies
3. THE Logging_Service SHALL use the Options pattern for all configuration
4. WHEN cross-cutting concerns are needed, THE Logging_Service SHALL use middleware or decorators

### Requirement 6: Testing Framework Modernization

**User Story:** As a QA engineer, I want the testing framework modernized to xUnit 3.x with latest tooling, so that tests run faster and provide better diagnostics.

#### Acceptance Criteria

1. THE Logging_Service SHALL use xUnit 3.x for all test projects
2. THE Logging_Service SHALL use Testcontainers 4.x for integration tests
3. THE Logging_Service SHALL use FsCheck 3.x for property-based tests
4. THE Logging_Service SHALL achieve minimum 80% code coverage
5. WHEN property tests run, THE Logging_Service SHALL execute minimum 100 iterations per property

### Requirement 7: Observability Enhancement

**User Story:** As an SRE, I want enhanced observability with OpenTelemetry 1.10+, so that I can effectively monitor and troubleshoot the service in production.

#### Acceptance Criteria

1. THE Logging_Service SHALL use OpenTelemetry 1.10+ for distributed tracing
2. THE Logging_Service SHALL export traces via OTLP protocol
3. THE Logging_Service SHALL include correlation IDs in all log entries and traces
4. WHEN errors occur, THE Logging_Service SHALL record exception details in spans
5. THE Logging_Service SHALL expose Prometheus metrics at /metrics endpoint

### Requirement 8: Security Hardening

**User Story:** As a security engineer, I want the service hardened against common vulnerabilities, so that sensitive data is protected and the service is production-ready.

#### Acceptance Criteria

1. WHEN PII is detected in log messages, THE PII_Masker SHALL mask it before storage
2. THE Logging_Service SHALL validate all input against injection attacks
3. THE Logging_Service SHALL use secure defaults for all configuration
4. WHEN authentication is required, THE Logging_Service SHALL support API key or JWT authentication
5. THE Logging_Service SHALL not expose internal error details to clients

### Requirement 9: Performance Optimization

**User Story:** As a performance engineer, I want the service optimized for high throughput, so that it can handle production load without degradation.

#### Acceptance Criteria

1. THE Logging_Service SHALL process minimum 10,000 log entries per second
2. WHEN batching logs, THE Logging_Service SHALL use configurable batch sizes
3. THE Logging_Service SHALL use object pooling for frequently allocated objects
4. THE Logging_Service SHALL use System.Text.Json source generators for serialization
5. WHEN the queue reaches capacity, THE Logging_Service SHALL apply backpressure gracefully

### Requirement 10: Production Readiness

**User Story:** As a DevOps engineer, I want the service fully production-ready with health checks and graceful shutdown, so that it can be deployed reliably in Kubernetes.

#### Acceptance Criteria

1. THE Logging_Service SHALL expose /health/live endpoint for liveness probes
2. THE Logging_Service SHALL expose /health/ready endpoint for readiness probes
3. WHEN shutdown is requested, THE Logging_Service SHALL drain the queue before terminating
4. THE Logging_Service SHALL support configuration via environment variables
5. THE Logging_Service SHALL include Dockerfile optimized for production deployment
