#pragma once

/**
 * @file hash_utils.h
 * @brief Centralized hash algorithm utilities
 * 
 * This header provides a single source of truth for hash algorithm
 * selection, sizes, and names used throughout the crypto-service.
 * 
 * Requirements: 4.5, 5.5
 */

#include <openssl/evp.h>
#include <cstddef>
#include <string_view>
#include <utility>

namespace crypto {

// ============================================================================
// Hash Algorithm Enumeration
// ============================================================================

/**
 * @brief Supported hash algorithms
 */
enum class HashAlgorithm {
    SHA256,
    SHA384,
    SHA512
};

// ============================================================================
// Elliptic Curve Enumeration
// ============================================================================

/**
 * @brief Supported elliptic curves for ECDSA
 */
enum class ECCurve {
    P256,   // secp256r1 / prime256v1
    P384,   // secp384r1
    P521    // secp521r1
};

// ============================================================================
// Hash Algorithm Utilities
// ============================================================================

/**
 * @brief Get the EVP_MD for a hash algorithm
 * @param algo Hash algorithm
 * @return Pointer to EVP_MD (never null for valid input)
 */
[[nodiscard]] constexpr const EVP_MD* get_evp_md(HashAlgorithm algo) noexcept {
    switch (algo) {
        case HashAlgorithm::SHA256: return EVP_sha256();
        case HashAlgorithm::SHA384: return EVP_sha384();
        case HashAlgorithm::SHA512: return EVP_sha512();
    }
    // Unreachable for valid enum values
    return EVP_sha256();
}

/**
 * @brief Get the output size in bytes for a hash algorithm
 * @param algo Hash algorithm
 * @return Hash output size in bytes
 */
[[nodiscard]] constexpr size_t get_hash_size(HashAlgorithm algo) noexcept {
    switch (algo) {
        case HashAlgorithm::SHA256: return 32;
        case HashAlgorithm::SHA384: return 48;
        case HashAlgorithm::SHA512: return 64;
    }
    return 32;
}

/**
 * @brief Get the name of a hash algorithm
 * @param algo Hash algorithm
 * @return Algorithm name as string view
 */
[[nodiscard]] constexpr std::string_view get_hash_name(HashAlgorithm algo) noexcept {
    switch (algo) {
        case HashAlgorithm::SHA256: return "SHA256";
        case HashAlgorithm::SHA384: return "SHA384";
        case HashAlgorithm::SHA512: return "SHA512";
    }
    return "SHA256";
}

/**
 * @brief Get the OpenSSL NID for a hash algorithm
 * @param algo Hash algorithm
 * @return OpenSSL NID
 */
[[nodiscard]] constexpr int get_hash_nid(HashAlgorithm algo) noexcept {
    switch (algo) {
        case HashAlgorithm::SHA256: return NID_sha256;
        case HashAlgorithm::SHA384: return NID_sha384;
        case HashAlgorithm::SHA512: return NID_sha512;
    }
    return NID_sha256;
}

// ============================================================================
// Elliptic Curve Utilities
// ============================================================================

/**
 * @brief Get the appropriate hash algorithm for an elliptic curve
 * 
 * NIST recommends using hash functions with output size matching
 * the curve's security level:
 * - P-256: SHA-256 (128-bit security)
 * - P-384: SHA-384 (192-bit security)
 * - P-521: SHA-512 (256-bit security)
 * 
 * @param curve Elliptic curve
 * @return Recommended hash algorithm
 */
[[nodiscard]] constexpr HashAlgorithm get_hash_for_curve(ECCurve curve) noexcept {
    switch (curve) {
        case ECCurve::P256: return HashAlgorithm::SHA256;
        case ECCurve::P384: return HashAlgorithm::SHA384;
        case ECCurve::P521: return HashAlgorithm::SHA512;
    }
    return HashAlgorithm::SHA256;
}

/**
 * @brief Get the EVP_MD for an elliptic curve
 * @param curve Elliptic curve
 * @return Pointer to EVP_MD
 */
[[nodiscard]] constexpr const EVP_MD* get_evp_md_for_curve(ECCurve curve) noexcept {
    return get_evp_md(get_hash_for_curve(curve));
}

/**
 * @brief Get the OpenSSL NID for an elliptic curve
 * @param curve Elliptic curve
 * @return OpenSSL NID
 */
[[nodiscard]] constexpr int get_curve_nid(ECCurve curve) noexcept {
    switch (curve) {
        case ECCurve::P256: return NID_X9_62_prime256v1;
        case ECCurve::P384: return NID_secp384r1;
        case ECCurve::P521: return NID_secp521r1;
    }
    return NID_X9_62_prime256v1;
}

/**
 * @brief Get the name of an elliptic curve
 * @param curve Elliptic curve
 * @return Curve name as string view
 */
[[nodiscard]] constexpr std::string_view get_curve_name(ECCurve curve) noexcept {
    switch (curve) {
        case ECCurve::P256: return "P-256";
        case ECCurve::P384: return "P-384";
        case ECCurve::P521: return "P-521";
    }
    return "P-256";
}

/**
 * @brief Get the key size in bits for an elliptic curve
 * @param curve Elliptic curve
 * @return Key size in bits
 */
[[nodiscard]] constexpr size_t get_curve_key_bits(ECCurve curve) noexcept {
    switch (curve) {
        case ECCurve::P256: return 256;
        case ECCurve::P384: return 384;
        case ECCurve::P521: return 521;
    }
    return 256;
}

/**
 * @brief Get the signature size in bytes for an elliptic curve (DER encoded max)
 * @param curve Elliptic curve
 * @return Maximum signature size in bytes
 */
[[nodiscard]] constexpr size_t get_curve_signature_size(ECCurve curve) noexcept {
    // DER encoded ECDSA signature: 2 * (key_bits/8) + overhead
    switch (curve) {
        case ECCurve::P256: return 72;   // 2*32 + 8 overhead
        case ECCurve::P384: return 104;  // 2*48 + 8 overhead
        case ECCurve::P521: return 139;  // 2*66 + 7 overhead
    }
    return 72;
}

// ============================================================================
// RSA Utilities
// ============================================================================

/**
 * @brief Supported RSA key sizes
 */
enum class RSAKeySize {
    RSA_2048 = 2048,
    RSA_3072 = 3072,
    RSA_4096 = 4096
};

/**
 * @brief Check if an RSA key size is valid
 * @param bits Key size in bits
 * @return true if valid
 */
[[nodiscard]] constexpr bool is_valid_rsa_key_size(size_t bits) noexcept {
    return bits == 2048 || bits == 3072 || bits == 4096;
}

/**
 * @brief Get the maximum plaintext size for RSA-OAEP encryption
 * 
 * For OAEP with SHA-256: max_size = key_bytes - 2*hash_size - 2
 * 
 * @param key_bits RSA key size in bits
 * @param hash_algo Hash algorithm used for OAEP
 * @return Maximum plaintext size in bytes
 */
[[nodiscard]] constexpr size_t get_rsa_oaep_max_plaintext(
    size_t key_bits, 
    HashAlgorithm hash_algo = HashAlgorithm::SHA256) noexcept {
    
    size_t key_bytes = key_bits / 8;
    size_t hash_size = get_hash_size(hash_algo);
    return key_bytes - 2 * hash_size - 2;
}

// ============================================================================
// AES Utilities
// ============================================================================

/**
 * @brief Supported AES key sizes
 */
enum class AESKeySize {
    AES_128 = 16,
    AES_256 = 32
};

/**
 * @brief Check if an AES key size is valid
 * @param bytes Key size in bytes
 * @return true if valid
 */
[[nodiscard]] constexpr bool is_valid_aes_key_size(size_t bytes) noexcept {
    return bytes == 16 || bytes == 32;
}

/**
 * @brief AES-GCM constants
 */
namespace aes_gcm {
    constexpr size_t IV_SIZE = 12;    // 96 bits (NIST recommended)
    constexpr size_t TAG_SIZE = 16;   // 128 bits
    constexpr size_t BLOCK_SIZE = 16; // 128 bits
}

/**
 * @brief AES-CBC constants
 */
namespace aes_cbc {
    constexpr size_t IV_SIZE = 16;    // 128 bits
    constexpr size_t BLOCK_SIZE = 16; // 128 bits
}

} // namespace crypto
