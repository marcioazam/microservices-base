/**
 * @file cache_client.cpp
 * @brief Implementation of gRPC client for cache-service
 * 
 * Requirements: 2.1, 2.2, 2.6
 */

#include "crypto/clients/cache_client.h"
#include "crypto/common/openssl_raii.h"
#include <grpcpp/grpcpp.h>
#include <list>
#include <unordered_map>
#include <mutex>
#include <atomic>
#include <format>
#include <openssl/evp.h>
#include <openssl/rand.h>

namespace crypto {

// ============================================================================
// Local LRU Cache
// ============================================================================

class LocalLRUCache {
public:
    explicit LocalLRUCache(size_t max_size) : max_size_(max_size) {}
    
    std::optional<std::vector<uint8_t>> get(const std::string& key) {
        std::lock_guard<std::mutex> lock(mutex_);
        
        auto it = cache_map_.find(key);
        if (it == cache_map_.end()) {
            return std::nullopt;
        }
        
        // Move to front (most recently used)
        lru_list_.splice(lru_list_.begin(), lru_list_, it->second.list_it);
        return it->second.value;
    }
    
    void set(const std::string& key, std::vector<uint8_t> value) {
        std::lock_guard<std::mutex> lock(mutex_);
        
        auto it = cache_map_.find(key);
        if (it != cache_map_.end()) {
            // Update existing
            it->second.value = std::move(value);
            lru_list_.splice(lru_list_.begin(), lru_list_, it->second.list_it);
            return;
        }
        
        // Evict if at capacity
        while (cache_map_.size() >= max_size_ && !lru_list_.empty()) {
            auto& lru_key = lru_list_.back();
            cache_map_.erase(lru_key);
            lru_list_.pop_back();
        }
        
        // Insert new
        lru_list_.push_front(key);
        cache_map_[key] = {std::move(value), lru_list_.begin()};
    }
    
    void del(const std::string& key) {
        std::lock_guard<std::mutex> lock(mutex_);
        
        auto it = cache_map_.find(key);
        if (it != cache_map_.end()) {
            lru_list_.erase(it->second.list_it);
            cache_map_.erase(it);
        }
    }
    
    bool exists(const std::string& key) {
        std::lock_guard<std::mutex> lock(mutex_);
        return cache_map_.find(key) != cache_map_.end();
    }
    
    void clear() {
        std::lock_guard<std::mutex> lock(mutex_);
        cache_map_.clear();
        lru_list_.clear();
    }

private:
    struct CacheEntry {
        std::vector<uint8_t> value;
        std::list<std::string>::iterator list_it;
    };
    
    size_t max_size_;
    std::list<std::string> lru_list_;
    std::unordered_map<std::string, CacheEntry> cache_map_;
    mutable std::mutex mutex_;
};

// ============================================================================
// Implementation
// ============================================================================

struct CacheClient::Impl {
    CacheClientConfig config;
    std::shared_ptr<grpc::Channel> channel;
    std::unique_ptr<LocalLRUCache> local_cache;
    
    std::atomic<bool> connected{false};
    std::atomic<size_t> local_hits{0};
    std::atomic<size_t> local_misses{0};
    
    explicit Impl(const CacheClientConfig& cfg) : config(cfg) {
        // Create gRPC channel
        grpc::ChannelArguments args;
        args.SetInt(GRPC_ARG_KEEPALIVE_TIME_MS, 10000);
        args.SetInt(GRPC_ARG_KEEPALIVE_TIMEOUT_MS, 5000);
        
        channel = grpc::CreateCustomChannel(
            config.address,
            grpc::InsecureChannelCredentials(),
            args
        );
        
        // Initialize local cache if enabled
        if (config.local_fallback_enabled) {
            local_cache = std::make_unique<LocalLRUCache>(config.local_cache_size);
        }
    }
    
    bool check_connection() {
        auto state = channel->GetState(true);
        connected = (state == GRPC_CHANNEL_READY || state == GRPC_CHANNEL_IDLE);
        return connected;
    }
    
    // Encrypt value if encryption key is configured
    std::expected<std::vector<uint8_t>, CacheError> 
    encrypt_value(std::span<const uint8_t> plaintext) {
        if (!config.encryption_key) {
            return std::vector<uint8_t>(plaintext.begin(), plaintext.end());
        }
        
        // AES-256-GCM encryption
        constexpr size_t IV_SIZE = 12;
        constexpr size_t TAG_SIZE = 16;
        
        std::vector<uint8_t> iv(IV_SIZE);
        if (!openssl::random_bytes(iv.data(), IV_SIZE)) {
            return std::unexpected(CacheError{CacheErrorCode::ENCRYPTION_ERROR, 
                "Failed to generate IV"});
        }
        
        auto ctx = openssl::make_cipher_ctx();
        if (!ctx) {
            return std::unexpected(CacheError{CacheErrorCode::ENCRYPTION_ERROR,
                "Failed to create cipher context"});
        }
        
        if (EVP_EncryptInit_ex(ctx.get(), EVP_aes_256_gcm(), nullptr,
                               config.encryption_key->data(), iv.data()) != 1) {
            return std::unexpected(CacheError{CacheErrorCode::ENCRYPTION_ERROR,
                "Failed to init encryption"});
        }
        
        std::vector<uint8_t> ciphertext(plaintext.size() + EVP_MAX_BLOCK_LENGTH);
        int len = 0;
        
        if (EVP_EncryptUpdate(ctx.get(), ciphertext.data(), &len,
                              plaintext.data(), static_cast<int>(plaintext.size())) != 1) {
            return std::unexpected(CacheError{CacheErrorCode::ENCRYPTION_ERROR,
                "Encryption failed"});
        }
        
        int ciphertext_len = len;
        if (EVP_EncryptFinal_ex(ctx.get(), ciphertext.data() + len, &len) != 1) {
            return std::unexpected(CacheError{CacheErrorCode::ENCRYPTION_ERROR,
                "Encryption finalization failed"});
        }
        ciphertext_len += len;
        ciphertext.resize(ciphertext_len);
        
        // Get tag
        std::vector<uint8_t> tag(TAG_SIZE);
        if (EVP_CIPHER_CTX_ctrl(ctx.get(), EVP_CTRL_GCM_GET_TAG, TAG_SIZE, tag.data()) != 1) {
            return std::unexpected(CacheError{CacheErrorCode::ENCRYPTION_ERROR,
                "Failed to get auth tag"});
        }
        
        // Format: IV || ciphertext || tag
        std::vector<uint8_t> result;
        result.reserve(IV_SIZE + ciphertext.size() + TAG_SIZE);
        result.insert(result.end(), iv.begin(), iv.end());
        result.insert(result.end(), ciphertext.begin(), ciphertext.end());
        result.insert(result.end(), tag.begin(), tag.end());
        
        return result;
    }
    
    // Decrypt value if encryption key is configured
    std::expected<std::vector<uint8_t>, CacheError>
    decrypt_value(std::span<const uint8_t> encrypted) {
        if (!config.encryption_key) {
            return std::vector<uint8_t>(encrypted.begin(), encrypted.end());
        }
        
        constexpr size_t IV_SIZE = 12;
        constexpr size_t TAG_SIZE = 16;
        
        if (encrypted.size() < IV_SIZE + TAG_SIZE) {
            return std::unexpected(CacheError{CacheErrorCode::ENCRYPTION_ERROR,
                "Invalid encrypted data"});
        }
        
        auto iv = encrypted.subspan(0, IV_SIZE);
        auto ciphertext = encrypted.subspan(IV_SIZE, encrypted.size() - IV_SIZE - TAG_SIZE);
        auto tag = encrypted.subspan(encrypted.size() - TAG_SIZE, TAG_SIZE);
        
        auto ctx = openssl::make_cipher_ctx();
        if (!ctx) {
            return std::unexpected(CacheError{CacheErrorCode::ENCRYPTION_ERROR,
                "Failed to create cipher context"});
        }
        
        if (EVP_DecryptInit_ex(ctx.get(), EVP_aes_256_gcm(), nullptr,
                               config.encryption_key->data(), iv.data()) != 1) {
            return std::unexpected(CacheError{CacheErrorCode::ENCRYPTION_ERROR,
                "Failed to init decryption"});
        }
        
        std::vector<uint8_t> plaintext(ciphertext.size());
        int len = 0;
        
        if (EVP_DecryptUpdate(ctx.get(), plaintext.data(), &len,
                              ciphertext.data(), static_cast<int>(ciphertext.size())) != 1) {
            return std::unexpected(CacheError{CacheErrorCode::ENCRYPTION_ERROR,
                "Decryption failed"});
        }
        
        int plaintext_len = len;
        
        // Set expected tag
        if (EVP_CIPHER_CTX_ctrl(ctx.get(), EVP_CTRL_GCM_SET_TAG, TAG_SIZE,
                                const_cast<uint8_t*>(tag.data())) != 1) {
            return std::unexpected(CacheError{CacheErrorCode::ENCRYPTION_ERROR,
                "Failed to set auth tag"});
        }
        
        if (EVP_DecryptFinal_ex(ctx.get(), plaintext.data() + len, &len) != 1) {
            return std::unexpected(CacheError{CacheErrorCode::ENCRYPTION_ERROR,
                "Authentication failed"});
        }
        plaintext_len += len;
        plaintext.resize(plaintext_len);
        
        return plaintext;
    }
};

// ============================================================================
// CacheClient Implementation
// ============================================================================

CacheClient::CacheClient(const CacheClientConfig& config)
    : impl_(std::make_unique<Impl>(config)) {}

CacheClient::~CacheClient() = default;

CacheClient::CacheClient(CacheClient&&) noexcept = default;
CacheClient& CacheClient::operator=(CacheClient&&) noexcept = default;

std::string CacheClient::build_key(std::string_view key) const {
    return std::format("{}:{}", impl_->config.namespace_prefix, key);
}

std::expected<std::vector<uint8_t>, CacheError> CacheClient::get(std::string_view key) {
    auto full_key = build_key(key);
    
    // Try remote cache first
    if (impl_->check_connection()) {
        // TODO: Implement actual gRPC call
        // For now, fall through to local cache
    }
    
    // Try local cache
    if (impl_->local_cache) {
        auto result = impl_->local_cache->get(full_key);
        if (result) {
            impl_->local_hits++;
            return impl_->decrypt_value(*result);
        }
        impl_->local_misses++;
    }
    
    return std::unexpected(CacheError{CacheErrorCode::NOT_FOUND, "Key not found"});
}

std::expected<void, CacheError> CacheClient::set(
    std::string_view key,
    std::span<const uint8_t> value,
    std::optional<std::chrono::seconds> ttl) {
    
    auto full_key = build_key(key);
    
    // Encrypt value
    auto encrypted = impl_->encrypt_value(value);
    if (!encrypted) {
        return std::unexpected(encrypted.error());
    }
    
    // Try remote cache
    if (impl_->check_connection()) {
        // TODO: Implement actual gRPC call
    }
    
    // Store in local cache
    if (impl_->local_cache) {
        impl_->local_cache->set(full_key, std::move(*encrypted));
    }
    
    return {};
}

std::expected<void, CacheError> CacheClient::del(std::string_view key) {
    auto full_key = build_key(key);
    
    // Try remote cache
    if (impl_->check_connection()) {
        // TODO: Implement actual gRPC call
    }
    
    // Delete from local cache
    if (impl_->local_cache) {
        impl_->local_cache->del(full_key);
    }
    
    return {};
}

bool CacheClient::exists(std::string_view key) {
    auto full_key = build_key(key);
    
    // Check local cache
    if (impl_->local_cache && impl_->local_cache->exists(full_key)) {
        return true;
    }
    
    // TODO: Check remote cache
    return false;
}

std::expected<std::map<std::string, std::vector<uint8_t>>, CacheError>
CacheClient::batch_get(const std::vector<std::string>& keys) {
    std::map<std::string, std::vector<uint8_t>> results;
    
    for (const auto& key : keys) {
        auto result = get(key);
        if (result) {
            results[key] = std::move(*result);
        }
    }
    
    return results;
}

std::expected<void, CacheError> CacheClient::batch_set(
    const std::map<std::string, std::vector<uint8_t>>& entries,
    std::optional<std::chrono::seconds> ttl) {
    
    for (const auto& [key, value] : entries) {
        auto result = set(key, value, ttl);
        if (!result) {
            return result;
        }
    }
    
    return {};
}

std::expected<void, CacheError> CacheClient::batch_del(const std::vector<std::string>& keys) {
    for (const auto& key : keys) {
        auto result = del(key);
        if (!result) {
            return result;
        }
    }
    
    return {};
}

bool CacheClient::is_connected() const {
    return impl_->connected;
}

void CacheClient::clear_local_cache() {
    if (impl_->local_cache) {
        impl_->local_cache->clear();
    }
}

size_t CacheClient::local_cache_hits() const {
    return impl_->local_hits;
}

size_t CacheClient::local_cache_misses() const {
    return impl_->local_misses;
}

// ============================================================================
// KeyCacheHelper Implementation
// ============================================================================

KeyCacheHelper::KeyCacheHelper(CacheClient& client) : client_(client) {}

std::expected<void, CacheError> KeyCacheHelper::cache_key(
    std::string_view key_id,
    std::span<const uint8_t> key_material,
    std::chrono::seconds ttl) {
    
    auto cache_key = std::format("{}{}", KEY_PREFIX, key_id);
    return client_.set(cache_key, key_material, ttl);
}

std::expected<std::vector<uint8_t>, CacheError> KeyCacheHelper::get_key(
    std::string_view key_id) {
    
    auto cache_key = std::format("{}{}", KEY_PREFIX, key_id);
    return client_.get(cache_key);
}

std::expected<void, CacheError> KeyCacheHelper::invalidate_key(
    std::string_view key_id) {
    
    auto cache_key = std::format("{}{}", KEY_PREFIX, key_id);
    return client_.del(cache_key);
}

std::expected<void, CacheError> KeyCacheHelper::invalidate_key_versions(
    std::string_view key_id_prefix) {
    
    // For now, just invalidate the base key
    // In production, would need to scan for all versions
    return invalidate_key(key_id_prefix);
}

} // namespace crypto
