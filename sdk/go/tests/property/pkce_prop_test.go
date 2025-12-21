package property

import (
"testing"

authplatform "github.com/auth-platform/sdk-go"
"pgregory.net/rapid"
)

// TestProperty7_PKCEVerifierConstraints validates Property 7:
// For any generated PKCE code verifier, the length SHALL be between 43 and 128
// characters, and all characters SHALL be from the unreserved character set.
// **Validates: Requirements 5.1**
func TestProperty7_PKCEVerifierConstraints(t *testing.T) {
rapid.Check(t, func(t *rapid.T) {
gen := authplatform.NewPKCEGenerator()
verifier, err := gen.GenerateVerifier()
if err != nil {
t.Fatalf("failed to generate verifier: %v", err)
}

// Length constraint: 43-128 characters
if len(verifier) < 43 {
t.Errorf("verifier too short: %d < 43", len(verifier))
}
if len(verifier) > 128 {
t.Errorf("verifier too long: %d > 128", len(verifier))
}

// Character set constraint: unreserved characters only
for i, c := range verifier {
if !authplatform.IsUnreservedChar(c) {
t.Errorf("invalid character at position %d: %c", i, c)
}
}
})
}

// TestProperty8_PKCEChallengeRoundTrip validates Property 8:
// For any valid code verifier, computing the S256 challenge and then
// verifying the verifier against the challenge SHALL succeed.
// **Validates: Requirements 5.2**
func TestProperty8_PKCEChallengeRoundTrip(t *testing.T) {
rapid.Check(t, func(t *rapid.T) {
gen := authplatform.NewPKCEGenerator()
verifier, err := gen.GenerateVerifier()
if err != nil {
t.Fatalf("failed to generate verifier: %v", err)
}

challenge := gen.ComputeChallenge(verifier)

// Verify round-trip
if !authplatform.VerifyPKCE(verifier, challenge) {
t.Errorf("PKCE verification failed for verifier: %s", verifier)
}
})
}

// TestPKCEVerifierLengthVariation tests different verifier lengths.
func TestPKCEVerifierLengthVariation(t *testing.T) {
rapid.Check(t, func(t *rapid.T) {
length := rapid.IntRange(43, 128).Draw(t, "length")
gen := &authplatform.DefaultPKCEGenerator{VerifierLength: length}
verifier, err := gen.GenerateVerifier()
if err != nil {
t.Fatalf("failed to generate verifier: %v", err)
}

if len(verifier) != length {
t.Errorf("expected length %d, got %d", length, len(verifier))
}

// Verify it still works
challenge := gen.ComputeChallenge(verifier)
if !authplatform.VerifyPKCE(verifier, challenge) {
t.Error("PKCE verification failed")
}
})
}

// TestPKCEVerifierUniqueness tests that generated verifiers are unique.
func TestPKCEVerifierUniqueness(t *testing.T) {
gen := authplatform.NewPKCEGenerator()
seen := make(map[string]bool)

for i := 0; i < 100; i++ {
verifier, err := gen.GenerateVerifier()
if err != nil {
t.Fatalf("failed to generate verifier: %v", err)
}
if seen[verifier] {
t.Errorf("duplicate verifier generated: %s", verifier)
}
seen[verifier] = true
}
}

// TestPKCEChallengeFormat tests that challenges are valid base64url.
func TestPKCEChallengeFormat(t *testing.T) {
rapid.Check(t, func(t *rapid.T) {
gen := authplatform.NewPKCEGenerator()
verifier, err := gen.GenerateVerifier()
if err != nil {
t.Fatalf("failed to generate verifier: %v", err)
}

challenge := gen.ComputeChallenge(verifier)

// S256 challenge should be 43 characters (256 bits / 6 bits per char)
if len(challenge) != 43 {
t.Errorf("expected challenge length 43, got %d", len(challenge))
}

// Should only contain base64url characters
for _, c := range challenge {
if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') ||
(c >= '0' && c <= '9') || c == '-' || c == '_') {
t.Errorf("invalid base64url character: %c", c)
}
}
})
}

// TestPKCEValidateVerifier tests the verifier validation function.
func TestPKCEValidateVerifier(t *testing.T) {
// Valid verifier
gen := authplatform.NewPKCEGenerator()
verifier, _ := gen.GenerateVerifier()
if err := authplatform.ValidateVerifier(verifier); err != nil {
t.Errorf("valid verifier rejected: %v", err)
}

// Too short
if err := authplatform.ValidateVerifier("short"); err == nil {
t.Error("expected error for short verifier")
}

// Invalid character
if err := authplatform.ValidateVerifier("valid" + string(rune(0)) + "chars"); err == nil {
t.Error("expected error for invalid character")
}
}

// TestGeneratePKCEPair tests the convenience function.
func TestGeneratePKCEPair(t *testing.T) {
rapid.Check(t, func(t *rapid.T) {
pair, err := authplatform.GeneratePKCE()
if err != nil {
t.Fatalf("failed to generate PKCE pair: %v", err)
}

if pair.Method != "S256" {
t.Errorf("expected method S256, got %s", pair.Method)
}

if !authplatform.VerifyPKCE(pair.Verifier, pair.Challenge) {
t.Error("PKCE pair verification failed")
}
})
}
