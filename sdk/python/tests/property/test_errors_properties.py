"""
Property-based tests for error module.

**Feature: python-sdk-modernization, Property 1: Error Serialization Round-Trip**
**Feature: python-sdk-modernization, Property 15: Error Hierarchy Inheritance**
**Validates: Requirements 5.1, 5.2, 5.5**
"""

from typing import Any

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


# Strategy for generating valid error codes
error_code_strategy = st.sampled_from(list(ErrorCode))

# Strategy for generating optional strings
optional_string = st.one_of(st.none(), st.text(min_size=1, max_size=100))

# Strategy for generating optional integers
optional_int = st.one_of(st.none(), st.integers(min_value=100, max_value=599))

# Strategy for generating details dict
details_strategy = st.one_of(
    st.none(),
    st.dictionaries(
        keys=st.text(min_size=1, max_size=20).filter(str.isidentifier),
        values=st.one_of(
            st.text(max_size=50),
            st.integers(),
            st.booleans(),
        ),
        max_size=5,
    ),
)


class TestErrorSerializationProperties:
    """Property tests for error serialization."""

    @given(
        message=st.text(min_size=1, max_size=200),
        code=error_code_strategy,
        status_code=optional_int,
        correlation_id=optional_string,
        details=details_strategy,
    )
    @settings(max_examples=100)
    def test_error_serialization_round_trip(
        self,
        message: str,
        code: ErrorCode,
        status_code: int | None,
        correlation_id: str | None,
        details: dict[str, Any] | None,
    ) -> None:
        """
        Property 1: Error Serialization Round-Trip
        For any AuthPlatformError instance with any combination of message,
        code, status_code, correlation_id, and details, calling to_dict()
        SHALL produce a dictionary that contains all the original values.
        """
        error = AuthPlatformError(
            message,
            code,
            status_code=status_code,
            correlation_id=correlation_id,
            details=details,
        )

        result = error.to_dict()

        assert result["error"] == message
        assert result["code"] == code.value
        assert result["status_code"] == status_code
        assert result["correlation_id"] == correlation_id
        assert result["details"] == (details or {})

    @given(
        message=st.text(min_size=1, max_size=200),
        code=st.text(min_size=1, max_size=20),
    )
    @settings(max_examples=100)
    def test_error_accepts_string_code(
        self,
        message: str,
        code: str,
    ) -> None:
        """
        Property 1: Error Serialization Round-Trip
        Error SHALL accept string codes and preserve them in serialization.
        """
        error = AuthPlatformError(message, code)
        result = error.to_dict()

        assert result["code"] == code

    @given(
        message=st.text(min_size=1, max_size=200),
        code=error_code_strategy,
    )
    @settings(max_examples=100)
    def test_error_message_preserved(
        self,
        message: str,
        code: ErrorCode,
    ) -> None:
        """
        Property 1: Error Serialization Round-Trip
        Error message SHALL be preserved in str() and to_dict().
        """
        error = AuthPlatformError(message, code)

        assert str(error) == message
        assert error.message == message
        assert error.to_dict()["error"] == message


class TestErrorHierarchyProperties:
    """Property tests for error hierarchy."""

    def test_all_error_classes_inherit_from_base(self) -> None:
        """
        Property 15: Error Hierarchy Inheritance
        For any error class in the SDK error module, it SHALL be a
        subclass of AuthPlatformError.
        """
        error_classes = [
            TokenExpiredError,
            TokenInvalidError,
            TokenRefreshError,
            ValidationError,
            NetworkError,
            TimeoutError,
            RateLimitError,
            InvalidConfigError,
            DPoPError,
            PKCEError,
            ServerError,
        ]

        for error_class in error_classes:
            assert issubclass(error_class, AuthPlatformError), (
                f"{error_class.__name__} must inherit from AuthPlatformError"
            )
            assert issubclass(error_class, Exception), (
                f"{error_class.__name__} must inherit from Exception"
            )

    @given(correlation_id=optional_string)
    @settings(max_examples=100)
    def test_token_expired_error_properties(
        self,
        correlation_id: str | None,
    ) -> None:
        """
        Property 15: Error Hierarchy Inheritance
        TokenExpiredError SHALL have correct code and status_code.
        """
        error = TokenExpiredError(correlation_id=correlation_id)

        assert isinstance(error, AuthPlatformError)
        assert error.code == ErrorCode.TOKEN_EXPIRED.value
        assert error.status_code == 401
        assert error.correlation_id == correlation_id

    @given(
        message=st.text(min_size=1, max_size=100),
        correlation_id=optional_string,
        details=details_strategy,
    )
    @settings(max_examples=100)
    def test_token_invalid_error_properties(
        self,
        message: str,
        correlation_id: str | None,
        details: dict[str, Any] | None,
    ) -> None:
        """
        Property 15: Error Hierarchy Inheritance
        TokenInvalidError SHALL preserve all provided attributes.
        """
        error = TokenInvalidError(
            message,
            correlation_id=correlation_id,
            details=details,
        )

        assert isinstance(error, AuthPlatformError)
        assert error.message == message
        assert error.code == ErrorCode.TOKEN_INVALID.value
        assert error.correlation_id == correlation_id
        assert error.details == (details or {})

    @given(
        message=st.text(min_size=1, max_size=100),
        correlation_id=optional_string,
    )
    @settings(max_examples=100)
    def test_validation_error_properties(
        self,
        message: str,
        correlation_id: str | None,
    ) -> None:
        """
        Property 15: Error Hierarchy Inheritance
        ValidationError SHALL have correct code and status_code.
        """
        error = ValidationError(message, correlation_id=correlation_id)

        assert isinstance(error, AuthPlatformError)
        assert error.code == ErrorCode.VALIDATION_ERROR.value
        assert error.status_code == 400

    @given(retry_after=st.one_of(st.none(), st.integers(min_value=1, max_value=3600)))
    @settings(max_examples=100)
    def test_rate_limit_error_properties(
        self,
        retry_after: int | None,
    ) -> None:
        """
        Property 15: Error Hierarchy Inheritance
        RateLimitError SHALL preserve retry_after attribute.
        """
        error = RateLimitError(retry_after=retry_after)

        assert isinstance(error, AuthPlatformError)
        assert error.code == ErrorCode.RATE_LIMITED.value
        assert error.status_code == 429
        assert error.retry_after == retry_after

    @given(
        message=st.text(min_size=1, max_size=100),
        dpop_nonce=optional_string,
    )
    @settings(max_examples=100)
    def test_dpop_error_properties(
        self,
        message: str,
        dpop_nonce: str | None,
    ) -> None:
        """
        Property 15: Error Hierarchy Inheritance
        DPoPError SHALL preserve dpop_nonce attribute.
        """
        error = DPoPError(message, dpop_nonce=dpop_nonce)

        assert isinstance(error, AuthPlatformError)
        assert error.dpop_nonce == dpop_nonce
        if dpop_nonce:
            assert error.details["dpop_nonce"] == dpop_nonce

    @given(status_code=st.integers(min_value=500, max_value=599))
    @settings(max_examples=100)
    def test_server_error_properties(
        self,
        status_code: int,
    ) -> None:
        """
        Property 15: Error Hierarchy Inheritance
        ServerError SHALL accept custom status codes in 5xx range.
        """
        error = ServerError(status_code=status_code)

        assert isinstance(error, AuthPlatformError)
        assert error.code == ErrorCode.SERVER_ERROR.value
        assert error.status_code == status_code
