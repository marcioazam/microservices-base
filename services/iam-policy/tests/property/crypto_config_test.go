package property

import (
	"testing"
	"time"

	"github.com/auth-platform/iam-policy-service/internal/config"
	"pgregory.net/rapid"
)

// TestCryptoConfigValidation validates Property 8: Configuration Validation
// For any invalid configuration (missing required fields, invalid key ID format),
// service initialization SHALL fail with descriptive error.
// **Validates: Requirements 5.7**
func TestCryptoConfigValidation(t *testing.T) {
	t.Run("valid_key_id_format_accepted", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			namespace := rapid.StringMatching(`[a-z][a-z0-9-]{0,20}`).Draw(t, "namespace")
			id := rapid.StringMatching(`[a-z][a-z0-9-]{0,20}`).Draw(t, "id")
			version := rapid.IntRange(1, 100).Draw(t, "version")

			keyID := namespace + "/" + id + "/" + cryptoIntToString(version)
			err := config.ValidateKeyID(keyID)

			if err != nil {
				t.Errorf("valid key ID %q should be accepted, got error: %v", keyID, err)
			}
		})
	})

	t.Run("invalid_key_id_format_rejected", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			// Generate invalid key IDs with wrong number of parts
			numParts := rapid.OneOf(
				rapid.Just(1),
				rapid.Just(2),
				rapid.Just(4),
				rapid.Just(5),
			).Draw(t, "numParts")

			var keyID string
			for i := 0; i < numParts; i++ {
				if i > 0 {
					keyID += "/"
				}
				keyID += rapid.StringMatching(`[a-z]{1,10}`).Draw(t, "part")
			}

			err := config.ValidateKeyID(keyID)
			if err == nil {
				t.Errorf("invalid key ID %q with %d parts should be rejected", keyID, numParts)
			}
		})
	})

	t.Run("empty_parts_rejected", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			emptyPart := rapid.IntRange(0, 2).Draw(t, "emptyPart")
			parts := []string{"namespace", "id", "1"}
			parts[emptyPart] = ""

			keyID := parts[0] + "/" + parts[1] + "/" + parts[2]
			err := config.ValidateKeyID(keyID)

			if err == nil {
				t.Errorf("key ID with empty part %d should be rejected: %q", emptyPart, keyID)
			}
		})
	})

	t.Run("crypto_config_requires_address_when_enabled", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			cfg := &config.CryptoConfig{
				Enabled: true,
				Address: "", // Empty address
				Timeout: time.Duration(rapid.Int64Range(1, 10000).Draw(t, "timeout")) * time.Millisecond,
			}

			err := cfg.Validate()
			if err == nil {
				t.Error("crypto config with empty address should fail validation when enabled")
			}
		})
	})

	t.Run("crypto_config_requires_encryption_key_when_cache_encryption_enabled", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			cfg := &config.CryptoConfig{
				Enabled:           true,
				Address:           "localhost:50051",
				Timeout:           5000000000, // 5s in nanoseconds
				CacheEncryption:   true,
				EncryptionKeyID:   "", // Empty key ID
			}

			err := cfg.Validate()
			if err == nil {
				t.Error("crypto config should require encryption key ID when cache encryption is enabled")
			}
		})
	})

	t.Run("crypto_config_requires_signing_key_when_signing_enabled", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			cfg := &config.CryptoConfig{
				Enabled:         true,
				Address:         "localhost:50051",
				Timeout:         5000000000, // 5s in nanoseconds
				DecisionSigning: true,
				SigningKeyID:    "", // Empty key ID
			}

			err := cfg.Validate()
			if err == nil {
				t.Error("crypto config should require signing key ID when decision signing is enabled")
			}
		})
	})

	t.Run("disabled_crypto_config_always_valid", func(t *testing.T) {
		rapid.Check(t, func(t *rapid.T) {
			cfg := &config.CryptoConfig{
				Enabled: false,
				// All other fields can be anything when disabled
				Address:         rapid.String().Draw(t, "address"),
				CacheEncryption: rapid.Bool().Draw(t, "cacheEncryption"),
				DecisionSigning: rapid.Bool().Draw(t, "decisionSigning"),
			}

			err := cfg.Validate()
			if err != nil {
				t.Errorf("disabled crypto config should always be valid, got error: %v", err)
			}
		})
	})
}

func cryptoIntToString(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
