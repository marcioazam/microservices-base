# Auth Platform Rust Libraries

Modernized Rust workspace for Auth Platform - December 2025.

## Crates

| Crate | Description |
|-------|-------------|
| `rust-common` | Shared error types, HTTP client, retry, circuit breaker, platform clients |
| `auth-caep` | CAEP (Continuous Access Evaluation Protocol) implementation |
| `auth-vault-client` | HashiCorp Vault client with circuit breaker |
| `auth-linkerd` | Linkerd service mesh types (mTLS, tracing, metrics) |
| `auth-pact` | Pact contract testing types |
| `test-utils` | Shared test utilities and proptest generators |
| `auth-integration-tests` | Cross-library integration tests |

## Features

- **Rust 2024 Edition** with native async traits
- **Workspace dependency inheritance** - single source of truth
- **Platform service integration** - Logging_Service and Cache_Service gRPC clients
- **Comprehensive property-based testing** - 15 correctness properties
- **Circuit breaker resilience** - automatic failure isolation
- **Zero redundancy** - centralized error handling and HTTP configuration

## Quick Start

```bash
# Build all crates
cargo build --workspace

# Run all tests
cargo test --workspace

# Run clippy
cargo clippy --workspace

# Generate docs
cargo doc --workspace --open
```

## Architecture

```
libs/rust/
├── Cargo.toml          # Workspace root
├── rust-common/        # Shared utilities
├── caep/               # CAEP protocol
├── vault/              # Vault client
├── linkerd/            # Service mesh types
├── pact/               # Contract testing
├── test-utils/         # Test utilities
└── integration/        # Integration tests
```

## Platform Integration

### Logging Service (gRPC :5001)
```rust
use rust_common::{LoggingClient, LoggingClientConfig, LogEntry, LogLevel};

let client = LoggingClient::new(LoggingClientConfig::default()).await?;
client.log(LogEntry::new(LogLevel::Info, "message")).await?;
```

### Cache Service (gRPC :50051)
```rust
use rust_common::{CacheClient, CacheClientConfig};

let client = CacheClient::new(CacheClientConfig::default()).await?;
client.set("key", b"value", Some(Duration::from_secs(300))).await?;
```

## Property Tests

All libraries include property-based tests validating correctness properties:

1. SET Signing Algorithm Default (ES256)
2. Circuit Breaker State Transitions
3. Credential Encryption Round-Trip
4. Log Batching Threshold
5. Log Context Propagation
6. Cache Namespace Isolation
7. Cache TTL Enforcement
8. Serialization Round-Trip
9. mTLS Connection Validity
10. Trace Context Propagation
11. Linkerd Latency Overhead
12. Contract Serialization Round-Trip
13. Contract Version Git Commit Match
14. Input Validation Rejection
15. Secret Non-Exposure in Debug Output

## License

Proprietary - Auth Platform
