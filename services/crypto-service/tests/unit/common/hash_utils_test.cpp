/**
 * @file hash_utils_test.cpp
 * @brief Unit tests for hash algorithm utilities
 * 
 * Requirements: 7.2
 */

#include <gtest/gtest.h>
#include <crypto/common/hash_utils.h>

namespace crypto::test {

// ============================================================================
// Hash Algorithm Tests
// ============================================================================

TEST(HashUtilsTest, GetEvpMdReturnsValidPointers) {
    EXPECT_NE(get_evp_md(HashAlgorithm::SHA256), nullptr);
    EXPECT_NE(get_evp_md(HashAlgorithm::SHA384), nullptr);
    EXPECT_NE(get_evp_md(HashAlgorithm::SHA512), nullptr);
}

TEST(HashUtilsTest, GetHashSizeReturnsCorrectValues) {
    EXPECT_EQ(get_hash_size(HashAlgorithm::SHA256), 32u);
    EXPECT_EQ(get_hash_size(HashAlgorithm::SHA384), 48u);
    EXPECT_EQ(get_hash_size(HashAlgorithm::SHA512), 64u);
}

TEST(HashUtilsTest, GetHashNameReturnsCorrectStrings) {
    EXPECT_EQ(get_hash_name(HashAlgorithm::SHA256), "SHA256");
    EXPECT_EQ(get_hash_name(HashAlgorithm::SHA384), "SHA384");
    EXPECT_EQ(get_hash_name(HashAlgorithm::SHA512), "SHA512");
}

TEST(HashUtilsTest, GetHashNidReturnsCorrectNids) {
    EXPECT_EQ(get_hash_nid(HashAlgorithm::SHA256), NID_sha256);
    EXPECT_EQ(get_hash_nid(HashAlgorithm::SHA384), NID_sha384);
    EXPECT_EQ(get_hash_nid(HashAlgorithm::SHA512), NID_sha512);
}

// ============================================================================
// Elliptic Curve Tests
// ============================================================================

TEST(HashUtilsTest, GetHashForCurveReturnsAppropriateHash) {
    // NIST recommendations: hash output should match curve security level
    EXPECT_EQ(get_hash_for_curve(ECCurve::P256), HashAlgorithm::SHA256);
    EXPECT_EQ(get_hash_for_curve(ECCurve::P384), HashAlgorithm::SHA384);
    EXPECT_EQ(get_hash_for_curve(ECCurve::P521), HashAlgorithm::SHA512);
}

TEST(HashUtilsTest, GetEvpMdForCurveReturnsValidPointers) {
    EXPECT_NE(get_evp_md_for_curve(ECCurve::P256), nullptr);
    EXPECT_NE(get_evp_md_for_curve(ECCurve::P384), nullptr);
    EXPECT_NE(get_evp_md_for_curve(ECCurve::P521), nullptr);
}

TEST(HashUtilsTest, GetCurveNidReturnsCorrectNids) {
    EXPECT_EQ(get_curve_nid(ECCurve::P256), NID_X9_62_prime256v1);
    EXPECT_EQ(get_curve_nid(ECCurve::P384), NID_secp384r1);
    EXPECT_EQ(get_curve_nid(ECCurve::P521), NID_secp521r1);
}

TEST(HashUtilsTest, GetCurveNameReturnsCorrectStrings) {
    EXPECT_EQ(get_curve_name(ECCurve::P256), "P-256");
    EXPECT_EQ(get_curve_name(ECCurve::P384), "P-384");
    EXPECT_EQ(get_curve_name(ECCurve::P521), "P-521");
}

TEST(HashUtilsTest, GetCurveKeyBitsReturnsCorrectValues) {
    EXPECT_EQ(get_curve_key_bits(ECCurve::P256), 256u);
    EXPECT_EQ(get_curve_key_bits(ECCurve::P384), 384u);
    EXPECT_EQ(get_curve_key_bits(ECCurve::P521), 521u);
}

TEST(HashUtilsTest, GetCurveSignatureSizeReturnsCorrectValues) {
    // DER encoded ECDSA signatures have variable size, these are maximums
    EXPECT_EQ(get_curve_signature_size(ECCurve::P256), 72u);
    EXPECT_EQ(get_curve_signature_size(ECCurve::P384), 104u);
    EXPECT_EQ(get_curve_signature_size(ECCurve::P521), 139u);
}

// ============================================================================
// RSA Utilities Tests
// ============================================================================

TEST(HashUtilsTest, IsValidRsaKeySizeAcceptsValidSizes) {
    EXPECT_TRUE(is_valid_rsa_key_size(2048));
    EXPECT_TRUE(is_valid_rsa_key_size(3072));
    EXPECT_TRUE(is_valid_rsa_key_size(4096));
}

TEST(HashUtilsTest, IsValidRsaKeySizeRejectsInvalidSizes) {
    EXPECT_FALSE(is_valid_rsa_key_size(512));
    EXPECT_FALSE(is_valid_rsa_key_size(1024));
    EXPECT_FALSE(is_valid_rsa_key_size(2000));
    EXPECT_FALSE(is_valid_rsa_key_size(8192));
}

TEST(HashUtilsTest, GetRsaOaepMaxPlaintextCalculatesCorrectly) {
    // RSA-OAEP with SHA-256: max = key_bytes - 2*32 - 2
    EXPECT_EQ(get_rsa_oaep_max_plaintext(2048, HashAlgorithm::SHA256), 
              256 - 64 - 2);  // 190 bytes
    EXPECT_EQ(get_rsa_oaep_max_plaintext(4096, HashAlgorithm::SHA256), 
              512 - 64 - 2);  // 446 bytes
    
    // RSA-OAEP with SHA-512: max = key_bytes - 2*64 - 2
    EXPECT_EQ(get_rsa_oaep_max_plaintext(4096, HashAlgorithm::SHA512), 
              512 - 128 - 2);  // 382 bytes
}

// ============================================================================
// AES Utilities Tests
// ============================================================================

TEST(HashUtilsTest, IsValidAesKeySizeAcceptsValidSizes) {
    EXPECT_TRUE(is_valid_aes_key_size(16));  // AES-128
    EXPECT_TRUE(is_valid_aes_key_size(32));  // AES-256
}

TEST(HashUtilsTest, IsValidAesKeySizeRejectsInvalidSizes) {
    EXPECT_FALSE(is_valid_aes_key_size(8));
    EXPECT_FALSE(is_valid_aes_key_size(24));  // AES-192 not supported
    EXPECT_FALSE(is_valid_aes_key_size(64));
}

TEST(HashUtilsTest, AesGcmConstantsAreCorrect) {
    EXPECT_EQ(aes_gcm::IV_SIZE, 12u);   // 96 bits
    EXPECT_EQ(aes_gcm::TAG_SIZE, 16u);  // 128 bits
    EXPECT_EQ(aes_gcm::BLOCK_SIZE, 16u);
}

TEST(HashUtilsTest, AesCbcConstantsAreCorrect) {
    EXPECT_EQ(aes_cbc::IV_SIZE, 16u);   // 128 bits
    EXPECT_EQ(aes_cbc::BLOCK_SIZE, 16u);
}

// ============================================================================
// Constexpr Tests (compile-time evaluation)
// ============================================================================

TEST(HashUtilsTest, FunctionsAreConstexpr) {
    // These should compile if functions are truly constexpr
    constexpr auto sha256_size = get_hash_size(HashAlgorithm::SHA256);
    constexpr auto sha256_name = get_hash_name(HashAlgorithm::SHA256);
    constexpr auto p256_hash = get_hash_for_curve(ECCurve::P256);
    constexpr auto p256_bits = get_curve_key_bits(ECCurve::P256);
    constexpr auto rsa_valid = is_valid_rsa_key_size(2048);
    constexpr auto aes_valid = is_valid_aes_key_size(32);
    constexpr auto oaep_max = get_rsa_oaep_max_plaintext(2048);
    
    EXPECT_EQ(sha256_size, 32u);
    EXPECT_EQ(sha256_name, "SHA256");
    EXPECT_EQ(p256_hash, HashAlgorithm::SHA256);
    EXPECT_EQ(p256_bits, 256u);
    EXPECT_TRUE(rsa_valid);
    EXPECT_TRUE(aes_valid);
    EXPECT_EQ(oaep_max, 190u);
}

// ============================================================================
// Consistency Tests
// ============================================================================

TEST(HashUtilsTest, HashSizeMatchesEvpMdSize) {
    // Verify our constants match OpenSSL's values
    EXPECT_EQ(get_hash_size(HashAlgorithm::SHA256), 
              static_cast<size_t>(EVP_MD_size(EVP_sha256())));
    EXPECT_EQ(get_hash_size(HashAlgorithm::SHA384), 
              static_cast<size_t>(EVP_MD_size(EVP_sha384())));
    EXPECT_EQ(get_hash_size(HashAlgorithm::SHA512), 
              static_cast<size_t>(EVP_MD_size(EVP_sha512())));
}

TEST(HashUtilsTest, CurveHashMatchesSecurityLevel) {
    // P-256 has 128-bit security, SHA-256 provides 128-bit collision resistance
    EXPECT_EQ(get_hash_size(get_hash_for_curve(ECCurve::P256)), 32u);
    
    // P-384 has 192-bit security, SHA-384 provides 192-bit collision resistance
    EXPECT_EQ(get_hash_size(get_hash_for_curve(ECCurve::P384)), 48u);
    
    // P-521 has 256-bit security, SHA-512 provides 256-bit collision resistance
    EXPECT_EQ(get_hash_size(get_hash_for_curve(ECCurve::P521)), 64u);
}

} // namespace crypto::test
