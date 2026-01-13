"""Property-based tests for AuthorizationBuilder - December 2025 State of Art.

Property 11: Authorization URL Construction
- All required OAuth 2.0 parameters are present
- PKCE parameters included when enabled
- State and nonce are cryptographically random
- URL encoding is correct
"""

from __future__ import annotations

from urllib.parse import parse_qs, urlparse

import pytest
from hypothesis import given, settings, assume, HealthCheck
from hypothesis import strategies as st

from auth_platform_sdk.config import AuthPlatformConfig
from auth_platform_sdk.core.auth_builder import AuthorizationBuilder


# Strategies for generating test data
client_id_strategy = st.text(
    alphabet=st.characters(whitelist_categories=("L", "N"), whitelist_characters="-_"),
    min_size=1,
    max_size=64,
).filter(lambda x: len(x.strip()) > 0)

redirect_uri_strategy = st.sampled_from([
    "https://example.com/callback",
    "https://app.example.org/oauth/callback",
    "http://localhost:8080/callback",
    "https://myapp.io/auth/redirect",
])

scope_strategy = st.lists(
    st.sampled_from(["openid", "profile", "email", "read", "write", "admin"]),
    min_size=0,
    max_size=5,
    unique=True,
)


def create_test_config(client_id: str) -> AuthPlatformConfig:
    """Create a test configuration."""
    return AuthPlatformConfig(
        client_id=client_id,
        base_url="https://auth.example.com",
    )


class TestAuthorizationURLConstruction:
    """Property tests for authorization URL construction."""

    @given(
        client_id=client_id_strategy,
        redirect_uri=redirect_uri_strategy,
        scopes=scope_strategy,
        use_pkce=st.booleans(),
    )
    @settings(max_examples=100, suppress_health_check=[HealthCheck.too_slow])
    def test_required_parameters_always_present(
        self,
        client_id: str,
        redirect_uri: str,
        scopes: list[str],
        use_pkce: bool,
    ) -> None:
        """Property: All required OAuth 2.0 parameters are always present."""
        config = create_test_config(client_id)
        builder = AuthorizationBuilder(config)

        url, state, pkce = builder.build_authorization_url(
            redirect_uri,
            scopes=scopes if scopes else None,
            use_pkce=use_pkce,
        )

        parsed = urlparse(url)
        params = parse_qs(parsed.query)

        # Required parameters must be present
        assert "response_type" in params
        assert params["response_type"][0] == "code"
        assert "client_id" in params
        assert params["client_id"][0] == client_id
        assert "redirect_uri" in params
        assert params["redirect_uri"][0] == redirect_uri
        assert "state" in params
        assert "nonce" in params

    @given(
        client_id=client_id_strategy,
        redirect_uri=redirect_uri_strategy,
    )
    @settings(max_examples=100)
    def test_pkce_parameters_when_enabled(
        self,
        client_id: str,
        redirect_uri: str,
    ) -> None:
        """Property: PKCE parameters are present when PKCE is enabled."""
        config = create_test_config(client_id)
        builder = AuthorizationBuilder(config)

        url, state, pkce = builder.build_authorization_url(
            redirect_uri,
            use_pkce=True,
        )

        parsed = urlparse(url)
        params = parse_qs(parsed.query)

        # PKCE parameters must be present
        assert "code_challenge" in params
        assert "code_challenge_method" in params
        assert params["code_challenge_method"][0] == "S256"
        assert pkce is not None
        assert pkce.code_challenge == params["code_challenge"][0]

    @given(
        client_id=client_id_strategy,
        redirect_uri=redirect_uri_strategy,
    )
    @settings(max_examples=100)
    def test_no_pkce_parameters_when_disabled(
        self,
        client_id: str,
        redirect_uri: str,
    ) -> None:
        """Property: PKCE parameters are absent when PKCE is disabled."""
        config = create_test_config(client_id)
        builder = AuthorizationBuilder(config)

        url, state, pkce = builder.build_authorization_url(
            redirect_uri,
            use_pkce=False,
        )

        parsed = urlparse(url)
        params = parse_qs(parsed.query)

        # PKCE parameters must be absent
        assert "code_challenge" not in params
        assert "code_challenge_method" not in params
        assert pkce is None

    @given(
        client_id=client_id_strategy,
        redirect_uri=redirect_uri_strategy,
    )
    @settings(max_examples=100)
    def test_state_uniqueness(
        self,
        client_id: str,
        redirect_uri: str,
    ) -> None:
        """Property: Generated states are unique across calls."""
        config = create_test_config(client_id)
        builder = AuthorizationBuilder(config)

        states = set()
        for _ in range(10):
            _, state, _ = builder.build_authorization_url(redirect_uri)
            states.add(state)

        # All states should be unique
        assert len(states) == 10

    @given(
        client_id=client_id_strategy,
        redirect_uri=redirect_uri_strategy,
        scopes=scope_strategy,
    )
    @settings(max_examples=100)
    def test_scope_encoding(
        self,
        client_id: str,
        redirect_uri: str,
        scopes: list[str],
    ) -> None:
        """Property: Scopes are correctly space-separated in URL."""
        assume(len(scopes) > 0)

        config = create_test_config(client_id)
        builder = AuthorizationBuilder(config)

        url, _, _ = builder.build_authorization_url(
            redirect_uri,
            scopes=scopes,
        )

        parsed = urlparse(url)
        params = parse_qs(parsed.query)

        assert "scope" in params
        url_scopes = set(params["scope"][0].split())
        assert url_scopes == set(scopes)

    @given(
        client_id=client_id_strategy,
        redirect_uri=redirect_uri_strategy,
        custom_state=st.text(min_size=16, max_size=64),
        custom_nonce=st.text(min_size=16, max_size=64),
    )
    @settings(max_examples=100)
    def test_custom_state_and_nonce_preserved(
        self,
        client_id: str,
        redirect_uri: str,
        custom_state: str,
        custom_nonce: str,
    ) -> None:
        """Property: Custom state and nonce are preserved in URL."""
        config = create_test_config(client_id)
        builder = AuthorizationBuilder(config)

        url, returned_state, _ = builder.build_authorization_url(
            redirect_uri,
            state=custom_state,
            nonce=custom_nonce,
        )

        parsed = urlparse(url)
        params = parse_qs(parsed.query)

        assert returned_state == custom_state
        assert params["state"][0] == custom_state
        assert params["nonce"][0] == custom_nonce

    @given(
        client_id=client_id_strategy,
        redirect_uri=redirect_uri_strategy,
    )
    @settings(max_examples=100)
    def test_url_base_matches_config(
        self,
        client_id: str,
        redirect_uri: str,
    ) -> None:
        """Property: URL base matches authorization endpoint from config."""
        config = create_test_config(client_id)
        builder = AuthorizationBuilder(config)

        url, _, _ = builder.build_authorization_url(redirect_uri)

        parsed = urlparse(url)
        expected_endpoint = urlparse(config.authorization_endpoint)

        assert parsed.scheme == expected_endpoint.scheme
        assert parsed.netloc == expected_endpoint.netloc
        assert parsed.path == expected_endpoint.path


class TestCallbackParsing:
    """Property tests for callback URL parsing."""

    @given(
        client_id=client_id_strategy,
        redirect_uri=redirect_uri_strategy,
        auth_code=st.text(min_size=16, max_size=128, alphabet="abcdefghijklmnopqrstuvwxyz0123456789"),
    )
    @settings(max_examples=100)
    def test_valid_callback_extracts_code(
        self,
        client_id: str,
        redirect_uri: str,
        auth_code: str,
    ) -> None:
        """Property: Valid callback URL correctly extracts authorization code."""
        config = create_test_config(client_id)
        builder = AuthorizationBuilder(config)

        # Generate authorization URL to get state
        _, state, _ = builder.build_authorization_url(redirect_uri)

        # Simulate callback URL
        callback_url = f"{redirect_uri}?code={auth_code}&state={state}"

        code, error = builder.parse_callback_url(callback_url, state)

        assert code == auth_code
        assert error is None

    @given(
        client_id=client_id_strategy,
        redirect_uri=redirect_uri_strategy,
        auth_code=st.text(min_size=16, max_size=128, alphabet="abcdefghijklmnopqrstuvwxyz0123456789"),
    )
    @settings(max_examples=100)
    def test_state_mismatch_raises_error(
        self,
        client_id: str,
        redirect_uri: str,
        auth_code: str,
    ) -> None:
        """Property: State mismatch raises ValueError."""
        config = create_test_config(client_id)
        builder = AuthorizationBuilder(config)

        # Generate authorization URL to get state
        _, state, _ = builder.build_authorization_url(redirect_uri)

        # Simulate callback URL with wrong state
        wrong_state = "wrong_state_value"
        callback_url = f"{redirect_uri}?code={auth_code}&state={wrong_state}"

        with pytest.raises(ValueError, match="State mismatch"):
            builder.parse_callback_url(callback_url, state)

    @given(
        client_id=client_id_strategy,
        redirect_uri=redirect_uri_strategy,
        error_code=st.sampled_from([
            "access_denied",
            "invalid_request",
            "unauthorized_client",
            "server_error",
        ]),
    )
    @settings(max_examples=100)
    def test_error_response_raises_error(
        self,
        client_id: str,
        redirect_uri: str,
        error_code: str,
    ) -> None:
        """Property: Error response in callback raises ValueError."""
        config = create_test_config(client_id)
        builder = AuthorizationBuilder(config)

        # Generate authorization URL to get state
        _, state, _ = builder.build_authorization_url(redirect_uri)

        # Simulate error callback URL
        callback_url = f"{redirect_uri}?error={error_code}&state={state}"

        with pytest.raises(ValueError, match="Authorization error"):
            builder.parse_callback_url(callback_url, state)
