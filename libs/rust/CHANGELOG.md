# Changelog

All notable changes to the Auth Platform Rust libraries.

## [0.1.0] - 2025-12-22

### Added

#### rust-common
- Centralized `PlatformError` enum with retryability classification
- `HttpConfig` and `build_http_client` for consistent HTTP configuration
- `RetryPolicy` with exponential backoff and jitter
- `CircuitBreaker` with configurable thresholds
- `LoggingClient` for Logging_Service gRPC integration (port 5001)
- `CacheClient` for Cache_Service gRPC integration (port 50051)
- AES-GCM encryption for cached secrets
- OpenTelemetry tracing configuration
- Prometheus metrics helpers

#### auth-caep
- Modernized to thiserror 2.0
- Native async traits (Rust 2024)
- ES256 as default signing algorithm
- Platform service integration (logging, cache)
- Property tests for SET signing and serialization

#### auth-vault-client
- Native async traits (Rust 2024)
- secrecy 0.10 with zeroize
- Circuit breaker integration
- Structured logging (secrets never logged)
- Property tests for secret non-exposure

#### auth-linkerd
- New crate for service mesh types
- `MtlsConnection` for mTLS validation
- `TraceContext` for W3C trace propagation
- `LinkerdMetrics` for proxy metrics
- Property tests for mTLS, tracing, latency

#### auth-pact
- New crate for contract testing types
- `Contract`, `Interaction`, `Request`, `Response` types
- `VerificationResult` and `ContractVersion`
- `CanIDeployResult` and `MatrixEntry`
- Property tests for serialization and versioning

#### test-utils
- Shared proptest generators
- Mock service clients
- Test fixtures

#### auth-integration-tests
- Cross-library integration tests
- Vault through mesh tests
- Secret rotation continuity tests
- Contract verification tests

### Changed
- Workspace uses Rust 2024 edition
- All dependencies inherited from workspace root
- Removed async-trait macro (native async traits)
- Updated to thiserror 2.0, secrecy 0.10

### Removed
- Duplicate error handling patterns
- Duplicate HTTP client configurations
- Legacy async-trait usage

## Migration Notes

### From async-trait to native async traits
```rust
// Before (async-trait)
#[async_trait]
pub trait SecretProvider {
    async fn get_secret(&self, path: &str) -> Result<Secret>;
}

// After (Rust 2024 native)
pub trait SecretProvider {
    fn get_secret(&self, path: &str) 
        -> impl Future<Output = Result<Secret>> + Send;
}
```

### Using platform services
```rust
// Logging
use rust_common::{LoggingClient, LogEntry, LogLevel};
let client = LoggingClient::new(config).await?;
client.log(LogEntry::new(LogLevel::Info, "message")).await?;

// Cache
use rust_common::CacheClient;
let client = CacheClient::new(config).await?;
client.set("key", value, ttl).await?;
```
