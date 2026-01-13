# Design Document: Auth Edge Service Modernization 2025

## Overview

This design document describes the modernization of the Auth Edge Service to December 2025 state-of-the-art standards. The modernization eliminates redundancies by centralizing shared logic in `libs/rust/rust-common`, integrates with platform services (`cache-service`, `logging-service`), upgrades all dependencies to latest stable versions, and ensures production-ready quality.

### Key Design Principles

1. **Zero Redundancy**: Every behavior exists in exactly one authoritative location
2. **Centralization**: Cross-cutting concerns (errors, circuit breaker, caching, logging) use rust-common
3. **State of the Art**: All dependencies upgraded to December 2025 stable versions
4. **Minimal Dense Code**: Small, direct, expressive implementations
5. **Security First**: No abstraction degrades security posture

## Architecture

### Current Architecture Issues

```
┌─────────────────────────────────────────────────────────────────┐
│                    Auth Edge Service (Current)                   │
├─────────────────────────────────────────────────────────────────┤
│  REDUNDANCIES IDENTIFIED:                                        │
│  ├── circuit_breaker/mod.rs (duplicate of rust-common)          │
│  ├── circuit_breaker/state.rs (duplicate state machine)         │
│  ├── error.rs (duplicate error types, sanitization)             │
│  ├── jwt/validator.rs::has_claim() (duplicate of token.rs)      │
│  └── observability/ (outdated OpenTelemetry 0.21)               │
│                                                                  │
│  MISSING INTEGRATIONS:                                           │
│  ├── No Cache_Service integration (local-only JWK cache)        │
│  └── No Logging_Service integration (local tracing only)        │
│                                                                  │
│  OUTDATED DEPENDENCIES:                                          │
│  ├── rustls 0.21 → 0.23                                         │
│  ├── opentelemetry 0.21 → 0.27                                  │
│  ├── thiserror 1.0 → 2.0                                        │
│  └── failsafe (deprecated) → remove                             │
└─────────────────────────────────────────────────────────────────┘
```

### Target Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    Auth Edge Service (Modernized)                        │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  ┌────────────────────────────────────────────────────────────────┐     │
│  │                    Tower Middleware Stack                       │     │
│  │  ┌─────────┐  ┌─────────┐  ┌──────────┐  ┌────────────────┐   │     │
│  │  │ Tracing │→ │ Timeout │→ │RateLimit │→ │ CircuitBreaker │   │     │
│  │  │ (OTel)  │  │ Layer   │  │ Layer    │  │ (rust-common)  │   │     │
│  │  └─────────┘  └─────────┘  └──────────┘  └────────────────┘   │     │
│  └────────────────────────────────────────────────────────────────┘     │
│                                    │                                     │
│  ┌─────────────────────────────────▼────────────────────────────────┐   │
│  │                      Core Services                                │   │
│  │  ┌─────────────────┐  ┌─────────────────┐  ┌──────────────────┐  │   │
│  │  │   JWT Module    │  │  mTLS/SPIFFE    │  │   JWK Cache      │  │   │
│  │  │                 │  │                 │  │                  │  │   │
│  │  │ Token<State>    │  │ SpiffeValidator │  │ CacheClient      │  │   │
│  │  │ (preserved)     │  │ (consolidated)  │  │ (rust-common)    │  │   │
│  │  └─────────────────┘  └─────────────────┘  └──────────────────┘  │   │
│  └──────────────────────────────────────────────────────────────────┘   │
│                                    │                                     │
│  ┌─────────────────────────────────▼────────────────────────────────┐   │
│  │                     Shared Infrastructure                         │   │
│  │  ┌─────────────────┐  ┌─────────────────┐  ┌──────────────────┐  │   │
│  │  │  PlatformError  │  │  LoggingClient  │  │  CircuitBreaker  │  │   │
│  │  │  (rust-common)  │  │  (rust-common)  │  │  (rust-common)   │  │   │
│  │  └─────────────────┘  └─────────────────┘  └──────────────────┘  │   │
│  └──────────────────────────────────────────────────────────────────┘   │
│                                    │                                     │
│  ┌─────────────────────────────────▼────────────────────────────────┐   │
│  │                     gRPC Service Layer                            │   │
│  │  ValidateToken │ IntrospectToken │ GetServiceIdentity             │   │
│  └──────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                    ┌───────────────┼───────────────┐
                    ▼               ▼               ▼
            ┌─────────────┐ ┌─────────────┐ ┌─────────────┐
            │   Cache     │ │   Logging   │ │  Downstream │
            │   Service   │ │   Service   │ │   Services  │
            └─────────────┘ └─────────────┘ └─────────────┘
```

## Components and Interfaces

### 1. Error Handling (Centralized)

**Current State**: Local `AuthEdgeError` with duplicate sanitization logic and error code constants.

**Target State**: Extend `PlatformError` from rust-common with domain-specific variants.

```rust
// src/error.rs - Modernized
use rust_common::PlatformError;
use thiserror::Error;

/// Auth Edge specific errors extending PlatformError
#[non_exhaustive]
#[derive(Error, Debug)]
pub enum AuthEdgeError {
    /// Token was not provided
    #[error("Token missing from request")]
    TokenMissing,

    /// Token signature invalid
    #[error("Token signature invalid")]
    TokenInvalid,

    /// Token has expired
    #[error("Token expired at {expired_at}")]
    TokenExpired { expired_at: chrono::DateTime<chrono::Utc> },

    /// Required claims missing
    #[error("Required claims invalid: {claims:?}")]
    ClaimsInvalid { claims: Vec<String> },

    /// SPIFFE validation failed
    #[error("SPIFFE ID error: {reason}")]
    SpiffeError { reason: String },

    /// Wraps PlatformError for infrastructure errors
    #[error(transparent)]
    Platform(#[from] PlatformError),
}

impl AuthEdgeError {
    /// Check if error is retryable (delegates to PlatformError for infra errors)
    pub fn is_retryable(&self) -> bool {
        match self {
            Self::Platform(e) => e.is_retryable(),
            _ => false,
        }
    }
}
```

### 2. Circuit Breaker (Centralized)

**Current State**: Duplicate implementations in `circuit_breaker/mod.rs` and `circuit_breaker/state.rs`.

**Target State**: Remove local implementation, use `CircuitBreaker` from rust-common.

```rust
// src/grpc/mod.rs - Using rust-common CircuitBreaker
use rust_common::{CircuitBreaker, CircuitBreakerConfig};

pub struct AuthEdgeServiceImpl {
    jwt_validator: JwtValidator,
    token_service_cb: CircuitBreaker,  // From rust-common
    iam_service_cb: CircuitBreaker,    // From rust-common
    logging_client: LoggingClient,     // From rust-common
}

impl AuthEdgeServiceImpl {
    pub async fn new(config: Config) -> Result<Self, AuthEdgeError> {
        let cb_config = CircuitBreakerConfig::default()
            .with_failure_threshold(config.circuit_breaker_failure_threshold)
            .with_timeout(Duration::from_secs(config.circuit_breaker_timeout_seconds));

        Ok(Self {
            jwt_validator: JwtValidator::new(/* ... */),
            token_service_cb: CircuitBreaker::new(cb_config.clone()),
            iam_service_cb: CircuitBreaker::new(cb_config),
            logging_client: LoggingClient::new(/* ... */).await?,
        })
    }
}
```

### 3. JWK Cache with Cache_Service Integration

**Current State**: Local-only cache with single-flight pattern.

**Target State**: Distributed cache via CacheClient with local fallback.

```rust
// src/jwt/jwk_cache.rs - Modernized
use rust_common::{CacheClient, CacheClientConfig, CircuitBreaker};

pub struct JwkCache {
    /// Remote cache client (Cache_Service)
    cache_client: CacheClient,
    /// Local fallback cache
    local_cache: ArcSwap<Option<LocalCacheEntry>>,
    /// Single-flight coordinator
    inflight: Arc<Mutex<Option<InflightFuture>>>,
    /// HTTP client for JWKS fetch
    http_client: reqwest::Client,
    /// Configuration
    config: JwkCacheConfig,
}

impl JwkCache {
    pub async fn new(config: JwkCacheConfig) -> Result<Self, AuthEdgeError> {
        let cache_config = CacheClientConfig::default()
            .with_address(&config.cache_service_url)
            .with_namespace("auth-edge:jwk")
            .with_default_ttl(Duration::from_secs(config.ttl_seconds))
            .with_encryption_key(config.encryption_key);

        Ok(Self {
            cache_client: CacheClient::new(cache_config).await?,
            local_cache: ArcSwap::new(Arc::new(None)),
            inflight: Arc::new(Mutex::new(None)),
            http_client: build_http_client(&HttpConfig::default())?,
            config,
        })
    }

    pub async fn get_key(&self, kid: &str) -> Result<DecodingKey, AuthEdgeError> {
        // 1. Try remote cache first
        if let Some(key_bytes) = self.cache_client.get(&format!("key:{}", kid)).await? {
            return self.deserialize_key(&key_bytes);
        }

        // 2. Try local cache
        if let Some(key) = self.try_get_local(kid) {
            return Ok(key);
        }

        // 3. Refresh with single-flight
        self.refresh_single_flight().await?;
        
        self.try_get_local(kid)
            .ok_or_else(|| AuthEdgeError::Platform(
                PlatformError::NotFound(format!("Key {} not found", kid))
            ))
    }
}
```

### 4. Logging Integration

**Current State**: Local tracing only.

**Target State**: Structured logging via LoggingClient with local fallback.

```rust
// src/observability/logging.rs - New module
use rust_common::{LoggingClient, LoggingClientConfig, LogEntry, LogLevel};

pub struct AuthEdgeLogger {
    client: LoggingClient,
}

impl AuthEdgeLogger {
    pub async fn new(config: &Config) -> Result<Self, AuthEdgeError> {
        let logging_config = LoggingClientConfig::default()
            .with_address(&config.logging_service_url)
            .with_service_id("auth-edge-service")
            .with_batch_size(100);

        Ok(Self {
            client: LoggingClient::new(logging_config).await?,
        })
    }

    pub async fn log_validation_success(&self, subject: &str, correlation_id: &str) {
        let entry = LogEntry::new(LogLevel::Info, "Token validated successfully", "auth-edge-service")
            .with_correlation_id(correlation_id)
            .with_metadata("subject", subject);
        self.client.log(entry).await;
    }

    pub async fn log_validation_failure(&self, error: &AuthEdgeError, correlation_id: &str) {
        let entry = LogEntry::new(LogLevel::Error, format!("Token validation failed: {}", error), "auth-edge-service")
            .with_correlation_id(correlation_id)
            .with_metadata("error_type", error.error_code());
        self.client.log(entry).await;
    }

    pub async fn flush(&self) {
        self.client.flush().await;
    }
}
```

### 5. Consolidated SPIFFE Validation

**Current State**: Separate `SpiffeExtractor` and `SpiffeValidator` with duplicate logic.

**Target State**: Single `SpiffeValidator` with all functionality.

```rust
// src/mtls/spiffe.rs - Consolidated
use std::borrow::Cow;
use std::collections::HashSet;

/// Unified SPIFFE ID validator with extraction and validation
pub struct SpiffeValidator {
    allowed_domains: HashSet<String>,
}

impl SpiffeValidator {
    pub fn new(allowed_domains: Vec<String>) -> Self {
        Self {
            allowed_domains: allowed_domains.into_iter().collect(),
        }
    }

    /// Parse and validate SPIFFE ID from URI
    pub fn parse_and_validate(&self, uri: &str) -> Result<SpiffeId<'_>, SpiffeError> {
        let id = SpiffeId::parse(uri)?;
        self.validate(&id)?;
        Ok(id)
    }

    /// Extract SPIFFE ID from X.509 certificate PEM
    pub fn extract_from_certificate(&self, cert_pem: &str) -> Result<OwnedSpiffeId, SpiffeError> {
        let pem = pem::parse(cert_pem)
            .map_err(|e| SpiffeError::CertificateError(format!("Invalid PEM: {}", e)))?;
        
        let (_, cert) = x509_parser::prelude::X509Certificate::from_der(pem.contents())
            .map_err(|e| SpiffeError::CertificateError(format!("Invalid certificate: {}", e)))?;

        // Extract SPIFFE ID from SAN extension
        for ext in cert.extensions() {
            if let x509_parser::prelude::ParsedExtension::SubjectAlternativeName(san) = ext.parsed_extension() {
                for name in &san.general_names {
                    if let x509_parser::prelude::GeneralName::URI(uri) = name {
                        if uri.starts_with("spiffe://") {
                            let id = OwnedSpiffeId::parse(uri)?;
                            self.validate_owned(&id)?;
                            return Ok(id);
                        }
                    }
                }
            }
        }

        Err(SpiffeError::NotFound)
    }

    /// Extract service name from SPIFFE ID path
    pub fn extract_service_name(spiffe_id: &OwnedSpiffeId) -> Option<String> {
        // Pattern: spiffe://domain/ns/namespace/sa/service-account
        if spiffe_id.path.len() >= 4 && spiffe_id.path[2] == "sa" {
            Some(spiffe_id.path[3].clone())
        } else {
            spiffe_id.path.last().cloned()
        }
    }
}
```

### 6. Modernized Middleware Stack

**Current State**: Multiple middleware files with some redundancy.

**Target State**: Single composable stack using Tower with rust-common components.

```rust
// src/middleware/stack.rs - Modernized
use rust_common::CircuitBreaker;
use tower::ServiceBuilder;

/// Build the complete middleware stack
pub fn build_middleware_stack<S>(
    inner: S,
    config: &Config,
    circuit_breaker: CircuitBreaker,
) -> impl Service<Request<Body>, Response = Response<Body>, Error = AuthEdgeError>
where
    S: Service<Request<Body>, Response = Response<Body>> + Clone + Send + 'static,
    S::Error: Into<AuthEdgeError> + Send + 'static,
    S::Future: Send + 'static,
{
    ServiceBuilder::new()
        // Outermost: OpenTelemetry tracing
        .layer(OtelTracingLayer::new("auth-edge-service"))
        // Timeout enforcement
        .layer(TimeoutLayer::new(Duration::from_secs(config.request_timeout_secs)))
        // Rate limiting
        .layer(RateLimiterLayer::new(config.rate_limit_config()))
        // Circuit breaker (from rust-common)
        .layer(CircuitBreakerLayer::new(circuit_breaker))
        .service(inner)
}
```

## Data Models

### Configuration Model

```rust
// src/config.rs - Modernized with validation
use serde::Deserialize;
use url::Url;

#[derive(Debug, Clone, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct Config {
    // Server
    #[serde(default = "default_host")]
    pub host: String,
    #[serde(default = "default_port")]
    pub port: u16,

    // Service URLs (validated)
    pub token_service_url: Url,
    pub session_service_url: Url,
    pub iam_service_url: Url,
    pub jwks_url: Url,
    pub cache_service_url: Url,
    pub logging_service_url: Url,

    // Cache settings
    #[serde(default = "default_jwks_cache_ttl")]
    pub jwks_cache_ttl_seconds: u64,

    // Circuit breaker
    #[serde(default = "default_cb_failure_threshold")]
    pub circuit_breaker_failure_threshold: u32,
    #[serde(default = "default_cb_timeout")]
    pub circuit_breaker_timeout_seconds: u64,

    // Timeouts
    #[serde(default = "default_request_timeout")]
    pub request_timeout_secs: u64,

    // Security
    #[serde(default)]
    pub allowed_spiffe_domains: Vec<String>,
}

impl Config {
    pub fn from_env() -> Result<Self, ConfigError> {
        dotenvy::dotenv().ok();
        
        let config: Config = config::Config::builder()
            .add_source(config::Environment::default())
            .build()?
            .try_deserialize()?;

        config.validate()?;
        Ok(config)
    }

    fn validate(&self) -> Result<(), ConfigError> {
        if self.port == 0 {
            return Err(ConfigError::InvalidPort);
        }
        if self.jwks_cache_ttl_seconds == 0 {
            return Err(ConfigError::InvalidTtl);
        }
        if self.circuit_breaker_failure_threshold == 0 {
            return Err(ConfigError::InvalidThreshold);
        }
        Ok(())
    }
}
```

### JWT Claims Model (Preserved)

```rust
// src/jwt/claims.rs - Preserved with minor cleanup
use serde::{Deserialize, Serialize};
use std::collections::HashMap;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Claims {
    pub iss: String,
    pub sub: String,
    pub aud: Vec<String>,
    pub exp: i64,
    pub iat: i64,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub nbf: Option<i64>,
    pub jti: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub session_id: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub scopes: Option<Vec<String>>,
    #[serde(flatten)]
    pub custom: HashMap<String, serde_json::Value>,
}

impl Claims {
    pub fn is_expired(&self) -> bool {
        chrono::Utc::now().timestamp() > self.exp
    }

    pub fn has_scope(&self, scope: &str) -> bool {
        self.scopes.as_ref().map_or(false, |s| s.contains(&scope.to_string()))
    }

    /// Single authoritative has_claim implementation
    pub fn has_claim(&self, claim_name: &str) -> bool {
        match claim_name {
            "iss" => !self.iss.is_empty(),
            "sub" => !self.sub.is_empty(),
            "aud" => !self.aud.is_empty(),
            "exp" | "iat" => true,
            "jti" => !self.jti.is_empty(),
            "session_id" => self.session_id.is_some(),
            "scopes" => self.scopes.is_some(),
            _ => self.custom.contains_key(claim_name),
        }
    }
}
```

## Dependency Updates

| Crate | Current | Target | Rationale |
|-------|---------|--------|-----------|
| tokio | 1.35 | 1.42 | Latest async runtime with performance improvements |
| tonic | 0.10 | 0.12 | Latest gRPC with improved streaming |
| prost | 0.12 | 0.13 | Latest protobuf with better codegen |
| jsonwebtoken | 9.2 | 9.3 | Security fixes and algorithm updates |
| thiserror | 1.0 | 2.0 | Improved error derive macros |
| opentelemetry | 0.21 | 0.27 | Latest observability with better OTLP |
| opentelemetry-otlp | 0.14 | 0.27 | Matching opentelemetry version |
| tracing-opentelemetry | 0.22 | 0.28 | Matching opentelemetry version |
| rustls | 0.21 | 0.23 | Latest TLS with security improvements |
| reqwest | 0.11 | 0.12 | Latest HTTP client with rustls 0.23 |
| proptest | 1.4 | 1.5 | Latest property testing |
| failsafe | 1.2 | REMOVE | Deprecated, use rust-common CircuitBreaker |
| borrow | 0.1 | REMOVE | Unused dependency |



## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system—essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*

### Property 1: Error Retryability Classification

*For any* AuthEdgeError, calling `is_retryable()` SHALL return `true` only for transient infrastructure errors (Unavailable, RateLimited, Timeout) and `false` for all domain errors (TokenMissing, TokenInvalid, TokenExpired, ClaimsInvalid, SpiffeError).

**Validates: Requirements 2.3**

### Property 2: Correlation ID Propagation

*For any* error converted to gRPC Status, log entry created, or error event recorded, the output SHALL contain a valid UUID correlation_id that matches the request's correlation_id.

**Validates: Requirements 2.6, 5.3, 8.5**

### Property 3: Circuit Breaker Error Type

*For any* circuit breaker that transitions to Open state, subsequent requests SHALL return `PlatformError::CircuitOpen` with the correct service name until the circuit transitions to HalfOpen.

**Validates: Requirements 3.5**

### Property 4: Cache Fallback Behavior

*For any* JWK cache request, if the remote Cache_Service is unavailable or its circuit breaker is open, the cache SHALL successfully return keys from local cache if they exist and are not expired.

**Validates: Requirements 4.4, 4.5**

### Property 5: Single-Flight Refresh

*For any* N concurrent requests for the same JWK key ID when the cache is stale, exactly one HTTP request SHALL be made to the JWKS endpoint, and all N requests SHALL receive the same result.

**Validates: Requirements 4.6**

### Property 6: Log Level Classification

*For any* authentication failure, the log level SHALL be `LogLevel::Error`. *For any* successful token validation, the log level SHALL be `LogLevel::Info`.

**Validates: Requirements 5.5, 5.6**

### Property 7: Logging Fallback

*For any* log entry, if the Logging_Service is unavailable, the entry SHALL be written to local tracing with the same level, message, and metadata.

**Validates: Requirements 5.7**

### Property 8: W3C Trace Context Propagation

*For any* incoming request with W3C Trace Context headers (traceparent, tracestate), the context SHALL be propagated to all downstream service calls and included in all spans.

**Validates: Requirements 8.2**

### Property 9: Span Attribute Recording

*For any* gRPC method invocation, a span SHALL be created with attributes including: method name, correlation_id, and result status.

**Validates: Requirements 8.3**

### Property 10: JWT Type-State Transitions

*For any* valid JWT string:
- `Token::parse()` SHALL return `Token<Unvalidated>`
- `token.validate_signature()` on `Token<Unvalidated>` SHALL return `Token<SignatureValidated>`
- `token.validate_claims()` on `Token<SignatureValidated>` SHALL return `Token<Validated>`

The `claims()` method SHALL only be callable on `Token<Validated>` (compile-time enforced).

**Validates: Requirements 9.4, 9.5, 9.6**

### Property 11: SPIFFE ID Round-Trip

*For any* valid SPIFFE ID, parsing the URI string and then converting back to URI string SHALL produce an equivalent URI (parse → to_uri → parse produces equivalent SpiffeId).

**Validates: Requirements 10.7**

### Property 12: Configuration Validation

*For any* configuration:
- All URL fields SHALL be valid URLs (parseable by url::Url)
- Port SHALL be in range 1-65535
- TTL values SHALL be > 0
- Missing required fields SHALL produce descriptive ConfigError
- Environment variables SHALL override default values

**Validates: Requirements 11.2, 11.3, 11.4, 11.5**

### Property 13: Graceful Shutdown Behavior

*For any* shutdown initiated by SIGTERM/SIGINT:
- In-flight requests SHALL be allowed to complete up to the configured timeout
- If timeout is exceeded, remaining tasks SHALL be aborted
- LoggingClient buffer SHALL be flushed before completion

**Validates: Requirements 12.3, 12.6**

### Property 14: Sensitive Data Protection

*For any* error message exposed externally or log entry created:
- The content SHALL NOT contain tokens, keys, passwords, or credentials
- Sensitive patterns (password, secret, token, key, credential, bearer, authorization, api_key, private) SHALL be sanitized

**Validates: Requirements 13.2, 13.3, 13.4**

### Property 15: Algorithm Confusion Rejection

*For any* JWT with algorithm "none" or algorithm mismatch between header and expected algorithm, the validation SHALL fail with `AuthEdgeError::TokenInvalid`.

**Validates: Requirements 13.6**

### Property 16: Minimum Key Size Enforcement

*For any* cryptographic key used for JWT validation:
- RSA keys SHALL be at least 2048 bits
- EC keys SHALL use P-256 or stronger curves
- Keys below minimum size SHALL be rejected

**Validates: Requirements 13.7**

## Error Handling

### Error Hierarchy

```
PlatformError (rust-common)
├── Http(reqwest::Error)
├── Grpc(tonic::Status)
├── Serialization(serde_json::Error)
├── CircuitOpen { service: String }
├── Unavailable(String)
├── AuthFailed(String)
├── NotFound(String)
├── RateLimited
├── InvalidInput(String)
├── Encryption(String)
├── Timeout(String)
└── Internal(String)

AuthEdgeError (local)
├── TokenMissing
├── TokenInvalid
├── TokenExpired { expired_at }
├── TokenNotYetValid { valid_from }
├── TokenMalformed { reason }
├── ClaimsInvalid { claims }
├── SpiffeError { reason }
├── CertificateError { reason }
└── Platform(PlatformError)  // Wraps infrastructure errors
```

### Error Sanitization

All errors exposed externally go through sanitization:

```rust
const SENSITIVE_PATTERNS: &[&str] = &[
    "password", "secret", "token", "key", "credential",
    "bearer", "authorization", "api_key", "apikey", "private",
];

fn sanitize_message(message: &str) -> String {
    let lower = message.to_lowercase();
    for pattern in SENSITIVE_PATTERNS {
        if lower.contains(pattern) {
            return "Invalid request".to_string();
        }
    }
    message.to_string()
}
```

### Error to gRPC Status Mapping

| AuthEdgeError | gRPC Code | User Message |
|---------------|-----------|--------------|
| TokenMissing | UNAUTHENTICATED | "Token is required" |
| TokenInvalid | UNAUTHENTICATED | "Token signature is invalid" |
| TokenExpired | UNAUTHENTICATED | "Token has expired" |
| TokenMalformed | INVALID_ARGUMENT | "Invalid token format" |
| ClaimsInvalid | PERMISSION_DENIED | "Missing required claims" |
| SpiffeError | UNAUTHENTICATED | "SPIFFE ID validation failed" |
| Platform::CircuitOpen | UNAVAILABLE | "Service temporarily unavailable" |
| Platform::RateLimited | RESOURCE_EXHAUSTED | "Rate limit exceeded" |
| Platform::Timeout | DEADLINE_EXCEEDED | "Request timed out" |
| Platform::* | INTERNAL | "Internal error" |

## Testing Strategy

### Dual Testing Approach

The service uses both unit tests and property-based tests for comprehensive coverage:

- **Unit Tests**: Verify specific examples, edge cases, and error conditions
- **Property Tests**: Verify universal properties across all inputs using proptest 1.5+

### Property-Based Testing Configuration

```rust
// proptest.toml
[default]
cases = 100
max_shrink_iters = 1000

[expensive]
cases = 1000
```

Each property test runs minimum 100 iterations with the following tag format:
```rust
// **Feature: auth-edge-modernization-2025, Property N: [Property Title]**
// **Validates: Requirements X.Y**
```

### Test Organization

```
tests/
├── unit/
│   ├── claims.rs          # Claims validation
│   ├── config.rs          # Configuration validation
│   ├── error.rs           # Error sanitization
│   ├── spiffe.rs          # SPIFFE ID parsing
│   └── token.rs           # JWT structure
├── property/
│   ├── generators.rs      # Proptest generators
│   ├── error_sanitization.rs  # Property 14
│   ├── circuit_breaker.rs     # Property 3
│   ├── rate_limiter.rs        # Rate limit enforcement
│   ├── spiffe.rs              # Property 11 (round-trip)
│   ├── jwt_typestate.rs       # Property 10
│   ├── config_validation.rs   # Property 12
│   └── correlation_id.rs      # Property 2
├── integration/
│   ├── validation_flow.rs # End-to-end validation
│   └── cache_fallback.rs  # Property 4, 5
└── contract/
    ├── token_service.rs   # Pact consumer tests
    ├── session_service.rs
    └── iam_service.rs
```

### Property Test Examples

```rust
// Property 11: SPIFFE ID Round-Trip
proptest! {
    #![proptest_config(ProptestConfig::with_cases(100))]
    
    // **Feature: auth-edge-modernization-2025, Property 11: SPIFFE ID Round-Trip**
    // **Validates: Requirements 10.7**
    #[test]
    fn spiffe_id_round_trip(
        trust_domain in "[a-z][a-z0-9-]*\\.[a-z]{2,}",
        path_segments in prop::collection::vec("[a-z][a-z0-9-]*", 0..5)
    ) {
        let uri = format!("spiffe://{}/{}", trust_domain, path_segments.join("/"));
        let parsed = SpiffeId::parse(&uri)?;
        let round_tripped = parsed.to_uri();
        let reparsed = SpiffeId::parse(&round_tripped)?;
        
        prop_assert_eq!(parsed.trust_domain, reparsed.trust_domain);
        prop_assert_eq!(parsed.path, reparsed.path);
    }
}

// Property 14: Sensitive Data Protection
proptest! {
    #![proptest_config(ProptestConfig::with_cases(100))]
    
    // **Feature: auth-edge-modernization-2025, Property 14: Sensitive Data Protection**
    // **Validates: Requirements 13.2, 13.3, 13.4**
    #[test]
    fn error_messages_never_contain_sensitive_data(
        error in arb_auth_edge_error(),
        correlation_id in arb_uuid()
    ) {
        let response = ErrorResponse::from_error(&error, correlation_id);
        let message = response.message.to_lowercase();
        
        for pattern in SENSITIVE_PATTERNS {
            prop_assert!(!message.contains(pattern),
                "Error message contains sensitive pattern: {}", pattern);
        }
    }
}
```

### Coverage Requirements

| Module | Target Coverage |
|--------|-----------------|
| error | 95% |
| jwt/claims | 92% |
| jwt/validator | 90% |
| jwt/jwk_cache | 88% |
| mtls/spiffe | 94% |
| config | 90% |
| grpc | 85% |
| **Overall** | **90%+** |

### Test Commands

```bash
# Run all tests
cargo test

# Run property tests only (100 iterations each)
cargo test --test property_tests

# Run with coverage
cargo tarpaulin --out Html --output-dir coverage

# Run contract tests
cargo test --features pact --test contract_tests

# Run specific property
cargo test spiffe_id_round_trip
```
