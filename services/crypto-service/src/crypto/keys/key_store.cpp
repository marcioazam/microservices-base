#include "crypto/keys/key_store.h"
#include "crypto/engine/aes_engine.h"
#include <fstream>
#include <filesystem>
#include <algorithm>

namespace crypto {

// InMemoryKeyStore implementation
Result<void> InMemoryKeyStore::store(const KeyId& id, const EncryptedKey& key) {
    std::lock_guard<std::mutex> lock(mutex_);
    keys_[id.toString()] = key;
    return Ok();
}

Result<EncryptedKey> InMemoryKeyStore::retrieve(const KeyId& id) {
    std::lock_guard<std::mutex> lock(mutex_);
    auto it = keys_.find(id.toString());
    if (it == keys_.end()) {
        return Err<EncryptedKey>(ErrorCode::KEY_NOT_FOUND, "Key not found: " + id.toString());
    }
    return Ok(it->second);
}

Result<void> InMemoryKeyStore::remove(const KeyId& id) {
    std::lock_guard<std::mutex> lock(mutex_);
    auto it = keys_.find(id.toString());
    if (it == keys_.end()) {
        return Err<void>(ErrorCode::KEY_NOT_FOUND, "Key not found: " + id.toString());
    }
    keys_.erase(it);
    return Ok();
}

Result<bool> InMemoryKeyStore::exists(const KeyId& id) {
    std::lock_guard<std::mutex> lock(mutex_);
    return Ok(keys_.find(id.toString()) != keys_.end());
}

Result<std::vector<KeyId>> InMemoryKeyStore::list(const std::string& namespace_prefix) {
    std::lock_guard<std::mutex> lock(mutex_);
    std::vector<KeyId> result;
    
    for (const auto& [key_str, _] : keys_) {
        auto key_id_result = KeyId::parse(key_str);
        if (key_id_result.has_value()) {
            if (namespace_prefix.empty() || 
                key_id_result->namespace_prefix == namespace_prefix) {
                result.push_back(*key_id_result);
            }
        }
    }
    
    return Ok(std::move(result));
}

Result<void> InMemoryKeyStore::updateMetadata(const KeyId& id, const KeyMetadata& metadata) {
    std::lock_guard<std::mutex> lock(mutex_);
    auto it = keys_.find(id.toString());
    if (it == keys_.end()) {
        return Err<void>(ErrorCode::KEY_NOT_FOUND, "Key not found: " + id.toString());
    }
    it->second.metadata = metadata;
    return Ok();
}

// LocalKeyStore implementation
LocalKeyStore::LocalKeyStore(const std::string& storage_path,
                            std::span<const uint8_t> master_key)
    : storage_path_(storage_path)
    , master_key_(master_key.begin(), master_key.end()) {
    ensureDirectory();
}

std::string LocalKeyStore::getKeyPath(const KeyId& id) const {
    // Replace colons with underscores for filesystem compatibility
    std::string filename = id.toString();
    std::replace(filename.begin(), filename.end(), ':', '_');
    return storage_path_ + "/" + filename + ".key";
}

Result<void> LocalKeyStore::ensureDirectory() const {
    try {
        std::filesystem::create_directories(storage_path_);
        return Ok();
    } catch (const std::exception& e) {
        return Err<void>(ErrorCode::INTERNAL_ERROR, 
            std::string("Failed to create directory: ") + e.what());
    }
}

Result<void> LocalKeyStore::store(const KeyId& id, const EncryptedKey& key) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    // Serialize the encrypted key
    // Format: [iv_len][iv][tag_len][tag][data_len][data][metadata...]
    std::vector<uint8_t> serialized;
    
    // IV
    uint32_t iv_len = static_cast<uint32_t>(key.iv.size());
    serialized.insert(serialized.end(), 
        reinterpret_cast<uint8_t*>(&iv_len),
        reinterpret_cast<uint8_t*>(&iv_len) + sizeof(iv_len));
    serialized.insert(serialized.end(), key.iv.begin(), key.iv.end());
    
    // Tag
    uint32_t tag_len = static_cast<uint32_t>(key.tag.size());
    serialized.insert(serialized.end(),
        reinterpret_cast<uint8_t*>(&tag_len),
        reinterpret_cast<uint8_t*>(&tag_len) + sizeof(tag_len));
    serialized.insert(serialized.end(), key.tag.begin(), key.tag.end());
    
    // Encrypted material
    uint32_t data_len = static_cast<uint32_t>(key.encrypted_material.size());
    serialized.insert(serialized.end(),
        reinterpret_cast<uint8_t*>(&data_len),
        reinterpret_cast<uint8_t*>(&data_len) + sizeof(data_len));
    serialized.insert(serialized.end(), 
        key.encrypted_material.begin(), key.encrypted_material.end());
    
    // Metadata (simplified - just key state and algorithm)
    uint32_t algo = static_cast<uint32_t>(key.metadata.algorithm);
    uint32_t state = static_cast<uint32_t>(key.metadata.state);
    serialized.insert(serialized.end(),
        reinterpret_cast<uint8_t*>(&algo),
        reinterpret_cast<uint8_t*>(&algo) + sizeof(algo));
    serialized.insert(serialized.end(),
        reinterpret_cast<uint8_t*>(&state),
        reinterpret_cast<uint8_t*>(&state) + sizeof(state));

    // Write to file
    std::string path = getKeyPath(id);
    std::ofstream file(path, std::ios::binary);
    if (!file) {
        return Err<void>(ErrorCode::INTERNAL_ERROR, "Failed to open file for writing");
    }
    
    file.write(reinterpret_cast<const char*>(serialized.data()), serialized.size());
    if (!file) {
        return Err<void>(ErrorCode::INTERNAL_ERROR, "Failed to write key file");
    }
    
    return Ok();
}

Result<EncryptedKey> LocalKeyStore::retrieve(const KeyId& id) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    std::string path = getKeyPath(id);
    std::ifstream file(path, std::ios::binary | std::ios::ate);
    if (!file) {
        return Err<EncryptedKey>(ErrorCode::KEY_NOT_FOUND, "Key not found: " + id.toString());
    }
    
    std::streamsize size = file.tellg();
    file.seekg(0, std::ios::beg);
    
    std::vector<uint8_t> data(size);
    if (!file.read(reinterpret_cast<char*>(data.data()), size)) {
        return Err<EncryptedKey>(ErrorCode::INTERNAL_ERROR, "Failed to read key file");
    }
    
    // Deserialize
    EncryptedKey key;
    size_t offset = 0;
    
    // IV
    if (offset + sizeof(uint32_t) > data.size()) {
        return Err<EncryptedKey>(ErrorCode::INTERNAL_ERROR, "Corrupted key file");
    }
    uint32_t iv_len = *reinterpret_cast<uint32_t*>(data.data() + offset);
    offset += sizeof(uint32_t);
    
    if (offset + iv_len > data.size()) {
        return Err<EncryptedKey>(ErrorCode::INTERNAL_ERROR, "Corrupted key file");
    }
    key.iv.assign(data.begin() + offset, data.begin() + offset + iv_len);
    offset += iv_len;
    
    // Tag
    if (offset + sizeof(uint32_t) > data.size()) {
        return Err<EncryptedKey>(ErrorCode::INTERNAL_ERROR, "Corrupted key file");
    }
    uint32_t tag_len = *reinterpret_cast<uint32_t*>(data.data() + offset);
    offset += sizeof(uint32_t);
    
    if (offset + tag_len > data.size()) {
        return Err<EncryptedKey>(ErrorCode::INTERNAL_ERROR, "Corrupted key file");
    }
    key.tag.assign(data.begin() + offset, data.begin() + offset + tag_len);
    offset += tag_len;
    
    // Encrypted material
    if (offset + sizeof(uint32_t) > data.size()) {
        return Err<EncryptedKey>(ErrorCode::INTERNAL_ERROR, "Corrupted key file");
    }
    uint32_t data_len = *reinterpret_cast<uint32_t*>(data.data() + offset);
    offset += sizeof(uint32_t);
    
    if (offset + data_len > data.size()) {
        return Err<EncryptedKey>(ErrorCode::INTERNAL_ERROR, "Corrupted key file");
    }
    key.encrypted_material.assign(data.begin() + offset, data.begin() + offset + data_len);
    offset += data_len;
    
    // Metadata
    if (offset + 2 * sizeof(uint32_t) > data.size()) {
        return Err<EncryptedKey>(ErrorCode::INTERNAL_ERROR, "Corrupted key file");
    }
    key.metadata.algorithm = static_cast<KeyAlgorithm>(
        *reinterpret_cast<uint32_t*>(data.data() + offset));
    offset += sizeof(uint32_t);
    key.metadata.state = static_cast<KeyState>(
        *reinterpret_cast<uint32_t*>(data.data() + offset));
    key.metadata.id = id;
    
    return Ok(std::move(key));
}

Result<void> LocalKeyStore::remove(const KeyId& id) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    std::string path = getKeyPath(id);
    try {
        if (!std::filesystem::remove(path)) {
            return Err<void>(ErrorCode::KEY_NOT_FOUND, "Key not found: " + id.toString());
        }
        return Ok();
    } catch (const std::exception& e) {
        return Err<void>(ErrorCode::INTERNAL_ERROR, 
            std::string("Failed to remove key: ") + e.what());
    }
}

Result<bool> LocalKeyStore::exists(const KeyId& id) {
    std::lock_guard<std::mutex> lock(mutex_);
    return Ok(std::filesystem::exists(getKeyPath(id)));
}

Result<std::vector<KeyId>> LocalKeyStore::list(const std::string& namespace_prefix) {
    std::lock_guard<std::mutex> lock(mutex_);
    std::vector<KeyId> result;
    
    try {
        for (const auto& entry : std::filesystem::directory_iterator(storage_path_)) {
            if (entry.path().extension() == ".key") {
                std::string filename = entry.path().stem().string();
                // Convert underscores back to colons
                std::replace(filename.begin(), filename.end(), '_', ':');
                
                auto key_id_result = KeyId::parse(filename);
                if (key_id_result.has_value()) {
                    if (namespace_prefix.empty() ||
                        key_id_result->namespace_prefix == namespace_prefix) {
                        result.push_back(*key_id_result);
                    }
                }
            }
        }
    } catch (const std::exception& e) {
        return Err<std::vector<KeyId>>(ErrorCode::INTERNAL_ERROR,
            std::string("Failed to list keys: ") + e.what());
    }
    
    return Ok(std::move(result));
}

Result<void> LocalKeyStore::updateMetadata(const KeyId& id, const KeyMetadata& metadata) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    // Read existing key
    auto key_result = retrieve(id);
    if (!key_result) {
        return Err<void>(key_result.error());
    }
    
    // Update metadata
    auto key = std::move(*key_result);
    key.metadata = metadata;
    
    // Store back
    return store(id, key);
}

} // namespace crypto
