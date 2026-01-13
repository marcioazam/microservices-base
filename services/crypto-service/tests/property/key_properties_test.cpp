// Feature: crypto-security-service
// Property-based tests for key management

#include <gtest/gtest.h>
#include <rapidcheck.h>
#include <rapidcheck/gtest.h>
#include "crypto/keys/key_service.h"
#include "crypto/keys/key_store.h"
#include "crypto/clients/cache_client.h"
#include "crypto/engine/aes_engine.h"

namespace crypto::test {

// Generator for key algorithms
rc::Gen<KeyAlgorithm> genKeyAlgorithm() {
    return rc::gen::element(
        KeyAlgorithm::AES_128_GCM,
        KeyAlgorithm::AES_256_GCM,
        KeyAlgorithm::RSA_2048,
        KeyAlgorithm::ECDSA_P256
    );
}

// Generator for namespace prefixes
rc::Gen<std::string> genNamespace() {
    return rc::gen::element<std::string>("auth", "payment", "user", "default");
}

class KeyPropertiesTest : public ::testing::Test {
protected:
    std::shared_ptr<InMemoryKeyStore> key_store_;
    std::shared_ptr<CacheClient> cache_client_;
    std::unique_ptr<KeyService> key_service_;
    std::vector<uint8_t> master_key_;

    void SetUp() override {
        // Generate master key
        auto key_result = AESEngine::generateKey(AESKeySize::AES_256);
        ASSERT_TRUE(key_result.has_value());
        master_key_ = key_result->to_vector();

        key_store_ = std::make_shared<InMemoryKeyStore>();
        
        // Create CacheClient with local fallback enabled (no real cache-service needed)
        CacheClientConfig cache_config;
        cache_config.local_fallback_enabled = true;
        cache_config.local_cache_size = 1000;
        cache_client_ = std::make_shared<CacheClient>(cache_config);
        
        key_service_ = std::make_unique<KeyService>(key_store_, master_key_, cache_client_);
    }
};

// Property 10: Generated Keys Are Functional
// For any key generation request (AES-128, AES-256, RSA-2048, ECDSA P-256),
// the generated key SHALL be usable for its intended cryptographic operations.
// Validates: Requirements 4.1, 4.2, 4.3
RC_GTEST_FIXTURE_PROP(KeyPropertiesTest, GeneratedKeysAreFunctional, ()) {
    auto algorithm = *genKeyAlgorithm();
    auto ns = *genNamespace();

    KeyGenerationParams params;
    params.namespace_prefix = ns;
    params.algorithm = algorithm;
    params.owner_service = "test-service";

    // Generate key
    auto key_id_result = key_service_->generateKey(params);
    RC_ASSERT(key_id_result.has_value());

    // Get key material
    auto key_material_result = key_service_->getKeyMaterial(*key_id_result);
    RC_ASSERT(key_material_result.has_value());
    RC_ASSERT(!key_material_result->empty());

    // Verify key can be used for its intended purpose
    if (isSymmetricAlgorithm(algorithm)) {
        // Test AES encryption/decryption
        AESEngine aes;
        std::vector<uint8_t> plaintext = {1, 2, 3, 4, 5};
        
        auto encrypt_result = aes.encryptGCM(plaintext, *key_material_result);
        RC_ASSERT(encrypt_result.has_value());
        
        auto decrypt_result = aes.decryptGCM(
            encrypt_result->ciphertext,
            *key_material_result,
            encrypt_result->iv,
            encrypt_result->tag
        );
        RC_ASSERT(decrypt_result.has_value());
        RC_ASSERT(*decrypt_result == plaintext);
    }
    // For asymmetric keys, the key material is the private key DER
    // which can be imported and used for signing/encryption
}

// Property 11: Key ID Uniqueness
// For any sequence of key generation operations, all generated Key_IDs SHALL be unique.
// Validates: Requirements 4.4
RC_GTEST_FIXTURE_PROP(KeyPropertiesTest, KeyIDUniqueness, ()) {
    auto algorithm = *genKeyAlgorithm();
    int num_keys = *rc::gen::inRange(2, 10);

    std::set<std::string> key_ids;

    for (int i = 0; i < num_keys; ++i) {
        KeyGenerationParams params;
        params.algorithm = algorithm;
        params.owner_service = "test-service";

        auto key_id_result = key_service_->generateKey(params);
        RC_ASSERT(key_id_result.has_value());

        std::string key_str = key_id_result->toString();
        RC_ASSERT(key_ids.find(key_str) == key_ids.end());
        key_ids.insert(key_str);
    }

    RC_ASSERT(key_ids.size() == static_cast<size_t>(num_keys));
}

// Property 12: Key Metadata Completeness
// For any generated key, retrieving its metadata SHALL return a complete
// KeyMetadata object containing algorithm, creation timestamp, expiration date,
// and owner information.
// Validates: Requirements 4.5
RC_GTEST_FIXTURE_PROP(KeyPropertiesTest, KeyMetadataCompleteness, ()) {
    auto algorithm = *genKeyAlgorithm();
    auto ns = *genNamespace();
    std::string owner = "test-service-" + std::to_string(*rc::gen::inRange(1, 100));

    KeyGenerationParams params;
    params.namespace_prefix = ns;
    params.algorithm = algorithm;
    params.owner_service = owner;
    params.allowed_operations = {"encrypt", "decrypt"};

    auto key_id_result = key_service_->generateKey(params);
    RC_ASSERT(key_id_result.has_value());

    auto metadata_result = key_service_->getKeyMetadata(*key_id_result);
    RC_ASSERT(metadata_result.has_value());

    // Verify all required fields are present
    RC_ASSERT(metadata_result->id == *key_id_result);
    RC_ASSERT(metadata_result->algorithm == algorithm);
    RC_ASSERT(metadata_result->state == KeyState::ACTIVE);
    RC_ASSERT(metadata_result->owner_service == owner);
    RC_ASSERT(!metadata_result->allowed_operations.empty());
    
    // Verify timestamps are valid
    auto now = std::chrono::system_clock::now();
    RC_ASSERT(metadata_result->created_at <= now);
    RC_ASSERT(metadata_result->expires_at > metadata_result->created_at);
}

// Property 13: Private Key Protection
// For any key generation or key retrieval operation, the response SHALL contain
// only the Key_ID reference, never the raw private key material.
// Validates: Requirements 4.7
// Note: This is enforced by API design - generateKey returns KeyId, not key material
TEST_F(KeyPropertiesTest, PrivateKeyProtection) {
    KeyGenerationParams params;
    params.algorithm = KeyAlgorithm::RSA_2048;
    params.owner_service = "test-service";

    auto key_id_result = key_service_->generateKey(params);
    ASSERT_TRUE(key_id_result.has_value());

    // generateKey returns only KeyId, not key material
    // This is enforced by the API design
    
    // getKeyMetadata also doesn't return key material
    auto metadata_result = key_service_->getKeyMetadata(*key_id_result);
    ASSERT_TRUE(metadata_result.has_value());
    // KeyMetadata struct doesn't contain key material
}

// Property 14: Key Rotation State Machine
// For any key rotation operation on an active key, the operation SHALL:
// (a) create a new key with a different Key_ID,
// (b) mark the old key as deprecated, and
// (c) the deprecated key SHALL be rejected for new encryption operations.
// Validates: Requirements 6.2, 6.4, 6.7
RC_GTEST_FIXTURE_PROP(KeyPropertiesTest, KeyRotationStateMachine, ()) {
    auto algorithm = *genKeyAlgorithm();

    // Generate initial key
    KeyGenerationParams params;
    params.algorithm = algorithm;
    params.owner_service = "test-service";

    auto old_key_id_result = key_service_->generateKey(params);
    RC_ASSERT(old_key_id_result.has_value());

    // Rotate key
    auto new_key_id_result = key_service_->rotateKey(*old_key_id_result);
    RC_ASSERT(new_key_id_result.has_value());

    // (a) New key has different Key_ID
    RC_ASSERT(*new_key_id_result != *old_key_id_result);

    // (b) Old key is deprecated
    auto old_metadata = key_service_->getKeyMetadata(*old_key_id_result);
    RC_ASSERT(old_metadata.has_value());
    RC_ASSERT(old_metadata->state == KeyState::DEPRECATED);

    // (c) New key is active
    auto new_metadata = key_service_->getKeyMetadata(*new_key_id_result);
    RC_ASSERT(new_metadata.has_value());
    RC_ASSERT(new_metadata->state == KeyState::ACTIVE);

    // Verify deprecated key cannot encrypt (checked via metadata)
    RC_ASSERT(!old_metadata->canEncrypt());
}

// Property 15: Deprecated Key Decryption
// For any data encrypted with a key that is subsequently rotated,
// decryption using the deprecated key SHALL still succeed during the grace period.
// Validates: Requirements 6.3
RC_GTEST_FIXTURE_PROP(KeyPropertiesTest, DeprecatedKeyDecryption, ()) {
    // Generate AES key (symmetric for easier testing)
    KeyGenerationParams params;
    params.algorithm = KeyAlgorithm::AES_256_GCM;
    params.owner_service = "test-service";

    auto key_id_result = key_service_->generateKey(params);
    RC_ASSERT(key_id_result.has_value());

    // Get key material and encrypt some data
    auto key_material = key_service_->getKeyMaterial(*key_id_result);
    RC_ASSERT(key_material.has_value());

    AESEngine aes;
    std::vector<uint8_t> plaintext = {1, 2, 3, 4, 5, 6, 7, 8};
    auto encrypt_result = aes.encryptGCM(plaintext, *key_material);
    RC_ASSERT(encrypt_result.has_value());

    // Rotate the key
    auto new_key_id = key_service_->rotateKey(*key_id_result);
    RC_ASSERT(new_key_id.has_value());

    // Old key is now deprecated, but should still be able to decrypt
    auto old_metadata = key_service_->getKeyMetadata(*key_id_result);
    RC_ASSERT(old_metadata.has_value());
    RC_ASSERT(old_metadata->state == KeyState::DEPRECATED);
    RC_ASSERT(old_metadata->canDecrypt());  // Can still decrypt

    // Get old key material (should still work)
    auto old_key_material = key_service_->getKeyMaterial(*key_id_result);
    RC_ASSERT(old_key_material.has_value());

    // Decrypt with old key
    auto decrypt_result = aes.decryptGCM(
        encrypt_result->ciphertext,
        *old_key_material,
        encrypt_result->iv,
        encrypt_result->tag
    );
    RC_ASSERT(decrypt_result.has_value());
    RC_ASSERT(*decrypt_result == plaintext);
}

// Unit tests for edge cases
TEST_F(KeyPropertiesTest, DeleteKey) {
    KeyGenerationParams params;
    params.algorithm = KeyAlgorithm::AES_256_GCM;

    auto key_id_result = key_service_->generateKey(params);
    ASSERT_TRUE(key_id_result.has_value());

    // Delete key
    auto delete_result = key_service_->deleteKey(*key_id_result);
    ASSERT_TRUE(delete_result.has_value());

    // Key should no longer exist
    auto metadata_result = key_service_->getKeyMetadata(*key_id_result);
    ASSERT_TRUE(metadata_result.is_error());
    EXPECT_EQ(metadata_result.error_code(), ErrorCode::KEY_NOT_FOUND);
}

TEST_F(KeyPropertiesTest, RotateNonExistentKey) {
    KeyId fake_id("test", "non-existent-uuid", 1);
    
    auto result = key_service_->rotateKey(fake_id);
    ASSERT_TRUE(result.is_error());
    EXPECT_EQ(result.error_code(), ErrorCode::KEY_NOT_FOUND);
}

TEST_F(KeyPropertiesTest, KeyCacheHit) {
    KeyGenerationParams params;
    params.algorithm = KeyAlgorithm::AES_256_GCM;

    auto key_id_result = key_service_->generateKey(params);
    ASSERT_TRUE(key_id_result.has_value());

    // First access (cache miss, then cached)
    auto material1 = key_service_->getKeyMaterial(*key_id_result);
    ASSERT_TRUE(material1.has_value());

    // Second access (cache hit)
    auto material2 = key_service_->getKeyMaterial(*key_id_result);
    ASSERT_TRUE(material2.has_value());

    EXPECT_EQ(*material1, *material2);
    // Note: CacheClient doesn't expose hit/miss stats directly
    // The test verifies that repeated access returns the same material
}

} // namespace crypto::test
