# Auth Platform Python SDK

Official Python SDK for the Auth Platform. Supports both synchronous and asynchronous clients, JWKS caching, and framework middleware.

## Installation

```bash
pip install auth-platform-sdk
# or
poetry add auth-platform-sdk
```

## Quick Start

```python
from auth_platform_sdk import AuthPlatformClient, AuthPlatformConfig

config = AuthPlatformConfig(
    base_url="https://auth.example.com",
    client_id="your-client-id",
    client_secret="your-client-secret",
)

# Synchronous client
with AuthPlatformClient(config) as client:
    # Client credentials flow
    tokens = client.client_credentials()
    
    # Validate a token
    claims = client.validate_token(access_token)
    print(f"User: {claims.sub}")
```

## Async Client

```python
from auth_platform_sdk import AsyncAuthPlatformClient, AuthPlatformConfig

config = AuthPlatformConfig(
    base_url="https://auth.example.com",
    client_id="your-client-id",
    client_secret="your-client-secret",
)

async with AsyncAuthPlatformClient(config) as client:
    tokens = await client.client_credentials()
    claims = await client.validate_token(access_token)
```

## Token Validation with JWKS Caching

The SDK automatically caches JWKS for efficient token validation with refresh-ahead support:

```python
from auth_platform_sdk import JWKSCache, AsyncJWKSCache

# Custom cache configuration
config = AuthPlatformConfig(
    base_url="https://auth.example.com",
    client_id="your-client-id",
    cache=CacheConfig(
        jwks_ttl=3600,           # Cache TTL in seconds (default: 1 hour)
        jwks_refresh_ahead=300,  # Refresh 5 minutes before expiry
    ),
)

# Validate tokens
claims = client.validate_token(token)
print(f"Subject: {claims.sub}")
print(f"Issuer: {claims.iss}")
print(f"Expires: {claims.exp}")
```

### Direct JWKS Cache Usage

For advanced use cases, you can use the JWKS cache directly:

```python
from auth_platform_sdk import JWKSCache

# Synchronous cache with refresh-ahead
cache = JWKSCache(
    jwks_uri="https://auth.example.com/.well-known/jwks.json",
    ttl_seconds=3600,
    refresh_ahead_seconds=300,  # Refresh before expiry
    http_timeout=10.0,
)

# Get signing key for token validation
signing_key = cache.get_signing_key(token)

# Get specific key by ID
key = cache.get_key_by_id("key-id-123")

# Check cache status
if cache.is_cached:
    print("JWKS is cached and valid")

# Force cache refresh
cache.invalidate()
```

### Async JWKS Cache

```python
from auth_platform_sdk import AsyncJWKSCache

cache = AsyncJWKSCache(
    jwks_uri="https://auth.example.com/.well-known/jwks.json",
    ttl_seconds=3600,
    refresh_ahead_seconds=300,
    http_timeout=10.0,
)

# Get signing key by ID
key = await cache.get_signing_key("key-id-123")

# Get all signing keys
all_keys = await cache.get_all_signing_keys()

# Invalidate cache
await cache.invalidate()
```

## Framework Middleware

### FastAPI

```python
from fastapi import FastAPI, Depends
from auth_platform_sdk import AuthPlatformConfig
from auth_platform_sdk.middleware import create_fastapi_middleware

app = FastAPI()
config = AuthPlatformConfig(
    base_url="https://auth.example.com",
    client_id="your-client-id",
)

get_current_user = create_fastapi_middleware(config)

@app.get("/protected")
async def protected_route(user: TokenClaims = Depends(get_current_user)):
    return {"user": user.sub}
```

### Flask

```python
from flask import Flask, g
from auth_platform_sdk import AuthPlatformConfig
from auth_platform_sdk.middleware import create_flask_middleware

app = Flask(__name__)
config = AuthPlatformConfig(
    base_url="https://auth.example.com",
    client_id="your-client-id",
)

require_auth = create_flask_middleware(config)

@app.route("/protected")
@require_auth
def protected_route():
    return {"user": g.current_user.sub}
```

### Django

```python
# settings.py
from auth_platform_sdk import AuthPlatformConfig
from auth_platform_sdk.middleware import create_django_middleware

AUTH_PLATFORM_CONFIG = AuthPlatformConfig(
    base_url="https://auth.example.com",
    client_id="your-client-id",
)

MIDDLEWARE = [
    # ...
    'your_app.middleware.AuthPlatformMiddleware',
]

# middleware.py
from django.conf import settings
from auth_platform_sdk.middleware import create_django_middleware

AuthPlatformMiddleware = create_django_middleware(settings.AUTH_PLATFORM_CONFIG)
```

## Rate Limiting

The SDK automatically handles rate limiting with exponential backoff:

```python
config = AuthPlatformConfig(
    base_url="https://auth.example.com",
    client_id="your-client-id",
    max_retries=3,
    retry_delay=1.0,  # Initial delay in seconds
)
```

## Circuit Breaker

The SDK includes a circuit breaker for resilience against failing services:

```python
from auth_platform_sdk.http import CircuitBreaker

# Create circuit breaker with custom settings
circuit_breaker = CircuitBreaker(
    failure_threshold=5,      # Open after 5 failures
    recovery_timeout=30.0,    # Try again after 30 seconds
    half_open_requests=1,     # Requests to test recovery
)

# Circuit states: CLOSED (normal) -> OPEN (failing) -> HALF_OPEN (testing)
print(circuit_breaker.state)  # CircuitState.CLOSED
```

The circuit breaker automatically:
- Opens after consecutive failures to prevent cascading failures
- Transitions to half-open state after recovery timeout
- Closes again after successful requests in half-open state

## Error Handling

All errors include structured information with error codes, correlation IDs, and details for observability:

```python
from auth_platform_sdk import (
    AuthPlatformError,
    TokenExpiredError,
    TokenInvalidError,
    ValidationError,
    RateLimitError,
    DPoPError,
    PKCEError,
    NetworkError,
    TimeoutError,
    ServerError,
    ErrorCode,
)

try:
    claims = client.validate_token(token)
except TokenExpiredError as e:
    # Token has expired (AUTH_1001)
    print(f"Correlation ID: {e.correlation_id}")
except TokenInvalidError as e:
    # Token is invalid or malformed (AUTH_1002)
    print(f"Details: {e.details}")
except ValidationError as e:
    # Input validation failed (VAL_2001)
    print(f"Validation error: {e.message}")
except RateLimitError as e:
    # Rate limited (RATE_4001), retry after delay
    time.sleep(e.retry_after or 60)
except DPoPError as e:
    # DPoP proof required or invalid (DPOP_6xxx)
    if e.dpop_nonce:
        # Retry with new nonce
        pass
except PKCEError as e:
    # PKCE challenge/verifier error (PKCE_7xxx)
    pass
except NetworkError as e:
    # Network request failed (NET_3001)
    print(f"Cause: {e.__cause__}")
except TimeoutError as e:
    # Request timed out (NET_3002)
    pass
except ServerError as e:
    # Server-side error (SRV_5001)
    print(f"Status: {e.status_code}")
except AuthPlatformError as e:
    # Base error with structured info
    print(f"Error: {e.code} - {e.message}")
    print(f"Serialized: {e.to_dict()}")
```

### Error Codes

| Category | Code Range | Examples |
|----------|------------|----------|
| Authentication | AUTH_1xxx | TOKEN_EXPIRED, TOKEN_INVALID, UNAUTHORIZED |
| Validation | VAL_2xxx | VALIDATION_ERROR, INVALID_CONFIG, INVALID_SCOPE |
| Network | NET_3xxx | NETWORK_ERROR, TIMEOUT_ERROR, CONNECTION_ERROR |
| Rate Limiting | RATE_4xxx | RATE_LIMITED, QUOTA_EXCEEDED |
| Server | SRV_5xxx | SERVER_ERROR, SERVICE_UNAVAILABLE |
| DPoP | DPOP_6xxx | DPOP_REQUIRED, DPOP_INVALID, DPOP_NONCE_REQUIRED |
| PKCE | PKCE_7xxx | PKCE_REQUIRED, PKCE_INVALID |

## Core Components (December 2025 State of Art)

The SDK provides centralized core components that eliminate code duplication between sync and async clients:

```python
from auth_platform_sdk import (
    # Error handling
    ErrorFactory,
    # JWKS caching
    JWKSCacheBase,
    # Token operations
    TokenOperations,
    TokenValidator,
    # Authorization
    AuthorizationBuilder,
    # HTTP execution
    SyncHTTPExecutor,
    AsyncHTTPExecutor,
)
```

### ErrorFactory

Centralized error creation with consistent structure, correlation IDs, and HTTP response transformation:

```python
from auth_platform_sdk import ErrorFactory
import httpx

# Transform HTTP response to SDK error
response = httpx.Response(429, headers={"Retry-After": "60"})
error = ErrorFactory.from_http_response(response, correlation_id="req-123")
# Returns RateLimitError with retry_after=60

# Transform exceptions
try:
    # ... network operation
except httpx.TimeoutException as e:
    error = ErrorFactory.from_exception(e, correlation_id="req-456")
    # Returns TimeoutError with correlation ID

# Create token validation errors with metadata
error = ErrorFactory.token_validation_error(
    "Invalid signature",
    token_metadata={"kid": "key-1", "alg": "ES256"},
)
```

### JWKSCacheBase

Base class with shared TTL and refresh-ahead logic for JWKS caching:

```python
from auth_platform_sdk import JWKSCacheBase

cache = JWKSCacheBase(
    jwks_uri="https://auth.example.com/.well-known/jwks.json",
    ttl_seconds=3600,
    refresh_ahead_seconds=300,
)

# Check cache status
if cache.should_refresh():
    # Time to refresh (within refresh-ahead window)
    pass

if cache.is_expired():
    # Cache is fully expired
    pass

# Timing information
print(f"Refresh in: {cache.time_until_refresh()}s")
print(f"Expires in: {cache.time_until_expiry()}s")

# Key lookup
key = cache.get_key("key-id-123")
signing_keys = cache.get_signing_keys()
```

### TokenValidator

Centralized JWT validation shared by sync and async clients:

```python
from auth_platform_sdk import TokenValidator, AuthPlatformConfig
from auth_platform_sdk.models import JWK

config = AuthPlatformConfig(
    base_url="https://auth.example.com",
    client_id="your-client-id",
)
validator = TokenValidator(config)

# Validate token with JWK
claims = validator.validate(
    token,
    jwk,
    issuer="https://auth.example.com",
)

# Extract key ID from token header
kid = validator.get_key_id(token)

# Verify DPoP binding
is_bound = validator.verify_dpop_binding(claims, expected_thumbprint)
```

### AuthorizationBuilder

Centralized authorization URL construction with PKCE support:

```python
from auth_platform_sdk import AuthorizationBuilder, AuthPlatformConfig

config = AuthPlatformConfig(
    base_url="https://auth.example.com",
    client_id="your-client-id",
)
builder = AuthorizationBuilder(config)

# Build authorization URL with PKCE
url, state, pkce = builder.build_authorization_url(
    redirect_uri="https://app.example.com/callback",
    scopes=["openid", "profile"],
    use_pkce=True,
)

# Parse callback URL
code, error = builder.parse_callback_url(callback_url, expected_state=state)
```

### TokenOperations

Centralized token request building and processing:

```python
from auth_platform_sdk import TokenOperations, AuthPlatformConfig

config = AuthPlatformConfig(
    base_url="https://auth.example.com",
    client_id="your-client-id",
    client_secret="your-secret",
)
ops = TokenOperations(config)

# Build request payloads
cc_request = ops.build_client_credentials_request(scopes=["read", "write"])
refresh_request = ops.build_refresh_token_request(refresh_token)
auth_code_request = ops.build_authorization_code_request(
    code="auth-code",
    redirect_uri="https://app.example.com/callback",
    code_verifier="pkce-verifier",
)

# Build headers with DPoP support
headers = ops.build_token_request_headers()
```

### HTTP Executors

Centralized HTTP execution with retry and circuit breaker:

```python
from auth_platform_sdk import SyncHTTPExecutor, AsyncHTTPExecutor
from auth_platform_sdk.config import RetryConfig
from auth_platform_sdk.http import CircuitBreaker
import httpx

# Sync executor
client = httpx.Client()
retry_config = RetryConfig(max_retries=3, initial_delay=1.0)
executor = SyncHTTPExecutor(client, retry_config)

response = executor.execute("POST", "https://api.example.com/token", data=payload)

# Async executor
async_client = httpx.AsyncClient()
async_executor = AsyncHTTPExecutor(async_client, retry_config)

response = await async_executor.execute("GET", "https://api.example.com/resource")
```

## API Reference

### AuthPlatformClient

| Method | Description |
|--------|-------------|
| `validate_token(token)` | Validate JWT and return claims |
| `client_credentials()` | Obtain token via client credentials |
| `get_access_token()` | Get valid access token |

### AsyncAuthPlatformClient

Same methods as `AuthPlatformClient`, but async.

### Core Components

| Component | Description |
|-----------|-------------|
| `ErrorFactory` | Centralized error creation with correlation IDs |
| `JWKSCacheBase` | Base JWKS cache with TTL and refresh-ahead logic |
| `TokenValidator` | Centralized JWT validation |
| `AuthorizationBuilder` | Authorization URL construction with PKCE |
| `TokenOperations` | Token request building and processing |
| `SyncHTTPExecutor` | Sync HTTP with retry and circuit breaker |
| `AsyncHTTPExecutor` | Async HTTP with retry and circuit breaker |

## License

MIT
