// Package crypto provides encryption and decryption functionality.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
)

var (
	// ErrInvalidKey indicates the encryption key is invalid.
	ErrInvalidKey = errors.New("invalid encryption key: must be 16, 24, or 32 bytes")
	// ErrInvalidCiphertext indicates the ciphertext is invalid.
	ErrInvalidCiphertext = errors.New("invalid ciphertext")
	// ErrDecryptionFailed indicates decryption failed.
	ErrDecryptionFailed = errors.New("decryption failed")
)

// AESEncryptor implements AES-GCM encryption.
type AESEncryptor struct {
	gcm cipher.AEAD
}

// NewAESEncryptor creates a new AES encryptor with the given key.
// Key must be 16, 24, or 32 bytes for AES-128, AES-192, or AES-256.
func NewAESEncryptor(key []byte) (*AESEncryptor, error) {
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		return nil, ErrInvalidKey
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return &AESEncryptor{gcm: gcm}, nil
}

// NewAESEncryptorFromString creates a new AES encryptor from a base64-encoded key.
func NewAESEncryptorFromString(keyStr string) (*AESEncryptor, error) {
	if keyStr == "" {
		return nil, ErrInvalidKey
	}

	key, err := base64.StdEncoding.DecodeString(keyStr)
	if err != nil {
		// Try using the string directly as key
		key = []byte(keyStr)
	}

	return NewAESEncryptor(key)
}

// Encrypt encrypts plaintext using AES-GCM.
func (e *AESEncryptor) Encrypt(plaintext []byte) ([]byte, error) {
	if len(plaintext) == 0 {
		return nil, errors.New("plaintext cannot be empty")
	}

	// Generate random nonce
	nonce := make([]byte, e.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	// Encrypt and prepend nonce
	ciphertext := e.gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt decrypts ciphertext using AES-GCM.
func (e *AESEncryptor) Decrypt(ciphertext []byte) ([]byte, error) {
	nonceSize := e.gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, ErrInvalidCiphertext
	}

	// Extract nonce and ciphertext
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// Decrypt
	plaintext, err := e.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, ErrDecryptionFailed
	}

	return plaintext, nil
}

// NoOpEncryptor is a no-operation encryptor for when encryption is disabled.
type NoOpEncryptor struct{}

// NewNoOpEncryptor creates a new no-op encryptor.
func NewNoOpEncryptor() *NoOpEncryptor {
	return &NoOpEncryptor{}
}

// Encrypt returns the plaintext unchanged.
func (e *NoOpEncryptor) Encrypt(plaintext []byte) ([]byte, error) {
	result := make([]byte, len(plaintext))
	copy(result, plaintext)
	return result, nil
}

// Decrypt returns the ciphertext unchanged.
func (e *NoOpEncryptor) Decrypt(ciphertext []byte) ([]byte, error) {
	result := make([]byte, len(ciphertext))
	copy(result, ciphertext)
	return result, nil
}

// GenerateKey generates a random AES key of the specified size.
func GenerateKey(size int) ([]byte, error) {
	if size != 16 && size != 24 && size != 32 {
		return nil, ErrInvalidKey
	}

	key := make([]byte, size)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, err
	}

	return key, nil
}

// GenerateKeyString generates a random AES key and returns it as base64.
func GenerateKeyString(size int) (string, error) {
	key, err := GenerateKey(size)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(key), nil
}
