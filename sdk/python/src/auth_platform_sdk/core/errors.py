"""Centralized error factory for Auth Platform SDK - December 2025 State of Art.

Provides consistent error creation and transformation across all SDK components.
"""

from __future__ import annotations

import uuid
from typing import TYPE_CHECKING, Any

import httpx

from ..errors import (
    AuthPlatformError,
    DPoPError,
    ErrorCode,
    InvalidConfigError,
    NetworkError,
    RateLimitError,
    ServerError,
    TimeoutError,
    TokenExpiredError,
    TokenInvalidError,
    TokenRefreshError,
    ValidationError,
)

if TYPE_CHECKING:
    pass


class ErrorFactory:
    """Centralized error creation with consistent structure.
    
    All errors created through this factory include:
    - Standardized error codes
    - Optional correlation IDs for tracing
    - Consistent detail structure for logging
    """

    @staticmethod
    def generate_correlation_id() -> str:
        """Generate a unique correlation ID."""
        return str(uuid.uuid4())

    @staticmethod
    def from_http_response(
        response: httpx.Response,
        *,
        correlation_id: str | None = None,
    ) -> AuthPlatformError:
        """Create SDK error from HTTP response.
        
        Args:
            response: HTTP response object.
            correlation_id: Optional correlation ID for tracing.
            
        Returns:
            Appropriate AuthPlatformError subclass.
        """
        status = response.status_code
        correlation_id = correlation_id or ErrorFactory.generate_correlation_id()
        
        # Try to extract error details from response body
        details: dict[str, Any] = {}
        try:
            body = response.json()
            if isinstance(body, dict):
                details["error"] = body.get("error")
                details["error_description"] = body.get("error_description")
        except Exception:
            pass
        
        if status == 400:
            error_type = details.get("error", "")
            if error_type == "invalid_grant":
                return TokenRefreshError(
                    details.get("error_description", "Invalid grant"),
                    correlation_id=correlation_id,
                )
            return ValidationError(
                details.get("error_description", f"Bad request: {status}"),
                correlation_id=correlation_id,
                details=details,
            )
        
        if status == 401:
            error_type = details.get("error", "")
            if error_type == "invalid_token":
                return TokenExpiredError(
                    details.get("error_description", "Token expired"),
                    correlation_id=correlation_id,
                )
            if error_type == "use_dpop_nonce":
                nonce = response.headers.get("DPoP-Nonce")
                return DPoPError(
                    "DPoP nonce required",
                    ErrorCode.DPOP_NONCE_REQUIRED,
                    correlation_id=correlation_id,
                    dpop_nonce=nonce,
                )
            return TokenInvalidError(
                details.get("error_description", "Authentication failed"),
                correlation_id=correlation_id,
                details=details,
            )
        
        if status == 403:
            return ValidationError(
                details.get("error_description", "Access denied"),
                correlation_id=correlation_id,
                details=details,
            )
        
        if status == 429:
            retry_after = response.headers.get("Retry-After")
            return RateLimitError(
                details.get("error_description", "Rate limit exceeded"),
                retry_after=int(retry_after) if retry_after and retry_after.isdigit() else None,
                correlation_id=correlation_id,
            )
        
        if status >= 500:
            return ServerError(
                details.get("error_description", f"Server error: {status}"),
                status_code=status,
                correlation_id=correlation_id,
            )
        
        return ValidationError(
            f"Request failed with status {status}",
            correlation_id=correlation_id,
            details=details,
        )

    @staticmethod
    def from_exception(
        exc: Exception,
        *,
        correlation_id: str | None = None,
    ) -> AuthPlatformError:
        """Create SDK error from exception.
        
        Args:
            exc: Original exception.
            correlation_id: Optional correlation ID for tracing.
            
        Returns:
            Appropriate AuthPlatformError subclass.
        """
        correlation_id = correlation_id or ErrorFactory.generate_correlation_id()
        
        if isinstance(exc, AuthPlatformError):
            # Already an SDK error, just ensure correlation ID
            if exc.correlation_id is None:
                exc.correlation_id = correlation_id
            return exc
        
        if isinstance(exc, httpx.TimeoutException):
            return TimeoutError(
                f"Request timed out: {exc}",
                correlation_id=correlation_id,
            )
        
        if isinstance(exc, httpx.ConnectError):
            return NetworkError(
                f"Connection failed: {exc}",
                correlation_id=correlation_id,
                cause=exc,
            )
        
        if isinstance(exc, httpx.HTTPStatusError):
            return ErrorFactory.from_http_response(
                exc.response,
                correlation_id=correlation_id,
            )
        
        if isinstance(exc, httpx.HTTPError):
            return NetworkError(
                f"HTTP error: {exc}",
                correlation_id=correlation_id,
                cause=exc,
            )
        
        return NetworkError(
            f"Unexpected error: {exc}",
            correlation_id=correlation_id,
            cause=exc,
        )

    @staticmethod
    def token_validation_error(
        message: str,
        *,
        token_metadata: dict[str, Any] | None = None,
        correlation_id: str | None = None,
    ) -> ValidationError:
        """Create token validation error with metadata.
        
        Args:
            message: Error message.
            token_metadata: Optional token metadata (kid, alg, etc.).
            correlation_id: Optional correlation ID.
            
        Returns:
            ValidationError with token details.
        """
        correlation_id = correlation_id or ErrorFactory.generate_correlation_id()
        details = {"token_metadata": token_metadata} if token_metadata else {}
        
        return ValidationError(
            message,
            correlation_id=correlation_id,
            details=details,
        )

    @staticmethod
    def config_error(
        message: str,
        *,
        field: str | None = None,
        value: Any = None,
    ) -> InvalidConfigError:
        """Create configuration error with field details.
        
        Args:
            message: Error message.
            field: Configuration field name.
            value: Invalid value (sanitized).
            
        Returns:
            InvalidConfigError with field details.
        """
        return InvalidConfigError(
            message,
            field=field,
        )
