// Package auth provides authentication utilities including PKCE and DPoP.
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"regexp"

	"github.com/auth-platform/sdk-go/src/errors"
)

const (
	// PKCEMethodS256 is the SHA-256 challenge method.
	PKCEMethodS256 = "S256"
	// PKCEMethodPlain is the plain challenge method (not recommended).
	PKCEMethodPlain = "plain"
	// DefaultVerifierLength is the default length for PKCE verifiers.
	DefaultVerifierLength = 64
	// MinVerifierLength is the minimum allowed verifier length per RFC 7636.
	MinVerifierLength = 43
	// MaxVerifierLength is the maximum allowed verifier length per RFC 7636.
	MaxVerifierLength = 128
)

// verifierCharset contains valid characters for PKCE verifiers per RFC 7636.
var verifierCharset = []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-._~")

// verifierRegex validates PKCE verifier format per RFC 7636.
var verifierRegex = regexp.MustCompile(`^[A-Za-z0-9._~-]{43,128}$`)

// PKCEPair holds a PKCE verifier and its corresponding challenge.
type PKCEPair struct {
	Verifier  string
	Challenge string
	Method    string
}

// PKCEGenerator defines the interface for PKCE generation.
type PKCEGenerator interface {
	GenerateVerifier() (string, error)
	ComputeChallenge(verifier string) string
}

// DefaultPKCEGenerator is the default PKCE generator implementation.
type DefaultPKCEGenerator struct {
	VerifierLength int
}

// NewPKCEGenerator creates a new PKCE generator with default settings.
func NewPKCEGenerator() *DefaultPKCEGenerator {
	return &DefaultPKCEGenerator{VerifierLength: DefaultVerifierLength}
}

// NewPKCEGeneratorWithLength creates a new PKCE generator with custom length.
func NewPKCEGeneratorWithLength(length int) (*DefaultPKCEGenerator, error) {
	if length < MinVerifierLength || length > MaxVerifierLength {
		return nil, errors.NewError(errors.ErrCodePKCEInvalid,
			"verifier length must be between 43 and 128")
	}
	return &DefaultPKCEGenerator{VerifierLength: length}, nil
}

// GenerateVerifier generates a cryptographically random PKCE verifier.
func (g *DefaultPKCEGenerator) GenerateVerifier() (string, error) {
	length := g.VerifierLength
	if length == 0 {
		length = DefaultVerifierLength
	}

	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", errors.WrapError(errors.ErrCodePKCEInvalid, "failed to generate random bytes", err)
	}

	// Map random bytes to valid verifier characters
	verifier := make([]byte, length)
	for i, b := range bytes {
		verifier[i] = verifierCharset[int(b)%len(verifierCharset)]
	}

	return string(verifier), nil
}

// ComputeChallenge computes the S256 challenge for a verifier.
func (g *DefaultPKCEGenerator) ComputeChallenge(verifier string) string {
	return ComputeChallenge(verifier)
}

// GenerateVerifier generates a cryptographically random PKCE verifier.
func GenerateVerifier() (string, error) {
	return NewPKCEGenerator().GenerateVerifier()
}

// GenerateVerifierWithLength generates a verifier with custom length.
func GenerateVerifierWithLength(length int) (string, error) {
	gen, err := NewPKCEGeneratorWithLength(length)
	if err != nil {
		return "", err
	}
	return gen.GenerateVerifier()
}

// ComputeChallenge computes the S256 challenge for a verifier.
// Uses SHA-256 hash and base64url encoding without padding.
func ComputeChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

// ComputeChallengeWithMethod computes the challenge using the specified method.
func ComputeChallengeWithMethod(verifier, method string) (string, error) {
	switch method {
	case PKCEMethodS256:
		return ComputeChallenge(verifier), nil
	case PKCEMethodPlain:
		return verifier, nil
	default:
		return "", errors.NewError(errors.ErrCodePKCEInvalid, "unsupported PKCE method")
	}
}

// VerifyPKCE verifies that a verifier matches a challenge (S256 method).
func VerifyPKCE(verifier, challenge string) bool {
	computed := ComputeChallenge(verifier)
	return computed == challenge
}

// VerifyPKCEWithMethod verifies using the specified method.
func VerifyPKCEWithMethod(verifier, challenge, method string) bool {
	switch method {
	case PKCEMethodS256:
		return VerifyPKCE(verifier, challenge)
	case PKCEMethodPlain:
		return verifier == challenge
	default:
		return false
	}
}

// ValidateVerifier validates a PKCE verifier per RFC 7636.
func ValidateVerifier(verifier string) error {
	if len(verifier) < MinVerifierLength {
		return errors.NewError(errors.ErrCodePKCEInvalid,
			"verifier too short, minimum 43 characters")
	}
	if len(verifier) > MaxVerifierLength {
		return errors.NewError(errors.ErrCodePKCEInvalid,
			"verifier too long, maximum 128 characters")
	}
	if !verifierRegex.MatchString(verifier) {
		return errors.NewError(errors.ErrCodePKCEInvalid,
			"verifier contains invalid characters")
	}
	return nil
}

// GeneratePKCE generates a complete PKCE pair (verifier + challenge).
func GeneratePKCE() (*PKCEPair, error) {
	verifier, err := GenerateVerifier()
	if err != nil {
		return nil, err
	}
	challenge := ComputeChallenge(verifier)
	return &PKCEPair{
		Verifier:  verifier,
		Challenge: challenge,
		Method:    PKCEMethodS256,
	}, nil
}

// GeneratePKCEWithLength generates a PKCE pair with custom verifier length.
func GeneratePKCEWithLength(length int) (*PKCEPair, error) {
	verifier, err := GenerateVerifierWithLength(length)
	if err != nil {
		return nil, err
	}
	challenge := ComputeChallenge(verifier)
	return &PKCEPair{
		Verifier:  verifier,
		Challenge: challenge,
		Method:    PKCEMethodS256,
	}, nil
}
