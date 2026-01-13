#include "crypto/resilience/shutdown.h"
#include <iostream>

namespace crypto {

// ShutdownHandler
ShutdownHandler& ShutdownHandler::instance() {
    static ShutdownHandler handler;
    return handler;
}

void ShutdownHandler::onShutdown(ShutdownCallback callback) {
    std::lock_guard<std::mutex> lock(mutex_);
    callbacks_.push_back(std::move(callback));
}

void ShutdownHandler::requestShutdown() {
    bool expected = false;
    if (shutdown_requested_.compare_exchange_strong(expected, true)) {
        executeCallbacks();
        cv_.notify_all();
    }
}

void ShutdownHandler::waitForShutdown() {
    std::unique_lock<std::mutex> lock(mutex_);
    cv_.wait(lock, [this] { return shutdown_requested_.load(); });
}

bool ShutdownHandler::waitForShutdown(std::chrono::milliseconds timeout) {
    std::unique_lock<std::mutex> lock(mutex_);
    return cv_.wait_for(lock, timeout, [this] { return shutdown_requested_.load(); });
}

void ShutdownHandler::installSignalHandlers() {
    std::signal(SIGTERM, signalHandler);
    std::signal(SIGINT, signalHandler);
}

void ShutdownHandler::executeCallbacks() {
    std::vector<ShutdownCallback> callbacks_copy;
    {
        std::lock_guard<std::mutex> lock(mutex_);
        callbacks_copy = callbacks_;
    }
    
    for (const auto& callback : callbacks_copy) {
        try {
            callback();
        } catch (const std::exception& e) {
            std::cerr << "Shutdown callback error: " << e.what() << std::endl;
        }
    }
}

void ShutdownHandler::signalHandler(int signal) {
    (void)signal;
    instance().requestShutdown();
}

// InFlightGuard
std::atomic<uint32_t> InFlightGuard::in_flight_count_{0};
std::atomic<bool> InFlightGuard::accepting_{true};
std::mutex InFlightGuard::drain_mutex_;
std::condition_variable InFlightGuard::drain_cv_;

InFlightGuard::InFlightGuard() {
    in_flight_count_.fetch_add(1);
}

InFlightGuard::~InFlightGuard() {
    uint32_t count = in_flight_count_.fetch_sub(1) - 1;
    if (count == 0 && !accepting_.load()) {
        drain_cv_.notify_all();
    }
}

bool InFlightGuard::acceptingRequests() {
    return accepting_.load();
}

bool InFlightGuard::waitForDrain(std::chrono::milliseconds timeout) {
    accepting_.store(false);
    
    std::unique_lock<std::mutex> lock(drain_mutex_);
    return drain_cv_.wait_for(lock, timeout, [] {
        return in_flight_count_.load() == 0;
    });
}

uint32_t InFlightGuard::count() {
    return in_flight_count_.load();
}

// GracefulShutdown
GracefulShutdown::GracefulShutdown(std::chrono::seconds timeout)
    : timeout_(timeout) {
    ShutdownHandler::instance().onShutdown([this] {
        shutdown();
    });
}

GracefulShutdown::~GracefulShutdown() {
    if (!shutting_down_.load()) {
        shutdown();
    }
}

void GracefulShutdown::shutdown() {
    bool expected = false;
    if (!shutting_down_.compare_exchange_strong(expected, true)) {
        return;  // Already shutting down
    }
    
    std::cout << "Starting graceful shutdown..." << std::endl;
    
    // Stop accepting new requests
    InFlightGuard::waitForDrain(
        std::chrono::duration_cast<std::chrono::milliseconds>(timeout_));
    
    // Shutdown components in reverse order
    std::vector<std::pair<std::string, std::function<void()>>> components_copy;
    {
        std::lock_guard<std::mutex> lock(mutex_);
        components_copy = components_;
    }
    
    for (auto it = components_copy.rbegin(); it != components_copy.rend(); ++it) {
        std::cout << "Shutting down: " << it->first << std::endl;
        try {
            it->second();
        } catch (const std::exception& e) {
            std::cerr << "Error shutting down " << it->first 
                      << ": " << e.what() << std::endl;
        }
    }
    
    std::cout << "Graceful shutdown complete" << std::endl;
}

void GracefulShutdown::addComponent(const std::string& name,
                                     std::function<void()> shutdown_fn) {
    std::lock_guard<std::mutex> lock(mutex_);
    components_.emplace_back(name, std::move(shutdown_fn));
}

} // namespace crypto
