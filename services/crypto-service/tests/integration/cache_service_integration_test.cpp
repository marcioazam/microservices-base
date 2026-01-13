/**
 * @file cache_service_integration_test.cpp
 * @brief Integration tests for CacheClient with real cache-service
 * 
 * Requirements: 7.4
 */

#include <gtest/gtest.h>
#include "crypto/clients/cache_client.h"
#include <thread>
#include <chrono>

namespace crypto::test {

/**
 * @brief Integration test fixture for CacheClient
 * 
 * Note: These tests require a running cache-service instance.
 * In CI, use Testcontainers to spin up the service.
 */
class CacheServiceIntegrationTest : public ::testing::Test {
protected:
    void SetUp() override {
        // Get cache service address from environment or use default
        const char* addr = std::getenv("CACHE_SERVICE_ADDRESS");
        config_.address = addr ? addr : "localhost:50051";
        config_.namespace_prefix = "crypto-test";
        config_.default_ttl = std::chrono::seconds{60};
        config_.local_fallback_enabled = true;
        config_.local_cache_size = 100;
    }

    CacheClientConfig config_;
};

TEST_F(CacheServiceIntegrationTest, DISABLED_ConnectsToCacheService) {
    // Disabled by default - enable when cache-service is available
    CacheClient client(config_);
    
    // Give time for connection
    std::this_thread::sleep_for(std::chrono::milliseconds{500});
    
    EXPECT_TRUE(client.is_connected());
}

TEST_F(CacheServiceIntegrationTest, DISABLED_SetAndGetValue) {
    CacheClient client(config_);
    std::this_thread::sleep_for(std::chrono::milliseconds{500});
    
    if (!client.is_connected()) {
        GTEST_SKIP() << "Cache service not available";
    }
    
    const std::string key = "test-key-1";
    const std::vector<uint8_t> value = {0x01, 0x02, 0x03, 0x04};
    
    // Set value
    auto set_result = client.set(key, value);
    ASSERT_TRUE(set_result.has_value()) << "Set failed";
    
    // Get value
    auto get_result = client.get(key);
    ASSERT_TRUE(get_result.has_value()) << "Get failed";
    EXPECT_EQ(get_result.value(), value);
}

TEST_F(CacheServiceIntegrationTest, DISABLED_DeleteValue) {
    CacheClient client(config_);
    std::this_thread::sleep_for(std::chrono::milliseconds{500});
    
    if (!client.is_connected()) {
        GTEST_SKIP() << "Cache service not available";
    }
    
    const std::string key = "test-key-delete";
    const std::vector<uint8_t> value = {0xAA, 0xBB};
    
    // Set value
    auto set_result = client.set(key, value);
    ASSERT_TRUE(set_result.has_value());
    
    // Delete value
    auto del_result = client.del(key);
    ASSERT_TRUE(del_result.has_value());
    
    // Get should return cache miss
    auto get_result = client.get(key);
    EXPECT_FALSE(get_result.has_value());
    EXPECT_EQ(get_result.error().code, ErrorCode::CACHE_MISS);
}

TEST_F(CacheServiceIntegrationTest, DISABLED_NamespacePrefixing) {
    CacheClient client(config_);
    std::this_thread::sleep_for(std::chrono::milliseconds{500});
    
    if (!client.is_connected()) {
        GTEST_SKIP() << "Cache service not available";
    }
    
    // Keys should be prefixed with namespace
    const std::string key = "namespace-test";
    const std::vector<uint8_t> value = {0x11, 0x22};
    
    auto set_result = client.set(key, value);
    ASSERT_TRUE(set_result.has_value());
    
    // The actual key in cache should be "crypto-test:namespace-test"
    auto get_result = client.get(key);
    ASSERT_TRUE(get_result.has_value());
    EXPECT_EQ(get_result.value(), value);
}

TEST_F(CacheServiceIntegrationTest, DISABLED_TTLExpiration) {
    CacheClient client(config_);
    std::this_thread::sleep_for(std::chrono::milliseconds{500});
    
    if (!client.is_connected()) {
        GTEST_SKIP() << "Cache service not available";
    }
    
    const std::string key = "ttl-test";
    const std::vector<uint8_t> value = {0xFF};
    
    // Set with 1 second TTL
    auto set_result = client.set(key, value, std::chrono::seconds{1});
    ASSERT_TRUE(set_result.has_value());
    
    // Should exist immediately
    auto get_result1 = client.get(key);
    ASSERT_TRUE(get_result1.has_value());
    
    // Wait for expiration
    std::this_thread::sleep_for(std::chrono::seconds{2});
    
    // Should be expired
    auto get_result2 = client.get(key);
    EXPECT_FALSE(get_result2.has_value());
}

TEST_F(CacheServiceIntegrationTest, DISABLED_LocalFallbackWhenDisconnected) {
    config_.address = "invalid:9999";  // Invalid address
    config_.local_fallback_enabled = true;
    
    CacheClient client(config_);
    std::this_thread::sleep_for(std::chrono::milliseconds{500});
    
    // Should not be connected to remote
    EXPECT_FALSE(client.is_connected());
    
    const std::string key = "fallback-test";
    const std::vector<uint8_t> value = {0xDE, 0xAD};
    
    // Should use local fallback
    auto set_result = client.set(key, value);
    EXPECT_TRUE(set_result.has_value());
    
    auto get_result = client.get(key);
    EXPECT_TRUE(get_result.has_value());
    EXPECT_EQ(get_result.value(), value);
}

} // namespace crypto::test
