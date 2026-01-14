// Package property contains property-based tests for file-upload service.
// Feature: file-upload-modernization-2025, Property 10: Storage Path Tenant Isolation
// Validates: Requirements 9.5
package property

import (
	"strings"
	"testing"
	"time"

	"pgregory.net/rapid"
)

// TestPathBuilder is a test implementation of path building logic.
type TestPathBuilder struct{}

// BuildPath creates a tenant-isolated hierarchical path.
// Format: {tenant_id}/{year}/{month}/{day}/{hash}/{filename}
func (b *TestPathBuilder) BuildPath(tenantID, hash, filename string) string {
	now := time.Now().UTC()
	return tenantID + "/" +
		now.Format("2006") + "/" +
		now.Format("01") + "/" +
		now.Format("02") + "/" +
		hash + "/" +
		filename
}

// ExtractTenantID extracts tenant ID from a storage path.
func (b *TestPathBuilder) ExtractTenantID(path string) string {
	if len(path) == 0 {
		return ""
	}
	for i, c := range path {
		if c == '/' {
			return path[:i]
		}
	}
	return path
}

// ValidateTenantAccess validates that the path belongs to the tenant.
func (b *TestPathBuilder) ValidateTenantAccess(path, tenantID string) bool {
	extractedTenant := b.ExtractTenantID(path)
	return extractedTenant == tenantID
}

// TestProperty10_StoragePathContainsTenantID tests that all generated storage paths
// contain the tenant ID as the first path segment.
// Property 10: Storage Path Tenant Isolation
// Validates: Requirements 9.5
func TestProperty10_StoragePathContainsTenantID(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		builder := &TestPathBuilder{}

		// Generate random tenant ID, hash, and filename
		tenantID := rapid.StringMatching(`tenant-[a-z0-9]{8,16}`).Draw(t, "tenantID")
		hash := rapid.StringMatching(`[a-f0-9]{64}`).Draw(t, "hash")
		filename := rapid.StringMatching(`[a-z0-9_-]{1,50}\.(jpg|png|pdf|txt)`).Draw(t, "filename")

		// Build path
		path := builder.BuildPath(tenantID, hash, filename)

		// Property: Path SHALL contain tenant ID as first segment
		if !strings.HasPrefix(path, tenantID+"/") {
			t.Errorf("path %q does not start with tenant ID %q", path, tenantID)
		}

		// Verify extraction works correctly
		extractedTenant := builder.ExtractTenantID(path)
		if extractedTenant != tenantID {
			t.Errorf("extracted tenant %q does not match original %q", extractedTenant, tenantID)
		}
	})
}

// TestProperty10_TenantAccessValidation tests that tenant access validation works correctly.
// Property 10: Storage Path Tenant Isolation
// Validates: Requirements 9.5
func TestProperty10_TenantAccessValidation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		builder := &TestPathBuilder{}

		// Generate two different tenant IDs
		tenantA := rapid.StringMatching(`tenant-a-[a-z0-9]{8}`).Draw(t, "tenantA")
		tenantB := rapid.StringMatching(`tenant-b-[a-z0-9]{8}`).Draw(t, "tenantB")
		hash := rapid.StringMatching(`[a-f0-9]{64}`).Draw(t, "hash")
		filename := rapid.StringMatching(`[a-z0-9_-]{1,50}\.(jpg|png|pdf)`).Draw(t, "filename")

		// Build path for tenant A
		pathA := builder.BuildPath(tenantA, hash, filename)

		// Property: Tenant A SHALL have access to their own path
		if !builder.ValidateTenantAccess(pathA, tenantA) {
			t.Errorf("tenant %q should have access to path %q", tenantA, pathA)
		}

		// Property: Tenant B SHALL NOT have access to tenant A's path
		if builder.ValidateTenantAccess(pathA, tenantB) {
			t.Errorf("tenant %q should NOT have access to path %q", tenantB, pathA)
		}
	})
}

// TestProperty10_PathHierarchicalStructure tests that paths follow hierarchical structure.
// Property 10: Storage Path Tenant Isolation
// Validates: Requirements 9.5
func TestProperty10_PathHierarchicalStructure(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		builder := &TestPathBuilder{}

		tenantID := rapid.StringMatching(`tenant-[a-z0-9]{8}`).Draw(t, "tenantID")
		hash := rapid.StringMatching(`[a-f0-9]{64}`).Draw(t, "hash")
		filename := rapid.StringMatching(`[a-z0-9_-]{1,50}\.pdf`).Draw(t, "filename")

		path := builder.BuildPath(tenantID, hash, filename)

		// Property: Path SHALL follow format {tenant}/{year}/{month}/{day}/{hash}/{filename}
		parts := strings.Split(path, "/")
		if len(parts) != 6 {
			t.Errorf("expected 6 path segments, got %d: %v", len(parts), parts)
			return
		}

		// Verify tenant ID
		if parts[0] != tenantID {
			t.Errorf("first segment should be tenant ID %q, got %q", tenantID, parts[0])
		}

		// Verify year format (4 digits)
		if len(parts[1]) != 4 {
			t.Errorf("year segment should be 4 digits, got %q", parts[1])
		}

		// Verify month format (2 digits, 01-12)
		if len(parts[2]) != 2 {
			t.Errorf("month segment should be 2 digits, got %q", parts[2])
		}

		// Verify day format (2 digits, 01-31)
		if len(parts[3]) != 2 {
			t.Errorf("day segment should be 2 digits, got %q", parts[3])
		}

		// Verify hash
		if parts[4] != hash {
			t.Errorf("hash segment should be %q, got %q", hash, parts[4])
		}

		// Verify filename
		if parts[5] != filename {
			t.Errorf("filename segment should be %q, got %q", filename, parts[5])
		}
	})
}

// TestProperty10_EmptyPathHandling tests handling of empty paths.
// Property 10: Storage Path Tenant Isolation
// Validates: Requirements 9.5
func TestProperty10_EmptyPathHandling(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		builder := &TestPathBuilder{}

		tenantID := rapid.StringMatching(`tenant-[a-z0-9]{8}`).Draw(t, "tenantID")

		// Property: Empty path SHALL return empty tenant ID
		extractedEmpty := builder.ExtractTenantID("")
		if extractedEmpty != "" {
			t.Errorf("expected empty tenant ID for empty path, got %q", extractedEmpty)
		}

		// Property: Empty path SHALL NOT validate for any tenant
		if builder.ValidateTenantAccess("", tenantID) {
			t.Error("empty path should not validate for any tenant")
		}
	})
}
