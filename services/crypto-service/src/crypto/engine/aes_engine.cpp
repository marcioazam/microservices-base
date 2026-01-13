/**
 * @file aes_engine.cpp
 * @brief AES encryption engine implementation using centralized utilities
 * 
 * Requirements: 4.3, 4.4, 5.2, 5.6, 10.5, 10.6
 */

#include "crypto/engine/aes_engine.h"
#include "crypto/common/openssl_raii.h"
#include "crypto/common/hash_utils.h"
#include "crypto/common/input_validation.h"
#include <openssl/evp.h>

namespace crypto {

namespace {

// Get OpenSSL cipher for key size (GCM)
[[nodiscard]] const EVP_CIPHER* getGCMCipher(size_t key_size) noexcept {
    switch (key_size) {
        case 16: return EVP_aes_128_gcm();
        case 32: return EVP_aes_256_gcm();
        default: return nullptr;
    }
}

// Get OpenSSL cipher for key size (CBC)
[[nodiscard]] const EVP_CIPHER* getCBCCipher(size_t key_size) noexcept {
    switch (key_size) {
        case 16: return EVP_aes_128_cbc();
        case 32: return EVP_aes_256_cbc();
        default: return nullptr;
    }
}

} // anonymous namespace

[[nodiscard]] Result<std::vector<uint8_t>> AESEngine::generateIV(size_t size) {
    std::vector<uint8_t> iv(size);
    if (!openssl::random_bytes(iv.data(), size)) {
        return Err<std::vector<uint8_t>>(ErrorCode::CRYPTO_ERROR, "Failed to generate random IV");
    }
    return Ok(std::move(iv));
}

[[nodiscard]] Result<SecureBuffer> AESEngine::generateKey(AESKeySize key_size) {
    size_t size = static_cast<size_t>(key_size);
    SecureBuffer key(size);
    if (!openssl::random_bytes(key.data(), size)) {
        return Err<SecureBuffer>(ErrorCode::KEY_GENERATION_FAILED, "Failed to generate random key");
    }
    return Ok(std::move(key));
}

[[nodiscard]] bool AESEngine::isValidKeySize(size_t size) noexcept {
    return is_valid_aes_key_size(size);
}

[[nodiscard]] Result<EncryptResult> AESEngine::encryptGCM(
    std::span<const uint8_t> plaintext,
    std::span<const uint8_t> key,
    std::span<const uint8_t> aad) {
    
    auto iv_result = generateIV(aes_gcm::IV_SIZE);
    if (!iv_result) {
        return Err<EncryptResult>(iv_result.error());
    }
    return encryptGCMWithIV(plaintext, key, iv_result.value(), aad);
}

[[nodiscard]] Result<EncryptResult> AESEngine::encryptGCMWithIV(
    std::span<const uint8_t> plaintext,
    std::span<const uint8_t> key,
    std::span<const uint8_t> iv,
    std::span<const uint8_t> aad) {
    
    // Input size validation (Requirement 10.5)
    if (auto result = validatePlaintextSize(plaintext.size()); !result) {
        return Err<EncryptResult>(result.error());
    }
    if (auto result = validateAADSize(aad.size()); !result) {
        return Err<EncryptResult>(result.error());
    }
    if (auto result = validateAESKeySize(key.size()); !result) {
        return Err<EncryptResult>(result.error());
    }
    if (auto result = validateGCMIVSize(iv.size()); !result) {
        return Err<EncryptResult>(result.error());
    }

    const EVP_CIPHER* cipher = getGCMCipher(key.size());
    if (!cipher) {
        return Err<EncryptResult>(makeSafeError(ErrorCode::ENCRYPTION_FAILED));
    }

    auto ctx = openssl::make_cipher_ctx();
    if (!ctx) {
        return Err<EncryptResult>(makeSafeError(ErrorCode::ENCRYPTION_FAILED));
    }

    // Initialize encryption
    if (EVP_EncryptInit_ex(ctx.get(), cipher, nullptr, nullptr, nullptr) != 1) {
        return Err<EncryptResult>(makeSafeError(ErrorCode::ENCRYPTION_FAILED));
    }

    // Set IV length
    if (EVP_CIPHER_CTX_ctrl(ctx.get(), EVP_CTRL_GCM_SET_IVLEN, 
                           static_cast<int>(iv.size()), nullptr) != 1) {
        return Err<EncryptResult>(makeSafeError(ErrorCode::ENCRYPTION_FAILED));
    }

    // Set key and IV
    if (EVP_EncryptInit_ex(ctx.get(), nullptr, nullptr, key.data(), iv.data()) != 1) {
        return Err<EncryptResult>(makeSafeError(ErrorCode::ENCRYPTION_FAILED));
    }

    int len = 0;

    // Process AAD if provided
    if (!aad.empty()) {
        if (EVP_EncryptUpdate(ctx.get(), nullptr, &len, aad.data(), 
                             static_cast<int>(aad.size())) != 1) {
            return Err<EncryptResult>(makeSafeError(ErrorCode::ENCRYPTION_FAILED));
        }
    }

    // Encrypt plaintext
    std::vector<uint8_t> ciphertext(plaintext.size() + aes_gcm::BLOCK_SIZE);
    if (EVP_EncryptUpdate(ctx.get(), ciphertext.data(), &len, 
                         plaintext.data(), static_cast<int>(plaintext.size())) != 1) {
        return Err<EncryptResult>(makeSafeError(ErrorCode::ENCRYPTION_FAILED));
    }
    int ciphertext_len = len;

    // Finalize encryption
    if (EVP_EncryptFinal_ex(ctx.get(), ciphertext.data() + len, &len) != 1) {
        return Err<EncryptResult>(makeSafeError(ErrorCode::ENCRYPTION_FAILED));
    }
    ciphertext_len += len;
    ciphertext.resize(static_cast<size_t>(ciphertext_len));

    // Get authentication tag
    std::vector<uint8_t> tag(aes_gcm::TAG_SIZE);
    if (EVP_CIPHER_CTX_ctrl(ctx.get(), EVP_CTRL_GCM_GET_TAG, 
                           static_cast<int>(aes_gcm::TAG_SIZE), tag.data()) != 1) {
        return Err<EncryptResult>(makeSafeError(ErrorCode::ENCRYPTION_FAILED));
    }

    EncryptResult result;
    result.ciphertext = std::move(ciphertext);
    result.iv = std::vector<uint8_t>(iv.begin(), iv.end());
    result.tag = std::move(tag);

    return Ok(std::move(result));
}

[[nodiscard]] Result<std::vector<uint8_t>> AESEngine::decryptGCM(
    std::span<const uint8_t> ciphertext,
    std::span<const uint8_t> key,
    std::span<const uint8_t> iv,
    std::span<const uint8_t> tag,
    std::span<const uint8_t> aad) {
    
    // Input size validation (Requirement 10.5)
    if (auto result = validateCiphertextSize(ciphertext.size()); !result) {
        return Err<std::vector<uint8_t>>(result.error());
    }
    if (auto result = validateAADSize(aad.size()); !result) {
        return Err<std::vector<uint8_t>>(result.error());
    }
    if (auto result = validateAESKeySize(key.size()); !result) {
        return Err<std::vector<uint8_t>>(result.error());
    }
    if (auto result = validateGCMIVSize(iv.size()); !result) {
        return Err<std::vector<uint8_t>>(result.error());
    }
    if (auto result = validateGCMTagSize(tag.size()); !result) {
        return Err<std::vector<uint8_t>>(result.error());
    }

    const EVP_CIPHER* cipher = getGCMCipher(key.size());
    if (!cipher) {
        return Err<std::vector<uint8_t>>(makeSafeError(ErrorCode::DECRYPTION_FAILED));
    }

    auto ctx = openssl::make_cipher_ctx();
    if (!ctx) {
        return Err<std::vector<uint8_t>>(makeSafeError(ErrorCode::DECRYPTION_FAILED));
    }

    // Initialize decryption
    if (EVP_DecryptInit_ex(ctx.get(), cipher, nullptr, nullptr, nullptr) != 1) {
        return Err<std::vector<uint8_t>>(makeSafeError(ErrorCode::DECRYPTION_FAILED));
    }

    // Set IV length
    if (EVP_CIPHER_CTX_ctrl(ctx.get(), EVP_CTRL_GCM_SET_IVLEN,
                           static_cast<int>(iv.size()), nullptr) != 1) {
        return Err<std::vector<uint8_t>>(makeSafeError(ErrorCode::DECRYPTION_FAILED));
    }

    // Set key and IV
    if (EVP_DecryptInit_ex(ctx.get(), nullptr, nullptr, key.data(), iv.data()) != 1) {
        return Err<std::vector<uint8_t>>(makeSafeError(ErrorCode::DECRYPTION_FAILED));
    }

    int len = 0;

    // Process AAD if provided
    if (!aad.empty()) {
        if (EVP_DecryptUpdate(ctx.get(), nullptr, &len, aad.data(),
                             static_cast<int>(aad.size())) != 1) {
            return Err<std::vector<uint8_t>>(makeSafeError(ErrorCode::DECRYPTION_FAILED));
        }
    }

    // Decrypt ciphertext
    std::vector<uint8_t> plaintext(ciphertext.size() + aes_gcm::BLOCK_SIZE);
    if (EVP_DecryptUpdate(ctx.get(), plaintext.data(), &len,
                         ciphertext.data(), static_cast<int>(ciphertext.size())) != 1) {
        return Err<std::vector<uint8_t>>(makeSafeError(ErrorCode::DECRYPTION_FAILED));
    }
    int plaintext_len = len;

    // Set expected tag
    if (EVP_CIPHER_CTX_ctrl(ctx.get(), EVP_CTRL_GCM_SET_TAG,
                           static_cast<int>(tag.size()), 
                           const_cast<uint8_t*>(tag.data())) != 1) {
        return Err<std::vector<uint8_t>>(makeSafeError(ErrorCode::DECRYPTION_FAILED));
    }

    // Finalize decryption and verify tag (Requirement 10.6 - safe error)
    int ret = EVP_DecryptFinal_ex(ctx.get(), plaintext.data() + len, &len);
    if (ret <= 0) {
        return Err<std::vector<uint8_t>>(makeSafeError(ErrorCode::INTEGRITY_ERROR));
    }
    plaintext_len += len;
    plaintext.resize(static_cast<size_t>(plaintext_len));

    return Ok(std::move(plaintext));
}

std::vector<uint8_t> AESEngine::addPKCS7Padding(std::span<const uint8_t> data) {
    size_t padding_len = aes_cbc::BLOCK_SIZE - (data.size() % aes_cbc::BLOCK_SIZE);
    std::vector<uint8_t> padded(data.begin(), data.end());
    padded.resize(data.size() + padding_len, static_cast<uint8_t>(padding_len));
    return padded;
}

[[nodiscard]] Result<std::vector<uint8_t>> AESEngine::removePKCS7Padding(
    std::span<const uint8_t> data) {
    if (data.empty()) {
        return Err<std::vector<uint8_t>>(ErrorCode::PADDING_ERROR, "Empty data");
    }

    uint8_t padding_len = data.back();
    if (padding_len == 0 || padding_len > aes_cbc::BLOCK_SIZE || padding_len > data.size()) {
        return Err<std::vector<uint8_t>>(ErrorCode::PADDING_ERROR, "Invalid padding length");
    }

    // Verify all padding bytes
    for (size_t i = data.size() - padding_len; i < data.size(); ++i) {
        if (data[i] != padding_len) {
            return Err<std::vector<uint8_t>>(ErrorCode::PADDING_ERROR, "Invalid padding bytes");
        }
    }

    return Ok(std::vector<uint8_t>(data.begin(), data.end() - padding_len));
}

[[nodiscard]] Result<EncryptResult> AESEngine::encryptCBC(
    std::span<const uint8_t> plaintext,
    std::span<const uint8_t> key) {
    
    auto iv_result = generateIV(aes_cbc::IV_SIZE);
    if (!iv_result) {
        return Err<EncryptResult>(iv_result.error());
    }
    return encryptCBCWithIV(plaintext, key, iv_result.value());
}

[[nodiscard]] Result<EncryptResult> AESEngine::encryptCBCWithIV(
    std::span<const uint8_t> plaintext,
    std::span<const uint8_t> key,
    std::span<const uint8_t> iv) {
    
    // Input size validation (Requirement 10.5)
    if (auto result = validatePlaintextSize(plaintext.size()); !result) {
        return Err<EncryptResult>(result.error());
    }
    if (auto result = validateAESKeySize(key.size()); !result) {
        return Err<EncryptResult>(result.error());
    }
    if (auto result = validateCBCIVSize(iv.size()); !result) {
        return Err<EncryptResult>(result.error());
    }

    const EVP_CIPHER* cipher = getCBCCipher(key.size());
    if (!cipher) {
        return Err<EncryptResult>(makeSafeError(ErrorCode::ENCRYPTION_FAILED));
    }

    auto ctx = openssl::make_cipher_ctx();
    if (!ctx) {
        return Err<EncryptResult>(makeSafeError(ErrorCode::ENCRYPTION_FAILED));
    }

    // Add PKCS7 padding
    std::vector<uint8_t> padded = addPKCS7Padding(plaintext);

    // Initialize encryption
    if (EVP_EncryptInit_ex(ctx.get(), cipher, nullptr, key.data(), iv.data()) != 1) {
        return Err<EncryptResult>(makeSafeError(ErrorCode::ENCRYPTION_FAILED));
    }

    // Disable OpenSSL's padding (we handle it ourselves)
    EVP_CIPHER_CTX_set_padding(ctx.get(), 0);

    int len = 0;
    std::vector<uint8_t> ciphertext(padded.size() + aes_cbc::BLOCK_SIZE);

    // Encrypt
    if (EVP_EncryptUpdate(ctx.get(), ciphertext.data(), &len,
                         padded.data(), static_cast<int>(padded.size())) != 1) {
        return Err<EncryptResult>(makeSafeError(ErrorCode::ENCRYPTION_FAILED));
    }
    int ciphertext_len = len;

    // Finalize
    if (EVP_EncryptFinal_ex(ctx.get(), ciphertext.data() + len, &len) != 1) {
        return Err<EncryptResult>(makeSafeError(ErrorCode::ENCRYPTION_FAILED));
    }
    ciphertext_len += len;
    ciphertext.resize(static_cast<size_t>(ciphertext_len));

    EncryptResult result;
    result.ciphertext = std::move(ciphertext);
    result.iv = std::vector<uint8_t>(iv.begin(), iv.end());

    return Ok(std::move(result));
}

[[nodiscard]] Result<std::vector<uint8_t>> AESEngine::decryptCBC(
    std::span<const uint8_t> ciphertext,
    std::span<const uint8_t> key,
    std::span<const uint8_t> iv) {
    
    // Input size validation (Requirement 10.5)
    if (auto result = validateCiphertextSize(ciphertext.size()); !result) {
        return Err<std::vector<uint8_t>>(result.error());
    }
    if (auto result = validateAESKeySize(key.size()); !result) {
        return Err<std::vector<uint8_t>>(result.error());
    }
    if (auto result = validateCBCIVSize(iv.size()); !result) {
        return Err<std::vector<uint8_t>>(result.error());
    }

    // Validate ciphertext is multiple of block size
    if (ciphertext.size() % aes_cbc::BLOCK_SIZE != 0) {
        return Err<std::vector<uint8_t>>(ErrorCode::INVALID_INPUT,
            "Ciphertext must be multiple of block size");
    }

    const EVP_CIPHER* cipher = getCBCCipher(key.size());
    if (!cipher) {
        return Err<std::vector<uint8_t>>(makeSafeError(ErrorCode::DECRYPTION_FAILED));
    }

    auto ctx = openssl::make_cipher_ctx();
    if (!ctx) {
        return Err<std::vector<uint8_t>>(makeSafeError(ErrorCode::DECRYPTION_FAILED));
    }

    // Initialize decryption
    if (EVP_DecryptInit_ex(ctx.get(), cipher, nullptr, key.data(), iv.data()) != 1) {
        return Err<std::vector<uint8_t>>(makeSafeError(ErrorCode::DECRYPTION_FAILED));
    }

    // Disable OpenSSL's padding (we handle it ourselves)
    EVP_CIPHER_CTX_set_padding(ctx.get(), 0);

    int len = 0;
    std::vector<uint8_t> plaintext(ciphertext.size() + aes_cbc::BLOCK_SIZE);

    // Decrypt
    if (EVP_DecryptUpdate(ctx.get(), plaintext.data(), &len,
                         ciphertext.data(), static_cast<int>(ciphertext.size())) != 1) {
        return Err<std::vector<uint8_t>>(makeSafeError(ErrorCode::DECRYPTION_FAILED));
    }
    int plaintext_len = len;

    // Finalize
    if (EVP_DecryptFinal_ex(ctx.get(), plaintext.data() + len, &len) != 1) {
        return Err<std::vector<uint8_t>>(makeSafeError(ErrorCode::DECRYPTION_FAILED));
    }
    plaintext_len += len;
    plaintext.resize(static_cast<size_t>(plaintext_len));

    // Remove PKCS7 padding
    return removePKCS7Padding(plaintext);
}

} // namespace crypto
