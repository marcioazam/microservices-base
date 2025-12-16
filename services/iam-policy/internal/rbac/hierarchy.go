package rbac

import (
	"sync"
)

type Role struct {
	ID          string
	Name        string
	Description string
	ParentID    string
	Permissions []string
}

type RoleHierarchy struct {
	mu    sync.RWMutex
	roles map[string]*Role
}

func NewRoleHierarchy() *RoleHierarchy {
	return &RoleHierarchy{
		roles: make(map[string]*Role),
	}
}

func (h *RoleHierarchy) AddRole(role *Role) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.roles[role.ID] = role
}

func (h *RoleHierarchy) GetRole(id string) *Role {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.roles[id]
}

func (h *RoleHierarchy) GetEffectivePermissions(roleID string) []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	permissions := make(map[string]bool)
	h.collectPermissions(roleID, permissions)

	result := make([]string, 0, len(permissions))
	for p := range permissions {
		result = append(result, p)
	}
	return result
}

func (h *RoleHierarchy) collectPermissions(roleID string, permissions map[string]bool) {
	role, exists := h.roles[roleID]
	if !exists {
		return
	}

	for _, p := range role.Permissions {
		permissions[p] = true
	}

	if role.ParentID != "" {
		h.collectPermissions(role.ParentID, permissions)
	}
}

func (h *RoleHierarchy) GetAncestors(roleID string) []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var ancestors []string
	currentID := roleID

	for {
		role, exists := h.roles[currentID]
		if !exists || role.ParentID == "" {
			break
		}
		ancestors = append(ancestors, role.ParentID)
		currentID = role.ParentID
	}

	return ancestors
}
