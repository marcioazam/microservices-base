"""Property-based tests for HTTP executors - December 2025 State of Art.

Property 8: Retry Exponential Backoff
- Delay increases exponentially with each attempt
- Delay never exceeds max_delay
- Jitter is within configured bounds
"""

from __future__ import annotations

import pytest
from hypothesis import given, settings, assume
from hypothesis import strategies as st

from auth_platform_sdk.config import RetryConfig
from auth_platform_sdk.core.http_executor import calculate_retry_delay


# Strategies for generating test data
initial_delay_strategy = st.floats(min_value=0.1, max_value=10.0)
max_delay_strategy = st.floats(min_value=10.0, max_value=300.0)
exponential_base_strategy = st.floats(min_value=1.5, max_value=3.0)
jitter_strategy = st.floats(min_value=0.0, max_value=1.0)
attempt_strategy = st.integers(min_value=0, max_value=10)


class TestRetryExponentialBackoff:
    """Property tests for retry exponential backoff."""

    @given(
        initial_delay=initial_delay_strategy,
        max_delay=max_delay_strategy,
        exponential_base=exponential_base_strategy,
        attempt=attempt_strategy,
    )
    @settings(max_examples=100)
    def test_delay_never_exceeds_max(
        self,
        initial_delay: float,
        max_delay: float,
        exponential_base: float,
        attempt: int,
    ) -> None:
        """Property: Delay never exceeds max_delay."""
        assume(max_delay > initial_delay)
        
        config = RetryConfig(
            max_retries=10,
            initial_delay=initial_delay,
            max_delay=max_delay,
            exponential_base=exponential_base,
            jitter=0.0,  # No jitter for this test
        )
        
        delay = calculate_retry_delay(config, attempt)
        
        assert delay <= max_delay

    @given(
        initial_delay=initial_delay_strategy,
        max_delay=max_delay_strategy,
        exponential_base=exponential_base_strategy,
    )
    @settings(max_examples=100)
    def test_delay_increases_with_attempts(
        self,
        initial_delay: float,
        max_delay: float,
        exponential_base: float,
    ) -> None:
        """Property: Delay increases with each attempt until max."""
        assume(max_delay > initial_delay * exponential_base)
        
        config = RetryConfig(
            max_retries=10,
            initial_delay=initial_delay,
            max_delay=max_delay,
            exponential_base=exponential_base,
            jitter=0.0,  # No jitter for deterministic comparison
        )
        
        delays = [calculate_retry_delay(config, i) for i in range(5)]
        
        # Each delay should be >= previous (until max is reached)
        for i in range(1, len(delays)):
            assert delays[i] >= delays[i - 1] or delays[i] == max_delay

    @given(
        initial_delay=initial_delay_strategy,
        max_delay=max_delay_strategy,
        exponential_base=exponential_base_strategy,
    )
    @settings(max_examples=100)
    def test_first_attempt_uses_initial_delay(
        self,
        initial_delay: float,
        max_delay: float,
        exponential_base: float,
    ) -> None:
        """Property: First attempt (0) uses initial_delay."""
        assume(max_delay > initial_delay)
        
        config = RetryConfig(
            max_retries=10,
            initial_delay=initial_delay,
            max_delay=max_delay,
            exponential_base=exponential_base,
            jitter=0.0,
        )
        
        delay = calculate_retry_delay(config, 0)
        
        assert delay == initial_delay

    @given(
        initial_delay=initial_delay_strategy,
        max_delay=max_delay_strategy,
        exponential_base=exponential_base_strategy,
        jitter=jitter_strategy,
        attempt=attempt_strategy,
    )
    @settings(max_examples=100)
    def test_jitter_within_bounds(
        self,
        initial_delay: float,
        max_delay: float,
        exponential_base: float,
        jitter: float,
        attempt: int,
    ) -> None:
        """Property: Jitter keeps delay within expected bounds."""
        assume(max_delay > initial_delay)
        
        config = RetryConfig(
            max_retries=10,
            initial_delay=initial_delay,
            max_delay=max_delay,
            exponential_base=exponential_base,
            jitter=jitter,
        )
        
        # Calculate base delay without jitter
        base_delay = min(
            initial_delay * (exponential_base ** attempt),
            max_delay,
        )
        jitter_range = base_delay * jitter
        
        # Sample multiple times to check jitter bounds
        for _ in range(10):
            delay = calculate_retry_delay(config, attempt)
            
            # Delay should be within jitter range of base
            assert delay >= base_delay - jitter_range - 0.001  # Small epsilon
            assert delay <= base_delay + jitter_range + 0.001

    @given(
        initial_delay=initial_delay_strategy,
        max_delay=max_delay_strategy,
        exponential_base=exponential_base_strategy,
    )
    @settings(max_examples=100)
    def test_exponential_growth_formula(
        self,
        initial_delay: float,
        max_delay: float,
        exponential_base: float,
    ) -> None:
        """Property: Delay follows exponential growth formula."""
        assume(max_delay > initial_delay * (exponential_base ** 3))
        
        config = RetryConfig(
            max_retries=10,
            initial_delay=initial_delay,
            max_delay=max_delay,
            exponential_base=exponential_base,
            jitter=0.0,
        )
        
        for attempt in range(4):
            delay = calculate_retry_delay(config, attempt)
            expected = initial_delay * (exponential_base ** attempt)
            
            assert abs(delay - expected) < 0.001


class TestRetryConfigValidation:
    """Property tests for retry configuration validation."""

    @given(
        max_retries=st.integers(min_value=0, max_value=10),
        initial_delay=st.floats(min_value=0.1, max_value=60.0),
        max_delay=st.floats(min_value=0.1, max_value=300.0),
        exponential_base=st.floats(min_value=1.5, max_value=3.0),
        jitter=st.floats(min_value=0.0, max_value=1.0),
    )
    @settings(max_examples=100)
    def test_valid_config_creates_successfully(
        self,
        max_retries: int,
        initial_delay: float,
        max_delay: float,
        exponential_base: float,
        jitter: float,
    ) -> None:
        """Property: Valid configuration values create config successfully."""
        config = RetryConfig(
            max_retries=max_retries,
            initial_delay=initial_delay,
            max_delay=max_delay,
            exponential_base=exponential_base,
            jitter=jitter,
        )
        
        assert config.max_retries == max_retries
        assert config.initial_delay == initial_delay
        assert config.max_delay == max_delay
        assert config.exponential_base == exponential_base
        assert config.jitter == jitter

    def test_invalid_max_retries_raises_error(self) -> None:
        """Property: Invalid max_retries raises validation error."""
        with pytest.raises(ValueError):
            RetryConfig(max_retries=-1)
        
        with pytest.raises(ValueError):
            RetryConfig(max_retries=11)

    def test_invalid_initial_delay_raises_error(self) -> None:
        """Property: Invalid initial_delay raises validation error."""
        with pytest.raises(ValueError):
            RetryConfig(initial_delay=0)
        
        with pytest.raises(ValueError):
            RetryConfig(initial_delay=-1)

    def test_invalid_exponential_base_raises_error(self) -> None:
        """Property: Invalid exponential_base raises validation error."""
        with pytest.raises(ValueError):
            RetryConfig(exponential_base=1.0)
        
        with pytest.raises(ValueError):
            RetryConfig(exponential_base=4.0)


class TestDelayDistribution:
    """Property tests for delay distribution characteristics."""

    @given(
        initial_delay=initial_delay_strategy,
        max_delay=max_delay_strategy,
        exponential_base=exponential_base_strategy,
        jitter=st.floats(min_value=0.05, max_value=0.5),
        attempt=attempt_strategy,
    )
    @settings(max_examples=50)
    def test_jitter_produces_variation(
        self,
        initial_delay: float,
        max_delay: float,
        exponential_base: float,
        jitter: float,
        attempt: int,
    ) -> None:
        """Property: Non-zero jitter produces variation in delays."""
        assume(max_delay > initial_delay)
        
        config = RetryConfig(
            max_retries=10,
            initial_delay=initial_delay,
            max_delay=max_delay,
            exponential_base=exponential_base,
            jitter=jitter,
        )
        
        # Sample multiple delays
        delays = [calculate_retry_delay(config, attempt) for _ in range(20)]
        
        # With jitter, we should see some variation
        unique_delays = set(delays)
        
        # At least some variation expected (may not be all unique due to float precision)
        assert len(unique_delays) > 1 or jitter < 0.01

    @given(
        initial_delay=initial_delay_strategy,
        max_delay=max_delay_strategy,
        exponential_base=exponential_base_strategy,
        attempt=attempt_strategy,
    )
    @settings(max_examples=100)
    def test_zero_jitter_deterministic(
        self,
        initial_delay: float,
        max_delay: float,
        exponential_base: float,
        attempt: int,
    ) -> None:
        """Property: Zero jitter produces deterministic delays."""
        assume(max_delay > initial_delay)
        
        config = RetryConfig(
            max_retries=10,
            initial_delay=initial_delay,
            max_delay=max_delay,
            exponential_base=exponential_base,
            jitter=0.0,
        )
        
        delays = [calculate_retry_delay(config, attempt) for _ in range(10)]
        
        # All delays should be identical
        assert len(set(delays)) == 1
