package auth

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

	"github.com/auth-platform/sdk-go/src/errors"
	"github.com/golang-jwt/jwt/v5"
)

// DPoPProver generates and validates DPoP proofs.
type DPoPProver interface {
	GenerateProof(ctx context.Context, method, uri string, accessToken string) (string, error)
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
		return nil, errors.WrapError(errors.ErrCodeDPoPInvalid, "failed to generate ES256 key pair", err)
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
		return nil, errors.WrapError(errors.ErrCodeDPoPInvalid, "failed to generate RS256 key pair", err)
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
	var keyBytes []byte
	switch k := publicKey.(type) {
	case *ecdsa.PublicKey:
		keyBytes = elliptic.Marshal(k.Curve, k.X, k.Y)
	case *rsa.PublicKey:
		keyBytes = k.N.Bytes()
	default:
		return "", errors.NewError(errors.ErrCodeDPoPInvalid, "unsupported key type")
	}

	hash := sha256.Sum256(keyBytes)
	return base64.RawURLEncoding.EncodeToString(hash[:16]), nil
}

// GenerateProof generates a DPoP proof JWT for the given HTTP method and URI.
func (p *DefaultDPoPProver) GenerateProof(ctx context.Context, method, uri string, accessToken string) (string, error) {
	if p.keyPair == nil {
		return "", errors.NewError(errors.ErrCodeDPoPRequired, "DPoP key pair not configured")
	}

	jtiBytes := make([]byte, 16)
	if _, err := rand.Read(jtiBytes); err != nil {
		return "", errors.WrapError(errors.ErrCodeDPoPInvalid, "failed to generate JTI", err)
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

	if accessToken != "" {
		claims.AccessTokenHash = ComputeATH(accessToken)
	}

	var signingMethod jwt.SigningMethod
	switch p.keyPair.Algorithm {
	case "ES256":
		signingMethod = jwt.SigningMethodES256
	case "RS256":
		signingMethod = jwt.SigningMethodRS256
	default:
		return "", errors.NewError(errors.ErrCodeDPoPInvalid, fmt.Sprintf("unsupported algorithm: %s", p.keyPair.Algorithm))
	}

	token := jwt.NewWithClaims(signingMethod, claims)
	token.Header["typ"] = "dpop+jwt"
	token.Header["jwk"] = p.buildJWK()

	signedToken, err := token.SignedString(p.keyPair.PrivateKey)
	if err != nil {
		return "", errors.WrapError(errors.ErrCodeDPoPInvalid, "failed to sign DPoP proof", err)
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
		jwk["e"] = base64.RawURLEncoding.EncodeToString([]byte{1, 0, 1})
	}

	return jwk
}

// ComputeATH computes the access token hash (ath) claim.
func ComputeATH(accessToken string) string {
	hash := sha256.Sum256([]byte(accessToken))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

// VerifyATH verifies that the access token hash matches the expected value.
func VerifyATH(accessToken, expectedATH string) bool {
	computed := ComputeATH(accessToken)
	return computed == expectedATH
}
