"""Error classes for Auth Platform SDK."""


class AuthPlatformError(Exception):
    """Base error for Auth Platform SDK."""

    def __init__(self, message: str, code: str, status_code: int | None = None):
        super().__init__(message)
        self.code = code
        self.status_code = status_code


class TokenExpiredError(AuthPlatformError):
    """Access token has expired."""

    def __init__(self, message: str = "Access token has expired"):
        super().__init__(message, "TOKEN_EXPIRED", 401)


class TokenRefreshError(AuthPlatformError):
    """Failed to refresh token."""

    def __init__(self, message: str = "Failed to refresh token"):
        super().__init__(message, "TOKEN_REFRESH_FAILED", 401)


class NetworkError(AuthPlatformError):
    """Network request failed."""

    def __init__(self, message: str = "Network request failed"):
        super().__init__(message, "NETWORK_ERROR", None)


class RateLimitError(AuthPlatformError):
    """Rate limit exceeded."""

    def __init__(self, message: str = "Rate limit exceeded", retry_after: int | None = None):
        super().__init__(message, "RATE_LIMITED", 429)
        self.retry_after = retry_after


class InvalidConfigError(AuthPlatformError):
    """Invalid configuration."""

    def __init__(self, message: str):
        super().__init__(message, "INVALID_CONFIG", None)


class ValidationError(AuthPlatformError):
    """Token validation failed."""

    def __init__(self, message: str):
        super().__init__(message, "VALIDATION_ERROR", 401)
