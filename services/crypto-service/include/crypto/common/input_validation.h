/**
 * @file input_validation.h
 * @brief Input validation utilities for security hardening
 * 
 * Provides centralized input validation to prevent:
 * - Buffer overflow attacks
 * - Denial of service via oversized inputs
 * - Information leakage through error messages
 * 
 * Requirements: 10.5, 10.6
 */

#pragma once

#include "crypto/common/result.h"
#include <span>
#include <cstdint>
#include <cstddef>

namespace crypto {

// ============================================================================
// Size Limits (Requirement 10.5)
// ============================================================================

namespace limits {

/**
 * @brief Maximum plaintext size for symmetric encryption (64 MB)
 * 
 * Prevents DoS attacks via memory exhaustion.
 */
constexpr size_t MAX_PLAINTEXT_SIZE = 64 * 1024 * 1024;

/**
 * @brief Maximum ciphertext size for decryption (64 MB + overhead)
 */
constexpr size_t MAX_CIPHERTEXT_SIZE = 64 * 1024 * 1024 + 1024;

/**
 * @brief Maximum data size for signing (16 MB)
 */
constexpr size_t MAX_SIGN_DATA_SIZE = 16 * 1024 * 1024;

/**
 * @brief Maximum file size for file encryption (1 GB)
 */
constexpr size_t MAX_FILE_SIZE = 1024 * 1024 * 1024;

/**
 * @brief Maximum RSA plaintext size (depends on key size)
 */
constexpr size_t MAX_RSA_PLAINTEXT_SIZE = 446;  // 4096-bit key with OAEP-SHA256

/**
 * @brief Maximum key material size (8 KB)
 */
constexpr size_t MAX_KEY_SIZE = 8 * 1024;

/**
 * @brief Maximum AAD size for AEAD (64 KB)
 */
constexpr size_t MAX_AAD_SIZE = 64 * 1024;

/**
 * @brief Maximum signature size (1 KB)
 */
constexpr size_t MAX_SIGNATURE_SIZE = 1024;

} // namespace limits

// ============================================================================
// Validation Functions
// ============================================================================

/**
 * @brief Validate plaintext size for symmetric encryption
 * @param size Size in bytes
 * @return Result<void> - success or SIZE_LIMIT_EXCEEDED error
 */
[[nodiscard]] inline Result<void> validatePlaintextSize(size_t size) {
    if (size > limits::MAX_PLAINTEXT_SIZE) {
        return Err<void>(ErrorCode::SIZE_LIMIT_EXCEEDED, 
                         "Input exceeds maximum allowed size");
    }
    return Ok();
}

/**
 * @brief Validate ciphertext size for decryption
 * @param size Size in bytes
 * @return Result<void> - success or SIZE_LIMIT_EXCEEDED error
 */
[[nodiscard]] inline Result<void> validateCiphertextSize(size_t size) {
    if (size > limits::MAX_CIPHERTEXT_SIZE) {
        return Err<void>(ErrorCode::SIZE_LIMIT_EXCEEDED,
                         "Ciphertext exceeds maximum allowed size");
    }
    return Ok();
}

/**
 * @brief Validate data size for signing
 * @param size Size in bytes
 * @return Result<void> - success or SIZE_LIMIT_EXCEEDED error
 */
[[nodiscard]] inline Result<void> validateSignDataSize(size_t size) {
    if (size > limits::MAX_SIGN_DATA_SIZE) {
        return Err<void>(ErrorCode::SIZE_LIMIT_EXCEEDED,
                         "Data exceeds maximum size for signing");
    }
    return Ok();
}

/**
 * @brief Validate file size for file encryption
 * @param size Size in bytes
 * @return Result<void> - success or SIZE_LIMIT_EXCEEDED error
 */
[[nodiscard]] inline Result<void> validateFileSize(size_t size) {
    if (size > limits::MAX_FILE_SIZE) {
        return Err<void>(ErrorCode::SIZE_LIMIT_EXCEEDED,
                         "File exceeds maximum allowed size");
    }
    return Ok();
}

/**
 * @brief Validate AAD size for AEAD encryption
 * @param size Size in bytes
 * @return Result<void> - success or SIZE_LIMIT_EXCEEDED error
 */
[[nodiscard]] inline Result<void> validateAADSize(size_t size) {
    if (size > limits::MAX_AAD_SIZE) {
        return Err<void>(ErrorCode::SIZE_LIMIT_EXCEEDED,
                         "AAD exceeds maximum allowed size");
    }
    return Ok();
}

/**
 * @brief Validate AES key size
 * @param size Size in bytes
 * @return Result<void> - success or INVALID_KEY_SIZE error
 */
[[nodiscard]] inline Result<void> validateAESKeySize(size_t size) {
    if (size != 16 && size != 32) {
        return Err<void>(ErrorCode::INVALID_KEY_SIZE,
                         "AES key must be 128 or 256 bits");
    }
    return Ok();
}

/**
 * @brief Validate RSA key size
 * @param bits Size in bits
 * @return Result<void> - success or INVALID_KEY_SIZE error
 */
[[nodiscard]] inline Result<void> validateRSAKeySize(size_t bits) {
    if (bits != 2048 && bits != 3072 && bits != 4096) {
        return Err<void>(ErrorCode::INVALID_KEY_SIZE,
                         "RSA key must be 2048, 3072, or 4096 bits");
    }
    return Ok();
}

/**
 * @brief Validate GCM IV size
 * @param size Size in bytes
 * @return Result<void> - success or INVALID_IV_SIZE error
 */
[[nodiscard]] inline Result<void> validateGCMIVSize(size_t size) {
    if (size != 12) {
        return Err<void>(ErrorCode::INVALID_IV_SIZE,
                         "GCM IV must be 96 bits");
    }
    return Ok();
}

/**
 * @brief Validate GCM tag size
 * @param size Size in bytes
 * @return Result<void> - success or INVALID_TAG_SIZE error
 */
[[nodiscard]] inline Result<void> validateGCMTagSize(size_t size) {
    if (size != 16) {
        return Err<void>(ErrorCode::INVALID_TAG_SIZE,
                         "GCM tag must be 128 bits");
    }
    return Ok();
}

/**
 * @brief Validate CBC IV size
 * @param size Size in bytes
 * @return Result<void> - success or INVALID_IV_SIZE error
 */
[[nodiscard]] inline Result<void> validateCBCIVSize(size_t size) {
    if (size != 16) {
        return Err<void>(ErrorCode::INVALID_IV_SIZE,
                         "CBC IV must be 128 bits");
    }
    return Ok();
}

// ============================================================================
// Safe Error Messages (Requirement 10.6)
// ============================================================================

namespace safe_errors {

/**
 * @brief Generic encryption failure message (no details)
 */
constexpr const char* ENCRYPTION_FAILED = "Encryption operation failed";

/**
 * @brief Generic decryption failure message (no details)
 */
constexpr const char* DECRYPTION_FAILED = "Decryption operation failed";

/**
 * @brief Generic signature failure message (no details)
 */
constexpr const char* SIGNATURE_FAILED = "Signature operation failed";

/**
 * @brief Generic verification failure message (no details)
 */
constexpr const char* VERIFICATION_FAILED = "Signature verification failed";

/**
 * @brief Generic key operation failure message (no details)
 */
constexpr const char* KEY_OPERATION_FAILED = "Key operation failed";

/**
 * @brief Generic integrity failure message (no details about what failed)
 */
constexpr const char* INTEGRITY_FAILED = "Data integrity verification failed";

} // namespace safe_errors

/**
 * @brief Create a safe error that doesn't leak sensitive information
 * @param code Error code
 * @return Error with safe message
 */
[[nodiscard]] inline Error makeSafeError(ErrorCode code) {
    switch (code) {
        case ErrorCode::ENCRYPTION_FAILED:
            return Error(code, safe_errors::ENCRYPTION_FAILED);
        case ErrorCode::DECRYPTION_FAILED:
            return Error(code, safe_errors::DECRYPTION_FAILED);
        case ErrorCode::SIGNATURE_INVALID:
            return Error(code, safe_errors::VERIFICATION_FAILED);
        case ErrorCode::INTEGRITY_ERROR:
            return Error(code, safe_errors::INTEGRITY_FAILED);
        default:
            return Error(code, "Operation failed");
    }
}

} // namespace crypto
