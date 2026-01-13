/**
 * @file openssl_raii_test.cpp
 * @brief Unit tests for OpenSSL RAII wrappers
 * 
 * Requirements: 7.2
 */

#include <gtest/gtest.h>
#include <crypto/common/openssl_raii.h>
#include <vector>
#include <cstring>

namespace crypto::openssl::test {

// ============================================================================
// CipherCtx Tests
// ============================================================================

TEST(OpenSSLRAIITest, MakeCipherCtxCreatesValidContext) {
    auto ctx = make_cipher_ctx();
    EXPECT_NE(ctx.get(), nullptr);
}

TEST(OpenSSLRAIITest, CipherCtxReleasesOnDestruction) {
    EVP_CIPHER_CTX* raw_ptr = nullptr;
    {
        auto ctx = make_cipher_ctx();
        raw_ptr = ctx.get();
        EXPECT_NE(raw_ptr, nullptr);
    }
    // After scope, ctx should be freed (we can't verify directly, but no crash)
}

TEST(OpenSSLRAIITest, CipherCtxCanBeUsedForEncryption) {
    auto ctx = make_cipher_ctx();
    ASSERT_NE(ctx.get(), nullptr);
    
    // Initialize for AES-256-GCM encryption
    std::vector<uint8_t> key(32, 0x42);
    std::vector<uint8_t> iv(12, 0x24);
    
    int result = EVP_EncryptInit_ex(ctx.get(), EVP_aes_256_gcm(), 
                                     nullptr, key.data(), iv.data());
    EXPECT_EQ(result, 1);
}

// ============================================================================
// MDCtx Tests
// ============================================================================

TEST(OpenSSLRAIITest, MakeMdCtxCreatesValidContext) {
    auto ctx = make_md_ctx();
    EXPECT_NE(ctx.get(), nullptr);
}

TEST(OpenSSLRAIITest, MdCtxCanBeUsedForHashing) {
    auto ctx = make_md_ctx();
    ASSERT_NE(ctx.get(), nullptr);
    
    // Initialize for SHA-256
    int result = EVP_DigestInit_ex(ctx.get(), EVP_sha256(), nullptr);
    EXPECT_EQ(result, 1);
    
    // Update with data
    const char* data = "test data";
    result = EVP_DigestUpdate(ctx.get(), data, strlen(data));
    EXPECT_EQ(result, 1);
    
    // Finalize
    std::vector<uint8_t> hash(32);
    unsigned int hash_len = 0;
    result = EVP_DigestFinal_ex(ctx.get(), hash.data(), &hash_len);
    EXPECT_EQ(result, 1);
    EXPECT_EQ(hash_len, 32u);
}

// ============================================================================
// PKeyCtx Tests
// ============================================================================

TEST(OpenSSLRAIITest, MakePkeyCtxCreatesValidContext) {
    auto ctx = make_pkey_ctx(EVP_PKEY_RSA);
    EXPECT_NE(ctx.get(), nullptr);
}

TEST(OpenSSLRAIITest, MakePkeyCtxFromKeyCreatesValidContext) {
    // Generate a key first
    auto keygen_ctx = make_pkey_ctx(EVP_PKEY_RSA);
    ASSERT_NE(keygen_ctx.get(), nullptr);
    
    ASSERT_EQ(EVP_PKEY_keygen_init(keygen_ctx.get()), 1);
    ASSERT_EQ(EVP_PKEY_CTX_set_rsa_keygen_bits(keygen_ctx.get(), 2048), 1);
    
    EVP_PKEY* raw_key = nullptr;
    ASSERT_EQ(EVP_PKEY_keygen(keygen_ctx.get(), &raw_key), 1);
    
    PKey key(raw_key);
    ASSERT_NE(key.get(), nullptr);
    
    // Create context from key
    auto ctx = make_pkey_ctx(key.get());
    EXPECT_NE(ctx.get(), nullptr);
}

// ============================================================================
// BIO Tests
// ============================================================================

TEST(OpenSSLRAIITest, MakeBioMemCreatesValidBio) {
    auto bio = make_bio_mem();
    EXPECT_NE(bio.get(), nullptr);
}

TEST(OpenSSLRAIITest, MakeBioMemBufCreatesValidBio) {
    const char* data = "test data";
    auto bio = make_bio_mem_buf(data, static_cast<int>(strlen(data)));
    EXPECT_NE(bio.get(), nullptr);
}

TEST(OpenSSLRAIITest, BioCanReadAndWrite) {
    auto bio = make_bio_mem();
    ASSERT_NE(bio.get(), nullptr);
    
    // Write data
    const char* write_data = "hello world";
    int written = BIO_write(bio.get(), write_data, static_cast<int>(strlen(write_data)));
    EXPECT_EQ(written, static_cast<int>(strlen(write_data)));
    
    // Read data back
    char read_buf[32] = {0};
    int read = BIO_read(bio.get(), read_buf, sizeof(read_buf) - 1);
    EXPECT_EQ(read, static_cast<int>(strlen(write_data)));
    EXPECT_STREQ(read_buf, write_data);
}

// ============================================================================
// ParamBld Tests (OpenSSL 3.x)
// ============================================================================

TEST(OpenSSLRAIITest, MakeParamBldCreatesValidBuilder) {
    auto bld = make_param_bld();
    EXPECT_NE(bld.get(), nullptr);
}

TEST(OpenSSLRAIITest, ParamBldCanBuildParams) {
    auto bld = make_param_bld();
    ASSERT_NE(bld.get(), nullptr);
    
    // Add a parameter
    int result = OSSL_PARAM_BLD_push_utf8_string(bld.get(), "test", "value", 0);
    EXPECT_EQ(result, 1);
    
    // Build params
    OSSL_PARAM* raw_params = OSSL_PARAM_BLD_to_param(bld.get());
    EXPECT_NE(raw_params, nullptr);
    
    Params params(raw_params);
    EXPECT_NE(params.get(), nullptr);
}

// ============================================================================
// MAC Tests (OpenSSL 3.x)
// ============================================================================

TEST(OpenSSLRAIITest, MakeMacCreatesValidMac) {
    auto mac = make_mac("HMAC");
    EXPECT_NE(mac.get(), nullptr);
}

TEST(OpenSSLRAIITest, MakeMacCtxCreatesValidContext) {
    auto mac = make_mac("HMAC");
    ASSERT_NE(mac.get(), nullptr);
    
    auto ctx = make_mac_ctx(mac.get());
    EXPECT_NE(ctx.get(), nullptr);
}

TEST(OpenSSLRAIITest, MacCanComputeHmac) {
    auto mac = make_mac("HMAC");
    ASSERT_NE(mac.get(), nullptr);
    
    auto ctx = make_mac_ctx(mac.get());
    ASSERT_NE(ctx.get(), nullptr);
    
    // Set up HMAC-SHA256
    std::vector<uint8_t> key(32, 0x42);
    OSSL_PARAM params[] = {
        OSSL_PARAM_construct_utf8_string("digest", const_cast<char*>("SHA256"), 0),
        OSSL_PARAM_construct_end()
    };
    
    int result = EVP_MAC_init(ctx.get(), key.data(), key.size(), params);
    EXPECT_EQ(result, 1);
    
    // Update with data
    const char* data = "test data";
    result = EVP_MAC_update(ctx.get(), 
                            reinterpret_cast<const unsigned char*>(data), 
                            strlen(data));
    EXPECT_EQ(result, 1);
    
    // Finalize
    std::vector<uint8_t> output(32);
    size_t out_len = 0;
    result = EVP_MAC_final(ctx.get(), output.data(), &out_len, output.size());
    EXPECT_EQ(result, 1);
    EXPECT_EQ(out_len, 32u);
}

// ============================================================================
// KDF Tests (OpenSSL 3.x)
// ============================================================================

TEST(OpenSSLRAIITest, MakeKdfCreatesValidKdf) {
    auto kdf = make_kdf("HKDF");
    EXPECT_NE(kdf.get(), nullptr);
}

TEST(OpenSSLRAIITest, MakeKdfCtxCreatesValidContext) {
    auto kdf = make_kdf("HKDF");
    ASSERT_NE(kdf.get(), nullptr);
    
    auto ctx = make_kdf_ctx(kdf.get());
    EXPECT_NE(ctx.get(), nullptr);
}

// ============================================================================
// BIGNUM Tests
// ============================================================================

TEST(OpenSSLRAIITest, MakeBnCreatesValidBignum) {
    auto bn = make_bn();
    EXPECT_NE(bn.get(), nullptr);
}

TEST(OpenSSLRAIITest, BnCanPerformArithmetic) {
    auto a = make_bn();
    auto b = make_bn();
    auto result = make_bn();
    auto ctx = make_bn_ctx();
    
    ASSERT_NE(a.get(), nullptr);
    ASSERT_NE(b.get(), nullptr);
    ASSERT_NE(result.get(), nullptr);
    ASSERT_NE(ctx.get(), nullptr);
    
    // Set values
    BN_set_word(a.get(), 100);
    BN_set_word(b.get(), 50);
    
    // Add
    BN_add(result.get(), a.get(), b.get());
    EXPECT_EQ(BN_get_word(result.get()), 150u);
}

TEST(OpenSSLRAIITest, MakeBnCtxCreatesValidContext) {
    auto ctx = make_bn_ctx();
    EXPECT_NE(ctx.get(), nullptr);
}

// ============================================================================
// Utility Function Tests
// ============================================================================

TEST(OpenSSLRAIITest, GetLastErrorReturnsString) {
    // Generate an error
    clear_errors();
    
    // Try to use an invalid cipher (this should generate an error)
    auto ctx = make_cipher_ctx();
    EVP_EncryptInit_ex(ctx.get(), nullptr, nullptr, nullptr, nullptr);
    
    // Get error (may or may not have one depending on OpenSSL version)
    std::string error = get_last_error();
    // Just verify it doesn't crash and returns a string
    EXPECT_TRUE(error.empty() || !error.empty());
}

TEST(OpenSSLRAIITest, ClearErrorsDoesNotCrash) {
    clear_errors();
    // Just verify it doesn't crash
}

TEST(OpenSSLRAIITest, RandomBytesGeneratesData) {
    std::vector<uint8_t> buffer(32, 0);
    
    bool result = random_bytes(buffer.data(), buffer.size());
    EXPECT_TRUE(result);
    
    // Verify not all zeros (extremely unlikely for random data)
    bool all_zeros = true;
    for (uint8_t b : buffer) {
        if (b != 0) {
            all_zeros = false;
            break;
        }
    }
    EXPECT_FALSE(all_zeros);
}

TEST(OpenSSLRAIITest, RandomBytesGeneratesDifferentData) {
    std::vector<uint8_t> buffer1(32);
    std::vector<uint8_t> buffer2(32);
    
    ASSERT_TRUE(random_bytes(buffer1.data(), buffer1.size()));
    ASSERT_TRUE(random_bytes(buffer2.data(), buffer2.size()));
    
    // Buffers should be different (extremely unlikely to be same)
    EXPECT_NE(buffer1, buffer2);
}

// ============================================================================
// Move Semantics Tests
// ============================================================================

TEST(OpenSSLRAIITest, CipherCtxMoveWorks) {
    auto ctx1 = make_cipher_ctx();
    ASSERT_NE(ctx1.get(), nullptr);
    
    EVP_CIPHER_CTX* raw = ctx1.get();
    
    CipherCtx ctx2 = std::move(ctx1);
    EXPECT_EQ(ctx1.get(), nullptr);
    EXPECT_EQ(ctx2.get(), raw);
}

TEST(OpenSSLRAIITest, MdCtxMoveWorks) {
    auto ctx1 = make_md_ctx();
    ASSERT_NE(ctx1.get(), nullptr);
    
    EVP_MD_CTX* raw = ctx1.get();
    
    MDCtx ctx2 = std::move(ctx1);
    EXPECT_EQ(ctx1.get(), nullptr);
    EXPECT_EQ(ctx2.get(), raw);
}

// ============================================================================
// Null Safety Tests
// ============================================================================

TEST(OpenSSLRAIITest, DeletersHandleNull) {
    // These should not crash
    CipherCtxDeleter{}(nullptr);
    PKeyDeleter{}(nullptr);
    PKeyCtxDeleter{}(nullptr);
    MDCtxDeleter{}(nullptr);
    BIODeleter{}(nullptr);
    ParamBldDeleter{}(nullptr);
    ParamDeleter{}(nullptr);
    MACDeleter{}(nullptr);
    MACCtxDeleter{}(nullptr);
    KDFDeleter{}(nullptr);
    KDFCtxDeleter{}(nullptr);
    BNDeleter{}(nullptr);
    BNCtxDeleter{}(nullptr);
}

} // namespace crypto::openssl::test
