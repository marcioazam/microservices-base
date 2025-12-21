"""
Property-based tests for HTTP module.

**Feature: python-sdk-modernization, Property 2: Retry Exponential Backoff**
**Validates: Requirements 2.3**
"""

from hypothesis import given, settings, strategies as st

from auth_platform_sdk.config import RetryConfig


class TestRetryExponentialBackoffProperties:
    """Property tests for retry exponential backoff."""

    @given(
        max_retries=st.integers(min_value=0, max_value=10),
        initial_delay=st.floats(min_value=0.1, max_value=60.0),
        max_delay=st.floats(min_value=1.0, max_value=300.0),
        exponential_base=st.floats(min_value=1.5, max_value=3.0),
        jitter=st.floats(min_value=0.0, max_value=1.0),
    )
    @settings(max_examples=100)
    def test_delay_at_attempt_zero_near_initial(
        self,
        max_retries: int,
        initial_delay: float,
        max_delay: float,
        exponential_base: float,
        jitter: float,
    ) -> None:
        """
        Property 2: Retry Exponential Backoff
        For attempt 0, delay SHALL be approximately initial_delay (± jitter).
        """
        if initial_delay > max_delay:
            return  # Skip invalid combinations

        config = RetryConfig(
            max_retries=max_retries,
            initial_delay=initial_delay,
            max_delay=max_delay,
            exponential_base=exponential_base,
            jitter=jitter,
        )

        delay = config.get_delay(0)

        # Delay should be initial_delay ± jitter range
        jitter_range = initial_delay * jitter
        min_expected = initial_delay - jitter_range
        max_expected = initial_delay + jitter_range

        assert min_expected <= delay <= max_expected, (
            f"Delay {delay} not in range [{min_expected}, {max_expected}]"
        )

    @given(
        initial_delay=st.floats(min_value=0.1, max_value=10.0),
        max_delay=st.floats(min_value=10.0, max_value=300.0),
        exponential_base=st.floats(min_value=1.5, max_value=3.0),
        attempt=st.integers(min_value=0, max_value=10),
    )
    @settings(max_examples=100)
    def test_delay_never_exceeds_max(
        self,
        initial_delay: float,
        max_delay: float,
        exponential_base: float,
        attempt: int,
    ) -> None:
        """
        Property 2: Retry Exponential Backoff
        For any attempt, delay SHALL be <= max_delay + jitter.
        """
        jitter = 0.1  # Fixed jitter for this test

        config = RetryConfig(
            initial_delay=initial_delay,
            max_delay=max_delay,
            exponential_base=exponential_base,
            jitter=jitter,
        )

        delay = config.get_delay(attempt)

        # Max possible delay is max_delay + jitter range
        max_possible = max_delay + (max_delay * jitter)

        assert delay <= max_possible, (
            f"Delay {delay} exceeds max possible {max_possible}"
        )

    @given(
        initial_delay=st.floats(min_value=0.1, max_value=5.0),
        max_delay=st.floats(min_value=50.0, max_value=300.0),
        exponential_base=st.floats(min_value=1.5, max_value=3.0),
    )
    @settings(max_examples=100)
    def test_delay_increases_exponentially(
        self,
        initial_delay: float,
        max_delay: float,
        exponential_base: float,
    ) -> None:
        """
        Property 2: Retry Exponential Backoff
        Delay SHALL follow exponential growth pattern until max_delay.
        """
        config = RetryConfig(
            initial_delay=initial_delay,
            max_delay=max_delay,
            exponential_base=exponential_base,
            jitter=0.0,  # No jitter for deterministic test
        )

        # Calculate expected delays
        for attempt in range(5):
            delay = config.get_delay(attempt)
            expected = min(initial_delay * (exponential_base ** attempt), max_delay)

            assert delay == expected, (
                f"Attempt {attempt}: delay {delay} != expected {expected}"
            )

    @given(
        initial_delay=st.floats(min_value=0.1, max_value=10.0),
        max_delay=st.floats(min_value=10.0, max_value=300.0),
        jitter=st.floats(min_value=0.0, max_value=1.0),
    )
    @settings(max_examples=100)
    def test_delay_always_positive(
        self,
        initial_delay: float,
        max_delay: float,
        jitter: float,
    ) -> None:
        """
        Property 2: Retry Exponential Backoff
        Delay SHALL always be positive.
        """
        config = RetryConfig(
            initial_delay=initial_delay,
            max_delay=max_delay,
            jitter=jitter,
        )

        for attempt in range(10):
            delay = config.get_delay(attempt)
            # Even with negative jitter, delay should be positive
            # (jitter can subtract up to jitter% of delay)
            min_possible = initial_delay * (1 - jitter)
            assert delay >= min_possible * 0.9, f"Delay {delay} too small"

    @given(
        initial_delay=st.floats(min_value=0.1, max_value=10.0),
        max_delay=st.floats(min_value=10.0, max_value=300.0),
    )
    @settings(max_examples=100)
    def test_jitter_adds_randomness(
        self,
        initial_delay: float,
        max_delay: float,
    ) -> None:
        """
        Property 2: Retry Exponential Backoff
        With jitter > 0, multiple calls SHALL produce different delays.
        """
        config = RetryConfig(
            initial_delay=initial_delay,
            max_delay=max_delay,
            jitter=0.5,  # 50% jitter
        )

        # Get multiple delays for same attempt
        delays = [config.get_delay(0) for _ in range(10)]

        # With 50% jitter, we should see variation
        # (statistically very unlikely to get 10 identical values)
        unique_delays = set(delays)
        assert len(unique_delays) > 1, "Jitter should produce variation"

    @given(
        initial_delay=st.floats(min_value=0.1, max_value=10.0),
        max_delay=st.floats(min_value=10.0, max_value=300.0),
    )
    @settings(max_examples=100)
    def test_zero_jitter_is_deterministic(
        self,
        initial_delay: float,
        max_delay: float,
    ) -> None:
        """
        Property 2: Retry Exponential Backoff
        With jitter = 0, delays SHALL be deterministic.
        """
        config = RetryConfig(
            initial_delay=initial_delay,
            max_delay=max_delay,
            jitter=0.0,
        )

        # Get multiple delays for same attempt
        delays = [config.get_delay(0) for _ in range(10)]

        # All should be identical
        assert all(d == delays[0] for d in delays), "Zero jitter should be deterministic"
