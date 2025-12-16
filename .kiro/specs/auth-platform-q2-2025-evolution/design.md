# Design Document - Auth Platform Q2 2025 Evolution

## Overview

Este documento descreve o design técnico para a evolução Q2 2025 da Auth Platform, focando em três áreas principais:

1. **Passkeys (WebAuthn Discoverable Credentials)** - Autenticação passwordless
2. **CAEP (Continuous Access Evaluation Protocol)** - Revogação em tempo real
3. **SDK Client Libraries** - TypeScript, Python e Go

A implementação segue uma abordagem em fases, com Passkeys sendo implementado primeiro (base para autenticação), seguido por CAEP (eventos de segurança), e finalmente SDKs (consumo das APIs).

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    Auth Platform Q2 2025 Architecture                        │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │                         SDK Layer                                    │    │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐                  │    │
│  │  │ TypeScript  │  │ Python SDK  │  │ Go SDK      │                  │    │
│  │  │ SDK         │  │ (sync/async)│  │ (http/gRPC) │                  │    │
│  │  │ • OAuth 2.1 │  │ • FastAPI   │  │ • Middleware│                  │    │
│  │  │ • Passkeys  │  │ • Flask     │  │ • Intercept │                  │    │
│  │  │ • CAEP Sub  │  │ • Django    │  │ • Options   │                  │    │
│  │  └─────────────┘  └─────────────┘  └─────────────┘                  │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                    │                                         │
│  ┌─────────────────────────────────▼─────────────────────────────────┐      │
│  │                    CAEP Layer (SSF/SET)                            │      │
│  │  ┌──────────────────────┐  ┌──────────────────────┐               │      │
│  │  │ CAEP Transmitter     │  │ CAEP Receiver        │               │      │
│  │  │ • session-revoked    │  │ • SET Validation     │               │      │
│  │  │ • credential-change  │  │ • Event Processing   │               │      │
│  │  │ • assurance-change   │  │ • Stream Management  │               │      │
│  │  │ • SET Signing (ES256)│  │ • Retry/Recovery     │               │      │
│  │  └──────────────────────┘  └──────────────────────┘               │      │
│  └────────────────────────────────────────────────────────────────────┘      │
│                                    │                                         │
│  ┌─────────────────────────────────▼─────────────────────────────────┐      │
│  │                    Passkeys Layer (WebAuthn)                       │      │
│  │  ┌──────────────────────┐  ┌──────────────────────┐               │      │
│  │  │ Registration         │  │ Authentication       │               │      │
│  │  │ • Discoverable Creds │  │ • Conditional UI     │               │      │
│  │  │ • Platform/Roaming   │  │ • Cross-Device       │               │      │
│  │  │ • Attestation        │  │ • Hybrid Transport   │               │      │
│  │  └──────────────────────┘  └──────────────────────┘               │      │
│  │  ┌──────────────────────┐  ┌──────────────────────┐               │      │
│  │  │ Credential Store     │  │ Management           │               │      │
│  │  │ • Encrypted Storage  │  │ • List/Rename/Delete │               │      │
│  │  │ • Sync Handling      │  │ • Re-auth Required   │               │      │
│  │  └──────────────────────┘  └──────────────────────┘               │      │
│  └────────────────────────────────────────────────────────────────────┘      │
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │                    Existing Services                                 │    │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐│    │
│  │  │ Auth Edge   │  │ Token Svc   │  │ Session     │  │ MFA Service ││    │
│  │  │ Service     │  │             │  │ Identity    │  │ (Extended)  ││    │
│  │  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘│    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Components and Interfaces

### 1. Passkeys (WebAuthn Discoverable Credentials)

#### 1.1 Registration Flow

```
┌──────────┐     ┌──────────────┐     ┌─────────────┐     ┌─────────────────┐
│  Client  │     │ Auth Edge    │     │ MFA Service │     │ Authenticator   │
└────┬─────┘     └──────┬───────┘     └──────┬──────┘     └────────┬────────┘
     │                  │                    │                     │
     │ POST /passkeys/register/begin        │                     │
     │─────────────────►│                    │                     │
     │                  │ CreateOptions()    │                     │
     │                  │───────────────────►│                     │
     │                  │                    │                     │
     │                  │ PublicKeyCredentialCreationOptions       │
     │◄─────────────────│◄───────────────────│                     │
     │                  │                    │                     │
     │ navigator.credentials.create()        │                     │
     │────────────────────────────────────────────────────────────►│
     │                  │                    │                     │
     │ PublicKeyCredential (attestation)     │                     │
     │◄────────────────────────────────────────────────────────────│
     │                  │                    │                     │
     │ POST /passkeys/register/finish        │                     │
     │─────────────────►│                    │                     │
     │                  │ VerifyAttestation()│                     │
     │                  │───────────────────►│                     │
     │                  │                    │ Store Credential    │
     │                  │                    │────────────────────►│
     │                  │ Success            │                     │
     │◄─────────────────│◄───────────────────│                     │
     │                  │                    │                     │
```

#### 1.2 WebAuthn Options Configuration

```elixir
# mfa-service/lib/mfa_service/passkeys/registration.ex
defmodule MfaService.Passkeys.Registration do
  @moduledoc """
  WebAuthn registration with discoverable credentials support.
  """

  @spec create_options(user_id :: String.t(), user_name :: String.t()) :: map()
  def create_options(user_id, user_name) do
    %{
      challenge: generate_challenge(),
      rp: %{
        name: "Auth Platform",
        id: Application.get_env(:mfa_service, :rp_id)
      },
      user: %{
        id: Base.url_encode64(user_id, padding: false),
        name: user_name,
        displayName: user_name
      },
      pubKeyCredParams: [
        %{type: "public-key", alg: -7},   # ES256
        %{type: "public-key", alg: -257}  # RS256
      ],
      authenticatorSelection: %{
        residentKey: "required",           # Discoverable credentials
        userVerification: "required",      # Biometric/PIN required
        authenticatorAttachment: "platform" # or "cross-platform"
      },
      attestation: "direct",
      timeout: 60_000
    }
  end
end
```

#### 1.3 Credential Storage Schema

```sql
-- PostgreSQL schema for passkey credentials
CREATE TABLE passkey_credentials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    credential_id BYTEA NOT NULL UNIQUE,
    public_key BYTEA NOT NULL,
    public_key_alg INTEGER NOT NULL,
    sign_count BIGINT NOT NULL DEFAULT 0,
    transports TEXT[] DEFAULT '{}',
    attestation_format TEXT,
    attestation_statement JSONB,
    aaguid UUID,
    device_name TEXT,
    is_discoverable BOOLEAN NOT NULL DEFAULT true,
    backed_up BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_used_at TIMESTAMPTZ,
    CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_passkey_user_id ON passkey_credentials(user_id);
CREATE INDEX idx_passkey_credential_id ON passkey_credentials(credential_id);
```

### 2. CAEP (Continuous Access Evaluation Protocol)

#### 2.1 Event Types (OpenID CAEP 1.0)

| Event Type | Description | Trigger |
|------------|-------------|---------|
| `session-revoked` | Session terminated | Logout, admin action, security event |
| `credential-change` | Credential modified | Password change, passkey added/removed |
| `assurance-level-change` | Risk level changed | Step-up auth, risk detection |
| `token-claims-change` | Token claims updated | Role change, permission update |
| `device-compliance-change` | Device status changed | MDM policy violation |

#### 2.2 SET (Security Event Token) Structure

```json
{
  "iss": "https://auth.example.com",
  "iat": 1734307200,
  "jti": "756E69717565-6964",
  "aud": "https://receiver.example.com",
  "events": {
    "https://schemas.openid.net/secevent/caep/event-type/session-revoked": {
      "subject": {
        "format": "iss_sub",
        "iss": "https://auth.example.com",
        "sub": "user-123"
      },
      "event_timestamp": 1734307200,
      "reason_admin": {
        "en": "Session revoked due to security policy"
      }
    }
  }
}
```

#### 2.3 CAEP Transmitter Architecture

```rust
// auth/shared/caep/src/transmitter.rs
use async_trait::async_trait;
use serde::{Deserialize, Serialize};

/// CAEP event types conforming to OpenID CAEP 1.0
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "kebab-case")]
pub enum CaepEventType {
    SessionRevoked,
    CredentialChange,
    AssuranceLevelChange,
    TokenClaimsChange,
    DeviceComplianceChange,
}

/// Subject identifier formats
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(tag = "format")]
pub enum SubjectIdentifier {
    #[serde(rename = "iss_sub")]
    IssSub { iss: String, sub: String },
    #[serde(rename = "email")]
    Email { email: String },
    #[serde(rename = "opaque")]
    Opaque { id: String },
}

/// CAEP Transmitter trait
#[async_trait]
pub trait CaepTransmitter: Send + Sync {
    /// Emit a security event to all registered streams
    async fn emit(&self, event: CaepEvent) -> Result<(), CaepError>;
    
    /// Register a new stream receiver
    async fn register_stream(&self, config: StreamConfig) -> Result<StreamId, CaepError>;
    
    /// Get stream status
    async fn stream_status(&self, stream_id: &StreamId) -> Result<StreamStatus, CaepError>;
}
```

#### 2.4 CAEP Receiver Architecture

```rust
// auth/shared/caep/src/receiver.rs

/// CAEP Receiver trait
#[async_trait]
pub trait CaepReceiver: Send + Sync {
    /// Process incoming SET
    async fn process_event(&self, set: &str) -> Result<(), CaepError>;
    
    /// Validate SET signature
    async fn validate_signature(&self, set: &str, jwks_uri: &str) -> Result<bool, CaepError>;
}

/// Event handler registry
pub struct EventHandlerRegistry {
    handlers: HashMap<CaepEventType, Box<dyn EventHandler>>,
}

impl EventHandlerRegistry {
    pub fn register<H: EventHandler + 'static>(&mut self, event_type: CaepEventType, handler: H) {
        self.handlers.insert(event_type, Box::new(handler));
    }
    
    pub async fn dispatch(&self, event: &CaepEvent) -> Result<(), CaepError> {
        if let Some(handler) = self.handlers.get(&event.event_type) {
            handler.handle(event).await
        } else {
            Err(CaepError::UnknownEventType)
        }
    }
}
```

### 3. SDK Client Libraries

#### 3.1 TypeScript SDK Structure

```typescript
// @auth-platform/sdk/src/index.ts
export interface AuthPlatformConfig {
  baseUrl: string;
  clientId: string;
  clientSecret?: string;
  scopes?: string[];
  storage?: TokenStorage;
}

export class AuthPlatformClient {
  private config: AuthPlatformConfig;
  private tokenManager: TokenManager;
  private passkeys: PasskeysClient;
  private caep: CaepSubscriber;

  constructor(config: AuthPlatformConfig) {
    this.config = config;
    this.tokenManager = new TokenManager(config);
    this.passkeys = new PasskeysClient(config);
    this.caep = new CaepSubscriber(config);
  }

  // OAuth 2.1 with PKCE
  async authorize(options?: AuthorizeOptions): Promise<AuthorizationResult> {
    const pkce = await generatePKCE();
    // ... implementation
  }

  // Passkey registration
  async registerPasskey(options?: PasskeyOptions): Promise<PasskeyCredential> {
    return this.passkeys.register(options);
  }

  // Passkey authentication
  async authenticateWithPasskey(): Promise<AuthenticationResult> {
    return this.passkeys.authenticate();
  }

  // CAEP event subscription
  onSecurityEvent(handler: (event: CaepEvent) => void): Unsubscribe {
    return this.caep.subscribe(handler);
  }
}
```

#### 3.2 Python SDK Structure

```python
# auth_platform_sdk/client.py
from typing import Optional, Union, AsyncIterator
from dataclasses import dataclass
import httpx

@dataclass
class AuthPlatformConfig:
    base_url: str
    client_id: str
    client_secret: Optional[str] = None
    scopes: list[str] = None
    timeout: float = 30.0

class AuthPlatformClient:
    """Synchronous Auth Platform client."""
    
    def __init__(self, config: AuthPlatformConfig):
        self.config = config
        self._http = httpx.Client(
            base_url=config.base_url,
            timeout=config.timeout
        )
        self._token_cache = TokenCache()
    
    def validate_token(self, token: str) -> TokenClaims:
        """Validate JWT and return claims."""
        jwks = self._get_jwks()
        return jwt.decode(token, jwks, algorithms=["ES256", "RS256"])
    
    def client_credentials(self) -> TokenResponse:
        """Obtain token using client credentials flow."""
        return self._http.post("/oauth/token", data={
            "grant_type": "client_credentials",
            "client_id": self.config.client_id,
            "client_secret": self.config.client_secret,
        }).json()

class AsyncAuthPlatformClient:
    """Asynchronous Auth Platform client."""
    
    def __init__(self, config: AuthPlatformConfig):
        self.config = config
        self._http = httpx.AsyncClient(
            base_url=config.base_url,
            timeout=config.timeout
        )
    
    async def validate_token(self, token: str) -> TokenClaims:
        """Validate JWT and return claims."""
        jwks = await self._get_jwks()
        return jwt.decode(token, jwks, algorithms=["ES256", "RS256"])
```

#### 3.3 Go SDK Structure

```go
// auth-platform-sdk-go/client.go
package authplatform

import (
    "context"
    "net/http"
    "time"
)

// Config holds client configuration
type Config struct {
    BaseURL      string
    ClientID     string
    ClientSecret string
    Timeout      time.Duration
}

// Option is a functional option for configuring the client
type Option func(*Client)

// WithTimeout sets the HTTP timeout
func WithTimeout(d time.Duration) Option {
    return func(c *Client) {
        c.httpClient.Timeout = d
    }
}

// WithHTTPClient sets a custom HTTP client
func WithHTTPClient(hc *http.Client) Option {
    return func(c *Client) {
        c.httpClient = hc
    }
}

// Client is the Auth Platform SDK client
type Client struct {
    config     Config
    httpClient *http.Client
    jwksCache  *JWKSCache
}

// New creates a new Auth Platform client
func New(config Config, opts ...Option) *Client {
    c := &Client{
        config:     config,
        httpClient: &http.Client{Timeout: 30 * time.Second},
        jwksCache:  NewJWKSCache(time.Hour),
    }
    for _, opt := range opts {
        opt(c)
    }
    return c
}

// ValidateToken validates a JWT and returns claims
func (c *Client) ValidateToken(ctx context.Context, token string) (*Claims, error) {
    jwks, err := c.jwksCache.Get(ctx, c.config.BaseURL+"/.well-known/jwks.json")
    if err != nil {
        return nil, err
    }
    return validateJWT(token, jwks)
}

// Middleware returns HTTP middleware for token validation
func (c *Client) Middleware() func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            token := extractBearerToken(r)
            claims, err := c.ValidateToken(r.Context(), token)
            if err != nil {
                http.Error(w, "Unauthorized", http.StatusUnauthorized)
                return
            }
            ctx := context.WithValue(r.Context(), claimsKey, claims)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}
```

## Data Models

### Passkey Credential

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "user_id": "user-123",
  "credential_id": "base64url-encoded-credential-id",
  "public_key": "base64url-encoded-public-key",
  "public_key_alg": -7,
  "sign_count": 42,
  "transports": ["internal", "hybrid"],
  "attestation_format": "packed",
  "aaguid": "00000000-0000-0000-0000-000000000000",
  "device_name": "MacBook Pro Touch ID",
  "is_discoverable": true,
  "backed_up": true,
  "created_at": "2025-01-15T00:00:00Z",
  "last_used_at": "2025-01-15T12:00:00Z"
}
```

### CAEP Stream Configuration

```json
{
  "stream_id": "stream-123",
  "issuer": "https://auth.example.com",
  "audience": "https://receiver.example.com",
  "delivery": {
    "method": "push",
    "endpoint_url": "https://receiver.example.com/caep/events"
  },
  "events_requested": [
    "session-revoked",
    "credential-change",
    "assurance-level-change"
  ],
  "format": "iss_sub",
  "status": "active",
  "created_at": "2025-01-15T00:00:00Z"
}
```

## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system-essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*

### Property 1: Passkey Registration Options Correctness

*For any* user initiating passkey registration, the generated WebAuthn options SHALL have residentKey="required" and userVerification="required", ensuring discoverable credentials with user verification.

**Validates: Requirements 1.1, 2.1**

### Property 2: Passkey Credential Storage Integrity

*For any* successfully registered passkey, the stored credential SHALL have is_discoverable=true and contain valid public key, credential ID, and sign count.

**Validates: Requirements 1.2, 1.3**

### Property 3: Passkey Authentication Session Metadata

*For any* successful passkey authentication, the created session SHALL contain passkey attestation metadata including authenticator type and backup status.

**Validates: Requirements 2.4, 2.5**

### Property 4: Cross-Device Authentication Fallback

*For any* failed cross-device authentication attempt, the system SHALL return available fallback authentication methods for the user.

**Validates: Requirements 3.5**

### Property 5: Passkey Management Re-authentication

*For any* passkey deletion request, the system SHALL reject the request if re-authentication was not performed within the last 5 minutes.

**Validates: Requirements 4.2, 4.3**

### Property 6: CAEP SET Signature Validity

*For any* emitted Security Event Token, the SET SHALL be signed with ES256 and verifiable using the platform's published JWKS.

**Validates: Requirements 5.5, 6.1**

### Property 7: CAEP Event Emission Completeness

*For any* security event (session revocation, credential change, assurance change), the CAEP Transmitter SHALL emit a corresponding SET with correct event type and subject identifier.

**Validates: Requirements 5.2, 5.3, 5.4**

### Property 8: CAEP Session Revocation Effect

*For any* received session-revoked event with valid signature, the affected session SHALL be terminated within 1 second of event processing.

**Validates: Requirements 6.2**

### Property 9: CAEP Stream Health Tracking

*For any* configured CAEP stream, the system SHALL track and expose delivery success rate, latency percentiles, and last successful delivery timestamp.

**Validates: Requirements 7.3, 7.5**

### Property 10: SDK PKCE Enforcement

*For any* OAuth authorization flow initiated through any SDK, the flow SHALL use PKCE with S256 challenge method.

**Validates: Requirements 8.2, 9.4, 10.1**

### Property 11: SDK Token Refresh Automation

*For any* expired access token in SDK token storage, the SDK SHALL automatically attempt refresh using the stored refresh token before returning an error.

**Validates: Requirements 8.4**

### Property 12: SDK JWKS Caching

*For any* token validation request, the SDK SHALL use cached JWKS if available and not expired, reducing network calls.

**Validates: Requirements 9.2**

### Property 13: SDK Error Type Safety

*For any* error returned by SDK methods, the error SHALL be an instance of a typed error class with error code and actionable message.

**Validates: Requirements 8.5, 10.4**

### Property 14: Passkey Latency SLO

*For any* passkey registration or authentication operation under normal load, the operation SHALL complete within 200ms (registration) or 100ms (authentication) at p99 latency.

**Validates: Requirements 12.1, 12.2**

### Property 15: CAEP Event Delivery Latency

*For any* security event emission, the SET SHALL be delivered to all active streams within 100ms at p99 latency.

**Validates: Requirements 12.3**

## Code Architecture - Generics, Patterns and Best Practices

### Generic Passkey Provider (Elixir - Behaviour)

```elixir
# mfa-service/lib/mfa_service/passkeys/provider.ex
defmodule MfaService.Passkeys.Provider do
  @moduledoc """
  Generic behaviour for passkey credential providers.
  Allows pluggable storage backends (PostgreSQL, Redis, etc.)
  """

  @type credential_id :: binary()
  @type user_id :: String.t()
  @type credential :: map()
  @type error :: {:error, atom() | String.t()}

  @callback store_credential(user_id(), credential()) :: {:ok, credential()} | error()
  @callback get_credential(credential_id()) :: {:ok, credential()} | {:error, :not_found}
  @callback get_credentials_for_user(user_id()) :: {:ok, [credential()]} | error()
  @callback update_credential(credential_id(), map()) :: {:ok, credential()} | error()
  @callback delete_credential(credential_id()) :: :ok | error()
  @callback increment_sign_count(credential_id(), non_neg_integer()) :: {:ok, credential()} | error()
end

defmodule MfaService.Passkeys.PostgresProvider do
  @behaviour MfaService.Passkeys.Provider
  
  alias MfaService.Repo
  alias MfaService.Passkeys.Credential
  
  @impl true
  def store_credential(user_id, credential) do
    %Credential{}
    |> Credential.changeset(Map.put(credential, :user_id, user_id))
    |> Repo.insert()
  end
  
  @impl true
  def get_credential(credential_id) do
    case Repo.get_by(Credential, credential_id: credential_id) do
      nil -> {:error, :not_found}
      cred -> {:ok, cred}
    end
  end
  
  # ... other implementations
end
```

### Generic CAEP Event Handler (Rust - Trait with Associated Types)

```rust
// auth/shared/caep/src/handler.rs
use async_trait::async_trait;
use std::marker::PhantomData;

/// Generic event handler trait with associated types
#[async_trait]
pub trait EventHandler<E>: Send + Sync
where
    E: CaepEvent + Send + Sync,
{
    type Output;
    type Error: std::error::Error + Send + Sync;
    
    /// Handle the event and return result
    async fn handle(&self, event: E) -> Result<Self::Output, Self::Error>;
    
    /// Check if this handler can process the event
    fn can_handle(&self, event: &E) -> bool;
}

/// Generic event processor with pluggable handlers
pub struct EventProcessor<E, H>
where
    E: CaepEvent + Send + Sync,
    H: EventHandler<E>,
{
    handlers: Vec<H>,
    _phantom: PhantomData<E>,
}

impl<E, H> EventProcessor<E, H>
where
    E: CaepEvent + Send + Sync + Clone,
    H: EventHandler<E>,
{
    pub fn new() -> Self {
        Self {
            handlers: Vec::new(),
            _phantom: PhantomData,
        }
    }
    
    pub fn register(mut self, handler: H) -> Self {
        self.handlers.push(handler);
        self
    }
    
    pub async fn process(&self, event: E) -> Result<Vec<H::Output>, H::Error> {
        let mut results = Vec::new();
        for handler in &self.handlers {
            if handler.can_handle(&event) {
                results.push(handler.handle(event.clone()).await?);
            }
        }
        Ok(results)
    }
}
```

### Generic SDK Client (TypeScript - Generics with Constraints)

```typescript
// @auth-platform/sdk/src/client.ts

/**
 * Generic HTTP client with type-safe request/response
 */
interface HttpClient<TConfig extends BaseConfig> {
  get<TResponse>(path: string, options?: RequestOptions): Promise<TResponse>;
  post<TRequest, TResponse>(path: string, body: TRequest, options?: RequestOptions): Promise<TResponse>;
  put<TRequest, TResponse>(path: string, body: TRequest, options?: RequestOptions): Promise<TResponse>;
  delete<TResponse>(path: string, options?: RequestOptions): Promise<TResponse>;
}

/**
 * Generic token storage interface
 */
interface TokenStorage<T extends TokenData = TokenData> {
  get(): Promise<T | null>;
  set(tokens: T): Promise<void>;
  clear(): Promise<void>;
}

/**
 * Generic result type for operations that can fail
 */
type Result<T, E extends Error = Error> = 
  | { success: true; data: T }
  | { success: false; error: E };

/**
 * Generic Auth Platform client with pluggable components
 */
class AuthPlatformClient<
  TStorage extends TokenStorage = TokenStorage,
  TConfig extends AuthPlatformConfig = AuthPlatformConfig
> {
  private readonly config: TConfig;
  private readonly storage: TStorage;
  private readonly http: HttpClient<TConfig>;
  
  constructor(config: TConfig, storage: TStorage) {
    this.config = config;
    this.storage = storage;
    this.http = new ResilientHttpClient(config);
  }
  
  /**
   * Generic API call with automatic token refresh
   */
  async call<TRequest, TResponse>(
    method: 'GET' | 'POST' | 'PUT' | 'DELETE',
    path: string,
    body?: TRequest
  ): Promise<Result<TResponse, AuthError>> {
    try {
      const tokens = await this.storage.get();
      if (tokens && this.isExpired(tokens.accessToken)) {
        await this.refreshTokens();
      }
      
      const response = await this.http[method.toLowerCase()]<TRequest, TResponse>(path, body);
      return { success: true, data: response };
    } catch (error) {
      return { success: false, error: this.mapError(error) };
    }
  }
}
```

### Generic SDK Client (Python - Generic Types with TypeVar)

```python
# auth_platform_sdk/client.py
from typing import TypeVar, Generic, Optional, Protocol, runtime_checkable
from dataclasses import dataclass
from abc import ABC, abstractmethod

T = TypeVar('T')
TConfig = TypeVar('TConfig', bound='BaseConfig')
TResponse = TypeVar('TResponse')
TError = TypeVar('TError', bound=Exception)

@dataclass
class Result(Generic[T, TError]):
    """Generic result type for operations that can fail."""
    success: bool
    data: Optional[T] = None
    error: Optional[TError] = None
    
    @classmethod
    def ok(cls, data: T) -> 'Result[T, TError]':
        return cls(success=True, data=data)
    
    @classmethod
    def err(cls, error: TError) -> 'Result[T, TError]':
        return cls(success=False, error=error)

@runtime_checkable
class TokenStorage(Protocol[T]):
    """Generic token storage protocol."""
    def get(self) -> Optional[T]: ...
    def set(self, tokens: T) -> None: ...
    def clear(self) -> None: ...

class BaseClient(Generic[TConfig], ABC):
    """Generic base client with common functionality."""
    
    def __init__(self, config: TConfig):
        self.config = config
        self._http = self._create_http_client()
    
    @abstractmethod
    def _create_http_client(self): ...
    
    def _with_retry(self, func, max_retries: int = 3):
        """Generic retry decorator with exponential backoff."""
        for attempt in range(max_retries):
            try:
                return func()
            except RateLimitError as e:
                if attempt == max_retries - 1:
                    raise
                time.sleep(2 ** attempt)
        raise MaxRetriesExceeded()

class AuthPlatformClient(BaseClient[AuthPlatformConfig]):
    """Synchronous Auth Platform client with generics."""
    
    def validate_token(self, token: str) -> Result[TokenClaims, ValidationError]:
        """Validate JWT and return typed result."""
        try:
            claims = self._validate_jwt(token)
            return Result.ok(claims)
        except ValidationError as e:
            return Result.err(e)
```

### Generic SDK Client (Go - Generics with Type Constraints)

```go
// auth-platform-sdk-go/client.go
package authplatform

import (
    "context"
    "encoding/json"
)

// Result is a generic result type for operations that can fail
type Result[T any] struct {
    Data  T
    Error error
}

// Ok creates a successful result
func Ok[T any](data T) Result[T] {
    return Result[T]{Data: data}
}

// Err creates a failed result
func Err[T any](err error) Result[T] {
    return Result[T]{Error: err}
}

// IsOk returns true if the result is successful
func (r Result[T]) IsOk() bool {
    return r.Error == nil
}

// TokenStorage is a generic interface for token storage
type TokenStorage[T any] interface {
    Get(ctx context.Context) (T, error)
    Set(ctx context.Context, tokens T) error
    Clear(ctx context.Context) error
}

// EventHandler is a generic interface for CAEP event handlers
type EventHandler[E any, R any] interface {
    Handle(ctx context.Context, event E) (R, error)
    CanHandle(event E) bool
}

// EventProcessor processes events with registered handlers
type EventProcessor[E any, R any] struct {
    handlers []EventHandler[E, R]
}

// NewEventProcessor creates a new event processor
func NewEventProcessor[E any, R any]() *EventProcessor[E, R] {
    return &EventProcessor[E, R]{
        handlers: make([]EventHandler[E, R], 0),
    }
}

// Register adds a handler to the processor
func (p *EventProcessor[E, R]) Register(handler EventHandler[E, R]) *EventProcessor[E, R] {
    p.handlers = append(p.handlers, handler)
    return p
}

// Process processes an event through all applicable handlers
func (p *EventProcessor[E, R]) Process(ctx context.Context, event E) ([]R, error) {
    var results []R
    for _, handler := range p.handlers {
        if handler.CanHandle(event) {
            result, err := handler.Handle(ctx, event)
            if err != nil {
                return nil, err
            }
            results = append(results, result)
        }
    }
    return results, nil
}

// Client is the generic Auth Platform client
type Client[TStorage TokenStorage[TokenData]] struct {
    config  Config
    storage TStorage
    http    *resilientHTTPClient
}

// NewClient creates a new client with generic storage
func NewClient[TStorage TokenStorage[TokenData]](config Config, storage TStorage, opts ...Option) *Client[TStorage] {
    c := &Client[TStorage]{
        config:  config,
        storage: storage,
        http:    newResilientHTTPClient(config),
    }
    for _, opt := range opts {
        opt(c)
    }
    return c
}
```

### Design Patterns Applied

#### 1. Repository Pattern (Passkey Storage)

```elixir
# Abstraction over data access
defmodule MfaService.Passkeys.Repository do
  @callback find_by_user(user_id :: String.t()) :: [Credential.t()]
  @callback find_by_credential_id(credential_id :: binary()) :: Credential.t() | nil
  @callback save(Credential.t()) :: {:ok, Credential.t()} | {:error, term()}
  @callback delete(Credential.t()) :: :ok | {:error, term()}
end
```

#### 2. Strategy Pattern (Authentication Methods)

```rust
// Pluggable authentication strategies
pub trait AuthStrategy: Send + Sync {
    fn authenticate(&self, request: &AuthRequest) -> Result<AuthResult, AuthError>;
    fn supports(&self, method: &str) -> bool;
}

pub struct PasskeyStrategy { /* ... */ }
pub struct TotpStrategy { /* ... */ }
pub struct PasswordStrategy { /* ... */ }

impl AuthStrategy for PasskeyStrategy {
    fn authenticate(&self, request: &AuthRequest) -> Result<AuthResult, AuthError> {
        // Passkey-specific authentication
    }
    
    fn supports(&self, method: &str) -> bool {
        method == "passkey" || method == "webauthn"
    }
}
```

#### 3. Observer Pattern (CAEP Events)

```typescript
// Event subscription and notification
interface CaepObserver {
  onEvent(event: CaepEvent): void;
}

class CaepEventBus {
  private observers: Map<CaepEventType, CaepObserver[]> = new Map();
  
  subscribe(eventType: CaepEventType, observer: CaepObserver): Unsubscribe {
    const observers = this.observers.get(eventType) || [];
    observers.push(observer);
    this.observers.set(eventType, observers);
    
    return () => {
      const idx = observers.indexOf(observer);
      if (idx > -1) observers.splice(idx, 1);
    };
  }
  
  notify(event: CaepEvent): void {
    const observers = this.observers.get(event.type) || [];
    observers.forEach(o => o.onEvent(event));
  }
}
```

#### 4. Builder Pattern (SDK Configuration)

```go
// Fluent configuration builder
type ClientBuilder struct {
    config Config
    opts   []Option
}

func NewClientBuilder() *ClientBuilder {
    return &ClientBuilder{
        config: DefaultConfig(),
        opts:   make([]Option, 0),
    }
}

func (b *ClientBuilder) WithBaseURL(url string) *ClientBuilder {
    b.config.BaseURL = url
    return b
}

func (b *ClientBuilder) WithTimeout(d time.Duration) *ClientBuilder {
    b.opts = append(b.opts, WithTimeout(d))
    return b
}

func (b *ClientBuilder) WithRetry(maxRetries int) *ClientBuilder {
    b.opts = append(b.opts, WithRetry(maxRetries))
    return b
}

func (b *ClientBuilder) Build() *Client {
    return New(b.config, b.opts...)
}
```

#### 5. Circuit Breaker Pattern (Resilient Clients)

```rust
// Generic circuit breaker for any operation
pub struct CircuitBreaker<T, E> {
    state: AtomicU8,
    failure_count: AtomicU32,
    failure_threshold: u32,
    recovery_timeout: Duration,
    last_failure: Mutex<Option<Instant>>,
    _phantom: PhantomData<(T, E)>,
}

impl<T, E> CircuitBreaker<T, E> {
    pub async fn call<F, Fut>(&self, f: F) -> Result<T, CircuitBreakerError<E>>
    where
        F: FnOnce() -> Fut,
        Fut: Future<Output = Result<T, E>>,
    {
        match self.state() {
            State::Open => {
                if self.should_attempt_reset() {
                    self.set_state(State::HalfOpen);
                } else {
                    return Err(CircuitBreakerError::Open);
                }
            }
            _ => {}
        }
        
        match f().await {
            Ok(result) => {
                self.on_success();
                Ok(result)
            }
            Err(e) => {
                self.on_failure();
                Err(CircuitBreakerError::Inner(e))
            }
        }
    }
}
```

### Best Practices Applied

| Practice | Implementation | Location |
|----------|----------------|----------|
| Type Safety | Generics with constraints | All SDKs |
| Error Handling | Result<T, E> pattern | All components |
| Dependency Injection | Constructor injection | All services |
| Interface Segregation | Small, focused traits/interfaces | Providers, Handlers |
| Single Responsibility | One purpose per module | All modules |
| Open/Closed | Extensible via traits/interfaces | Strategies, Handlers |
| Immutability | Immutable data structures | Events, Credentials |
| Fail Fast | Early validation | All inputs |
| Graceful Degradation | Fallbacks, circuit breakers | Clients, Services |
| Observability | Structured logging, metrics | All components |

## Error Handling

### Passkey Errors

| Error | Handling | Recovery |
|-------|----------|----------|
| NotAllowedError | User cancelled or timeout | Retry with user prompt |
| InvalidStateError | Credential already exists | Offer to manage existing |
| NotSupportedError | Browser/device not supported | Fallback to TOTP |
| SecurityError | Origin mismatch | Check RP ID configuration |
| AbortError | Operation aborted | Retry or fallback |

### CAEP Errors

| Error | Handling | Recovery |
|-------|----------|----------|
| Invalid signature | Reject event, log | Refresh JWKS, retry |
| Unknown event type | Log warning | Ignore or forward |
| Stream delivery failed | Retry with backoff | Alert after 3 failures |
| Subject not found | Log, no action | Ignore stale events |

### SDK Errors

| Error | Handling | Recovery |
|-------|----------|----------|
| TokenExpiredError | Auto-refresh | Retry original request |
| NetworkError | Retry with backoff | Surface after 3 retries |
| InvalidConfigError | Fail fast | Fix configuration |
| RateLimitError | Wait and retry | Exponential backoff |

## Testing Strategy

### Dual Testing Approach

This implementation uses both unit tests and property-based tests:

- **Unit tests**: Verify specific examples, edge cases, and error conditions
- **Property-based tests**: Verify universal properties across all valid inputs

### Property-Based Testing Framework

- **Elixir services**: `StreamData` with 100+ iterations per property
- **Rust services**: `proptest` with 100+ iterations per property
- **TypeScript SDK**: `fast-check` with 100+ iterations per property
- **Python SDK**: `hypothesis` with 100+ iterations per property
- **Go SDK**: `gopter` with 100+ iterations per property

### Test Categories

1. **Passkey Tests**
   - Registration options generation
   - Attestation verification
   - Authentication ceremony
   - Credential management

2. **CAEP Tests**
   - SET generation and signing
   - Signature validation
   - Event processing
   - Stream management

3. **SDK Tests**
   - OAuth flow with PKCE
   - Token validation and refresh
   - Error handling
   - Middleware integration

### Test Annotations

Each property-based test MUST be annotated with:
```
**Feature: auth-platform-q2-2025-evolution, Property {number}: {property_text}**
**Validates: Requirements {X.Y}**
```

