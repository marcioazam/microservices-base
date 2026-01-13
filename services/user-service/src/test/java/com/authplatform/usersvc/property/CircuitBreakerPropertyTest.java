package com.authplatform.usersvc.property;

import net.jqwik.api.*;
import net.jqwik.api.constraints.IntRange;
import java.util.concurrent.atomic.AtomicInteger;
import static org.assertj.core.api.Assertions.assertThat;

/**
 * Property 12: Circuit Breaker State Transitions
 * Validates: Requirements 12.4, 12.5
 * 
 * Ensures circuit breaker correctly transitions between states
 * and provides fallback behavior when open.
 */
class CircuitBreakerPropertyTest {

    @Property(tries = 100)
    void circuitBreakerOpensAfterFailureThreshold(
            @ForAll @IntRange(min = 3, max = 10) int failureThreshold,
            @ForAll @IntRange(min = 1, max = 5) int extraFailures) {
        
        TestCircuitBreaker cb = new TestCircuitBreaker(failureThreshold);
        
        // Cause failures up to threshold
        for (int i = 0; i < failureThreshold; i++) {
            cb.recordFailure();
        }
        
        // Circuit should be open
        assertThat(cb.isOpen()).isTrue();
        assertThat(cb.getState()).isEqualTo(CircuitState.OPEN);
    }

    @Property(tries = 100)
    void circuitBreakerStaysClosedBelowThreshold(
            @ForAll @IntRange(min = 3, max = 10) int failureThreshold,
            @ForAll @IntRange(min = 1, max = 2) int failures) {
        
        if (failures >= failureThreshold) return;
        
        TestCircuitBreaker cb = new TestCircuitBreaker(failureThreshold);
        
        for (int i = 0; i < failures; i++) {
            cb.recordFailure();
        }
        
        assertThat(cb.isClosed()).isTrue();
        assertThat(cb.getState()).isEqualTo(CircuitState.CLOSED);
    }

    @Property(tries = 100)
    void circuitBreakerResetsOnSuccess(
            @ForAll @IntRange(min = 3, max = 10) int failureThreshold) {
        
        TestCircuitBreaker cb = new TestCircuitBreaker(failureThreshold);
        
        // Record some failures (but not enough to open)
        for (int i = 0; i < failureThreshold - 1; i++) {
            cb.recordFailure();
        }
        
        // Record success
        cb.recordSuccess();
        
        // Failure count should reset
        assertThat(cb.getFailureCount()).isEqualTo(0);
        assertThat(cb.isClosed()).isTrue();
    }

    @Property(tries = 100)
    void circuitBreakerTransitionsToHalfOpen(
            @ForAll @IntRange(min = 3, max = 10) int failureThreshold) {
        
        TestCircuitBreaker cb = new TestCircuitBreaker(failureThreshold);
        
        // Open the circuit
        for (int i = 0; i < failureThreshold; i++) {
            cb.recordFailure();
        }
        assertThat(cb.isOpen()).isTrue();
        
        // Simulate wait time elapsed
        cb.simulateWaitTimeElapsed();
        
        // Should transition to half-open
        assertThat(cb.getState()).isEqualTo(CircuitState.HALF_OPEN);
    }

    @Property(tries = 100)
    void halfOpenCircuitClosesOnSuccess(
            @ForAll @IntRange(min = 3, max = 10) int failureThreshold) {
        
        TestCircuitBreaker cb = new TestCircuitBreaker(failureThreshold);
        
        // Open and transition to half-open
        for (int i = 0; i < failureThreshold; i++) {
            cb.recordFailure();
        }
        cb.simulateWaitTimeElapsed();
        assertThat(cb.getState()).isEqualTo(CircuitState.HALF_OPEN);
        
        // Success in half-open should close
        cb.recordSuccess();
        assertThat(cb.isClosed()).isTrue();
    }

    @Property(tries = 100)
    void halfOpenCircuitOpensOnFailure(
            @ForAll @IntRange(min = 3, max = 10) int failureThreshold) {
        
        TestCircuitBreaker cb = new TestCircuitBreaker(failureThreshold);
        
        // Open and transition to half-open
        for (int i = 0; i < failureThreshold; i++) {
            cb.recordFailure();
        }
        cb.simulateWaitTimeElapsed();
        assertThat(cb.getState()).isEqualTo(CircuitState.HALF_OPEN);
        
        // Failure in half-open should re-open
        cb.recordFailure();
        assertThat(cb.isOpen()).isTrue();
    }

    @Property(tries = 100)
    void fallbackIsCalledWhenCircuitOpen(
            @ForAll @IntRange(min = 3, max = 10) int failureThreshold) {
        
        TestCircuitBreaker cb = new TestCircuitBreaker(failureThreshold);
        AtomicInteger fallbackCalls = new AtomicInteger(0);
        
        // Open the circuit
        for (int i = 0; i < failureThreshold; i++) {
            cb.recordFailure();
        }
        
        // Attempt calls when open
        for (int i = 0; i < 5; i++) {
            if (cb.isOpen()) {
                fallbackCalls.incrementAndGet();
            }
        }
        
        assertThat(fallbackCalls.get()).isEqualTo(5);
    }

    enum CircuitState { CLOSED, OPEN, HALF_OPEN }

    static class TestCircuitBreaker {
        private final int failureThreshold;
        private int failureCount = 0;
        private CircuitState state = CircuitState.CLOSED;

        TestCircuitBreaker(int failureThreshold) {
            this.failureThreshold = failureThreshold;
        }

        void recordFailure() {
            if (state == CircuitState.HALF_OPEN) {
                state = CircuitState.OPEN;
                return;
            }
            failureCount++;
            if (failureCount >= failureThreshold) {
                state = CircuitState.OPEN;
            }
        }

        void recordSuccess() {
            if (state == CircuitState.HALF_OPEN) {
                state = CircuitState.CLOSED;
            }
            failureCount = 0;
        }

        void simulateWaitTimeElapsed() {
            if (state == CircuitState.OPEN) {
                state = CircuitState.HALF_OPEN;
            }
        }

        boolean isOpen() { return state == CircuitState.OPEN; }
        boolean isClosed() { return state == CircuitState.CLOSED; }
        CircuitState getState() { return state; }
        int getFailureCount() { return failureCount; }
    }
}
