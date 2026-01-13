// Package crypto provides cryptographic operations via the crypto-service.
package crypto

import (
	"fmt"
	"strconv"
	"strings"
)

// KeyID identifies a cryptographic key in the crypto-service.
type KeyID struct {
	Namespace string
	ID        string
	Version   uint32
}

// ParseKeyID parses a key ID from string format "namespace/id/version".
func ParseKeyID(s string) (KeyID, error) {
	parts := strings.Split(s, "/")
	if len(parts) != 3 {
		return KeyID{}, fmt.Errorf("invalid key ID format: expected 'namespace/id/version', got %q", s)
	}

	if parts[0] == "" {
		return KeyID{}, fmt.Errorf("key namespace cannot be empty")
	}
	if parts[1] == "" {
		return KeyID{}, fmt.Errorf("key id cannot be empty")
	}

	version, err := strconv.ParseUint(parts[2], 10, 32)
	if err != nil {
		return KeyID{}, fmt.Errorf("invalid key version %q: %w", parts[2], err)
	}

	return KeyID{
		Namespace: parts[0],
		ID:        parts[1],
		Version:   uint32(version),
	}, nil
}

// String returns the string representation of the key ID.
func (k KeyID) String() string {
	return fmt.Sprintf("%s/%s/%d", k.Namespace, k.ID, k.Version)
}

// IsZero returns true if the key ID is empty.
func (k KeyID) IsZero() bool {
	return k.Namespace == "" && k.ID == "" && k.Version == 0
}

// EncryptResult holds the result of an encryption operation.
type EncryptResult struct {
	Ciphertext []byte
	IV         []byte
	Tag        []byte
	KeyID      KeyID
	Algorithm  string
}

// DecryptResult holds the result of a decryption operation.
type DecryptResult struct {
	Plaintext []byte
}

// SignResult holds the result of a signing operation.
type SignResult struct {
	Signature []byte
	KeyID     KeyID
	Algorithm string
}

// VerifyResult holds the result of a verification operation.
type VerifyResult struct {
	Valid bool
	KeyID KeyID
}

// HealthStatus holds the health status of the crypto-service.
type HealthStatus struct {
	Connected     bool
	HSMConnected  bool
	KMSConnected  bool
	Version       string
	UptimeSeconds int64
	LatencyMs     int64
}

// CryptoError represents an error from crypto operations.
type CryptoError struct {
	Code          string
	Message       string
	CorrelationID string
	Cause         error
}

// Error codes for crypto operations.
const (
	ErrCodeEncryptionFailed   = "ENCRYPTION_FAILED"
	ErrCodeDecryptionFailed   = "DECRYPTION_FAILED"
	ErrCodeSignatureFailed    = "SIGNATURE_FAILED"
	ErrCodeSignatureInvalid   = "SIGNATURE_INVALID"
	ErrCodeKeyNotFound        = "KEY_NOT_FOUND"
	ErrCodeServiceUnavailable = "CRYPTO_SERVICE_UNAVAILABLE"
	ErrCodeAADMismatch        = "AAD_MISMATCH"
	ErrCodeInvalidInput       = "INVALID_INPUT"
	ErrCodeNotImplemented     = "NOT_IMPLEMENTED"
)

// Error implements the error interface.
func (e *CryptoError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (correlation_id=%s): %v", e.Code, e.Message, e.CorrelationID, e.Cause)
	}
	return fmt.Sprintf("%s: %s (correlation_id=%s)", e.Code, e.Message, e.CorrelationID)
}

// Unwrap returns the underlying error.
func (e *CryptoError) Unwrap() error {
	return e.Cause
}

// NewCryptoError creates a new CryptoError.
func NewCryptoError(code, message, correlationID string, cause error) *CryptoError {
	return &CryptoError{
		Code:          code,
		Message:       message,
		CorrelationID: correlationID,
		Cause:         cause,
	}
}

// IsServiceUnavailable returns true if the error indicates the crypto-service is unavailable.
func IsServiceUnavailable(err error) bool {
	if cryptoErr, ok := err.(*CryptoError); ok {
		return cryptoErr.Code == ErrCodeServiceUnavailable
	}
	return false
}

// IsAADMismatch returns true if the error indicates an AAD mismatch.
func IsAADMismatch(err error) bool {
	if cryptoErr, ok := err.(*CryptoError); ok {
		return cryptoErr.Code == ErrCodeAADMismatch
	}
	return false
}

// IsSignatureInvalid returns true if the error indicates an invalid signature.
func IsSignatureInvalid(err error) bool {
	if cryptoErr, ok := err.(*CryptoError); ok {
		return cryptoErr.Code == ErrCodeSignatureInvalid
	}
	return false
}
