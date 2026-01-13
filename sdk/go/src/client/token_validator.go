package client

import (
	"context"

	"github.com/auth-platform/sdk-go/src/token"
	"github.com/auth-platform/sdk-go/src/types"
)

// ValidateToken implements middleware.TokenValidator interface.
func (c *Client) ValidateToken(tokenStr string, audience string) (*token.ValidationResult, error) {
	opts := types.ValidationOptions{
		Audience: audience,
	}

	claims, err := c.jwksCache.ValidateTokenWithOpts(context.Background(), tokenStr, opts)
	if err != nil {
		return nil, err
	}

	return token.NewValidationResult(claims, tokenStr, token.SchemeBearer), nil
}

// TokenResponse represents an OAuth token response.
type TokenResponse = types.TokenResponse

// Claims represents JWT claims.
type Claims = types.Claims
