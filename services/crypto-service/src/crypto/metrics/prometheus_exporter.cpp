#include "crypto/metrics/prometheus_exporter.h"
#include <sstream>
#include <iomanip>
#include <algorithm>

namespace crypto {

// Global exporter singleton
static std::unique_ptr<PrometheusExporter> g_exporter;
static std::once_flag g_exporter_init;

PrometheusExporter& getMetricsExporter() {
    std::call_once(g_exporter_init, []() {
        g_exporter = std::make_unique<PrometheusExporter>();
    });
    return *g_exporter;
}

// Histogram implementation
Histogram::Histogram(const std::vector<double>& buckets) {
    for (double bound : buckets) {
        buckets_.push_back({bound, 0});
    }
    // Add +Inf bucket
    buckets_.push_back({std::numeric_limits<double>::infinity(), 0});
}

void Histogram::observe(double value) {
    for (auto& bucket : buckets_) {
        if (value <= bucket.upper_bound) {
            bucket.count.fetch_add(1);
        }
    }
    count_.fetch_add(1);
    
    // Atomic add for sum (simplified - in production use compare_exchange)
    double current = sum_.load();
    while (!sum_.compare_exchange_weak(current, current + value)) {}
}

std::string Histogram::serialize(const std::string& name,
                                  const std::string& labels) const {
    std::ostringstream ss;
    
    for (const auto& bucket : buckets_) {
        ss << name << "_bucket{";
        if (!labels.empty()) {
            ss << labels << ",";
        }
        ss << "le=\"";
        if (bucket.upper_bound == std::numeric_limits<double>::infinity()) {
            ss << "+Inf";
        } else {
            ss << bucket.upper_bound;
        }
        ss << "\"} " << bucket.count.load() << "\n";
    }
    
    ss << name << "_sum";
    if (!labels.empty()) {
        ss << "{" << labels << "}";
    }
    ss << " " << sum_.load() << "\n";
    
    ss << name << "_count";
    if (!labels.empty()) {
        ss << "{" << labels << "}";
    }
    ss << " " << count_.load() << "\n";
    
    return ss.str();
}

// Counter implementation
void Counter::increment(uint64_t value) {
    value_.fetch_add(value);
}

std::string Counter::serialize(const std::string& name,
                                const std::string& labels) const {
    std::ostringstream ss;
    ss << name;
    if (!labels.empty()) {
        ss << "{" << labels << "}";
    }
    ss << " " << value_.load() << "\n";
    return ss.str();
}

// Gauge implementation
void Gauge::set(double value) {
    value_.store(value);
}

void Gauge::increment(double value) {
    double current = value_.load();
    while (!value_.compare_exchange_weak(current, current + value)) {}
}

void Gauge::decrement(double value) {
    increment(-value);
}

std::string Gauge::serialize(const std::string& name,
                              const std::string& labels) const {
    std::ostringstream ss;
    ss << name;
    if (!labels.empty()) {
        ss << "{" << labels << "}";
    }
    ss << " " << value_.load() << "\n";
    return ss.str();
}

// PrometheusExporter implementation
std::vector<double> PrometheusExporter::defaultLatencyBuckets() {
    return {0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0};
}

PrometheusExporter::PrometheusExporter() {
    auto buckets = defaultLatencyBuckets();
    encrypt_latency_ = std::make_unique<Histogram>(buckets);
    decrypt_latency_ = std::make_unique<Histogram>(buckets);
    sign_latency_ = std::make_unique<Histogram>(buckets);
    verify_latency_ = std::make_unique<Histogram>(buckets);
    key_operation_latency_ = std::make_unique<Histogram>(buckets);
}

void PrometheusExporter::recordEncrypt(bool success) {
    encrypt_total_.increment();
    if (success) {
        encrypt_success_.increment();
    }
}

void PrometheusExporter::recordDecrypt(bool success) {
    decrypt_total_.increment();
    if (success) {
        decrypt_success_.increment();
    }
}

void PrometheusExporter::recordSign(bool success) {
    sign_total_.increment();
    if (success) {
        sign_success_.increment();
    }
}

void PrometheusExporter::recordVerify(bool success) {
    verify_total_.increment();
    if (success) {
        verify_success_.increment();
    }
}

void PrometheusExporter::recordKeyGenerate(bool success) {
    key_generate_total_.increment();
}

void PrometheusExporter::recordKeyRotate(bool success) {
    key_rotate_total_.increment();
}

void PrometheusExporter::recordKeyDelete(bool success) {
    key_delete_total_.increment();
}

void PrometheusExporter::recordEncryptLatency(std::chrono::nanoseconds duration) {
    double seconds = duration.count() / 1e9;
    encrypt_latency_->observe(seconds);
}

void PrometheusExporter::recordDecryptLatency(std::chrono::nanoseconds duration) {
    double seconds = duration.count() / 1e9;
    decrypt_latency_->observe(seconds);
}

void PrometheusExporter::recordSignLatency(std::chrono::nanoseconds duration) {
    double seconds = duration.count() / 1e9;
    sign_latency_->observe(seconds);
}

void PrometheusExporter::recordVerifyLatency(std::chrono::nanoseconds duration) {
    double seconds = duration.count() / 1e9;
    verify_latency_->observe(seconds);
}

void PrometheusExporter::recordKeyOperationLatency(std::chrono::nanoseconds duration) {
    double seconds = duration.count() / 1e9;
    key_operation_latency_->observe(seconds);
}

// Error recording with ErrorCode label (Requirement 9.5)
void PrometheusExporter::recordError(ErrorCode code) {
    std::lock_guard<std::mutex> lock(error_mutex_);
    error_code_counters_[code].increment();
}

void PrometheusExporter::recordError(const std::string& error_type) {
    std::lock_guard<std::mutex> lock(error_mutex_);
    error_counters_[error_type].increment();
}

void PrometheusExporter::recordError(const Error& error) {
    recordError(error.code);
}

void PrometheusExporter::setHSMConnected(bool connected) {
    hsm_connected_.set(connected ? 1.0 : 0.0);
}

void PrometheusExporter::setKMSConnected(bool connected) {
    kms_connected_.set(connected ? 1.0 : 0.0);
}

void PrometheusExporter::setLoggingServiceConnected(bool connected) {
    logging_service_connected_.set(connected ? 1.0 : 0.0);
}

void PrometheusExporter::setCacheServiceConnected(bool connected) {
    cache_service_connected_.set(connected ? 1.0 : 0.0);
}

std::string PrometheusExporter::serialize() const {
    std::ostringstream ss;
    
    // Operation counters
    ss << "# HELP crypto_encrypt_operations_total Total encrypt operations\n";
    ss << "# TYPE crypto_encrypt_operations_total counter\n";
    ss << encrypt_total_.serialize("crypto_encrypt_operations_total");
    
    ss << "# HELP crypto_decrypt_operations_total Total decrypt operations\n";
    ss << "# TYPE crypto_decrypt_operations_total counter\n";
    ss << decrypt_total_.serialize("crypto_decrypt_operations_total");
    
    ss << "# HELP crypto_sign_operations_total Total sign operations\n";
    ss << "# TYPE crypto_sign_operations_total counter\n";
    ss << sign_total_.serialize("crypto_sign_operations_total");
    
    ss << "# HELP crypto_verify_operations_total Total verify operations\n";
    ss << "# TYPE crypto_verify_operations_total counter\n";
    ss << verify_total_.serialize("crypto_verify_operations_total");
    
    ss << "# HELP crypto_key_operations_total Total key operations\n";
    ss << "# TYPE crypto_key_operations_total counter\n";
    ss << key_generate_total_.serialize("crypto_key_operations_total", "operation=\"generate\"");
    ss << key_rotate_total_.serialize("crypto_key_operations_total", "operation=\"rotate\"");
    ss << key_delete_total_.serialize("crypto_key_operations_total", "operation=\"delete\"");
    
    // Latency histograms
    ss << "# HELP crypto_operation_latency_seconds Operation latency\n";
    ss << "# TYPE crypto_operation_latency_seconds histogram\n";
    ss << encrypt_latency_->serialize("crypto_operation_latency_seconds", "operation=\"encrypt\"");
    ss << decrypt_latency_->serialize("crypto_operation_latency_seconds", "operation=\"decrypt\"");
    ss << sign_latency_->serialize("crypto_operation_latency_seconds", "operation=\"sign\"");
    ss << verify_latency_->serialize("crypto_operation_latency_seconds", "operation=\"verify\"");
    
    // Error counters with error_code label (Requirement 9.5)
    {
        std::lock_guard<std::mutex> lock(error_mutex_);
        if (!error_code_counters_.empty()) {
            ss << "# HELP crypto_errors_total Total errors by error_code\n";
            ss << "# TYPE crypto_errors_total counter\n";
            for (const auto& [code, counter] : error_code_counters_) {
                std::string label = "error_code=\"" + std::string(error_code_to_string(code)) + "\"";
                ss << counter.serialize("crypto_errors_total", label);
            }
        }
        // Legacy error counters
        if (!error_counters_.empty()) {
            for (const auto& [type, counter] : error_counters_) {
                ss << counter.serialize("crypto_errors_total", 
                                        "error_type=\"" + type + "\"");
            }
        }
    }
    
    // Connection status
    ss << "# HELP crypto_hsm_connected HSM connection status\n";
    ss << "# TYPE crypto_hsm_connected gauge\n";
    ss << hsm_connected_.serialize("crypto_hsm_connected");
    
    ss << "# HELP crypto_kms_connected KMS connection status\n";
    ss << "# TYPE crypto_kms_connected gauge\n";
    ss << kms_connected_.serialize("crypto_kms_connected");
    
    ss << "# HELP crypto_logging_service_connected Logging service connection status\n";
    ss << "# TYPE crypto_logging_service_connected gauge\n";
    ss << logging_service_connected_.serialize("crypto_logging_service_connected");
    
    ss << "# HELP crypto_cache_service_connected Cache service connection status\n";
    ss << "# TYPE crypto_cache_service_connected gauge\n";
    ss << cache_service_connected_.serialize("crypto_cache_service_connected");
    
    return ss.str();
}

// LatencyTimer implementation
LatencyTimer::LatencyTimer(Callback callback)
    : callback_(std::move(callback))
    , start_(std::chrono::steady_clock::now()) {}

LatencyTimer::~LatencyTimer() {
    auto end = std::chrono::steady_clock::now();
    auto duration = std::chrono::duration_cast<std::chrono::nanoseconds>(end - start_);
    if (callback_) {
        callback_(duration);
    }
}

std::chrono::nanoseconds LatencyTimer::elapsed() const {
    auto now = std::chrono::steady_clock::now();
    return std::chrono::duration_cast<std::chrono::nanoseconds>(now - start_);
}

} // namespace crypto
