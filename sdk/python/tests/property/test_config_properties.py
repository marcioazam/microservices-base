"""
Property-based tests for configuration module.

**Feature: python-sdk-modernization, Property 11: Configuration Immutability**
**Feature: python-sdk-modernization, Property 12: Configuration Endpoint Derivation**
**Feature: python-sdk-modernization, Property 13: Configuration Validation**
**Validates: Requirements 11.1, 11.2, 11.3, 11.5**
"""

import pytest
from hypothesis import given, settings, strategies as st
from pydantic import ValidationError as PydanticValidationError

from auth_platform_sdk.config import (
    AuthPlatformConfig,
    CacheConfig,
    DPoPConfig,
    RetryConfig,
    TelemetryConfig,
)


# Strategy for valid base URLs
valid_base_url = st.sampled_from([
    "https://auth.example.com",
    "https://api.auth.io",
    "https://identity.company.org",
    "https://auth.test.local:8443",
])

# Strategy for valid client IDs
valid_client_id = st.text(min_size=1, max_size=100).filter(lambda x: len(x.strip()) > 0)


class TestConfigurationImmutabilityProperties:
    """Property tests for configuration immutability."""

    @given(
        base_url=valid_base_url,
        client_id=valid_client_id,
    )
    @settings(max_examples=100)
    def test_config_is_frozen(
        self,
        base_url: str,
        client_id: str,
    ) -> None:
        """
        Property 11: Configuration Immutability
        For any AuthPlatformConfig instance, attempting to modify any
        attribute SHALL raise an exception (frozen model).
        """
        config = AuthPlatformConfig(
            base_url=base_url,
            client_id=client_id,
        )

        with pytest.raises(PydanticValidationError):
            config.client_id = "new_client_id"

    @given(
        max_retries=st.integers(min_value=0, max_value=10),
        initial_delay=st.floats(min_value=0.1, max_value=60.0),
    )
    @settings(max_examples=100)
    def test_retry_config_is_frozen(
        self,
        max_retries: int,
        initial_delay: float,
    ) -> None:
        """
        Property 11: Configuration Immutability
        RetryConfig SHALL be immutable (frozen model).
        """
        config = RetryConfig(
            max_retries=max_retries,
            initial_delay=initial_delay,
        )

        with pytest.raises(PydanticValidationError):
            config.max_retries = 5

    @given(enabled=st.booleans())
    @settings(max_examples=100)
    def test_telemetry_config_is_frozen(self, enabled: bool) -> None:
        """
        Property 11: Configuration Immutability
        TelemetryConfig SHALL be immutable (frozen model).
        """
        config = TelemetryConfig(enabled=enabled)

        with pytest.raises(PydanticValidationError):
            config.enabled = not enabled

    @given(enabled=st.booleans())
    @settings(max_examples=100)
    def test_dpop_config_is_frozen(self, enabled: bool) -> None:
        """
        Property 11: Configuration Immutability
        DPoPConfig SHALL be immutable (frozen model).
        """
        config = DPoPConfig(enabled=enabled)

        with pytest.raises(PydanticValidationError):
            config.enabled = not enabled

    @given(jwks_ttl=st.integers(min_value=1, max_value=86400))
    @settings(max_examples=100)
    def test_cache_config_is_frozen(self, jwks_ttl: int) -> None:
        """
        Property 11: Configuration Immutability
        CacheConfig SHALL be immutable (frozen model).
        """
        config = CacheConfig(jwks_ttl=jwks_ttl)

        with pytest.raises(PydanticValidationError):
            config.jwks_ttl = 7200


class TestConfigurationEndpointDerivationProperties:
    """Property tests for endpoint derivation."""

    @given(
        base_url=valid_base_url,
        client_id=valid_client_id,
    )
    @settings(max_examples=100)
    def test_token_endpoint_derivation(
        self,
        base_url: str,
        client_id: str,
    ) -> None:
        """
        Property 12: Configuration Endpoint Derivation
        For any base_url, token_endpoint SHALL be {base_url}/oauth/token.
        """
        config = AuthPlatformConfig(
            base_url=base_url,
            client_id=client_id,
        )

        expected = f"{base_url.rstrip('/')}/oauth/token"
        assert config.token_endpoint == expected

    @given(
        base_url=valid_base_url,
        client_id=valid_client_id,
    )
    @settings(max_examples=100)
    def test_authorization_endpoint_derivation(
        self,
        base_url: str,
        client_id: str,
    ) -> None:
        """
        Property 12: Configuration Endpoint Derivation
        For any base_url, authorization_endpoint SHALL be {base_url}/oauth/authorize.
        """
        config = AuthPlatformConfig(
            base_url=base_url,
            client_id=client_id,
        )

        expected = f"{base_url.rstrip('/')}/oauth/authorize"
        assert config.authorization_endpoint == expected

    @given(
        base_url=valid_base_url,
        client_id=valid_client_id,
    )
    @settings(max_examples=100)
    def test_jwks_uri_derivation(
        self,
        base_url: str,
        client_id: str,
    ) -> None:
        """
        Property 12: Configuration Endpoint Derivation
        For any base_url, jwks_uri SHALL be {base_url}/.well-known/jwks.json.
        """
        config = AuthPlatformConfig(
            base_url=base_url,
            client_id=client_id,
        )

        expected = f"{base_url.rstrip('/')}/.well-known/jwks.json"
        assert config.jwks_uri == expected

    @given(
        base_url=valid_base_url,
        client_id=valid_client_id,
    )
    @settings(max_examples=100)
    def test_userinfo_endpoint_derivation(
        self,
        base_url: str,
        client_id: str,
    ) -> None:
        """
        Property 12: Configuration Endpoint Derivation
        For any base_url, userinfo_endpoint SHALL be {base_url}/oauth/userinfo.
        """
        config = AuthPlatformConfig(
            base_url=base_url,
            client_id=client_id,
        )

        expected = f"{base_url.rstrip('/')}/oauth/userinfo"
        assert config.userinfo_endpoint == expected

    @given(
        base_url=valid_base_url,
        client_id=valid_client_id,
    )
    @settings(max_examples=100)
    def test_custom_endpoints_override_defaults(
        self,
        base_url: str,
        client_id: str,
    ) -> None:
        """
        Property 12: Configuration Endpoint Derivation
        Custom endpoints SHALL override derived defaults.
        """
        custom_token = "https://custom.example.com/token"
        config = AuthPlatformConfig(
            base_url=base_url,
            client_id=client_id,
            token_endpoint=custom_token,
        )

        assert config.token_endpoint == custom_token


class TestConfigurationValidationProperties:
    """Property tests for configuration validation."""

    def test_missing_base_url_raises_error(self) -> None:
        """
        Property 13: Configuration Validation
        Missing base_url SHALL raise ValidationError.
        """
        with pytest.raises(PydanticValidationError):
            AuthPlatformConfig(client_id="test-client")

    def test_missing_client_id_raises_error(self) -> None:
        """
        Property 13: Configuration Validation
        Missing client_id SHALL raise ValidationError.
        """
        with pytest.raises(PydanticValidationError):
            AuthPlatformConfig(base_url="https://auth.example.com")

    @given(client_id=st.text(max_size=0))
    @settings(max_examples=10)
    def test_empty_client_id_raises_error(self, client_id: str) -> None:
        """
        Property 13: Configuration Validation
        Empty client_id SHALL raise ValidationError.
        """
        with pytest.raises(PydanticValidationError):
            AuthPlatformConfig(
                base_url="https://auth.example.com",
                client_id=client_id,
            )

    @given(timeout=st.floats(max_value=0))
    @settings(max_examples=50)
    def test_invalid_timeout_raises_error(self, timeout: float) -> None:
        """
        Property 13: Configuration Validation
        Timeout <= 0 SHALL raise ValidationError.
        """
        with pytest.raises(PydanticValidationError):
            AuthPlatformConfig(
                base_url="https://auth.example.com",
                client_id="test-client",
                timeout=timeout,
            )

    @given(max_retries=st.integers(max_value=-1))
    @settings(max_examples=50)
    def test_negative_max_retries_raises_error(self, max_retries: int) -> None:
        """
        Property 13: Configuration Validation
        Negative max_retries SHALL raise ValidationError.
        """
        with pytest.raises(PydanticValidationError):
            RetryConfig(max_retries=max_retries)

    @given(max_retries=st.integers(min_value=11))
    @settings(max_examples=50)
    def test_excessive_max_retries_raises_error(self, max_retries: int) -> None:
        """
        Property 13: Configuration Validation
        max_retries > 10 SHALL raise ValidationError.
        """
        with pytest.raises(PydanticValidationError):
            RetryConfig(max_retries=max_retries)

    @given(
        algorithm=st.text(min_size=1, max_size=10).filter(
            lambda x: x not in {"ES256", "ES384", "ES512", "RS256", "RS384", "RS512"}
        )
    )
    @settings(max_examples=50)
    def test_invalid_dpop_algorithm_raises_error(self, algorithm: str) -> None:
        """
        Property 13: Configuration Validation
        Unsupported DPoP algorithm SHALL raise ValidationError.
        """
        with pytest.raises(PydanticValidationError):
            DPoPConfig(algorithm=algorithm)

    @given(
        base_url=valid_base_url,
        client_id=valid_client_id,
        timeout=st.floats(min_value=0.1, max_value=300.0),
    )
    @settings(max_examples=100)
    def test_valid_config_creates_successfully(
        self,
        base_url: str,
        client_id: str,
        timeout: float,
    ) -> None:
        """
        Property 13: Configuration Validation
        Valid configuration SHALL create successfully.
        """
        config = AuthPlatformConfig(
            base_url=base_url,
            client_id=client_id,
            timeout=timeout,
        )

        assert str(config.base_url).rstrip("/") == base_url.rstrip("/")
        assert config.client_id == client_id
        assert config.timeout == timeout
