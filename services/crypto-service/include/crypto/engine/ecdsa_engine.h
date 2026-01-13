#pragma once

/**
 * @file ecdsa_engine.h
 * @brief ECDSA signature engine using centralized utilities
 * 
 * Requirements: 4.3, 4.4, 4.5, 5.2
 */

#include "crypto/common/result.h"
#include "crypto/common/hash_utils.h"
#include <vector>
#include <span>
#include <cstdint>
#include <memory>

// Forward declarations for OpenSSL types
typedef struct evp_pkey_st EVP_PKEY;

namespace crypto {

// Note: ECCurve is now defined in hash_utils.h for centralization

// EC key pair wrapper
class ECKeyPair {
public:
    ECKeyPair();
    ~ECKeyPair();
    
    // Move only
    ECKeyPair(ECKeyPair&& other) noexcept;
    ECKeyPair& operator=(ECKeyPair&& other) noexcept;
    ECKeyPair(const ECKeyPair&) = delete;
    ECKeyPair& operator=(const ECKeyPair&) = delete;

    // Access internal key
    EVP_PKEY* get() const { return key_; }
    bool isValid() const { return key_ != nullptr; }

    // Get curve
    ECCurve curve() const { return curve_; }

    // Export keys
    [[nodiscard]] Result<std::vector<uint8_t>> exportPublicKeyDER() const;
    [[nodiscard]] Result<std::vector<uint8_t>> exportPrivateKeyDER() const;
    [[nodiscard]] Result<std::string> exportPublicKeyPEM() const;
    [[nodiscard]] Result<std::string> exportPrivateKeyPEM() const;

    // Import keys
    [[nodiscard]] static Result<ECKeyPair> importPublicKeyDER(std::span<const uint8_t> der, ECCurve curve);
    [[nodiscard]] static Result<ECKeyPair> importPrivateKeyDER(std::span<const uint8_t> der, ECCurve curve);
    [[nodiscard]] static Result<ECKeyPair> importPublicKeyPEM(std::string_view pem, ECCurve curve);
    [[nodiscard]] static Result<ECKeyPair> importPrivateKeyPEM(std::string_view pem, ECCurve curve);

private:
    friend class ECDSAEngine;
    ECKeyPair(EVP_PKEY* key, ECCurve curve);
    EVP_PKEY* key_;
    ECCurve curve_;
};

// ECDSA Engine for elliptic curve signatures
class ECDSAEngine {
public:
    ECDSAEngine() = default;
    ~ECDSAEngine() = default;

    // Non-copyable
    ECDSAEngine(const ECDSAEngine&) = delete;
    ECDSAEngine& operator=(const ECDSAEngine&) = delete;

    // Key generation
    [[nodiscard]] Result<ECKeyPair> generateKeyPair(ECCurve curve);

    // Sign data with ECDSA
    [[nodiscard]] Result<std::vector<uint8_t>> sign(
        std::span<const uint8_t> data,
        const ECKeyPair& private_key);

    // Verify ECDSA signature
    [[nodiscard]] Result<bool> verify(
        std::span<const uint8_t> data,
        std::span<const uint8_t> signature,
        const ECKeyPair& public_key);

    // Get curve name (use get_curve_name from hash_utils.h instead)
    [[nodiscard]] static const char* curveName(ECCurve curve);

private:
    static int curveNID(ECCurve curve);
};

} // namespace crypto
