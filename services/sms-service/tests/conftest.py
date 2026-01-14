"""Pytest configuration and fixtures."""

import asyncio
from collections.abc import AsyncGenerator, Generator
from typing import Any

import pytest
from hypothesis import settings

settings.register_profile("ci", max_examples=100)
settings.register_profile("dev", max_examples=50)
settings.load_profile("dev")


@pytest.fixture(scope="session")
def event_loop() -> Generator[asyncio.AbstractEventLoop, None, None]:
    """Create event loop for async tests."""
    loop = asyncio.new_event_loop()
    yield loop
    loop.close()


@pytest.fixture
def sample_phone_e164() -> str:
    """Valid E.164 phone number."""
    return "+5511999998888"


@pytest.fixture
def sample_message() -> str:
    """Sample SMS message."""
    return "Your verification code is 123456"


@pytest.fixture
def sample_idempotency_key() -> str:
    """Sample idempotency key."""
    return "idem-key-12345678-abcd-1234-efgh-123456789012"
