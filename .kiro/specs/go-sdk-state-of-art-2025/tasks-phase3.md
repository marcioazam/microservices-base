# Phase 3: Resilience and Caching

## Task 3.1: Implement Retry Logic with Generics
**Requirement:** REQ-5 (Retry and Resilience Consolidation)
**Acceptance Criteria:** 5.1, 5.2, 5.3, 5.4, 5.5, 5.6

Implement retry logic in `src/retry/`:

**Files to create:**
- `src/retry/retry.go` - Generic retry implementation
- `src/retry/policy.go` - Retry policy configuration
- `tests/retry/retry_test.go` - Unit tests
- `tests/retry/retry_prop_test.go` - Property tests

**Property Tests (Properties 10-13):**
```go
// Property 10: Retry Delay Exponential Backoff
func TestProperty_RetryDelayExponentialBackoff(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        attempt := rapid.IntRange(0, 10).Draw(t, "attempt")
        baseDelay := time.Duration(rapid.IntRange(100, 1000).Draw(t, "baseMs")) * time.Millisecond
        maxDelay := baseDelay * 100
        jitter := 0.1
        
        policy := &Policy{BaseDelay: baseDelay, MaxDelay: maxDelay, Jitter: jitter}
        delay := policy.CalculateDelay(attempt)
        
        expectedBase := baseDelay * time.Duration(1<<attempt)
        if expectedBase > maxDelay {
            expectedBase = maxDelay
        }
        
        // Delay should be within jitter range
        minDelay := time.Duration(float64(expectedBase) * (1 - jitter))
        maxDelayWithJitter := time.Duration(float64(expectedBase) * (1 + jitter))
        
        assert.GreaterOrEqual(t, delay, minDelay)
        assert.LessOrEqual(t, delay, maxDelayWithJitter)
    })
}

// Property 11: Retry-After Header Parsing
func TestProperty_RetryAfterParsing(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        seconds := rapid.IntRange(1, 3600).Draw(t, "seconds")
        header := strconv.Itoa(seconds)
        
        duration, ok := ParseRetryAfter(header)
        assert.True(t, ok)
        assert.Equal(t, time.Duration(seconds)*time.Second, duration)
    })
}

// Property 12: Retry Success Behavior
func TestProperty_RetrySuccessBehavior(t *rapid.T) { ... }

// Property 13: Retry Exhaustion Behavior
func TestProperty_RetryExhaustionBehavior(t *rapid.T) { ... }
```

**Deliverables:**
- [ ] Implement `Policy` struct with configurable parameters
- [ ] Implement `CalculateDelay()` with exponential backoff and jitter
- [ ] Implement `ParseRetryAfter()` for header parsing
- [ ] Implement generic `Retry[T]()` function
- [ ] Implement `RetryWithResponse()` for HTTP operations
- [ ] Implement context cancellation support
- [ ] Property tests for Properties 10-13 (100+ iterations each)
- [ ] Unit tests for edge cases (zero retries, immediate success)

---

## Task 3.2: Implement JWKS Cache
**Requirement:** REQ-6 (JWKS Cache Optimization)
**Acceptance Criteria:** 6.1, 6.2, 6.3, 6.4, 6.5, 6.6

Implement JWKS caching in `src/token/`:

**Files to create:**
- `src/token/jwks.go` - JWKS cache implementation
- `src/token/validator.go` - Token validation with JWKS
- `tests/token/jwks_test.go` - Unit tests
- `tests/token/jwks_prop_test.go` - Property tests

**Property Tests (Property 14):**
```go
// Property 14: JWKS Cache Metrics Consistency
func TestProperty_JWKSCacheMetricsConsistency(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        cache := NewJWKSCache("https://example.com/.well-known/jwks.json", time.Hour)
        
        numOps := rapid.IntRange(1, 50).Draw(t, "numOps")
        for i := 0; i < numOps; i++ {
            // Simulate cache operations
            cache.lookup(ctx)
        }
        
        metrics := cache.GetMetrics()
        assert.Equal(t, int64(numOps), metrics.Hits+metrics.Misses)
    })
}
```

**Deliverables:**
- [ ] Implement `JWKSCache` with configurable TTL (1min-24h)
- [ ] Implement automatic background refresh
- [ ] Implement fallback keys for refresh failures
- [ ] Implement `JWKSMetrics` for cache statistics
- [ ] Implement `ValidateToken()` and `ValidateTokenWithOpts()`
- [ ] Implement `AddJWKSEndpoint()` for multiple endpoints
- [ ] Property test for Property 14 (100+ iterations)
- [ ] Unit tests for cache expiry and refresh scenarios
