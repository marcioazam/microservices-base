# Changelog

All notable changes to the Auth Platform will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
