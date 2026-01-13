#include "crypto/services/encryption_service.h"

namespace crypto {

EncryptionService::EncryptionService(
    std::shared_ptr<KeyService> key_service,
    std::shared_ptr<LoggingClient> logging_client)
    : key_service_(std::move(key_service))
    , logging_client_(std::move(logging_client)) {}

void EncryptionService::logOperation(
    std::string_view operation, const KeyId& key_id,
    const EncryptionContext& ctx, bool success,
    const std::optional<std::string>& error) {
    
    if (!logging_client_) return;
    
    std::map<std::string, std::string> fields = {
        {"operation", std::string(operation)},
        {"key_id", key_id.toString()},
        {"caller_identity", ctx.caller_identity},
        {"caller_service", ctx.caller_service},
        {"source_ip", ctx.source_ip},
        {"success", success ? "true" : "false"}
    };
    
    if (error) {
        fields["error_code"] = *error;
    }
    
    if (success) {
        logging_client_->log(LogLevel::INFO, 
            std::string(operation) + " operation completed",
            ctx.correlation_id, fields);
    } else {
        logging_client_->log(LogLevel::ERROR,
            std::string(operation) + " operation failed",
            ctx.correlation_id, fields);
    }
}

Result<EncryptionResult> EncryptionService::encrypt(
    std::span<const uint8_t> plaintext,
    const KeyId& key_id,
    const EncryptionContext& ctx) {
    
    // Get key material
    auto key_result = key_service_->getKey(key_id);
    if (!key_result) {
        logOperation("encrypt", key_id, ctx, false, "KEY_NOT_FOUND");
        return Err<EncryptionResult>(key_result.error());
    }
    
    const auto& key_data = *key_result;
    if (key_data.metadata.type != KeyType::AES) {
        logOperation("encrypt", key_id, ctx, false, "INVALID_KEY_TYPE");
        return Err<EncryptionResult>(ErrorCode::INVALID_KEY_TYPE,
                                     "Key is not an AES key");
    }
    
    // Perform encryption
    Result<AESCiphertext> encrypt_result;
    if (ctx.aad) {
        encrypt_result = aes_engine_.encryptGCM(plaintext, key_data.key_material, *ctx.aad);
    } else {
        encrypt_result = aes_engine_.encryptGCM(plaintext, key_data.key_material);
    }
    
    if (!encrypt_result) {
        logOperation("encrypt", key_id, ctx, false, "ENCRYPTION_FAILED");
        return Err<EncryptionResult>(encrypt_result.error());
    }
    
    logOperation("encrypt", key_id, ctx, true);
    
    return Ok(EncryptionResult{
        .ciphertext = std::move(encrypt_result->ciphertext),
        .iv = std::move(encrypt_result->iv),
        .tag = std::move(encrypt_result->tag),
        .key_id = key_id,
        .algorithm = "AES-256-GCM"
    });
}

Result<EncryptionResult> EncryptionService::encryptWithNewKey(
    std::span<const uint8_t> plaintext,
    const std::string& key_namespace,
    const EncryptionContext& ctx) {
    
    // Generate new AES key
    auto key_result = key_service_->generateKey(KeyType::AES, 256, key_namespace);
    if (!key_result) {
        return Err<EncryptionResult>(key_result.error());
    }
    
    return encrypt(plaintext, *key_result, ctx);
}

Result<std::vector<uint8_t>> EncryptionService::decrypt(
    const DecryptionRequest& request,
    const EncryptionContext& ctx) {
    
    // Get key material
    auto key_result = key_service_->getKey(request.key_id);
    if (!key_result) {
        logOperation("decrypt", request.key_id, ctx, false, "KEY_NOT_FOUND");
        return Err<std::vector<uint8_t>>(key_result.error());
    }
    
    const auto& key_data = *key_result;
    if (key_data.metadata.type != KeyType::AES) {
        logOperation("decrypt", request.key_id, ctx, false, "INVALID_KEY_TYPE");
        return Err<std::vector<uint8_t>>(ErrorCode::INVALID_KEY_TYPE,
                                         "Key is not an AES key");
    }
    
    // Perform decryption
    Result<std::vector<uint8_t>> decrypt_result;
    if (request.aad) {
        decrypt_result = aes_engine_.decryptGCM(
            request.ciphertext, key_data.key_material,
            request.iv, request.tag, *request.aad);
    } else {
        decrypt_result = aes_engine_.decryptGCM(
            request.ciphertext, key_data.key_material,
            request.iv, request.tag);
    }
    
    if (!decrypt_result) {
        logOperation("decrypt", request.key_id, ctx, false, "DECRYPTION_FAILED");
        return decrypt_result;
    }
    
    logOperation("decrypt", request.key_id, ctx, true);
    return decrypt_result;
}

Result<EncryptionResult> EncryptionService::encryptCBC(
    std::span<const uint8_t> plaintext,
    const KeyId& key_id,
    const EncryptionContext& ctx) {
    
    auto key_result = key_service_->getKey(key_id);
    if (!key_result) {
        logOperation("encrypt_cbc", key_id, ctx, false, "KEY_NOT_FOUND");
        return Err<EncryptionResult>(key_result.error());
    }
    
    const auto& key_data = *key_result;
    if (key_data.metadata.type != KeyType::AES) {
        logOperation("encrypt_cbc", key_id, ctx, false, "INVALID_KEY_TYPE");
        return Err<EncryptionResult>(ErrorCode::INVALID_KEY_TYPE,
                                     "Key is not an AES key");
    }
    
    auto encrypt_result = aes_engine_.encryptCBC(plaintext, key_data.key_material);
    if (!encrypt_result) {
        logOperation("encrypt_cbc", key_id, ctx, false, "ENCRYPTION_FAILED");
        return Err<EncryptionResult>(encrypt_result.error());
    }
    
    logOperation("encrypt_cbc", key_id, ctx, true);
    
    return Ok(EncryptionResult{
        .ciphertext = std::move(encrypt_result->ciphertext),
        .iv = std::move(encrypt_result->iv),
        .tag = {},
        .key_id = key_id,
        .algorithm = "AES-256-CBC"
    });
}

Result<std::vector<uint8_t>> EncryptionService::decryptCBC(
    std::span<const uint8_t> ciphertext,
    std::span<const uint8_t> iv,
    const KeyId& key_id,
    const EncryptionContext& ctx) {
    
    auto key_result = key_service_->getKey(key_id);
    if (!key_result) {
        logOperation("decrypt_cbc", key_id, ctx, false, "KEY_NOT_FOUND");
        return Err<std::vector<uint8_t>>(key_result.error());
    }
    
    const auto& key_data = *key_result;
    if (key_data.metadata.type != KeyType::AES) {
        logOperation("decrypt_cbc", key_id, ctx, false, "INVALID_KEY_TYPE");
        return Err<std::vector<uint8_t>>(ErrorCode::INVALID_KEY_TYPE,
                                         "Key is not an AES key");
    }
    
    auto decrypt_result = aes_engine_.decryptCBC(ciphertext, key_data.key_material, iv);
    if (!decrypt_result) {
        logOperation("decrypt_cbc", key_id, ctx, false, "DECRYPTION_FAILED");
        return decrypt_result;
    }
    
    logOperation("decrypt_cbc", key_id, ctx, true);
    return decrypt_result;
}

} // namespace crypto
