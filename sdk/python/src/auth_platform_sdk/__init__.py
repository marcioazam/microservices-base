"""Auth Platform Python SDK - December 2025 State of Art.

A modern, type-safe Python SDK for the Auth Platform with support for:
- OAuth 2.0 flows (client credentials, authorization code)
- PKCE (Proof Key for Code Exchange) for public clients
- DPoP (Demonstrating Proof of Possession) for sender-constrained tokens
- JWKS caching with automatic refresh
- OpenTelemetry integration for observability
- Framework middleware (FastAPI, Flask, Django)

Example:
    >>> from auth_platform_sdk import AuthPlatformClient, AuthPlatformConfig
    >>> config = AuthPlatformConfig(
    ...     base_url="https://auth.example.com",
    ...     client_id="your-client-id",
    ...     client_secret="your-client-secret",
    ... )
    >>> with AuthPlatformClient(config) as client:
    ...     tokens = client.client_credentials()
    ...     claims = client.validate_token(tokens.access_token)
"""

from .async_client import AsyncAuthPlatformClient
from .client import AuthPlatformClient
from .config import (
    AuthPlatformConfig,
    CacheConfig,
    DPoPConfig,
    RetryConfig,
    TelemetryConfig,
)
from .dpop import DPoPKeyPair, DPoPProof
from .errors import (
    AuthPlatformError,
    DPoPError,
    ErrorCode,
    InvalidConfigError,
    NetworkError,
    PKCEError,
    RateLimitError,
    ServerError,
    TimeoutError,
    TokenExpiredError,
    TokenInvalidError,
    TokenRefreshError,
    ValidationError,
)
from .jwks import AsyncJWKSCache, JWKSCache
from .models import (
    AuthorizationRequest,
    JWK,
    JWKS,
    PKCEChallenge,
    TokenClaims,
    TokenData,
    TokenRequest,
    TokenResponse,
)
from .pkce import (
    create_pkce_challenge,
    generate_code_challenge,
    generate_code_verifier,
    generate_nonce,
    generate_state,
    verify_code_challenge,
)

# Core components (December 2025 State of Art)
from .core import (
    AuthorizationBuilder,
    AsyncHTTPExecutor,
    ErrorFactory,
    JWKSCacheBase,
    SyncHTTPExecutor,
    TokenOperations,
    TokenValidator,
)

__all__ = [
    # Clients
    "AuthPlatformClient",
    "AsyncAuthPlatformClient",
    # Configuration
    "AuthPlatformConfig",
    "RetryConfig",
    "TelemetryConfig",
    "DPoPConfig",
    "CacheConfig",
    # Models
    "TokenResponse",
    "TokenData",
    "TokenClaims",
    "TokenRequest",
    "AuthorizationRequest",
    "PKCEChallenge",
    "DPoPProof",
    "JWK",
    "JWKS",
    # Errors
    "AuthPlatformError",
    "ErrorCode",
    "TokenExpiredError",
    "TokenInvalidError",
    "TokenRefreshError",
    "ValidationError",
    "NetworkError",
    "TimeoutError",
    "RateLimitError",
    "InvalidConfigError",
    "DPoPError",
    "PKCEError",
    "ServerError",
    # JWKS
    "JWKSCache",
    "AsyncJWKSCache",
    # DPoP
    "DPoPKeyPair",
    # PKCE
    "create_pkce_challenge",
    "generate_code_verifier",
    "generate_code_challenge",
    "verify_code_challenge",
    "generate_state",
    "generate_nonce",
    # Core components (December 2025 State of Art)
    "ErrorFactory",
    "JWKSCacheBase",
    "TokenOperations",
    "AuthorizationBuilder",
    "TokenValidator",
    "SyncHTTPExecutor",
    "AsyncHTTPExecutor",
]

__version__ = "1.0.0"
