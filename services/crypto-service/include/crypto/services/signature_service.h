#pragma once

/**
 * @file signature_service.h
 * @brief High-level signature service with LoggingClient integration
 * 
 * Requirements: 1.2, 1.4
 */

#include "crypto/common/result.h"
#include "crypto/common/hash_utils.h"
#include "crypto/engine/rsa_engine.h"
#include "crypto/engine/ecdsa_engine.h"
#include "crypto/keys/key_service.h"
#include "crypto/clients/logging_client.h"
#include <memory>
#include <string>
#include <span>

namespace crypto {

// Signature context
struct SignatureContext {
    std::string correlation_id;
    std::string caller_identity;
    std::string caller_service;
    std::string source_ip;
};

// Signature result
struct SignatureResult {
    std::vector<uint8_t> signature;
    KeyId key_id;
    std::string algorithm;
    HashAlgorithm hash_algorithm;
};

// Verification result
struct VerificationResult {
    bool valid;
    KeyId key_id;
    std::string algorithm;
};

// High-level signature service
class SignatureService {
public:
    SignatureService(std::shared_ptr<KeyService> key_service,
                     std::shared_ptr<LoggingClient> logging_client);
    ~SignatureService() = default;

    // Sign data using RSA-PSS
    [[nodiscard]] Result<SignatureResult> signRSA(
        std::span<const uint8_t> data,
        const KeyId& key_id,
        HashAlgorithm hash_algo,
        const SignatureContext& ctx);

    // Verify RSA-PSS signature
    [[nodiscard]] Result<VerificationResult> verifyRSA(
        std::span<const uint8_t> data,
        std::span<const uint8_t> signature,
        const KeyId& key_id,
        HashAlgorithm hash_algo,
        const SignatureContext& ctx);

    // Sign data using ECDSA
    [[nodiscard]] Result<SignatureResult> signECDSA(
        std::span<const uint8_t> data,
        const KeyId& key_id,
        const SignatureContext& ctx);

    // Verify ECDSA signature
    [[nodiscard]] Result<VerificationResult> verifyECDSA(
        std::span<const uint8_t> data,
        std::span<const uint8_t> signature,
        const KeyId& key_id,
        const SignatureContext& ctx);

    // Auto-detect key type and sign
    [[nodiscard]] Result<SignatureResult> sign(
        std::span<const uint8_t> data,
        const KeyId& key_id,
        const SignatureContext& ctx);

    // Auto-detect key type and verify
    [[nodiscard]] Result<VerificationResult> verify(
        std::span<const uint8_t> data,
        std::span<const uint8_t> signature,
        const KeyId& key_id,
        const SignatureContext& ctx);

private:
    std::shared_ptr<KeyService> key_service_;
    std::shared_ptr<LoggingClient> logging_client_;
    RSAEngine rsa_engine_;
    ECDSAEngine ecdsa_engine_;

    void logOperation(std::string_view operation, const KeyId& key_id,
                      const SignatureContext& ctx, bool success,
                      const std::optional<std::string>& error = std::nullopt);
};

} // namespace crypto
