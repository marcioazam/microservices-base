/**
 * @file health.h
 * @brief Health check endpoints for Kubernetes and Linkerd integration
 * 
 * Provides /health and /ready endpoints compatible with:
 * - Kubernetes liveness and readiness probes
 * - Linkerd service mesh health checks
 * 
 * Requirements: 3.5
 */

#pragma once

#include <string>
#include <chrono>
#include <atomic>
#include <functional>
#include <vector>
#include <map>

namespace crypto {

// ============================================================================
// Health Status Types
// ============================================================================

/**
 * @brief Health check status
 */
enum class HealthStatus {
    HEALTHY,
    DEGRADED,
    UNHEALTHY
};

/**
 * @brief Convert health status to string
 */
[[nodiscard]] constexpr const char* healthStatusToString(HealthStatus status) noexcept {
    switch (status) {
        case HealthStatus::HEALTHY: return "healthy";
        case HealthStatus::DEGRADED: return "degraded";
        case HealthStatus::UNHEALTHY: return "unhealthy";
    }
    return "unknown";
}

/**
 * @brief Individual component health check result
 */
struct ComponentHealth {
    std::string name;
    HealthStatus status;
    std::string message;
    std::chrono::milliseconds latency{0};
};

/**
 * @brief Overall health check response
 */
struct HealthResponse {
    HealthStatus status;
    std::string version;
    std::chrono::system_clock::time_point timestamp;
    std::vector<ComponentHealth> components;
    
    /**
     * @brief Serialize to JSON for HTTP response
     */
    [[nodiscard]] std::string toJson() const;
    
    /**
     * @brief Get HTTP status code based on health status
     */
    [[nodiscard]] int httpStatusCode() const {
        switch (status) {
            case HealthStatus::HEALTHY: return 200;
            case HealthStatus::DEGRADED: return 200;  // Still serving
            case HealthStatus::UNHEALTHY: return 503;
        }
        return 500;
    }
};

/**
 * @brief Readiness check response
 */
struct ReadinessResponse {
    bool ready;
    std::string reason;
    
    [[nodiscard]] std::string toJson() const;
    [[nodiscard]] int httpStatusCode() const { return ready ? 200 : 503; }
};

// ============================================================================
// Health Check Manager
// ============================================================================

/**
 * @brief Health check callback type
 */
using HealthCheckFn = std::function<ComponentHealth()>;

/**
 * @brief Manages health checks for the crypto-service
 * 
 * Integrates with:
 * - Kubernetes liveness probe (/health)
 * - Kubernetes readiness probe (/ready)
 * - Linkerd health checks
 */
class HealthCheckManager {
public:
    explicit HealthCheckManager(std::string version = "1.0.0");
    ~HealthCheckManager() = default;
    
    // ========================================================================
    // Component Registration
    // ========================================================================
    
    /**
     * @brief Register a health check for a component
     * @param name Component name
     * @param check Health check function
     */
    void registerCheck(const std::string& name, HealthCheckFn check);
    
    /**
     * @brief Unregister a health check
     * @param name Component name
     */
    void unregisterCheck(const std::string& name);
    
    // ========================================================================
    // Health Endpoints (Requirement 3.5)
    // ========================================================================
    
    /**
     * @brief Liveness check - is the service alive?
     * 
     * Used by Kubernetes liveness probe and Linkerd.
     * Returns healthy if the service is running, even if dependencies are down.
     */
    [[nodiscard]] HealthResponse checkHealth() const;
    
    /**
     * @brief Readiness check - is the service ready to receive traffic?
     * 
     * Used by Kubernetes readiness probe.
     * Returns ready only if all critical dependencies are available.
     */
    [[nodiscard]] ReadinessResponse checkReadiness() const;
    
    // ========================================================================
    // Service State
    // ========================================================================
    
    /**
     * @brief Mark service as ready to receive traffic
     */
    void setReady(bool ready);
    
    /**
     * @brief Check if service is marked as ready
     */
    [[nodiscard]] bool isReady() const { return ready_.load(); }
    
    /**
     * @brief Mark service as shutting down (for graceful shutdown)
     */
    void setShuttingDown(bool shutting_down);
    
    /**
     * @brief Check if service is shutting down
     */
    [[nodiscard]] bool isShuttingDown() const { return shutting_down_.load(); }

private:
    std::string version_;
    std::atomic<bool> ready_{false};
    std::atomic<bool> shutting_down_{false};
    mutable std::mutex checks_mutex_;
    std::map<std::string, HealthCheckFn> checks_;
};

// ============================================================================
// Default Health Checks
// ============================================================================

/**
 * @brief Create health check for OpenSSL
 */
[[nodiscard]] HealthCheckFn createOpenSSLHealthCheck();

/**
 * @brief Create health check for logging service connection
 */
[[nodiscard]] HealthCheckFn createLoggingServiceHealthCheck(
    std::function<bool()> isConnected);

/**
 * @brief Create health check for cache service connection
 */
[[nodiscard]] HealthCheckFn createCacheServiceHealthCheck(
    std::function<bool()> isConnected);

// ============================================================================
// Global Health Manager
// ============================================================================

/**
 * @brief Get the global health check manager
 */
HealthCheckManager& getHealthManager();

} // namespace crypto
