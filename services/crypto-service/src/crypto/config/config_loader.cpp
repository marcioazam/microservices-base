/**
 * @file config_loader.cpp
 * @brief Configuration loading implementation
 * 
 * Requirements: 8.1, 8.2, 8.3, 8.4, 8.5
 */

#include "crypto/config/config_loader.h"
#include <cstdlib>
#include <fstream>

namespace crypto {

std::string ConfigLoader::getEnv(const std::string& name,
                                  const std::string& default_value) {
    const char* value = std::getenv(name.c_str());
    return value ? std::string(value) : default_value;
}

Result<std::string> ConfigLoader::getRequiredEnv(const std::string& name) {
    const char* value = std::getenv(name.c_str());
    if (!value || std::string(value).empty()) {
        return Err<std::string>(ErrorCode::CONFIGURATION_ERROR,
                                "Required environment variable not set: " + name);
    }
    return Ok(std::string(value));
}

Result<CryptoServiceConfig> ConfigLoader::loadFromEnvironment() {
    CryptoServiceConfig config;
    
    loadServerConfig(config.server);
    loadKeysConfig(config.keys);
    loadLoggingClientConfig(config.logging_client);
    loadCacheClientConfig(config.cache_client);
    loadPerformanceConfig(config.performance);
    loadJWTConfig(config.jwt);
    
    auto validation = validate(config);
    if (!validation) {
        return Err<CryptoServiceConfig>(validation.error());
    }
    
    return Ok(std::move(config));
}

void ConfigLoader::loadServerConfig(CryptoServiceConfig::Server& server) {
    auto grpc_port = getEnv(EnvVars::GRPC_PORT, "50051");
    server.grpc_port = static_cast<uint16_t>(std::stoi(grpc_port));
    
    auto rest_port = getEnv(EnvVars::REST_PORT, "8080");
    server.rest_port = static_cast<uint16_t>(std::stoi(rest_port));
    
    server.tls_cert_path = getEnv(EnvVars::TLS_CERT_PATH);
    server.tls_key_path = getEnv(EnvVars::TLS_KEY_PATH);
    server.tls_ca_path = getEnv(EnvVars::TLS_CA_PATH);
    
    auto thread_pool = getEnv(EnvVars::THREAD_POOL_SIZE, "4");
    server.thread_pool_size = std::stoul(thread_pool);
}

void ConfigLoader::loadKeysConfig(CryptoServiceConfig::Keys& keys) {
    keys.kms_provider = getEnv(EnvVars::KMS_PROVIDER, "local");
    keys.hsm_slot_id = getEnv(EnvVars::HSM_SLOT_ID);
    keys.aws_kms_key_arn = getEnv(EnvVars::AWS_KMS_KEY_ARN);
    keys.aws_region = getEnv(EnvVars::AWS_REGION, "us-east-1");
    keys.azure_kv_url = getEnv(EnvVars::AZURE_KV_URL);
    keys.local_key_path = getEnv(EnvVars::LOCAL_KEY_PATH, 
                                  "/var/lib/crypto-service/keys");
    
    auto cache_ttl = getEnv(EnvVars::KEY_CACHE_TTL, "300");
    keys.key_cache_ttl = std::chrono::seconds(std::stoi(cache_ttl));
    
    auto cache_size = getEnv(EnvVars::KEY_CACHE_MAX_SIZE, "1000");
    keys.key_cache_max_size = std::stoul(cache_size);
}

void ConfigLoader::loadLoggingClientConfig(LoggingClientConfig& config) {
    config.address = getEnv(EnvVars::LOGGING_SERVICE_ADDRESS, "localhost:5001");
    config.service_id = "crypto-service";
    
    auto batch_size = getEnv(EnvVars::LOGGING_BATCH_SIZE, "100");
    config.batch_size = std::stoul(batch_size);
    
    auto flush_interval = getEnv(EnvVars::LOGGING_FLUSH_INTERVAL_MS, "5000");
    config.flush_interval = std::chrono::milliseconds(std::stoi(flush_interval));
    
    auto min_level = getEnv(EnvVars::LOGGING_MIN_LEVEL, "INFO");
    if (min_level == "DEBUG") config.min_level = LogLevel::DEBUG;
    else if (min_level == "INFO") config.min_level = LogLevel::INFO;
    else if (min_level == "WARN") config.min_level = LogLevel::WARN;
    else if (min_level == "ERROR") config.min_level = LogLevel::ERROR;
    else if (min_level == "FATAL") config.min_level = LogLevel::FATAL;
    
    auto fallback = getEnv(EnvVars::LOGGING_FALLBACK_ENABLED, "true");
    config.fallback_enabled = (fallback == "true" || fallback == "1");
}

void ConfigLoader::loadCacheClientConfig(CacheClientConfig& config) {
    config.address = getEnv(EnvVars::CACHE_SERVICE_ADDRESS, "localhost:50051");
    config.namespace_prefix = getEnv(EnvVars::CACHE_NAMESPACE, "crypto");
    
    auto default_ttl = getEnv(EnvVars::CACHE_DEFAULT_TTL, "300");
    config.default_ttl = std::chrono::seconds(std::stoi(default_ttl));
    
    auto local_fallback = getEnv(EnvVars::CACHE_LOCAL_FALLBACK, "true");
    config.local_fallback_enabled = (local_fallback == "true" || local_fallback == "1");
    
    auto local_size = getEnv(EnvVars::CACHE_LOCAL_SIZE, "1000");
    config.local_cache_size = std::stoul(local_size);
}

void ConfigLoader::loadPerformanceConfig(CryptoServiceConfig::Performance& perf) {
    // Use defaults
}

void ConfigLoader::loadJWTConfig(CryptoServiceConfig::JWT& jwt) {
    jwt.public_key_path = getEnv(EnvVars::JWT_PUBLIC_KEY_PATH);
    jwt.jwks_url = getEnv(EnvVars::JWT_JWKS_URL);
    jwt.expected_issuer = getEnv(EnvVars::JWT_ISSUER);
    jwt.expected_audience = getEnv(EnvVars::JWT_AUDIENCE);
}

Result<CryptoServiceConfig> ConfigLoader::loadFromFile(const std::string& path) {
    std::ifstream file(path);
    if (!file.is_open()) {
        return Err<CryptoServiceConfig>(ErrorCode::CONFIGURATION_ERROR,
                                        "Cannot open config file: " + path);
    }
    
    std::string line;
    while (std::getline(file, line)) {
        if (line.empty() || line[0] == '#') continue;
        
        auto pos = line.find('=');
        if (pos == std::string::npos) continue;
        
        std::string key = line.substr(0, pos);
        std::string value = line.substr(pos + 1);
        
        while (!key.empty() && std::isspace(key.back())) key.pop_back();
        while (!value.empty() && std::isspace(value.front())) value.erase(0, 1);
        
        #ifdef _WIN32
        _putenv_s(key.c_str(), value.c_str());
        #else
        setenv(key.c_str(), value.c_str(), 1);
        #endif
    }
    
    return loadFromEnvironment();
}

Result<void> ConfigLoader::validate(const CryptoServiceConfig& config) {
    // Server validation
    if (config.server.grpc_port == 0) {
        return Err<void>(ErrorCode::CONFIGURATION_ERROR, "Invalid gRPC port");
    }
    if (config.server.rest_port == 0) {
        return Err<void>(ErrorCode::CONFIGURATION_ERROR, "Invalid REST port");
    }
    if (config.server.grpc_port == config.server.rest_port) {
        return Err<void>(ErrorCode::CONFIGURATION_ERROR, 
                         "gRPC and REST ports must be different");
    }
    
    // KMS validation
    const auto& kms = config.keys.kms_provider;
    if (kms != "local" && kms != "hsm" && kms != "aws_kms" && kms != "azure_kv") {
        return Err<void>(ErrorCode::CONFIGURATION_ERROR,
                         "Invalid KMS provider: " + kms);
    }
    
    if (kms == "hsm" && config.keys.hsm_slot_id.empty()) {
        return Err<void>(ErrorCode::CONFIGURATION_ERROR,
                         "HSM slot ID required for HSM provider");
    }
    if (kms == "aws_kms" && config.keys.aws_kms_key_arn.empty()) {
        return Err<void>(ErrorCode::CONFIGURATION_ERROR,
                         "AWS KMS key ARN required for AWS KMS provider");
    }
    if (kms == "azure_kv" && config.keys.azure_kv_url.empty()) {
        return Err<void>(ErrorCode::CONFIGURATION_ERROR,
                         "Azure Key Vault URL required for Azure KV provider");
    }
    
    // Logging client validation
    if (config.logging_client.address.empty()) {
        return Err<void>(ErrorCode::CONFIGURATION_ERROR,
                         "Logging service address is required");
    }
    if (config.logging_client.batch_size == 0) {
        return Err<void>(ErrorCode::CONFIGURATION_ERROR,
                         "Logging batch size must be > 0");
    }
    
    // Cache client validation
    if (config.cache_client.address.empty()) {
        return Err<void>(ErrorCode::CONFIGURATION_ERROR,
                         "Cache service address is required");
    }
    
    return Ok();
}

} // namespace crypto
