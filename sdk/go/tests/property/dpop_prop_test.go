package property

import (
	"context"
	"strings"
	"testing"

	authplatform "github.com/auth-platform/sdk-go"
	"pgregory.net/rapid"
)

// TestProperty9_DPoPProofRequiredClaims validates Property 9
func TestProperty9_DPoPProofRequiredClaims(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		keyPair, err := authplatform.GenerateES256KeyPair()
		if err != nil {
			t.Fatalf("failed to generate key pair: %v", err)
		}

		prover := authplatform.NewDPoPProver(keyPair)
		method := rapid.SampledFrom([]string{"GET", "POST", "PUT", "DELETE"}).Draw(t, "method")

		proof, err := prover.GenerateProof(context.Background(), method, "https://api.example.com/test", "")
		if err != nil {
			t.Fatalf("failed to generate proof: %v", err)
		}

		claims, err := prover.ValidateProof(context.Background(), proof, method, "https://api.example.com/test")
		if err != nil {
			t.Fatalf("failed to validate proof: %v", err)
		}

		if claims.ID == "" {
			t.Error("missing jti claim")
		}
		if claims.HTTPMethod != method {
			t.Errorf("expected htm %s, got %s", method, claims.HTTPMethod)
		}
		if claims.IssuedAt == nil {
			t.Error("missing iat claim")
		}
	})
}


// TestProperty10_DPoPATHClaimCorrectness validates Property 10
func TestProperty10_DPoPATHClaimCorrectness(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		keyPair, err := authplatform.GenerateES256KeyPair()
		if err != nil {
			t.Fatalf("failed to generate key pair: %v", err)
		}

		prover := authplatform.NewDPoPProver(keyPair)
		accessToken := rapid.StringMatching("[A-Za-z0-9]{32,64}").Draw(t, "accessToken")

		proof, err := prover.GenerateProof(context.Background(), "POST", "https://api.example.com/token", accessToken)
		if err != nil {
			t.Fatalf("failed to generate proof: %v", err)
		}

		claims, err := prover.ValidateProof(context.Background(), proof, "POST", "https://api.example.com/token")
		if err != nil {
			t.Fatalf("failed to validate proof: %v", err)
		}

		if claims.AccessTokenHash == "" {
			t.Error("missing ath claim when access token provided")
		}

		if !authplatform.VerifyATH(accessToken, claims.AccessTokenHash) {
			t.Error("ATH verification failed")
		}
	})
}

// TestProperty11_DPoPAlgorithmSupport validates Property 11
func TestProperty11_DPoPAlgorithmSupport(t *testing.T) {
	t.Run("ES256", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			keyPair, err := authplatform.GenerateES256KeyPair()
			if err != nil {
				t.Fatalf("failed to generate ES256 key pair: %v", err)
			}
			if keyPair.Algorithm != "ES256" {
				t.Errorf("expected algorithm ES256, got %s", keyPair.Algorithm)
			}
			prover := authplatform.NewDPoPProver(keyPair)
			proof, err := prover.GenerateProof(context.Background(), "POST", "https://api.example.com/resource", "")
			if err != nil {
				t.Fatalf("failed to generate proof: %v", err)
			}
			_, err = prover.ValidateProof(context.Background(), proof, "POST", "https://api.example.com/resource")
			if err != nil {
				t.Fatalf("failed to validate proof: %v", err)
			}
		})
	})

	t.Run("RS256", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			keyPair, err := authplatform.GenerateRS256KeyPair()
			if err != nil {
				t.Fatalf("failed to generate RS256 key pair: %v", err)
			}
			if keyPair.Algorithm != "RS256" {
				t.Errorf("expected algorithm RS256, got %s", keyPair.Algorithm)
			}
			prover := authplatform.NewDPoPProver(keyPair)
			proof, err := prover.GenerateProof(context.Background(), "POST", "https://api.example.com/resource", "")
			if err != nil {
				t.Fatalf("failed to generate proof: %v", err)
			}
			_, err = prover.ValidateProof(context.Background(), proof, "POST", "https://api.example.com/resource")
			if err != nil {
				t.Fatalf("failed to validate proof: %v", err)
			}
		})
	})
}


// TestProperty12_DPoPValidationCorrectness validates Property 12
func TestProperty12_DPoPValidationCorrectness(t *testing.T) {
	t.Run("ValidProof", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			keyPair, _ := authplatform.GenerateES256KeyPair()
			prover := authplatform.NewDPoPProver(keyPair)
			proof, err := prover.GenerateProof(context.Background(), "GET", "https://api.example.com/data", "")
			if err != nil {
				t.Fatalf("failed to generate proof: %v", err)
			}
			_, err = prover.ValidateProof(context.Background(), proof, "GET", "https://api.example.com/data")
			if err != nil {
				t.Errorf("valid proof rejected: %v", err)
			}
		})
	})

	t.Run("WrongMethod", func(t *testing.T) {
		keyPair, _ := authplatform.GenerateES256KeyPair()
		prover := authplatform.NewDPoPProver(keyPair)
		proof, _ := prover.GenerateProof(context.Background(), "GET", "https://api.example.com/data", "")
		_, err := prover.ValidateProof(context.Background(), proof, "POST", "https://api.example.com/data")
		if err == nil {
			t.Error("expected error for wrong method")
		}
	})

	t.Run("WrongURI", func(t *testing.T) {
		keyPair, _ := authplatform.GenerateES256KeyPair()
		prover := authplatform.NewDPoPProver(keyPair)
		proof, _ := prover.GenerateProof(context.Background(), "GET", "https://api.example.com/data", "")
		_, err := prover.ValidateProof(context.Background(), proof, "GET", "https://api.example.com/other")
		if err == nil {
			t.Error("expected error for wrong URI")
		}
	})

	t.Run("TamperedSignature", func(t *testing.T) {
		keyPair, _ := authplatform.GenerateES256KeyPair()
		prover := authplatform.NewDPoPProver(keyPair)
		proof, _ := prover.GenerateProof(context.Background(), "GET", "https://api.example.com/data", "")
		parts := strings.Split(proof, ".")
		if len(parts) == 3 {
			parts[2] = "tampered_signature"
			tamperedProof := strings.Join(parts, ".")
			_, err := prover.ValidateProof(context.Background(), tamperedProof, "GET", "https://api.example.com/data")
			if err == nil {
				t.Error("expected error for tampered signature")
			}
		}
	})
}

// TestDPoPKeyIDUniqueness tests that generated key IDs are unique
func TestDPoPKeyIDUniqueness(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 50; i++ {
		keyPair, err := authplatform.GenerateES256KeyPair()
		if err != nil {
			t.Fatalf("failed to generate key pair: %v", err)
		}
		if seen[keyPair.KeyID] {
			t.Errorf("duplicate key ID: %s", keyPair.KeyID)
		}
		seen[keyPair.KeyID] = true
	}
}

// TestDPoPJTIUniqueness tests that generated JTIs are unique
func TestDPoPJTIUniqueness(t *testing.T) {
	keyPair, _ := authplatform.GenerateES256KeyPair()
	prover := authplatform.NewDPoPProver(keyPair)
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		proof, err := prover.GenerateProof(context.Background(), "GET", "https://api.example.com", "")
		if err != nil {
			t.Fatalf("failed to generate proof: %v", err)
		}
		claims, err := prover.ValidateProof(context.Background(), proof, "GET", "https://api.example.com")
		if err != nil {
			t.Fatalf("failed to validate proof: %v", err)
		}
		if seen[claims.ID] {
			t.Errorf("duplicate JTI: %s", claims.ID)
		}
		seen[claims.ID] = true
	}
}
