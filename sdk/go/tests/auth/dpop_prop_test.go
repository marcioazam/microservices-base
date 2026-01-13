// Package auth provides property-based tests for DPoP implementation.
package auth

import (
	"context"
	"testing"

	"github.com/auth-platform/sdk-go/src/auth"
	"pgregory.net/rapid"
)

// Property 15: DPoP Proof Generation and Validation Round-Trip
func TestProperty_DPoPRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		method := rapid.SampledFrom([]string{"GET", "POST", "PUT", "DELETE", "PATCH"}).Draw(t, "method")
		path := rapid.StringMatching(`[a-z]{1,20}`).Draw(t, "path")
		uri := "https://example.com/" + path

		keyPair, err := auth.GenerateES256KeyPair()
		if err != nil {
			t.Fatalf("failed to generate key pair: %v", err)
		}

		prover := auth.NewDPoPProver(keyPair)
		ctx := context.Background()

		proof, err := prover.GenerateProof(ctx, method, uri, "")
		if err != nil {
			t.Fatalf("failed to generate proof: %v", err)
		}

		claims, err := prover.ValidateProof(ctx, proof, method, uri)
		if err != nil {
			t.Fatalf("failed to validate proof: %v", err)
		}

		if claims.HTTPMethod != method {
			t.Fatalf("method = %s, want %s", claims.HTTPMethod, method)
		}
		if claims.HTTPUri != uri {
			t.Fatalf("uri = %s, want %s", claims.HTTPUri, uri)
		}
	})
}

// Property 16: DPoP ATH Computation Round-Trip
func TestProperty_DPoPATHRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		accessToken := rapid.StringMatching(`[A-Za-z0-9._-]{20,100}`).Draw(t, "accessToken")

		ath := auth.ComputeATH(accessToken)
		if !auth.VerifyATH(accessToken, ath) {
			t.Fatal("ATH round-trip verification failed")
		}
	})
}

// Property 17: JWK Thumbprint Determinism
func TestProperty_JWKThumbprintDeterminism(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		keyPair, err := auth.GenerateES256KeyPair()
		if err != nil {
			t.Fatalf("failed to generate key pair: %v", err)
		}

		thumbprint1, err := auth.ComputeJWKThumbprint(keyPair.PublicKey)
		if err != nil {
			t.Fatalf("failed to compute thumbprint: %v", err)
		}

		thumbprint2, err := auth.ComputeJWKThumbprint(keyPair.PublicKey)
		if err != nil {
			t.Fatalf("failed to compute thumbprint: %v", err)
		}

		if thumbprint1 != thumbprint2 {
			t.Fatal("JWK thumbprint should be deterministic")
		}
	})
}

// Property: Different keys produce different thumbprints
func TestProperty_DifferentKeysProduceDifferentThumbprints(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		keyPair1, _ := auth.GenerateES256KeyPair()
		keyPair2, _ := auth.GenerateES256KeyPair()

		thumbprint1, _ := auth.ComputeJWKThumbprint(keyPair1.PublicKey)
		thumbprint2, _ := auth.ComputeJWKThumbprint(keyPair2.PublicKey)

		if thumbprint1 == thumbprint2 {
			t.Fatal("different keys should produce different thumbprints")
		}
	})
}

// Property: ATH is deterministic
func TestProperty_ATHDeterministic(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		accessToken := rapid.StringMatching(`[A-Za-z0-9._-]{20,100}`).Draw(t, "accessToken")

		ath1 := auth.ComputeATH(accessToken)
		ath2 := auth.ComputeATH(accessToken)

		if ath1 != ath2 {
			t.Fatal("ATH should be deterministic")
		}
	})
}

// Property: Different tokens produce different ATH
func TestProperty_DifferentTokensDifferentATH(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		token1 := rapid.StringMatching(`[A-Za-z0-9._-]{20,100}`).Draw(t, "token1")
		token2 := rapid.StringMatching(`[A-Za-z0-9._-]{20,100}`).Draw(t, "token2")

		if token1 == token2 {
			t.Skip("tokens are equal, skip")
		}

		ath1 := auth.ComputeATH(token1)
		ath2 := auth.ComputeATH(token2)

		if ath1 == ath2 {
			t.Fatal("different tokens should produce different ATH")
		}
	})
}

// Property: Wrong token fails ATH verification
func TestProperty_WrongTokenFailsATH(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		token1 := rapid.StringMatching(`[A-Za-z0-9._-]{20,100}`).Draw(t, "token1")
		token2 := rapid.StringMatching(`[A-Za-z0-9._-]{20,100}`).Draw(t, "token2")

		if token1 == token2 {
			t.Skip("tokens are equal, skip")
		}

		ath := auth.ComputeATH(token1)
		if auth.VerifyATH(token2, ath) {
			t.Fatal("wrong token should fail ATH verification")
		}
	})
}

// Property: RS256 key pairs work correctly
func TestProperty_RS256KeyPairWorks(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		method := rapid.SampledFrom([]string{"GET", "POST"}).Draw(t, "method")
		uri := "https://example.com/api"

		keyPair, err := auth.GenerateRS256KeyPair()
		if err != nil {
			t.Fatalf("failed to generate RS256 key pair: %v", err)
		}

		prover := auth.NewDPoPProver(keyPair)
		ctx := context.Background()

		proof, err := prover.GenerateProof(ctx, method, uri, "")
		if err != nil {
			t.Fatalf("failed to generate proof: %v", err)
		}

		claims, err := prover.ValidateProof(ctx, proof, method, uri)
		if err != nil {
			t.Fatalf("failed to validate proof: %v", err)
		}

		if claims.HTTPMethod != method {
			t.Fatalf("method mismatch")
		}
	})
}

// Property: Proof with ATH validates correctly
func TestProperty_ProofWithATHValidates(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		accessToken := rapid.StringMatching(`[A-Za-z0-9._-]{20,50}`).Draw(t, "accessToken")
		method := "POST"
		uri := "https://example.com/token"

		keyPair, _ := auth.GenerateES256KeyPair()
		prover := auth.NewDPoPProver(keyPair)
		ctx := context.Background()

		proof, err := prover.GenerateProof(ctx, method, uri, accessToken)
		if err != nil {
			t.Fatalf("failed to generate proof: %v", err)
		}

		claims, err := prover.ValidateProof(ctx, proof, method, uri)
		if err != nil {
			t.Fatalf("failed to validate proof: %v", err)
		}

		// Verify ATH is present and correct
		if claims.AccessTokenHash == "" {
			t.Fatal("ATH should be present when access token provided")
		}
		if !auth.VerifyATH(accessToken, claims.AccessTokenHash) {
			t.Fatal("ATH should match access token")
		}
	})
}

// Property: Key ID is unique per key pair
func TestProperty_KeyIDUnique(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		numKeys := rapid.IntRange(5, 20).Draw(t, "numKeys")
		seen := make(map[string]bool)

		for i := 0; i < numKeys; i++ {
			keyPair, err := auth.GenerateES256KeyPair()
			if err != nil {
				t.Fatalf("failed to generate key pair: %v", err)
			}
			if seen[keyPair.KeyID] {
				t.Fatal("duplicate key ID generated")
			}
			seen[keyPair.KeyID] = true
		}
	})
}
