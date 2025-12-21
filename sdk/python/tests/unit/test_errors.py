"""Unit tests for error classes.

Tests error hierarchy, serialization, and error codes.
"""

import pytest
from hypothesis import given, settings, strategies as st

from auth_platform_sdk.errors import (
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


class TestErrorCode:
    """Tests for ErrorCode enum."""

    def test_error_codes_are_strings(self) -> None:
        """Error codes should be string values."""
        assert ErrorCode.TOKEN_EXPIRED == "AUTH_1001"
        assert ErrorCode.VALIDATION_ERROR == "VAL_2001"
        assert ErrorCode.NETWORK_ERROR == "NET_3001"
        assert ErrorCode.RATE_LIMITED == "RATE_4001"
        assert ErrorCode.SERVER_ERROR == "SRV_5001"
        assert ErrorCode.DPOP_REQUIRED == "DPOP_6001"
        assert ErrorCode.PKCE_REQUIRED == "PKCE_7001"

    def test_error_code_categories(self) -> None:
        """Error codes should follow category pattern."""
        # Authentication errors start with AUTH_1
        assert ErrorCode.TOKEN_EXPIRED.value.startswith("AUTH_1")
        assert ErrorCode.TOKEN_INVALID.value.startswith("AUTH_1")

        # Validation errors start with VAL_2
        assert ErrorCode.VALIDATION_ERROR.value.startswith("VAL_2")
        assert ErrorCode.INVALID_CONFIG.value.startswith("VAL_2")

        # Network errors start with NET_3
        assert ErrorCode.NETWORK_ERROR.value.startswith("NET_3")
        assert ErrorCode.TIMEOUT_ERROR.value.startswith("NET_3")


class TestAuthPlatformError:
    """Tests for base AuthPlatformError."""

    def test_basic_error(self) -> None:
        """Should create error with message and code."""
        error = AuthPlatformError("Test error", ErrorCode.TOKEN_EXPIRED)

        assert str(error) == "Test error"
        assert error.message == "Test error"
        assert error.code == "AUTH_1001"

    def test_error_with_status_code(self) -> None:
        """Should include status code."""
        error = AuthPlatformError(
            "Unauthorized",
            ErrorCode.UNAUTHORIZED,
            status_code=401,
        )

        assert error.status_code == 401

    def test_error_with_correlation_id(self) -> None:
        """Should include correlation ID."""
        error = AuthPlatformError(
            "Error",
            ErrorCode.SERVER_ERROR,
            correlation_id="req-123",
        )

        assert error.correlation_id == "req-123"

    def test_error_with_details(self) -> None:
        """Should include additional details."""
        error = AuthPlatformError(
            "Validation failed",
            ErrorCode.VALIDATION_ERROR,
            details={"field": "email", "reason": "invalid format"},
        )

        assert error.details["field"] == "email"
        assert error.details["reason"] == "invalid format"

    def test_to_dict(self) -> None:
        """Should serialize to dictionary."""
        error = AuthPlatformError(
            "Test error",
            ErrorCode.TOKEN_EXPIRED,
            status_code=401,
            correlation_id="req-123",
            details={"extra": "info"},
        )

        result = error.to_dict()

        assert result["error"] == "Test error"
        assert result["code"] == "AUTH_1001"
        assert result["status_code"] == 401
        assert result["correlation_id"] == "req-123"
        assert result["details"]["extra"] == "info"

    def test_repr(self) -> None:
        """Should have useful repr."""
        error = AuthPlatformError("Test", ErrorCode.TOKEN_EXPIRED)
        repr_str = repr(error)

        assert "AuthPlatformError" in repr_str
        assert "AUTH_1001" in repr_str
        assert "Test" in repr_str


class TestTokenErrors:
    """Tests for token-related errors."""

    def test_token_expired_error(self) -> None:
        """TokenExpiredError should have correct defaults."""
        error = TokenExpiredError()

        assert error.code == "AUTH_1001"
        assert error.status_code == 401
        assert "expired" in error.message.lower()

    def test_token_expired_custom_message(self) -> None:
        """TokenExpiredError should accept custom message."""
        error = TokenExpiredError("Custom expiry message")

        assert error.message == "Custom expiry message"

    def test_token_invalid_error(self) -> None:
        """TokenInvalidError should have correct defaults."""
        error = TokenInvalidError()

        assert error.code == "AUTH_1002"
        assert error.status_code == 401

    def test_token_invalid_with_details(self) -> None:
        """TokenInvalidError should accept details."""
        error = TokenInvalidError(
            "Invalid signature",
            details={"algorithm": "RS256"},
        )

        assert error.details["algorithm"] == "RS256"

    def test_token_refresh_error(self) -> None:
        """TokenRefreshError should have correct defaults."""
        error = TokenRefreshError()

        assert error.code == "AUTH_1003"
        assert error.status_code == 401


class TestValidationError:
    """Tests for ValidationError."""

    def test_validation_error(self) -> None:
        """ValidationError should have correct defaults."""
        error = ValidationError("Invalid input")

        assert error.code == "VAL_2001"
        assert error.status_code == 400

    def test_validation_error_with_details(self) -> None:
        """ValidationError should accept details."""
        error = ValidationError(
            "Field validation failed",
            details={"field": "email", "value": "invalid"},
        )

        assert error.details["field"] == "email"


class TestNetworkErrors:
    """Tests for network-related errors."""

    def test_network_error(self) -> None:
        """NetworkError should have correct defaults."""
        error = NetworkError()

        assert error.code == "NET_3001"
        assert error.status_code is None

    def test_network_error_with_cause(self) -> None:
        """NetworkError should chain cause."""
        cause = ConnectionError("Connection refused")
        error = NetworkError("Failed to connect", cause=cause)

        assert error.__cause__ is cause
        assert "Connection refused" in error.details["cause"]

    def test_timeout_error(self) -> None:
        """TimeoutError should have correct defaults."""
        error = TimeoutError()

        assert error.code == "NET_3002"
        assert error.status_code == 408

    def test_timeout_error_with_duration(self) -> None:
        """TimeoutError should include timeout duration."""
        error = TimeoutError("Request timed out", timeout_seconds=30.0)

        assert error.details["timeout_seconds"] == 30.0


class TestRateLimitError:
    """Tests for RateLimitError."""

    def test_rate_limit_error(self) -> None:
        """RateLimitError should have correct defaults."""
        error = RateLimitError()

        assert error.code == "RATE_4001"
        assert error.status_code == 429

    def test_rate_limit_with_retry_after(self) -> None:
        """RateLimitError should include retry_after."""
        error = RateLimitError(retry_after=60)

        assert error.retry_after == 60
        assert error.details["retry_after"] == 60


class TestConfigError:
    """Tests for InvalidConfigError."""

    def test_invalid_config_error(self) -> None:
        """InvalidConfigError should have correct defaults."""
        error = InvalidConfigError("Missing required field")

        assert error.code == "VAL_2002"
        assert error.status_code is None

    def test_invalid_config_with_field(self) -> None:
        """InvalidConfigError should include field name."""
        error = InvalidConfigError("Invalid value", field="base_url")

        assert error.details["field"] == "base_url"


class TestDPoPError:
    """Tests for DPoPError."""

    def test_dpop_error(self) -> None:
        """DPoPError should have correct defaults."""
        error = DPoPError("Invalid proof")

        assert error.code == "DPOP_6002"
        assert error.status_code == 401

    def test_dpop_error_with_nonce(self) -> None:
        """DPoPError should include nonce."""
        error = DPoPError(
            "Nonce required",
            ErrorCode.DPOP_NONCE_REQUIRED,
            dpop_nonce="server_nonce",
        )

        assert error.dpop_nonce == "server_nonce"
        assert error.details["dpop_nonce"] == "server_nonce"


class TestPKCEError:
    """Tests for PKCEError."""

    def test_pkce_error(self) -> None:
        """PKCEError should have correct defaults."""
        error = PKCEError("Invalid verifier")

        assert error.code == "PKCE_7002"
        assert error.status_code == 400


class TestServerError:
    """Tests for ServerError."""

    def test_server_error(self) -> None:
        """ServerError should have correct defaults."""
        error = ServerError()

        assert error.code == "SRV_5001"
        assert error.status_code == 500

    def test_server_error_custom_status(self) -> None:
        """ServerError should accept custom status code."""
        error = ServerError("Service unavailable", status_code=503)

        assert error.status_code == 503


class TestErrorInheritance:
    """Tests for error inheritance."""

    def test_all_errors_inherit_from_base(self) -> None:
        """All errors should inherit from AuthPlatformError."""
        errors = [
            TokenExpiredError(),
            TokenInvalidError(),
            TokenRefreshError(),
            ValidationError("test"),
            NetworkError(),
            TimeoutError(),
            RateLimitError(),
            InvalidConfigError("test"),
            DPoPError("test"),
            PKCEError("test"),
            ServerError(),
        ]

        for error in errors:
            assert isinstance(error, AuthPlatformError)
            assert isinstance(error, Exception)

    def test_errors_are_catchable(self) -> None:
        """Errors should be catchable by base class."""
        with pytest.raises(AuthPlatformError):
            raise TokenExpiredError()

        with pytest.raises(AuthPlatformError):
            raise NetworkError()
