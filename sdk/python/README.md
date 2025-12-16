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

The SDK automatically caches JWKS for efficient token validation:

```python
from auth_platform_sdk import JWKSCache

# Custom cache TTL (default: 1 hour)
config = AuthPlatformConfig(
    base_url="https://auth.example.com",
    client_id="your-client-id",
    jwks_cache_ttl=3600,  # seconds
)

# Validate tokens
claims = client.validate_token(token)
print(f"Subject: {claims.sub}")
print(f"Issuer: {claims.iss}")
print(f"Expires: {claims.exp}")
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

## Error Handling

```python
from auth_platform_sdk import (
    AuthPlatformError,
    TokenExpiredError,
    ValidationError,
    RateLimitError,
)

try:
    claims = client.validate_token(token)
except TokenExpiredError:
    # Token has expired
    pass
except ValidationError as e:
    # Token validation failed
    print(f"Validation error: {e}")
except RateLimitError as e:
    # Rate limited, retry after delay
    time.sleep(e.retry_after or 60)
except AuthPlatformError as e:
    # Other SDK error
    print(f"Error: {e.code} - {e}")
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

## License

MIT
