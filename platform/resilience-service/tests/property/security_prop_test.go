package property

import (
	"testing"

	"github.com/auth-platform/platform/resilience-service/internal/infrastructure/security"
	"pgregory.net/rapid"
)

// TestProperty_PathValidation validates path security validation.
func TestProperty_PathValidation(t *testing.T) {
	t.Run("valid_paths_accepted", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			segment := rapid.StringMatching(`[a-zA-Z][a-zA-Z0-9_-]{0,20}`).Draw(t, "segment")
			path := segment + ".json"

			err := security.ValidatePolicyPath(path, ".")
			if err != nil {
				t.Fatalf("valid path rejected: %s, error: %v", path, err)
			}
		})
	})

	t.Run("path_traversal_rejected", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			prefix := rapid.StringMatching(`[a-zA-Z]{1,5}`).Draw(t, "prefix")
			path := prefix + "/../etc/passwd"

			err := security.ValidatePolicyPath(path, ".")
			if err == nil {
				t.Fatalf("path traversal should be rejected: %s", path)
			}
		})
	})

	t.Run("null_bytes_rejected", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			prefix := rapid.StringMatching(`[a-zA-Z]{1,5}`).Draw(t, "prefix")
			path := prefix + "\x00/test"

			err := security.ValidatePolicyPath(path, ".")
			if err == nil {
				t.Fatalf("null bytes should be rejected: %s", path)
			}
		})
	})
}
