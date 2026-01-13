// Package property contains property-based tests for the cache service.
// Feature: cache-microservice
package property

import (
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"

	"github.com/auth-platform/cache-service/internal/crypto"
)

// Property 11: Encryption Round-Trip
// For any value stored with encryption enabled, decrypting the stored value
// SHALL return the original plaintext.
// Validates: Requirements 5.3
func TestProperty11_EncryptionRoundTrip(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = PropertyTestIterations
	parameters.Rng.Seed(PropertyTestSeed)

	properties := gopter.NewProperties(parameters)

	properties.Property("Encrypt then decrypt returns original", prop.ForAll(
		func(plaintext []byte) bool {
			if len(plaintext) == 0 {
				return true
			}

			// Generate a random key
			key, err := crypto.GenerateKey(32)
			if err != nil {
				return false
			}

			encryptor, err := crypto.NewAESEncryptor(key)
			if err != nil {
				return false
			}

			// Encrypt
			ciphertext, err := encryptor.Encrypt(plaintext)
			if err != nil {
				return false
			}

			// Ciphertext should be different from plaintext
			if len(ciphertext) == len(plaintext) {
				same := true
				for i := range plaintext {
					if ciphertext[i] != plaintext[i] {
						same = false
						break
					}
				}
				if same {
					return false
				}
			}

			// Decrypt
			decrypted, err := encryptor.Decrypt(ciphertext)
			if err != nil {
				return false
			}

			// Verify round-trip
			if len(decrypted) != len(plaintext) {
				return false
			}
			for i := range plaintext {
				if decrypted[i] != plaintext[i] {
					return false
				}
			}

			return true
		},
		gen.SliceOf(gen.UInt8()).SuchThat(func(b []byte) bool { return len(b) > 0 && len(b) < 10000 }),
	))

	properties.Property("Different plaintexts produce different ciphertexts", prop.ForAll(
		func(plaintext1, plaintext2 []byte) bool {
			if len(plaintext1) == 0 || len(plaintext2) == 0 {
				return true
			}

			// Skip if plaintexts are the same
			if len(plaintext1) == len(plaintext2) {
				same := true
				for i := range plaintext1 {
					if plaintext1[i] != plaintext2[i] {
						same = false
						break
					}
				}
				if same {
					return true
				}
			}

			key, err := crypto.GenerateKey(32)
			if err != nil {
				return false
			}

			encryptor, err := crypto.NewAESEncryptor(key)
			if err != nil {
				return false
			}

			ciphertext1, err := encryptor.Encrypt(plaintext1)
			if err != nil {
				return false
			}

			ciphertext2, err := encryptor.Encrypt(plaintext2)
			if err != nil {
				return false
			}

			// Ciphertexts should be different
			if len(ciphertext1) == len(ciphertext2) {
				same := true
				for i := range ciphertext1 {
					if ciphertext1[i] != ciphertext2[i] {
						same = false
						break
					}
				}
				return !same
			}

			return true
		},
		gen.SliceOf(gen.UInt8()).SuchThat(func(b []byte) bool { return len(b) > 0 && len(b) < 1000 }),
		gen.SliceOf(gen.UInt8()).SuchThat(func(b []byte) bool { return len(b) > 0 && len(b) < 1000 }),
	))

	properties.Property("Same plaintext encrypted twice produces different ciphertexts (due to random nonce)", prop.ForAll(
		func(plaintext []byte) bool {
			if len(plaintext) == 0 {
				return true
			}

			key, err := crypto.GenerateKey(32)
			if err != nil {
				return false
			}

			encryptor, err := crypto.NewAESEncryptor(key)
			if err != nil {
				return false
			}

			ciphertext1, err := encryptor.Encrypt(plaintext)
			if err != nil {
				return false
			}

			ciphertext2, err := encryptor.Encrypt(plaintext)
			if err != nil {
				return false
			}

			// Ciphertexts should be different due to random nonce
			if len(ciphertext1) != len(ciphertext2) {
				return true
			}

			different := false
			for i := range ciphertext1 {
				if ciphertext1[i] != ciphertext2[i] {
					different = true
					break
				}
			}

			return different
		},
		gen.SliceOf(gen.UInt8()).SuchThat(func(b []byte) bool { return len(b) > 0 && len(b) < 1000 }),
	))

	properties.TestingRun(t)
}
