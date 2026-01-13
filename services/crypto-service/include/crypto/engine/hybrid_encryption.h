#pragma once

/**
 * @file hybrid_encryption.h
 * @brief Hybrid encryption combining RSA key wrapping with AES-GCM
 * 
 * Requirements: 5.2
 */

#include "crypto/common/result.h"
#include "crypto/engine/aes_engine.h"
#include "crypto/engine/rsa_engine.h"
#include <vector>
#include <span>
#include <cstdint>

namespace crypto {

// Hybrid encryption result containing wrapped key and encrypted data
struct HybridEncryptResult {
    std::vector<uint8_t> wrapped_key;    // RSA-encrypted AES key
    std::vector<uint8_t> ciphertext;     // AES-GCM encrypted data
    std::vector<uint8_t> iv;             // AES-GCM IV
    std::vector<uint8_t> tag;            // AES-GCM authentication tag
};

// Hybrid Encryption: RSA for key wrapping, AES for data encryption
// Allows encrypting arbitrary-sized data with RSA key pairs
class HybridEncryption {
public:
    HybridEncryption() = default;
    ~HybridEncryption() = default;

    // Non-copyable
    HybridEncryption(const HybridEncryption&) = delete;
    HybridEncryption& operator=(const HybridEncryption&) = delete;

    // Encrypt data using hybrid encryption
    // 1. Generate random AES-256 key
    // 2. Encrypt data with AES-256-GCM
    // 3. Wrap AES key with RSA-OAEP
    [[nodiscard]] Result<HybridEncryptResult> encrypt(
        std::span<const uint8_t> plaintext,
        const RSAKeyPair& public_key,
        std::span<const uint8_t> aad = {});

    // Decrypt data using hybrid encryption
    // 1. Unwrap AES key with RSA-OAEP
    // 2. Decrypt data with AES-256-GCM
    [[nodiscard]] Result<std::vector<uint8_t>> decrypt(
        const HybridEncryptResult& encrypted,
        const RSAKeyPair& private_key,
        std::span<const uint8_t> aad = {});

    // Decrypt from components
    [[nodiscard]] Result<std::vector<uint8_t>> decrypt(
        std::span<const uint8_t> wrapped_key,
        std::span<const uint8_t> ciphertext,
        std::span<const uint8_t> iv,
        std::span<const uint8_t> tag,
        const RSAKeyPair& private_key,
        std::span<const uint8_t> aad = {});

private:
    AESEngine aes_engine_;
    RSAEngine rsa_engine_;
};

} // namespace crypto
