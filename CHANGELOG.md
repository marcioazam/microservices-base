# Changelog

All notable changes to the Auth Platform will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Python SDK: Enhanced error hierarchy with standardized `ErrorCode` enum (AUTH_1xxx, VAL_2xxx, NET_3xxx, RATE_4xxx, SRV_5xxx, DPOP_6xxx, PKCE_7xxx)
- Python SDK: New error classes - `TokenInvalidError`, `TimeoutError`, `DPoPError`, `PKCEError`, `ServerError`
- Python SDK: Correlation ID support on all errors for distributed tracing
- Python SDK: `to_dict()` method on errors for structured logging/serialization
- Python SDK: JWKS cache refresh-ahead support (`refresh_ahead_seconds` parameter)
- Python SDK: `JWKSCache.get_key_by_id()` method for direct key lookup
- Python SDK: `AsyncJWKSCache.get_all_signing_keys()` method
- Python SDK: `is_cached` property on both sync and async JWKS caches
- Python SDK: Configurable `http_timeout` for JWKS fetching
- Python SDK: Circuit breaker pattern for HTTP resilience (`CircuitBreaker` class)
- Python SDK: HTTP client factories with OpenTelemetry integration (`create_http_client`, `create_async_http_client`)
- Python SDK: Retry utilities with circuit breaker support (`request_with_retry`, `async_request_with_retry`)

### Changed
- Python SDK: `AsyncJWKSCache` now uses `asyncio.Lock()` instead of threading lock for proper async safety
- Python SDK: JWKS cache now uses typed `JWK` and `JWKS` models instead of raw dictionaries
- Python SDK: Improved exception chaining with `from e` for better error traceability

### Changed
- Reorganized monorepo structure for state-of-the-art 2025 patterns
- Moved Rust libraries from `libs/go/` to `libs/rust/`
- Moved Postman collection to `docs/api/postman/`
- Removed empty `infra/` directory (consolidated with `platform/`)
- Added `docs/adr/` for Architecture Decision Records

## [2.0.0] - 2025-12-01

### Added
- HashiCorp Vault integration for secrets management
- Linkerd service mesh with automatic mTLS
- Pact contract testing for gRPC services
- Property-based testing with 15 correctness properties
- CAEP (Continuous Access Evaluation Protocol) support
- Passkeys/WebAuthn Level 2 support

### Changed
- Upgraded to OAuth 2.1 with mandatory PKCE
- Migrated to Kubernetes Gateway API v1.4
- Enhanced observability with OpenTelemetry

### Security
- Implemented DPoP (RFC 9449) for sender-constrained tokens
- Added constant-time comparison for all secret operations
- Zero-trust architecture with SPIFFE/SPIRE

## [1.0.0] - 2024-06-01

### Added
- Initial release of Auth Platform
- OAuth 2.0 with PKCE support
- JWT token service with key rotation
- Session management with event sourcing
- IAM policy service with OPA
- MFA service (TOTP, WebAuthn)
- Envoy Gateway integration
