package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/auth-platform/iam-policy-service/internal/crypto"
	"github.com/auth-platform/iam-policy-service/internal/logging"
)

// EncryptedDecisionCache wraps DecisionCache with encryption support.
type EncryptedDecisionCache struct {
	inner        *DecisionCache
	cryptoClient *crypto.Client
	enabled      bool
	logger       *logging.Logger
}

// NewEncryptedDecisionCache creates an encrypted cache wrapper.
func NewEncryptedDecisionCache(inner *DecisionCache, client *crypto.Client, logger *logging.Logger) *EncryptedDecisionCache {
	enabled := client != nil && client.IsCacheEncryptionEnabled()

	return &EncryptedDecisionCache{
		inner:        inner,
		cryptoClient: client,
		enabled:      enabled,
		logger:       logger,
	}
}

// Get retrieves and decrypts a cached decision.
func (ec *EncryptedDecisionCache) Get(ctx context.Context, input map[string]interface{}) (*Decision, bool) {
	if !ec.enabled {
		return ec.inner.Get(ctx, input)
	}

	// Get encrypted entry from inner cache
	decision, found := ec.inner.Get(ctx, input)
	if !found {
		return nil, false
	}

	// If we got a decision directly (fallback mode), return it
	if decision != nil && decision.Reason != "" {
		return decision, true
	}

	// Try to get raw encrypted data
	key := ec.inner.generateKey(input)
	encryptedData, found := ec.getRawEntry(ctx, key)
	if !found {
		return nil, false
	}

	// Parse encrypted entry
	var entry EncryptedCacheEntry
	if err := json.Unmarshal(encryptedData, &entry); err != nil {
		ec.logError(ctx, "failed to unmarshal encrypted entry", err)
		ec.inner.Delete(ctx, input)
		return nil, false
	}

	// Check expiration
	if entry.IsExpired(time.Now().Unix()) {
		ec.inner.Delete(ctx, input)
		return nil, false
	}

	// Generate AAD from input
	aad := ec.generateAAD(input)

	// Decrypt
	plaintext, err := ec.cryptoClient.Decrypt(ctx, entry.Ciphertext, entry.IV, entry.Tag, aad)
	if err != nil {
		if crypto.IsAADMismatch(err) {
			ec.logWarn(ctx, "AAD mismatch, invalidating cache entry")
		} else {
			ec.logError(ctx, "decryption failed", err)
		}
		ec.inner.Delete(ctx, input)
		return nil, false
	}

	// Unmarshal decision
	var decryptedDecision Decision
	if err := json.Unmarshal(plaintext, &decryptedDecision); err != nil {
		ec.logError(ctx, "failed to unmarshal decrypted decision", err)
		ec.inner.Delete(ctx, input)
		return nil, false
	}

	return &decryptedDecision, true
}

// Set encrypts and stores a decision.
func (ec *EncryptedDecisionCache) Set(ctx context.Context, input map[string]interface{}, decision *Decision) error {
	if !ec.enabled {
		return ec.inner.Set(ctx, input, decision)
	}

	// Marshal decision to JSON
	plaintext, err := json.Marshal(decision)
	if err != nil {
		return fmt.Errorf("failed to marshal decision: %w", err)
	}

	// Generate AAD from input
	aad := ec.generateAAD(input)

	// Encrypt
	result, err := ec.cryptoClient.Encrypt(ctx, plaintext, aad)
	if err != nil {
		if crypto.IsServiceUnavailable(err) {
			ec.logWarn(ctx, "crypto service unavailable, storing unencrypted")
			return ec.inner.Set(ctx, input, decision)
		}
		return fmt.Errorf("encryption failed: %w", err)
	}

	// Create encrypted entry
	now := time.Now()
	entry := &EncryptedCacheEntry{
		Ciphertext: result.Ciphertext,
		IV:         result.IV,
		Tag:        result.Tag,
		KeyID:      result.KeyID,
		Algorithm:  result.Algorithm,
		CachedAt:   now.Unix(),
		ExpiresAt:  now.Add(ec.inner.ttl).Unix(),
	}

	// Store encrypted entry
	return ec.setRawEntry(ctx, input, entry)
}

// Delete removes a decision from cache.
func (ec *EncryptedDecisionCache) Delete(ctx context.Context, input map[string]interface{}) error {
	return ec.inner.Delete(ctx, input)
}

// Invalidate clears all cached decisions.
func (ec *EncryptedDecisionCache) Invalidate(ctx context.Context) error {
	return ec.inner.Invalidate(ctx)
}

// Close closes the cache client.
func (ec *EncryptedDecisionCache) Close() error {
	return ec.inner.Close()
}

// IsEncryptionEnabled returns true if encryption is enabled.
func (ec *EncryptedDecisionCache) IsEncryptionEnabled() bool {
	return ec.enabled
}

// generateAAD creates Additional Authenticated Data from authorization input.
// AAD binds the encrypted data to the specific subject and resource.
func (ec *EncryptedDecisionCache) generateAAD(input map[string]interface{}) []byte {
	var subjectID, resourceID string

	if subject, ok := input["subject"].(map[string]interface{}); ok {
		if id, ok := subject["id"].(string); ok {
			subjectID = id
		}
	}

	if resource, ok := input["resource"].(map[string]interface{}); ok {
		if id, ok := resource["id"].(string); ok {
			resourceID = id
		}
	}

	// AAD format: "subject_id:resource_id"
	aad := fmt.Sprintf("%s:%s", subjectID, resourceID)
	return []byte(aad)
}

// getRawEntry retrieves raw bytes from cache (for encrypted entries).
func (ec *EncryptedDecisionCache) getRawEntry(ctx context.Context, key string) ([]byte, bool) {
	if ec.inner.client != nil && !ec.inner.useLocal {
		result := ec.inner.client.Get(ctx, key)
		if result.IsOk() {
			entry := result.Unwrap()
			return entry.Value, true
		}
	}
	return nil, false
}

// setRawEntry stores raw bytes in cache (for encrypted entries).
func (ec *EncryptedDecisionCache) setRawEntry(ctx context.Context, input map[string]interface{}, entry *EncryptedCacheEntry) error {
	key := ec.inner.generateKey(input)

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal encrypted entry: %w", err)
	}

	if ec.inner.client != nil && !ec.inner.useLocal {
		if err := ec.inner.client.Set(ctx, key, data, ec.inner.ttl); err != nil {
			ec.logError(ctx, "failed to store encrypted entry in remote cache", err)
		}
	}

	return nil
}

func (ec *EncryptedDecisionCache) logWarn(ctx context.Context, msg string) {
	if ec.logger != nil {
		ec.logger.Warn(ctx, msg)
	}
}

func (ec *EncryptedDecisionCache) logError(ctx context.Context, msg string, err error) {
	if ec.logger != nil {
		ec.logger.Error(ctx, msg, logging.Error(err))
	}
}
