/**
 * @file rsa_engine.h
 * @brief RSA encryption engine using centralized utilities
 * 
 * Requirements: 4.3, 4.4, 4.5, 5.2, 5.6
 */

#pragma once

#include "crypto/common/result.h"
#include "crypto/common/hash_utils.h"
#include "crypto/common/secure_memory.h"
#include <vector>
#include <span>
#include <cstdint>
#include <memory>

// Forward declarations for OpenSSL types
typedef struct evp_pkey_st EVP_PKEY;

namespace crypto {

// RSA key pair wrapper
class RSAKeyPair {
public:
    RSAKeyPair();
    ~RSAKeyPair();
    
    // Move only
    RSAKeyPair(RSAKeyPair&& other) noexcept;
    RSAKeyPair& operator=(RSAKeyPair&& other) noexcept;
    RSAKeyPair(const RSAKeyPair&) = delete;
    RSAKeyPair& operator=(const RSAKeyPair&) = delete;

    // Access internal key
    [[nodiscard]] EVP_PKEY* get() const noexcept { return key_; }
    [[nodiscard]] bool isValid() const noexcept { return key_ != nullptr; }

    // Get key size in bits
    [[nodiscard]] size_t keySize() const noexcept;

    // Export keys
    [[nodiscard]] Result<std::vector<uint8_t>> exportPublicKeyDER() const;
    [[nodiscard]] Result<std::vector<uint8_t>> exportPrivateKeyDER() const;
    [[nodiscard]] Result<std::string> exportPublicKeyPEM() const;
    [[nodiscard]] Result<std::string> exportPrivateKeyPEM() const;

    // Import keys
    [[nodiscard]] static Result<RSAKeyPair> importPublicKeyDER(std::span<const uint8_t> der);
    [[nodiscard]] static Result<RSAKeyPair> importPrivateKeyDER(std::span<const uint8_t> der);
    [[nodiscard]] static Result<RSAKeyPair> importPublicKeyPEM(std::string_view pem);
    [[nodiscard]] static Result<RSAKeyPair> importPrivateKeyPEM(std::string_view pem);

    // Maximum plaintext size for OAEP encryption
    [[nodiscard]] size_t maxPlaintextSize(HashAlgorithm hash_algo = HashAlgorithm::SHA256) const noexcept;

private:
    friend class RSAEngine;
    explicit RSAKeyPair(EVP_PKEY* key);
    EVP_PKEY* key_;
};

// RSA Engine for asymmetric encryption and signatures
class RSAEngine {
public:
    RSAEngine() = default;
    ~RSAEngine() = default;

    // Non-copyable
    RSAEngine(const RSAEngine&) = delete;
    RSAEngine& operator=(const RSAEngine&) = delete;

    // Key generation
    [[nodiscard]] Result<RSAKeyPair> generateKeyPair(RSAKeySize key_size);

    // OAEP Encryption (RSA-OAEP with configurable hash)
    [[nodiscard]] Result<std::vector<uint8_t>> encryptOAEP(
        std::span<const uint8_t> plaintext,
        const RSAKeyPair& public_key,
        HashAlgorithm hash_algo = HashAlgorithm::SHA256);

    // OAEP Decryption
    [[nodiscard]] Result<std::vector<uint8_t>> decryptOAEP(
        std::span<const uint8_t> ciphertext,
        const RSAKeyPair& private_key,
        HashAlgorithm hash_algo = HashAlgorithm::SHA256);

    // PSS Signatures (RSA-PSS)
    [[nodiscard]] Result<std::vector<uint8_t>> signPSS(
        std::span<const uint8_t> data,
        const RSAKeyPair& private_key,
        HashAlgorithm hash_algo = HashAlgorithm::SHA256);

    // PSS Verification
    [[nodiscard]] Result<bool> verifyPSS(
        std::span<const uint8_t> data,
        std::span<const uint8_t> signature,
        const RSAKeyPair& public_key,
        HashAlgorithm hash_algo = HashAlgorithm::SHA256);

    // Utility - use centralized function
    [[nodiscard]] static bool isValidKeySize(size_t bits) noexcept {
        return is_valid_rsa_key_size(bits);
    }
};

} // namespace crypto
