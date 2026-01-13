# Changelog

All notable changes to the Logging Service will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [2.0.0] - 2025-12-21

### Changed

- **BREAKING**: Upgraded to .NET 10.0 with C# preview language features
- **BREAKING**: Replaced NEST 7.17.5 with Elastic.Clients.Elasticsearch 8.17.0
- **BREAKING**: Upgraded RabbitMQ.Client from 6.8.1 to 7.1.0 (async-first API)
- Upgraded OpenTelemetry packages to 1.11.0
- Upgraded xUnit to 3.x
- Upgraded FsCheck to 3.x
- Upgraded Testcontainers to 4.x
- Upgraded all Microsoft.Extensions.* packages to 10.0.0
- Upgraded prometheus-net to 8.2.1

### Added

- Centralized metrics registry (`LoggingMetrics.cs`)
- Centralized configuration model (`LoggingServiceOptions`)
- System.Text.Json source generators for LogEntry serialization
- OpenTelemetry OTLP exporter for distributed tracing
- Prometheus metrics exporter at `/metrics`
- Health check endpoints (`/health/live`, `/health/ready`)
- Graceful shutdown with queue draining
- Non-root user in Dockerfile for security
- Property-based tests for all correctness properties

### Removed

- Duplicate ErrorResponse definitions (consolidated in Core)
- Duplicate metric definitions across services
- NEST client dependency (replaced with Elastic.Clients.Elasticsearch)
- Synchronous RabbitMQ API usage

### Fixed

- Connection pooling for Elasticsearch client
- Automatic reconnection for RabbitMQ
- PII masking in metadata fields

## [1.0.0] - 2024-01-15

### Added

- Initial release with .NET 8
- gRPC and REST API for log ingestion
- Elasticsearch storage backend
- RabbitMQ queue for async processing
- Basic health checks
- Prometheus metrics
