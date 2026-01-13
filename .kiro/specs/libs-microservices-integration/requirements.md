# Requirements Document

## Introduction

Esta spec define a refatoração arquitetural para integrar as bibliotecas compartilhadas (`libs/go`) com os microserviços de plataforma (`platform/cache-service` e `platform/logging-service`). O objetivo é eliminar código duplicado, centralizar funcionalidades de cache e logging nos microserviços dedicados, e garantir que o `cache-service` utilize 100% das libs disponíveis.

## Glossary

- **Libs_Go**: Bibliotecas compartilhadas Go localizadas em `libs/go/src/`
- **Cache_Service**: Microserviço de cache distribuído em `platform/cache-service/`
- **Logging_Service**: Microserviço de logging centralizado em `platform/logging-service/`
- **Cache_Client**: Cliente gRPC para comunicação com o Cache_Service
- **Logging_Client**: Cliente gRPC para comunicação com o Logging_Service
- **LRU_Cache**: Implementação de cache local com eviction Least Recently Used
- **Circuit_Breaker**: Padrão de resiliência para proteção contra falhas em cascata
- **Fault_Tolerance**: Conjunto de padrões de resiliência (retry, timeout, bulkhead, rate limit)

## Requirements

### Requirement 1: Criar Cache Client nas Libs

**User Story:** As a developer, I want a cache client library that communicates with the cache-service, so that I can use distributed caching without implementing Redis directly.

#### Acceptance Criteria

1. THE Cache_Client SHALL provide Get, Set, Delete, BatchGet, and BatchSet operations via gRPC
2. THE Cache_Client SHALL support namespace isolation for multi-tenant scenarios
3. THE Cache_Client SHALL integrate with Circuit_Breaker from `libs/go/src/fault/` for resilience
4. THE Cache_Client SHALL support configurable connection pooling and timeouts
5. WHEN the Cache_Service is unavailable, THE Cache_Client SHALL return appropriate errors with retry hints
6. THE Cache_Client SHALL support optional local fallback cache using LRU_Cache from `libs/go/src/collections/`

### Requirement 2: Criar Logging Client nas Libs

**User Story:** As a developer, I want a logging client library that sends logs to the logging-service, so that I can have centralized logging without direct Elasticsearch access.

#### Acceptance Criteria

1. THE Logging_Client SHALL provide structured logging methods (Debug, Info, Warn, Error) via gRPC
2. THE Logging_Client SHALL support async batched log shipping for performance
3. THE Logging_Client SHALL integrate with observability context (correlation_id, trace_id, span_id)
4. WHEN the Logging_Service is unavailable, THE Logging_Client SHALL fallback to local stdout logging
5. THE Logging_Client SHALL support PII redaction before sending logs
6. THE Logging_Client SHALL support configurable buffer size and flush intervals

### Requirement 3: Remover LRU Cache Local das Libs

**User Story:** As a platform architect, I want to remove the local LRU cache implementation from libs, so that all caching goes through the centralized cache-service.

#### Acceptance Criteria

1. THE Libs_Go SHALL remove `libs/go/src/collections/lru.go` after Cache_Client is available
2. THE Libs_Go SHALL update `libs/go/src/patterns/cached_repository.go` to use Cache_Client instead of local cache
3. WHEN removing LRU_Cache, THE Libs_Go SHALL provide migration guide for existing consumers
4. THE Cache_Client SHALL provide a local-only mode for testing without Cache_Service dependency

### Requirement 4: Remover Logger Local das Libs

**User Story:** As a platform architect, I want to remove the local logger implementation from libs, so that all logging goes through the centralized logging-service.

#### Acceptance Criteria

1. THE Libs_Go SHALL remove `libs/go/src/observability/logger.go` after Logging_Client is available
2. THE Libs_Go SHALL keep context propagation utilities in `libs/go/src/observability/context.go`
3. WHEN removing Logger, THE Libs_Go SHALL provide migration guide for existing consumers
4. THE Logging_Client SHALL provide a local-only mode for testing without Logging_Service dependency

### Requirement 5: Migrar Cache-Service para Usar 100% das Libs

**User Story:** As a developer, I want the cache-service to use all available shared libraries, so that we eliminate code duplication and maintain consistency.

#### Acceptance Criteria

1. THE Cache_Service SHALL replace `internal/circuitbreaker/` with `libs/go/src/fault/circuitbreaker.go`
2. THE Cache_Service SHALL replace `internal/broker/retry.go` with `libs/go/src/fault/retry.go`
3. THE Cache_Service SHALL replace `internal/observability/context.go` with `libs/go/src/observability/context.go`
4. THE Cache_Service SHALL use `libs/go/src/http/middleware.go` for HTTP middleware
5. THE Cache_Service SHALL use `libs/go/src/http/health.go` for health check handlers
6. THE Cache_Service SHALL use `libs/go/src/grpc/errors.go` for gRPC error conversion
7. THE Cache_Service SHALL use `libs/go/src/config/config.go` for configuration loading
8. THE Cache_Service SHALL use `libs/go/src/server/shutdown.go` for graceful shutdown
9. THE Cache_Service SHALL use `libs/go/src/functional/result.go` for error handling patterns
10. WHEN migrating, THE Cache_Service SHALL maintain backward compatibility with existing API contracts

### Requirement 6: Atualizar go.mod do Cache-Service

**User Story:** As a developer, I want the cache-service to properly depend on libs/go modules, so that I can use the shared libraries.

#### Acceptance Criteria

1. THE Cache_Service go.mod SHALL include dependencies for all used libs/go modules
2. THE Cache_Service SHALL use Go workspace or replace directives for local development
3. WHEN building, THE Cache_Service SHALL resolve all libs/go dependencies correctly

### Requirement 7: Remover Código Duplicado do Cache-Service

**User Story:** As a platform architect, I want to remove duplicated code from cache-service after migration, so that we have a single source of truth.

#### Acceptance Criteria

1. THE Cache_Service SHALL delete `internal/circuitbreaker/` directory after migration
2. THE Cache_Service SHALL delete `internal/broker/retry.go` after migration
3. THE Cache_Service SHALL delete `internal/observability/context.go` after migration
4. THE Cache_Service SHALL delete `internal/localcache/` directory (will use Cache_Client for local fallback)
5. WHEN deleting code, THE Cache_Service SHALL ensure all tests pass with new implementations

### Requirement 8: Manter Compatibilidade de API

**User Story:** As a service consumer, I want the cache-service API to remain unchanged, so that my integrations continue working.

#### Acceptance Criteria

1. THE Cache_Service gRPC API SHALL maintain the same protobuf contract
2. THE Cache_Service HTTP API SHALL maintain the same endpoints and response formats
3. THE Cache_Service health endpoints SHALL continue reporting the same status structure
4. WHEN migrating internally, THE Cache_Service SHALL not change any public interfaces
