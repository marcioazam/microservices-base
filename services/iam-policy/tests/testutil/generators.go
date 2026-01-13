// Package testutil provides test utilities and generators for IAM Policy Service.
package testutil

import (
	"pgregory.net/rapid"
)

// AuthorizationInputGen generates random authorization inputs for property testing.
func AuthorizationInputGen() *rapid.Generator[map[string]interface{}] {
	return rapid.Custom(func(t *rapid.T) map[string]interface{} {
		subjectID := rapid.String().Draw(t, "subjectID")
		resourceType := rapid.SampledFrom([]string{"document", "user", "project", "api"}).Draw(t, "resourceType")
		resourceID := rapid.String().Draw(t, "resourceID")
		action := rapid.SampledFrom([]string{"read", "write", "delete", "create", "update"}).Draw(t, "action")
		role := rapid.SampledFrom([]string{"admin", "editor", "viewer", "guest"}).Draw(t, "role")

		return map[string]interface{}{
			"subject": map[string]interface{}{
				"id": subjectID,
				"attributes": map[string]interface{}{
					"role":  role,
					"roles": []string{role},
				},
			},
			"resource": map[string]interface{}{
				"type": resourceType,
				"id":   resourceID,
				"attributes": map[string]interface{}{
					"owner_id": subjectID,
				},
			},
			"action":      action,
			"environment": map[string]interface{}{},
		}
	})
}

// RoleGen generates random roles for property testing.
func RoleGen() *rapid.Generator[Role] {
	return rapid.Custom(func(t *rapid.T) Role {
		return Role{
			ID:          rapid.StringMatching(`^[a-z]{3,10}$`).Draw(t, "roleID"),
			Name:        rapid.String().Draw(t, "roleName"),
			Description: rapid.String().Draw(t, "roleDesc"),
			ParentID:    "",
			Permissions: rapid.SliceOfN(rapid.SampledFrom([]string{"read", "write", "delete", "create"}), 1, 4).Draw(t, "permissions"),
		}
	})
}

// Role represents a role for testing.
type Role struct {
	ID          string
	Name        string
	Description string
	ParentID    string
	Permissions []string
}

// NonEmptyStringGen generates non-empty strings.
func NonEmptyStringGen() *rapid.Generator[string] {
	return rapid.StringMatching(`^[a-zA-Z0-9_-]{1,50}$`)
}

// CorrelationIDGen generates valid correlation IDs.
func CorrelationIDGen() *rapid.Generator[string] {
	return rapid.StringMatching(`^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$`)
}

// TraceIDGen generates valid trace IDs.
func TraceIDGen() *rapid.Generator[string] {
	return rapid.StringMatching(`^[a-f0-9]{32}$`)
}

// SpanIDGen generates valid span IDs.
func SpanIDGen() *rapid.Generator[string] {
	return rapid.StringMatching(`^[a-f0-9]{16}$`)
}
