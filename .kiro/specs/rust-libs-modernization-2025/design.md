# Design Document: Rust Libraries Modernization 2025

## Overview

This design document describes the architecture and implementation approach for modernizing the `libs/rust` directory to December 2025 state-of-the-art standards. The modernization creates a unified Cargo workspace with centralized dependencies, eliminates code redundancy, integrates with platform services (Logging_Service and Cache_Service), and ensures comprehensive property-based testing.

## Architecture

### High-Level Structure

```
libs/rust/
├── Cargo.toml                    # Workspace root
├── rust-common/                  # Shared library (NEW)
│   ├── Cargo.toml
│   └── src/
│       ├── lib.rs
│       ├── error.rs              # Centralized error types
│       ├── http.rs               # HTTP client builder
│       ├── retry.rs              # Retry policy
│       ├── circuit_breaker.rs    # Circuit breaker
│       ├── logging_client.rs     # Logging_Service gRPC client
│       ├── cache_client.rs       # Cache_Service gRPC client
│       ├── tracing.rs            # OpenTelemetry integration
│       └── metrics.rs            # Prometheus helpers
├── caep/                         # CAEP library (MODERNIZED)
│   ├── Cargo.toml
│   ├── src/
│   └── tests/
├── vault/                        # Vault client (MODERNIZED)
│   ├── Cargo.toml
│   ├── src/
│   └── tests/
├── linkerd/                      # Linkerd types (MODERNIZED)
│   ├── Cargo.toml
│   ├── src/
│   └── tests/
├── pact/                         # Pact types (MODERNIZED)
│   ├── Cargo.toml
│   ├── src/
│   └── tests/
├── integration/                  # Integration tests (MODERNIZED)
│   ├── Cargo.toml
│   └── tests/
└── test-utils/                   # Shared test utilities (NEW)
    ├── Cargo.toml
    └── src/
```


### Workspace Configuration

The root `Cargo.toml` defines workspace-level configuration:

```toml
[workspace]
resolver = "2"
members = [
    "rust-common",
    "caep",
    "vault",
    "linkerd",
    "pact",
    "integration",
    "test-utils",
]

[workspace.package]
edition = "2024"
license = "Proprietary"
repository = "https://github.com/auth-platform/auth-platform"

[workspace.dependencies]
# Async runtime
tokio = { version = "1.42", features = ["full"] }

# HTTP client
reqwest = { version = "0.12.24", features = ["json", "rustls-tls"] }

# Serialization
serde = { version = "1.0", features = ["derive"] }
serde_json = "1.0"

# Error handling
thiserror = "2.0"

# JWT
jsonwebtoken = "9.3"

# Time
chrono = { version = "0.4.42", features = ["serde"] }

# Security
secrecy = { version = "0.10", features = ["serde"] }
zeroize = { version = "1.8", features = ["derive"] }
subtle = "2.6"

# Observability
tracing = "0.1"
tracing-subscriber = { version = "0.3", features = ["json"] }
opentelemetry = "0.27"
opentelemetry-otlp = "0.27"

# gRPC
tonic = "0.12"
prost = "0.13"

# Testing
proptest = "1.7"
tokio-test = "0.4"
wiremock = "0.6"

# Utilities
uuid = { version = "1.11", features = ["v4", "serde"] }
base64 = "0.22"

[workspace.lints.rust]
unsafe_code = "forbid"
missing_docs = "warn"

[workspace.lints.clippy]
all = "warn"
pedantic = "warn"
nursery = "warn"
```

## Components and Interfaces


### rust-common Crate

The `rust-common` crate provides shared functionality for all Rust libraries.

#### Error Types

```rust
//! Centralized error types for all Rust libraries.

use thiserror::Error;

/// Common error type for platform operations
#[derive(Error, Debug)]
pub enum PlatformError {
    #[error("HTTP request failed: {0}")]
    Http(#[from] reqwest::Error),

    #[error("gRPC error: {0}")]
    Grpc(#[from] tonic::Status),

    #[error("Serialization error: {0}")]
    Serialization(#[from] serde_json::Error),

    #[error("Circuit breaker open for {service}")]
    CircuitOpen { service: String },

    #[error("Service unavailable: {0}")]
    Unavailable(String),

    #[error("Authentication failed: {0}")]
    AuthFailed(String),

    #[error("Not found: {0}")]
    NotFound(String),

    #[error("Rate limited")]
    RateLimited,
}

impl PlatformError {
    /// Check if error is retryable
    pub fn is_retryable(&self) -> bool {
        matches!(
            self,
            Self::Http(_) | Self::Unavailable(_) | Self::RateLimited
        )
    }
}
```

#### HTTP Client Builder

```rust
//! Centralized HTTP client configuration.

use reqwest::{Client, ClientBuilder};
use std::time::Duration;

/// HTTP client configuration
#[derive(Debug, Clone)]
pub struct HttpConfig {
    pub timeout: Duration,
    pub connect_timeout: Duration,
    pub pool_idle_timeout: Duration,
    pub pool_max_idle_per_host: usize,
    pub user_agent: String,
}

impl Default for HttpConfig {
    fn default() -> Self {
        Self {
            timeout: Duration::from_secs(30),
            connect_timeout: Duration::from_secs(10),
            pool_idle_timeout: Duration::from_secs(90),
            pool_max_idle_per_host: 10,
            user_agent: "auth-platform-rust/1.0".to_string(),
        }
    }
}

/// Build configured HTTP client
pub fn build_http_client(config: &HttpConfig) -> Result<Client, reqwest::Error> {
    ClientBuilder::new()
        .timeout(config.timeout)
        .connect_timeout(config.connect_timeout)
        .pool_idle_timeout(config.pool_idle_timeout)
        .pool_max_idle_per_host(config.pool_max_idle_per_host)
        .user_agent(&config.user_agent)
        .use_rustls_tls()
        .build()
}
```


#### Circuit Breaker

```rust
//! Circuit breaker implementation for resilience.

use std::sync::atomic::{AtomicU32, AtomicU64, Ordering};
use std::sync::Arc;
use std::time::{Duration, Instant};
use tokio::sync::RwLock;

/// Circuit breaker state
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum CircuitState {
    Closed,
    Open,
    HalfOpen,
}

/// Circuit breaker configuration
#[derive(Debug, Clone)]
pub struct CircuitBreakerConfig {
    pub failure_threshold: u32,
    pub success_threshold: u32,
    pub timeout: Duration,
    pub half_open_max_requests: u32,
}

impl Default for CircuitBreakerConfig {
    fn default() -> Self {
        Self {
            failure_threshold: 5,
            success_threshold: 2,
            timeout: Duration::from_secs(30),
            half_open_max_requests: 3,
        }
    }
}

/// Circuit breaker for protecting external services
pub struct CircuitBreaker {
    config: CircuitBreakerConfig,
    state: RwLock<CircuitState>,
    failures: AtomicU32,
    successes: AtomicU32,
    last_failure: RwLock<Option<Instant>>,
    half_open_requests: AtomicU32,
}

impl CircuitBreaker {
    pub fn new(config: CircuitBreakerConfig) -> Self {
        Self {
            config,
            state: RwLock::new(CircuitState::Closed),
            failures: AtomicU32::new(0),
            successes: AtomicU32::new(0),
            last_failure: RwLock::new(None),
            half_open_requests: AtomicU32::new(0),
        }
    }

    /// Check if request is allowed
    pub async fn allow_request(&self) -> bool {
        let state = *self.state.read().await;
        match state {
            CircuitState::Closed => true,
            CircuitState::Open => {
                if let Some(last) = *self.last_failure.read().await {
                    if last.elapsed() >= self.config.timeout {
                        *self.state.write().await = CircuitState::HalfOpen;
                        self.half_open_requests.store(0, Ordering::SeqCst);
                        true
                    } else {
                        false
                    }
                } else {
                    false
                }
            }
            CircuitState::HalfOpen => {
                self.half_open_requests.fetch_add(1, Ordering::SeqCst)
                    < self.config.half_open_max_requests
            }
        }
    }

    /// Record successful request
    pub async fn record_success(&self) {
        let state = *self.state.read().await;
        match state {
            CircuitState::HalfOpen => {
                let successes = self.successes.fetch_add(1, Ordering::SeqCst) + 1;
                if successes >= self.config.success_threshold {
                    *self.state.write().await = CircuitState::Closed;
                    self.failures.store(0, Ordering::SeqCst);
                    self.successes.store(0, Ordering::SeqCst);
                }
            }
            CircuitState::Closed => {
                self.failures.store(0, Ordering::SeqCst);
            }
            _ => {}
        }
    }

    /// Record failed request
    pub async fn record_failure(&self) {
        let failures = self.failures.fetch_add(1, Ordering::SeqCst) + 1;
        *self.last_failure.write().await = Some(Instant::now());

        if failures >= self.config.failure_threshold {
            *self.state.write().await = CircuitState::Open;
        }
    }

    /// Get current state
    pub async fn state(&self) -> CircuitState {
        *self.state.read().await
    }
}
```


#### Logging Service Client

```rust
//! gRPC client for centralized Logging_Service.

use crate::{CircuitBreaker, CircuitBreakerConfig, PlatformError};
use std::collections::VecDeque;
use std::sync::Arc;
use std::time::Duration;
use tokio::sync::{mpsc, RwLock};
use tonic::transport::Channel;
use tracing::{error, info, warn};

/// Log level matching Logging_Service
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
#[repr(i32)]
pub enum LogLevel {
    Debug = 1,
    Info = 2,
    Warn = 3,
    Error = 4,
    Fatal = 5,
}

/// Log entry for sending to Logging_Service
#[derive(Debug, Clone)]
pub struct LogEntry {
    pub level: LogLevel,
    pub message: String,
    pub service_id: String,
    pub correlation_id: Option<String>,
    pub trace_id: Option<String>,
    pub span_id: Option<String>,
    pub metadata: std::collections::HashMap<String, String>,
}

/// Logging client configuration
#[derive(Debug, Clone)]
pub struct LoggingClientConfig {
    pub address: String,
    pub batch_size: usize,
    pub flush_interval: Duration,
    pub buffer_size: usize,
    pub service_id: String,
    pub circuit_breaker: CircuitBreakerConfig,
}

impl Default for LoggingClientConfig {
    fn default() -> Self {
        Self {
            address: "http://localhost:50052".to_string(),
            batch_size: 100,
            flush_interval: Duration::from_secs(5),
            buffer_size: 10000,
            service_id: "rust-service".to_string(),
            circuit_breaker: CircuitBreakerConfig::default(),
        }
    }
}

/// Logging client with batching and circuit breaker
pub struct LoggingClient {
    config: LoggingClientConfig,
    buffer: Arc<RwLock<VecDeque<LogEntry>>>,
    circuit_breaker: Arc<CircuitBreaker>,
    channel: Option<Channel>,
}

impl LoggingClient {
    pub async fn new(config: LoggingClientConfig) -> Result<Self, PlatformError> {
        let channel = Channel::from_shared(config.address.clone())
            .map_err(|e| PlatformError::Unavailable(e.to_string()))?
            .connect_timeout(Duration::from_secs(5))
            .connect()
            .await
            .ok();

        Ok(Self {
            circuit_breaker: Arc::new(CircuitBreaker::new(config.circuit_breaker.clone())),
            buffer: Arc::new(RwLock::new(VecDeque::with_capacity(config.buffer_size))),
            config,
            channel,
        })
    }

    /// Log a message (buffered)
    pub async fn log(&self, entry: LogEntry) {
        let mut buffer = self.buffer.write().await;
        if buffer.len() < self.config.buffer_size {
            buffer.push_back(entry);
        }

        if buffer.len() >= self.config.batch_size {
            drop(buffer);
            self.flush().await;
        }
    }

    /// Flush buffered logs to Logging_Service
    pub async fn flush(&self) {
        if !self.circuit_breaker.allow_request().await {
            warn!("Logging circuit breaker open, falling back to local tracing");
            self.fallback_to_local().await;
            return;
        }

        let entries: Vec<LogEntry> = {
            let mut buffer = self.buffer.write().await;
            buffer.drain(..).collect()
        };

        if entries.is_empty() {
            return;
        }

        if let Some(ref _channel) = self.channel {
            // Send via gRPC (implementation details omitted)
            // On success: self.circuit_breaker.record_success().await;
            // On failure: self.circuit_breaker.record_failure().await;
            self.circuit_breaker.record_success().await;
        } else {
            self.circuit_breaker.record_failure().await;
            self.fallback_to_local().await;
        }
    }

    async fn fallback_to_local(&self) {
        let buffer = self.buffer.read().await;
        for entry in buffer.iter() {
            match entry.level {
                LogLevel::Debug => tracing::debug!("{}", entry.message),
                LogLevel::Info => tracing::info!("{}", entry.message),
                LogLevel::Warn => tracing::warn!("{}", entry.message),
                LogLevel::Error => tracing::error!("{}", entry.message),
                LogLevel::Fatal => tracing::error!(fatal = true, "{}", entry.message),
            }
        }
    }
}
```


#### Cache Service Client

```rust
//! gRPC client for centralized Cache_Service.

use crate::{CircuitBreaker, CircuitBreakerConfig, PlatformError};
use aes_gcm::{aead::Aead, Aes256Gcm, KeyInit, Nonce};
use std::collections::HashMap;
use std::sync::Arc;
use std::time::Duration;
use tokio::sync::RwLock;
use tonic::transport::Channel;

/// Cache client configuration
#[derive(Debug, Clone)]
pub struct CacheClientConfig {
    pub address: String,
    pub namespace: String,
    pub default_ttl: Duration,
    pub local_cache_size: usize,
    pub encryption_key: Option<[u8; 32]>,
    pub circuit_breaker: CircuitBreakerConfig,
}

impl Default for CacheClientConfig {
    fn default() -> Self {
        Self {
            address: "http://localhost:50051".to_string(),
            namespace: "default".to_string(),
            default_ttl: Duration::from_secs(3600),
            local_cache_size: 1000,
            encryption_key: None,
            circuit_breaker: CircuitBreakerConfig::default(),
        }
    }
}

/// Local cache entry
struct LocalCacheEntry {
    value: Vec<u8>,
    expires_at: std::time::Instant,
}

/// Cache client with local fallback and encryption
pub struct CacheClient {
    config: CacheClientConfig,
    circuit_breaker: Arc<CircuitBreaker>,
    local_cache: Arc<RwLock<HashMap<String, LocalCacheEntry>>>,
    cipher: Option<Aes256Gcm>,
    channel: Option<Channel>,
}

impl CacheClient {
    pub async fn new(config: CacheClientConfig) -> Result<Self, PlatformError> {
        let cipher = config.encryption_key.map(|key| Aes256Gcm::new(&key.into()));

        let channel = Channel::from_shared(config.address.clone())
            .map_err(|e| PlatformError::Unavailable(e.to_string()))?
            .connect_timeout(Duration::from_secs(5))
            .connect()
            .await
            .ok();

        Ok(Self {
            circuit_breaker: Arc::new(CircuitBreaker::new(config.circuit_breaker.clone())),
            local_cache: Arc::new(RwLock::new(HashMap::new())),
            cipher,
            config,
            channel,
        })
    }

    /// Get value from cache
    pub async fn get(&self, key: &str) -> Result<Option<Vec<u8>>, PlatformError> {
        let namespaced_key = format!("{}:{}", self.config.namespace, key);

        // Try remote cache first if circuit allows
        if self.circuit_breaker.allow_request().await {
            if let Some(ref _channel) = self.channel {
                // gRPC call to Cache_Service (implementation details omitted)
                self.circuit_breaker.record_success().await;
            }
        }

        // Fallback to local cache
        let cache = self.local_cache.read().await;
        if let Some(entry) = cache.get(&namespaced_key) {
            if entry.expires_at > std::time::Instant::now() {
                return Ok(Some(self.decrypt(&entry.value)?));
            }
        }

        Ok(None)
    }

    /// Set value in cache
    pub async fn set(&self, key: &str, value: &[u8], ttl: Option<Duration>) -> Result<(), PlatformError> {
        let namespaced_key = format!("{}:{}", self.config.namespace, key);
        let ttl = ttl.unwrap_or(self.config.default_ttl);
        let encrypted = self.encrypt(value)?;

        // Try remote cache first if circuit allows
        if self.circuit_breaker.allow_request().await {
            if let Some(ref _channel) = self.channel {
                // gRPC call to Cache_Service (implementation details omitted)
                self.circuit_breaker.record_success().await;
            }
        }

        // Always update local cache
        let mut cache = self.local_cache.write().await;
        cache.insert(
            namespaced_key,
            LocalCacheEntry {
                value: encrypted,
                expires_at: std::time::Instant::now() + ttl,
            },
        );

        // Evict if over size limit
        if cache.len() > self.config.local_cache_size {
            // Simple eviction: remove expired entries
            cache.retain(|_, v| v.expires_at > std::time::Instant::now());
        }

        Ok(())
    }

    fn encrypt(&self, data: &[u8]) -> Result<Vec<u8>, PlatformError> {
        if let Some(ref cipher) = self.cipher {
            let nonce = Nonce::from_slice(&[0u8; 12]); // In production, use random nonce
            cipher
                .encrypt(nonce, data)
                .map_err(|e| PlatformError::Unavailable(e.to_string()))
        } else {
            Ok(data.to_vec())
        }
    }

    fn decrypt(&self, data: &[u8]) -> Result<Vec<u8>, PlatformError> {
        if let Some(ref cipher) = self.cipher {
            let nonce = Nonce::from_slice(&[0u8; 12]);
            cipher
                .decrypt(nonce, data)
                .map_err(|e| PlatformError::Unavailable(e.to_string()))
        } else {
            Ok(data.to_vec())
        }
    }
}
```

## Data Models


### CAEP Data Models (Modernized)

```rust
//! CAEP event types using modern Rust patterns.

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use thiserror::Error;

/// CAEP-specific errors using thiserror 2.0
#[derive(Error, Debug)]
pub enum CaepError {
    #[error("Failed to sign SET: {0}")]
    SigningError(String),

    #[error("Failed to verify SET signature: {0}")]
    VerificationError(String),

    #[error("Invalid SET structure: {0}")]
    InvalidSet(String),

    #[error("Unknown event type: {0}")]
    UnknownEventType(String),

    #[error("Stream not found: {0}")]
    StreamNotFound(String),

    #[error("Stream delivery failed: {0}")]
    DeliveryFailed(String),

    #[error("JWKS fetch failed: {0}")]
    JwksFetchError(String),

    #[error("Processing error: {0}")]
    ProcessingError(String),

    #[error(transparent)]
    Platform(#[from] rust_common::PlatformError),
}

/// CAEP event types conforming to OpenID CAEP 1.0
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq, Hash)]
#[serde(rename_all = "kebab-case")]
pub enum CaepEventType {
    SessionRevoked,
    CredentialChange,
    AssuranceLevelChange,
    TokenClaimsChange,
    DeviceComplianceChange,
}

impl CaepEventType {
    pub const fn uri(&self) -> &'static str {
        match self {
            Self::SessionRevoked => "https://schemas.openid.net/secevent/caep/event-type/session-revoked",
            Self::CredentialChange => "https://schemas.openid.net/secevent/caep/event-type/credential-change",
            Self::AssuranceLevelChange => "https://schemas.openid.net/secevent/caep/event-type/assurance-level-change",
            Self::TokenClaimsChange => "https://schemas.openid.net/secevent/caep/event-type/token-claims-change",
            Self::DeviceComplianceChange => "https://schemas.openid.net/secevent/caep/event-type/device-compliance-change",
        }
    }
}

/// Subject identifier formats per OpenID SSF
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(tag = "format", rename_all = "snake_case")]
pub enum SubjectIdentifier {
    IssSub { iss: String, sub: String },
    Email { email: String },
    Opaque { id: String },
    SessionId { session_id: String },
}

/// CAEP Event structure
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CaepEvent {
    pub event_type: CaepEventType,
    pub subject: SubjectIdentifier,
    pub event_timestamp: DateTime<Utc>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub reason_admin: Option<String>,
    #[serde(flatten)]
    pub extra: serde_json::Value,
}
```

### Vault Data Models (Modernized)

```rust
//! Vault client types using modern Rust patterns.

use secrecy::{ExposeSecret, SecretString, Zeroize};
use serde::{Deserialize, Serialize};
use std::time::Duration;
use thiserror::Error;

/// Vault-specific errors
#[derive(Error, Debug)]
pub enum VaultError {
    #[error("Vault unavailable: {0}")]
    Unavailable(String),

    #[error("Authentication failed: {0}")]
    AuthenticationFailed(String),

    #[error("Secret not found at path: {0}")]
    SecretNotFound(String),

    #[error("Lease renewal failed: {0}")]
    LeaseRenewalFailed(String),

    #[error("Permission denied: {0}")]
    PermissionDenied(String),

    #[error("Rate limited")]
    RateLimited,

    #[error(transparent)]
    Platform(#[from] rust_common::PlatformError),
}

impl VaultError {
    pub fn is_retryable(&self) -> bool {
        matches!(self, Self::Unavailable(_) | Self::RateLimited)
    }
}

/// Secret metadata
#[derive(Debug, Clone)]
pub struct SecretMetadata {
    pub lease_id: Option<String>,
    pub ttl: Duration,
    pub renewable: bool,
    pub version: Option<u32>,
}

/// Database credentials with automatic zeroization
#[derive(Debug, Clone, Zeroize)]
#[zeroize(drop)]
pub struct DatabaseCredentials {
    pub username: String,
    #[zeroize(skip)]
    pub password: SecretString,
    pub lease_id: String,
    #[zeroize(skip)]
    pub ttl: Duration,
    pub renewable: bool,
}

impl DatabaseCredentials {
    /// Check if credentials should be renewed (at 80% of TTL)
    pub fn should_renew(&self, elapsed: Duration) -> bool {
        elapsed.as_secs_f64() >= self.ttl.as_secs_f64() * 0.8
    }
}

/// Generic trait for secret providers (native async trait - Rust 2024)
pub trait SecretProvider: Send + Sync {
    type Error: std::error::Error + Send + Sync;

    /// Get a secret and deserialize to type T
    fn get_secret<T>(&self, path: &str) -> impl std::future::Future<Output = Result<(T, SecretMetadata), Self::Error>> + Send
    where
        T: serde::de::DeserializeOwned + Send;

    /// Renew a lease
    fn renew_lease(&self, lease_id: &str, increment: Duration) -> impl std::future::Future<Output = Result<Duration, Self::Error>> + Send;
}
```


## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system—essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*

### Property 1: SET Signing Algorithm Default

*For any* Security Event Token created by the CAEP library, the default signing algorithm SHALL be ES256 unless explicitly overridden.

**Validates: Requirements 5.6**

### Property 2: Circuit Breaker State Transitions

*For any* circuit breaker instance, after recording N consecutive failures (where N equals the failure threshold), the circuit SHALL transition to Open state and reject subsequent requests until the timeout expires.

**Validates: Requirements 6.5, 8.3**

### Property 3: Credential Encryption Round-Trip

*For any* credential value cached via the Cache_Client with encryption enabled, encrypting then decrypting SHALL produce the original value.

**Validates: Requirements 6.6, 9.5**

### Property 4: Log Batching Threshold

*For any* sequence of log entries sent to the Logging_Client, when the buffer reaches the configured batch size, the client SHALL flush all buffered entries.

**Validates: Requirements 8.2**

### Property 5: Log Context Propagation

*For any* log entry sent through the Logging_Client, the entry SHALL include correlation ID and trace context when available in the current span.

**Validates: Requirements 8.5**

### Property 6: Cache Namespace Isolation

*For any* two cache operations with different namespaces, keys with the same name SHALL be stored and retrieved independently without collision.

**Validates: Requirements 9.2**

### Property 7: Cache TTL Enforcement

*For any* cached entry with a TTL, after the TTL expires, the entry SHALL NOT be returned by subsequent get operations.

**Validates: Requirements 9.4**

### Property 8: Serialization Round-Trip

*For any* valid domain object (CaepEvent, SecurityEventToken, DatabaseCredentials), serializing to JSON then deserializing SHALL produce an equivalent object.

**Validates: Requirements 10.4**

### Property 9: mTLS Connection Validity

*For any* mTLS connection through Linkerd, both source and destination identities SHALL be valid SPIFFE URIs and certificates SHALL be valid.

**Validates: Requirements 11.1**

### Property 10: Trace Context Propagation

*For any* request traversing multiple services, the W3C Trace Context traceparent header SHALL preserve the trace ID across all hops while updating the parent ID.

**Validates: Requirements 11.2**

### Property 11: Linkerd Latency Overhead

*For any* request through Linkerd proxy, the added latency overhead SHALL NOT exceed 2ms at p99.

**Validates: Requirements 11.3**

### Property 12: Contract Serialization Round-Trip

*For any* Pact contract, serializing to JSON then deserializing SHALL produce an equivalent contract.

**Validates: Requirements 12.1**

### Property 13: Contract Version Git Commit Match

*For any* published contract version, the tags SHALL include the git commit SHA (40 hexadecimal characters).

**Validates: Requirements 12.2**

### Property 14: Input Validation Rejection

*For any* invalid input (malformed JSON, missing required fields, out-of-range values), the system SHALL reject the input with an appropriate error before processing.

**Validates: Requirements 15.3**

### Property 15: Secret Non-Exposure in Debug Output

*For any* type containing secrets (SecretString, DatabaseCredentials), the Debug implementation SHALL NOT expose the secret value in its output.

**Validates: Requirements 15.6**

## Error Handling


### Error Handling Strategy

All errors use `thiserror` 2.0 for derive macros and follow these patterns:

1. **Centralized Error Types**: Common errors in `rust-common::PlatformError`
2. **Domain-Specific Errors**: Each crate has its own error type that wraps `PlatformError`
3. **Error Conversion**: Use `#[from]` for automatic conversion between error types
4. **Retryable Errors**: Each error type implements `is_retryable()` method
5. **No Panic**: All operations return `Result` types, no unwrap in production code

```rust
// Example error handling pattern
use thiserror::Error;

#[derive(Error, Debug)]
pub enum ServiceError {
    #[error("Operation failed: {0}")]
    OperationFailed(String),

    #[error(transparent)]
    Platform(#[from] rust_common::PlatformError),

    #[error(transparent)]
    Vault(#[from] VaultError),
}

impl ServiceError {
    pub fn is_retryable(&self) -> bool {
        match self {
            Self::Platform(e) => e.is_retryable(),
            Self::Vault(e) => e.is_retryable(),
            Self::OperationFailed(_) => false,
        }
    }
}
```

### Fallback Behavior

1. **Logging_Service unavailable**: Fall back to local tracing with structured JSON output
2. **Cache_Service unavailable**: Fall back to local in-memory LRU cache
3. **Vault unavailable**: Use cached credentials within grace period, then fail

## Testing Strategy

### Dual Testing Approach

The testing strategy combines unit tests and property-based tests:

- **Unit tests**: Verify specific examples, edge cases, and error conditions
- **Property tests**: Verify universal properties across all inputs using proptest 1.7

### Property-Based Testing Configuration

- **Library**: proptest 1.7
- **Minimum iterations**: 100 per property test
- **Tag format**: `**Feature: rust-libs-modernization-2025, Property N: [property_text]**`

### Test Organization

```
libs/rust/
├── test-utils/
│   └── src/
│       ├── lib.rs
│       ├── generators.rs      # Shared proptest generators
│       ├── mocks.rs           # Mock implementations
│       └── fixtures.rs        # Test fixtures
├── caep/
│   └── tests/
│       └── property_tests.rs  # CAEP property tests
├── vault/
│   └── tests/
│       └── property_tests.rs  # Vault property tests
├── linkerd/
│   └── tests/
│       └── property_tests.rs  # Linkerd property tests
├── pact/
│   └── tests/
│       └── property_tests.rs  # Pact property tests
└── integration/
    └── tests/
        └── e2e_tests.rs       # End-to-end integration tests
```

### Shared Generators

```rust
//! Shared proptest generators for all Rust libraries.

use proptest::prelude::*;

/// Generate valid CAEP event types
pub fn caep_event_type_strategy() -> impl Strategy<Value = CaepEventType> {
    prop_oneof![
        Just(CaepEventType::SessionRevoked),
        Just(CaepEventType::CredentialChange),
        Just(CaepEventType::AssuranceLevelChange),
        Just(CaepEventType::TokenClaimsChange),
        Just(CaepEventType::DeviceComplianceChange),
    ]
}

/// Generate valid subject identifiers
pub fn subject_identifier_strategy() -> impl Strategy<Value = SubjectIdentifier> {
    prop_oneof![
        ("[a-z]{5,20}", "[a-z0-9]{10,30}").prop_map(|(iss, sub)| {
            SubjectIdentifier::IssSub {
                iss: format!("https://{}.example.com", iss),
                sub,
            }
        }),
        "[a-z0-9._%+-]+@[a-z0-9.-]+\\.[a-z]{2,4}".prop_map(|email| {
            SubjectIdentifier::Email { email }
        }),
        "[a-z0-9]{32}".prop_map(|id| SubjectIdentifier::Opaque { id }),
    ]
}

/// Generate valid SPIFFE identities
pub fn spiffe_identity_strategy() -> impl Strategy<Value = String> {
    "[a-z][a-z0-9-]{2,20}".prop_map(|name| {
        format!("spiffe://auth-platform.local/ns/auth-platform/sa/{}", name)
    })
}

/// Generate W3C Trace Context traceparent
pub fn traceparent_strategy() -> impl Strategy<Value = String> {
    (
        Just("00"),
        "[0-9a-f]{32}",
        "[0-9a-f]{16}",
        prop_oneof![Just("00"), Just("01")],
    ).prop_map(|(version, trace_id, parent_id, flags)| {
        format!("{}-{}-{}-{}", version, trace_id, parent_id, flags)
    })
}

/// Generate valid secret paths
pub fn secret_path_strategy() -> impl Strategy<Value = String> {
    "[a-z][a-z0-9-]{0,20}(/[a-z][a-z0-9-]{0,20}){0,3}"
}

/// Generate TTL values (1 minute to 24 hours)
pub fn ttl_strategy() -> impl Strategy<Value = std::time::Duration> {
    (60u64..86400).prop_map(std::time::Duration::from_secs)
}
```
