// Unit tests for CacheClient
// Tests connection, get/set/del operations, and local fallback behavior

#include <gtest/gtest.h>
#include <gmock/gmock.h>
#include "crypto/clients/cache_client.h"
#include <thread>
#include <chrono>

namespace crypto::test {

// ============================================================================
// Test Fixture
// ============================================================================

class CacheClientTest : public ::testing::Test {
protected:
    void SetUp() override {
        config_.address = "localhost:50051";
        config_.namespace_prefix = "crypto-test";
        config_.default_ttl = std::chrono::seconds(300);
        config_.local_fallback_enabled = true;
        config_.local_cache_size = 100;
    }
    
    CacheClientConfig config_;
};

// ============================================================================
// Construction Tests
// ============================================================================

TEST_F(CacheClientTest, ConstructWithDefaultConfig) {
    CacheClientConfig default_config;
    CacheClient cache(default_config);
    
    EXPECT_EQ(cache.local_cache_hits(), 0);
    EXPECT_EQ(cache.local_cache_misses(), 0);
}

TEST_F(CacheClientTest, ConstructWithCustomConfig) {
    config_.namespace_prefix = "custom-namespace";
    config_.local_cache_size = 500;
    
    CacheClient cache(config_);
    
    EXPECT_EQ(cache.local_cache_hits(), 0);
}

TEST_F(CacheClientTest, MoveConstruction) {
    CacheClient cache1(config_);
    std::vector<uint8_t> value = {1, 2, 3};
    cache1.set("test_key", value);
    
    CacheClient cache2(std::move(cache1));
    
    // cache2 should have the data
    auto result = cache2.get("test_key");
    EXPECT_TRUE(result.has_value());
}

TEST_F(CacheClientTest, MoveAssignment) {
    CacheClient cache1(config_);
    CacheClient cache2(config_);
    
    std::vector<uint8_t> value = {4, 5, 6};
    cache1.set("move_key", value);
    
    cache2 = std::move(cache1);
    
    auto result = cache2.get("move_key");
    EXPECT_TRUE(result.has_value());
}

// ============================================================================
// Basic Operations Tests
// ============================================================================

TEST_F(CacheClientTest, SetAndGet) {
    CacheClient cache(config_);
    
    std::vector<uint8_t> value = {0x01, 0x02, 0x03, 0x04};
    auto set_result = cache.set("basic_key", value);
    ASSERT_TRUE(set_result.has_value());
    
    auto get_result = cache.get("basic_key");
    ASSERT_TRUE(get_result.has_value());
    EXPECT_EQ(*get_result, value);
}

TEST_F(CacheClientTest, GetNonExistent) {
    CacheClient cache(config_);
    
    auto result = cache.get("nonexistent_key_12345");
    ASSERT_FALSE(result.has_value());
    EXPECT_EQ(result.error().code, CacheErrorCode::NOT_FOUND);
}

TEST_F(CacheClientTest, Delete) {
    CacheClient cache(config_);
    
    std::vector<uint8_t> value = {0xAB, 0xCD};
    cache.set("delete_key", value);
    
    auto del_result = cache.del("delete_key");
    EXPECT_TRUE(del_result.has_value());
    
    auto get_result = cache.get("delete_key");
    EXPECT_FALSE(get_result.has_value());
}

TEST_F(CacheClientTest, DeleteNonExistent) {
    CacheClient cache(config_);
    
    // Deleting non-existent key should succeed (idempotent)
    auto result = cache.del("nonexistent_delete_key");
    EXPECT_TRUE(result.has_value());
}

TEST_F(CacheClientTest, Exists) {
    CacheClient cache(config_);
    
    std::vector<uint8_t> value = {0x11, 0x22};
    cache.set("exists_key", value);
    
    EXPECT_TRUE(cache.exists("exists_key"));
    EXPECT_FALSE(cache.exists("not_exists_key"));
}

TEST_F(CacheClientTest, Overwrite) {
    CacheClient cache(config_);
    
    std::vector<uint8_t> value1 = {0x01};
    std::vector<uint8_t> value2 = {0x02, 0x03};
    
    cache.set("overwrite_key", value1);
    cache.set("overwrite_key", value2);
    
    auto result = cache.get("overwrite_key");
    ASSERT_TRUE(result.has_value());
    EXPECT_EQ(*result, value2);
}

// ============================================================================
// TTL Tests
// ============================================================================

TEST_F(CacheClientTest, SetWithCustomTTL) {
    CacheClient cache(config_);
    
    std::vector<uint8_t> value = {0xFF};
    auto result = cache.set("ttl_key", value, std::chrono::seconds(60));
    
    EXPECT_TRUE(result.has_value());
}

TEST_F(CacheClientTest, SetWithDefaultTTL) {
    CacheClient cache(config_);
    
    std::vector<uint8_t> value = {0xEE};
    auto result = cache.set("default_ttl_key", value);
    
    EXPECT_TRUE(result.has_value());
}

// ============================================================================
// Batch Operations Tests
// ============================================================================

TEST_F(CacheClientTest, BatchSet) {
    CacheClient cache(config_);
    
    std::map<std::string, std::vector<uint8_t>> entries = {
        {"batch_key_1", {0x01}},
        {"batch_key_2", {0x02}},
        {"batch_key_3", {0x03}}
    };
    
    auto result = cache.batch_set(entries);
    EXPECT_TRUE(result.has_value());
    
    // Verify all were set
    EXPECT_TRUE(cache.exists("batch_key_1"));
    EXPECT_TRUE(cache.exists("batch_key_2"));
    EXPECT_TRUE(cache.exists("batch_key_3"));
}

TEST_F(CacheClientTest, BatchGet) {
    CacheClient cache(config_);
    
    // Set up data
    cache.set("bg_key_1", std::vector<uint8_t>{0x11});
    cache.set("bg_key_2", std::vector<uint8_t>{0x22});
    
    std::vector<std::string> keys = {"bg_key_1", "bg_key_2", "bg_key_missing"};
    
    auto result = cache.batch_get(keys);
    ASSERT_TRUE(result.has_value());
    
    // Should have 2 results (missing key not included)
    EXPECT_EQ(result->size(), 2);
    EXPECT_TRUE(result->count("bg_key_1") == 1);
    EXPECT_TRUE(result->count("bg_key_2") == 1);
    EXPECT_TRUE(result->count("bg_key_missing") == 0);
}

TEST_F(CacheClientTest, BatchDelete) {
    CacheClient cache(config_);
    
    // Set up data
    cache.set("bd_key_1", std::vector<uint8_t>{0x01});
    cache.set("bd_key_2", std::vector<uint8_t>{0x02});
    cache.set("bd_key_3", std::vector<uint8_t>{0x03});
    
    std::vector<std::string> keys = {"bd_key_1", "bd_key_2"};
    
    auto result = cache.batch_del(keys);
    EXPECT_TRUE(result.has_value());
    
    // Deleted keys should be gone
    EXPECT_FALSE(cache.exists("bd_key_1"));
    EXPECT_FALSE(cache.exists("bd_key_2"));
    // Undeleted key should remain
    EXPECT_TRUE(cache.exists("bd_key_3"));
}

// ============================================================================
// Local Cache Tests
// ============================================================================

TEST_F(CacheClientTest, LocalCacheHitsMisses) {
    CacheClient cache(config_);
    
    // Initial state
    EXPECT_EQ(cache.local_cache_hits(), 0);
    EXPECT_EQ(cache.local_cache_misses(), 0);
}

TEST_F(CacheClientTest, ClearLocalCache) {
    CacheClient cache(config_);
    
    // Add some data
    cache.set("clear_key_1", std::vector<uint8_t>{0x01});
    cache.set("clear_key_2", std::vector<uint8_t>{0x02});
    
    // Clear local cache
    cache.clear_local_cache();
    
    // Stats should be reset
    EXPECT_EQ(cache.local_cache_hits(), 0);
    EXPECT_EQ(cache.local_cache_misses(), 0);
}

// ============================================================================
// KeyCacheHelper Tests
// ============================================================================

TEST_F(CacheClientTest, KeyCacheHelperCacheKey) {
    CacheClient cache(config_);
    KeyCacheHelper helper(cache);
    
    std::string key_id = "test:key:v1";
    std::vector<uint8_t> key_material(32, 0xAB);
    
    auto result = helper.cache_key(key_id, key_material, std::chrono::seconds(300));
    EXPECT_TRUE(result.has_value());
}

TEST_F(CacheClientTest, KeyCacheHelperGetKey) {
    CacheClient cache(config_);
    KeyCacheHelper helper(cache);
    
    std::string key_id = "test:get:v1";
    std::vector<uint8_t> key_material(32, 0xCD);
    
    helper.cache_key(key_id, key_material, std::chrono::seconds(300));
    
    auto result = helper.get_key(key_id);
    ASSERT_TRUE(result.has_value());
    EXPECT_EQ(*result, key_material);
}

TEST_F(CacheClientTest, KeyCacheHelperGetKeyNotFound) {
    CacheClient cache(config_);
    KeyCacheHelper helper(cache);
    
    auto result = helper.get_key("nonexistent:key:v1");
    EXPECT_FALSE(result.has_value());
}

TEST_F(CacheClientTest, KeyCacheHelperInvalidateKey) {
    CacheClient cache(config_);
    KeyCacheHelper helper(cache);
    
    std::string key_id = "test:invalidate:v1";
    std::vector<uint8_t> key_material(32, 0xEF);
    
    helper.cache_key(key_id, key_material, std::chrono::seconds(300));
    
    auto invalidate_result = helper.invalidate_key(key_id);
    EXPECT_TRUE(invalidate_result.has_value());
    
    auto get_result = helper.get_key(key_id);
    EXPECT_FALSE(get_result.has_value());
}

// ============================================================================
// Edge Cases
// ============================================================================

TEST_F(CacheClientTest, EmptyValue) {
    CacheClient cache(config_);
    
    std::vector<uint8_t> empty_value;
    auto set_result = cache.set("empty_value_key", empty_value);
    ASSERT_TRUE(set_result.has_value());
    
    auto get_result = cache.get("empty_value_key");
    ASSERT_TRUE(get_result.has_value());
    EXPECT_TRUE(get_result->empty());
}

TEST_F(CacheClientTest, LargeValue) {
    CacheClient cache(config_);
    
    // 1MB value
    std::vector<uint8_t> large_value(1024 * 1024, 0x55);
    auto set_result = cache.set("large_value_key", large_value);
    ASSERT_TRUE(set_result.has_value());
    
    auto get_result = cache.get("large_value_key");
    ASSERT_TRUE(get_result.has_value());
    EXPECT_EQ(*get_result, large_value);
}

TEST_F(CacheClientTest, SpecialCharactersInKey) {
    CacheClient cache(config_);
    
    std::string key = "key:with:colons-and-dashes_underscores";
    std::vector<uint8_t> value = {0x01, 0x02};
    
    cache.set(key, value);
    
    auto result = cache.get(key);
    ASSERT_TRUE(result.has_value());
    EXPECT_EQ(*result, value);
}

TEST_F(CacheClientTest, NamespaceIsolation) {
    CacheClientConfig config1 = config_;
    config1.namespace_prefix = "namespace1";
    CacheClient cache1(config1);
    
    CacheClientConfig config2 = config_;
    config2.namespace_prefix = "namespace2";
    CacheClient cache2(config2);
    
    std::string key = "shared_key";
    std::vector<uint8_t> value1 = {0x01};
    std::vector<uint8_t> value2 = {0x02};
    
    cache1.set(key, value1);
    cache2.set(key, value2);
    
    auto result1 = cache1.get(key);
    auto result2 = cache2.get(key);
    
    ASSERT_TRUE(result1.has_value());
    ASSERT_TRUE(result2.has_value());
    EXPECT_EQ(*result1, value1);
    EXPECT_EQ(*result2, value2);
}

TEST_F(CacheClientTest, EmptyBatchGet) {
    CacheClient cache(config_);
    
    std::vector<std::string> empty_keys;
    auto result = cache.batch_get(empty_keys);
    
    ASSERT_TRUE(result.has_value());
    EXPECT_TRUE(result->empty());
}

TEST_F(CacheClientTest, EmptyBatchSet) {
    CacheClient cache(config_);
    
    std::map<std::string, std::vector<uint8_t>> empty_entries;
    auto result = cache.batch_set(empty_entries);
    
    EXPECT_TRUE(result.has_value());
}

TEST_F(CacheClientTest, EmptyBatchDelete) {
    CacheClient cache(config_);
    
    std::vector<std::string> empty_keys;
    auto result = cache.batch_del(empty_keys);
    
    EXPECT_TRUE(result.has_value());
}

} // namespace crypto::test
