"""
Property-based tests for client module.

**Feature: python-sdk-modernization, Property 14: Client Context Manager**
**Validates: Requirements 12.3, 12.5**
"""

from unittest.mock import MagicMock, patch

import pytest
from hypothesis import given, settings, strategies as st

from auth_platform_sdk.config import AuthPlatformConfig


# Strategy for valid base URLs
valid_base_url = st.sampled_from([
    "https://auth.example.com",
    "https://api.auth.io",
    "https://identity.company.org",
])

# Strategy for valid client IDs
valid_client_id = st.text(min_size=1, max_size=50).filter(
    lambda x: len(x.strip()) > 0
)


class TestClientContextManagerProperties:
    """Property tests for client context manager."""

    @given(
        base_url=valid_base_url,
        client_id=valid_client_id,
    )
    @settings(max_examples=50)
    def test_sync_client_context_manager_closes_connection(
        self,
        base_url: str,
        client_id: str,
    ) -> None:
        """
        Property 14: Client Context Manager
        For any AuthPlatformClient, using the context manager protocol
        SHALL properly close HTTP connections on exit.
        """
        from auth_platform_sdk.client import AuthPlatformClient

        config = AuthPlatformConfig(
            base_url=base_url,
            client_id=client_id,
        )

        with patch("auth_platform_sdk.client.create_http_client") as mock_create:
            mock_client = MagicMock()
            mock_create.return_value = mock_client

            with AuthPlatformClient(config) as client:
                assert client is not None

            # Verify close was called
            mock_client.close.assert_called_once()

    @given(
        base_url=valid_base_url,
        client_id=valid_client_id,
    )
    @settings(max_examples=50)
    def test_sync_client_closes_on_exception(
        self,
        base_url: str,
        client_id: str,
    ) -> None:
        """
        Property 14: Client Context Manager
        Client SHALL close connections even when exception occurs.
        """
        from auth_platform_sdk.client import AuthPlatformClient

        config = AuthPlatformConfig(
            base_url=base_url,
            client_id=client_id,
        )

        with patch("auth_platform_sdk.client.create_http_client") as mock_create:
            mock_client = MagicMock()
            mock_create.return_value = mock_client

            with pytest.raises(ValueError):
                with AuthPlatformClient(config):
                    raise ValueError("Test exception")

            # Verify close was still called
            mock_client.close.assert_called_once()

    @given(
        base_url=valid_base_url,
        client_id=valid_client_id,
    )
    @settings(max_examples=50)
    def test_sync_client_explicit_close(
        self,
        base_url: str,
        client_id: str,
    ) -> None:
        """
        Property 14: Client Context Manager
        Client SHALL support explicit close() method.
        """
        from auth_platform_sdk.client import AuthPlatformClient

        config = AuthPlatformConfig(
            base_url=base_url,
            client_id=client_id,
        )

        with patch("auth_platform_sdk.client.create_http_client") as mock_create:
            mock_client = MagicMock()
            mock_create.return_value = mock_client

            client = AuthPlatformClient(config)
            client.close()

            mock_client.close.assert_called_once()


class TestAsyncClientContextManagerProperties:
    """Property tests for async client context manager."""

    @given(
        base_url=valid_base_url,
        client_id=valid_client_id,
    )
    @settings(max_examples=50)
    def test_async_client_context_manager_closes_connection(
        self,
        base_url: str,
        client_id: str,
    ) -> None:
        """
        Property 14: Client Context Manager
        For any AsyncAuthPlatformClient, using async context manager
        SHALL properly close HTTP connections on exit.
        """
        import asyncio

        from auth_platform_sdk.async_client import AsyncAuthPlatformClient

        config = AuthPlatformConfig(
            base_url=base_url,
            client_id=client_id,
        )

        async def test_context_manager() -> None:
            with patch(
                "auth_platform_sdk.async_client.create_async_http_client"
            ) as mock_create:
                mock_client = MagicMock()
                mock_client.aclose = MagicMock(return_value=asyncio.Future())
                mock_client.aclose.return_value.set_result(None)
                mock_create.return_value = mock_client

                async with AsyncAuthPlatformClient(config) as client:
                    assert client is not None

                mock_client.aclose.assert_called_once()

        asyncio.get_event_loop().run_until_complete(test_context_manager())

    @given(
        base_url=valid_base_url,
        client_id=valid_client_id,
    )
    @settings(max_examples=50)
    def test_async_client_closes_on_exception(
        self,
        base_url: str,
        client_id: str,
    ) -> None:
        """
        Property 14: Client Context Manager
        Async client SHALL close connections even when exception occurs.
        """
        import asyncio

        from auth_platform_sdk.async_client import AsyncAuthPlatformClient

        config = AuthPlatformConfig(
            base_url=base_url,
            client_id=client_id,
        )

        async def test_exception_handling() -> None:
            with patch(
                "auth_platform_sdk.async_client.create_async_http_client"
            ) as mock_create:
                mock_client = MagicMock()
                mock_client.aclose = MagicMock(return_value=asyncio.Future())
                mock_client.aclose.return_value.set_result(None)
                mock_create.return_value = mock_client

                with pytest.raises(ValueError):
                    async with AsyncAuthPlatformClient(config):
                        raise ValueError("Test exception")

                mock_client.aclose.assert_called_once()

        asyncio.get_event_loop().run_until_complete(test_exception_handling())
