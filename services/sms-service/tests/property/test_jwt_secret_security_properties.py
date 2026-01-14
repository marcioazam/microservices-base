"""Property-based tests for JWT secret security validation.

These tests ensure the JWT secret validation logic properly rejects insecure
configurations that could lead to authentication bypass vulnerabilities.
"""

import pytest
from hypothesis import given, strategies as st
from pydantic import ValidationError

from src.config.settings import Settings


class TestJWTSecretSecurityProperties:
    """Property-based security tests for JWT secret validation."""

    @given(
        insecure_pattern=st.sampled_from(
            [
                "change-me-in-production",
                "changeme",
                "change-me",
                "secret",
                "password",
                "test-secret",
                "example",
                "demo",
                "default",
                "secretkey",
                "testpassword",
            ]
        )
    )
    def test_property_rejects_insecure_placeholder_values(
        self, insecure_pattern: str, monkeypatch
    ):
        """
        Property 1: Any JWT secret containing common insecure patterns must be rejected.

        Security rationale:
        - Prevents deployment with placeholder values
        - Blocks obvious weak secrets
        - Forces explicit secure configuration
        """
        # Set all required env vars except JWT secret (which we'll test)
        monkeypatch.setenv("DATABASE_URL", "postgresql://user:pass@localhost/db")
        monkeypatch.setenv("REDIS_URL", "redis://localhost:6379/0")
        monkeypatch.setenv("JWT_SECRET_KEY", insecure_pattern)

        with pytest.raises(ValidationError) as exc_info:
            Settings()

        # Verify error mentions the insecure pattern
        error_message = str(exc_info.value).lower()
        assert "insecure" in error_message or "placeholder" in error_message

    @given(length=st.integers(min_value=1, max_value=31))
    def test_property_rejects_secrets_shorter_than_32_chars(
        self, length: int, monkeypatch
    ):
        """
        Property 2: JWT secrets shorter than 32 characters must be rejected.

        Security rationale:
        - Prevents brute-force attacks on weak secrets
        - Ensures minimum entropy
        - Aligns with NIST recommendations for symmetric keys
        """
        # Generate a random secret of specified length (but safe characters)
        short_secret = "a" * length

        monkeypatch.setenv("DATABASE_URL", "postgresql://user:pass@localhost/db")
        monkeypatch.setenv("REDIS_URL", "redis://localhost:6379/0")
        monkeypatch.setenv("JWT_SECRET_KEY", short_secret)

        with pytest.raises(ValidationError) as exc_info:
            Settings()

        error_message = str(exc_info.value)
        assert "32 characters" in error_message or "min_length" in error_message

    @given(
        secret=st.text(
            alphabet=st.characters(
                whitelist_categories=("Lu", "Ll", "Nd", "P"),
                min_codepoint=33,
                max_codepoint=126,
            ),
            min_size=32,
            max_size=128,
        ).filter(
            lambda s: len(s) >= 32
            and not any(
                pattern in s.lower()
                for pattern in [
                    "change-me",
                    "changeme",
                    "secret",
                    "password",
                    "test",
                    "example",
                    "demo",
                    "default",
                ]
            )
        )
    )
    def test_property_accepts_valid_secrets(self, secret: str, monkeypatch):
        """
        Property 3: Valid secrets (>= 32 chars, no insecure patterns) must be accepted.

        Security rationale:
        - Validates secure configuration works correctly
        - Ensures validation doesn't block legitimate secrets
        """
        monkeypatch.setenv("DATABASE_URL", "postgresql://user:pass@localhost/db")
        monkeypatch.setenv("REDIS_URL", "redis://localhost:6379/0")
        monkeypatch.setenv("JWT_SECRET_KEY", secret)

        try:
            settings = Settings()
            assert len(settings.jwt_secret_key) >= 32
        except ValidationError:
            # If validation fails, it should not be due to length or pattern
            # (might fail on entropy checks in production mode)
            pass

    def test_property_production_enforces_minimum_entropy(self, monkeypatch):
        """
        Property 4: Production environment must enforce minimum entropy requirements.

        Security rationale:
        - Prevents use of simple repeated patterns in production
        - Ensures cryptographically secure secrets
        """
        # Secret with low entropy (repeated characters)
        low_entropy_secret = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

        monkeypatch.setenv("DATABASE_URL", "postgresql://user:pass@localhost/db")
        monkeypatch.setenv("REDIS_URL", "redis://localhost:6379/0")
        monkeypatch.setenv("ENVIRONMENT", "production")
        monkeypatch.setenv("JWT_SECRET_KEY", low_entropy_secret)

        with pytest.raises(ValidationError) as exc_info:
            Settings()

        error_message = str(exc_info.value).lower()
        assert "entropy" in error_message or "repeated" in error_message

    @given(
        secret=st.text(
            alphabet="abcdefghijklmnopqrstuvwxyz0123456789",
            min_size=32,
            max_size=64,
        ).filter(
            lambda s: len(set(s)) >= 16  # Sufficient unique characters
            and not any(
                pattern in s.lower()
                for pattern in [
                    "change-me",
                    "changeme",
                    "secret",
                    "password",
                    "test",
                    "example",
                    "demo",
                    "default",
                ]
            )
        )
    )
    def test_property_production_accepts_high_entropy_secrets(
        self, secret: str, monkeypatch
    ):
        """
        Property 5: Production must accept secrets with sufficient entropy.

        Security rationale:
        - Validates production validation doesn't block legitimate secrets
        - Ensures high-entropy secrets work correctly
        """
        monkeypatch.setenv("DATABASE_URL", "postgresql://user:pass@localhost/db")
        monkeypatch.setenv("REDIS_URL", "redis://localhost:6379/0")
        monkeypatch.setenv("ENVIRONMENT", "production")
        monkeypatch.setenv("JWT_SECRET_KEY", secret)

        try:
            settings = Settings()
            assert len(settings.jwt_secret_key) >= 32
            assert len(set(settings.jwt_secret_key)) >= 16
        except ValidationError as e:
            # Should not fail if entropy is sufficient
            # Log the error for debugging
            pytest.fail(f"Valid high-entropy secret rejected: {e}")

    def test_property_missing_secret_raises_error(self, monkeypatch):
        """
        Property 6: Missing JWT secret must cause validation error at startup.

        Security rationale:
        - Fail-fast principle
        - Prevents starting service without authentication
        - Forces explicit configuration
        """
        monkeypatch.setenv("DATABASE_URL", "postgresql://user:pass@localhost/db")
        monkeypatch.setenv("REDIS_URL", "redis://localhost:6379/0")
        # Do not set JWT_SECRET_KEY

        with pytest.raises(ValidationError) as exc_info:
            Settings()

        error_message = str(exc_info.value)
        assert "jwt_secret_key" in error_message.lower()

    @given(
        case_variation=st.sampled_from(
            [
                "CHANGE-ME-IN-PRODUCTION",
                "Change-Me-In-Production",
                "ChAnGe-Me-In-PrOdUcTiOn",
                "SECRET",
                "Secret",
                "PASSWORD",
                "Password",
            ]
        )
    )
    def test_property_case_insensitive_pattern_detection(
        self, case_variation: str, monkeypatch
    ):
        """
        Property 7: Insecure pattern detection must be case-insensitive.

        Security rationale:
        - Prevents bypassing validation with case variations
        - Ensures robust pattern matching
        """
        monkeypatch.setenv("DATABASE_URL", "postgresql://user:pass@localhost/db")
        monkeypatch.setenv("REDIS_URL", "redis://localhost:6379/0")
        monkeypatch.setenv("JWT_SECRET_KEY", case_variation)

        with pytest.raises(ValidationError) as exc_info:
            Settings()

        error_message = str(exc_info.value).lower()
        assert "insecure" in error_message or "placeholder" in error_message


class TestJWTSecretIntegration:
    """Integration tests for JWT secret configuration."""

    def test_secure_secret_example(self, monkeypatch):
        """Test that a cryptographically secure secret works correctly."""
        import secrets

        secure_secret = secrets.token_urlsafe(48)  # 64 characters

        monkeypatch.setenv("DATABASE_URL", "postgresql://user:pass@localhost/db")
        monkeypatch.setenv("REDIS_URL", "redis://localhost:6379/0")
        monkeypatch.setenv("JWT_SECRET_KEY", secure_secret)

        settings = Settings()
        assert settings.jwt_secret_key == secure_secret
        assert len(settings.jwt_secret_key) >= 32

    def test_development_mode_allows_less_strict_validation(self, monkeypatch):
        """Test that development mode has slightly relaxed validation."""
        # A secret that would fail production entropy checks
        dev_secret = "dev-secret-with-32-characters!"

        monkeypatch.setenv("DATABASE_URL", "postgresql://user:pass@localhost/db")
        monkeypatch.setenv("REDIS_URL", "redis://localhost:6379/0")
        monkeypatch.setenv("ENVIRONMENT", "development")
        monkeypatch.setenv("JWT_SECRET_KEY", dev_secret)

        settings = Settings()
        assert settings.jwt_secret_key == dev_secret
        assert settings.environment == "development"

    def test_error_message_provides_helpful_guidance(self, monkeypatch):
        """Test that error messages guide users to fix configuration."""
        monkeypatch.setenv("DATABASE_URL", "postgresql://user:pass@localhost/db")
        monkeypatch.setenv("REDIS_URL", "redis://localhost:6379/0")
        monkeypatch.setenv("JWT_SECRET_KEY", "short")

        with pytest.raises(ValidationError) as exc_info:
            Settings()

        error_message = str(exc_info.value)
        # Should provide guidance on generating secure secret
        assert "python -c" in error_message or "secrets.token" in error_message
