# Design: Correctness Properties for Testing

*A property is a characteristic or behavior that should hold true across all valid executions of a system—essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*

## Error Handling Properties (1-4)

### Property 1: Error Structure Completeness
*For any* SDKError instance, it SHALL have a non-empty Code and a non-empty Message.
**Validates: Requirements 2.3**

### Property 2: Error Helper Functions Correctness
*For any* SDKError with a specific ErrorCode, the corresponding Is* helper function SHALL return true, and all other Is* functions SHALL return false.
**Validates: Requirements 2.4**

### Property 3: Error Message Sanitization
*For any* error message containing sensitive patterns (tokens, secrets, passwords, JWT prefixes), the sanitized version SHALL NOT contain those patterns.
**Validates: Requirements 2.5**

### Property 4: Error Chain Preservation
*For any* error wrapped with WrapError, calling errors.Unwrap SHALL return the original cause, and errors.Is SHALL correctly identify the wrapped error.
**Validates: Requirements 2.6**

## Result/Option Properties (5-6)

### Property 5: Result/Option Functor Laws
*For any* Result[T] value r and functions f and g:
- Map(Ok(v), f) SHALL equal Ok(f(v))
- Map(Err(e), f) SHALL equal Err(e)
- Map(Map(r, f), g) SHALL equal Map(r, compose(g, f))
**Validates: Requirements 3.3**

### Property 6: Result/Option Conversion Round-Trip
*For any* successful Result[T], converting to Option and checking IsSome SHALL return true. For any failed Result, ToOption SHALL return None.
**Validates: Requirements 3.5**

## Token Extraction Properties (7-9)

### Property 7: Token Extraction Scheme Correctness
*For any* valid authorization header with "Bearer" or "DPoP" prefix, extraction SHALL succeed and return the correct TokenScheme.
**Validates: Requirements 4.2, 4.3**

### Property 8: Chained Extractor Fallback
*For any* ChainedExtractor where at least one extractor succeeds, the chain SHALL return the first successful result.
**Validates: Requirements 4.4**

### Property 9: Token Format Validation
*For any* malformed authorization header (missing scheme, empty token, unsupported scheme), extraction SHALL fail with an appropriate error.
**Validates: Requirements 4.5**

## Retry Properties (10-13)

### Property 10: Retry Delay Exponential Backoff
*For any* attempt number n and RetryPolicy p, the calculated delay SHALL be approximately baseDelay * 2^n, bounded by [baseDelay, maxDelay], with jitter within ±jitter%.
**Validates: Requirements 5.2**

### Property 11: Retry-After Header Parsing
*For any* valid Retry-After header (integer seconds or HTTP-date), ParseRetryAfter SHALL return the correct duration. For invalid headers, it SHALL return (0, false).
**Validates: Requirements 5.3**

### Property 12: Retry Success Behavior
*For any* operation that succeeds on attempt N (where N ≤ maxRetries+1), Retry SHALL return success with Attempts equal to N.
**Validates: Requirements 5.4**

### Property 13: Retry Exhaustion Behavior
*For any* operation that always fails, Retry SHALL return the last error after exactly maxRetries+1 attempts.
**Validates: Requirements 5.5**

## JWKS Cache Properties (14)

### Property 14: JWKS Cache Metrics Consistency
*For any* sequence of cache operations, the sum of Hits + Misses SHALL equal the total number of lookup operations.
**Validates: Requirements 6.4**

## DPoP Properties (15-17)

### Property 15: DPoP Proof Generation and Validation Round-Trip
*For any* generated DPoP proof with method M and URI U, validating that proof with the same M and U SHALL succeed and return matching claims.
**Validates: Requirements 7.1, 7.2, 7.6**

### Property 16: DPoP ATH Computation Round-Trip
*For any* access token T, ComputeATH(T) SHALL be deterministic, and VerifyATH(T, ComputeATH(T)) SHALL return true.
**Validates: Requirements 7.3**

### Property 17: JWK Thumbprint Determinism
*For any* public key K, ComputeJWKThumbprint(K) SHALL always return the same value (deterministic).
**Validates: Requirements 7.4**

## PKCE Properties (18-20)

### Property 18: PKCE Verifier Generation
*For any* generated verifier, its length SHALL be in [43, 128] and all characters SHALL be from the unreserved set [A-Za-z0-9-._~].
**Validates: Requirements 8.1**

### Property 19: PKCE Round-Trip
*For any* valid verifier V, VerifyPKCE(V, ComputeChallenge(V)) SHALL return true.
**Validates: Requirements 8.5**

### Property 20: PKCE Validation
*For any* string S, ValidateVerifier SHALL return nil if and only if S has length in [43, 128] and contains only unreserved characters. Otherwise, it SHALL return an error indicating the specific violation.
**Validates: Requirements 8.3, 8.4**

## Middleware Properties (21-24)

### Property 21: Middleware Skip Patterns
*For any* request path matching a skip pattern, the middleware SHALL pass the request through without authentication.
**Validates: Requirements 9.1**

### Property 22: Middleware Claims Context
*For any* successfully authenticated request, GetClaimsFromContext SHALL return the validated claims.
**Validates: Requirements 9.3**

### Property 23: Error to Status Code Mapping
*For any* SDKError with a specific ErrorCode, MapToGRPCError SHALL return the corresponding gRPC status code as defined in the mapping table.
**Validates: Requirements 9.5**

### Property 24: Middleware Audience/Issuer Validation
*For any* token with audience A and issuer I, validation with expected audience A' and issuer I' SHALL succeed if and only if A contains A' and I equals I'.
**Validates: Requirements 9.6**

## Observability Properties (25)

### Property 25: Sensitive Data Filtering
*For any* log message or trace attribute containing sensitive patterns, the filtered output SHALL replace those patterns with "[REDACTED]".
**Validates: Requirements 10.3, 10.5**

## Configuration Properties (26-29)

### Property 26: Config Environment Loading
*For any* set of environment variables matching AUTH_PLATFORM_* pattern, LoadFromEnv SHALL return a Config with corresponding values set.
**Validates: Requirements 11.1**

### Property 27: Config Validation
*For any* Config with invalid values (empty BaseURL, empty ClientID, negative Timeout, TTL outside [1min, 24h]), Validate SHALL return an error describing the specific violation.
**Validates: Requirements 11.2, 11.4**

### Property 28: Config Defaults
*For any* Config with zero values for optional fields, ApplyDefaults SHALL set non-zero default values (Timeout=30s, JWKSCacheTTL=1h, MaxRetries=3, BaseDelay=1s, MaxDelay=30s).
**Validates: Requirements 11.3**

### Property 29: Config Functional Options
*For any* functional option applied to a Client, the corresponding Config field SHALL reflect the option's value.
**Validates: Requirements 11.5**
