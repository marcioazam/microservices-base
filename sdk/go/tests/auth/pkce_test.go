// Package auth provides unit tests for PKCE implementation.
package auth

import (
	"testing"

	"github.com/auth-platform/sdk-go/src/auth"
)

func TestGenerateVerifier(t *testing.T) {
	verifier, err := auth.GenerateVerifier()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(verifier) != auth.DefaultVerifierLength {
		t.Errorf("verifier length = %d, want %d", len(verifier), auth.DefaultVerifierLength)
	}

	if err := auth.ValidateVerifier(verifier); err != nil {
		t.Errorf("generated verifier is invalid: %v", err)
	}
}

func TestGenerateVerifierWithLength(t *testing.T) {
	tests := []struct {
		name    string
		length  int
		wantErr bool
	}{
		{"minimum length", 43, false},
		{"maximum length", 128, false},
		{"default length", 64, false},
		{"too short", 42, true},
		{"too long", 129, true},
		{"zero", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			verifier, err := auth.GenerateVerifierWithLength(tt.length)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(verifier) != tt.length {
				t.Errorf("verifier length = %d, want %d", len(verifier), tt.length)
			}
		})
	}
}

func TestComputeChallenge(t *testing.T) {
	// Known test vector from RFC 7636 Appendix B
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	expectedChallenge := "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"

	challenge := auth.ComputeChallenge(verifier)
	if challenge != expectedChallenge {
		t.Errorf("challenge = %s, want %s", challenge, expectedChallenge)
	}
}

func TestVerifyPKCE(t *testing.T) {
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	challenge := "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"

	if !auth.VerifyPKCE(verifier, challenge) {
		t.Error("VerifyPKCE should return true for valid pair")
	}

	if auth.VerifyPKCE(verifier, "wrong-challenge") {
		t.Error("VerifyPKCE should return false for invalid challenge")
	}

	if auth.VerifyPKCE("wrong-verifier", challenge) {
		t.Error("VerifyPKCE should return false for invalid verifier")
	}
}

func TestValidateVerifier(t *testing.T) {
	tests := []struct {
		name     string
		verifier string
		wantErr  bool
	}{
		{"valid 43 chars", "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopq", false},
		{"valid with special chars", "ABCDEFGHIJKLMNOPQRSTUVWXYZ._~-abcdefghijklm", false},
		{"too short", "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmno", true},
		{"invalid char space", "ABCDEFGHIJKLMNOPQRSTUVWXYZ abcdefghijklmnop", true},
		{"invalid char plus", "ABCDEFGHIJKLMNOPQRSTUVWXYZ+abcdefghijklmnop", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := auth.ValidateVerifier(tt.verifier)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestGeneratePKCE(t *testing.T) {
	pair, err := auth.GeneratePKCE()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pair.Verifier == "" {
		t.Error("verifier should not be empty")
	}
	if pair.Challenge == "" {
		t.Error("challenge should not be empty")
	}
	if pair.Method != auth.PKCEMethodS256 {
		t.Errorf("method = %s, want %s", pair.Method, auth.PKCEMethodS256)
	}

	// Verify round-trip
	if !auth.VerifyPKCE(pair.Verifier, pair.Challenge) {
		t.Error("generated PKCE pair should verify")
	}
}

func TestVerifyPKCEWithMethod(t *testing.T) {
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	s256Challenge := "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"

	// S256 method
	if !auth.VerifyPKCEWithMethod(verifier, s256Challenge, auth.PKCEMethodS256) {
		t.Error("S256 verification should pass")
	}

	// Plain method
	if !auth.VerifyPKCEWithMethod(verifier, verifier, auth.PKCEMethodPlain) {
		t.Error("plain verification should pass")
	}

	// Unknown method
	if auth.VerifyPKCEWithMethod(verifier, verifier, "unknown") {
		t.Error("unknown method should fail")
	}
}

func TestComputeChallengeWithMethod(t *testing.T) {
	verifier := "test-verifier-with-enough-length-to-be-valid"

	// S256
	challenge, err := auth.ComputeChallengeWithMethod(verifier, auth.PKCEMethodS256)
	if err != nil {
		t.Fatalf("S256 should not error: %v", err)
	}
	if challenge == verifier {
		t.Error("S256 challenge should differ from verifier")
	}

	// Plain
	challenge, err = auth.ComputeChallengeWithMethod(verifier, auth.PKCEMethodPlain)
	if err != nil {
		t.Fatalf("plain should not error: %v", err)
	}
	if challenge != verifier {
		t.Error("plain challenge should equal verifier")
	}

	// Unknown
	_, err = auth.ComputeChallengeWithMethod(verifier, "unknown")
	if err == nil {
		t.Error("unknown method should error")
	}
}

func TestPKCEGenerator(t *testing.T) {
	gen := auth.NewPKCEGenerator()

	verifier, err := gen.GenerateVerifier()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	challenge := gen.ComputeChallenge(verifier)
	if !auth.VerifyPKCE(verifier, challenge) {
		t.Error("generator should produce valid pairs")
	}
}

func TestVerifierUniqueness(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		verifier, err := auth.GenerateVerifier()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if seen[verifier] {
			t.Fatal("generated duplicate verifier")
		}
		seen[verifier] = true
	}
}
