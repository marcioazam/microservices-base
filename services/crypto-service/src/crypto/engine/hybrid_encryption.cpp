#include "crypto/engine/hybrid_encryption.h"

namespace crypto {

Result<HybridEncryptResult> HybridEncryption::encrypt(
    std::span<const uint8_t> plaintext,
    const RSAKeyPair& public_key,
    std::span<const uint8_t> aad) {
    
    if (!public_key.isValid()) {
        return Err<HybridEncryptResult>(ErrorCode::INVALID_INPUT, "Invalid public key");
    }

    // 1. Generate random AES-256 key
    auto key_result = AESEngine::generateKey(AESKeySize::AES_256);
    if (!key_result) {
        return Err<HybridEncryptResult>(key_result.error());
    }

    // 2. Encrypt data with AES-256-GCM
    auto encrypt_result = aes_engine_.encryptGCM(plaintext, key_result->span(), aad);
    if (!encrypt_result) {
        return Err<HybridEncryptResult>(encrypt_result.error());
    }

    // 3. Wrap AES key with RSA-OAEP
    auto wrap_result = rsa_engine_.encryptOAEP(key_result->span(), public_key);
    if (!wrap_result) {
        return Err<HybridEncryptResult>(wrap_result.error());
    }

    HybridEncryptResult result;
    result.wrapped_key = std::move(*wrap_result);
    result.ciphertext = std::move(encrypt_result->ciphertext);
    result.iv = std::move(encrypt_result->iv);
    result.tag = std::move(encrypt_result->tag);

    return Ok(std::move(result));
}

Result<std::vector<uint8_t>> HybridEncryption::decrypt(
    const HybridEncryptResult& encrypted,
    const RSAKeyPair& private_key,
    std::span<const uint8_t> aad) {
    
    return decrypt(
        encrypted.wrapped_key,
        encrypted.ciphertext,
        encrypted.iv,
        encrypted.tag,
        private_key,
        aad
    );
}

Result<std::vector<uint8_t>> HybridEncryption::decrypt(
    std::span<const uint8_t> wrapped_key,
    std::span<const uint8_t> ciphertext,
    std::span<const uint8_t> iv,
    std::span<const uint8_t> tag,
    const RSAKeyPair& private_key,
    std::span<const uint8_t> aad) {
    
    if (!private_key.isValid()) {
        return Err<std::vector<uint8_t>>(ErrorCode::INVALID_INPUT, "Invalid private key");
    }

    // 1. Unwrap AES key with RSA-OAEP
    auto unwrap_result = rsa_engine_.decryptOAEP(wrapped_key, private_key);
    if (!unwrap_result) {
        return Err<std::vector<uint8_t>>(unwrap_result.error());
    }

    // Validate unwrapped key size
    if (unwrap_result->size() != static_cast<size_t>(AESKeySize::AES_256)) {
        return Err<std::vector<uint8_t>>(ErrorCode::INVALID_KEY_SIZE, 
            "Unwrapped key has invalid size");
    }

    // 2. Decrypt data with AES-256-GCM
    return aes_engine_.decryptGCM(ciphertext, *unwrap_result, iv, tag, aad);
}

} // namespace crypto
