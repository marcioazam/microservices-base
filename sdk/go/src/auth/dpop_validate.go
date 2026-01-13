package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/auth-platform/sdk-go/src/errors"
	"github.com/golang-jwt/jwt/v5"
)

// DPoPProofMaxAge is the maximum age for a DPoP proof (5 minutes per spec).
const DPoPProofMaxAge = 5 * time.Minute

// ValidateProof validates a DPoP proof JWT.
func (p *DefaultDPoPProver) ValidateProof(ctx context.Context, proof string, method, uri string) (*DPoPClaims, error) {
	token, _, err := jwt.NewParser().ParseUnverified(proof, &DPoPClaims{})
	if err != nil {
		return nil, errors.WrapError(errors.ErrCodeDPoPInvalid, "failed to parse DPoP proof", err)
	}

	if typ, ok := token.Header["typ"].(string); !ok || typ != "dpop+jwt" {
		return nil, errors.NewError(errors.ErrCodeDPoPInvalid, "invalid DPoP proof type")
	}

	jwkData, ok := token.Header["jwk"].(map[string]interface{})
	if !ok {
		return nil, errors.NewError(errors.ErrCodeDPoPInvalid, "missing JWK in DPoP proof header")
	}

	publicKey, err := ParseJWK(jwkData)
	if err != nil {
		return nil, err
	}

	var signingMethod jwt.SigningMethod
	switch token.Method.Alg() {
	case "ES256":
		signingMethod = jwt.SigningMethodES256
	case "RS256":
		signingMethod = jwt.SigningMethodRS256
	default:
		return nil, errors.NewError(errors.ErrCodeDPoPInvalid, fmt.Sprintf("unsupported algorithm: %s", token.Method.Alg()))
	}

	verifiedToken, err := jwt.ParseWithClaims(proof, &DPoPClaims{}, func(t *jwt.Token) (interface{}, error) {
		if t.Method.Alg() != signingMethod.Alg() {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return publicKey, nil
	})
	if err != nil {
		return nil, errors.WrapError(errors.ErrCodeDPoPInvalid, "DPoP proof signature verification failed", err)
	}

	claims, ok := verifiedToken.Claims.(*DPoPClaims)
	if !ok {
		return nil, errors.NewError(errors.ErrCodeDPoPInvalid, "invalid DPoP claims")
	}

	if claims.HTTPMethod != method {
		return nil, errors.NewError(errors.ErrCodeDPoPInvalid,
			fmt.Sprintf("HTTP method mismatch: expected %s, got %s", method, claims.HTTPMethod))
	}
	if claims.HTTPUri != uri {
		return nil, errors.NewError(errors.ErrCodeDPoPInvalid,
			fmt.Sprintf("HTTP URI mismatch: expected %s, got %s", uri, claims.HTTPUri))
	}

	if claims.ID == "" {
		return nil, errors.NewError(errors.ErrCodeDPoPInvalid, "missing JTI in DPoP proof")
	}

	if claims.IssuedAt == nil {
		return nil, errors.NewError(errors.ErrCodeDPoPInvalid, "missing iat in DPoP proof")
	}
	if time.Since(claims.IssuedAt.Time) > DPoPProofMaxAge {
		return nil, errors.NewError(errors.ErrCodeDPoPInvalid, "DPoP proof expired")
	}

	return claims, nil
}

// ValidateProofWithATH validates a DPoP proof and verifies the access token hash.
func (p *DefaultDPoPProver) ValidateProofWithATH(ctx context.Context, proof, method, uri, accessToken string) (*DPoPClaims, error) {
	claims, err := p.ValidateProof(ctx, proof, method, uri)
	if err != nil {
		return nil, err
	}

	if claims.AccessTokenHash != "" && accessToken != "" {
		if !VerifyATH(accessToken, claims.AccessTokenHash) {
			return nil, errors.NewError(errors.ErrCodeDPoPInvalid, "access token hash mismatch")
		}
	}

	return claims, nil
}
