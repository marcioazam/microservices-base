# Phase 2: Token Handling and Security

## Task 2.1: Implement Unified Token Extraction
**Requirement:** REQ-4 (Token Extraction Centralization)
**Acceptance Criteria:** 4.1, 4.2, 4.3, 4.4, 4.5

Centralize token extraction in `src/token/`:

**Files to create:**
- `src/token/extractor.go` - Unified extraction interface and implementations
- `src/token/schemes.go` - Token scheme definitions
- `tests/token/extractor_test.go` - Unit tests
- `tests/token/extractor_prop_test.go` - Property tests

**Property Tests (Properties 7-9):**
```go
// Property 7: Token Extraction Scheme Correctness
func TestProperty_TokenExtractionScheme(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        scheme := rapid.SampledFrom([]string{"Bearer", "DPoP"}).Draw(t, "scheme")
        token := rapid.StringMatching(`[A-Za-z0-9._-]+`).Draw(t, "token")
        header := scheme + " " + token
        
        extracted, extractedScheme, err := extractFromHeader(header)
        assert.NoError(t, err)
        assert.Equal(t, token, extracted)
        assert.Equal(t, TokenScheme(scheme), extractedScheme)
    })
}

// Property 8: Chained Extractor Fallback
func TestProperty_ChainedExtractorFallback(t *rapid.T) { ... }

// Property 9: Token Format Validation
func TestProperty_TokenFormatValidation(t *rapid.T) { ... }
```

**Deliverables:**
- [ ] Define `TokenScheme` type (Bearer, DPoP, Unknown)
- [ ] Define `Extractor` interface
- [ ] Implement `HTTPExtractor` for Authorization headers
- [ ] Implement `GRPCExtractor` for gRPC metadata
- [ ] Implement `CookieExtractor` for HTTP cookies
- [ ] Implement `ChainedExtractor` with fallback logic
- [ ] Property tests for Properties 7-9 (100+ iterations each)
- [ ] Unit tests for malformed headers and edge cases

---

## Task 2.2: Implement PKCE Support
**Requirement:** REQ-8 (PKCE Implementation)
**Acceptance Criteria:** 8.1, 8.2, 8.3, 8.4, 8.5

Implement PKCE in `src/auth/`:

**Files to create:**
- `src/auth/pkce.go` - PKCE implementation
- `tests/auth/pkce_test.go` - Unit tests
- `tests/auth/pkce_prop_test.go` - Property tests

**Property Tests (Properties 18-20):**
```go
// Property 18: PKCE Verifier Generation
func TestProperty_PKCEVerifierGeneration(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        verifier, err := GenerateVerifier()
        assert.NoError(t, err)
        assert.GreaterOrEqual(t, len(verifier), 43)
        assert.LessOrEqual(t, len(verifier), 128)
        assert.Regexp(t, `^[A-Za-z0-9._~-]+$`, verifier)
    })
}

// Property 19: PKCE Round-Trip
func TestProperty_PKCERoundTrip(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        verifier, _ := GenerateVerifier()
        challenge := ComputeChallenge(verifier)
        assert.True(t, VerifyPKCE(verifier, challenge))
    })
}

// Property 20: PKCE Validation
func TestProperty_PKCEValidation(t *rapid.T) { ... }
```

**Deliverables:**
- [ ] Implement `GenerateVerifier()` with crypto/rand
- [ ] Implement `ComputeChallenge()` using SHA-256 + base64url
- [ ] Implement `VerifyPKCE()` for round-trip verification
- [ ] Implement `ValidateVerifier()` per RFC 7636
- [ ] Implement `PKCEPair` struct and `GeneratePKCE()` helper
- [ ] Property tests for Properties 18-20 (100+ iterations each)
- [ ] Unit tests for invalid characters and boundary lengths

---

## Task 2.3: Implement DPoP Support
**Requirement:** REQ-7 (DPoP Implementation Enhancement)
**Acceptance Criteria:** 7.1, 7.2, 7.3, 7.4, 7.5, 7.6

Implement DPoP in `src/auth/`:

**Files to create:**
- `src/auth/dpop.go` - DPoP proof generation and validation
- `src/auth/dpop_jwk.go` - JWK utilities for DPoP
- `tests/auth/dpop_test.go` - Unit tests
- `tests/auth/dpop_prop_test.go` - Property tests

**Property Tests (Properties 15-17):**
```go
// Property 15: DPoP Proof Generation and Validation Round-Trip
func TestProperty_DPoPRoundTrip(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        method := rapid.SampledFrom([]string{"GET", "POST", "PUT", "DELETE"}).Draw(t, "method")
        uri := "https://example.com/" + rapid.StringMatching(`[a-z]+`).Draw(t, "path")
        
        keyPair, _ := GenerateES256KeyPair()
        prover := NewDPoPProver(keyPair)
        
        proof, err := prover.GenerateProof(ctx, method, uri, "")
        assert.NoError(t, err)
        
        claims, err := prover.ValidateProof(ctx, proof, method, uri)
        assert.NoError(t, err)
        assert.Equal(t, method, claims.HTTPMethod)
        assert.Equal(t, uri, claims.HTTPUri)
    })
}

// Property 16: DPoP ATH Computation Round-Trip
func TestProperty_DPoPATHRoundTrip(t *rapid.T) { ... }

// Property 17: JWK Thumbprint Determinism
func TestProperty_JWKThumbprintDeterminism(t *rapid.T) { ... }
```

**Deliverables:**
- [ ] Implement `DPoPProver` interface
- [ ] Implement `GenerateProof()` with ES256/RS256 support
- [ ] Implement `ValidateProof()` with method, URI, timestamp checks
- [ ] Implement `ComputeATH()` and `VerifyATH()` for access token hash
- [ ] Implement `ComputeJWKThumbprint()` per RFC 7638
- [ ] Implement `GenerateES256KeyPair()` and `GenerateRS256KeyPair()`
- [ ] Implement 5-minute expiry validation
- [ ] Property tests for Properties 15-17 (100+ iterations each)
- [ ] Unit tests for expired proofs and algorithm mismatches
