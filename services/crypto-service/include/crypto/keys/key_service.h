#pragma once

/**
 * @file key_service.h
 * @brief Key management service with platform cache integration
 * 
 * Requirements: 2.2, 2.3, 2.4
 */

#include "crypto/common/result.h"
#include "crypto/keys/key_types.h"
#include "crypto/keys/key_store.h"
#include "crypto/clients/cache_client.h"
#include "crypto/engine/aes_engine.h"
#include "crypto/engine/rsa_engine.h"
#include "crypto/engine/ecdsa_engine.h"
#include <memory>
#include <mutex>

namespace crypto {

// Key Service interface
class IKeyService {
public:
    virtual ~IKeyService() = default;

    [[nodiscard]] virtual Result<KeyId> generateKey(const KeyGenerationParams& params) = 0;
    [[nodiscard]] virtual Result<KeyId> rotateKey(const KeyId& old_key_id) = 0;
    [[nodiscard]] virtual Result<KeyMetadata> getKeyMetadata(const KeyId& key_id) = 0;
    [[nodiscard]] virtual Result<void> deleteKey(const KeyId& key_id) = 0;
    
    // Get key material for internal use (never exposed via API)
    [[nodiscard]] virtual Result<std::vector<uint8_t>> getKeyMaterial(const KeyId& key_id) = 0;
};

// Key Service implementation with CacheClient integration
class KeyService : public IKeyService {
public:
    /**
     * @brief Construct KeyService with CacheClient for distributed caching
     * @param key_store Persistent key storage
     * @param master_key Master encryption key for key material
     * @param cache_client Optional CacheClient for distributed caching
     */
    KeyService(std::shared_ptr<IKeyStore> key_store,
               std::span<const uint8_t> master_key,
               std::shared_ptr<CacheClient> cache_client = nullptr);
    ~KeyService() override = default;

    // Non-copyable
    KeyService(const KeyService&) = delete;
    KeyService& operator=(const KeyService&) = delete;

    [[nodiscard]] Result<KeyId> generateKey(const KeyGenerationParams& params) override;
    [[nodiscard]] Result<KeyId> rotateKey(const KeyId& old_key_id) override;
    [[nodiscard]] Result<KeyMetadata> getKeyMetadata(const KeyId& key_id) override;
    [[nodiscard]] Result<void> deleteKey(const KeyId& key_id) override;
    [[nodiscard]] Result<std::vector<uint8_t>> getKeyMaterial(const KeyId& key_id) override;

    // Additional methods
    [[nodiscard]] Result<void> deprecateKey(const KeyId& key_id);
    [[nodiscard]] Result<std::vector<KeyId>> listKeys(const std::string& namespace_prefix = "");

private:
    std::shared_ptr<IKeyStore> key_store_;
    std::vector<uint8_t> master_key_;
    std::shared_ptr<CacheClient> cache_client_;
    AESEngine aes_engine_;
    RSAEngine rsa_engine_;
    ECDSAEngine ecdsa_engine_;
    mutable std::mutex mutex_;

    // Cache key prefix for key material
    static constexpr std::string_view CACHE_KEY_PREFIX = "keymaterial:";
    static constexpr std::chrono::seconds CACHE_TTL{300};

    // Encrypt key material with master key
    [[nodiscard]] Result<EncryptedKey> encryptKeyMaterial(
        std::span<const uint8_t> key_material,
        const KeyMetadata& metadata);

    // Decrypt key material with master key
    [[nodiscard]] Result<std::vector<uint8_t>> decryptKeyMaterial(const EncryptedKey& encrypted_key);

    // Generate raw key material based on algorithm
    [[nodiscard]] Result<std::vector<uint8_t>> generateRawKeyMaterial(KeyAlgorithm algorithm);

    // Cache helpers
    void cacheKeyMaterial(const KeyId& key_id, std::span<const uint8_t> material);
    [[nodiscard]] std::optional<std::vector<uint8_t>> getCachedKeyMaterial(const KeyId& key_id);
    void invalidateCachedKey(const KeyId& key_id);
};

} // namespace crypto
