"""Property tests for ErrorFactory - December 2025 State of Art.

Feature: python-sdk-state-of-art-2025, Property 6: Error Handling Consistency
Validates: Requirements 4.1, 4.2, 4.3, 4.4, 4.5
"""

from __future__ import annotations

from typing import Any
from unittest.mock import MagicMock

import httpx
import pytest
from hypothesis import given, settings, strategies as st

from auth_platform_sdk.core.errors import ErrorFactory
from auth_platform_sdk.errors import (
    AuthPlatformError,
    NetworkError,
    RateLimitError,
    ServerError,
    TimeoutError,
    TokenInvalidError,
    ValidationError,
)


# Strategies for generating test data
http_status_codes = st.sampled_from([400, 401, 403, 404, 429, 500, 502, 503, 504])
error_messages = st.text(min_size=1, max_size=100, alphabet=st.characters(whitelist_categories=("L", "N", "P", "S")))
correlation_ids = st.uuids().map(str)


def create_mock_response(status_code: int, body: dict[str, Any] | None = None) -> MagicMock:
    """Create a mock HTTP response."""
    response = MagicMock(spec=httpx.Response)
    response.status_code = status_code
    response.headers = {}
    if body:
        response.json.return_value = body
    else:
        response.json.side_effect = ValueError("No JSON")
    return response


class TestErrorFactoryProperties:
    """Property tests for ErrorFactory."""

    @given(status_code=http_status_codes, correlation_id=correlation_ids)
    @settings(max_examples=100)
    def test_from_http_response_always_returns_auth_platform_error(
        self,
        status_code: int,
        correlation_id: str,
    ) -> None:
        """Property 6: For any HTTP status code, ErrorFactory produces AuthPlatformError.
        
        Feature: python-sdk-state-of-art-2025, Property 6: Error Handling Consistency
        Validates: Requirements 4.1, 4.2
        """
        response = create_mock_response(status_code)
        
        error = ErrorFactory.from_http_response(response, correlation_id=correlation_id)
        
        assert isinstance(error, AuthPlatformError)
        assert error.correlation_id == correlation_id
        assert error.code is not None
        assert error.message is not None

    @given(status_code=st.integers(min_value=500, max_value=599))
    @settings(max_examples=100)
    def test_server_errors_produce_server_error_type(self, status_code: int) -> None:
        """Property 6: Server errors (5xx) produce ServerError type.
        
        Feature: python-sdk-state-of-art-2025, Property 6: Error Handling Consistency
        Validates: Requirements 4.2
        """
        response = create_mock_response(status_code)
        
        error = ErrorFactory.from_http_response(response)
        
        assert isinstance(error, ServerError)
        assert error.status_code == status_code

    @given(retry_after=st.integers(min_value=1, max_value=3600))
    @settings(max_examples=100)
    def test_rate_limit_includes_retry_after(self, retry_after: int) -> None:
        """Property 6: Rate limit errors include retry_after when present.
        
        Feature: python-sdk-state-of-art-2025, Property 6: Error Handling Consistency
        Validates: Requirements 4.2
        """
        response = create_mock_response(429)
        response.headers = {"Retry-After": str(retry_after)}
        
        error = ErrorFactory.from_http_response(response)
        
        assert isinstance(error, RateLimitError)
        assert error.retry_after == retry_after

    @given(correlation_id=correlation_ids)
    @settings(max_examples=100)
    def test_correlation_id_always_present(self, correlation_id: str) -> None:
        """Property 6: Correlation IDs are always included when provided.
        
        Feature: python-sdk-state-of-art-2025, Property 6: Error Handling Consistency
        Validates: Requirements 4.3
        """
        response = create_mock_response(400)
        
        error = ErrorFactory.from_http_response(response, correlation_id=correlation_id)
        
        assert error.correlation_id == correlation_id

    @given(status_code=http_status_codes)
    @settings(max_examples=100)
    def test_error_serialization_produces_valid_dict(self, status_code: int) -> None:
        """Property 6: Error serialization produces valid dict with required fields.
        
        Feature: python-sdk-state-of-art-2025, Property 6: Error Handling Consistency
        Validates: Requirements 4.4
        """
        response = create_mock_response(status_code)
        
        error = ErrorFactory.from_http_response(response)
        error_dict = error.to_dict()
        
        assert isinstance(error_dict, dict)
        assert "error" in error_dict
        assert "code" in error_dict
        assert "correlation_id" in error_dict
        assert "details" in error_dict

    @given(
        kid=st.text(min_size=1, max_size=50, alphabet=st.characters(whitelist_categories=("L", "N"))),
        alg=st.sampled_from(["ES256", "ES384", "RS256", "RS384", "RS512"]),
    )
    @settings(max_examples=100)
    def test_token_validation_error_includes_metadata(self, kid: str, alg: str) -> None:
        """Property 6: Token validation errors include token metadata.
        
        Feature: python-sdk-state-of-art-2025, Property 6: Error Handling Consistency
        Validates: Requirements 4.5
        """
        metadata = {"kid": kid, "alg": alg}
        
        error = ErrorFactory.token_validation_error(
            "Validation failed",
            token_metadata=metadata,
        )
        
        assert isinstance(error, ValidationError)
        assert error.details.get("token_metadata") == metadata


class TestErrorFactoryExceptionHandling:
    """Property tests for exception transformation."""

    @given(message=error_messages)
    @settings(max_examples=100)
    def test_timeout_exception_produces_timeout_error(self, message: str) -> None:
        """Property 6: Timeout exceptions produce TimeoutError.
        
        Feature: python-sdk-state-of-art-2025, Property 6: Error Handling Consistency
        Validates: Requirements 4.2
        """
        exc = httpx.TimeoutException(message)
        
        error = ErrorFactory.from_exception(exc)
        
        assert isinstance(error, TimeoutError)
        assert message in error.message

    @given(message=error_messages)
    @settings(max_examples=100)
    def test_connect_error_produces_network_error(self, message: str) -> None:
        """Property 6: Connection errors produce NetworkError.
        
        Feature: python-sdk-state-of-art-2025, Property 6: Error Handling Consistency
        Validates: Requirements 4.2
        """
        exc = httpx.ConnectError(message)
        
        error = ErrorFactory.from_exception(exc)
        
        assert isinstance(error, NetworkError)
        assert message in error.message

    @given(correlation_id=correlation_ids)
    @settings(max_examples=100)
    def test_exception_transformation_preserves_correlation_id(
        self,
        correlation_id: str,
    ) -> None:
        """Property 6: Exception transformation preserves correlation ID.
        
        Feature: python-sdk-state-of-art-2025, Property 6: Error Handling Consistency
        Validates: Requirements 4.3
        """
        exc = Exception("Test error")
        
        error = ErrorFactory.from_exception(exc, correlation_id=correlation_id)
        
        assert error.correlation_id == correlation_id

    def test_auth_platform_error_passthrough(self) -> None:
        """Property 6: AuthPlatformError passes through unchanged.
        
        Feature: python-sdk-state-of-art-2025, Property 6: Error Handling Consistency
        Validates: Requirements 4.1
        """
        original = ValidationError("Test", correlation_id="test-id")
        
        result = ErrorFactory.from_exception(original)
        
        assert result is original
