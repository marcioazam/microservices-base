// Package auth provides property-based tests for PKCE implementation.
package auth

import (
	"regexp"
	"testing"

	"github.com/auth-platform/sdk-go/src/auth"
	"pgregory.net/rapid"
)

var validVerifierRegex = regexp.MustCompile(`^[A-Za-z0-9._~-]+$`)

// Property 18: PKCE Verifier Generation
func TestProperty_PKCEVerifierGeneration(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		verifier, err := auth.GenerateVerifier()
		if err != nil {
			t.Fatalf("GenerateVerifier failed: %v", err)
		}

		// Length must be within RFC 7636 bounds
		if len(verifier) < auth.MinVerifierLength {
			t.Fatalf("verifier too short: %d < %d", len(verifier), auth.MinVerifierLength)
		}
		if len(verifier) > auth.MaxVerifierLength {
			t.Fatalf("verifier too long: %d > %d", len(verifier), auth.MaxVerifierLength)
		}

		// Must match valid character set
		if !validVerifierRegex.MatchString(verifier) {
			t.Fatalf("verifier contains invalid characters: %s", verifier)
		}
	})
}

// Property 19: PKCE Round-Trip
func TestProperty_PKCERoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		verifier, err := auth.GenerateVerifier()
		if err != nil {
			t.Fatalf("GenerateVerifier failed: %v", err)
		}

		challenge := auth.ComputeChallenge(verifier)
		if !auth.VerifyPKCE(verifier, challenge) {
			t.Fatal("PKCE round-trip verification failed")
		}
	})
}

// Property 20: PKCE Validation
func TestProperty_PKCEValidation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate valid verifier
		verifier, err := auth.GenerateVerifier()
		if err != nil {
			t.Fatalf("GenerateVerifier failed: %v", err)
		}

		// Validation should pass
		if err := auth.ValidateVerifier(verifier); err != nil {
			t.Fatalf("valid verifier failed validation: %v", err)
		}
	})
}

// Property: Verifier length is configurable within bounds
func TestProperty_VerifierLengthConfigurable(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		length := rapid.IntRange(auth.MinVerifierLength, auth.MaxVerifierLength).Draw(t, "length")

		verifier, err := auth.GenerateVerifierWithLength(length)
		if err != nil {
			t.Fatalf("GenerateVerifierWithLength(%d) failed: %v", length, err)
		}

		if len(verifier) != length {
			t.Fatalf("verifier length = %d, want %d", len(verifier), length)
		}
	})
}

// Property: Invalid lengths are rejected
func TestProperty_InvalidLengthsRejected(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate length outside valid range
		tooShort := rapid.IntRange(0, auth.MinVerifierLength-1).Draw(t, "tooShort")
		_, err := auth.GenerateVerifierWithLength(tooShort)
		if err == nil {
			t.Fatalf("length %d should be rejected", tooShort)
		}
	})
}

// Property: Challenge is deterministic for same verifier
func TestProperty_ChallengeDeterministic(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		verifier, _ := auth.GenerateVerifier()

		challenge1 := auth.ComputeChallenge(verifier)
		challenge2 := auth.ComputeChallenge(verifier)

		if challenge1 != challenge2 {
			t.Fatal("challenge should be deterministic")
		}
	})
}

// Property: Different verifiers produce different challenges
func TestProperty_DifferentVerifiersDifferentChallenges(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		verifier1, _ := auth.GenerateVerifier()
		verifier2, _ := auth.GenerateVerifier()

		if verifier1 == verifier2 {
			t.Skip("rare collision, skip")
		}

		challenge1 := auth.ComputeChallenge(verifier1)
		challenge2 := auth.ComputeChallenge(verifier2)

		if challenge1 == challenge2 {
			t.Fatal("different verifiers should produce different challenges")
		}
	})
}

// Property: Wrong verifier fails verification
func TestProperty_WrongVerifierFails(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		verifier1, _ := auth.GenerateVerifier()
		verifier2, _ := auth.GenerateVerifier()

		if verifier1 == verifier2 {
			t.Skip("rare collision, skip")
		}

		challenge := auth.ComputeChallenge(verifier1)
		if auth.VerifyPKCE(verifier2, challenge) {
			t.Fatal("wrong verifier should fail verification")
		}
	})
}

// Property: GeneratePKCE produces valid pairs
func TestProperty_GeneratePKCEProducesValidPairs(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		pair, err := auth.GeneratePKCE()
		if err != nil {
			t.Fatalf("GeneratePKCE failed: %v", err)
		}

		if pair.Method != auth.PKCEMethodS256 {
			t.Fatalf("method = %s, want %s", pair.Method, auth.PKCEMethodS256)
		}

		if !auth.VerifyPKCE(pair.Verifier, pair.Challenge) {
			t.Fatal("generated pair should verify")
		}
	})
}

// Property: Challenge has expected length (base64url of SHA-256)
func TestProperty_ChallengeLength(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		verifier, _ := auth.GenerateVerifier()
		challenge := auth.ComputeChallenge(verifier)

		// SHA-256 = 32 bytes, base64url without padding = 43 chars
		expectedLen := 43
		if len(challenge) != expectedLen {
			t.Fatalf("challenge length = %d, want %d", len(challenge), expectedLen)
		}
	})
}

// Property: Plain method verification
func TestProperty_PlainMethodVerification(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		verifier, _ := auth.GenerateVerifier()

		// Plain method: challenge equals verifier
		if !auth.VerifyPKCEWithMethod(verifier, verifier, auth.PKCEMethodPlain) {
			t.Fatal("plain method should verify when challenge equals verifier")
		}

		// Plain method with different challenge should fail
		otherVerifier, _ := auth.GenerateVerifier()
		if verifier != otherVerifier && auth.VerifyPKCEWithMethod(verifier, otherVerifier, auth.PKCEMethodPlain) {
			t.Fatal("plain method should fail with different challenge")
		}
	})
}

// Property: Verifier uniqueness
func TestProperty_VerifierUniqueness(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		numVerifiers := rapid.IntRange(10, 50).Draw(t, "numVerifiers")
		seen := make(map[string]bool)

		for i := 0; i < numVerifiers; i++ {
			verifier, err := auth.GenerateVerifier()
			if err != nil {
				t.Fatalf("GenerateVerifier failed: %v", err)
			}
			if seen[verifier] {
				t.Fatal("duplicate verifier generated")
			}
			seen[verifier] = true
		}
	})
}
