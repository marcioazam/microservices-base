package authplatform

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
)

// parseJWK parses a JWK map into a crypto.PublicKey.
func parseJWK(jwkData map[string]interface{}) (crypto.PublicKey, error) {
	kty, ok := jwkData["kty"].(string)
	if !ok {
		return nil, &SDKError{
			Code:    ErrCodeDPoPInvalid,
			Message: "missing kty in JWK",
		}
	}

	switch kty {
	case "EC":
		return parseECJWK(jwkData)
	case "RSA":
		return parseRSAJWK(jwkData)
	default:
		return nil, &SDKError{
			Code:    ErrCodeDPoPInvalid,
			Message: fmt.Sprintf("unsupported key type: %s", kty),
		}
	}
}

func parseECJWK(jwkData map[string]interface{}) (*ecdsa.PublicKey, error) {
	crv, ok := jwkData["crv"].(string)
	if !ok || crv != "P-256" {
		return nil, &SDKError{
			Code:    ErrCodeDPoPInvalid,
			Message: "unsupported curve, expected P-256",
		}
	}

	xStr, ok := jwkData["x"].(string)
	if !ok {
		return nil, &SDKError{
			Code:    ErrCodeDPoPInvalid,
			Message: "missing x coordinate in EC JWK",
		}
	}
	yStr, ok := jwkData["y"].(string)
	if !ok {
		return nil, &SDKError{
			Code:    ErrCodeDPoPInvalid,
			Message: "missing y coordinate in EC JWK",
		}
	}

	xBytes, err := base64.RawURLEncoding.DecodeString(xStr)
	if err != nil {
		return nil, &SDKError{
			Code:    ErrCodeDPoPInvalid,
			Message: "invalid x coordinate encoding",
			Cause:   err,
		}
	}
	yBytes, err := base64.RawURLEncoding.DecodeString(yStr)
	if err != nil {
		return nil, &SDKError{
			Code:    ErrCodeDPoPInvalid,
			Message: "invalid y coordinate encoding",
			Cause:   err,
		}
	}

	curve := elliptic.P256()
	x, y := elliptic.Unmarshal(curve, append([]byte{0x04}, append(xBytes, yBytes...)...))
	if x == nil {
		// Try direct assignment if unmarshal fails
		return &ecdsa.PublicKey{
			Curve: curve,
			X:     new(big.Int).SetBytes(xBytes),
			Y:     new(big.Int).SetBytes(yBytes),
		}, nil
	}

	return &ecdsa.PublicKey{
		Curve: curve,
		X:     x,
		Y:     y,
	}, nil
}

func parseRSAJWK(jwkData map[string]interface{}) (*rsa.PublicKey, error) {
	nStr, ok := jwkData["n"].(string)
	if !ok {
		return nil, &SDKError{
			Code:    ErrCodeDPoPInvalid,
			Message: "missing n in RSA JWK",
		}
	}
	eStr, ok := jwkData["e"].(string)
	if !ok {
		return nil, &SDKError{
			Code:    ErrCodeDPoPInvalid,
			Message: "missing e in RSA JWK",
		}
	}

	nBytes, err := base64.RawURLEncoding.DecodeString(nStr)
	if err != nil {
		return nil, &SDKError{
			Code:    ErrCodeDPoPInvalid,
			Message: "invalid n encoding",
			Cause:   err,
		}
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(eStr)
	if err != nil {
		return nil, &SDKError{
			Code:    ErrCodeDPoPInvalid,
			Message: "invalid e encoding",
			Cause:   err,
		}
	}

	n := new(big.Int).SetBytes(nBytes)
	e := 0
	for _, b := range eBytes {
		e = e<<8 + int(b)
	}

	return &rsa.PublicKey{
		N: n,
		E: e,
	}, nil
}

// DPoPConfirmation represents the cnf claim for DPoP binding.
type DPoPConfirmation struct {
	JWKThumbprint string `json:"jkt"`
}

// ComputeJWKThumbprint computes the JWK thumbprint for DPoP binding.
func ComputeJWKThumbprint(publicKey crypto.PublicKey) (string, error) {
	var thumbprintInput []byte

	switch k := publicKey.(type) {
	case *ecdsa.PublicKey:
		// Canonical JWK representation for EC keys
		jwk := map[string]string{
			"crv": "P-256",
			"kty": "EC",
			"x":   base64.RawURLEncoding.EncodeToString(k.X.Bytes()),
			"y":   base64.RawURLEncoding.EncodeToString(k.Y.Bytes()),
		}
		var err error
		thumbprintInput, err = json.Marshal(jwk)
		if err != nil {
			return "", err
		}
	case *rsa.PublicKey:
		// Canonical JWK representation for RSA keys
		jwk := map[string]string{
			"e":   base64.RawURLEncoding.EncodeToString([]byte{1, 0, 1}),
			"kty": "RSA",
			"n":   base64.RawURLEncoding.EncodeToString(k.N.Bytes()),
		}
		var err error
		thumbprintInput, err = json.Marshal(jwk)
		if err != nil {
			return "", err
		}
	default:
		return "", &SDKError{
			Code:    ErrCodeDPoPInvalid,
			Message: "unsupported key type for thumbprint",
		}
	}

	hash := sha256.Sum256(thumbprintInput)
	return base64.RawURLEncoding.EncodeToString(hash[:]), nil
}
