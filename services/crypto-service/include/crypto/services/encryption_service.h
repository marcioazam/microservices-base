#pragma once

/**
 * @file encryption_service.h
 * @brief High-level encryption service with LoggingClient integration
 * 
 * Requirements: 1.2, 1.4
 */

#include "crypto/common/result.h"
#include "crypto/engine/aes_engine.h"
#include "crypto/keys/key_service.h"
#include "crypto/clients/logging_client.h"
#include <memory>
#include <string>
#include <span>

namespace crypto {

// Encryption request context
struct EncryptionContext {
    std::string correlation_id;
    std::string caller_identity;
    std::string caller_service;
    std::string source_ip;
    std::optional<std::span<const uint8_t>> aad;  // Additional authenticated data
};

// Encryption result with metadata
struct EncryptionResult {
    std::vector<uint8_t> ciphertext;
    std::vector<uint8_t> iv;
    std::vector<uint8_t> tag;
    KeyId key_id;
    std::string algorithm;
};

// Decryption request
struct DecryptionRequest {
    std::span<const uint8_t> ciphertext;
    std::span<const uint8_t> iv;
    std::span<const uint8_t> tag;
    KeyId key_id;
    std::optional<std::span<const uint8_t>> aad;
};

// High-level encryption service
class EncryptionService {
public:
    EncryptionService(std::shared_ptr<KeyService> key_service,
                      std::shared_ptr<LoggingClient> logging_client);
    ~EncryptionService() = default;

    // Encrypt data using AES-GCM with specified key
    [[nodiscard]] Result<EncryptionResult> encrypt(
        std::span<const uint8_t> plaintext,
        const KeyId& key_id,
        const EncryptionContext& ctx);

    // Encrypt data using AES-GCM with auto-generated key
    [[nodiscard]] Result<EncryptionResult> encryptWithNewKey(
        std::span<const uint8_t> plaintext,
        const std::string& key_namespace,
        const EncryptionContext& ctx);

    // Decrypt data
    [[nodiscard]] Result<std::vector<uint8_t>> decrypt(
        const DecryptionRequest& request,
        const EncryptionContext& ctx);

    // Encrypt using AES-CBC (legacy mode)
    [[nodiscard]] Result<EncryptionResult> encryptCBC(
        std::span<const uint8_t> plaintext,
        const KeyId& key_id,
        const EncryptionContext& ctx);

    // Decrypt using AES-CBC (legacy mode)
    [[nodiscard]] Result<std::vector<uint8_t>> decryptCBC(
        std::span<const uint8_t> ciphertext,
        std::span<const uint8_t> iv,
        const KeyId& key_id,
        const EncryptionContext& ctx);

private:
    std::shared_ptr<KeyService> key_service_;
    std::shared_ptr<LoggingClient> logging_client_;
    AESEngine aes_engine_;

    void logOperation(std::string_view operation, const KeyId& key_id, 
                      const EncryptionContext& ctx, bool success,
                      const std::optional<std::string>& error = std::nullopt);
};

} // namespace crypto
