// Package property contains property-based tests for IAM Policy Service.
package property

import (
	"testing"

	"github.com/auth-platform/iam-policy-service/internal/rbac"
	"github.com/auth-platform/iam-policy-service/tests/testutil"
	"pgregory.net/rapid"
)

// TestPermissionInheritanceCompleteness validates Property 6.
// Child roles must inherit all permissions from parent roles.
func TestPermissionInheritanceCompleteness(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		hierarchy := rbac.NewRoleHierarchy()

		// Create parent role with permissions
		parentPerms := rapid.SliceOfN(
			rapid.SampledFrom([]string{"read", "write", "delete", "create", "update"}),
			1, 3,
		).Draw(t, "parentPerms")

		parent := &rbac.Role{
			ID:          "parent",
			Name:        "Parent Role",
			Permissions: parentPerms,
		}
		_ = hierarchy.AddRole(parent)

		// Create child role with additional permissions
		childPerms := rapid.SliceOfN(
			rapid.SampledFrom([]string{"export", "import", "share"}),
			0, 2,
		).Draw(t, "childPerms")

		child := &rbac.Role{
			ID:          "child",
			Name:        "Child Role",
			ParentID:    "parent",
			Permissions: childPerms,
		}
		_ = hierarchy.AddRole(child)

		// Get effective permissions for child
		effectivePerms := hierarchy.GetEffectivePermissions("child")
		effectiveMap := make(map[string]bool)
		for _, p := range effectivePerms {
			effectiveMap[p] = true
		}

		// Property: all parent permissions must be in child's effective permissions
		for _, p := range parentPerms {
			if !effectiveMap[p] {
				t.Errorf("parent permission %s not inherited by child", p)
			}
		}

		// Property: all child's own permissions must be in effective permissions
		for _, p := range childPerms {
			if !effectiveMap[p] {
				t.Errorf("child's own permission %s not in effective permissions", p)
			}
		}
	})
}

// TestCircularDependencyDetection validates Property 7.
// Circular dependencies must be detected and prevented.
func TestCircularDependencyDetection(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		hierarchy := rbac.NewRoleHierarchy()

		// Create a chain of roles
		chainLength := rapid.IntRange(2, 5).Draw(t, "chainLength")

		for i := 0; i < chainLength; i++ {
			role := &rbac.Role{
				ID:   testutil.NonEmptyStringGen().Draw(t, "roleID"),
				Name: "Role " + string(rune('A'+i)),
			}
			if i > 0 {
				// Get previous role ID
				prevRole := hierarchy.GetRole(hierarchy.GetAncestors(role.ID)[0])
				if prevRole != nil {
					role.ParentID = prevRole.ID
				}
			}
			_ = hierarchy.AddRole(role)
		}

		// Property: no circular dependency should exist in valid hierarchy
		if hierarchy.HasCircularDependency() {
			t.Error("unexpected circular dependency in valid hierarchy")
		}
	})
}

// TestCircularDependencyPrevention validates that cycles are prevented.
func TestCircularDependencyPrevention(t *testing.T) {
	hierarchy := rbac.NewRoleHierarchy()

	// Create A -> B -> C chain
	_ = hierarchy.AddRole(&rbac.Role{ID: "A", Name: "Role A"})
	_ = hierarchy.AddRole(&rbac.Role{ID: "B", Name: "Role B", ParentID: "A"})
	_ = hierarchy.AddRole(&rbac.Role{ID: "C", Name: "Role C", ParentID: "B"})

	// Try to create cycle: A -> C (would create C -> B -> A -> C)
	err := hierarchy.AddRole(&rbac.Role{ID: "A", Name: "Role A", ParentID: "C"})

	// Property: cycle creation must be prevented
	if err == nil {
		t.Error("circular dependency should have been prevented")
	}
}

// TestSelfReferenceDetection validates that self-references are detected.
func TestSelfReferenceDetection(t *testing.T) {
	hierarchy := rbac.NewRoleHierarchy()

	// Try to create self-referencing role
	err := hierarchy.AddRole(&rbac.Role{
		ID:       "self",
		Name:     "Self Role",
		ParentID: "self",
	})

	// Property: self-reference must be prevented
	if err == nil {
		t.Error("self-reference should have been prevented")
	}
}

// TestPermissionCaching validates that permission caching works correctly.
func TestPermissionCaching(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		hierarchy := rbac.NewRoleHierarchy()

		perms := rapid.SliceOfN(
			rapid.SampledFrom([]string{"read", "write", "delete"}),
			1, 3,
		).Draw(t, "perms")

		_ = hierarchy.AddRole(&rbac.Role{
			ID:          "role1",
			Name:        "Role 1",
			Permissions: perms,
		})

		// Get permissions multiple times
		perms1 := hierarchy.GetEffectivePermissions("role1")
		perms2 := hierarchy.GetEffectivePermissions("role1")

		// Property: cached results must match
		if len(perms1) != len(perms2) {
			t.Errorf("permission count mismatch: %d vs %d", len(perms1), len(perms2))
		}

		permsMap := make(map[string]bool)
		for _, p := range perms1 {
			permsMap[p] = true
		}

		for _, p := range perms2 {
			if !permsMap[p] {
				t.Errorf("permission %s in second call but not first", p)
			}
		}
	})
}

// TestAncestorChain validates ancestor chain retrieval.
func TestAncestorChain(t *testing.T) {
	hierarchy := rbac.NewRoleHierarchy()

	// Create chain: grandparent -> parent -> child
	_ = hierarchy.AddRole(&rbac.Role{ID: "grandparent", Name: "Grandparent"})
	_ = hierarchy.AddRole(&rbac.Role{ID: "parent", Name: "Parent", ParentID: "grandparent"})
	_ = hierarchy.AddRole(&rbac.Role{ID: "child", Name: "Child", ParentID: "parent"})

	ancestors := hierarchy.GetAncestors("child")

	// Property: ancestors must be in correct order
	if len(ancestors) != 2 {
		t.Fatalf("expected 2 ancestors, got %d", len(ancestors))
	}
	if ancestors[0] != "parent" {
		t.Errorf("first ancestor should be parent, got %s", ancestors[0])
	}
	if ancestors[1] != "grandparent" {
		t.Errorf("second ancestor should be grandparent, got %s", ancestors[1])
	}
}

// TestHasPermission validates permission checking.
func TestHasPermission(t *testing.T) {
	hierarchy := rbac.NewRoleHierarchy()

	_ = hierarchy.AddRole(&rbac.Role{
		ID:          "admin",
		Name:        "Admin",
		Permissions: []string{"read", "write", "delete"},
	})

	// Property: role should have its permissions
	if !hierarchy.HasPermission("admin", "read") {
		t.Error("admin should have read permission")
	}
	if !hierarchy.HasPermission("admin", "write") {
		t.Error("admin should have write permission")
	}

	// Property: role should not have permissions it wasn't given
	if hierarchy.HasPermission("admin", "superadmin") {
		t.Error("admin should not have superadmin permission")
	}
}
