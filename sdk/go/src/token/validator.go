package token

import (
	"context"
	"fmt"

	"github.com/auth-platform/sdk-go/src/errors"
	"github.com/auth-platform/sdk-go/src/types"
	"github.com/lestrrat-go/jwx/v3/jwt"
)

// ValidateToken validates a JWT using cached JWKS.
func (c *JWKSCache) ValidateToken(ctx context.Context, token string, audience string) (*types.Claims, error) {
	keySet, err := c.getKeySet(ctx)
	if err != nil {
		c.recordError()
		return nil, errors.WrapError(errors.ErrCodeValidation, "failed to fetch JWKS", err)
	}

	parseOpts := []jwt.ParseOption{
		jwt.WithKeySet(keySet),
		jwt.WithValidate(true),
	}

	if audience != "" {
		parseOpts = append(parseOpts, jwt.WithAudience(audience))
	}

	parsed, err := jwt.Parse([]byte(token), parseOpts...)
	if err != nil {
		return nil, errors.WrapError(errors.ErrCodeTokenInvalid, "token validation failed", err)
	}

	return extractClaims(parsed)
}

// ValidateTokenWithOpts validates a JWT with custom validation options.
func (c *JWKSCache) ValidateTokenWithOpts(ctx context.Context, token string, opts types.ValidationOptions) (*types.Claims, error) {
	keySet, err := c.getKeySet(ctx)
	if err != nil {
		c.recordError()
		return nil, errors.WrapError(errors.ErrCodeValidation, "failed to fetch JWKS", err)
	}

	parseOpts := []jwt.ParseOption{
		jwt.WithKeySet(keySet),
	}

	if !opts.SkipExpiry {
		parseOpts = append(parseOpts, jwt.WithValidate(true))
	}

	if opts.Audience != "" {
		parseOpts = append(parseOpts, jwt.WithAudience(opts.Audience))
	}

	if opts.Issuer != "" {
		parseOpts = append(parseOpts, jwt.WithIssuer(opts.Issuer))
	}

	parsed, err := jwt.Parse([]byte(token), parseOpts...)
	if err != nil {
		return nil, errors.WrapError(errors.ErrCodeTokenInvalid, "token validation failed", err)
	}

	for _, claim := range opts.RequiredClaims {
		var v interface{}
		if err := parsed.Get(claim, &v); err != nil {
			return nil, errors.NewError(errors.ErrCodeTokenInvalid,
				fmt.Sprintf("missing required claim: %s", claim))
		}
	}

	return extractClaims(parsed)
}

func extractClaims(parsed jwt.Token) (*types.Claims, error) {
	claims := &types.Claims{}

	if sub, ok := parsed.Subject(); ok {
		claims.Subject = sub
	}
	if iss, ok := parsed.Issuer(); ok {
		claims.Issuer = iss
	}
	if exp, ok := parsed.Expiration(); ok && !exp.IsZero() {
		claims.ExpiresAt = exp.Unix()
	}
	if iat, ok := parsed.IssuedAt(); ok && !iat.IsZero() {
		claims.IssuedAt = iat.Unix()
	}
	if aud, ok := parsed.Audience(); ok && len(aud) > 0 {
		claims.Audience = aud
	}

	var scope string
	if err := parsed.Get("scope", &scope); err == nil {
		claims.Scope = scope
	}
	var clientID string
	if err := parsed.Get("client_id", &clientID); err == nil {
		claims.ClientID = clientID
	}

	return claims, nil
}
