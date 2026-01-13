#pragma once

/**
 * @file prometheus_exporter.h
 * @brief Prometheus metrics exporter with error_code labels and latency histograms
 * 
 * Requirements: 9.1, 9.5, 9.6
 */

#include "crypto/common/result.h"
#include <string>
#include <string_view>
#include <atomic>
#include <chrono>
#include <mutex>
#include <unordered_map>
#include <vector>
#include <functional>
#include <memory>

namespace crypto {

// ============================================================================
// Metric Types
// ============================================================================

/**
 * @brief Histogram bucket for latency tracking
 */
struct HistogramBucket {
    double upper_bound;
    std::atomic<uint64_t> count{0};
};

/**
 * @brief Histogram for latency tracking with configurable buckets
 */
class Histogram {
public:
    explicit Histogram(const std::vector<double>& buckets);
    
    void observe(double value);
    [[nodiscard]] std::string serialize(const std::string& name, 
                                        const std::string& labels = "") const;
    [[nodiscard]] uint64_t count() const { return count_.load(); }
    [[nodiscard]] double sum() const { return sum_.load(); }

private:
    std::vector<HistogramBucket> buckets_;
    std::atomic<uint64_t> count_{0};
    std::atomic<double> sum_{0.0};
};

/**
 * @brief Counter metric (monotonically increasing)
 */
class Counter {
public:
    Counter() = default;
    
    void increment(uint64_t value = 1);
    [[nodiscard]] uint64_t value() const { return value_.load(); }
    [[nodiscard]] std::string serialize(const std::string& name,
                                        const std::string& labels = "") const;

private:
    std::atomic<uint64_t> value_{0};
};

/**
 * @brief Gauge metric (can increase or decrease)
 */
class Gauge {
public:
    Gauge() = default;
    
    void set(double value);
    void increment(double value = 1.0);
    void decrement(double value = 1.0);
    [[nodiscard]] double value() const { return value_.load(); }
    [[nodiscard]] std::string serialize(const std::string& name,
                                        const std::string& labels = "") const;

private:
    std::atomic<double> value_{0.0};
};

// ============================================================================
// Prometheus Exporter
// ============================================================================

/**
 * @brief Prometheus metrics exporter with error_code labels
 * 
 * Provides metrics for:
 * - Operation counters (encrypt, decrypt, sign, verify, key ops)
 * - Latency histograms (p50, p95, p99)
 * - Error counters with error_code labels
 * - Connection status gauges
 * 
 * Requirements: 9.1, 9.5, 9.6
 */
class PrometheusExporter {
public:
    PrometheusExporter();
    ~PrometheusExporter() = default;
    
    // ========================================================================
    // Operation Recording
    // ========================================================================
    
    /**
     * @brief Record an encryption operation
     * @param success Whether the operation succeeded
     */
    void recordEncrypt(bool success);
    
    /**
     * @brief Record a decryption operation
     * @param success Whether the operation succeeded
     */
    void recordDecrypt(bool success);
    
    /**
     * @brief Record a signing operation
     * @param success Whether the operation succeeded
     */
    void recordSign(bool success);
    
    /**
     * @brief Record a verification operation
     * @param success Whether the operation succeeded
     */
    void recordVerify(bool success);
    
    /**
     * @brief Record a key generation operation
     * @param success Whether the operation succeeded
     */
    void recordKeyGenerate(bool success);
    
    /**
     * @brief Record a key rotation operation
     * @param success Whether the operation succeeded
     */
    void recordKeyRotate(bool success);
    
    /**
     * @brief Record a key deletion operation
     * @param success Whether the operation succeeded
     */
    void recordKeyDelete(bool success);
    
    // ========================================================================
    // Latency Recording
    // ========================================================================
    
    void recordEncryptLatency(std::chrono::nanoseconds duration);
    void recordDecryptLatency(std::chrono::nanoseconds duration);
    void recordSignLatency(std::chrono::nanoseconds duration);
    void recordVerifyLatency(std::chrono::nanoseconds duration);
    void recordKeyOperationLatency(std::chrono::nanoseconds duration);
    
    // ========================================================================
    // Error Recording with ErrorCode Labels (Requirement 9.5)
    // ========================================================================
    
    /**
     * @brief Record an error with error_code label
     * @param code The ErrorCode from the operation
     * 
     * Emits a counter metric with error_code label set to the specific code.
     * Example: crypto_errors_total{error_code="INVALID_KEY_SIZE"} 1
     */
    void recordError(ErrorCode code);
    
    /**
     * @brief Record an error with string type (legacy)
     * @param error_type Error type string
     */
    void recordError(const std::string& error_type);
    
    /**
     * @brief Record an error from a Result
     * @param error The Error from a failed Result
     */
    void recordError(const Error& error);
    
    // ========================================================================
    // Connection Status
    // ========================================================================
    
    void setHSMConnected(bool connected);
    void setKMSConnected(bool connected);
    void setLoggingServiceConnected(bool connected);
    void setCacheServiceConnected(bool connected);
    
    // ========================================================================
    // Serialization
    // ========================================================================
    
    /**
     * @brief Serialize all metrics to Prometheus text format
     * @return Prometheus-formatted metrics string
     */
    [[nodiscard]] std::string serialize() const;

private:
    // Operation counters
    Counter encrypt_total_;
    Counter encrypt_success_;
    Counter decrypt_total_;
    Counter decrypt_success_;
    Counter sign_total_;
    Counter sign_success_;
    Counter verify_total_;
    Counter verify_success_;
    Counter key_generate_total_;
    Counter key_rotate_total_;
    Counter key_delete_total_;
    
    // Latency histograms
    std::unique_ptr<Histogram> encrypt_latency_;
    std::unique_ptr<Histogram> decrypt_latency_;
    std::unique_ptr<Histogram> sign_latency_;
    std::unique_ptr<Histogram> verify_latency_;
    std::unique_ptr<Histogram> key_operation_latency_;
    
    // Error counters by error_code (Requirement 9.5)
    mutable std::mutex error_mutex_;
    std::unordered_map<ErrorCode, Counter> error_code_counters_;
    std::unordered_map<std::string, Counter> error_counters_;  // Legacy
    
    // Connection gauges
    Gauge hsm_connected_;
    Gauge kms_connected_;
    Gauge logging_service_connected_;
    Gauge cache_service_connected_;
    
    static std::vector<double> defaultLatencyBuckets();
};

// ============================================================================
// RAII Latency Timer
// ============================================================================

/**
 * @brief RAII timer for automatic latency recording
 */
class LatencyTimer {
public:
    using Callback = std::function<void(std::chrono::nanoseconds)>;
    
    explicit LatencyTimer(Callback callback);
    ~LatencyTimer();
    
    LatencyTimer(const LatencyTimer&) = delete;
    LatencyTimer& operator=(const LatencyTimer&) = delete;
    
    /**
     * @brief Get elapsed time without stopping the timer
     */
    [[nodiscard]] std::chrono::nanoseconds elapsed() const;

private:
    Callback callback_;
    std::chrono::steady_clock::time_point start_;
};

// ============================================================================
// Global Exporter Access
// ============================================================================

/**
 * @brief Get the global PrometheusExporter instance
 * @return Reference to the singleton exporter
 */
PrometheusExporter& getMetricsExporter();

} // namespace crypto
