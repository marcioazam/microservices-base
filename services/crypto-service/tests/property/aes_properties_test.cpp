// Feature: crypto-security-service
// Property-based tests for AES encryption engine

#include <gtest/gtest.h>
#include <rapidcheck.h>
#include <rapidcheck/gtest.h>
#include "crypto/engine/aes_engine.h"
#include "crypto/common/secure_memory.h"

namespace crypto::test {

// Generator for valid AES key sizes
rc::Gen<size_t> genAESKeySize() {
    return rc::gen::element(16, 32);
}

// Generator for AES keys
rc::Gen<std::vector<uint8_t>> genAESKey(size_t size) {
    return rc::gen::container<std::vector<uint8_t>>(size, rc::gen::arbitrary<uint8_t>());
}

// Generator for arbitrary binary data (plaintext)
rc::Gen<std::vector<uint8_t>> genPlaintext() {
    return rc::gen::withSize([](int size) {
        return rc::gen::container<std::vector<uint8_t>>(
            rc::gen::inRange(0, std::max(1, size * 100)),
            rc::gen::arbitrary<uint8_t>()
        );
    });
}

// Generator for AAD (Additional Authenticated Data)
rc::Gen<std::vector<uint8_t>> genAAD() {
    return rc::gen::withSize([](int size) {
        return rc::gen::container<std::vector<uint8_t>>(
            rc::gen::inRange(0, std::max(1, size * 50)),
            rc::gen::arbitrary<uint8_t>()
        );
    });
}

class AESPropertiesTest : public ::testing::Test {
protected:
    AESEngine engine_;
};

// Property 1: AES Encryption Round-Trip
// For any valid plaintext data and any valid AES key (128-bit or 256-bit),
// encrypting the plaintext using AES-GCM and then decrypting the result
// with the same key SHALL produce the original plaintext byte-for-byte.
// Validates: Requirements 1.1, 1.2, 1.3, 1.4, 1.8
RC_GTEST_FIXTURE_PROP(AESPropertiesTest, GCMRoundTrip, ()) {
    auto key_size = *genAESKeySize();
    auto key = *genAESKey(key_size);
    auto plaintext = *genPlaintext();
    
    // Encrypt
    auto encrypt_result = engine_.encryptGCM(plaintext, key);
    RC_ASSERT(encrypt_result.has_value());
    
    // Decrypt
    auto decrypt_result = engine_.decryptGCM(
        encrypt_result->ciphertext,
        key,
        encrypt_result->iv,
        encrypt_result->tag
    );
    RC_ASSERT(decrypt_result.has_value());
    
    // Verify round-trip produces original plaintext
    RC_ASSERT(*decrypt_result == plaintext);
}

// Property 1 (continued): AES-GCM round-trip with AAD
// Validates: Requirements 1.1, 1.2, 1.6, 1.8
RC_GTEST_FIXTURE_PROP(AESPropertiesTest, GCMRoundTripWithAAD, ()) {
    auto key_size = *genAESKeySize();
    auto key = *genAESKey(key_size);
    auto plaintext = *genPlaintext();
    auto aad = *genAAD();
    
    // Encrypt with AAD
    auto encrypt_result = engine_.encryptGCM(plaintext, key, aad);
    RC_ASSERT(encrypt_result.has_value());
    
    // Decrypt with same AAD
    auto decrypt_result = engine_.decryptGCM(
        encrypt_result->ciphertext,
        key,
        encrypt_result->iv,
        encrypt_result->tag,
        aad
    );
    RC_ASSERT(decrypt_result.has_value());
    
    // Verify round-trip produces original plaintext
    RC_ASSERT(*decrypt_result == plaintext);
}

// Property 1 (continued): AES-CBC round-trip
// Validates: Requirements 1.4, 1.8
RC_GTEST_FIXTURE_PROP(AESPropertiesTest, CBCRoundTrip, ()) {
    auto key_size = *genAESKeySize();
    auto key = *genAESKey(key_size);
    auto plaintext = *genPlaintext();
    
    // Encrypt
    auto encrypt_result = engine_.encryptCBC(plaintext, key);
    RC_ASSERT(encrypt_result.has_value());
    
    // Decrypt
    auto decrypt_result = engine_.decryptCBC(
        encrypt_result->ciphertext,
        key,
        encrypt_result->iv
    );
    RC_ASSERT(decrypt_result.has_value());
    
    // Verify round-trip produces original plaintext
    RC_ASSERT(*decrypt_result == plaintext);
}

// Property 2: AES IV Uniqueness
// For any two encryption operations using AES-GCM mode, even with identical
// plaintext and key, the generated IVs SHALL be different.
// Validates: Requirements 1.5
RC_GTEST_FIXTURE_PROP(AESPropertiesTest, IVUniqueness, ()) {
    auto key_size = *genAESKeySize();
    auto key = *genAESKey(key_size);
    auto plaintext = *genPlaintext();
    
    // Encrypt twice with same plaintext and key
    auto result1 = engine_.encryptGCM(plaintext, key);
    auto result2 = engine_.encryptGCM(plaintext, key);
    
    RC_ASSERT(result1.has_value());
    RC_ASSERT(result2.has_value());
    
    // IVs must be different
    RC_ASSERT(result1->iv != result2->iv);
}

// Property 3: AES AAD Binding
// For any AES-GCM encryption with Additional Authenticated Data (AAD),
// attempting to decrypt with different AAD SHALL fail with an integrity error.
// Validates: Requirements 1.6
RC_GTEST_FIXTURE_PROP(AESPropertiesTest, AADBinding, ()) {
    auto key_size = *genAESKeySize();
    auto key = *genAESKey(key_size);
    auto plaintext = *genPlaintext();
    auto aad1 = *genAAD();
    auto aad2 = *genAAD();
    
    // Ensure AADs are different
    RC_PRE(aad1 != aad2);
    
    // Encrypt with aad1
    auto encrypt_result = engine_.encryptGCM(plaintext, key, aad1);
    RC_ASSERT(encrypt_result.has_value());
    
    // Attempt to decrypt with different AAD (aad2)
    auto decrypt_result = engine_.decryptGCM(
        encrypt_result->ciphertext,
        key,
        encrypt_result->iv,
        encrypt_result->tag,
        aad2
    );
    
    // Decryption must fail with integrity error
    RC_ASSERT(decrypt_result.is_error());
    RC_ASSERT(decrypt_result.error_code() == ErrorCode::INTEGRITY_ERROR);
}

// Property 4: AES Tamper Detection
// For any AES-GCM encrypted ciphertext, modifying any byte of the ciphertext,
// IV, or authentication tag SHALL cause decryption to fail with an integrity error.
// Validates: Requirements 1.7
RC_GTEST_FIXTURE_PROP(AESPropertiesTest, TamperDetectionCiphertext, ()) {
    auto key_size = *genAESKeySize();
    auto key = *genAESKey(key_size);
    auto plaintext = *genPlaintext();
    
    // Need non-empty plaintext to have ciphertext to tamper with
    RC_PRE(!plaintext.empty());
    
    // Encrypt
    auto encrypt_result = engine_.encryptGCM(plaintext, key);
    RC_ASSERT(encrypt_result.has_value());
    
    // Tamper with ciphertext
    auto tampered_ciphertext = encrypt_result->ciphertext;
    size_t tamper_pos = *rc::gen::inRange<size_t>(0, tampered_ciphertext.size());
    tampered_ciphertext[tamper_pos] ^= 0xFF;  // Flip all bits
    
    // Attempt to decrypt tampered ciphertext
    auto decrypt_result = engine_.decryptGCM(
        tampered_ciphertext,
        key,
        encrypt_result->iv,
        encrypt_result->tag
    );
    
    // Decryption must fail with integrity error
    RC_ASSERT(decrypt_result.is_error());
    RC_ASSERT(decrypt_result.error_code() == ErrorCode::INTEGRITY_ERROR);
}

RC_GTEST_FIXTURE_PROP(AESPropertiesTest, TamperDetectionTag, ()) {
    auto key_size = *genAESKeySize();
    auto key = *genAESKey(key_size);
    auto plaintext = *genPlaintext();
    
    // Encrypt
    auto encrypt_result = engine_.encryptGCM(plaintext, key);
    RC_ASSERT(encrypt_result.has_value());
    
    // Tamper with tag
    auto tampered_tag = encrypt_result->tag;
    size_t tamper_pos = *rc::gen::inRange<size_t>(0, tampered_tag.size());
    tampered_tag[tamper_pos] ^= 0xFF;  // Flip all bits
    
    // Attempt to decrypt with tampered tag
    auto decrypt_result = engine_.decryptGCM(
        encrypt_result->ciphertext,
        key,
        encrypt_result->iv,
        tampered_tag
    );
    
    // Decryption must fail with integrity error
    RC_ASSERT(decrypt_result.is_error());
    RC_ASSERT(decrypt_result.error_code() == ErrorCode::INTEGRITY_ERROR);
}

RC_GTEST_FIXTURE_PROP(AESPropertiesTest, TamperDetectionIV, ()) {
    auto key_size = *genAESKeySize();
    auto key = *genAESKey(key_size);
    auto plaintext = *genPlaintext();
    
    // Encrypt
    auto encrypt_result = engine_.encryptGCM(plaintext, key);
    RC_ASSERT(encrypt_result.has_value());
    
    // Tamper with IV
    auto tampered_iv = encrypt_result->iv;
    size_t tamper_pos = *rc::gen::inRange<size_t>(0, tampered_iv.size());
    tampered_iv[tamper_pos] ^= 0xFF;  // Flip all bits
    
    // Attempt to decrypt with tampered IV
    auto decrypt_result = engine_.decryptGCM(
        encrypt_result->ciphertext,
        key,
        tampered_iv,
        encrypt_result->tag
    );
    
    // Decryption must fail with integrity error
    RC_ASSERT(decrypt_result.is_error());
    RC_ASSERT(decrypt_result.error_code() == ErrorCode::INTEGRITY_ERROR);
}

// Additional unit tests for edge cases
TEST_F(AESPropertiesTest, EmptyPlaintextGCM) {
    auto key_result = AESEngine::generateKey(AESKeySize::AES_256);
    ASSERT_TRUE(key_result.has_value());
    
    std::vector<uint8_t> empty_plaintext;
    
    auto encrypt_result = engine_.encryptGCM(empty_plaintext, key_result->span());
    ASSERT_TRUE(encrypt_result.has_value());
    
    auto decrypt_result = engine_.decryptGCM(
        encrypt_result->ciphertext,
        key_result->span(),
        encrypt_result->iv,
        encrypt_result->tag
    );
    ASSERT_TRUE(decrypt_result.has_value());
    EXPECT_EQ(*decrypt_result, empty_plaintext);
}

TEST_F(AESPropertiesTest, InvalidKeySize) {
    std::vector<uint8_t> invalid_key(15);  // Invalid size
    std::vector<uint8_t> plaintext = {1, 2, 3, 4};
    
    auto result = engine_.encryptGCM(plaintext, invalid_key);
    ASSERT_TRUE(result.is_error());
    EXPECT_EQ(result.error_code(), ErrorCode::INVALID_KEY_SIZE);
}

TEST_F(AESPropertiesTest, InvalidIVSize) {
    auto key_result = AESEngine::generateKey(AESKeySize::AES_256);
    ASSERT_TRUE(key_result.has_value());
    
    std::vector<uint8_t> invalid_iv(8);  // Invalid size (should be 12)
    std::vector<uint8_t> plaintext = {1, 2, 3, 4};
    
    auto result = engine_.encryptGCMWithIV(plaintext, key_result->span(), invalid_iv);
    ASSERT_TRUE(result.is_error());
    EXPECT_EQ(result.error_code(), ErrorCode::INVALID_IV_SIZE);
}

} // namespace crypto::test
