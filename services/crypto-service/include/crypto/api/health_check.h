#pragma once

#include <string>
#include <chrono>
#include <atomic>
#include <functional>
#include <vector>

namespace crypto {

// Health status
enum class HealthStatus {
    UNKNOWN,
    HEALTHY,
    DEGRADED,
    UNHEALTHY
};

// Component health
struct ComponentHealth {
    std::string name;
    HealthStatus status;
    std::string message;
    std::chrono::milliseconds latency{0};
};

// Overall health response
struct HealthResponse {
    HealthStatus status;
    std::string version;
    std::chrono::seconds uptime;
    std::vector<ComponentHealth> components;
    bool hsm_connected;
    bool kms_connected;
};

// Health check callback type
using HealthCheckCallback = std::function<ComponentHealth()>;

// Health checker
class HealthChecker {
public:
    HealthChecker();
    ~HealthChecker() = default;
    
    // Register a health check
    void registerCheck(const std::string& name, HealthCheckCallback callback);
    
    // Remove a health check
    void removeCheck(const std::string& name);
    
    // Run all health checks
    HealthResponse check();
    
    // Set version
    void setVersion(const std::string& version) { version_ = version; }
    
    // Get uptime
    std::chrono::seconds uptime() const;

private:
    std::vector<std::pair<std::string, HealthCheckCallback>> checks_;
    std::chrono::steady_clock::time_point start_time_;
    std::string version_ = "1.0.0";
    
    HealthStatus aggregateStatus(const std::vector<ComponentHealth>& components);
};

// Common health check implementations
namespace HealthChecks {
    // Check HSM connectivity
    HealthCheckCallback hsmCheck(std::function<bool()> ping_fn);
    
    // Check KMS connectivity
    HealthCheckCallback kmsCheck(std::function<bool()> ping_fn);
    
    // Check key store
    HealthCheckCallback keyStoreCheck(std::function<bool()> ping_fn);
    
    // Check audit logger
    HealthCheckCallback auditLoggerCheck(std::function<bool()> ping_fn);
}

} // namespace crypto
