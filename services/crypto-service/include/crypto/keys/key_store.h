#pragma once

#include "crypto/common/result.h"
#include "crypto/keys/key_types.h"
#include <memory>
#include <mutex>
#include <unordered_map>

namespace crypto {

// Interface for key storage backends
class IKeyStore {
public:
    virtual ~IKeyStore() = default;

    virtual Result<void> store(const KeyId& id, const EncryptedKey& key) = 0;
    virtual Result<EncryptedKey> retrieve(const KeyId& id) = 0;
    virtual Result<void> remove(const KeyId& id) = 0;
    virtual Result<bool> exists(const KeyId& id) = 0;
    virtual Result<std::vector<KeyId>> list(const std::string& namespace_prefix = "") = 0;
    virtual Result<void> updateMetadata(const KeyId& id, const KeyMetadata& metadata) = 0;
};

// In-memory key store (for testing and development)
class InMemoryKeyStore : public IKeyStore {
public:
    InMemoryKeyStore() = default;
    ~InMemoryKeyStore() override = default;

    Result<void> store(const KeyId& id, const EncryptedKey& key) override;
    Result<EncryptedKey> retrieve(const KeyId& id) override;
    Result<void> remove(const KeyId& id) override;
    Result<bool> exists(const KeyId& id) override;
    Result<std::vector<KeyId>> list(const std::string& namespace_prefix = "") override;
    Result<void> updateMetadata(const KeyId& id, const KeyMetadata& metadata) override;

private:
    mutable std::mutex mutex_;
    std::unordered_map<std::string, EncryptedKey> keys_;
};

// Local file-based key store (encrypted storage)
class LocalKeyStore : public IKeyStore {
public:
    explicit LocalKeyStore(const std::string& storage_path, 
                          std::span<const uint8_t> master_key);
    ~LocalKeyStore() override = default;

    Result<void> store(const KeyId& id, const EncryptedKey& key) override;
    Result<EncryptedKey> retrieve(const KeyId& id) override;
    Result<void> remove(const KeyId& id) override;
    Result<bool> exists(const KeyId& id) override;
    Result<std::vector<KeyId>> list(const std::string& namespace_prefix = "") override;
    Result<void> updateMetadata(const KeyId& id, const KeyMetadata& metadata) override;

private:
    std::string storage_path_;
    std::vector<uint8_t> master_key_;
    mutable std::mutex mutex_;

    std::string getKeyPath(const KeyId& id) const;
    Result<void> ensureDirectory() const;
};

} // namespace crypto
