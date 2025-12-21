"""Error classes for Auth Platform SDK - December 2025 State of Art.

Implements structured error hierarchy with error codes, correlation IDs,
and OpenTelemetry integration for observability.
"""

from __future__ import annotations

from enum import StrEnum
from typing import Any


class ErrorCode(StrEnum):
    """Standardized error codes for Auth Platform SDK."""

    # Authentication errors (1xxx)
    TOKEN_EXPIRED = "AUTH_1001"
    TOKEN_INVALID = "AUTH_1002"
    TOKEN_REFRESH_FAILED = "AUTH_1003"
    INVALID_CREDENTIALS = "AUTH_1004"
    UNAUTHORIZED = "AUTH_1005"

    # Validation errors (2xxx)
    VALIDATION_ERROR = "VAL_2001"
    INVALID_CONFIG = "VAL_2002"
    INVALID_SCOPE = "VAL_2003"
    INVALID_GRANT = "VAL_2004"

    # Network errors (3xxx)
    NETWORK_ERROR = "NET_3001"
    TIMEOUT_ERROR = "NET_3002"
    CONNECTION_ERROR = "NET_3003"

    # Rate limiting (4xxx)
    RATE_LIMITED = "RATE_4001"
    QUOTA_EXCEEDED = "RATE_4002"

    # Server errors (5xxx)
    SERVER_ERROR = "SRV_5001"
    SERVICE_UNAVAILABLE = "SRV_5002"

    # DPoP errors (6xxx)
    DPOP_REQUIRED = "DPOP_6001"
    DPOP_INVALID = "DPOP_6002"
    DPOP_NONCE_REQUIRED = "DPOP_6003"

    # PKCE errors (7xxx)
    PKCE_REQUIRED = "PKCE_7001"
    PKCE_INVALID = "PKCE_7002"


class AuthPlatformError(Exception):
    """Base error for Auth Platform SDK with structured error information."""

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
        """Convert error to dictionary for logging/serialization."""
        return {
            "error": self.message,
            "code": self.code,
            "status_code": self.status_code,
            "correlation_id": self.correlation_id,
            "details": self.details,
        }

    def __repr__(self) -> str:
        return f"{self.__class__.__name__}(code={self.code!r}, message={self.message!r})"


class TokenExpiredError(AuthPlatformError):
    """Access token has expired."""

    def __init__(
        self,
        message: str = "Access token has expired",
        *,
        correlation_id: str | None = None,
    ) -> None:
        super().__init__(
            message,
            ErrorCode.TOKEN_EXPIRED,
            status_code=401,
            correlation_id=correlation_id,
        )


class TokenInvalidError(AuthPlatformError):
    """Token is invalid or malformed."""

    def __init__(
        self,
        message: str = "Token is invalid",
        *,
        correlation_id: str | None = None,
        details: dict[str, Any] | None = None,
    ) -> None:
        super().__init__(
            message,
            ErrorCode.TOKEN_INVALID,
            status_code=401,
            correlation_id=correlation_id,
            details=details,
        )


class TokenRefreshError(AuthPlatformError):
    """Failed to refresh token."""

    def __init__(
        self,
        message: str = "Failed to refresh token",
        *,
        correlation_id: str | None = None,
    ) -> None:
        super().__init__(
            message,
            ErrorCode.TOKEN_REFRESH_FAILED,
            status_code=401,
            correlation_id=correlation_id,
        )


class ValidationError(AuthPlatformError):
    """Token or input validation failed."""

    def __init__(
        self,
        message: str,
        *,
        correlation_id: str | None = None,
        details: dict[str, Any] | None = None,
    ) -> None:
        super().__init__(
            message,
            ErrorCode.VALIDATION_ERROR,
            status_code=400,
            correlation_id=correlation_id,
            details=details,
        )


class NetworkError(AuthPlatformError):
    """Network request failed."""

    def __init__(
        self,
        message: str = "Network request failed",
        *,
        correlation_id: str | None = None,
        cause: Exception | None = None,
    ) -> None:
        super().__init__(
            message,
            ErrorCode.NETWORK_ERROR,
            correlation_id=correlation_id,
            details={"cause": str(cause)} if cause else None,
        )
        self.__cause__ = cause


class TimeoutError(AuthPlatformError):
    """Request timed out."""

    def __init__(
        self,
        message: str = "Request timed out",
        *,
        correlation_id: str | None = None,
        timeout_seconds: float | None = None,
    ) -> None:
        super().__init__(
            message,
            ErrorCode.TIMEOUT_ERROR,
            status_code=408,
            correlation_id=correlation_id,
            details={"timeout_seconds": timeout_seconds} if timeout_seconds else None,
        )


class RateLimitError(AuthPlatformError):
    """Rate limit exceeded."""

    def __init__(
        self,
        message: str = "Rate limit exceeded",
        *,
        retry_after: int | None = None,
        correlation_id: str | None = None,
    ) -> None:
        super().__init__(
            message,
            ErrorCode.RATE_LIMITED,
            status_code=429,
            correlation_id=correlation_id,
            details={"retry_after": retry_after} if retry_after else None,
        )
        self.retry_after = retry_after


class InvalidConfigError(AuthPlatformError):
    """Invalid SDK configuration."""

    def __init__(
        self,
        message: str,
        *,
        field: str | None = None,
    ) -> None:
        super().__init__(
            message,
            ErrorCode.INVALID_CONFIG,
            details={"field": field} if field else None,
        )


class DPoPError(AuthPlatformError):
    """DPoP-related error."""

    def __init__(
        self,
        message: str,
        code: ErrorCode = ErrorCode.DPOP_INVALID,
        *,
        correlation_id: str | None = None,
        dpop_nonce: str | None = None,
    ) -> None:
        super().__init__(
            message,
            code,
            status_code=401,
            correlation_id=correlation_id,
            details={"dpop_nonce": dpop_nonce} if dpop_nonce else None,
        )
        self.dpop_nonce = dpop_nonce


class PKCEError(AuthPlatformError):
    """PKCE-related error."""

    def __init__(
        self,
        message: str,
        code: ErrorCode = ErrorCode.PKCE_INVALID,
        *,
        correlation_id: str | None = None,
    ) -> None:
        super().__init__(
            message,
            code,
            status_code=400,
            correlation_id=correlation_id,
        )


class ServerError(AuthPlatformError):
    """Server-side error."""

    def __init__(
        self,
        message: str = "Server error",
        *,
        status_code: int = 500,
        correlation_id: str | None = None,
    ) -> None:
        super().__init__(
            message,
            ErrorCode.SERVER_ERROR,
            status_code=status_code,
            correlation_id=correlation_id,
        )
