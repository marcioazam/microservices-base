// Package property contains property-based tests for the cache service.
// Feature: cache-microservice
package property

import (
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"

	"github.com/auth-platform/cache-service/internal/auth"
)

// Property 10: JWT Authentication
// For any request, if the JWT token is valid and not expired, the request SHALL be processed;
// if invalid or expired, a 401 response SHALL be returned.
// Validates: Requirements 5.1, 5.2
func TestProperty10_JWTAuthentication(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = PropertyTestIterations
	parameters.Rng.Seed(PropertyTestSeed)

	properties := gopter.NewProperties(parameters)

	secret := "test-secret-key-for-jwt-signing"
	issuer := "cache-service"
	validator := auth.NewJWTValidator(secret, issuer)

	properties.Property("Valid token is accepted", prop.ForAll(
		func(namespace string, scopes []string) bool {
			if namespace == "" {
				return true
			}

			// Generate valid token
			token, err := validator.GenerateToken(namespace, scopes, time.Hour)
			if err != nil {
				return false
			}

			// Validate token
			claims, err := validator.Validate(token)
			if err != nil {
				return false
			}

			// Verify claims
			return claims.Namespace == namespace
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) < 50 }),
		gen.SliceOf(gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) < 20 })),
	))

	properties.Property("Expired token is rejected", prop.ForAll(
		func(namespace string) bool {
			if namespace == "" {
				return true
			}

			// Generate expired token
			token, err := validator.GenerateToken(namespace, nil, -time.Hour)
			if err != nil {
				return false
			}

			// Validate token should fail
			_, err = validator.Validate(token)
			return err == auth.ErrExpiredToken
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) < 50 }),
	))

	properties.Property("Invalid token is rejected", prop.ForAll(
		func(invalidToken string) bool {
			if invalidToken == "" {
				return true
			}

			// Validate invalid token should fail
			_, err := validator.Validate(invalidToken)
			return err != nil
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) < 100 }),
	))

	properties.Property("Token with wrong secret is rejected", prop.ForAll(
		func(namespace string) bool {
			if namespace == "" {
				return true
			}

			// Generate token with different secret
			wrongValidator := auth.NewJWTValidator("wrong-secret", issuer)
			token, err := wrongValidator.GenerateToken(namespace, nil, time.Hour)
			if err != nil {
				return false
			}

			// Validate with correct validator should fail
			_, err = validator.Validate(token)
			return err == auth.ErrInvalidToken
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) < 50 }),
	))

	properties.Property("Token with Bearer prefix is accepted", prop.ForAll(
		func(namespace string) bool {
			if namespace == "" {
				return true
			}

			// Generate valid token
			token, err := validator.GenerateToken(namespace, nil, time.Hour)
			if err != nil {
				return false
			}

			// Validate with Bearer prefix
			claims, err := validator.Validate("Bearer " + token)
			if err != nil {
				return false
			}

			return claims.Namespace == namespace
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) < 50 }),
	))

	properties.Property("Empty token is rejected", prop.ForAll(
		func(dummy int) bool {
			_, err := validator.Validate("")
			return err == auth.ErrMissingToken
		},
		gen.Const(0),
	))

	properties.TestingRun(t)
}
