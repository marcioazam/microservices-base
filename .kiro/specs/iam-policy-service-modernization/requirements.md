# Requirements Document

## Introduction

This document specifies the requirements for modernizing the IAM Policy Service to state-of-the-art December 2025 standards. The service provides policy decision point (PDP) functionality implementing RBAC and ABAC using Open Policy Agent (OPA). The modernization focuses on eliminating redundancy, integrating with platform services (logging-service, cache-service), adopting Go 1.24+ features, and achieving production-ready architecture with comprehensive observability.

## Glossary

- **IAM_Policy_Service**: The policy decision point microservice that evaluates authorization requests
- **Policy_Engine**: The OPA-based component that evaluates Rego policies
- **RBAC_Module**: Role-Based Access Control module with hierarchical role support
- **ABAC_Module**: Attribute-Based Access Control module for fine-grained policies
- **Cache_Client**: Client for distributed caching via platform/cache-service
- **Logging_Client**: Client for centralized logging via platform/logging-service
- **CAEP_Emitter**: Continuous Access Evaluation Protocol event emitter
- **gRPC_Handler**: Handler for gRPC authorization requests
- **Config_Manager**: Centralized configuration management component
- **Health_Manager**: Health check and readiness probe manager
- **Observability_Provider**: OpenTelemetry-based tracing and metrics provider

## Requirements

### Requirement 1: Centralized Configuration Management

**User Story:** As a platform operator, I want centralized configuration management, so that I can configure the service consistently using environment variables and config files.

#### Acceptance Criteria

1. THE Config_Manager SHALL load configuration from environment variables with prefix `IAM_POLICY_`
2. THE Config_Manager SHALL support YAML configuration file loading as fallback
3. THE Config_Manager SHALL validate all required configuration keys at startup
4. IF a required configuration key is missing, THEN THE Config_Manager SHALL return a descriptive error and prevent service startup
5. THE Config_Manager SHALL use the centralized `libs/go/src/config` package for configuration loading
6. THE Config_Manager SHALL support hot-reload of non-critical configuration values

### Requirement 2: Platform Service Integration

**User Story:** As a platform developer, I want the IAM Policy Service to integrate with platform services, so that I can leverage centralized logging and caching infrastructure.

#### Acceptance Criteria

1. THE IAM_Policy_Service SHALL integrate with logging-service via `libs/go/src/logging` client
2. THE IAM_Policy_Service SHALL integrate with cache-service via `libs/go/src/cache` client
3. WHEN logging-service is unavailable, THE Logging_Client SHALL fallback to local structured logging
4. WHEN cache-service is unavailable, THE Cache_Client SHALL fallback to local in-memory cache
5. THE IAM_Policy_Service SHALL use namespace `iam-policy` for all cache operations
6. THE IAM_Policy_Service SHALL include correlation_id, trace_id, and span_id in all log entries

### Requirement 3: Policy Engine Modernization

**User Story:** As a security engineer, I want a modernized policy engine, so that I can evaluate authorization requests efficiently with caching support.

#### Acceptance Criteria

1. THE Policy_Engine SHALL load and compile Rego policies from the configured policy path
2. THE Policy_Engine SHALL cache compiled policy queries for performance
3. WHEN a policy file changes, THE Policy_Engine SHALL hot-reload the policy without service restart
4. THE Policy_Engine SHALL support both RBAC and ABAC policy evaluation
5. WHEN evaluating a policy, THE Policy_Engine SHALL return allow/deny decision, matched policy ID, and matched rules
6. THE Policy_Engine SHALL cache authorization decisions with configurable TTL
7. IF a cached decision exists, THEN THE Policy_Engine SHALL return it without re-evaluation
8. THE Policy_Engine SHALL invalidate cached decisions when policies are reloaded

### Requirement 4: RBAC Role Hierarchy

**User Story:** As an identity administrator, I want hierarchical role support, so that I can define role inheritance for permission management.

#### Acceptance Criteria

1. THE RBAC_Module SHALL support hierarchical role definitions with parent-child relationships
2. THE RBAC_Module SHALL resolve effective permissions by traversing the role hierarchy
3. WHEN a role has a parent, THE RBAC_Module SHALL inherit all parent permissions
4. THE RBAC_Module SHALL detect and prevent circular role dependencies
5. THE RBAC_Module SHALL cache resolved permissions per role

### Requirement 5: gRPC Service Implementation

**User Story:** As a service developer, I want a high-performance gRPC API, so that I can make authorization decisions with low latency.

#### Acceptance Criteria

1. THE gRPC_Handler SHALL implement the Authorize RPC for single authorization requests
2. THE gRPC_Handler SHALL implement the BatchAuthorize RPC for multiple authorization requests
3. THE gRPC_Handler SHALL implement the GetUserPermissions RPC to retrieve user permissions
4. THE gRPC_Handler SHALL implement the GetUserRoles RPC to retrieve user roles
5. THE gRPC_Handler SHALL implement the ReloadPolicies RPC to trigger policy hot-reload
6. THE gRPC_Handler SHALL use `libs/go/src/grpc` interceptors for error mapping and logging
7. THE gRPC_Handler SHALL implement gRPC health check protocol for Kubernetes probes
8. WHEN an authorization request is received, THE gRPC_Handler SHALL log the decision with audit details

### Requirement 6: CAEP Event Emission

**User Story:** As a security architect, I want CAEP event emission, so that I can notify downstream systems of authorization-relevant changes.

#### Acceptance Criteria

1. THE CAEP_Emitter SHALL emit assurance-level-change events when user assurance level changes
2. THE CAEP_Emitter SHALL emit token-claims-change events when user roles or permissions change
3. WHEN CAEP is disabled, THE CAEP_Emitter SHALL skip event emission without error
4. THE CAEP_Emitter SHALL include event_type, subject, timestamp, and extra fields in all events
5. IF CAEP transmitter returns error, THEN THE CAEP_Emitter SHALL log the error and continue operation

### Requirement 7: Observability and Tracing

**User Story:** As a platform operator, I want comprehensive observability, so that I can monitor service health and debug issues.

#### Acceptance Criteria

1. THE Observability_Provider SHALL integrate OpenTelemetry for distributed tracing
2. THE Observability_Provider SHALL propagate trace context across gRPC calls
3. THE IAM_Policy_Service SHALL expose Prometheus metrics at `/metrics` endpoint
4. THE IAM_Policy_Service SHALL track authorization decision latency histogram
5. THE IAM_Policy_Service SHALL track cache hit/miss rates
6. THE IAM_Policy_Service SHALL track policy evaluation counts by policy ID
7. THE Health_Manager SHALL implement `/health/live` for liveness probes
8. THE Health_Manager SHALL implement `/health/ready` for readiness probes
9. WHILE the service is shutting down, THE Health_Manager SHALL return unhealthy status

### Requirement 8: Graceful Shutdown

**User Story:** As a platform operator, I want graceful shutdown support, so that I can deploy updates without dropping requests.

#### Acceptance Criteria

1. WHEN SIGTERM or SIGINT is received, THE IAM_Policy_Service SHALL initiate graceful shutdown
2. THE IAM_Policy_Service SHALL stop accepting new requests immediately
3. THE IAM_Policy_Service SHALL wait for in-flight requests to complete with configurable timeout
4. THE IAM_Policy_Service SHALL flush logging buffers before shutdown
5. THE IAM_Policy_Service SHALL close cache and logging client connections
6. THE IAM_Policy_Service SHALL use `libs/go/src/server` shutdown utilities

### Requirement 9: Error Handling and Resilience

**User Story:** As a service developer, I want consistent error handling, so that I can debug issues and handle failures gracefully.

#### Acceptance Criteria

1. THE IAM_Policy_Service SHALL use `libs/go/src/errors` for error construction
2. THE IAM_Policy_Service SHALL use `libs/go/src/fault` for circuit breaker patterns
3. WHEN cache-service fails repeatedly, THE Cache_Client SHALL open circuit breaker
4. WHEN logging-service fails repeatedly, THE Logging_Client SHALL fallback to local logging
5. THE gRPC_Handler SHALL map internal errors to appropriate gRPC status codes
6. THE IAM_Policy_Service SHALL include correlation IDs in all error responses

### Requirement 10: Security Hardening

**User Story:** As a security engineer, I want security hardening, so that I can protect the authorization service from attacks.

#### Acceptance Criteria

1. THE IAM_Policy_Service SHALL validate all input parameters before processing
2. THE IAM_Policy_Service SHALL sanitize log output to prevent log injection
3. THE IAM_Policy_Service SHALL not expose internal error details in responses
4. THE IAM_Policy_Service SHALL support TLS for gRPC connections
5. THE IAM_Policy_Service SHALL rate-limit authorization requests per client

### Requirement 11: Test Architecture

**User Story:** As a developer, I want comprehensive test coverage, so that I can ensure service correctness and prevent regressions.

#### Acceptance Criteria

1. THE IAM_Policy_Service SHALL have unit tests for all core components
2. THE IAM_Policy_Service SHALL have property-based tests for policy evaluation consistency
3. THE IAM_Policy_Service SHALL have integration tests for platform service integration
4. THE IAM_Policy_Service SHALL separate test code from source code in `tests/` directory
5. THE IAM_Policy_Service SHALL use `libs/go/src/testing` for test utilities and generators
6. THE IAM_Policy_Service SHALL achieve minimum 80% code coverage

### Requirement 12: Code Organization and Architecture

**User Story:** As a maintainer, I want clean code organization, so that I can understand and modify the codebase efficiently.

#### Acceptance Criteria

1. THE IAM_Policy_Service SHALL follow the standard Go project layout
2. THE IAM_Policy_Service SHALL have no file exceeding 400 lines
3. THE IAM_Policy_Service SHALL eliminate all code duplication
4. THE IAM_Policy_Service SHALL use generics where appropriate for type safety
5. THE IAM_Policy_Service SHALL centralize all business logic in the internal package
6. THE IAM_Policy_Service SHALL use dependency injection for testability
