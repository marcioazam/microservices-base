#include "crypto/services/signature_service.h"

namespace crypto {

SignatureService::SignatureService(
    std::shared_ptr<KeyService> key_service,
    std::shared_ptr<LoggingClient> logging_client)
    : key_service_(std::move(key_service))
    , logging_client_(std::move(logging_client)) {}

void SignatureService::logOperation(
    std::string_view operation, const KeyId& key_id,
    const SignatureContext& ctx, bool success,
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

Result<SignatureResult> SignatureService::signRSA(
    std::span<const uint8_t> data,
    const KeyId& key_id,
    HashAlgorithm hash_algo,
    const SignatureContext& ctx) {
    
    auto key_result = key_service_->getKey(key_id);
    if (!key_result) {
        logOperation("sign_rsa", key_id, ctx, false, "KEY_NOT_FOUND");
        return Err<SignatureResult>(key_result.error());
    }
    
    const auto& key_data = *key_result;
    if (key_data.metadata.type != KeyType::RSA) {
        logOperation("sign_rsa", key_id, ctx, false, "INVALID_KEY_TYPE");
        return Err<SignatureResult>(ErrorCode::INVALID_KEY_TYPE, "Key is not an RSA key");
    }
    
    auto sign_result = rsa_engine_.signPSS(data, key_data.private_key, hash_algo);
    if (!sign_result) {
        logOperation("sign_rsa", key_id, ctx, false, "SIGN_FAILED");
        return Err<SignatureResult>(sign_result.error());
    }
    
    logOperation("sign_rsa", key_id, ctx, true);
    
    std::string algo = "RSA-PSS-" + std::string(get_hash_name(hash_algo));
    
    return Ok(SignatureResult{
        .signature = std::move(*sign_result),
        .key_id = key_id,
        .algorithm = algo,
        .hash_algorithm = hash_algo
    });
}

Result<VerificationResult> SignatureService::verifyRSA(
    std::span<const uint8_t> data,
    std::span<const uint8_t> signature,
    const KeyId& key_id,
    HashAlgorithm hash_algo,
    const SignatureContext& ctx) {
    
    auto key_result = key_service_->getKey(key_id);
    if (!key_result) {
        logOperation("verify_rsa", key_id, ctx, false, "KEY_NOT_FOUND");
        return Err<VerificationResult>(key_result.error());
    }
    
    const auto& key_data = *key_result;
    if (key_data.metadata.type != KeyType::RSA) {
        logOperation("verify_rsa", key_id, ctx, false, "INVALID_KEY_TYPE");
        return Err<VerificationResult>(ErrorCode::INVALID_KEY_TYPE, "Key is not an RSA key");
    }
    
    auto verify_result = rsa_engine_.verifyPSS(data, signature, key_data.public_key, hash_algo);
    if (!verify_result) {
        logOperation("verify_rsa", key_id, ctx, false, "VERIFY_FAILED");
        return Err<VerificationResult>(verify_result.error());
    }
    
    logOperation("verify_rsa", key_id, ctx, true);
    
    std::string algo = "RSA-PSS-" + std::string(get_hash_name(hash_algo));
    
    return Ok(VerificationResult{
        .valid = *verify_result,
        .key_id = key_id,
        .algorithm = algo
    });
}

Result<SignatureResult> SignatureService::signECDSA(
    std::span<const uint8_t> data,
    const KeyId& key_id,
    const SignatureContext& ctx) {
    
    auto key_result = key_service_->getKey(key_id);
    if (!key_result) {
        logOperation("sign_ecdsa", key_id, ctx, false, "KEY_NOT_FOUND");
        return Err<SignatureResult>(key_result.error());
    }
    
    const auto& key_data = *key_result;
    if (key_data.metadata.type != KeyType::ECDSA) {
        logOperation("sign_ecdsa", key_id, ctx, false, "INVALID_KEY_TYPE");
        return Err<SignatureResult>(ErrorCode::INVALID_KEY_TYPE, "Key is not an ECDSA key");
    }
    
    auto sign_result = ecdsa_engine_.sign(data, key_data.private_key);
    if (!sign_result) {
        logOperation("sign_ecdsa", key_id, ctx, false, "SIGN_FAILED");
        return Err<SignatureResult>(sign_result.error());
    }
    
    logOperation("sign_ecdsa", key_id, ctx, true);
    
    return Ok(SignatureResult{
        .signature = std::move(*sign_result),
        .key_id = key_id,
        .algorithm = "ECDSA",
        .hash_algorithm = HashAlgorithm::SHA256
    });
}

Result<VerificationResult> SignatureService::verifyECDSA(
    std::span<const uint8_t> data,
    std::span<const uint8_t> signature,
    const KeyId& key_id,
    const SignatureContext& ctx) {
    
    auto key_result = key_service_->getKey(key_id);
    if (!key_result) {
        logOperation("verify_ecdsa", key_id, ctx, false, "KEY_NOT_FOUND");
        return Err<VerificationResult>(key_result.error());
    }
    
    const auto& key_data = *key_result;
    if (key_data.metadata.type != KeyType::ECDSA) {
        logOperation("verify_ecdsa", key_id, ctx, false, "INVALID_KEY_TYPE");
        return Err<VerificationResult>(ErrorCode::INVALID_KEY_TYPE, "Key is not an ECDSA key");
    }
    
    auto verify_result = ecdsa_engine_.verify(data, signature, key_data.public_key);
    if (!verify_result) {
        logOperation("verify_ecdsa", key_id, ctx, false, "VERIFY_FAILED");
        return Err<VerificationResult>(verify_result.error());
    }
    
    logOperation("verify_ecdsa", key_id, ctx, true);
    
    return Ok(VerificationResult{
        .valid = *verify_result,
        .key_id = key_id,
        .algorithm = "ECDSA"
    });
}

Result<SignatureResult> SignatureService::sign(
    std::span<const uint8_t> data,
    const KeyId& key_id,
    const SignatureContext& ctx) {
    
    auto key_result = key_service_->getKey(key_id);
    if (!key_result) {
        return Err<SignatureResult>(key_result.error());
    }
    
    switch (key_result->metadata.type) {
        case KeyType::RSA:
            return signRSA(data, key_id, HashAlgorithm::SHA256, ctx);
        case KeyType::ECDSA:
            return signECDSA(data, key_id, ctx);
        default:
            return Err<SignatureResult>(ErrorCode::INVALID_KEY_TYPE,
                                        "Key type does not support signing");
    }
}

Result<VerificationResult> SignatureService::verify(
    std::span<const uint8_t> data,
    std::span<const uint8_t> signature,
    const KeyId& key_id,
    const SignatureContext& ctx) {
    
    auto key_result = key_service_->getKey(key_id);
    if (!key_result) {
        return Err<VerificationResult>(key_result.error());
    }
    
    switch (key_result->metadata.type) {
        case KeyType::RSA:
            return verifyRSA(data, signature, key_id, HashAlgorithm::SHA256, ctx);
        case KeyType::ECDSA:
            return verifyECDSA(data, signature, key_id, ctx);
        default:
            return Err<VerificationResult>(ErrorCode::INVALID_KEY_TYPE,
                                           "Key type does not support verification");
    }
}

} // namespace crypto
