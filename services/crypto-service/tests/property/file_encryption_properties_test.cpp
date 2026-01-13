/**
 * Property-based tests for FileEncryptionService
 * 
 * Feature: crypto-security-service
 * Tests Properties 17, 18, 19 from design document
 */

#include <gtest/gtest.h>
#include <rapidcheck.h>
#include <rapidcheck/gtest.h>
#include "crypto/services/file_encryption_service.h"
#include "crypto/keys/key_service.h"
#include "crypto/audit/audit_logger.h"
#include <sstream>
#include <set>

namespace crypto::test {

class FileEncryptionPropertyTest : public ::testing::Test {
protected:
    void SetUp() override {
        auto key_store = std::make_shared<LocalKeyStore>("/tmp/test_keys");
        key_service_ = std::make_shared<KeyService>(key_store);
        audit_logger_ = std::make_shared<InMemoryAuditLogger>();
        file_service_ = std::make_unique<FileEncryptionService>(key_service_, audit_logger_);
        
        // Generate a KEK for testing
        auto kek_result = key_service_->generateKey(KeyType::AES, 256, "test");
        ASSERT_TRUE(kek_result.has_value());
        test_kek_id_ = *kek_result;
    }
    
    std::shared_ptr<KeyService> key_service_;
    std::shared_ptr<IAuditLogger> audit_logger_;
    std::unique_ptr<FileEncryptionService> file_service_;
    KeyId test_kek_id_;
    
    FileEncryptionContext createContext() {
        return FileEncryptionContext{
            .correlation_id = "test-" + std::to_string(rand()),
            .caller_identity = "test-user",
            .caller_service = "test-service",
            .source_ip = "127.0.0.1"
        };
    }
};

/**
 * Property 17: File DEK Uniqueness
 * 
 * For any two file encryption operations, even for identical files,
 * the generated Data Encryption Keys (DEKs) SHALL be different.
 * 
 * Validates: Requirements 7.4
 */
RC_GTEST_FIXTURE_PROP(FileEncryptionPropertyTest, DEKUniqueness,
                      (std::vector<uint8_t> file_data)) {
    RC_PRE(file_data.size() <= 1024 * 1024);  // Limit to 1MB for testing
    
    std::set<std::vector<uint8_t>> wrapped_deks;
    constexpr int NUM_ENCRYPTIONS = 10;
    
    for (int i = 0; i < NUM_ENCRYPTIONS; ++i) {
        std::istringstream input(std::string(file_data.begin(), file_data.end()));
        std::ostringstream output;
        
        auto result = file_service_->encryptStream(
            input, output, test_kek_id_, createContext(), file_data.size());
        RC_ASSERT(result.has_value());
        
        // Extract wrapped DEK from output
        std::string encrypted = output.str();
        std::istringstream encrypted_stream(encrypted);
        
        auto header_result = file_service_->readHeader("/dev/stdin");
        // For this test, we parse the header from the output
        uint32_t header_size;
        encrypted_stream.read(reinterpret_cast<char*>(&header_size), sizeof(header_size));
        
        std::vector<uint8_t> header_data(header_size);
        encrypted_stream.read(reinterpret_cast<char*>(header_data.data()), header_size);
        
        auto header = FileEncryptionHeader::deserialize(header_data);
        RC_ASSERT(header.has_value());
        
        // Each wrapped DEK should be unique
        auto [it, inserted] = wrapped_deks.insert(header->wrapped_dek);
        RC_ASSERT(inserted);  // Should always be a new unique DEK
    }
}

/**
 * Property 18: File Header Completeness
 * 
 * For any encrypted file, the file header SHALL contain the wrapped DEK,
 * IV, and authentication tag.
 * 
 * Validates: Requirements 7.6
 */
RC_GTEST_FIXTURE_PROP(FileEncryptionPropertyTest, HeaderCompleteness,
                      (std::vector<uint8_t> file_data)) {
    RC_PRE(file_data.size() <= 1024 * 1024);  // Limit to 1MB
    
    std::istringstream input(std::string(file_data.begin(), file_data.end()));
    std::ostringstream output;
    
    auto result = file_service_->encryptStream(
        input, output, test_kek_id_, createContext(), file_data.size());
    RC_ASSERT(result.has_value());
    
    // Parse header from encrypted output
    std::string encrypted = output.str();
    std::istringstream encrypted_stream(encrypted);
    
    uint32_t header_size;
    encrypted_stream.read(reinterpret_cast<char*>(&header_size), sizeof(header_size));
    RC_ASSERT(header_size > 0);
    
    std::vector<uint8_t> header_data(header_size);
    encrypted_stream.read(reinterpret_cast<char*>(header_data.data()), header_size);
    
    auto header = FileEncryptionHeader::deserialize(header_data);
    RC_ASSERT(header.has_value());
    
    // Verify header completeness
    RC_ASSERT(header->magic == FileEncryptionHeader::MAGIC);
    RC_ASSERT(header->version == FileEncryptionHeader::VERSION);
    RC_ASSERT(!header->wrapped_dek.empty());  // Must have wrapped DEK
    RC_ASSERT(!header->iv.empty());           // Must have IV
    RC_ASSERT(!header->tag.empty());          // Must have authentication tag
    RC_ASSERT(header->iv.size() == AESEngine::GCM_IV_SIZE);
    RC_ASSERT(header->tag.size() == AESEngine::GCM_TAG_SIZE);
    RC_ASSERT(header->original_size == file_data.size());
    RC_ASSERT(header->key_id.toString() == test_kek_id_.toString());
}

/**
 * Property 19: File Encryption Round-Trip
 * 
 * For any valid file (up to 10GB), encrypting and then decrypting
 * SHALL produce a byte-identical copy of the original file.
 * 
 * Validates: Requirements 7.7
 */
RC_GTEST_FIXTURE_PROP(FileEncryptionPropertyTest, RoundTrip,
                      (std::vector<uint8_t> file_data)) {
    RC_PRE(file_data.size() <= 10 * 1024 * 1024);  // Limit to 10MB for testing
    
    // Encrypt
    std::istringstream input(std::string(file_data.begin(), file_data.end()));
    std::ostringstream encrypted_output;
    
    auto encrypt_result = file_service_->encryptStream(
        input, encrypted_output, test_kek_id_, createContext(), file_data.size());
    RC_ASSERT(encrypt_result.has_value());
    
    // Decrypt
    std::string encrypted = encrypted_output.str();
    std::istringstream encrypted_input(encrypted);
    std::ostringstream decrypted_output;
    
    auto decrypt_result = file_service_->decryptStream(
        encrypted_input, decrypted_output, createContext());
    RC_ASSERT(decrypt_result.has_value());
    
    // Verify round-trip
    std::string decrypted = decrypted_output.str();
    std::vector<uint8_t> decrypted_data(decrypted.begin(), decrypted.end());
    
    RC_ASSERT(decrypted_data.size() == file_data.size());
    RC_ASSERT(decrypted_data == file_data);
}

/**
 * Additional test: Empty file handling
 */
RC_GTEST_FIXTURE_PROP(FileEncryptionPropertyTest, EmptyFileRoundTrip, ()) {
    std::vector<uint8_t> empty_data;
    
    std::istringstream input("");
    std::ostringstream encrypted_output;
    
    auto encrypt_result = file_service_->encryptStream(
        input, encrypted_output, test_kek_id_, createContext(), 0);
    RC_ASSERT(encrypt_result.has_value());
    
    std::string encrypted = encrypted_output.str();
    std::istringstream encrypted_input(encrypted);
    std::ostringstream decrypted_output;
    
    auto decrypt_result = file_service_->decryptStream(
        encrypted_input, decrypted_output, createContext());
    RC_ASSERT(decrypt_result.has_value());
    
    RC_ASSERT(decrypted_output.str().empty());
}

/**
 * Additional test: Large chunk handling
 */
RC_GTEST_FIXTURE_PROP(FileEncryptionPropertyTest, LargeFileRoundTrip,
                      (uint32_t seed)) {
    // Generate deterministic large data
    std::vector<uint8_t> large_data(5 * 1024 * 1024);  // 5MB
    std::mt19937 gen(seed);
    std::uniform_int_distribution<> dis(0, 255);
    for (auto& byte : large_data) {
        byte = static_cast<uint8_t>(dis(gen));
    }
    
    std::istringstream input(std::string(large_data.begin(), large_data.end()));
    std::ostringstream encrypted_output;
    
    auto encrypt_result = file_service_->encryptStream(
        input, encrypted_output, test_kek_id_, createContext(), large_data.size());
    RC_ASSERT(encrypt_result.has_value());
    
    std::string encrypted = encrypted_output.str();
    std::istringstream encrypted_input(encrypted);
    std::ostringstream decrypted_output;
    
    auto decrypt_result = file_service_->decryptStream(
        encrypted_input, decrypted_output, createContext());
    RC_ASSERT(decrypt_result.has_value());
    
    std::string decrypted = decrypted_output.str();
    RC_ASSERT(decrypted.size() == large_data.size());
    RC_ASSERT(std::equal(decrypted.begin(), decrypted.end(), large_data.begin()));
}

} // namespace crypto::test
