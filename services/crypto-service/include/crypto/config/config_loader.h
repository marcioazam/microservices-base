#pragma once

/**
 * @file config_loader.h
 * @brief Configuration loading with platform service integration
 * 
 * Requirements: 8.1, 8.2, 8.3, 8.4, 8.5
 */

#include "crypto/common/result.h"
#include "crypto/clients/logging_client.h"
#include "crypto/clients/cache_client.h"
#include <string>
#include <chrono>
#include <optional>
#include <map>

namespace crypto {

// Service configuration
struct CryptoServiceConfig {
    // Server configuration
    struct Server {
        uint16_t grpc_port = 50051;
        uint16_t rest_port = 8080;
        std::string tls_cert_path;
        std::string tls_key_path;
        std::string tls_ca_path;
        size_t thread_pool_size = 4;
    } server;
    
    // Key management configuration
    struct Keys {
        std::string kms_provider = "local";
        std::string hsm_slot_id;
        std::string aws_kms_key_arn;
        std::string aws_region;
        std::string azure_kv_url;
        std::string local_key_path = "/var/lib/crypto-service/keys";
        std::chrono::seconds key_cache_ttl{300};
        size_t key_cache_max_size = 1000;
    } keys;
    
    // Platform logging client configuration
    LoggingClientConfig logging_client;
    
    // Platform cache client configuration  
    CacheClientConfig cache_client;
    
    // Performance configuration
    struct Performance {
        size_t file_chunk_size = 65536;
        size_t max_file_size = 10737418240;
        size_t connection_pool_size = 10;
    } performance;
    
    // JWT configuration
    struct JWT {
        std::string public_key_path;
        std::string jwks_url;
        std::string expected_issuer;
        std::string expected_audience;
    } jwt;
};

// Configuration loader
class ConfigLoader {
public:
    ConfigLoader() = default;
    ~ConfigLoader() = default;
    
    [[nodiscard]] Result<CryptoServiceConfig> loadFromEnvironment();
    [[nodiscard]] Result<CryptoServiceConfig> loadFromFile(const std::string& path);
    [[nodiscard]] Result<void> validate(const CryptoServiceConfig& config);
    
    [[nodiscard]] static std::string getEnv(const std::string& name, 
                                            const std::string& default_value = "");
    [[nodiscard]] static Result<std::string> getRequiredEnv(const std::string& name);

private:
    void loadServerConfig(CryptoServiceConfig::Server& server);
    void loadKeysConfig(CryptoServiceConfig::Keys& keys);
    void loadLoggingClientConfig(LoggingClientConfig& config);
    void loadCacheClientConfig(CacheClientConfig& config);
    void loadPerformanceConfig(CryptoServiceConfig::Performance& perf);
    void loadJWTConfig(CryptoServiceConfig::JWT& jwt);
};

// Environment variable names
namespace EnvVars {
    // Server
    constexpr const char* GRPC_PORT = "CRYPTO_GRPC_PORT";
    constexpr const char* REST_PORT = "CRYPTO_REST_PORT";
    constexpr const char* TLS_CERT_PATH = "CRYPTO_TLS_CERT_PATH";
    constexpr const char* TLS_KEY_PATH = "CRYPTO_TLS_KEY_PATH";
    constexpr const char* TLS_CA_PATH = "CRYPTO_TLS_CA_PATH";
    constexpr const char* THREAD_POOL_SIZE = "CRYPTO_THREAD_POOL_SIZE";
    
    // Keys
    constexpr const char* KMS_PROVIDER = "CRYPTO_KMS_PROVIDER";
    constexpr const char* HSM_SLOT_ID = "CRYPTO_HSM_SLOT_ID";
    constexpr const char* AWS_KMS_KEY_ARN = "CRYPTO_AWS_KMS_KEY_ARN";
    constexpr const char* AWS_REGION = "AWS_REGION";
    constexpr const char* AZURE_KV_URL = "CRYPTO_AZURE_KV_URL";
    constexpr const char* LOCAL_KEY_PATH = "CRYPTO_LOCAL_KEY_PATH";
    constexpr const char* KEY_CACHE_TTL = "CRYPTO_KEY_CACHE_TTL";
    constexpr const char* KEY_CACHE_MAX_SIZE = "CRYPTO_KEY_CACHE_MAX_SIZE";
    
    // Logging Client
    constexpr const char* LOGGING_SERVICE_ADDRESS = "LOGGING_SERVICE_ADDRESS";
    constexpr const char* LOGGING_BATCH_SIZE = "LOGGING_BATCH_SIZE";
    constexpr const char* LOGGING_FLUSH_INTERVAL_MS = "LOGGING_FLUSH_INTERVAL_MS";
    constexpr const char* LOGGING_MIN_LEVEL = "LOGGING_MIN_LEVEL";
    constexpr const char* LOGGING_FALLBACK_ENABLED = "LOGGING_FALLBACK_ENABLED";
    
    // Cache Client
    constexpr const char* CACHE_SERVICE_ADDRESS = "CACHE_SERVICE_ADDRESS";
    constexpr const char* CACHE_NAMESPACE = "CACHE_NAMESPACE";
    constexpr const char* CACHE_DEFAULT_TTL = "CACHE_DEFAULT_TTL";
    constexpr const char* CACHE_LOCAL_FALLBACK = "CACHE_LOCAL_FALLBACK";
    constexpr const char* CACHE_LOCAL_SIZE = "CACHE_LOCAL_SIZE";
    
    // JWT
    constexpr const char* JWT_PUBLIC_KEY_PATH = "CRYPTO_JWT_PUBLIC_KEY_PATH";
    constexpr const char* JWT_JWKS_URL = "CRYPTO_JWT_JWKS_URL";
    constexpr const char* JWT_ISSUER = "CRYPTO_JWT_ISSUER";
    constexpr const char* JWT_AUDIENCE = "CRYPTO_JWT_AUDIENCE";
}

} // namespace crypto
