"""PKCE (Proof Key for Code Exchange) implementation - December 2025 State of Art.

Implements RFC 7636 with S256 challenge method for secure
authorization code flow in public clients.
"""

from __future__ import annotations

import base64
import hashlib
import secrets
from typing import TYPE_CHECKING

from .models import PKCEChallenge

if TYPE_CHECKING:
    pass


def generate_code_verifier(length: int = 64) -> str:
    """Generate a cryptographically random code verifier.

    Args:
        length: Length of the verifier (43-128 characters per RFC 7636).

    Returns:
        URL-safe base64-encoded random string.

    Raises:
        ValueError: If length is outside valid range.
    """
    if not 43 <= length <= 128:
        msg = "Code verifier length must be between 43 and 128 characters"
        raise ValueError(msg)

    # Generate random bytes and encode as URL-safe base64
    # We need more bytes than the final length due to base64 encoding
    num_bytes = (length * 3) // 4 + 1
    random_bytes = secrets.token_bytes(num_bytes)

    # URL-safe base64 without padding
    verifier = base64.urlsafe_b64encode(random_bytes).decode("ascii")
    verifier = verifier.rstrip("=")

    return verifier[:length]


def generate_code_challenge(code_verifier: str) -> str:
    """Generate S256 code challenge from code verifier.

    Args:
        code_verifier: The code verifier string.

    Returns:
        Base64url-encoded SHA-256 hash of the verifier.
    """
    # SHA-256 hash of the verifier
    digest = hashlib.sha256(code_verifier.encode("ascii")).digest()

    # Base64url encode without padding
    challenge = base64.urlsafe_b64encode(digest).decode("ascii")
    return challenge.rstrip("=")


def create_pkce_challenge(verifier_length: int = 64) -> PKCEChallenge:
    """Create a complete PKCE challenge with verifier.

    Args:
        verifier_length: Length of the code verifier.

    Returns:
        PKCEChallenge containing verifier, challenge, and method.
    """
    code_verifier = generate_code_verifier(verifier_length)
    code_challenge = generate_code_challenge(code_verifier)

    return PKCEChallenge(
        code_verifier=code_verifier,
        code_challenge=code_challenge,
        code_challenge_method="S256",
    )


def verify_code_challenge(code_verifier: str, code_challenge: str) -> bool:
    """Verify that a code verifier matches a code challenge.

    Args:
        code_verifier: The original code verifier.
        code_challenge: The code challenge to verify against.

    Returns:
        True if the verifier produces the challenge, False otherwise.
    """
    expected_challenge = generate_code_challenge(code_verifier)
    return secrets.compare_digest(expected_challenge, code_challenge)


def generate_state(length: int = 32) -> str:
    """Generate a cryptographically random state parameter.

    Args:
        length: Length of the state string.

    Returns:
        URL-safe random string for CSRF protection.
    """
    return secrets.token_urlsafe(length)


def generate_nonce(length: int = 32) -> str:
    """Generate a cryptographically random nonce for OpenID Connect.

    Args:
        length: Length of the nonce string.

    Returns:
        URL-safe random string for replay protection.
    """
    return secrets.token_urlsafe(length)
