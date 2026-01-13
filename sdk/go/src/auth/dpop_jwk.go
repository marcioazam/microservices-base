package auth

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"

	"github.com/auth-platform/sdk-go/src/errors"
)

// ParseJWK parses a JWK map into a crypto.PublicKey.
func ParseJWK(jwkData map[string]interface{}) (crypto.PublicKey, error) {
	kty, ok := jwkData["kty"].(string)
	if !ok {
		return nil, errors.NewError(errors.ErrCodeDPoPInvalid, "missing kty in JWK")
	}

	switch kty {
	case "EC":
		return parseECJWK(jwkData)
	case "RSA":
		return parseRSAJWK(jwkData)
	default:
		return nil, errors.NewError(errors.ErrCodeDPoPInvalid, "unsupported key type: "+kty)
	}
}

func parseECJWK(jwkData map[string]interface{}) (*ecdsa.PublicKey, error) {
	crv, ok := jwkData["crv"].(string)
	if !ok || crv != "P-256" {
		return nil, errors.NewError(errors.ErrCodeDPoPInvalid, "unsupported curve, expected P-256")
	}

	xStr, ok := jwkData["x"].(string)
	if !ok {
		return nil, errors.NewError(errors.ErrCodeDPoPInvalid, "missing x coordinate in EC JWK")
	}
	yStr, ok := jwkData["y"].(string)
	if !ok {
		return nil, errors.NewError(errors.ErrCodeDPoPInvalid, "missing y coordinate in EC JWK")
	}

	xBytes, err := base64.RawURLEncoding.DecodeString(xStr)
	if err != nil {
		return nil, errors.WrapError(errors.ErrCodeDPoPInvalid, "invalid x coordinate encoding", err)
	}
	yBytes, err := base64.RawURLEncoding.DecodeString(yStr)
	if err != nil {
		return nil, errors.WrapError(errors.ErrCodeDPoPInvalid, "invalid y coordinate encoding", err)
	}

	return &ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     new(big.Int).SetBytes(xBytes),
		Y:     new(big.Int).SetBytes(yBytes),
	}, nil
}

func parseRSAJWK(jwkData map[string]interface{}) (*rsa.PublicKey, error) {
	nStr, ok := jwkData["n"].(string)
	if !ok {
		return nil, errors.NewError(errors.ErrCodeDPoPInvalid, "missing n in RSA JWK")
	}
	eStr, ok := jwkData["e"].(string)
	if !ok {
		return nil, errors.NewError(errors.ErrCodeDPoPInvalid, "missing e in RSA JWK")
	}

	nBytes, err := base64.RawURLEncoding.DecodeString(nStr)
	if err != nil {
		return nil, errors.WrapError(errors.ErrCodeDPoPInvalid, "invalid n encoding", err)
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(eStr)
	if err != nil {
		return nil, errors.WrapError(errors.ErrCodeDPoPInvalid, "invalid e encoding", err)
	}

	n := new(big.Int).SetBytes(nBytes)
	e := 0
	for _, b := range eBytes {
		e = e<<8 + int(b)
	}

	return &rsa.PublicKey{N: n, E: e}, nil
}

// DPoPConfirmation represents the cnf claim for DPoP binding.
type DPoPConfirmation struct {
	JWKThumbprint string `json:"jkt"`
}

// ComputeJWKThumbprint computes the JWK thumbprint per RFC 7638.
func ComputeJWKThumbprint(publicKey crypto.PublicKey) (string, error) {
	var thumbprintInput []byte

	switch k := publicKey.(type) {
	case *ecdsa.PublicKey:
		jwk := map[string]string{
			"crv": "P-256",
			"kty": "EC",
			"x":   base64.RawURLEncoding.EncodeToString(k.X.Bytes()),
			"y":   base64.RawURLEncoding.EncodeToString(k.Y.Bytes()),
		}
		var err error
		thumbprintInput, err = json.Marshal(jwk)
		if err != nil {
			return "", errors.WrapError(errors.ErrCodeDPoPInvalid, "failed to marshal JWK", err)
		}
	case *rsa.PublicKey:
		jwk := map[string]string{
			"e":   base64.RawURLEncoding.EncodeToString([]byte{1, 0, 1}),
			"kty": "RSA",
			"n":   base64.RawURLEncoding.EncodeToString(k.N.Bytes()),
		}
		var err error
		thumbprintInput, err = json.Marshal(jwk)
		if err != nil {
			return "", errors.WrapError(errors.ErrCodeDPoPInvalid, "failed to marshal JWK", err)
		}
	default:
		return "", errors.NewError(errors.ErrCodeDPoPInvalid, "unsupported key type for thumbprint")
	}

	hash := sha256.Sum256(thumbprintInput)
	return base64.RawURLEncoding.EncodeToString(hash[:]), nil
}
