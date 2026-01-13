#include "crypto/keys/key_service.h"
#include <openssl/rand.h>
#include <format>

namespace crypto {

KeyService::KeyService(std::shared_ptr<IKeyStore> key_store,
                       std::span<const uint8_t> master_key,
                       std::shared_ptr<CacheClient> cache_client)
    : key_store_(std::move(key_store))
    , master_key_(master_key.begin(), master_key.end())
    , cache_client_(std::move(cache_client)) {}

// Cache helper implementations
void KeyService::cacheKeyMaterial(const KeyId& key_id, std::span<const uint8_t> material) {
    if (!cache_client_) return;
    
    auto cache_key = std::format("{}{}", CACHE_KEY_PREFIX, key_id.toString());
    auto result = cache_client_->set(cache_key, material, CACHE_TTL);
    // Silently ignore cache errors - cache is optional optimization
}

std::optional<std::vector<uint8_t>> KeyService::getCachedKeyMaterial(const KeyId& key_id) {
    if (!cache_client_) return std::nullopt;
    
    auto cache_key = std::format("{}{}", CACHE_KEY_PREFIX, key_id.toString());
    auto result = cache_client_->get(cache_key);
    if (result) {
        return std::move(*result);
    }
    return std::nullopt;
}

void KeyService::invalidateCachedKey(const KeyId& key_id) {
    if (!cache_client_) return;
    
    auto cache_key = std::format("{}{}", CACHE_KEY_PREFIX, key_id.toString());
    cache_client_->del(cache_key);
    // Silently ignore cache errors
}

Result<std::vector<uint8_t>> KeyService::generateRawKeyMaterial(KeyAlgorithm algorithm) {
    size_t key_size = getKeySize(algorithm);
    if (key_size == 0) {
        return Err<std::vector<uint8_t>>(ErrorCode::INVALID_INPUT, "Invalid algorithm");
    }

    if (isSymmetricAlgorithm(algorithm)) {
        // Generate random bytes for symmetric key
        std::vector<uint8_t> key(key_size);
        if (RAND_bytes(key.data(), static_cast<int>(key_size)) != 1) {
            return Err<std::vector<uint8_t>>(ErrorCode::KEY_GENERATION_FAILED,
                "Failed to generate random key");
        }
        return Ok(std::move(key));
    }

    // For asymmetric keys, generate key pair and return private key DER
    switch (algorithm) {
        case KeyAlgorithm::RSA_2048: {
            auto result = rsa_engine_.generateKeyPair(RSAKeySize::RSA_2048);
            if (!result) return Err<std::vector<uint8_t>>(result.error());
            return result->exportPrivateKeyDER();
        }
        case KeyAlgorithm::RSA_3072: {
            auto result = rsa_engine_.generateKeyPair(RSAKeySize::RSA_3072);
            if (!result) return Err<std::vector<uint8_t>>(result.error());
            return result->exportPrivateKeyDER();
        }
        case KeyAlgorithm::RSA_4096: {
            auto result = rsa_engine_.generateKeyPair(RSAKeySize::RSA_4096);
            if (!result) return Err<std::vector<uint8_t>>(result.error());
            return result->exportPrivateKeyDER();
        }
        case KeyAlgorithm::ECDSA_P256: {
            auto result = ecdsa_engine_.generateKeyPair(ECCurve::P256);
            if (!result) return Err<std::vector<uint8_t>>(result.error());
            return result->exportPrivateKeyDER();
        }
        case KeyAlgorithm::ECDSA_P384: {
            auto result = ecdsa_engine_.generateKeyPair(ECCurve::P384);
            if (!result) return Err<std::vector<uint8_t>>(result.error());
            return result->exportPrivateKeyDER();
        }
        case KeyAlgorithm::ECDSA_P521: {
            auto result = ecdsa_engine_.generateKeyPair(ECCurve::P521);
            if (!result) return Err<std::vector<uint8_t>>(result.error());
            return result->exportPrivateKeyDER();
        }
        default:
            return Err<std::vector<uint8_t>>(ErrorCode::INVALID_INPUT, "Unsupported algorithm");
    }
}

Result<EncryptedKey> KeyService::encryptKeyMaterial(
    std::span<const uint8_t> key_material,
    const KeyMetadata& metadata) {
    
    // Encrypt key material with master key using AES-256-GCM
    auto encrypt_result = aes_engine_.encryptGCM(key_material, master_key_);
    if (!encrypt_result) {
        return Err<EncryptedKey>(encrypt_result.error());
    }

    EncryptedKey encrypted;
    encrypted.encrypted_material = std::move(encrypt_result->ciphertext);
    encrypted.iv = std::move(encrypt_result->iv);
    encrypted.tag = std::move(encrypt_result->tag);
    encrypted.metadata = metadata;

    return Ok(std::move(encrypted));
}

Result<std::vector<uint8_t>> KeyService::decryptKeyMaterial(const EncryptedKey& encrypted_key) {
    return aes_engine_.decryptGCM(
        encrypted_key.encrypted_material,
        master_key_,
        encrypted_key.iv,
        encrypted_key.tag
    );
}

Result<KeyId> KeyService::generateKey(const KeyGenerationParams& params) {
    std::lock_guard<std::mutex> lock(mutex_);

    // Generate key ID
    KeyId key_id = KeyId::generate(params.namespace_prefix);

    // Generate raw key material
    auto key_material_result = generateRawKeyMaterial(params.algorithm);
    if (!key_material_result) {
        return Err<KeyId>(key_material_result.error());
    }

    // Create metadata
    KeyMetadata metadata;
    metadata.id = key_id;
    metadata.algorithm = params.algorithm;
    metadata.type = isSymmetricAlgorithm(params.algorithm) 
        ? KeyType::SYMMETRIC 
        : KeyType::ASYMMETRIC_PRIVATE;
    metadata.state = KeyState::ACTIVE;
    metadata.created_at = std::chrono::system_clock::now();
    metadata.expires_at = metadata.created_at + params.validity_period;
    metadata.owner_service = params.owner_service;
    metadata.allowed_operations = params.allowed_operations;
    metadata.usage_count = 0;

    // Encrypt key material
    auto encrypted_result = encryptKeyMaterial(*key_material_result, metadata);
    if (!encrypted_result) {
        return Err<KeyId>(encrypted_result.error());
    }

    // Store encrypted key
    auto store_result = key_store_->store(key_id, *encrypted_result);
    if (!store_result) {
        return Err<KeyId>(store_result.error());
    }

    // Cache the key material
    cacheKeyMaterial(key_id, *key_material_result);

    return Ok(key_id);
}

Result<KeyId> KeyService::rotateKey(const KeyId& old_key_id) {
    std::lock_guard<std::mutex> lock(mutex_);

    // Get old key metadata
    auto old_key_result = key_store_->retrieve(old_key_id);
    if (!old_key_result) {
        return Err<KeyId>(old_key_result.error());
    }

    const auto& old_metadata = old_key_result->metadata;

    // Check if key can be rotated
    if (old_metadata.state != KeyState::ACTIVE) {
        return Err<KeyId>(ErrorCode::KEY_ROTATION_FAILED,
            "Only active keys can be rotated");
    }

    // Generate new key with same algorithm
    KeyGenerationParams params;
    params.namespace_prefix = old_key_id.namespace_prefix;
    params.algorithm = old_metadata.algorithm;
    params.owner_service = old_metadata.owner_service;
    params.allowed_operations = old_metadata.allowed_operations;

    // Generate new key material
    auto key_material_result = generateRawKeyMaterial(params.algorithm);
    if (!key_material_result) {
        return Err<KeyId>(key_material_result.error());
    }

    // Create new key ID with incremented version
    KeyId new_key_id(old_key_id.namespace_prefix, 
                     UUID::generate().to_string(),
                     old_key_id.version + 1);

    // Create new metadata
    KeyMetadata new_metadata;
    new_metadata.id = new_key_id;
    new_metadata.algorithm = old_metadata.algorithm;
    new_metadata.type = old_metadata.type;
    new_metadata.state = KeyState::ACTIVE;
    new_metadata.created_at = std::chrono::system_clock::now();
    new_metadata.expires_at = new_metadata.created_at + 
        std::chrono::duration_cast<std::chrono::seconds>(
            old_metadata.expires_at - old_metadata.created_at);
    new_metadata.rotated_at = new_metadata.created_at;
    new_metadata.previous_version = old_key_id;
    new_metadata.owner_service = old_metadata.owner_service;
    new_metadata.allowed_operations = old_metadata.allowed_operations;
    new_metadata.usage_count = 0;

    // Encrypt new key material
    auto encrypted_result = encryptKeyMaterial(*key_material_result, new_metadata);
    if (!encrypted_result) {
        return Err<KeyId>(encrypted_result.error());
    }

    // Store new key
    auto store_result = key_store_->store(new_key_id, *encrypted_result);
    if (!store_result) {
        return Err<KeyId>(store_result.error());
    }

    // Deprecate old key
    auto deprecate_result = deprecateKey(old_key_id);
    if (!deprecate_result) {
        // Rollback: remove new key
        key_store_->remove(new_key_id);
        return Err<KeyId>(deprecate_result.error());
    }

    // Update cache - invalidate old, cache new
    invalidateCachedKey(old_key_id);
    cacheKeyMaterial(new_key_id, *key_material_result);

    return Ok(new_key_id);
}

Result<void> KeyService::deprecateKey(const KeyId& key_id) {
    auto key_result = key_store_->retrieve(key_id);
    if (!key_result) {
        return Err<void>(key_result.error());
    }

    auto metadata = key_result->metadata;
    metadata.state = KeyState::DEPRECATED;

    return key_store_->updateMetadata(key_id, metadata);
}

Result<KeyMetadata> KeyService::getKeyMetadata(const KeyId& key_id) {
    std::lock_guard<std::mutex> lock(mutex_);

    // Metadata is not cached - always fetch from store
    auto key_result = key_store_->retrieve(key_id);
    if (!key_result) {
        return Err<KeyMetadata>(key_result.error());
    }

    return Ok(key_result->metadata);
}

Result<void> KeyService::deleteKey(const KeyId& key_id) {
    std::lock_guard<std::mutex> lock(mutex_);

    // Invalidate cache first
    invalidateCachedKey(key_id);

    return key_store_->remove(key_id);
}

Result<std::vector<uint8_t>> KeyService::getKeyMaterial(const KeyId& key_id) {
    std::lock_guard<std::mutex> lock(mutex_);

    // Check cache first
    auto cached = getCachedKeyMaterial(key_id);
    if (cached) {
        return Ok(std::move(*cached));
    }

    // Retrieve from store
    auto key_result = key_store_->retrieve(key_id);
    if (!key_result) {
        return Err<std::vector<uint8_t>>(key_result.error());
    }

    // Decrypt key material
    auto decrypted = decryptKeyMaterial(*key_result);
    if (!decrypted) {
        return Err<std::vector<uint8_t>>(decrypted.error());
    }

    // Cache the key material
    cacheKeyMaterial(key_id, *decrypted);

    return decrypted;
}

Result<std::vector<KeyId>> KeyService::listKeys(const std::string& namespace_prefix) {
    return key_store_->list(namespace_prefix);
}

} // namespace crypto
