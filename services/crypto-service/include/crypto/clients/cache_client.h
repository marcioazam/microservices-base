#pragma once

/**
 * @file cache_client.h
 * @brief gRPC client for centralized cache-service integration
 * 
 * This client provides cache operations via the platform cache-service
 * with namespace isolation, local fallback, and optional encryption.
 * 
 * Requirements: 2.1, 2.6
 */

#include "crypto/common/result.h"
#include <string>
#include <string_view>
#include <vector>
#include <map>
#include <memory>
#include <chrono>
#include <optional>
#include <span>
#include <array>

namespace crypto {

// ============================================================================
// Cache Errors
// ============================================================================

/**
 * @brief Cache-specific error codes
 */
enum class CacheErrorCode {
    OK = 0,
    NOT_FOUND,
    CONNECTION_ERROR,
    TIMEOUT,
    SERIALIZATION_ERROR,
    ENCRYPTION_ERROR,
    INVALID_KEY,
    QUOTA_EXCEEDED
};

/**
 * @brief Cache operation error
 */
struct CacheError {
    CacheErrorCode code;
    std::string message;
    
    CacheError(CacheErrorCode c, std::string msg = "")
        : code(c), message(std::move(msg)) {}
};

// ============================================================================
// Configuration
// ============================================================================

/**
 * @brief Configuration for CacheClient
 */
struct CacheClientConfig {
    /// gRPC address of cache-service (host:port)
    std::string address = "localhost:50051";
    
    /// Namespace prefix for all keys (isolation)
    std::string namespace_prefix = "crypto";
    
    /// Default TTL for cache entries
    std::chrono::seconds default_ttl{300};
    
    /// Optional AES-256 encryption key for cached values
    std::optional<std::array<uint8_t, 32>> encryption_key;
    
    /// Enable local LRU cache fallback
    bool local_fallback_enabled = true;
    
    /// Maximum entries in local fallback cache
    size_t local_cache_size = 1000;
    
    /// Connection timeout
    std::chrono::milliseconds connect_timeout{5000};
    
    /// Request timeout
    std::chrono::milliseconds request_timeout{1000};
    
    /// JWT token for authentication (if required)
    std::string auth_token;
};

// ============================================================================
// CacheClient
// ============================================================================

/**
 * @brief gRPC client for centralized caching
 * 
 * Features:
 * - Namespace-isolated key operations
 * - Optional value encryption (AES-256-GCM)
 * - Local LRU cache fallback when service unavailable
 * - Batch operations for efficiency
 * 
 * Usage:
 *   CacheClientConfig config;
 *   config.address = "cache-service:50051";
 *   config.namespace_prefix = "crypto";
 *   CacheClient cache(config);
 *   
 *   cache.set("key:123", key_data, std::chrono::minutes(5));
 *   auto result = cache.get("key:123");
 */
class CacheClient {
public:
    /**
     * @brief Construct cache client with configuration
     * @param config Client configuration
     */
    explicit CacheClient(const CacheClientConfig& config);
    
    /**
     * @brief Destructor
     */
    ~CacheClient();
    
    // Non-copyable, movable
    CacheClient(const CacheClient&) = delete;
    CacheClient& operator=(const CacheClient&) = delete;
    CacheClient(CacheClient&&) noexcept;
    CacheClient& operator=(CacheClient&&) noexcept;
    
    // ========================================================================
    // Basic Operations
    // ========================================================================
    
    /**
     * @brief Get value from cache
     * @param key Cache key (will be prefixed with namespace)
     * @return Value bytes or error
     */
    [[nodiscard]] std::expected<std::vector<uint8_t>, CacheError> 
        get(std::string_view key);
    
    /**
     * @brief Set value in cache
     * @param key Cache key (will be prefixed with namespace)
     * @param value Value bytes to cache
     * @param ttl Optional TTL (uses default if not specified)
     * @return Success or error
     */
    [[nodiscard]] std::expected<void, CacheError> 
        set(std::string_view key,
            std::span<const uint8_t> value,
            std::optional<std::chrono::seconds> ttl = std::nullopt);
    
    /**
     * @brief Delete value from cache
     * @param key Cache key (will be prefixed with namespace)
     * @return Success or error
     */
    [[nodiscard]] std::expected<void, CacheError> 
        del(std::string_view key);
    
    /**
     * @brief Check if key exists in cache
     * @param key Cache key
     * @return true if exists
     */
    [[nodiscard]] bool exists(std::string_view key);
    
    // ========================================================================
    // Batch Operations
    // ========================================================================
    
    /**
     * @brief Get multiple values from cache
     * @param keys List of cache keys
     * @return Map of key to value (missing keys not included)
     */
    [[nodiscard]] std::expected<std::map<std::string, std::vector<uint8_t>>, CacheError>
        batch_get(const std::vector<std::string>& keys);
    
    /**
     * @brief Set multiple values in cache
     * @param entries Map of key to value
     * @param ttl Optional TTL for all entries
     * @return Success or error
     */
    [[nodiscard]] std::expected<void, CacheError>
        batch_set(const std::map<std::string, std::vector<uint8_t>>& entries,
                  std::optional<std::chrono::seconds> ttl = std::nullopt);
    
    /**
     * @brief Delete multiple values from cache
     * @param keys List of cache keys
     * @return Success or error
     */
    [[nodiscard]] std::expected<void, CacheError>
        batch_del(const std::vector<std::string>& keys);
    
    // ========================================================================
    // Control Methods
    // ========================================================================
    
    /**
     * @brief Check if connected to cache service
     * @return true if connected and healthy
     */
    [[nodiscard]] bool is_connected() const;
    
    /**
     * @brief Clear local fallback cache
     */
    void clear_local_cache();
    
    /**
     * @brief Get local cache hit count
     */
    [[nodiscard]] size_t local_cache_hits() const;
    
    /**
     * @brief Get local cache miss count
     */
    [[nodiscard]] size_t local_cache_misses() const;

private:
    struct Impl;
    std::unique_ptr<Impl> impl_;
    
    /// Build full key with namespace prefix
    [[nodiscard]] std::string build_key(std::string_view key) const;
};

// ============================================================================
// Key Cache Helper
// ============================================================================

/**
 * @brief Specialized cache helper for cryptographic keys
 * 
 * Provides type-safe caching for key material with automatic
 * serialization and secure memory handling.
 */
class KeyCacheHelper {
public:
    explicit KeyCacheHelper(CacheClient& client);
    
    /**
     * @brief Cache a key
     * @param key_id Key identifier
     * @param key_material Key bytes
     * @param ttl Cache TTL
     */
    [[nodiscard]] std::expected<void, CacheError>
        cache_key(std::string_view key_id,
                  std::span<const uint8_t> key_material,
                  std::chrono::seconds ttl);
    
    /**
     * @brief Get cached key
     * @param key_id Key identifier
     * @return Key bytes or error
     */
    [[nodiscard]] std::expected<std::vector<uint8_t>, CacheError>
        get_key(std::string_view key_id);
    
    /**
     * @brief Invalidate cached key
     * @param key_id Key identifier
     */
    [[nodiscard]] std::expected<void, CacheError>
        invalidate_key(std::string_view key_id);
    
    /**
     * @brief Invalidate all versions of a key
     * @param key_id_prefix Key ID prefix (without version)
     */
    [[nodiscard]] std::expected<void, CacheError>
        invalidate_key_versions(std::string_view key_id_prefix);

private:
    CacheClient& client_;
    
    static constexpr std::string_view KEY_PREFIX = "key:";
};

} // namespace crypto
