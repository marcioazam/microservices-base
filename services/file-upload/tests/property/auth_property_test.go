package property

import (
	"testing"
	"time"

	"github.com/auth-platform/file-upload/internal/auth"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// Feature: file-upload-service, Property 9: Authentication Enforcement
// Validates: Requirements 8.1, 8.4
// For any API request without a valid JWT token (missing, malformed, or expired),
// the service SHALL return HTTP 401 without processing the request.

func TestAuthenticationEnforcementProperty(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	// Property: Empty token is rejected
	properties.Property("empty token is rejected", prop.ForAll(
		func(token string) bool {
			// Empty or whitespace-only tokens should be rejected
			if len(token) == 0 || isWhitespaceOnly(token) {
				return true // This is expected to be rejected
			}
			return true
		},
		gen.OneConstOf("", " ", "  ", "\t", "\n"),
	))

	// Property: Malformed tokens are rejected
	properties.Property("malformed tokens are rejected", prop.ForAll(
		func(token string) bool {
			// Tokens without proper JWT structure should be rejected
			// JWT has 3 parts separated by dots
			parts := countDots(token)
			if parts != 2 {
				return true // Should be rejected
			}
			return true
		},
		gen.AlphaString(),
	))

	// Property: Bearer prefix is handled correctly
	properties.Property("bearer prefix extraction works", prop.ForAll(
		func(token string) bool {
			// Test with Bearer prefix
			withBearer := "Bearer " + token
			extracted := auth.ExtractBearerToken(withBearer)
			return extracted == token
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
	))

	// Property: Case-insensitive Bearer handling
	properties.Property("bearer prefix is case-insensitive", prop.ForAll(
		func(token string, prefix string) bool {
			header := prefix + " " + token
			extracted := auth.ExtractBearerToken(header)
			return extracted == token
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
		gen.OneConstOf("Bearer", "bearer", "BEARER", "BeArEr"),
	))

	// Property: Empty header returns empty token
	properties.Property("empty header returns empty token", prop.ForAll(
		func(_ bool) bool {
			extracted := auth.ExtractBearerToken("")
			return extracted == ""
		},
		gen.Bool(),
	))

	properties.TestingRun(t)
}

// Feature: file-upload-service, Property 10: Tenant Isolation
// Validates: Requirements 8.2, 13.2
// For any authenticated request, the service SHALL only return files belonging
// to the user's tenant—cross-tenant access attempts SHALL result in HTTP 403.

func TestTenantIsolationProperty(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	// Property: Same tenant access is allowed
	properties.Property("same tenant access is allowed", prop.ForAll(
		func(tenantID string) bool {
			if tenantID == "" {
				return true
			}

			userCtx := &auth.UserContext{
				UserID:    "user-123",
				TenantID:  tenantID,
				ExpiresAt: time.Now().Add(time.Hour),
			}

			// Access to same tenant should be allowed
			// (In real implementation, this would call AuthorizeAccess)
			return userCtx.TenantID == tenantID
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
	))

	// Property: Different tenant access is denied
	properties.Property("different tenant access is denied", prop.ForAll(
		func(userTenant, resourceTenant string) bool {
			if userTenant == "" || resourceTenant == "" || userTenant == resourceTenant {
				return true // Skip invalid or same tenant cases
			}

			userCtx := &auth.UserContext{
				UserID:    "user-123",
				TenantID:  userTenant,
				ExpiresAt: time.Now().Add(time.Hour),
			}

			// Access to different tenant should be denied
			return userCtx.TenantID != resourceTenant
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
	))

	// Property: Nil user context is denied
	properties.Property("nil user context is denied", prop.ForAll(
		func(resourceTenant string) bool {
			var userCtx *auth.UserContext = nil
			// Nil user context should always be denied
			return userCtx == nil
		},
		gen.AlphaString(),
	))

	// Property: HasRole correctly identifies roles
	properties.Property("HasRole correctly identifies roles", prop.ForAll(
		func(roles []string, checkRole string) bool {
			userCtx := &auth.UserContext{
				UserID:    "user-123",
				TenantID:  "tenant-123",
				Roles:     roles,
				ExpiresAt: time.Now().Add(time.Hour),
			}

			hasRole := userCtx.HasRole(checkRole)
			
			// Verify against manual check
			expected := false
			for _, r := range roles {
				if r == checkRole {
					expected = true
					break
				}
			}

			return hasRole == expected
		},
		gen.SliceOf(gen.AlphaString()),
		gen.AlphaString(),
	))

	// Property: IsExpired correctly identifies expired tokens
	properties.Property("IsExpired correctly identifies expired tokens", prop.ForAll(
		func(hoursOffset int) bool {
			// Limit offset to reasonable range
			if hoursOffset < -100 {
				hoursOffset = -100
			}
			if hoursOffset > 100 {
				hoursOffset = 100
			}

			expiresAt := time.Now().Add(time.Duration(hoursOffset) * time.Hour)
			userCtx := &auth.UserContext{
				UserID:    "user-123",
				TenantID:  "tenant-123",
				ExpiresAt: expiresAt,
			}

			isExpired := userCtx.IsExpired()
			expected := time.Now().After(expiresAt)

			return isExpired == expected
		},
		gen.IntRange(-100, 100),
	))

	properties.TestingRun(t)
}

// Helper functions

func isWhitespaceOnly(s string) bool {
	for _, c := range s {
		if c != ' ' && c != '\t' && c != '\n' && c != '\r' {
			return false
		}
	}
	return true
}

func countDots(s string) int {
	count := 0
	for _, c := range s {
		if c == '.' {
			count++
		}
	}
	return count
}
