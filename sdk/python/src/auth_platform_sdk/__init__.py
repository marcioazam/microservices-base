"""Auth Platform Python SDK."""

from .client import AuthPlatformClient, AsyncAuthPlatformClient
from .config import AuthPlatformConfig
from .errors import (
    AuthPlatformError,
    TokenExpiredError,
    TokenRefreshError,
    NetworkError,
    RateLimitError,
    InvalidConfigError,
)
from .jwks import JWKSCache
from .types import TokenData, TokenResponse

__all__ = [
    "AuthPlatformClient",
    "AsyncAuthPlatformClient",
    "AuthPlatformConfig",
    "AuthPlatformError",
    "TokenExpiredError",
    "TokenRefreshError",
    "NetworkError",
    "RateLimitError",
    "InvalidConfigError",
    "JWKSCache",
    "TokenData",
    "TokenResponse",
]

__version__ = "0.1.0"
