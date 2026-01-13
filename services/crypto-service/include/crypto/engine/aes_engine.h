/**
 * @file aes_engine.h
 * @brief AES encryption engine using centralized utilities
 * 
 * Requirements: 4.3, 4.4, 5.2, 5.6
 */

#pragma once

#include "crypto/common/result.h"
#include "crypto/common/hash_utils.h"
#include "crypto/common/secure_memory.h"
#include <vector>
#include <span>
#include <cstdint>

namespace crypto {

// Encryption result containing ciphertext, IV, and tag
struct EncryptResult {
    std::vector<uint8_t> ciphertext;
    std::vector<uint8_t> iv;
    std::vector<uint8_t> tag;  // For GCM mode only
};

// AES Engine for symmetric encryption
class AESEngine {
public:
    // Use centralized constants from hash_utils.h
    static constexpr size_t GCM_IV_SIZE = aes_gcm::IV_SIZE;
    static constexpr size_t GCM_TAG_SIZE = aes_gcm::TAG_SIZE;
    static constexpr size_t CBC_IV_SIZE = aes_cbc::IV_SIZE;
    static constexpr size_t BLOCK_SIZE = aes_gcm::BLOCK_SIZE;

    AESEngine() = default;
    ~AESEngine() = default;

    // Non-copyable, non-movable (stateless)
    AESEngine(const AESEngine&) = delete;
    AESEngine& operator=(const AESEngine&) = delete;

    // GCM Mode Operations
    
    [[nodiscard]] Result<EncryptResult> encryptGCM(
        std::span<const uint8_t> plaintext,
        std::span<const uint8_t> key,
        std::span<const uint8_t> aad = {});

    [[nodiscard]] Result<EncryptResult> encryptGCMWithIV(
        std::span<const uint8_t> plaintext,
        std::span<const uint8_t> key,
        std::span<const uint8_t> iv,
        std::span<const uint8_t> aad = {});

    [[nodiscard]] Result<std::vector<uint8_t>> decryptGCM(
        std::span<const uint8_t> ciphertext,
        std::span<const uint8_t> key,
        std::span<const uint8_t> iv,
        std::span<const uint8_t> tag,
        std::span<const uint8_t> aad = {});

    // CBC Mode Operations (legacy compatibility)
    
    [[nodiscard]] Result<EncryptResult> encryptCBC(
        std::span<const uint8_t> plaintext,
        std::span<const uint8_t> key);

    [[nodiscard]] Result<EncryptResult> encryptCBCWithIV(
        std::span<const uint8_t> plaintext,
        std::span<const uint8_t> key,
        std::span<const uint8_t> iv);

    [[nodiscard]] Result<std::vector<uint8_t>> decryptCBC(
        std::span<const uint8_t> ciphertext,
        std::span<const uint8_t> key,
        std::span<const uint8_t> iv);

    // Utility functions
    
    [[nodiscard]] static Result<std::vector<uint8_t>> generateIV(size_t size);
    [[nodiscard]] static Result<SecureBuffer> generateKey(AESKeySize key_size);
    [[nodiscard]] static bool isValidKeySize(size_t size) noexcept;

private:
    static std::vector<uint8_t> addPKCS7Padding(std::span<const uint8_t> data);
    [[nodiscard]] static Result<std::vector<uint8_t>> removePKCS7Padding(
        std::span<const uint8_t> data);
};

} // namespace crypto
