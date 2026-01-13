package cache

import (
	"encoding/json"

	"github.com/auth-platform/iam-policy-service/internal/crypto"
)

// EncryptedCacheEntry represents an encrypted decision stored in cache.
type EncryptedCacheEntry struct {
	Ciphertext []byte       `json:"ciphertext"`
	IV         []byte       `json:"iv"`
	Tag        []byte       `json:"tag"`
	KeyID      crypto.KeyID `json:"key_id"`
	Algorithm  string       `json:"algorithm"`
	CachedAt   int64        `json:"cached_at"`
	ExpiresAt  int64        `json:"expires_at"`
}

// MarshalJSON implements json.Marshaler for EncryptedCacheEntry.
func (e *EncryptedCacheEntry) MarshalJSON() ([]byte, error) {
	type Alias EncryptedCacheEntry
	return json.Marshal(&struct {
		*Alias
		KeyID string `json:"key_id"`
	}{
		Alias: (*Alias)(e),
		KeyID: e.KeyID.String(),
	})
}

// UnmarshalJSON implements json.Unmarshaler for EncryptedCacheEntry.
func (e *EncryptedCacheEntry) UnmarshalJSON(data []byte) error {
	type Alias EncryptedCacheEntry
	aux := &struct {
		*Alias
		KeyID string `json:"key_id"`
	}{
		Alias: (*Alias)(e),
	}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	if aux.KeyID != "" {
		keyID, err := crypto.ParseKeyID(aux.KeyID)
		if err != nil {
			return err
		}
		e.KeyID = keyID
	}

	return nil
}

// IsExpired returns true if the entry has expired.
func (e *EncryptedCacheEntry) IsExpired(nowUnix int64) bool {
	return nowUnix >= e.ExpiresAt
}
