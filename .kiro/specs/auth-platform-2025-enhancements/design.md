# Design Document - Auth Platform 2025 Enhancements

## Overview

Este documento descreve o design técnico para as três melhorias arquiteturais de alta prioridade identificadas na análise arquitetural:

1. **HashiCorp Vault Integration** - Secrets management centralizado
2. **Linkerd Service Mesh** - mTLS automático entre serviços
3. **Pact gRPC Contract Testing** - Testes de contrato para APIs gRPC

A implementação segue uma abordagem em fases, com Vault sendo implantado primeiro (fornece secrets para outros componentes), seguido por Linkerd (usa Vault para certificados), e finalmente Pact (testa serviços através do mesh).

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         Auth Platform 2025 Architecture                      │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐          │
│  │   Pact Broker   │    │  cert-manager   │    │   Prometheus    │          │
│  │  (Contracts)    │    │  (Cert Rotation)│    │   (Metrics)     │          │
│  └────────┬────────┘    └────────┬────────┘    └────────┬────────┘          │
│           │                      │                      │                    │
│  ┌────────▼──────────────────────▼──────────────────────▼────────┐          │
│  │                    Linkerd Service Mesh                        │          │
│  │  ┌──────────────────────────────────────────────────────────┐ │          │
│  │  │                   Control Plane                           │ │          │
│  │  │  • Identity (mTLS certs)  • Destination (service discovery)│ │          │
│  │  │  • Proxy Injector         • Policy Controller              │ │          │
│  │  └──────────────────────────────────────────────────────────┘ │          │
│  │                                                                │          │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐            │          │
│  │  │ Auth Edge   │  │ Token       │  │ Session     │            │          │
│  │  │ + Proxy     │◄─►│ + Proxy     │◄─►│ + Proxy     │            │          │
│  │  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘            │          │
│  │         │                │                │                    │          │
│  │  ┌──────▼──────┐  ┌──────▼──────┐  ┌──────▼──────┐            │          │
│  │  │ IAM Policy  │  │ MFA Service │  │ Shared Libs │            │          │
│  │  │ + Proxy     │  │ + Proxy     │  │             │            │          │
│  │  └─────────────┘  └─────────────┘  └─────────────┘            │          │
│  └────────────────────────────────────────────────────────────────┘          │
│                                    │                                         │
│  ┌─────────────────────────────────▼─────────────────────────────┐          │
│  │                    HashiCorp Vault                             │          │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐         │          │
│  │  │ KV Secrets   │  │ Database     │  │ PKI Engine   │         │          │
│  │  │ Engine       │  │ Engine       │  │              │         │          │
│  │  │ • API Keys   │  │ • PostgreSQL │  │ • Root CA    │         │          │
│  │  │ • JWT Keys   │  │ • Redis      │  │ • Issuer CA  │         │          │
│  │  └──────────────┘  └──────────────┘  └──────────────┘         │          │
│  │                                                                │          │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐         │          │
│  │  │ K8s Auth     │  │ Transit      │  │ Audit Log    │         │          │
│  │  │ Method       │  │ Engine       │  │              │         │          │
│  │  └──────────────┘  └──────────────┘  └──────────────┘         │          │
│  └────────────────────────────────────────────────────────────────┘          │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Components and Interfaces

### 1. HashiCorp Vault

#### 1.1 Deployment Architecture

```yaml
# Vault HA deployment with Raft storage
vault:
  server:
    ha:
      enabled: true
      replicas: 3
      raft:
        enabled: true
    auditStorage:
      enabled: true
      size: 10Gi
```

#### 1.2 Secrets Engines

| Engine | Purpose | Configuration |
|--------|---------|---------------|
| KV v2 | Static secrets (API keys, JWT signing keys) | `secret/auth-platform/*` |
| Database | Dynamic PostgreSQL/Redis credentials | `database/auth-platform/*` |
| PKI | Certificate management for Linkerd | `pki/auth-platform/*` |
| Transit | Encryption-as-a-service | `transit/auth-platform/*` |

#### 1.3 Authentication Methods

```hcl
# Kubernetes auth method configuration
path "auth/kubernetes" {
  type = "kubernetes"
  config = {
    kubernetes_host = "https://kubernetes.default.svc"
    kubernetes_ca_cert = "@/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
  }
}

# Role for auth-edge-service
path "auth/kubernetes/role/auth-edge-service" {
  bound_service_account_names = ["auth-edge-service"]
  bound_service_account_namespaces = ["auth-platform"]
  policies = ["auth-edge-policy"]
  ttl = "1h"
}
```

#### 1.4 Dynamic Database Credentials

```hcl
# PostgreSQL dynamic credentials
path "database/config/auth-platform-postgres" {
  plugin_name = "postgresql-database-plugin"
  connection_url = "postgresql://{{username}}:{{password}}@postgres:5432/auth_platform"
  allowed_roles = ["auth-platform-*"]
  username = "vault-admin"
  password = "{{vault_password}}"
}

path "database/roles/auth-platform-readonly" {
  db_name = "auth-platform-postgres"
  creation_statements = [
    "CREATE ROLE \"{{name}}\" WITH LOGIN PASSWORD '{{password}}' VALID UNTIL '{{expiration}}';",
    "GRANT SELECT ON ALL TABLES IN SCHEMA public TO \"{{name}}\";"
  ]
  default_ttl = "1h"
  max_ttl = "24h"
}
```

### 2. Linkerd Service Mesh

#### 2.1 Installation with Helm

```yaml
# values.yaml for Linkerd
identity:
  issuer:
    scheme: kubernetes.io/tls
  externalCA: true  # Use cert-manager

proxy:
  resources:
    cpu:
      request: 100m
      limit: 500m
    memory:
      request: 20Mi
      limit: 250Mi

proxyInit:
  resources:
    cpu:
      request: 10m
      limit: 100m
    memory:
      request: 10Mi
      limit: 50Mi
```

#### 2.2 cert-manager Integration

```yaml
# Trust anchor certificate (10 years validity)
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: linkerd-trust-anchor
  namespace: linkerd
spec:
  isCA: true
  commonName: root.linkerd.cluster.local
  secretName: linkerd-trust-anchor
  duration: 87600h  # 10 years
  renewBefore: 8760h  # 1 year
  privateKey:
    algorithm: ECDSA
    size: 256
  issuerRef:
    name: linkerd-self-signed-issuer
    kind: ClusterIssuer
    group: cert-manager.io

---
# Identity issuer certificate (48 hours validity)
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: linkerd-identity-issuer
  namespace: linkerd
spec:
  isCA: true
  commonName: identity.linkerd.cluster.local
  secretName: linkerd-identity-issuer
  duration: 48h
  renewBefore: 25h
  privateKey:
    algorithm: ECDSA
    size: 256
  issuerRef:
    name: linkerd-trust-anchor
    kind: Issuer
```

#### 2.3 Service Annotations

```yaml
# Pod annotation for automatic injection
metadata:
  annotations:
    linkerd.io/inject: enabled
    config.linkerd.io/proxy-cpu-request: "100m"
    config.linkerd.io/proxy-memory-request: "20Mi"
```

### 3. Pact Contract Testing

#### 3.1 Pact Broker Deployment

```yaml
# Pact Broker with PostgreSQL backend
apiVersion: apps/v1
kind: Deployment
metadata:
  name: pact-broker
spec:
  template:
    spec:
      containers:
      - name: pact-broker
        image: pactfoundation/pact-broker:latest
        env:
        - name: PACT_BROKER_DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: pact-broker-secrets
              key: database-url
        - name: PACT_BROKER_WEBHOOK_HOST_WHITELIST
          value: "*.auth-platform.svc.cluster.local"
```

#### 3.2 Consumer Test Structure (Rust)

```rust
// auth-edge-service/tests/pact_consumer_test.rs
use pact_consumer::prelude::*;
use pact_consumer::mock_server::StartMockServerAsync;

#[tokio::test]
async fn test_token_service_contract() {
    let pact = PactBuilder::new("auth-edge-service", "token-service")
        .interaction("issue token pair", "", |mut i| async move {
            i.given("user exists");
            i.request
                .path("/auth.token.TokenService/IssueTokenPair")
                .method("POST")
                .content_type("application/grpc");
            i.response
                .status(200)
                .content_type("application/grpc");
            i
        })
        .await
        .build();

    let mock_server = pact.start_mock_server_async(None).await;
    // Test consumer against mock server
    // ...
    
    // Write pact to file
    pact.write_pact(None, false).unwrap();
}
```

#### 3.3 Provider Verification (Rust)

```rust
// token-service/tests/pact_provider_test.rs
use pact_verifier::*;

#[tokio::test]
async fn verify_token_service_contracts() {
    let provider = ProviderInfo {
        name: "token-service".to_string(),
        host: "localhost".to_string(),
        port: Some(50052),
        ..Default::default()
    };

    let pact_source = PactSource::BrokerWithDynamicConfiguration {
        provider_name: "token-service".to_string(),
        broker_url: "http://pact-broker:9292".to_string(),
        enable_pending: true,
        include_wip_pacts_since: Some("2025-01-01".to_string()),
        ..Default::default()
    };

    let verification_options = VerificationOptions {
        publish: true,
        provider_version: Some(env!("CARGO_PKG_VERSION").to_string()),
        ..Default::default()
    };

    verify_provider_async(provider, vec![pact_source], verification_options)
        .await
        .unwrap();
}
```

#### 3.4 CI/CD Integration

```yaml
# .github/workflows/contract-tests.yml
name: Contract Tests

on: [push, pull_request]

jobs:
  consumer-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Run consumer tests
        run: cargo test --test pact_consumer_test
      - name: Publish contracts
        run: |
          pact-broker publish ./target/pacts \
            --broker-base-url ${{ secrets.PACT_BROKER_URL }} \
            --consumer-app-version ${{ github.sha }} \
            --tag ${{ github.ref_name }}

  can-i-deploy:
    needs: consumer-tests
    runs-on: ubuntu-latest
    steps:
      - name: Can I Deploy?
        run: |
          pact-broker can-i-deploy \
            --pacticipant auth-edge-service \
            --version ${{ github.sha }} \
            --to-environment production \
            --broker-base-url ${{ secrets.PACT_BROKER_URL }}
```

## Data Models

### Vault Secret Structure

```json
{
  "secret/auth-platform/jwt-signing-key": {
    "private_key": "-----BEGIN EC PRIVATE KEY-----...",
    "public_key": "-----BEGIN PUBLIC KEY-----...",
    "algorithm": "ES256",
    "key_id": "key-2025-01",
    "created_at": "2025-01-15T00:00:00Z"
  }
}
```

### Pact Contract Structure

```json
{
  "consumer": { "name": "auth-edge-service" },
  "provider": { "name": "token-service" },
  "interactions": [
    {
      "description": "issue token pair",
      "providerState": "user exists",
      "request": {
        "method": "POST",
        "path": "/auth.token.TokenService/IssueTokenPair",
        "headers": { "Content-Type": "application/grpc" }
      },
      "response": {
        "status": 200,
        "headers": { "Content-Type": "application/grpc" }
      }
    }
  ],
  "metadata": {
    "pactSpecification": { "version": "4.0" },
    "pact-protobuf-plugin": { "version": "0.4.0" }
  }
}
```

## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system-essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*

### Property 1: Vault Secrets Lifecycle

*For any* service requesting secrets from Vault, the returned credentials SHALL be unique, have TTL within configured limits, and be logged in the audit trail with accessor identity.

**Validates: Requirements 1.1, 1.2, 1.3, 1.4**

### Property 2: Dynamic Credentials Uniqueness

*For any* two requests for database credentials, the returned username/password pairs SHALL be different, ensuring no credential reuse.

**Validates: Requirements 1.2**

### Property 3: Secret Renewal Before Expiration

*For any* secret with TTL, the Vault Agent SHALL initiate renewal when remaining TTL is less than 20% of original TTL, ensuring continuous availability.

**Validates: Requirements 1.3**

### Property 4: PKI Certificate Validity

*For any* certificate issued by Vault PKI engine, the validity period SHALL not exceed the configured maximum (72 hours for service certs, 48 hours for Linkerd identity issuer).

**Validates: Requirements 2.1, 2.3**

### Property 5: Linkerd mTLS Establishment

*For any* two meshed pods communicating, the connection SHALL use mTLS with valid workload certificates, verifiable through Linkerd tap or metrics.

**Validates: Requirements 3.1, 3.2, 3.3**

### Property 6: Trace Context Propagation

*For any* request traversing multiple services through Linkerd, the W3C Trace Context headers (traceparent, tracestate) SHALL be present in all hops.

**Validates: Requirements 4.2**

### Property 7: Contract Verification Pipeline

*For any* provider service modification, the CI pipeline SHALL verify against all consumer contracts before allowing deployment, blocking on verification failure.

**Validates: Requirements 5.2, 6.2, 6.3**

### Property 8: Contract Storage and Versioning

*For any* generated contract, the Pact Broker SHALL store it with version tag matching the git commit SHA, enabling traceability.

**Validates: Requirements 5.4, 6.4, 6.5**

### Property 9: Secret Rotation Continuity

*For any* secret rotation event, the dependent services SHALL continue operating without errors or restarts, verified by zero error rate increase during rotation window.

**Validates: Requirements 7.4**

### Property 10: Generic Secret Provider Type Safety

*For any* type T implementing Deserialize, the SecretProvider<T> trait SHALL return correctly typed secrets without runtime type errors.

**Validates: Requirements 13.1, 13.3**

### Property 11: Resilient Client Retry Behavior

*For any* transient failure, the ResilientClient SHALL retry according to configured policy and succeed if service recovers within retry window.

**Validates: Requirements 13.2**

### Property 12: Vault Latency SLO Compliance

*For any* secret request under normal load, the Vault response time SHALL be within SLO bounds (p99 ≤50ms).

**Validates: Requirements 14.1, 14.4, 14.5**

### Property 13: Linkerd Latency Overhead

*For any* request through Linkerd proxy, the added latency SHALL not exceed 2ms at p99.

**Validates: Requirements 14.2**

### Property 14: Secure Memory Zeroization

*For any* SensitiveString instance, the underlying memory SHALL be zeroized when the instance is dropped.

**Validates: Requirements 15.2**

### Property 15: Constant-Time Comparison

*For any* two byte arrays of equal length, the secure_compare function SHALL take the same time regardless of content (no timing side-channel).

**Validates: Requirements 15.1**

## Error Handling

### Vault Errors

| Error | Handling | Recovery |
|-------|----------|----------|
| Vault unavailable | Use cached credentials | Retry with exponential backoff (max 5 min) |
| Authentication failed | Log error, alert | Verify service account, check policies |
| Secret not found | Return error to caller | Check secret path, verify permissions |
| Lease renewal failed | Request new credentials | Automatic via Vault Agent |

### Linkerd Errors

| Error | Handling | Recovery |
|-------|----------|----------|
| Proxy injection failed | Pod starts without mesh | Check annotation, verify injector |
| mTLS handshake failed | Connection refused | Check certificates, verify identity |
| Control plane unavailable | Data plane continues | Automatic reconnection |

### Pact Errors

| Error | Handling | Recovery |
|-------|----------|----------|
| Contract verification failed | Block deployment | Fix provider or update contract |
| Pact Broker unavailable | Fail CI job | Retry, check broker health |
| can-i-deploy failed | Block deployment | Verify all contracts first |

## Code Architecture - Generics and Reusability

### Generic Secret Provider (Rust)

```rust
// auth/shared/src/secrets/provider.rs
use async_trait::async_trait;
use serde::{Deserialize, Serialize};

/// Generic trait for secret providers with type-safe retrieval
#[async_trait]
pub trait SecretProvider<T>: Send + Sync
where
    T: for<'de> Deserialize<'de> + Send,
{
    type Error: std::error::Error + Send + Sync;
    
    async fn get_secret(&self, path: &str) -> Result<T, Self::Error>;
    async fn get_secret_with_version(&self, path: &str, version: u32) -> Result<T, Self::Error>;
}

/// Generic Vault client implementation
pub struct VaultClient<C: HttpClient> {
    client: C,
    config: VaultConfig,
}

impl<C: HttpClient> SecretProvider<T> for VaultClient<C>
where
    T: for<'de> Deserialize<'de> + Send,
{
    type Error = VaultError;
    
    async fn get_secret(&self, path: &str) -> Result<T, Self::Error> {
        let response = self.client
            .get(&format!("{}/v1/{}", self.config.addr, path))
            .with_retry(self.config.retry_policy.clone())
            .with_circuit_breaker(self.config.circuit_breaker.clone())
            .send()
            .await?;
        
        response.json::<VaultResponse<T>>()?.data
    }
}
```

### Generic HTTP Client with Resilience (Rust)

```rust
// auth/shared/src/http/client.rs
use std::time::Duration;

/// Generic HTTP client trait with resilience patterns
#[async_trait]
pub trait HttpClient: Send + Sync + Clone {
    async fn get<T: DeserializeOwned>(&self, url: &str) -> Result<T, HttpError>;
    async fn post<T: Serialize, R: DeserializeOwned>(&self, url: &str, body: &T) -> Result<R, HttpError>;
}

/// Resilient client wrapper with configurable policies
pub struct ResilientClient<C: HttpClient> {
    inner: C,
    retry_policy: RetryPolicy,
    circuit_breaker: CircuitBreaker,
    timeout: Duration,
}

impl<C: HttpClient> ResilientClient<C> {
    pub fn new(client: C) -> Self {
        Self {
            inner: client,
            retry_policy: RetryPolicy::exponential_backoff(3, Duration::from_millis(100)),
            circuit_breaker: CircuitBreaker::new(5, Duration::from_secs(30)),
            timeout: Duration::from_secs(30),
        }
    }
    
    pub fn with_retry(mut self, policy: RetryPolicy) -> Self {
        self.retry_policy = policy;
        self
    }
    
    pub fn with_circuit_breaker(mut self, cb: CircuitBreaker) -> Self {
        self.circuit_breaker = cb;
        self
    }
}
```

### Generic Error Handling (Rust)

```rust
// auth/shared/src/error/mod.rs
use thiserror::Error;

/// Centralized error type with generic conversion
#[derive(Error, Debug)]
pub enum AuthError<E: std::error::Error> {
    #[error("Vault error: {0}")]
    Vault(#[from] VaultError),
    
    #[error("Network error: {0}")]
    Network(#[from] NetworkError),
    
    #[error("Serialization error: {0}")]
    Serialization(#[from] SerializationError),
    
    #[error("Service-specific error: {0}")]
    Service(E),
}

/// Generic Result type alias
pub type AuthResult<T, E = Box<dyn std::error::Error>> = Result<T, AuthError<E>>;

/// Trait for converting service errors to AuthError
pub trait IntoAuthError<E: std::error::Error> {
    fn into_auth_error(self) -> AuthError<E>;
}
```

### Generic Configuration Loader (Rust)

```rust
// auth/shared/src/config/loader.rs
use serde::de::DeserializeOwned;

/// Generic configuration loader supporting multiple sources
pub struct ConfigLoader<T: DeserializeOwned> {
    _phantom: std::marker::PhantomData<T>,
}

impl<T: DeserializeOwned> ConfigLoader<T> {
    /// Load configuration from multiple sources with priority
    pub async fn load() -> Result<T, ConfigError> {
        // Priority: Vault > Environment > File > Defaults
        let mut builder = config::Config::builder();
        
        // 1. Load defaults
        builder = builder.add_source(config::File::with_name("config/default"));
        
        // 2. Load environment-specific file
        if let Ok(env) = std::env::var("APP_ENV") {
            builder = builder.add_source(config::File::with_name(&format!("config/{}", env)));
        }
        
        // 3. Load environment variables
        builder = builder.add_source(config::Environment::with_prefix("AUTH"));
        
        // 4. Load from Vault if configured
        if let Ok(vault_addr) = std::env::var("VAULT_ADDR") {
            let vault_config = VaultConfigSource::new(&vault_addr).await?;
            builder = builder.add_source(vault_config);
        }
        
        builder.build()?.try_deserialize()
    }
}
```

### Secure Memory Handling (Rust)

```rust
// auth/shared/src/security/memory.rs
use secrecy::{ExposeSecret, Secret};
use zeroize::Zeroize;

/// Wrapper for sensitive data that zeroizes on drop
#[derive(Clone)]
pub struct SensitiveString(Secret<String>);

impl SensitiveString {
    pub fn new(value: String) -> Self {
        Self(Secret::new(value))
    }
    
    /// Expose the secret for use - caller must handle securely
    pub fn expose(&self) -> &str {
        self.0.expose_secret()
    }
}

/// Constant-time comparison for secrets
pub fn secure_compare(a: &[u8], b: &[u8]) -> bool {
    use subtle::ConstantTimeEq;
    a.ct_eq(b).into()
}
```

## Testing Strategy

### Dual Testing Approach

This implementation uses both unit tests and property-based tests:

- **Unit tests**: Verify specific examples, edge cases, and error conditions
- **Property-based tests**: Verify universal properties across all valid inputs

### Property-Based Testing Framework

- **Rust services**: `proptest` with 100+ iterations per property
- **Elixir services**: `StreamData` with 100+ iterations per property
- **Go services**: `gopter` with 100+ iterations per property

### Test Categories

1. **Vault Integration Tests**
   - Secret retrieval round-trip
   - Dynamic credential generation
   - Certificate issuance and renewal
   - Audit log verification

2. **Linkerd Integration Tests**
   - mTLS establishment verification
   - Certificate rotation without downtime
   - Metrics exposure validation
   - Trace context propagation

3. **Pact Contract Tests**
   - Consumer contract generation
   - Provider verification
   - can-i-deploy checks
   - Webhook triggering

4. **Performance Tests**
   - Vault latency benchmarks (p50 ≤10ms, p95 ≤25ms, p99 ≤50ms)
   - Linkerd overhead benchmarks (p50 ≤1ms, p95 ≤1.5ms, p99 ≤2ms)
   - Throughput tests (≥50,000 concurrent connections)

### Test Annotations

Each property-based test MUST be annotated with:
```
**Feature: auth-platform-2025-enhancements, Property {number}: {property_text}**
**Validates: Requirements {X.Y}**
```
