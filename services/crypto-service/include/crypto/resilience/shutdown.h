#pragma once

#include <atomic>
#include <chrono>
#include <condition_variable>
#include <functional>
#include <mutex>
#include <vector>
#include <csignal>

namespace crypto {

// Shutdown handler for graceful termination
class ShutdownHandler {
public:
    using ShutdownCallback = std::function<void()>;
    
    static ShutdownHandler& instance();
    
    // Register shutdown callback
    void onShutdown(ShutdownCallback callback);
    
    // Check if shutdown is requested
    bool isShutdownRequested() const { return shutdown_requested_.load(); }
    
    // Request shutdown
    void requestShutdown();
    
    // Wait for shutdown signal
    void waitForShutdown();
    
    // Wait for shutdown with timeout
    bool waitForShutdown(std::chrono::milliseconds timeout);
    
    // Install signal handlers (SIGTERM, SIGINT)
    void installSignalHandlers();
    
    // Set shutdown timeout
    void setShutdownTimeout(std::chrono::seconds timeout) {
        shutdown_timeout_ = timeout;
    }

private:
    ShutdownHandler() = default;
    
    std::atomic<bool> shutdown_requested_{false};
    std::vector<ShutdownCallback> callbacks_;
    std::mutex mutex_;
    std::condition_variable cv_;
    std::chrono::seconds shutdown_timeout_{30};
    
    void executeCallbacks();
    static void signalHandler(int signal);
};

// RAII guard for tracking in-flight requests
class InFlightGuard {
public:
    InFlightGuard();
    ~InFlightGuard();
    
    // Check if new requests should be accepted
    static bool acceptingRequests();
    
    // Wait for all in-flight requests to complete
    static bool waitForDrain(std::chrono::milliseconds timeout);
    
    // Get current in-flight count
    static uint32_t count();

private:
    static std::atomic<uint32_t> in_flight_count_;
    static std::atomic<bool> accepting_;
    static std::mutex drain_mutex_;
    static std::condition_variable drain_cv_;
};

// Graceful shutdown coordinator
class GracefulShutdown {
public:
    explicit GracefulShutdown(std::chrono::seconds timeout = std::chrono::seconds(30));
    ~GracefulShutdown();
    
    // Start shutdown process
    void shutdown();
    
    // Add component to shutdown
    void addComponent(const std::string& name, std::function<void()> shutdown_fn);
    
    // Check if shutdown is in progress
    bool isShuttingDown() const { return shutting_down_.load(); }

private:
    std::chrono::seconds timeout_;
    std::atomic<bool> shutting_down_{false};
    std::vector<std::pair<std::string, std::function<void()>>> components_;
    std::mutex mutex_;
};

} // namespace crypto
