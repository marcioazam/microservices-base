package authplatform

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// DPoPProver generates and validates DPoP proofs.
type DPoPProver interface {
	// GenerateProof generates a DPoP proof for a request.
	GenerateProof(ctx context.Context, method, uri string, accessToken string) (string, error)
	// ValidateProof validates a DPoP proof.
	ValidateProof(ctx context.Context, proof string, method, uri string) (*DPoPClaims, error)
}

// DPoPClaims represents the claims in a DPoP proof JWT.
type DPoPClaims struct {
	jwt.RegisteredClaims
	HTTPMethod      string `json:"htm"`
	HTTPUri         string `json:"htu"`
	AccessTokenHash string `json:"ath,omitempty"`
}

// DPoPKeyPair holds a key pair for DPoP signing.
type DPoPKeyPair struct {
	PrivateKey crypto.Signer
	PublicKey  crypto.PublicKey
	Algorithm  string
	KeyID      string
}

// DefaultDPoPProver is the default implementation of DPoPProver.
type DefaultDPoPProver struct {
	keyPair *DPoPKeyPair
}

// NewDPoPProver creates a new DPoP prover with the given key pair.
func NewDPoPProver(keyPair *DPoPKeyPair) *DefaultDPoPProver {
	return &DefaultDPoPProver{keyPair: keyPair}
}


// GenerateES256KeyPair generates a new ES256 (P-256) key pair for DPoP.
func GenerateES256KeyPair() (*DPoPKeyPair, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, &SDKError{
			Code:    ErrCodeDPoPInvalid,
			Message: "failed to generate ES256 key pair",
			Cause:   err,
		}
	}

	keyID, err := generateKeyID(&privateKey.PublicKey)
	if err != nil {
		return nil, err
	}

	return &DPoPKeyPair{
		PrivateKey: privateKey,
		PublicKey:  &privateKey.PublicKey,
		Algorithm:  "ES256",
		KeyID:      keyID,
	}, nil
}

// GenerateRS256KeyPair generates a new RS256 (RSA 2048) key pair for DPoP.
func GenerateRS256KeyPair() (*DPoPKeyPair, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, &SDKError{
			Code:    ErrCodeDPoPInvalid,
			Message: "failed to generate RS256 key pair",
			Cause:   err,
		}
	}

	keyID, err := generateKeyID(&privateKey.PublicKey)
	if err != nil {
		return nil, err
	}

	return &DPoPKeyPair{
		PrivateKey: privateKey,
		PublicKey:  &privateKey.PublicKey,
		Algorithm:  "RS256",
		KeyID:      keyID,
	}, nil
}

func generateKeyID(publicKey crypto.PublicKey) (string, error) {
	// Generate a thumbprint of the public key
	var keyBytes []byte
	switch k := publicKey.(type) {
	case *ecdsa.PublicKey:
		keyBytes = elliptic.Marshal(k.Curve, k.X, k.Y)
	case *rsa.PublicKey:
		keyBytes = k.N.Bytes()
	default:
		return "", &SDKError{
			Code:    ErrCodeDPoPInvalid,
			Message: "unsupported key type",
		}
	}

	hash := sha256.Sum256(keyBytes)
	return base64.RawURLEncoding.EncodeToString(hash[:16]), nil
}

// GenerateProof generates a DPoP proof JWT for the given HTTP method and URI.
func (p *DefaultDPoPProver) GenerateProof(ctx context.Context, method, uri string, accessToken string) (string, error) {
	if p.keyPair == nil {
		return "", &SDKError{
			Code:    ErrCodeDPoPRequired,
			Message: "DPoP key pair not configured",
		}
	}

	// Generate unique JTI
	jtiBytes := make([]byte, 16)
	if _, err := rand.Read(jtiBytes); err != nil {
		return "", &SDKError{
			Code:    ErrCodeDPoPInvalid,
			Message: "failed to generate JTI",
			Cause:   err,
		}
	}
	jti := base64.RawURLEncoding.EncodeToString(jtiBytes)

	now := time.Now()
	claims := DPoPClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ID:       jti,
			IssuedAt: jwt.NewNumericDate(now),
		},
		HTTPMethod: method,
		HTTPUri:    uri,
	}

	// Add access token hash if provided
	if accessToken != "" {
		claims.AccessTokenHash = computeATH(accessToken)
	}

	// Create token with appropriate signing method
	var signingMethod jwt.SigningMethod
	switch p.keyPair.Algorithm {
	case "ES256":
		signingMethod = jwt.SigningMethodES256
	case "RS256":
		signingMethod = jwt.SigningMethodRS256
	default:
		return "", &SDKError{
			Code:    ErrCodeDPoPInvalid,
			Message: fmt.Sprintf("unsupported algorithm: %s", p.keyPair.Algorithm),
		}
	}

	token := jwt.NewWithClaims(signingMethod, claims)
	token.Header["typ"] = "dpop+jwt"

	// Add JWK thumbprint
	token.Header["jwk"] = p.buildJWK()

	signedToken, err := token.SignedString(p.keyPair.PrivateKey)
	if err != nil {
		return "", &SDKError{
			Code:    ErrCodeDPoPInvalid,
			Message: "failed to sign DPoP proof",
			Cause:   err,
		}
	}

	return signedToken, nil
}


func (p *DefaultDPoPProver) buildJWK() map[string]interface{} {
	jwk := map[string]interface{}{
		"kty": "",
		"kid": p.keyPair.KeyID,
	}

	switch k := p.keyPair.PublicKey.(type) {
	case *ecdsa.PublicKey:
		jwk["kty"] = "EC"
		jwk["crv"] = "P-256"
		jwk["x"] = base64.RawURLEncoding.EncodeToString(k.X.Bytes())
		jwk["y"] = base64.RawURLEncoding.EncodeToString(k.Y.Bytes())
	case *rsa.PublicKey:
		jwk["kty"] = "RSA"
		jwk["n"] = base64.RawURLEncoding.EncodeToString(k.N.Bytes())
		jwk["e"] = base64.RawURLEncoding.EncodeToString([]byte{1, 0, 1}) // 65537
	}

	return jwk
}

// computeATH computes the access token hash (ath) claim.
func computeATH(accessToken string) string {
	hash := sha256.Sum256([]byte(accessToken))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

// ValidateProof validates a DPoP proof JWT.
func (p *DefaultDPoPProver) ValidateProof(ctx context.Context, proof string, method, uri string) (*DPoPClaims, error) {
	// Parse the token without verification first to get the header
	token, _, err := jwt.NewParser().ParseUnverified(proof, &DPoPClaims{})
	if err != nil {
		return nil, &SDKError{
			Code:    ErrCodeDPoPInvalid,
			Message: "failed to parse DPoP proof",
			Cause:   err,
		}
	}

	// Verify typ header
	if typ, ok := token.Header["typ"].(string); !ok || typ != "dpop+jwt" {
		return nil, &SDKError{
			Code:    ErrCodeDPoPInvalid,
			Message: "invalid DPoP proof type",
		}
	}

	// Extract JWK from header
	jwkData, ok := token.Header["jwk"].(map[string]interface{})
	if !ok {
		return nil, &SDKError{
			Code:    ErrCodeDPoPInvalid,
			Message: "missing JWK in DPoP proof header",
		}
	}

	// Parse the public key from JWK
	publicKey, err := parseJWK(jwkData)
	if err != nil {
		return nil, err
	}

	// Now verify the signature
	var signingMethod jwt.SigningMethod
	switch token.Method.Alg() {
	case "ES256":
		signingMethod = jwt.SigningMethodES256
	case "RS256":
		signingMethod = jwt.SigningMethodRS256
	default:
		return nil, &SDKError{
			Code:    ErrCodeDPoPInvalid,
			Message: fmt.Sprintf("unsupported algorithm: %s", token.Method.Alg()),
		}
	}

	// Parse and verify
	verifiedToken, err := jwt.ParseWithClaims(proof, &DPoPClaims{}, func(t *jwt.Token) (interface{}, error) {
		if t.Method.Alg() != signingMethod.Alg() {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return publicKey, nil
	})
	if err != nil {
		return nil, &SDKError{
			Code:    ErrCodeDPoPInvalid,
			Message: "DPoP proof signature verification failed",
			Cause:   err,
		}
	}

	claims, ok := verifiedToken.Claims.(*DPoPClaims)
	if !ok {
		return nil, &SDKError{
			Code:    ErrCodeDPoPInvalid,
			Message: "invalid DPoP claims",
		}
	}

	// Verify method and URI
	if claims.HTTPMethod != method {
		return nil, &SDKError{
			Code:    ErrCodeDPoPInvalid,
			Message: fmt.Sprintf("HTTP method mismatch: expected %s, got %s", method, claims.HTTPMethod),
		}
	}
	if claims.HTTPUri != uri {
		return nil, &SDKError{
			Code:    ErrCodeDPoPInvalid,
			Message: fmt.Sprintf("HTTP URI mismatch: expected %s, got %s", uri, claims.HTTPUri),
		}
	}

	// Verify JTI is present
	if claims.ID == "" {
		return nil, &SDKError{
			Code:    ErrCodeDPoPInvalid,
			Message: "missing JTI in DPoP proof",
		}
	}

	// Verify iat is present and not too old (5 minute window)
	if claims.IssuedAt == nil {
		return nil, &SDKError{
			Code:    ErrCodeDPoPInvalid,
			Message: "missing iat in DPoP proof",
		}
	}
	if time.Since(claims.IssuedAt.Time) > 5*time.Minute {
		return nil, &SDKError{
			Code:    ErrCodeDPoPInvalid,
			Message: "DPoP proof expired",
		}
	}

	return claims, nil
}

// VerifyATH verifies that the access token hash matches the expected value.
func VerifyATH(accessToken, expectedATH string) bool {
	computed := computeATH(accessToken)
	return computed == expectedATH
}
