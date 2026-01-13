// Feature: crypto-security-service
// Property-based tests for RSA encryption engine

#include <gtest/gtest.h>
#include <rapidcheck.h>
#include <rapidcheck/gtest.h>
#include "crypto/engine/rsa_engine.h"

namespace crypto::test {

// Generator for RSA key sizes
rc::Gen<RSAKeySize> genRSAKeySize() {
    return rc::gen::element(RSAKeySize::RSA_2048, RSAKeySize::RSA_3072, RSAKeySize::RSA_4096);
}

// Generator for hash algorithms
rc::Gen<HashAlgorithm> genHashAlgorithm() {
    return rc::gen::element(HashAlgorithm::SHA256, HashAlgorithm::SHA384, HashAlgorithm::SHA512);
}

// Generator for plaintext that fits within RSA OAEP limits
rc::Gen<std::vector<uint8_t>> genRSAPlaintext(size_t max_size) {
    return rc::gen::container<std::vector<uint8_t>>(
        rc::gen::inRange<size_t>(1, max_size),
        rc::gen::arbitrary<uint8_t>()
    );
}

// Generator for arbitrary data (for signatures)
rc::Gen<std::vector<uint8_t>> genData() {
    return rc::gen::withSize([](int size) {
        return rc::gen::container<std::vector<uint8_t>>(
            rc::gen::inRange(1, std::max(2, size * 100)),
            rc::gen::arbitrary<uint8_t>()
        );
    });
}

class RSAPropertiesTest : public ::testing::Test {
protected:
    RSAEngine engine_;
    
    // Cache key pairs for performance (key generation is slow)
    static std::map<RSAKeySize, RSAKeyPair> key_cache_;
    
    RSAKeyPair& getOrCreateKeyPair(RSAKeySize size) {
        auto it = key_cache_.find(size);
        if (it == key_cache_.end()) {
            auto result = engine_.generateKeyPair(size);
            if (!result.has_value()) {
                throw std::runtime_error("Failed to generate key pair");
            }
            auto [inserted_it, _] = key_cache_.emplace(size, std::move(*result));
            return inserted_it->second;
        }
        return it->second;
    }
};

std::map<RSAKeySize, RSAKeyPair> RSAPropertiesTest::key_cache_;

// Property 5: RSA Encryption Round-Trip
// For any valid plaintext within the size limit for the key and any valid RSA key pair
// (2048, 3072, or 4096 bits), encrypting with the public key and decrypting with the
// private key SHALL produce the original plaintext.
// Validates: Requirements 2.1, 2.2, 2.3, 2.7
RC_GTEST_FIXTURE_PROP(RSAPropertiesTest, EncryptionRoundTrip, ()) {
    auto key_size = *genRSAKeySize();
    auto& key_pair = getOrCreateKeyPair(key_size);
    
    // Generate plaintext that fits within OAEP limits
    size_t max_plaintext = key_pair.maxPlaintextSize();
    auto plaintext = *genRSAPlaintext(max_plaintext);
    
    // Encrypt
    auto encrypt_result = engine_.encryptOAEP(plaintext, key_pair);
    RC_ASSERT(encrypt_result.has_value());
    
    // Decrypt
    auto decrypt_result = engine_.decryptOAEP(*encrypt_result, key_pair);
    RC_ASSERT(decrypt_result.has_value());
    
    // Verify round-trip
    RC_ASSERT(*decrypt_result == plaintext);
}

// Property 6: RSA Size Limit Enforcement
// For any RSA key, attempting to encrypt plaintext larger than the maximum allowed
// size for that key SHALL return a size limit error.
// Validates: Requirements 2.5
RC_GTEST_FIXTURE_PROP(RSAPropertiesTest, SizeLimitEnforcement, ()) {
    auto key_size = *genRSAKeySize();
    auto& key_pair = getOrCreateKeyPair(key_size);
    
    // Generate plaintext larger than allowed
    size_t max_plaintext = key_pair.maxPlaintextSize();
    size_t oversized = max_plaintext + *rc::gen::inRange<size_t>(1, 100);
    
    std::vector<uint8_t> plaintext(oversized);
    std::fill(plaintext.begin(), plaintext.end(), 0x42);
    
    // Attempt to encrypt oversized plaintext
    auto result = engine_.encryptOAEP(plaintext, key_pair);
    
    // Must fail with size limit error
    RC_ASSERT(result.is_error());
    RC_ASSERT(result.error_code() == ErrorCode::SIZE_LIMIT_EXCEEDED);
}

// Property 8: Signature Consistency
// For any valid data and any valid signing key pair (RSA-PSS with SHA-256/384/512),
// signing the data with the private key and verifying with the corresponding public
// key SHALL always return valid.
// Validates: Requirements 3.1, 3.2, 3.3, 3.7
RC_GTEST_FIXTURE_PROP(RSAPropertiesTest, SignatureConsistency, ()) {
    auto key_size = *genRSAKeySize();
    auto hash_algo = *genHashAlgorithm();
    auto& key_pair = getOrCreateKeyPair(key_size);
    auto data = *genData();
    
    // Sign
    auto sign_result = engine_.signPSS(data, key_pair, hash_algo);
    RC_ASSERT(sign_result.has_value());
    
    // Verify
    auto verify_result = engine_.verifyPSS(data, *sign_result, key_pair, hash_algo);
    RC_ASSERT(verify_result.has_value());
    RC_ASSERT(*verify_result == true);
}

// Property 9: Invalid Signature Rejection
// For any valid data and signature, verifying the signature against different data
// or a different public key SHALL return invalid (false).
// Validates: Requirements 3.6
RC_GTEST_FIXTURE_PROP(RSAPropertiesTest, InvalidSignatureRejectionDifferentData, ()) {
    auto key_size = *genRSAKeySize();
    auto hash_algo = *genHashAlgorithm();
    auto& key_pair = getOrCreateKeyPair(key_size);
    auto data1 = *genData();
    auto data2 = *genData();
    
    // Ensure data is different
    RC_PRE(data1 != data2);
    
    // Sign data1
    auto sign_result = engine_.signPSS(data1, key_pair, hash_algo);
    RC_ASSERT(sign_result.has_value());
    
    // Verify against data2 (should fail)
    auto verify_result = engine_.verifyPSS(data2, *sign_result, key_pair, hash_algo);
    RC_ASSERT(verify_result.has_value());
    RC_ASSERT(*verify_result == false);
}

RC_GTEST_FIXTURE_PROP(RSAPropertiesTest, InvalidSignatureRejectionDifferentKey, ()) {
    auto hash_algo = *genHashAlgorithm();
    auto data = *genData();
    
    // Use two different key pairs
    auto& key_pair1 = getOrCreateKeyPair(RSAKeySize::RSA_2048);
    
    // Generate a fresh key pair for key_pair2
    auto key_pair2_result = engine_.generateKeyPair(RSAKeySize::RSA_2048);
    RC_ASSERT(key_pair2_result.has_value());
    
    // Sign with key_pair1
    auto sign_result = engine_.signPSS(data, key_pair1, hash_algo);
    RC_ASSERT(sign_result.has_value());
    
    // Verify with key_pair2 (should fail)
    auto verify_result = engine_.verifyPSS(data, *sign_result, *key_pair2_result, hash_algo);
    RC_ASSERT(verify_result.has_value());
    RC_ASSERT(*verify_result == false);
}

RC_GTEST_FIXTURE_PROP(RSAPropertiesTest, InvalidSignatureRejectionTamperedSignature, ()) {
    auto key_size = *genRSAKeySize();
    auto hash_algo = *genHashAlgorithm();
    auto& key_pair = getOrCreateKeyPair(key_size);
    auto data = *genData();
    
    // Sign
    auto sign_result = engine_.signPSS(data, key_pair, hash_algo);
    RC_ASSERT(sign_result.has_value());
    
    // Tamper with signature
    auto tampered_sig = *sign_result;
    size_t tamper_pos = *rc::gen::inRange<size_t>(0, tampered_sig.size());
    tampered_sig[tamper_pos] ^= 0xFF;
    
    // Verify tampered signature (should fail)
    auto verify_result = engine_.verifyPSS(data, tampered_sig, key_pair, hash_algo);
    RC_ASSERT(verify_result.has_value());
    RC_ASSERT(*verify_result == false);
}

// Unit tests for edge cases
TEST_F(RSAPropertiesTest, KeyGeneration2048) {
    auto result = engine_.generateKeyPair(RSAKeySize::RSA_2048);
    ASSERT_TRUE(result.has_value());
    EXPECT_EQ(result->keySize(), 2048);
}

TEST_F(RSAPropertiesTest, KeyGeneration4096) {
    auto result = engine_.generateKeyPair(RSAKeySize::RSA_4096);
    ASSERT_TRUE(result.has_value());
    EXPECT_EQ(result->keySize(), 4096);
}

TEST_F(RSAPropertiesTest, KeyExportImportRoundTrip) {
    auto key_result = engine_.generateKeyPair(RSAKeySize::RSA_2048);
    ASSERT_TRUE(key_result.has_value());
    
    // Export public key
    auto pub_der = key_result->exportPublicKeyDER();
    ASSERT_TRUE(pub_der.has_value());
    
    // Import public key
    auto imported = RSAKeyPair::importPublicKeyDER(*pub_der);
    ASSERT_TRUE(imported.has_value());
    EXPECT_EQ(imported->keySize(), 2048);
}

TEST_F(RSAPropertiesTest, EmptyPlaintextEncryption) {
    auto key_result = engine_.generateKeyPair(RSAKeySize::RSA_2048);
    ASSERT_TRUE(key_result.has_value());
    
    std::vector<uint8_t> empty_plaintext;
    
    // Empty plaintext should work (OAEP can encrypt empty data)
    auto encrypt_result = engine_.encryptOAEP(empty_plaintext, *key_result);
    ASSERT_TRUE(encrypt_result.has_value());
    
    auto decrypt_result = engine_.decryptOAEP(*encrypt_result, *key_result);
    ASSERT_TRUE(decrypt_result.has_value());
    EXPECT_EQ(*decrypt_result, empty_plaintext);
}

} // namespace crypto::test
