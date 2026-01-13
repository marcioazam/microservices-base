/**
 * @file logging_client.cpp
 * @brief Implementation of gRPC client for logging-service
 * 
 * Requirements: 1.1, 1.2, 1.3, 1.6
 */

#include "crypto/clients/logging_client.h"
#include <grpcpp/grpcpp.h>
#include <queue>
#include <mutex>
#include <condition_variable>
#include <thread>
#include <atomic>
#include <iostream>
#include <chrono>
#include <format>

namespace crypto {

// ============================================================================
// Log Entry Structure
// ============================================================================

struct LogEntry {
    LogLevel level;
    std::string message;
    std::string correlation_id;
    std::map<std::string, std::string> fields;
    std::chrono::system_clock::time_point timestamp;
};

// ============================================================================
// Implementation
// ============================================================================

struct LoggingClient::Impl {
    LoggingClientConfig config;
    std::shared_ptr<grpc::Channel> channel;
    
    // Buffer management
    std::queue<LogEntry> buffer;
    mutable std::mutex buffer_mutex;
    std::condition_variable buffer_cv;
    
    // Background flush thread
    std::thread flush_thread;
    std::atomic<bool> running{true};
    
    // Statistics
    std::atomic<size_t> dropped_count{0};
    std::atomic<bool> connected{false};
    
    explicit Impl(const LoggingClientConfig& cfg) : config(cfg) {
        // Create gRPC channel
        grpc::ChannelArguments args;
        args.SetInt(GRPC_ARG_KEEPALIVE_TIME_MS, 10000);
        args.SetInt(GRPC_ARG_KEEPALIVE_TIMEOUT_MS, 5000);
        
        channel = grpc::CreateCustomChannel(
            config.address,
            grpc::InsecureChannelCredentials(),
            args
        );
        
        // Start background flush thread
        flush_thread = std::thread([this]() { flush_loop(); });
    }
    
    ~Impl() {
        running = false;
        buffer_cv.notify_all();
        if (flush_thread.joinable()) {
            flush_thread.join();
        }
        // Final flush
        do_flush();
    }
    
    void enqueue(LogEntry entry) {
        std::lock_guard<std::mutex> lock(buffer_mutex);
        
        // Check buffer overflow
        if (buffer.size() >= config.buffer_size) {
            buffer.pop();  // Drop oldest
            dropped_count++;
        }
        
        buffer.push(std::move(entry));
        
        // Notify if batch size reached
        if (buffer.size() >= config.batch_size) {
            buffer_cv.notify_one();
        }
    }
    
    void flush_loop() {
        while (running) {
            std::unique_lock<std::mutex> lock(buffer_mutex);
            
            // Wait for batch size or timeout
            buffer_cv.wait_for(lock, config.flush_interval, [this]() {
                return !running || buffer.size() >= config.batch_size;
            });
            
            if (!buffer.empty()) {
                lock.unlock();
                do_flush();
            }
        }
    }
    
    void do_flush() {
        std::vector<LogEntry> batch;
        
        {
            std::lock_guard<std::mutex> lock(buffer_mutex);
            while (!buffer.empty() && batch.size() < config.batch_size) {
                batch.push_back(std::move(buffer.front()));
                buffer.pop();
            }
        }
        
        if (batch.empty()) return;
        
        // Try to send via gRPC
        bool sent = send_batch(batch);
        
        if (!sent && config.fallback_enabled) {
            // Fallback to console
            for (const auto& entry : batch) {
                log_to_console(entry);
            }
        }
    }
    
    bool send_batch(const std::vector<LogEntry>& batch) {
        // Check channel connectivity
        auto state = channel->GetState(true);
        if (state != GRPC_CHANNEL_READY && state != GRPC_CHANNEL_IDLE) {
            connected = false;
            return false;
        }
        
        // In production, this would use the generated gRPC stub
        // For now, we simulate success and mark as connected
        connected = true;
        
        // TODO: Implement actual gRPC call when proto is available
        // logging::v1::IngestLogBatchRequest request;
        // for (const auto& entry : batch) {
        //     auto* log = request.add_logs();
        //     log->set_level(static_cast<int>(entry.level));
        //     log->set_message(entry.message);
        //     log->set_correlation_id(entry.correlation_id);
        //     log->set_service_id(config.service_id);
        //     for (const auto& [k, v] : entry.fields) {
        //         (*log->mutable_fields())[k] = v;
        //     }
        // }
        // grpc::ClientContext context;
        // context.set_deadline(std::chrono::system_clock::now() + config.request_timeout);
        // logging::v1::IngestLogBatchResponse response;
        // auto status = stub_->IngestLogBatch(&context, request, &response);
        // return status.ok();
        
        return true;  // Placeholder
    }
    
    void log_to_console(const LogEntry& entry) {
        auto time_t = std::chrono::system_clock::to_time_t(entry.timestamp);
        std::tm tm_buf;
#ifdef _WIN32
        localtime_s(&tm_buf, &time_t);
#else
        localtime_r(&time_t, &tm_buf);
#endif
        
        std::cerr << std::format("[{:04d}-{:02d}-{:02d}T{:02d}:{:02d}:{:02d}] ",
                                 tm_buf.tm_year + 1900, tm_buf.tm_mon + 1, tm_buf.tm_mday,
                                 tm_buf.tm_hour, tm_buf.tm_min, tm_buf.tm_sec)
                  << "[" << log_level_to_string(entry.level) << "] "
                  << "[" << config.service_id << "] ";
        
        if (!entry.correlation_id.empty()) {
            std::cerr << "[" << entry.correlation_id << "] ";
        }
        
        std::cerr << entry.message;
        
        if (!entry.fields.empty()) {
            std::cerr << " {";
            bool first = true;
            for (const auto& [k, v] : entry.fields) {
                if (!first) std::cerr << ", ";
                std::cerr << k << "=" << v;
                first = false;
            }
            std::cerr << "}";
        }
        
        std::cerr << std::endl;
    }
};

// ============================================================================
// LoggingClient Implementation
// ============================================================================

LoggingClient::LoggingClient(const LoggingClientConfig& config)
    : impl_(std::make_unique<Impl>(config)) {}

LoggingClient::~LoggingClient() = default;

LoggingClient::LoggingClient(LoggingClient&&) noexcept = default;
LoggingClient& LoggingClient::operator=(LoggingClient&&) noexcept = default;

void LoggingClient::debug(std::string_view message,
                          const std::map<std::string, std::string>& fields) {
    log(LogLevel::DEBUG, message, "", fields);
}

void LoggingClient::info(std::string_view message,
                         const std::map<std::string, std::string>& fields) {
    log(LogLevel::INFO, message, "", fields);
}

void LoggingClient::warn(std::string_view message,
                         const std::map<std::string, std::string>& fields) {
    log(LogLevel::WARN, message, "", fields);
}

void LoggingClient::error(std::string_view message,
                          const std::map<std::string, std::string>& fields) {
    log(LogLevel::ERROR, message, "", fields);
}

void LoggingClient::fatal(std::string_view message,
                          const std::map<std::string, std::string>& fields) {
    log(LogLevel::FATAL, message, "", fields);
}

void LoggingClient::log(LogLevel level,
                        std::string_view message,
                        std::string_view correlation_id,
                        const std::map<std::string, std::string>& fields) {
    // Filter by minimum level
    if (static_cast<int>(level) < static_cast<int>(impl_->config.min_level)) {
        return;
    }
    
    LogEntry entry;
    entry.level = level;
    entry.message = std::string(message);
    entry.correlation_id = std::string(correlation_id);
    entry.fields = fields;
    entry.timestamp = std::chrono::system_clock::now();
    
    impl_->enqueue(std::move(entry));
}

void LoggingClient::flush() {
    impl_->do_flush();
}

bool LoggingClient::is_connected() const {
    return impl_->connected;
}

size_t LoggingClient::pending_count() const {
    std::lock_guard<std::mutex> lock(impl_->buffer_mutex);
    return impl_->buffer.size();
}

size_t LoggingClient::dropped_count() const {
    return impl_->dropped_count;
}

// ============================================================================
// ScopedLogger Implementation
// ============================================================================

ScopedLogger::ScopedLogger(LoggingClient& client,
                           std::string_view operation,
                           std::string_view correlation_id,
                           const std::map<std::string, std::string>& fields)
    : client_(client)
    , operation_(operation)
    , correlation_id_(correlation_id)
    , fields_(fields)
    , start_time_(std::chrono::steady_clock::now()) {
    
    auto start_fields = fields_;
    start_fields["event"] = "start";
    client_.log(LogLevel::INFO, 
                std::format("Operation {} started", operation_),
                correlation_id_,
                start_fields);
}

ScopedLogger::~ScopedLogger() {
    auto end_time = std::chrono::steady_clock::now();
    auto duration = std::chrono::duration_cast<std::chrono::milliseconds>(
        end_time - start_time_);
    
    fields_["event"] = "end";
    fields_["duration_ms"] = std::to_string(duration.count());
    
    if (failed_) {
        fields_["status"] = "failed";
        if (!error_message_.empty()) {
            fields_["error"] = error_message_;
        }
        client_.log(LogLevel::ERROR,
                    std::format("Operation {} failed", operation_),
                    correlation_id_,
                    fields_);
    } else {
        fields_["status"] = "success";
        client_.log(LogLevel::INFO,
                    std::format("Operation {} completed", operation_),
                    correlation_id_,
                    fields_);
    }
}

void ScopedLogger::set_failed(std::string_view error_message) {
    failed_ = true;
    error_message_ = std::string(error_message);
}

void ScopedLogger::add_field(std::string_view key, std::string_view value) {
    fields_[std::string(key)] = std::string(value);
}

} // namespace crypto
