// Feature: crypto-service-modernization-2025
// Property 6: Input Validation and Error Safety
// Property-based tests for input validation and safe error handling

#include <gtest/gtest.h>
#include <rapidcheck.h>
#include <rapidcheck/gtest.h>
#include "crypto/common/input_validation.h"
#include "crypto/common/result.h"
#include <string>
#include <vector>
#include <regex>

namespace crypto::test {

// ============================================================================
// Generators
// ============================================================================

/// Generator for valid plaintext sizes (within limit)
rc::Gen<size_t> genValidPlaintextSize() {
    return rc::gen::inRange<size_t>(0, limits::MAX_PLAINTEXT_SIZE);
}

/// Generator for oversized plaintext (exceeds limit)
rc::Gen<size_t> genOversizedPlaintextSize() {
    return rc::gen::inRange<size_t>(
        limits::MAX_PLAINTEXT_SIZE + 1,
        limits::MAX_PLAINTEXT_SIZE + 1024 * 1024
    );
}

/// Generator for valid ciphertext sizes
rc::Gen<size_t> genValidCiphertextSize() {
    return rc::gen::inRange<size_t>(0, limits::MAX_CIPHERTEXT_SIZE);
}

/// Generator for oversized ciphertext
rc::Gen<size_t> genOversizedCiphertextSize() {
    return rc::gen::inRange<size_t>(
        limits::MAX_CIPHERTEXT_SIZE + 1,
        limits::MAX_CIPHERTEXT_SIZE + 1024 * 1024
    );
}

/// Generator for valid sign data sizes
rc::Gen<size_t> genValidSignDataSize() {
    return rc::gen::inRange<size_t>(0, limits::MAX_SIGN_DATA_SIZE);
}

/// Generator for oversized sign data
rc::Gen<size_t> genOversizedSignDataSize() {
    return rc::gen::inRange<size_t>(
        limits::MAX_SIGN_DATA_SIZE + 1,
        limits::MAX_SIGN_DATA_SIZE + 1024 * 1024
    );
}

/// Generator for valid file sizes
rc::Gen<size_t> genValidFileSize() {
    return rc::gen::inRange<size_t>(0, limits::MAX_FILE_SIZE);
}

/// Generator for oversized files
rc::Gen<size_t> genOversizedFileSize() {
    return rc::gen::inRange<size_t>(
        limits::MAX_FILE_SIZE + 1,
        limits::MAX_FILE_SIZE + 1024 * 1024
    );
}

/// Generator for valid AAD sizes
rc::Gen<size_t> genValidAADSize() {
    return rc::gen::inRange<size_t>(0, limits::MAX_AAD_SIZE);
}

/// Generator for oversized AAD
rc::Gen<size_t> genOversizedAADSize() {
    return rc::gen::inRange<size_t>(
        limits::MAX_AAD_SIZE + 1,
        limits::MAX_AAD_SIZE + 1024
    );
}

/// Generator for valid AES key sizes
rc::Gen<size_t> genValidAESKeySize() {
    return rc::gen::element<size_t>(16, 32);
}

/// Generator for invalid AES key sizes
rc::Gen<size_t> genInvalidAESKeySize() {
    return rc::gen::suchThat(
        rc::gen::inRange<size_t>(0, 64),
        [](size_t s) { return s != 16 && s != 32; }
    );
}

/// Generator for valid RSA key sizes (in bits)
rc::Gen<size_t> genValidRSAKeySize() {
    return rc::gen::element<size_t>(2048, 3072, 4096);
}

/// Generator for invalid RSA key sizes
rc::Gen<size_t> genInvalidRSAKeySize() {
    return rc::gen::suchThat(
        rc::gen::inRange<size_t>(512, 8192),
        [](size_t s) { return s != 2048 && s != 3072 && s != 4096; }
    );
}

/// Generator for valid GCM IV size
rc::Gen<size_t> genValidGCMIVSize() {
    return rc::gen::just<size_t>(12);
}

/// Generator for invalid GCM IV sizes
rc::Gen<size_t> genInvalidGCMIVSize() {
    return rc::gen::suchThat(
        rc::gen::inRange<size_t>(0, 32),
        [](size_t s) { return s != 12; }
    );
}

/// Generator for valid GCM tag size
rc::Gen<size_t> genValidGCMTagSize() {
    return rc::gen::just<size_t>(16);
}

/// Generator for invalid GCM tag sizes
rc::Gen<size_t> genInvalidGCMTagSize() {
    return rc::gen::suchThat(
        rc::gen::inRange<size_t>(0, 32),
        [](size_t s) { return s != 16; }
    );
}

/// Generator for error codes that should have safe messages
rc::Gen<ErrorCode> genSensitiveErrorCode() {
    return rc::gen::element(
        ErrorCode::ENCRYPTION_FAILED,
        ErrorCode::DECRYPTION_FAILED,
        ErrorCode::SIGNATURE_INVALID,
        ErrorCode::INTEGRITY_ERROR
    );
}

// ============================================================================
// Test Fixture
// ============================================================================

class InputValidationPropertiesTest : public ::testing::Test {};

// ============================================================================
// Property 6: Input Validation and Error Safety
// For any input to a cryptographic operation, the Crypto_Service SHALL:
// - Validate input sizes before processing
// - Return errors that do not leak sensitive information
// Validates: Requirements 10.5, 10.6
// ============================================================================

// --- Size Validation Properties ---

RC_GTEST_FIXTURE_PROP(InputValidationPropertiesTest, ValidPlaintextAccepted, ()) {
    auto size = *genValidPlaintextSize();
    
    auto result = validatePlaintextSize(size);
    RC_ASSERT(result.has_value());
}

RC_GTEST_FIXTURE_PROP(InputValidationPropertiesTest, OversizedPlaintextRejected, ()) {
    auto size = *genOversizedPlaintextSize();
    
    auto result = validatePlaintextSize(size);
    RC_ASSERT(result.is_error());
    RC_ASSERT(result.error_code() == ErrorCode::SIZE_LIMIT_EXCEEDED);
}

RC_GTEST_FIXTURE_PROP(InputValidationPropertiesTest, ValidCiphertextAccepted, ()) {
    auto size = *genValidCiphertextSize();
    
    auto result = validateCiphertextSize(size);
    RC_ASSERT(result.has_value());
}

RC_GTEST_FIXTURE_PROP(InputValidationPropertiesTest, OversizedCiphertextRejected, ()) {
    auto size = *genOversizedCiphertextSize();
    
    auto result = validateCiphertextSize(size);
    RC_ASSERT(result.is_error());
    RC_ASSERT(result.error_code() == ErrorCode::SIZE_LIMIT_EXCEEDED);
}

RC_GTEST_FIXTURE_PROP(InputValidationPropertiesTest, ValidSignDataAccepted, ()) {
    auto size = *genValidSignDataSize();
    
    auto result = validateSignDataSize(size);
    RC_ASSERT(result.has_value());
}

RC_GTEST_FIXTURE_PROP(InputValidationPropertiesTest, OversizedSignDataRejected, ()) {
    auto size = *genOversizedSignDataSize();
    
    auto result = validateSignDataSize(size);
    RC_ASSERT(result.is_error());
    RC_ASSERT(result.error_code() == ErrorCode::SIZE_LIMIT_EXCEEDED);
}

RC_GTEST_FIXTURE_PROP(InputValidationPropertiesTest, ValidFileSizeAccepted, ()) {
    auto size = *genValidFileSize();
    
    auto result = validateFileSize(size);
    RC_ASSERT(result.has_value());
}

RC_GTEST_FIXTURE_PROP(InputValidationPropertiesTest, OversizedFileRejected, ()) {
    auto size = *genOversizedFileSize();
    
    auto result = validateFileSize(size);
    RC_ASSERT(result.is_error());
    RC_ASSERT(result.error_code() == ErrorCode::SIZE_LIMIT_EXCEEDED);
}

RC_GTEST_FIXTURE_PROP(InputValidationPropertiesTest, ValidAADAccepted, ()) {
    auto size = *genValidAADSize();
    
    auto result = validateAADSize(size);
    RC_ASSERT(result.has_value());
}

RC_GTEST_FIXTURE_PROP(InputValidationPropertiesTest, OversizedAADRejected, ()) {
    auto size = *genOversizedAADSize();
    
    auto result = validateAADSize(size);
    RC_ASSERT(result.is_error());
    RC_ASSERT(result.error_code() == ErrorCode::SIZE_LIMIT_EXCEEDED);
}

// --- Key Size Validation Properties ---

RC_GTEST_FIXTURE_PROP(InputValidationPropertiesTest, ValidAESKeySizeAccepted, ()) {
    auto size = *genValidAESKeySize();
    
    auto result = validateAESKeySize(size);
    RC_ASSERT(result.has_value());
}

RC_GTEST_FIXTURE_PROP(InputValidationPropertiesTest, InvalidAESKeySizeRejected, ()) {
    auto size = *genInvalidAESKeySize();
    
    auto result = validateAESKeySize(size);
    RC_ASSERT(result.is_error());
    RC_ASSERT(result.error_code() == ErrorCode::INVALID_KEY_SIZE);
}

RC_GTEST_FIXTURE_PROP(InputValidationPropertiesTest, ValidRSAKeySizeAccepted, ()) {
    auto bits = *genValidRSAKeySize();
    
    auto result = validateRSAKeySize(bits);
    RC_ASSERT(result.has_value());
}

RC_GTEST_FIXTURE_PROP(InputValidationPropertiesTest, InvalidRSAKeySizeRejected, ()) {
    auto bits = *genInvalidRSAKeySize();
    
    auto result = validateRSAKeySize(bits);
    RC_ASSERT(result.is_error());
    RC_ASSERT(result.error_code() == ErrorCode::INVALID_KEY_SIZE);
}

// --- IV/Tag Size Validation Properties ---

RC_GTEST_FIXTURE_PROP(InputValidationPropertiesTest, ValidGCMIVAccepted, ()) {
    auto size = *genValidGCMIVSize();
    
    auto result = validateGCMIVSize(size);
    RC_ASSERT(result.has_value());
}

RC_GTEST_FIXTURE_PROP(InputValidationPropertiesTest, InvalidGCMIVRejected, ()) {
    auto size = *genInvalidGCMIVSize();
    
    auto result = validateGCMIVSize(size);
    RC_ASSERT(result.is_error());
    RC_ASSERT(result.error_code() == ErrorCode::INVALID_IV_SIZE);
}

RC_GTEST_FIXTURE_PROP(InputValidationPropertiesTest, ValidGCMTagAccepted, ()) {
    auto size = *genValidGCMTagSize();
    
    auto result = validateGCMTagSize(size);
    RC_ASSERT(result.has_value());
}

RC_GTEST_FIXTURE_PROP(InputValidationPropertiesTest, InvalidGCMTagRejected, ()) {
    auto size = *genInvalidGCMTagSize();
    
    auto result = validateGCMTagSize(size);
    RC_ASSERT(result.is_error());
    RC_ASSERT(result.error_code() == ErrorCode::INVALID_TAG_SIZE);
}

// --- Safe Error Message Properties ---

RC_GTEST_FIXTURE_PROP(InputValidationPropertiesTest, SafeErrorsNoSensitiveData, ()) {
    auto code = *genSensitiveErrorCode();
    
    auto error = makeSafeError(code);
    
    // Error message should not contain sensitive patterns
    std::string msg = error.message;
    
    // Should not contain key material patterns
    RC_ASSERT(msg.find("-----BEGIN") == std::string::npos);
    RC_ASSERT(msg.find("key=") == std::string::npos);
    RC_ASSERT(msg.find("password") == std::string::npos);
    RC_ASSERT(msg.find("secret") == std::string::npos);
    
    // Should not contain hex dumps
    std::regex hex_pattern("[0-9a-fA-F]{32,}");
    RC_ASSERT(!std::regex_search(msg, hex_pattern));
    
    // Should not contain base64 encoded data
    std::regex base64_pattern("[A-Za-z0-9+/]{20,}={0,2}");
    RC_ASSERT(!std::regex_search(msg, base64_pattern));
}

RC_GTEST_FIXTURE_PROP(InputValidationPropertiesTest, SafeErrorsAreGeneric, ()) {
    auto code = *genSensitiveErrorCode();
    
    auto error = makeSafeError(code);
    
    // Message should be generic (not specific about what failed)
    RC_ASSERT(error.message.length() < 100);
    RC_ASSERT(error.message.find("byte") == std::string::npos);
    RC_ASSERT(error.message.find("offset") == std::string::npos);
    RC_ASSERT(error.message.find("position") == std::string::npos);
}

// ============================================================================
// Unit Tests for Edge Cases
// ============================================================================

TEST_F(InputValidationPropertiesTest, ZeroSizeAccepted) {
    EXPECT_TRUE(validatePlaintextSize(0).has_value());
    EXPECT_TRUE(validateCiphertextSize(0).has_value());
    EXPECT_TRUE(validateSignDataSize(0).has_value());
    EXPECT_TRUE(validateFileSize(0).has_value());
    EXPECT_TRUE(validateAADSize(0).has_value());
}

TEST_F(InputValidationPropertiesTest, ExactLimitAccepted) {
    EXPECT_TRUE(validatePlaintextSize(limits::MAX_PLAINTEXT_SIZE).has_value());
    EXPECT_TRUE(validateCiphertextSize(limits::MAX_CIPHERTEXT_SIZE).has_value());
    EXPECT_TRUE(validateSignDataSize(limits::MAX_SIGN_DATA_SIZE).has_value());
    EXPECT_TRUE(validateFileSize(limits::MAX_FILE_SIZE).has_value());
    EXPECT_TRUE(validateAADSize(limits::MAX_AAD_SIZE).has_value());
}

TEST_F(InputValidationPropertiesTest, OneBytePastLimitRejected) {
    EXPECT_FALSE(validatePlaintextSize(limits::MAX_PLAINTEXT_SIZE + 1).has_value());
    EXPECT_FALSE(validateCiphertextSize(limits::MAX_CIPHERTEXT_SIZE + 1).has_value());
    EXPECT_FALSE(validateSignDataSize(limits::MAX_SIGN_DATA_SIZE + 1).has_value());
    EXPECT_FALSE(validateFileSize(limits::MAX_FILE_SIZE + 1).has_value());
    EXPECT_FALSE(validateAADSize(limits::MAX_AAD_SIZE + 1).has_value());
}

TEST_F(InputValidationPropertiesTest, AESKeySize128Accepted) {
    EXPECT_TRUE(validateAESKeySize(16).has_value());
}

TEST_F(InputValidationPropertiesTest, AESKeySize256Accepted) {
    EXPECT_TRUE(validateAESKeySize(32).has_value());
}

TEST_F(InputValidationPropertiesTest, AESKeySize192Rejected) {
    // AES-192 is not supported (24 bytes)
    auto result = validateAESKeySize(24);
    EXPECT_FALSE(result.has_value());
    EXPECT_EQ(result.error().code, ErrorCode::INVALID_KEY_SIZE);
}

TEST_F(InputValidationPropertiesTest, RSAKeySize1024Rejected) {
    // RSA-1024 is too weak
    auto result = validateRSAKeySize(1024);
    EXPECT_FALSE(result.has_value());
    EXPECT_EQ(result.error().code, ErrorCode::INVALID_KEY_SIZE);
}

TEST_F(InputValidationPropertiesTest, CBCIVSize16Accepted) {
    EXPECT_TRUE(validateCBCIVSize(16).has_value());
}

TEST_F(InputValidationPropertiesTest, CBCIVSize12Rejected) {
    auto result = validateCBCIVSize(12);
    EXPECT_FALSE(result.has_value());
    EXPECT_EQ(result.error().code, ErrorCode::INVALID_IV_SIZE);
}

TEST_F(InputValidationPropertiesTest, SafeErrorMessages) {
    EXPECT_EQ(safe_errors::ENCRYPTION_FAILED, "Encryption operation failed");
    EXPECT_EQ(safe_errors::DECRYPTION_FAILED, "Decryption operation failed");
    EXPECT_EQ(safe_errors::SIGNATURE_FAILED, "Signature operation failed");
    EXPECT_EQ(safe_errors::VERIFICATION_FAILED, "Signature verification failed");
    EXPECT_EQ(safe_errors::KEY_OPERATION_FAILED, "Key operation failed");
    EXPECT_EQ(safe_errors::INTEGRITY_FAILED, "Data integrity verification failed");
}

TEST_F(InputValidationPropertiesTest, MakeSafeErrorPreservesCode) {
    auto error = makeSafeError(ErrorCode::ENCRYPTION_FAILED);
    EXPECT_EQ(error.code, ErrorCode::ENCRYPTION_FAILED);
    
    error = makeSafeError(ErrorCode::DECRYPTION_FAILED);
    EXPECT_EQ(error.code, ErrorCode::DECRYPTION_FAILED);
    
    error = makeSafeError(ErrorCode::INTEGRITY_ERROR);
    EXPECT_EQ(error.code, ErrorCode::INTEGRITY_ERROR);
}

TEST_F(InputValidationPropertiesTest, LimitsAreReasonable) {
    // Verify limits are set to reasonable values
    EXPECT_EQ(limits::MAX_PLAINTEXT_SIZE, 64 * 1024 * 1024);  // 64 MB
    EXPECT_EQ(limits::MAX_SIGN_DATA_SIZE, 16 * 1024 * 1024);  // 16 MB
    EXPECT_EQ(limits::MAX_FILE_SIZE, 1024 * 1024 * 1024);     // 1 GB
    EXPECT_EQ(limits::MAX_AAD_SIZE, 64 * 1024);               // 64 KB
    EXPECT_EQ(limits::MAX_KEY_SIZE, 8 * 1024);                // 8 KB
}

} // namespace crypto::test
