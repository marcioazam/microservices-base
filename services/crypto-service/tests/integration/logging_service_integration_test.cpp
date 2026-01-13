/**
 * @file logging_service_integration_test.cpp
 * @brief Integration tests for LoggingClient with real logging-service
 * 
 * Requirements: 7.4
 */

#include <gtest/gtest.h>
#include "crypto/clients/logging_client.h"
#include <thread>
#include <chrono>

namespace crypto::test {

/**
 * @brief Integration test fixture for LoggingClient
 * 
 * Note: These tests require a running logging-service instance.
 * In CI, use Testcontainers to spin up the service.
 */
class LoggingServiceIntegrationTest : public ::testing::Test {
protected:
    void SetUp() override {
        // Get logging service address from environment or use default
        const char* addr = std::getenv("LOGGING_SERVICE_ADDRESS");
        config_.address = addr ? addr : "localhost:5001";
        config_.service_id = "crypto-service-test";
        config_.batch_size = 10;
        config_.flush_interval = std::chrono::milliseconds{100};
    }

    LoggingClientConfig config_;
};

TEST_F(LoggingServiceIntegrationTest, DISABLED_ConnectsToLoggingService) {
    // Disabled by default - enable when logging-service is available
    LoggingClient client(config_);
    
    // Give time for connection
    std::this_thread::sleep_for(std::chrono::milliseconds{500});
    
    EXPECT_TRUE(client.is_connected());
}

TEST_F(LoggingServiceIntegrationTest, DISABLED_SendsLogEntries) {
    LoggingClient client(config_);
    std::this_thread::sleep_for(std::chrono::milliseconds{500});
    
    if (!client.is_connected()) {
        GTEST_SKIP() << "Logging service not available";
    }
    
    // Send various log levels
    client.info("Test info message", {{"test_key", "test_value"}});
    client.warn("Test warning message", {{"correlation_id", "test-123"}});
    client.error("Test error message", {{"error_code", "TEST_ERROR"}});
    
    // Flush and verify no exceptions
    EXPECT_NO_THROW(client.flush());
}

TEST_F(LoggingServiceIntegrationTest, DISABLED_BatchesLogEntries) {
    config_.batch_size = 5;
    LoggingClient client(config_);
    std::this_thread::sleep_for(std::chrono::milliseconds{500});
    
    if (!client.is_connected()) {
        GTEST_SKIP() << "Logging service not available";
    }
    
    // Send more than batch size
    for (int i = 0; i < 10; ++i) {
        client.info("Batch test message " + std::to_string(i), 
                   {{"index", std::to_string(i)}});
    }
    
    // Should have auto-flushed at least once
    client.flush();
}

TEST_F(LoggingServiceIntegrationTest, DISABLED_FallsBackToConsoleWhenDisconnected) {
    config_.address = "invalid:9999";  // Invalid address
    LoggingClient client(config_);
    
    std::this_thread::sleep_for(std::chrono::milliseconds{500});
    
    // Should not be connected
    EXPECT_FALSE(client.is_connected());
    
    // Should not throw - falls back to console
    EXPECT_NO_THROW(client.info("Fallback test message"));
    EXPECT_NO_THROW(client.flush());
}

TEST_F(LoggingServiceIntegrationTest, DISABLED_IncludesCorrelationIdInAllEntries) {
    LoggingClient client(config_);
    std::this_thread::sleep_for(std::chrono::milliseconds{500});
    
    if (!client.is_connected()) {
        GTEST_SKIP() << "Logging service not available";
    }
    
    const std::string correlation_id = "corr-12345";
    
    client.log(LogLevel::INFO, "Test with correlation", correlation_id, 
               {{"operation", "test"}});
    
    client.flush();
    // Verification would require querying the logging service
}

} // namespace crypto::test
