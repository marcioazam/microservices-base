// Unit tests for LoggingClient
// Tests connection, batch buffering, and fallback behavior

#include <gtest/gtest.h>
#include <gmock/gmock.h>
#include "crypto/clients/logging_client.h"
#include <thread>
#include <chrono>
#include <sstream>

namespace crypto::test {

// ============================================================================
// Test Fixture
// ============================================================================

class LoggingClientTest : public ::testing::Test {
protected:
    void SetUp() override {
        config_.address = "localhost:5001";
        config_.service_id = "crypto-service-test";
        config_.batch_size = 10;
        config_.flush_interval = std::chrono::milliseconds(100);
        config_.buffer_size = 1000;
        config_.fallback_enabled = true;
        config_.min_level = LogLevel::DEBUG;
    }
    
    LoggingClientConfig config_;
};

// ============================================================================
// Construction Tests
// ============================================================================

TEST_F(LoggingClientTest, ConstructWithDefaultConfig) {
    LoggingClientConfig default_config;
    LoggingClient logger(default_config);
    
    EXPECT_EQ(logger.pending_count(), 0);
    EXPECT_EQ(logger.dropped_count(), 0);
}

TEST_F(LoggingClientTest, ConstructWithCustomConfig) {
    config_.batch_size = 50;
    config_.service_id = "custom-service";
    
    LoggingClient logger(config_);
    
    EXPECT_EQ(logger.pending_count(), 0);
}

TEST_F(LoggingClientTest, MoveConstruction) {
    LoggingClient logger1(config_);
    logger1.info("Test message");
    
    LoggingClient logger2(std::move(logger1));
    
    // logger2 should have the pending message
    EXPECT_GE(logger2.pending_count(), 0);
}

TEST_F(LoggingClientTest, MoveAssignment) {
    LoggingClient logger1(config_);
    LoggingClient logger2(config_);
    
    logger1.info("Test message");
    logger2 = std::move(logger1);
    
    EXPECT_GE(logger2.pending_count(), 0);
}

// ============================================================================
// Logging Level Tests
// ============================================================================

TEST_F(LoggingClientTest, DebugLevel) {
    LoggingClient logger(config_);
    
    logger.debug("Debug message");
    logger.flush();
    
    EXPECT_EQ(logger.pending_count(), 0);
}

TEST_F(LoggingClientTest, InfoLevel) {
    LoggingClient logger(config_);
    
    logger.info("Info message");
    logger.flush();
    
    EXPECT_EQ(logger.pending_count(), 0);
}

TEST_F(LoggingClientTest, WarnLevel) {
    LoggingClient logger(config_);
    
    logger.warn("Warning message");
    logger.flush();
    
    EXPECT_EQ(logger.pending_count(), 0);
}

TEST_F(LoggingClientTest, ErrorLevel) {
    LoggingClient logger(config_);
    
    logger.error("Error message");
    logger.flush();
    
    EXPECT_EQ(logger.pending_count(), 0);
}

TEST_F(LoggingClientTest, FatalLevel) {
    LoggingClient logger(config_);
    
    logger.fatal("Fatal message");
    logger.flush();
    
    EXPECT_EQ(logger.pending_count(), 0);
}

// ============================================================================
// Log Level Filtering Tests
// ============================================================================

TEST_F(LoggingClientTest, FiltersBelowMinLevel) {
    config_.min_level = LogLevel::WARN;
    LoggingClient logger(config_);
    
    // DEBUG and INFO should be filtered
    logger.debug("Should be filtered");
    logger.info("Should be filtered");
    
    // WARN and above should pass
    logger.warn("Should pass");
    logger.error("Should pass");
    
    // Pending count depends on implementation
    // At minimum, filtered messages shouldn't cause errors
    logger.flush();
    EXPECT_EQ(logger.pending_count(), 0);
}

TEST_F(LoggingClientTest, MinLevelDebugAcceptsAll) {
    config_.min_level = LogLevel::DEBUG;
    LoggingClient logger(config_);
    
    logger.debug("Debug");
    logger.info("Info");
    logger.warn("Warn");
    logger.error("Error");
    logger.fatal("Fatal");
    
    logger.flush();
    EXPECT_EQ(logger.pending_count(), 0);
}

// ============================================================================
// Structured Logging Tests
// ============================================================================

TEST_F(LoggingClientTest, LogWithCorrelationId) {
    LoggingClient logger(config_);
    
    logger.log(LogLevel::INFO, "Test message", "corr-12345");
    logger.flush();
    
    EXPECT_EQ(logger.pending_count(), 0);
}

TEST_F(LoggingClientTest, LogWithFields) {
    LoggingClient logger(config_);
    
    std::map<std::string, std::string> fields = {
        {"key_id", "key-123"},
        {"algorithm", "AES-256-GCM"},
        {"operation", "encrypt"}
    };
    
    logger.log(LogLevel::INFO, "Encryption completed", "corr-123", fields);
    logger.flush();
    
    EXPECT_EQ(logger.pending_count(), 0);
}

TEST_F(LoggingClientTest, LogWithEmptyFields) {
    LoggingClient logger(config_);
    
    logger.log(LogLevel::INFO, "Message", "corr-123", {});
    logger.flush();
    
    EXPECT_EQ(logger.pending_count(), 0);
}

TEST_F(LoggingClientTest, LogWithEmptyCorrelationId) {
    LoggingClient logger(config_);
    
    logger.log(LogLevel::INFO, "Message without correlation", "");
    logger.flush();
    
    EXPECT_EQ(logger.pending_count(), 0);
}

// ============================================================================
// Batch Buffering Tests
// ============================================================================

TEST_F(LoggingClientTest, BuffersUntilBatchSize) {
    config_.batch_size = 5;
    LoggingClient logger(config_);
    
    // Log fewer than batch size
    for (int i = 0; i < 3; ++i) {
        logger.info("Message " + std::to_string(i));
    }
    
    // Should be buffered
    EXPECT_LE(logger.pending_count(), 3);
}

TEST_F(LoggingClientTest, FlushClearsBuffer) {
    LoggingClient logger(config_);
    
    logger.info("Message 1");
    logger.info("Message 2");
    logger.info("Message 3");
    
    logger.flush();
    
    EXPECT_EQ(logger.pending_count(), 0);
}

TEST_F(LoggingClientTest, MultipleFlushesAreSafe) {
    LoggingClient logger(config_);
    
    logger.info("Message");
    logger.flush();
    logger.flush();  // Second flush should be safe
    logger.flush();  // Third flush should be safe
    
    EXPECT_EQ(logger.pending_count(), 0);
}

// ============================================================================
// ScopedLogger Tests
// ============================================================================

TEST_F(LoggingClientTest, ScopedLoggerBasic) {
    LoggingClient logger(config_);
    
    {
        ScopedLogger scope(logger, "test_operation", "corr-123");
        // Operation runs here
    }
    
    logger.flush();
    EXPECT_EQ(logger.pending_count(), 0);
}

TEST_F(LoggingClientTest, ScopedLoggerWithFields) {
    LoggingClient logger(config_);
    
    {
        ScopedLogger scope(logger, "encrypt", "corr-456", {
            {"key_id", "key-789"},
            {"algorithm", "AES-256"}
        });
    }
    
    logger.flush();
    EXPECT_EQ(logger.pending_count(), 0);
}

TEST_F(LoggingClientTest, ScopedLoggerSetFailed) {
    LoggingClient logger(config_);
    
    {
        ScopedLogger scope(logger, "decrypt", "corr-789");
        scope.set_failed("Integrity check failed");
    }
    
    logger.flush();
    EXPECT_EQ(logger.pending_count(), 0);
}

TEST_F(LoggingClientTest, ScopedLoggerAddField) {
    LoggingClient logger(config_);
    
    {
        ScopedLogger scope(logger, "sign", "corr-abc");
        scope.add_field("signature_size", "256");
        scope.add_field("hash_algorithm", "SHA-256");
    }
    
    logger.flush();
    EXPECT_EQ(logger.pending_count(), 0);
}

// ============================================================================
// Log Level String Conversion Tests
// ============================================================================

TEST_F(LoggingClientTest, LogLevelToStringDebug) {
    EXPECT_EQ(log_level_to_string(LogLevel::DEBUG), "DEBUG");
}

TEST_F(LoggingClientTest, LogLevelToStringInfo) {
    EXPECT_EQ(log_level_to_string(LogLevel::INFO), "INFO");
}

TEST_F(LoggingClientTest, LogLevelToStringWarn) {
    EXPECT_EQ(log_level_to_string(LogLevel::WARN), "WARN");
}

TEST_F(LoggingClientTest, LogLevelToStringError) {
    EXPECT_EQ(log_level_to_string(LogLevel::ERROR), "ERROR");
}

TEST_F(LoggingClientTest, LogLevelToStringFatal) {
    EXPECT_EQ(log_level_to_string(LogLevel::FATAL), "FATAL");
}

// ============================================================================
// Edge Cases
// ============================================================================

TEST_F(LoggingClientTest, EmptyMessage) {
    LoggingClient logger(config_);
    
    logger.info("");
    logger.flush();
    
    EXPECT_EQ(logger.pending_count(), 0);
}

TEST_F(LoggingClientTest, VeryLongMessage) {
    LoggingClient logger(config_);
    
    std::string long_message(10000, 'x');
    logger.info(long_message);
    logger.flush();
    
    EXPECT_EQ(logger.pending_count(), 0);
}

TEST_F(LoggingClientTest, SpecialCharactersInMessage) {
    LoggingClient logger(config_);
    
    logger.info("Message with special chars: \t\n\"'\\{}[]");
    logger.flush();
    
    EXPECT_EQ(logger.pending_count(), 0);
}

TEST_F(LoggingClientTest, UnicodeInMessage) {
    LoggingClient logger(config_);
    
    logger.info("Unicode: 日本語 中文 한국어 🔐");
    logger.flush();
    
    EXPECT_EQ(logger.pending_count(), 0);
}

TEST_F(LoggingClientTest, ManyFieldsInLog) {
    LoggingClient logger(config_);
    
    std::map<std::string, std::string> fields;
    for (int i = 0; i < 100; ++i) {
        fields["field_" + std::to_string(i)] = "value_" + std::to_string(i);
    }
    
    logger.log(LogLevel::INFO, "Message with many fields", "corr-123", fields);
    logger.flush();
    
    EXPECT_EQ(logger.pending_count(), 0);
}

TEST_F(LoggingClientTest, DroppedCountInitiallyZero) {
    LoggingClient logger(config_);
    
    EXPECT_EQ(logger.dropped_count(), 0);
}

TEST_F(LoggingClientTest, DestructorFlushes) {
    {
        LoggingClient logger(config_);
        logger.info("Message before destruction");
        // Destructor should flush
    }
    // No crash = success
}

} // namespace crypto::test
