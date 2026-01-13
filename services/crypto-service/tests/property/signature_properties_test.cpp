// Feature: crypto-security-service
// Property-based tests for signature engines (RSA-PSS and ECDSA)

#include <gtest/gtest.h>
#include <rapidcheck.h>
#include <rapidcheck/gtest.h>
#include "crypto/engine/rsa_engine.h"
#include "crypto/engine/ecdsa_engine.h"

namespace crypto::test {

// Generator for EC curves
rc::Gen<ECCurve> genECCurve() {
    return rc::gen::element(ECCurve::P256, ECCurve::P384, ECCurve::P521);
}

// Generator for arbitrary data
rc::Gen<std::vector<uint8_t>> genSignatureData() {
    return rc::gen::withSize([](int size) {
        return rc::gen::container<std::vector<uint8_t>>(
            rc::gen::inRange(1, std::max(2, size * 100)),
            rc::gen::arbitrary<uint8_t>()
        );
    });
}

class ECDSAPropertiesTest : public ::testing::Test {
protected:
    ECDSAEngine engine_;
    
    // Cache key pairs for performance
    static std::map<ECCurve, ECKeyPair> key_cache_;
    
    ECKeyPair& getOrCreateKeyPair(ECCurve curve) {
        auto it = key_cache_.find(curve);
        if (it == key_cache_.end()) {
            auto result = engine_.generateKeyPair(curve);
            if (!result.has_value()) {
                throw std::runtime_error("Failed to generate EC key pair");
            }
            auto [inserted_it, _] = key_cache_.emplace(curve, std::move(*result));
            return inserted_it->second;
        }
        return it->second;
    }
};

std::map<ECCurve, ECKeyPair> ECDSAPropertiesTest::key_cache_;

// Property 8: Signature Consistency (ECDSA)
// For any valid data and any valid signing key pair (ECDSA with P-256/384/521),
// signing the data with the private key and verifying with the corresponding
// public key SHALL always return valid.
// Validates: Requirements 3.1, 3.2, 3.4, 3.7
RC_GTEST_FIXTURE_PROP(ECDSAPropertiesTest, SignatureConsistency, ()) {
    auto curve = *genECCurve();
    auto& key_pair = getOrCreateKeyPair(curve);
    auto data = *genSignatureData();
    
    // Sign
    auto sign_result = engine_.sign(data, key_pair);
    RC_ASSERT(sign_result.has_value());
    
    // Verify
    auto verify_result = engine_.verify(data, *sign_result, key_pair);
    RC_ASSERT(verify_result.has_value());
    RC_ASSERT(*verify_result == true);
}

// Property 9: Invalid Signature Rejection (ECDSA)
// For any valid data and signature, verifying the signature against different
// data or a different public key SHALL return invalid (false).
// Validates: Requirements 3.6
RC_GTEST_FIXTURE_PROP(ECDSAPropertiesTest, InvalidSignatureRejectionDifferentData, ()) {
    auto curve = *genECCurve();
    auto& key_pair = getOrCreateKeyPair(curve);
    auto data1 = *genSignatureData();
    auto data2 = *genSignatureData();
    
    // Ensure data is different
    RC_PRE(data1 != data2);
    
    // Sign data1
    auto sign_result = engine_.sign(data1, key_pair);
    RC_ASSERT(sign_result.has_value());
    
    // Verify against data2 (should fail)
    auto verify_result = engine_.verify(data2, *sign_result, key_pair);
    RC_ASSERT(verify_result.has_value());
    RC_ASSERT(*verify_result == false);
}

RC_GTEST_FIXTURE_PROP(ECDSAPropertiesTest, InvalidSignatureRejectionDifferentKey, ()) {
    auto curve = *genECCurve();
    auto data = *genSignatureData();
    
    // Use two different key pairs
    auto& key_pair1 = getOrCreateKeyPair(curve);
    
    // Generate a fresh key pair
    auto key_pair2_result = engine_.generateKeyPair(curve);
    RC_ASSERT(key_pair2_result.has_value());
    
    // Sign with key_pair1
    auto sign_result = engine_.sign(data, key_pair1);
    RC_ASSERT(sign_result.has_value());
    
    // Verify with key_pair2 (should fail)
    auto verify_result = engine_.verify(data, *sign_result, *key_pair2_result);
    RC_ASSERT(verify_result.has_value());
    RC_ASSERT(*verify_result == false);
}

RC_GTEST_FIXTURE_PROP(ECDSAPropertiesTest, InvalidSignatureRejectionTamperedSignature, ()) {
    auto curve = *genECCurve();
    auto& key_pair = getOrCreateKeyPair(curve);
    auto data = *genSignatureData();
    
    // Sign
    auto sign_result = engine_.sign(data, key_pair);
    RC_ASSERT(sign_result.has_value());
    
    // Tamper with signature
    auto tampered_sig = *sign_result;
    size_t tamper_pos = *rc::gen::inRange<size_t>(0, tampered_sig.size());
    tampered_sig[tamper_pos] ^= 0xFF;
    
    // Verify tampered signature (should fail)
    auto verify_result = engine_.verify(data, tampered_sig, key_pair);
    RC_ASSERT(verify_result.has_value());
    RC_ASSERT(*verify_result == false);
}

// Unit tests for edge cases
TEST_F(ECDSAPropertiesTest, KeyGenerationP256) {
    auto result = engine_.generateKeyPair(ECCurve::P256);
    ASSERT_TRUE(result.has_value());
    EXPECT_EQ(result->curve(), ECCurve::P256);
}

TEST_F(ECDSAPropertiesTest, KeyGenerationP384) {
    auto result = engine_.generateKeyPair(ECCurve::P384);
    ASSERT_TRUE(result.has_value());
    EXPECT_EQ(result->curve(), ECCurve::P384);
}

TEST_F(ECDSAPropertiesTest, KeyGenerationP521) {
    auto result = engine_.generateKeyPair(ECCurve::P521);
    ASSERT_TRUE(result.has_value());
    EXPECT_EQ(result->curve(), ECCurve::P521);
}

TEST_F(ECDSAPropertiesTest, KeyExportImportRoundTrip) {
    auto key_result = engine_.generateKeyPair(ECCurve::P256);
    ASSERT_TRUE(key_result.has_value());
    
    // Export public key
    auto pub_der = key_result->exportPublicKeyDER();
    ASSERT_TRUE(pub_der.has_value());
    
    // Import public key
    auto imported = ECKeyPair::importPublicKeyDER(*pub_der, ECCurve::P256);
    ASSERT_TRUE(imported.has_value());
}

TEST_F(ECDSAPropertiesTest, EmptyDataSignature) {
    auto key_result = engine_.generateKeyPair(ECCurve::P256);
    ASSERT_TRUE(key_result.has_value());
    
    std::vector<uint8_t> empty_data;
    
    // Empty data should work
    auto sign_result = engine_.sign(empty_data, *key_result);
    ASSERT_TRUE(sign_result.has_value());
    
    auto verify_result = engine_.verify(empty_data, *sign_result, *key_result);
    ASSERT_TRUE(verify_result.has_value());
    EXPECT_TRUE(*verify_result);
}

TEST_F(ECDSAPropertiesTest, LargeDataSignature) {
    auto key_result = engine_.generateKeyPair(ECCurve::P256);
    ASSERT_TRUE(key_result.has_value());
    
    // 1MB of data
    std::vector<uint8_t> large_data(1024 * 1024, 0x42);
    
    auto sign_result = engine_.sign(large_data, *key_result);
    ASSERT_TRUE(sign_result.has_value());
    
    auto verify_result = engine_.verify(large_data, *sign_result, *key_result);
    ASSERT_TRUE(verify_result.has_value());
    EXPECT_TRUE(*verify_result);
}

} // namespace crypto::test
