// Package auth provides unit tests for DPoP implementation.
package auth

import (
	"context"
	"testing"
	"time"

	"github.com/auth-platform/sdk-go/src/auth"
)

func TestGenerateES256KeyPair(t *testing.T) {
	keyPair, err := auth.GenerateES256KeyPair()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if keyPair.PrivateKey == nil {
		t.Error("private key should not be nil")
	}
	if keyPair.PublicKey == nil {
		t.Error("public key should not be nil")
	}
	if keyPair.Algorithm != "ES256" {
		t.Errorf("algorithm = %s, want ES256", keyPair.Algorithm)
	}
	if keyPair.KeyID == "" {
		t.Error("key ID should not be empty")
	}
}

func TestGenerateRS256KeyPair(t *testing.T) {
	keyPair, err := auth.GenerateRS256KeyPair()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if keyPair.PrivateKey == nil {
		t.Error("private key should not be nil")
	}
	if keyPair.PublicKey == nil {
		t.Error("public key should not be nil")
	}
	if keyPair.Algorithm != "RS256" {
		t.Errorf("algorithm = %s, want RS256", keyPair.Algorithm)
	}
	if keyPair.KeyID == "" {
		t.Error("key ID should not be empty")
	}
}

func TestDPoPProver_GenerateProof_ES256(t *testing.T) {
	keyPair, err := auth.GenerateES256KeyPair()
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	prover := auth.NewDPoPProver(keyPair)
	ctx := context.Background()

	proof, err := prover.GenerateProof(ctx, "POST", "https://example.com/token", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if proof == "" {
		t.Error("proof should not be empty")
	}
}

func TestDPoPProver_GenerateProof_RS256(t *testing.T) {
	keyPair, err := auth.GenerateRS256KeyPair()
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	prover := auth.NewDPoPProver(keyPair)
	ctx := context.Background()

	proof, err := prover.GenerateProof(ctx, "GET", "https://example.com/resource", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if proof == "" {
		t.Error("proof should not be empty")
	}
}

func TestDPoPProver_GenerateProof_WithAccessToken(t *testing.T) {
	keyPair, err := auth.GenerateES256KeyPair()
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	prover := auth.NewDPoPProver(keyPair)
	ctx := context.Background()

	accessToken := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.test"
	proof, err := prover.GenerateProof(ctx, "GET", "https://example.com/api", accessToken)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if proof == "" {
		t.Error("proof should not be empty")
	}
}

func TestDPoPProver_ValidateProof_RoundTrip(t *testing.T) {
	keyPair, err := auth.GenerateES256KeyPair()
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	prover := auth.NewDPoPProver(keyPair)
	ctx := context.Background()
	method := "POST"
	uri := "https://example.com/token"

	proof, err := prover.GenerateProof(ctx, method, uri, "")
	if err != nil {
		t.Fatalf("failed to generate proof: %v", err)
	}

	claims, err := prover.ValidateProof(ctx, proof, method, uri)
	if err != nil {
		t.Fatalf("failed to validate proof: %v", err)
	}

	if claims.HTTPMethod != method {
		t.Errorf("method = %s, want %s", claims.HTTPMethod, method)
	}
	if claims.HTTPUri != uri {
		t.Errorf("uri = %s, want %s", claims.HTTPUri, uri)
	}
}

func TestDPoPProver_ValidateProof_MethodMismatch(t *testing.T) {
	keyPair, err := auth.GenerateES256KeyPair()
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	prover := auth.NewDPoPProver(keyPair)
	ctx := context.Background()

	proof, err := prover.GenerateProof(ctx, "POST", "https://example.com/token", "")
	if err != nil {
		t.Fatalf("failed to generate proof: %v", err)
	}

	_, err = prover.ValidateProof(ctx, proof, "GET", "https://example.com/token")
	if err == nil {
		t.Error("expected error for method mismatch")
	}
}

func TestDPoPProver_ValidateProof_URIMismatch(t *testing.T) {
	keyPair, err := auth.GenerateES256KeyPair()
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	prover := auth.NewDPoPProver(keyPair)
	ctx := context.Background()

	proof, err := prover.GenerateProof(ctx, "POST", "https://example.com/token", "")
	if err != nil {
		t.Fatalf("failed to generate proof: %v", err)
	}

	_, err = prover.ValidateProof(ctx, proof, "POST", "https://other.com/token")
	if err == nil {
		t.Error("expected error for URI mismatch")
	}
}

func TestDPoPProver_NilKeyPair(t *testing.T) {
	prover := auth.NewDPoPProver(nil)
	ctx := context.Background()

	_, err := prover.GenerateProof(ctx, "GET", "https://example.com", "")
	if err == nil {
		t.Error("expected error for nil key pair")
	}
}

func TestComputeATH(t *testing.T) {
	accessToken := "test-access-token"
	ath := auth.ComputeATH(accessToken)

	if ath == "" {
		t.Error("ATH should not be empty")
	}

	// Should be deterministic
	ath2 := auth.ComputeATH(accessToken)
	if ath != ath2 {
		t.Error("ATH should be deterministic")
	}
}

func TestVerifyATH(t *testing.T) {
	accessToken := "test-access-token"
	ath := auth.ComputeATH(accessToken)

	if !auth.VerifyATH(accessToken, ath) {
		t.Error("VerifyATH should return true for valid ATH")
	}

	if auth.VerifyATH(accessToken, "wrong-ath") {
		t.Error("VerifyATH should return false for invalid ATH")
	}

	if auth.VerifyATH("wrong-token", ath) {
		t.Error("VerifyATH should return false for wrong token")
	}
}

func TestComputeJWKThumbprint_ES256(t *testing.T) {
	keyPair, err := auth.GenerateES256KeyPair()
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	thumbprint, err := auth.ComputeJWKThumbprint(keyPair.PublicKey)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if thumbprint == "" {
		t.Error("thumbprint should not be empty")
	}

	// Should be deterministic
	thumbprint2, _ := auth.ComputeJWKThumbprint(keyPair.PublicKey)
	if thumbprint != thumbprint2 {
		t.Error("thumbprint should be deterministic")
	}
}

func TestComputeJWKThumbprint_RS256(t *testing.T) {
	keyPair, err := auth.GenerateRS256KeyPair()
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	thumbprint, err := auth.ComputeJWKThumbprint(keyPair.PublicKey)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if thumbprint == "" {
		t.Error("thumbprint should not be empty")
	}
}

func TestDPoPProofExpiry(t *testing.T) {
	// This test verifies the 5-minute expiry window is enforced
	// We can't easily test expired proofs without mocking time
	// but we verify the constant is set correctly
	if auth.DPoPProofMaxAge != 5*time.Minute {
		t.Errorf("DPoPProofMaxAge = %v, want 5m", auth.DPoPProofMaxAge)
	}
}
