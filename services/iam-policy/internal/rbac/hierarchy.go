// Package rbac provides role-based access control for IAM Policy Service.
package rbac

import (
	"fmt"
	"sync"
)

// Role represents a role in the hierarchy.
type Role struct {
	ID          string
	Name        string
	Description string
	ParentID    string
	Permissions []string
}

// RoleHierarchy manages roles and their inheritance.
type RoleHierarchy struct {
	mu              sync.RWMutex
	roles           map[string]*Role
	permissionCache map[string][]string
}

// NewRoleHierarchy creates a new role hierarchy.
func NewRoleHierarchy() *RoleHierarchy {
	return &RoleHierarchy{
		roles:           make(map[string]*Role),
		permissionCache: make(map[string][]string),
	}
}

// AddRole adds a role to the hierarchy.
func (h *RoleHierarchy) AddRole(role *Role) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Check for circular dependency before adding
	if role.ParentID != "" {
		if h.wouldCreateCycle(role.ID, role.ParentID) {
			return fmt.Errorf("adding role %s with parent %s would create circular dependency", role.ID, role.ParentID)
		}
	}

	h.roles[role.ID] = role
	h.invalidateCache()
	return nil
}

// GetRole retrieves a role by ID.
func (h *RoleHierarchy) GetRole(id string) *Role {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.roles[id]
}

// GetEffectivePermissions returns all permissions for a role including inherited.
func (h *RoleHierarchy) GetEffectivePermissions(roleID string) []string {
	h.mu.RLock()

	// Check cache first
	if cached, ok := h.permissionCache[roleID]; ok {
		h.mu.RUnlock()
		return cached
	}
	h.mu.RUnlock()

	// Compute permissions
	h.mu.Lock()
	defer h.mu.Unlock()

	permissions := make(map[string]bool)
	visited := make(map[string]bool)
	h.collectPermissions(roleID, permissions, visited)

	result := make([]string, 0, len(permissions))
	for p := range permissions {
		result = append(result, p)
	}

	h.permissionCache[roleID] = result
	return result
}

func (h *RoleHierarchy) collectPermissions(roleID string, permissions map[string]bool, visited map[string]bool) {
	if visited[roleID] {
		return // Prevent infinite loop
	}
	visited[roleID] = true

	role, exists := h.roles[roleID]
	if !exists {
		return
	}

	for _, p := range role.Permissions {
		permissions[p] = true
	}

	if role.ParentID != "" {
		h.collectPermissions(role.ParentID, permissions, visited)
	}
}

// HasCircularDependency checks if the hierarchy has any circular dependencies.
func (h *RoleHierarchy) HasCircularDependency() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for roleID := range h.roles {
		if h.detectCycle(roleID) {
			return true
		}
	}
	return false
}

// detectCycle checks if a role has a circular dependency.
func (h *RoleHierarchy) detectCycle(roleID string) bool {
	visited := make(map[string]bool)
	current := roleID

	for {
		if visited[current] {
			return true
		}
		visited[current] = true

		role, exists := h.roles[current]
		if !exists || role.ParentID == "" {
			return false
		}
		current = role.ParentID
	}
}

// wouldCreateCycle checks if adding a parent would create a cycle.
func (h *RoleHierarchy) wouldCreateCycle(roleID, parentID string) bool {
	if roleID == parentID {
		return true
	}

	visited := make(map[string]bool)
	current := parentID

	for {
		if current == roleID {
			return true
		}
		if visited[current] {
			return false
		}
		visited[current] = true

		role, exists := h.roles[current]
		if !exists || role.ParentID == "" {
			return false
		}
		current = role.ParentID
	}
}

// GetAncestors returns all ancestor roles.
func (h *RoleHierarchy) GetAncestors(roleID string) []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var ancestors []string
	visited := make(map[string]bool)
	currentID := roleID

	for {
		if visited[currentID] {
			break // Prevent infinite loop
		}
		visited[currentID] = true

		role, exists := h.roles[currentID]
		if !exists || role.ParentID == "" {
			break
		}
		ancestors = append(ancestors, role.ParentID)
		currentID = role.ParentID
	}

	return ancestors
}

// GetDescendants returns all descendant roles.
func (h *RoleHierarchy) GetDescendants(roleID string) []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var descendants []string
	for id, role := range h.roles {
		if role.ParentID == roleID {
			descendants = append(descendants, id)
			descendants = append(descendants, h.getDescendantsRecursive(id)...)
		}
	}
	return descendants
}

func (h *RoleHierarchy) getDescendantsRecursive(roleID string) []string {
	var descendants []string
	for id, role := range h.roles {
		if role.ParentID == roleID {
			descendants = append(descendants, id)
			descendants = append(descendants, h.getDescendantsRecursive(id)...)
		}
	}
	return descendants
}

// HasPermission checks if a role has a specific permission.
func (h *RoleHierarchy) HasPermission(roleID, permission string) bool {
	permissions := h.GetEffectivePermissions(roleID)
	for _, p := range permissions {
		if p == permission {
			return true
		}
	}
	return false
}

// invalidateCache clears the permission cache.
func (h *RoleHierarchy) invalidateCache() {
	h.permissionCache = make(map[string][]string)
}

// RoleCount returns the number of roles.
func (h *RoleHierarchy) RoleCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.roles)
}
