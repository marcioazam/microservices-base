"""Property tests for CircuitBreaker - December 2025 State of Art.

Feature: python-sdk-state-of-art-2025, Property 9: Circuit Breaker State Transitions
Validates: Requirements 2.3
"""

from __future__ import annotations

import time
from unittest.mock import patch

import pytest
from hypothesis import given, settings, strategies as st, assume
from hypothesis.stateful import RuleBasedStateMachine, rule, invariant, initialize

from auth_platform_sdk.http import CircuitBreaker, CircuitState


# Strategies for generating test data
failure_thresholds = st.integers(min_value=1, max_value=20)
recovery_timeouts = st.floats(min_value=1.0, max_value=300.0, allow_nan=False, allow_infinity=False)
half_open_requests = st.integers(min_value=1, max_value=10)


class TestCircuitBreakerStateTransitions:
    """Property tests for circuit breaker state transitions."""

    @given(threshold=failure_thresholds)
    @settings(max_examples=100)
    def test_starts_in_closed_state(self, threshold: int) -> None:
        """Property 9: Circuit breaker starts in CLOSED state.
        
        Feature: python-sdk-state-of-art-2025, Property 9: Circuit Breaker State Transitions
        Validates: Requirements 2.3
        """
        cb = CircuitBreaker(failure_threshold=threshold)
        
        assert cb.state == CircuitState.CLOSED
        assert cb.allow_request() is True

    @given(threshold=failure_thresholds)
    @settings(max_examples=100)
    def test_opens_after_threshold_failures(self, threshold: int) -> None:
        """Property 9: Circuit opens after threshold consecutive failures.
        
        Feature: python-sdk-state-of-art-2025, Property 9: Circuit Breaker State Transitions
        Validates: Requirements 2.3
        """
        cb = CircuitBreaker(failure_threshold=threshold)
        
        # Record failures up to threshold
        for _ in range(threshold):
            cb.record_failure()
        
        assert cb.state == CircuitState.OPEN
        assert cb.allow_request() is False

    @given(threshold=failure_thresholds)
    @settings(max_examples=100)
    def test_stays_closed_below_threshold(self, threshold: int) -> None:
        """Property 9: Circuit stays closed below failure threshold.
        
        Feature: python-sdk-state-of-art-2025, Property 9: Circuit Breaker State Transitions
        Validates: Requirements 2.3
        """
        assume(threshold > 1)
        cb = CircuitBreaker(failure_threshold=threshold)
        
        # Record failures below threshold
        for _ in range(threshold - 1):
            cb.record_failure()
        
        assert cb.state == CircuitState.CLOSED
        assert cb.allow_request() is True

    @given(
        threshold=failure_thresholds,
        recovery_timeout=st.floats(min_value=1.0, max_value=60.0, allow_nan=False, allow_infinity=False),
    )
    @settings(max_examples=100)
    def test_transitions_to_half_open_after_timeout(
        self,
        threshold: int,
        recovery_timeout: float,
    ) -> None:
        """Property 9: Circuit transitions to HALF_OPEN after recovery timeout.
        
        Feature: python-sdk-state-of-art-2025, Property 9: Circuit Breaker State Transitions
        Validates: Requirements 2.3
        """
        cb = CircuitBreaker(
            failure_threshold=threshold,
            recovery_timeout=recovery_timeout,
        )
        
        # Open the circuit
        for _ in range(threshold):
            cb.record_failure()
        
        assert cb.state == CircuitState.OPEN
        
        # Simulate time passing beyond recovery timeout
        with patch.object(time, "time", return_value=cb._last_failure_time + recovery_timeout + 1):
            assert cb.state == CircuitState.HALF_OPEN
            assert cb.allow_request() is True

    @given(
        threshold=failure_thresholds,
        half_open_reqs=half_open_requests,
    )
    @settings(max_examples=100)
    def test_closes_after_half_open_successes(
        self,
        threshold: int,
        half_open_reqs: int,
    ) -> None:
        """Property 9: Circuit closes after successful requests in HALF_OPEN.
        
        Feature: python-sdk-state-of-art-2025, Property 9: Circuit Breaker State Transitions
        Validates: Requirements 2.3
        """
        cb = CircuitBreaker(
            failure_threshold=threshold,
            recovery_timeout=1.0,
            half_open_requests=half_open_reqs,
        )
        
        # Open the circuit
        for _ in range(threshold):
            cb.record_failure()
        
        # Transition to half-open
        with patch.object(time, "time", return_value=cb._last_failure_time + 2.0):
            assert cb.state == CircuitState.HALF_OPEN
            
            # Record successes
            for _ in range(half_open_reqs):
                cb.record_success()
            
            assert cb.state == CircuitState.CLOSED

    @given(threshold=failure_thresholds)
    @settings(max_examples=100)
    def test_reopens_on_half_open_failure(self, threshold: int) -> None:
        """Property 9: Circuit reopens on failure in HALF_OPEN state.
        
        Feature: python-sdk-state-of-art-2025, Property 9: Circuit Breaker State Transitions
        Validates: Requirements 2.3
        """
        cb = CircuitBreaker(
            failure_threshold=threshold,
            recovery_timeout=1.0,
        )
        
        # Open the circuit
        for _ in range(threshold):
            cb.record_failure()
        
        # Transition to half-open
        with patch.object(time, "time", return_value=cb._last_failure_time + 2.0):
            assert cb.state == CircuitState.HALF_OPEN
            
            # Record failure in half-open
            cb.record_failure()
            
            assert cb.state == CircuitState.OPEN

    @given(threshold=failure_thresholds)
    @settings(max_examples=100)
    def test_success_resets_failure_count_in_closed(self, threshold: int) -> None:
        """Property 9: Success resets failure count in CLOSED state.
        
        Feature: python-sdk-state-of-art-2025, Property 9: Circuit Breaker State Transitions
        Validates: Requirements 2.3
        """
        assume(threshold > 1)
        cb = CircuitBreaker(failure_threshold=threshold)
        
        # Record some failures
        for _ in range(threshold - 1):
            cb.record_failure()
        
        # Record success
        cb.record_success()
        
        # Should reset count, so threshold-1 more failures needed
        for _ in range(threshold - 1):
            cb.record_failure()
        
        assert cb.state == CircuitState.CLOSED


class CircuitBreakerStateMachine(RuleBasedStateMachine):
    """Stateful property test for circuit breaker using Hypothesis."""

    def __init__(self) -> None:
        super().__init__()
        self.cb: CircuitBreaker | None = None
        self.expected_state = CircuitState.CLOSED
        self.failure_count = 0
        self.half_open_successes = 0
        self.mock_time = 0.0

    @initialize(
        threshold=failure_thresholds,
        recovery=st.floats(min_value=1.0, max_value=10.0, allow_nan=False, allow_infinity=False),
        half_open=half_open_requests,
    )
    def init_circuit_breaker(self, threshold: int, recovery: float, half_open: int) -> None:
        """Initialize circuit breaker with random parameters."""
        self.cb = CircuitBreaker(
            failure_threshold=threshold,
            recovery_timeout=recovery,
            half_open_requests=half_open,
        )
        self.expected_state = CircuitState.CLOSED
        self.failure_count = 0
        self.half_open_successes = 0
        self.mock_time = time.time()

    @rule()
    def record_success(self) -> None:
        """Record a successful request."""
        if self.cb is None:
            return
        
        with patch.object(time, "time", return_value=self.mock_time):
            self.cb.record_success()
            
            if self.expected_state == CircuitState.HALF_OPEN:
                self.half_open_successes += 1
                if self.half_open_successes >= self.cb.half_open_requests:
                    self.expected_state = CircuitState.CLOSED
                    self.failure_count = 0
            elif self.expected_state == CircuitState.CLOSED:
                self.failure_count = 0

    @rule()
    def record_failure(self) -> None:
        """Record a failed request."""
        if self.cb is None:
            return
        
        with patch.object(time, "time", return_value=self.mock_time):
            self.cb.record_failure()
            self.failure_count += 1
            
            if self.expected_state == CircuitState.HALF_OPEN:
                self.expected_state = CircuitState.OPEN
            elif self.failure_count >= self.cb.failure_threshold:
                self.expected_state = CircuitState.OPEN

    @rule()
    def advance_time_small(self) -> None:
        """Advance time by a small amount (not enough for recovery)."""
        if self.cb is None:
            return
        self.mock_time += 0.1

    @rule()
    def advance_time_large(self) -> None:
        """Advance time past recovery timeout."""
        if self.cb is None:
            return
        self.mock_time += self.cb.recovery_timeout + 1
        
        if self.expected_state == CircuitState.OPEN:
            self.expected_state = CircuitState.HALF_OPEN
            self.half_open_successes = 0

    @invariant()
    def state_matches_expected(self) -> None:
        """Invariant: Circuit breaker state matches expected state."""
        if self.cb is None:
            return
        
        with patch.object(time, "time", return_value=self.mock_time):
            actual_state = self.cb.state
            assert actual_state == self.expected_state, (
                f"Expected {self.expected_state}, got {actual_state}"
            )

    @invariant()
    def allow_request_consistent_with_state(self) -> None:
        """Invariant: allow_request is consistent with state."""
        if self.cb is None:
            return
        
        with patch.object(time, "time", return_value=self.mock_time):
            state = self.cb.state
            allowed = self.cb.allow_request()
            
            if state == CircuitState.OPEN:
                assert allowed is False
            else:
                assert allowed is True


TestCircuitBreakerStateful = CircuitBreakerStateMachine.TestCase
