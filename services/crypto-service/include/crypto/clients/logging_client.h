#pragma once

/**
 * @file logging_client.h
 * @brief gRPC client for centralized logging-service integration
 * 
 * This client provides async logging to the platform logging-service
 * with batch buffering, automatic flush, and local fallback.
 * 
 * Requirements: 1.1, 1.2, 1.3, 1.4
 */

#include <string>
#include <string_view>
#include <map>
#include <memory>
#include <chrono>
#include <functional>

namespace crypto {

// ============================================================================
// Log Levels
// ============================================================================

/**
 * @brief Log severity levels matching logging-service proto
 */
enum class LogLevel {
    DEBUG = 1,
    INFO = 2,
    WARN = 3,
    ERROR = 4,
    FATAL = 5
};

/**
 * @brief Convert log level to string
 */
[[nodiscard]] constexpr std::string_view log_level_to_string(LogLevel level) noexcept {
    switch (level) {
        case LogLevel::DEBUG: return "DEBUG";
        case LogLevel::INFO: return "INFO";
        case LogLevel::WARN: return "WARN";
        case LogLevel::ERROR: return "ERROR";
        case LogLevel::FATAL: return "FATAL";
    }
    return "UNKNOWN";
}

// ============================================================================
// Configuration
// ============================================================================

/**
 * @brief Configuration for LoggingClient
 */
struct LoggingClientConfig {
    /// gRPC address of logging-service (host:port)
    std::string address = "localhost:5001";
    
    /// Service identifier for log entries
    std::string service_id = "crypto-service";
    
    /// Number of log entries to buffer before flush
    size_t batch_size = 100;
    
    /// Maximum time before automatic flush
    std::chrono::milliseconds flush_interval{5000};
    
    /// Maximum buffer size (drops oldest if exceeded)
    size_t buffer_size = 10000;
    
    /// Enable local console fallback when service unavailable
    bool fallback_enabled = true;
    
    /// Minimum log level to send
    LogLevel min_level = LogLevel::INFO;
    
    /// Connection timeout
    std::chrono::milliseconds connect_timeout{5000};
    
    /// Request timeout
    std::chrono::milliseconds request_timeout{2000};
};

// ============================================================================
// LoggingClient
// ============================================================================

/**
 * @brief Async gRPC client for centralized logging
 * 
 * Features:
 * - Async batch logging to logging-service
 * - Automatic flush on batch size or interval
 * - Local console fallback when service unavailable
 * - Thread-safe log submission
 * 
 * Usage:
 *   LoggingClientConfig config;
 *   config.address = "logging-service:5001";
 *   LoggingClient logger(config);
 *   
 *   logger.info("Operation completed", {{"key_id", "abc123"}});
 */
class LoggingClient {
public:
    /**
     * @brief Construct logging client with configuration
     * @param config Client configuration
     */
    explicit LoggingClient(const LoggingClientConfig& config);
    
    /**
     * @brief Destructor - flushes pending logs
     */
    ~LoggingClient();
    
    // Non-copyable, movable
    LoggingClient(const LoggingClient&) = delete;
    LoggingClient& operator=(const LoggingClient&) = delete;
    LoggingClient(LoggingClient&&) noexcept;
    LoggingClient& operator=(LoggingClient&&) noexcept;
    
    // ========================================================================
    // Convenience logging methods
    // ========================================================================
    
    /**
     * @brief Log debug message
     */
    void debug(std::string_view message,
               const std::map<std::string, std::string>& fields = {});
    
    /**
     * @brief Log info message
     */
    void info(std::string_view message,
              const std::map<std::string, std::string>& fields = {});
    
    /**
     * @brief Log warning message
     */
    void warn(std::string_view message,
              const std::map<std::string, std::string>& fields = {});
    
    /**
     * @brief Log error message
     */
    void error(std::string_view message,
               const std::map<std::string, std::string>& fields = {});
    
    /**
     * @brief Log fatal message
     */
    void fatal(std::string_view message,
               const std::map<std::string, std::string>& fields = {});
    
    // ========================================================================
    // Structured logging
    // ========================================================================
    
    /**
     * @brief Log with full context
     * @param level Log severity level
     * @param message Log message
     * @param correlation_id Request correlation ID for tracing
     * @param fields Additional structured fields
     */
    void log(LogLevel level,
             std::string_view message,
             std::string_view correlation_id = "",
             const std::map<std::string, std::string>& fields = {});
    
    // ========================================================================
    // Control methods
    // ========================================================================
    
    /**
     * @brief Flush all buffered log entries
     * 
     * Blocks until all pending logs are sent or timeout.
     */
    void flush();
    
    /**
     * @brief Check if connected to logging service
     * @return true if connected and healthy
     */
    [[nodiscard]] bool is_connected() const;
    
    /**
     * @brief Get number of pending log entries
     * @return Number of buffered entries
     */
    [[nodiscard]] size_t pending_count() const;
    
    /**
     * @brief Get number of dropped log entries (buffer overflow)
     * @return Number of dropped entries since start
     */
    [[nodiscard]] size_t dropped_count() const;

private:
    struct Impl;
    std::unique_ptr<Impl> impl_;
};

// ============================================================================
// Scoped Logger
// ============================================================================

/**
 * @brief RAII helper for operation logging with timing
 * 
 * Logs operation start and completion with duration.
 * 
 * Usage:
 *   {
 *       ScopedLogger scope(logger, "encrypt", correlation_id);
 *       // ... operation ...
 *   } // Logs completion with duration
 */
class ScopedLogger {
public:
    ScopedLogger(LoggingClient& client,
                 std::string_view operation,
                 std::string_view correlation_id = "",
                 const std::map<std::string, std::string>& fields = {});
    
    ~ScopedLogger();
    
    /// Mark operation as failed (changes completion log level)
    void set_failed(std::string_view error_message = "");
    
    /// Add additional field to completion log
    void add_field(std::string_view key, std::string_view value);

private:
    LoggingClient& client_;
    std::string operation_;
    std::string correlation_id_;
    std::map<std::string, std::string> fields_;
    std::chrono::steady_clock::time_point start_time_;
    bool failed_ = false;
    std::string error_message_;
};

} // namespace crypto
