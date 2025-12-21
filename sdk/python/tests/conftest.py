"""
Shared test fixtures for Auth Platform SDK tests.

Provides common fixtures for HTTP mocking, configuration,
and test data generation.
"""

import pytest
from unittest.mock import MagicMock

from auth_platform_sdk.config import (
    AuthPlatformConfig,
    CacheConfig,
    DPoPConfig,
    RetryConfig,
    TelemetryConfig,
)


@pytest.fixture
def base_config() -> AuthPlatformConfig:
    """Provide a basic SDK configuration for testing."""
    return AuthPlatformConfig(
        base_url="https://auth.example.com",
        client_id="test-client-id",
        client_secret="test-client-secret",
    )


@pytest.fixture
def config_with_dpop() -> AuthPlatformConfig:
    """Provide SDK configuration with DPoP enabled."""
    return AuthPlatformConfig(
        base_url="https://auth.example.com",
        client_id="test-client-id",
        client_secret="test-client-secret",
        dpop=DPoPConfig(enabled=True, algorithm="ES256"),
    )


@pytest.fixture
def retry_config() -> RetryConfig:
    """Provide retry configuration for testing."""
    return RetryConfig(
        max_retries=3,
        initial_delay=0.1,
        max_delay=1.0,
        exponential_base=2.0,
        jitter=0.0,
    )


@pytest.fixture
def cache_config() -> CacheConfig:
    """Provide cache configuration for testing."""
    return CacheConfig(
        jwks_ttl=3600,
        jwks_refresh_ahead=300,
        token_buffer=60,
    )


@pytest.fixture
def telemetry_config() -> TelemetryConfig:
    """Provide telemetry configuration for testing."""
    return TelemetryConfig(
        enabled=False,
        service_name="test-sdk",
        trace_requests=False,
        trace_token_operations=False,
    )


@pytest.fixture
def mock_http_client() -> MagicMock:
    """Provide a mock HTTP client."""
    client = MagicMock()
    client.request = MagicMock()
    client.close = MagicMock()
    return client


@pytest.fixture
def mock_async_http_client() -> MagicMock:
    """Provide a mock async HTTP client."""
    import asyncio

    client = MagicMock()
    client.request = MagicMock(return_value=asyncio.Future())
    client.aclose = MagicMock(return_value=asyncio.Future())
    client.aclose.return_value.set_result(None)
    return client


@pytest.fixture
def sample_token_response() -> dict:
    """Provide a sample OAuth token response."""
    return {
        "access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.test",
        "token_type": "Bearer",
        "expires_in": 3600,
        "refresh_token": "refresh_token_value",
        "scope": "openid profile email",
    }


@pytest.fixture
def sample_jwks() -> dict:
    """Provide a sample JWKS response."""
    return {
        "keys": [
            {
                "kty": "RSA",
                "kid": "test-key-1",
                "use": "sig",
                "alg": "RS256",
                "n": "0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbISD08qNLyrdkt-bFTWhAI4vMQFh6WeZu0fM4lFd2NcRwr3XPksINHaQ-G_xBniIqbw0Ls1jF44-csFCur-kEgU8awapJzKnqDKgw",
                "e": "AQAB",
            }
        ]
    }


@pytest.fixture
def sample_token_claims() -> dict:
    """Provide sample JWT claims."""
    import time

    now = int(time.time())
    return {
        "sub": "user-123",
        "iss": "https://auth.example.com",
        "aud": "test-client-id",
        "exp": now + 3600,
        "iat": now,
        "scope": "openid profile email",
    }
