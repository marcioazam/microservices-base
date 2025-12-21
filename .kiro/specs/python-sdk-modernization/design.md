# Design Document: Python SDK Modernization

## Overview

This design document describes the architecture and implementation approach for modernizing the Auth Platform Python SDK to December 2025 state-of-art standards. The modernization eliminates redundancies, centralizes logic, improves test organization, and ensures compliance with current Python best practices.

### Key Findings from Analysis

**Redundancies Identified:**
1. `types.py` contains duplicate dataclass definitions that mirror Pydantic models in `models.py`
2. `test_jwks_cache.py` is at root of tests/ instead of in unit/ directory

**Current State:**
- Python 3.11+ support ✓
- Pydantic v2 ✓
- httpx for HTTP ✓
- Hypothesis for PBT ✓
- Ruff for linting ✓
- OpenTelemetry integration ✓

**Improvements Needed:**
- Remove duplicate type definitions
- Reorganize test structure
- Add missing property tests
- Ensure consistent error handling

## Architecture

```
sdk/python/
├── pyproject.toml              # Single source of configuration
├── README.md                   # Documentation
├── src/
│   └── auth_platform_sdk/
│       ├── __init__.py         # Public API exports
│       ├── client.py           # Sync client
│       ├── async_client.py     # Async client
│       ├── config.py           # Configuration (Pydantic v2)
│       ├── models.py           # Data models (single source of truth)
│       ├── errors.py           # Error hierarchy
│       ├── http.py             # HTTP client, retry, circuit breaker
│       ├── jwks.py             # JWKS caching
│       ├── dpop.py             # DPoP implementation (RFC 9449)
│       ├── pkce.py             # PKCE implementation (RFC 7636)
│       ├── middleware.py       # Framework middleware
│       └── telemetry.py        # OpenTelemetry integration
└── tests/
    ├── __init__.py
    ├── conftest.py             # Shared fixtures
    ├── unit/
    │   ├── __init__.py
    │   ├── test_config.py
    │   ├── test_models.py
    │   ├── test_errors.py
    │   ├── test_http.py
    │   ├── test_jwks.py
    │   ├── test_dpop.py
    │   ├── test_pkce.py
    │   └── test_telemetry.py
    ├── integration/
    │   ├── __init__.py
    │   ├── test_client.py
    │   ├── test_async_client.py
    │   └── test_middleware.py
    └── property/
        ├── __init__.py
        ├── test_dpop_properties.py
        ├── test_pkce_properties.py
        ├── test_jwks_properties.py
        ├── test_config_properties.py
        └── test_errors_properties.py
```

## Components and Interfaces

### 1. Configuration Module (`config.py`)

Centralized configuration using Pydantic v2 with frozen models for immutability.

```python
from pydantic import BaseModel, ConfigDict, Field, HttpUrl, SecretStr

class RetryConfig(BaseModel):
    """Retry configuration with exponential backoff."""
    model_config = ConfigDict(frozen=True)
    
    max_retries: int = Field(default=3, ge=0, le=10)
    initial_delay: float = Field(default=1.0, gt=0, le=60)
    max_delay: float = Field(default=30.0, gt=0, le=300)
    exponential_base: float = Field(default=2.0, ge=1.5, le=3.0)
    jitter: float = Field(default=0.1, ge=0, le=1.0)
    
    def get_delay(self, attempt: int) -> float:
        """Calculate delay with exponential backoff and jitter."""
        delay = min(self.initial_delay * (self.exponential_base ** attempt), self.max_delay)
        jitter_range = delay * self.jitter
        return delay + random.uniform(-jitter_range, jitter_range)

class AuthPlatformConfig(BaseModel):
    """Main SDK configuration."""
    model_config = ConfigDict(frozen=True, validate_default=True)
    
    base_url: HttpUrl
    client_id: str = Field(..., min_length=1)
    client_secret: SecretStr | None = None
    # ... other fields
```

### 2. Models Module (`models.py`)

Single source of truth for all data models using Pydantic v2.

```python
class TokenResponse(BaseModel):
    """OAuth 2.0 token response - SINGLE DEFINITION."""
    model_config = ConfigDict(frozen=True, extra="ignore")
    
    access_token: str = Field(..., min_length=1)
    token_type: str = Field(default="Bearer")
    expires_in: int = Field(..., gt=0)
    refresh_token: str | None = None
    scope: str | None = None
    id_token: str | None = None

class TokenData(BaseModel):
    """Internal token storage - SINGLE DEFINITION."""
    model_config = ConfigDict(frozen=True)
    
    access_token: str
    token_type: str
    expires_at: datetime
    refresh_token: str | None = None
    
    def is_expired(self) -> bool:
        return datetime.now(UTC) >= self.expires_at

class TokenClaims(BaseModel):
    """JWT claims - SINGLE DEFINITION."""
    model_config = ConfigDict(frozen=True, extra="allow")
    
    sub: str
    iss: str
    aud: str | list[str]
    exp: int
    iat: int
```

### 3. Error Module (`errors.py`)

Centralized error hierarchy with structured error information.

```python
class ErrorCode(StrEnum):
    """Standardized error codes."""
    TOKEN_EXPIRED = "AUTH_1001"
    TOKEN_INVALID = "AUTH_1002"
    VALIDATION_ERROR = "VAL_2001"
    NETWORK_ERROR = "NET_3001"
    RATE_LIMITED = "RATE_4001"
    SERVER_ERROR = "SRV_5001"
    DPOP_INVALID = "DPOP_6002"
    PKCE_INVALID = "PKCE_7002"

class AuthPlatformError(Exception):
    """Base error with structured information."""
    
    def __init__(
        self,
        message: str,
        code: ErrorCode | str,
        *,
        status_code: int | None = None,
        correlation_id: str | None = None,
        details: dict[str, Any] | None = None,
    ) -> None:
        super().__init__(message)
        self.message = message
        self.code = code if isinstance(code, str) else code.value
        self.status_code = status_code
        self.correlation_id = correlation_id
        self.details = details or {}
    
    def to_dict(self) -> dict[str, Any]:
        """Serialize error for logging/API responses."""
        return {
            "error": self.message,
            "code": self.code,
            "status_code": self.status_code,
            "correlation_id": self.correlation_id,
            "details": self.details,
        }
```

### 4. HTTP Module (`http.py`)

Centralized HTTP client creation, retry logic, and circuit breaker.

```python
class CircuitBreaker:
    """Circuit breaker for resilience."""
    
    def __init__(
        self,
        failure_threshold: int = 5,
        recovery_timeout: float = 30.0,
        half_open_requests: int = 1,
    ) -> None:
        self._state = CircuitState.CLOSED
        self._failure_count = 0
        self._last_failure_time = 0.0
    
    def allow_request(self) -> bool:
        return self.state != CircuitState.OPEN
    
    def record_success(self) -> None:
        # Transition to CLOSED if in HALF_OPEN
        pass
    
    def record_failure(self) -> None:
        # Increment failure count, potentially open circuit
        pass

def request_with_retry(
    client: httpx.Client,
    method: str,
    url: str,
    retry_config: RetryConfig,
    *,
    circuit_breaker: CircuitBreaker | None = None,
    **kwargs: Any,
) -> httpx.Response:
    """Make HTTP request with retry and circuit breaker."""
    for attempt in range(retry_config.max_retries + 1):
        if circuit_breaker and not circuit_breaker.allow_request():
            raise NetworkError("Circuit breaker is open")
        
        try:
            response = client.request(method, url, **kwargs)
            if circuit_breaker:
                circuit_breaker.record_success()
            return response
        except httpx.HTTPError as e:
            if circuit_breaker:
                circuit_breaker.record_failure()
            if attempt < retry_config.max_retries:
                time.sleep(retry_config.get_delay(attempt))
            else:
                raise NetworkError(str(e), cause=e) from e
```

### 5. JWKS Cache (`jwks.py`)

Thread-safe and async-safe JWKS caching with refresh-ahead.

```python
class JWKSCache:
    """Thread-safe JWKS cache."""
    
    def __init__(
        self,
        jwks_uri: str,
        *,
        ttl_seconds: int = 3600,
        refresh_ahead_seconds: int = 300,
    ) -> None:
        self._jwks: JWKS | None = None
        self._cache_time: float = 0
        self._lock = threading.RLock()
    
    def _should_refresh(self) -> bool:
        if self._jwks is None:
            return True
        elapsed = time.time() - self._cache_time
        return elapsed > (self.ttl_seconds - self.refresh_ahead_seconds)
    
    def invalidate(self) -> None:
        with self._lock:
            self._jwks = None
            self._cache_time = 0
```

### 6. DPoP Implementation (`dpop.py`)

RFC 9449 compliant DPoP proof generation and verification.

```python
class DPoPKeyPair:
    """DPoP key pair for proof generation."""
    
    def __init__(self, algorithm: str = "ES256") -> None:
        self._private_key = ec.generate_private_key(self._get_curve(algorithm))
        self._thumbprint = self._compute_thumbprint()
    
    def create_proof(
        self,
        http_method: str,
        http_uri: str,
        *,
        access_token: str | None = None,
        nonce: str | None = None,
    ) -> DPoPProof:
        """Create DPoP proof JWT per RFC 9449."""
        header = {"typ": "dpop+jwt", "alg": self.algorithm, "jwk": self._jwk}
        payload = {
            "jti": str(uuid.uuid4()),
            "htm": http_method.upper(),
            "htu": http_uri,
            "iat": int(time.time()),
        }
        if access_token:
            payload["ath"] = self._hash_token(access_token)
        if nonce:
            payload["nonce"] = nonce
        
        proof = jwt.encode(payload, self._private_key, algorithm=self.algorithm, headers=header)
        return DPoPProof(proof=proof, thumbprint=self._thumbprint, nonce=nonce)
```

### 7. PKCE Implementation (`pkce.py`)

RFC 7636 compliant PKCE challenge generation and verification.

```python
def generate_code_verifier(length: int = 64) -> str:
    """Generate cryptographically random code verifier."""
    if not 43 <= length <= 128:
        raise ValueError("Code verifier length must be between 43 and 128")
    return secrets.token_urlsafe(length)[:length]

def generate_code_challenge(code_verifier: str) -> str:
    """Generate S256 code challenge."""
    digest = hashlib.sha256(code_verifier.encode("ascii")).digest()
    return base64.urlsafe_b64encode(digest).decode().rstrip("=")

def verify_code_challenge(code_verifier: str, code_challenge: str) -> bool:
    """Verify code challenge with timing-safe comparison."""
    expected = generate_code_challenge(code_verifier)
    return secrets.compare_digest(expected, code_challenge)
```

## Data Models

All data models are defined in `models.py` using Pydantic v2:

| Model | Purpose | Frozen |
|-------|---------|--------|
| TokenResponse | OAuth 2.0 token response | Yes |
| TokenData | Internal token storage | Yes |
| TokenClaims | JWT claims | Yes |
| PKCEChallenge | PKCE verifier/challenge pair | Yes |
| DPoPProof | DPoP proof JWT | Yes |
| JWK | JSON Web Key | Yes |
| JWKS | JSON Web Key Set | Yes |
| AuthorizationRequest | OAuth authorization params | Yes |
| TokenRequest | OAuth token request params | Yes |

## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system—essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*

### Property 1: Error Serialization Round-Trip

*For any* AuthPlatformError instance with any combination of message, code, status_code, correlation_id, and details, calling `to_dict()` SHALL produce a dictionary that contains all the original values.

**Validates: Requirements 5.5**

### Property 2: Retry Exponential Backoff

*For any* RetryConfig and attempt number, the calculated delay SHALL be:
- Greater than or equal to initial_delay for attempt 0
- Less than or equal to max_delay for all attempts
- Following the formula: `min(initial_delay * (base ** attempt), max_delay) ± jitter`

**Validates: Requirements 2.3**

### Property 3: JWKS Cache TTL Behavior

*For any* JWKSCache with TTL t and refresh_ahead r, the cache SHALL:
- Return `_should_refresh() == True` when elapsed time > (t - r)
- Return `_should_refresh() == True` when cache is empty
- Return `_should_refresh() == False` when elapsed time <= (t - r) and cache is populated

**Validates: Requirements 6.1, 6.2**

### Property 4: JWKS Cache Thread Safety

*For any* JWKSCache, concurrent calls to `invalidate()` from multiple threads SHALL NOT raise exceptions and SHALL result in cache being invalidated.

**Validates: Requirements 6.3**

### Property 5: DPoP Proof Structure

*For any* DPoPKeyPair and valid HTTP method/URI, the generated proof SHALL:
- Have header with `typ: "dpop+jwt"`, `alg`, and `jwk`
- Have payload with `jti`, `htm`, `htu`, and `iat`
- Include `ath` claim when access_token is provided
- Include `nonce` claim when nonce is provided

**Validates: Requirements 7.1, 7.4, 7.5**

### Property 6: DPoP Thumbprint Determinism

*For any* DPoPKeyPair, multiple calls to `thumbprint` property SHALL return the same value.

**Validates: Requirements 7.3**

### Property 7: DPoP Algorithm Support

*For any* algorithm in {ES256, ES384, ES512}, DPoPKeyPair creation SHALL succeed and produce valid proofs.

**Validates: Requirements 7.2**

### Property 8: PKCE Verifier Length

*For any* length in [43, 128], `generate_code_verifier(length)` SHALL return a string of exactly that length containing only URL-safe characters.

**Validates: Requirements 8.1, 8.3**

### Property 9: PKCE Challenge Verification Round-Trip

*For any* valid code verifier, `verify_code_challenge(verifier, generate_code_challenge(verifier))` SHALL return True.

**Validates: Requirements 8.2, 8.4**

### Property 10: PKCE State/Nonce Uniqueness

*For any* N calls to `generate_state()` or `generate_nonce()`, the resulting set SHALL have N unique values (with high probability for cryptographically random generation).

**Validates: Requirements 8.5**

### Property 11: Configuration Immutability

*For any* AuthPlatformConfig instance, attempting to modify any attribute SHALL raise an exception (frozen model).

**Validates: Requirements 11.2**

### Property 12: Configuration Endpoint Derivation

*For any* base_url, the derived endpoints SHALL follow the pattern:
- token_endpoint: `{base_url}/oauth/token`
- authorization_endpoint: `{base_url}/oauth/authorize`
- jwks_uri: `{base_url}/.well-known/jwks.json`

**Validates: Requirements 11.3**

### Property 13: Configuration Validation

*For any* invalid configuration (missing required fields, invalid URLs, out-of-range values), AuthPlatformConfig creation SHALL raise ValidationError.

**Validates: Requirements 11.1, 11.5**

### Property 14: Client Context Manager

*For any* AuthPlatformClient or AsyncAuthPlatformClient, using the context manager protocol SHALL properly close HTTP connections on exit.

**Validates: Requirements 12.3, 12.5**

### Property 15: Error Hierarchy Inheritance

*For any* error class in the SDK error module, it SHALL be a subclass of AuthPlatformError.

**Validates: Requirements 5.1**

## Error Handling

All errors follow the centralized hierarchy:

```
AuthPlatformError (base)
├── TokenExpiredError (AUTH_1001)
├── TokenInvalidError (AUTH_1002)
├── TokenRefreshError (AUTH_1003)
├── ValidationError (VAL_2001)
├── InvalidConfigError (VAL_2002)
├── NetworkError (NET_3001)
├── TimeoutError (NET_3002)
├── RateLimitError (RATE_4001)
├── ServerError (SRV_5001)
├── DPoPError (DPOP_6xxx)
└── PKCEError (PKCE_7xxx)
```

Error handling patterns:
1. All errors include `code`, `message`, and optional `correlation_id`
2. All errors serialize via `to_dict()` for logging
3. Network errors chain the underlying cause
4. Rate limit errors include `retry_after` when available
5. DPoP errors include `dpop_nonce` when server provides one

## Testing Strategy

### Dual Testing Approach

The SDK uses both unit tests and property-based tests:

- **Unit tests**: Verify specific examples, edge cases, and error conditions
- **Property tests**: Verify universal properties across all inputs using Hypothesis

### Test Organization

```
tests/
├── unit/           # Specific examples and edge cases
├── integration/    # Component integration tests
└── property/       # Property-based tests (Hypothesis)
```

### Property-Based Testing Configuration

- Library: Hypothesis
- Minimum iterations: 100 per property
- Tag format: `**Feature: python-sdk-modernization, Property N: description**`

### Test Coverage Requirements

| Component | Unit Tests | Property Tests |
|-----------|------------|----------------|
| config.py | Validation, defaults | Immutability, derivation |
| models.py | Serialization, methods | Round-trip, validation |
| errors.py | Error creation, codes | Hierarchy, serialization |
| http.py | Retry, circuit breaker | Backoff calculation |
| jwks.py | Cache operations | TTL, thread safety |
| dpop.py | Proof creation/verification | Structure, algorithms |
| pkce.py | Challenge generation | Length, uniqueness |
| middleware.py | Framework integration | - |
| telemetry.py | Tracing, logging | - |
