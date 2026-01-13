/**
 * @file health.cpp
 * @brief Health check implementation for Kubernetes and Linkerd
 * 
 * Requirements: 3.5
 */

#include "crypto/api/health.h"
#include <sstream>
#include <iomanip>
#include <openssl/crypto.h>

namespace crypto {

// ============================================================================
// HealthResponse Implementation
// ============================================================================

std::string HealthResponse::toJson() const {
    std::ostringstream ss;
    ss << "{\n";
    ss << "  \"status\": \"" << healthStatusToString(status) << "\",\n";
    ss << "  \"version\": \"" << version << "\",\n";
    
    // ISO 8601 timestamp
    auto time_t = std::chrono::system_clock::to_time_t(timestamp);
    ss << "  \"timestamp\": \"" << std::put_time(std::gmtime(&time_t), "%FT%TZ") << "\",\n";
    
    ss << "  \"components\": [\n";
    for (size_t i = 0; i < components.size(); ++i) {
        const auto& c = components[i];
        ss << "    {\n";
        ss << "      \"name\": \"" << c.name << "\",\n";
        ss << "      \"status\": \"" << healthStatusToString(c.status) << "\",\n";
        ss << "      \"message\": \"" << c.message << "\",\n";
        ss << "      \"latency_ms\": " << c.latency.count() << "\n";
        ss << "    }";
        if (i < components.size() - 1) ss << ",";
        ss << "\n";
    }
    ss << "  ]\n";
    ss << "}";
    
    return ss.str();
}

// ============================================================================
// ReadinessResponse Implementation
// ============================================================================

std::string ReadinessResponse::toJson() const {
    std::ostringstream ss;
    ss << "{\n";
    ss << "  \"ready\": " << (ready ? "true" : "false") << ",\n";
    ss << "  \"reason\": \"" << reason << "\"\n";
    ss << "}";
    return ss.str();
}

// ============================================================================
// HealthCheckManager Implementation
// ============================================================================

HealthCheckManager::HealthCheckManager(std::string version)
    : version_(std::move(version)) {}

void HealthCheckManager::registerCheck(const std::string& name, HealthCheckFn check) {
    std::lock_guard<std::mutex> lock(checks_mutex_);
    checks_[name] = std::move(check);
}

void HealthCheckManager::unregisterCheck(const std::string& name) {
    std::lock_guard<std::mutex> lock(checks_mutex_);
    checks_.erase(name);
}

HealthResponse HealthCheckManager::checkHealth() const {
    HealthResponse response;
    response.version = version_;
    response.timestamp = std::chrono::system_clock::now();
    response.status = HealthStatus::HEALTHY;
    
    // Check if shutting down
    if (shutting_down_.load()) {
        response.status = HealthStatus::UNHEALTHY;
        response.components.push_back({
            "service",
            HealthStatus::UNHEALTHY,
            "Service is shutting down",
            std::chrono::milliseconds{0}
        });
        return response;
    }
    
    // Run all registered health checks
    std::lock_guard<std::mutex> lock(checks_mutex_);
    for (const auto& [name, check] : checks_) {
        auto start = std::chrono::steady_clock::now();
        auto result = check();
        auto end = std::chrono::steady_clock::now();
        
        result.latency = std::chrono::duration_cast<std::chrono::milliseconds>(end - start);
        response.components.push_back(result);
        
        // Update overall status (worst status wins)
        if (result.status == HealthStatus::UNHEALTHY) {
            response.status = HealthStatus::UNHEALTHY;
        } else if (result.status == HealthStatus::DEGRADED && 
                   response.status == HealthStatus::HEALTHY) {
            response.status = HealthStatus::DEGRADED;
        }
    }
    
    return response;
}

ReadinessResponse HealthCheckManager::checkReadiness() const {
    ReadinessResponse response;
    
    // Not ready if shutting down
    if (shutting_down_.load()) {
        response.ready = false;
        response.reason = "Service is shutting down";
        return response;
    }
    
    // Not ready if explicitly marked not ready
    if (!ready_.load()) {
        response.ready = false;
        response.reason = "Service is not yet ready";
        return response;
    }
    
    // Check critical components
    auto health = checkHealth();
    if (health.status == HealthStatus::UNHEALTHY) {
        response.ready = false;
        response.reason = "Critical component unhealthy";
        return response;
    }
    
    response.ready = true;
    response.reason = "All systems operational";
    return response;
}

void HealthCheckManager::setReady(bool ready) {
    ready_.store(ready);
}

void HealthCheckManager::setShuttingDown(bool shutting_down) {
    shutting_down_.store(shutting_down);
    if (shutting_down) {
        ready_.store(false);
    }
}

// ============================================================================
// Default Health Checks
// ============================================================================

HealthCheckFn createOpenSSLHealthCheck() {
    return []() -> ComponentHealth {
        ComponentHealth result;
        result.name = "openssl";
        
        // Check OpenSSL is initialized
        if (OPENSSL_init_crypto(0, nullptr) == 1) {
            result.status = HealthStatus::HEALTHY;
            result.message = "OpenSSL initialized";
        } else {
            result.status = HealthStatus::UNHEALTHY;
            result.message = "OpenSSL initialization failed";
        }
        
        return result;
    };
}

HealthCheckFn createLoggingServiceHealthCheck(std::function<bool()> isConnected) {
    return [isConnected = std::move(isConnected)]() -> ComponentHealth {
        ComponentHealth result;
        result.name = "logging-service";
        
        if (isConnected()) {
            result.status = HealthStatus::HEALTHY;
            result.message = "Connected";
        } else {
            // Logging is non-critical - degraded, not unhealthy
            result.status = HealthStatus::DEGRADED;
            result.message = "Disconnected - using local fallback";
        }
        
        return result;
    };
}

HealthCheckFn createCacheServiceHealthCheck(std::function<bool()> isConnected) {
    return [isConnected = std::move(isConnected)]() -> ComponentHealth {
        ComponentHealth result;
        result.name = "cache-service";
        
        if (isConnected()) {
            result.status = HealthStatus::HEALTHY;
            result.message = "Connected";
        } else {
            // Cache is non-critical - degraded, not unhealthy
            result.status = HealthStatus::DEGRADED;
            result.message = "Disconnected - using local fallback";
        }
        
        return result;
    };
}

// ============================================================================
// Global Health Manager
// ============================================================================

static std::unique_ptr<HealthCheckManager> g_health_manager;
static std::once_flag g_health_init;

HealthCheckManager& getHealthManager() {
    std::call_once(g_health_init, []() {
        g_health_manager = std::make_unique<HealthCheckManager>();
        // Register default checks
        g_health_manager->registerCheck("openssl", createOpenSSLHealthCheck());
    });
    return *g_health_manager;
}

} // namespace crypto
