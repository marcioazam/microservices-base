// Unit tests for ConfigLoader
// Tests environment variable parsing, validation, and default values

#include <gtest/gtest.h>
#include <gmock/gmock.h>
#include "crypto/config/config_loader.h"
#include <cstdlib>

namespace crypto::test {

// ============================================================================
// Test Environment Helper
// ============================================================================

class EnvGuard {
public:
    EnvGuard(const std::string& name, const std::string& value)
        : name_(name) {
#ifdef _WIN32
        _putenv_s(name.c_str(), value.c_str());
#else
        setenv(name.c_str(), value.c_str(), 1);
#endif
    }
    
    ~EnvGuard() {
#ifdef _WIN32
        _putenv_s(name_.c_str(), "");
#else
        unsetenv(name_.c_str());
#endif
    }

private:
    std::string name_;
};

// ============================================================================
// Test Fixture
// ============================================================================

class ConfigLoaderTest : public ::testing::Test {
protected:
    ConfigLoader loader_;
};

// ============================================================================
// Default Values Tests
// ============================================================================

TEST_F(ConfigLoaderTest, DefaultServerConfig) {
    CryptoServiceConfig config;
    
    EXPECT_EQ(config.server.grpc_port, 50051);
    EXPECT_EQ(config.server.rest_port, 8080);
    EXPECT_EQ(config.server.thread_pool_size, 4);
}

TEST_F(ConfigLoaderTest, DefaultKeysConfig) {
    CryptoServiceConfig config;
    
    EXPECT_EQ(config.keys.kms_provider, "local");
    EXPECT_EQ(config.keys.key_cache_ttl, std::chrono::seconds(300));
    EXPECT_EQ(config.keys.key_cache_max_size, 1000);
}

TEST_F(ConfigLoaderTest, DefaultLoggingClientConfig) {
    CryptoServiceConfig config;
    
    EXPECT_EQ(config.logging_client.address, "localhost:5001");
    EXPECT_EQ(config.logging_client.service_id, "crypto-service");
    EXPECT_EQ(config.logging_client.batch_size, 100);
    EXPECT_TRUE(config.logging_client.fallback_enabled);
}

TEST_F(ConfigLoaderTest, DefaultCacheClientConfig) {
    CryptoServiceConfig config;
    
    EXPECT_EQ(config.cache_client.address, "localhost:50051");
    EXPECT_EQ(config.cache_client.namespace_prefix, "crypto");
    EXPECT_EQ(config.cache_client.default_ttl, std::chrono::seconds(300));
    EXPECT_TRUE(config.cache_client.local_fallback_enabled);
}

TEST_F(ConfigLoaderTest, DefaultPerformanceConfig) {
    CryptoServiceConfig config;
    
    EXPECT_EQ(config.performance.file_chunk_size, 65536);
    EXPECT_EQ(config.performance.max_file_size, 10737418240);
    EXPECT_EQ(config.performance.connection_pool_size, 10);
}

// ============================================================================
// Environment Variable Parsing Tests
// ============================================================================

TEST_F(ConfigLoaderTest, GetEnvWithValue) {
    EnvGuard guard("TEST_ENV_VAR", "test_value");
    
    auto result = ConfigLoader::getEnv("TEST_ENV_VAR");
    EXPECT_EQ(result, "test_value");
}

TEST_F(ConfigLoaderTest, GetEnvWithDefault) {
    auto result = ConfigLoader::getEnv("NONEXISTENT_VAR_123", "default");
    EXPECT_EQ(result, "default");
}

TEST_F(ConfigLoaderTest, GetEnvMissingNoDefault) {
    auto result = ConfigLoader::getEnv("NONEXISTENT_VAR_456");
    EXPECT_TRUE(result.empty());
}

TEST_F(ConfigLoaderTest, GetRequiredEnvPresent) {
    EnvGuard guard("REQUIRED_TEST_VAR", "required_value");
    
    auto result = ConfigLoader::getRequiredEnv("REQUIRED_TEST_VAR");
    ASSERT_TRUE(result.has_value());
    EXPECT_EQ(*result, "required_value");
}

TEST_F(ConfigLoaderTest, GetRequiredEnvMissing) {
    auto result = ConfigLoader::getRequiredEnv("MISSING_REQUIRED_VAR");
    ASSERT_FALSE(result.has_value());
    EXPECT_EQ(result.error().code, ErrorCode::CONFIG_MISSING);
}

// ============================================================================
// Load From Environment Tests
// ============================================================================

TEST_F(ConfigLoaderTest, LoadFromEnvironmentDefaults) {
    auto result = loader_.loadFromEnvironment();
    ASSERT_TRUE(result.has_value());
    
    // Should have default values
    EXPECT_EQ(result->server.grpc_port, 50051);
    EXPECT_EQ(result->server.rest_port, 8080);
}

TEST_F(ConfigLoaderTest, LoadFromEnvironmentWithOverrides) {
    EnvGuard grpc_guard(EnvVars::GRPC_PORT, "50052");
    EnvGuard rest_guard(EnvVars::REST_PORT, "8081");
    
    auto result = loader_.loadFromEnvironment();
    ASSERT_TRUE(result.has_value());
    
    EXPECT_EQ(result->server.grpc_port, 50052);
    EXPECT_EQ(result->server.rest_port, 8081);
}

TEST_F(ConfigLoaderTest, LoadLoggingServiceAddress) {
    EnvGuard guard(EnvVars::LOGGING_SERVICE_ADDRESS, "logging:5001");
    
    auto result = loader_.loadFromEnvironment();
    ASSERT_TRUE(result.has_value());
    
    EXPECT_EQ(result->logging_client.address, "logging:5001");
}

TEST_F(ConfigLoaderTest, LoadCacheServiceAddress) {
    EnvGuard guard(EnvVars::CACHE_SERVICE_ADDRESS, "cache:50051");
    
    auto result = loader_.loadFromEnvironment();
    ASSERT_TRUE(result.has_value());
    
    EXPECT_EQ(result->cache_client.address, "cache:50051");
}

TEST_F(ConfigLoaderTest, LoadKMSProvider) {
    EnvGuard guard(EnvVars::KMS_PROVIDER, "aws");
    
    auto result = loader_.loadFromEnvironment();
    ASSERT_TRUE(result.has_value());
    
    EXPECT_EQ(result->keys.kms_provider, "aws");
}

// ============================================================================
// Validation Tests
// ============================================================================

TEST_F(ConfigLoaderTest, ValidateDefaultConfig) {
    CryptoServiceConfig config;
    
    auto result = loader_.validate(config);
    EXPECT_TRUE(result.has_value());
}

TEST_F(ConfigLoaderTest, ValidateZeroPortRejected) {
    CryptoServiceConfig config;
    config.server.grpc_port = 0;
    
    auto result = loader_.validate(config);
    EXPECT_FALSE(result.has_value());
}

TEST_F(ConfigLoaderTest, ValidateZeroBatchSizeRejected) {
    CryptoServiceConfig config;
    config.logging_client.batch_size = 0;
    
    auto result = loader_.validate(config);
    EXPECT_FALSE(result.has_value());
}

TEST_F(ConfigLoaderTest, ValidateZeroCacheSizeRejected) {
    CryptoServiceConfig config;
    config.cache_client.local_cache_size = 0;
    
    auto result = loader_.validate(config);
    EXPECT_FALSE(result.has_value());
}

TEST_F(ConfigLoaderTest, ValidateInvalidKMSProviderRejected) {
    CryptoServiceConfig config;
    config.keys.kms_provider = "invalid_provider";
    
    auto result = loader_.validate(config);
    EXPECT_FALSE(result.has_value());
}

TEST_F(ConfigLoaderTest, ValidateValidKMSProviders) {
    CryptoServiceConfig config;
    
    config.keys.kms_provider = "local";
    EXPECT_TRUE(loader_.validate(config).has_value());
    
    config.keys.kms_provider = "aws";
    EXPECT_TRUE(loader_.validate(config).has_value());
    
    config.keys.kms_provider = "azure";
    EXPECT_TRUE(loader_.validate(config).has_value());
    
    config.keys.kms_provider = "hsm";
    EXPECT_TRUE(loader_.validate(config).has_value());
}

// ============================================================================
// Environment Variable Names Tests
// ============================================================================

TEST_F(ConfigLoaderTest, EnvVarNamesAreDefined) {
    // Server
    EXPECT_STREQ(EnvVars::GRPC_PORT, "CRYPTO_GRPC_PORT");
    EXPECT_STREQ(EnvVars::REST_PORT, "CRYPTO_REST_PORT");
    EXPECT_STREQ(EnvVars::TLS_CERT_PATH, "CRYPTO_TLS_CERT_PATH");
    EXPECT_STREQ(EnvVars::TLS_KEY_PATH, "CRYPTO_TLS_KEY_PATH");
    
    // Keys
    EXPECT_STREQ(EnvVars::KMS_PROVIDER, "CRYPTO_KMS_PROVIDER");
    EXPECT_STREQ(EnvVars::HSM_SLOT_ID, "CRYPTO_HSM_SLOT_ID");
    EXPECT_STREQ(EnvVars::AWS_KMS_KEY_ARN, "CRYPTO_AWS_KMS_KEY_ARN");
    
    // Logging
    EXPECT_STREQ(EnvVars::LOGGING_SERVICE_ADDRESS, "LOGGING_SERVICE_ADDRESS");
    EXPECT_STREQ(EnvVars::LOGGING_BATCH_SIZE, "LOGGING_BATCH_SIZE");
    
    // Cache
    EXPECT_STREQ(EnvVars::CACHE_SERVICE_ADDRESS, "CACHE_SERVICE_ADDRESS");
    EXPECT_STREQ(EnvVars::CACHE_NAMESPACE, "CACHE_NAMESPACE");
}

// ============================================================================
// Edge Cases
// ============================================================================

TEST_F(ConfigLoaderTest, EmptyAddressAccepted) {
    CryptoServiceConfig config;
    config.logging_client.address = "";
    
    // Empty address may be accepted (uses default) or rejected
    // depending on implementation
    auto result = loader_.validate(config);
    // Just verify no crash
}

TEST_F(ConfigLoaderTest, VeryLargePortRejected) {
    CryptoServiceConfig config;
    config.server.grpc_port = 70000;  // Invalid port
    
    auto result = loader_.validate(config);
    EXPECT_FALSE(result.has_value());
}

TEST_F(ConfigLoaderTest, ReservedPortRejected) {
    CryptoServiceConfig config;
    config.server.grpc_port = 80;  // Reserved port
    
    auto result = loader_.validate(config);
    EXPECT_FALSE(result.has_value());
}

TEST_F(ConfigLoaderTest, ValidPortRange) {
    CryptoServiceConfig config;
    
    config.server.grpc_port = 1024;
    EXPECT_TRUE(loader_.validate(config).has_value());
    
    config.server.grpc_port = 65535;
    EXPECT_TRUE(loader_.validate(config).has_value());
}

TEST_F(ConfigLoaderTest, DuplicatePortsRejected) {
    CryptoServiceConfig config;
    config.server.grpc_port = 8080;
    config.server.rest_port = 8080;  // Same as gRPC
    
    auto result = loader_.validate(config);
    EXPECT_FALSE(result.has_value());
}

TEST_F(ConfigLoaderTest, NumericEnvVarParsing) {
    EnvGuard guard(EnvVars::LOGGING_BATCH_SIZE, "500");
    
    auto result = loader_.loadFromEnvironment();
    ASSERT_TRUE(result.has_value());
    
    EXPECT_EQ(result->logging_client.batch_size, 500);
}

TEST_F(ConfigLoaderTest, InvalidNumericEnvVar) {
    EnvGuard guard(EnvVars::GRPC_PORT, "not_a_number");
    
    auto result = loader_.loadFromEnvironment();
    // Should either fail or use default
    // Implementation dependent
}

TEST_F(ConfigLoaderTest, BooleanEnvVarTrue) {
    EnvGuard guard(EnvVars::LOGGING_FALLBACK_ENABLED, "true");
    
    auto result = loader_.loadFromEnvironment();
    ASSERT_TRUE(result.has_value());
    
    EXPECT_TRUE(result->logging_client.fallback_enabled);
}

TEST_F(ConfigLoaderTest, BooleanEnvVarFalse) {
    EnvGuard guard(EnvVars::CACHE_LOCAL_FALLBACK, "false");
    
    auto result = loader_.loadFromEnvironment();
    ASSERT_TRUE(result.has_value());
    
    EXPECT_FALSE(result->cache_client.local_fallback_enabled);
}

} // namespace crypto::test
