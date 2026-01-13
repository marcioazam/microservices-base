#include "crypto/api/health_check.h"
#include <algorithm>

namespace crypto {

HealthChecker::HealthChecker()
    : start_time_(std::chrono::steady_clock::now()) {}

void HealthChecker::registerCheck(const std::string& name, 
                                   HealthCheckCallback callback) {
    checks_.emplace_back(name, std::move(callback));
}

void HealthChecker::removeCheck(const std::string& name) {
    checks_.erase(
        std::remove_if(checks_.begin(), checks_.end(),
            [&name](const auto& pair) { return pair.first == name; }),
        checks_.end());
}

HealthResponse HealthChecker::check() {
    HealthResponse response;
    response.version = version_;
    response.uptime = uptime();
    response.hsm_connected = false;
    response.kms_connected = false;
    
    for (const auto& [name, callback] : checks_) {
        auto start = std::chrono::steady_clock::now();
        auto component = callback();
        auto end = std::chrono::steady_clock::now();
        
        component.name = name;
        component.latency = std::chrono::duration_cast<std::chrono::milliseconds>(
            end - start);
        
        response.components.push_back(component);
        
        // Track HSM/KMS status
        if (name == "hsm") {
            response.hsm_connected = (component.status == HealthStatus::HEALTHY);
        } else if (name == "kms") {
            response.kms_connected = (component.status == HealthStatus::HEALTHY);
        }
    }
    
    response.status = aggregateStatus(response.components);
    return response;
}

std::chrono::seconds HealthChecker::uptime() const {
    auto now = std::chrono::steady_clock::now();
    return std::chrono::duration_cast<std::chrono::seconds>(now - start_time_);
}

HealthStatus HealthChecker::aggregateStatus(
    const std::vector<ComponentHealth>& components) {
    
    if (components.empty()) {
        return HealthStatus::UNKNOWN;
    }
    
    bool has_unhealthy = false;
    bool has_degraded = false;
    
    for (const auto& component : components) {
        switch (component.status) {
            case HealthStatus::UNHEALTHY:
                has_unhealthy = true;
                break;
            case HealthStatus::DEGRADED:
                has_degraded = true;
                break;
            default:
                break;
        }
    }
    
    if (has_unhealthy) {
        return HealthStatus::UNHEALTHY;
    }
    if (has_degraded) {
        return HealthStatus::DEGRADED;
    }
    return HealthStatus::HEALTHY;
}

namespace HealthChecks {

HealthCheckCallback hsmCheck(std::function<bool()> ping_fn) {
    return [ping_fn]() -> ComponentHealth {
        ComponentHealth health;
        try {
            if (ping_fn()) {
                health.status = HealthStatus::HEALTHY;
                health.message = "HSM connected";
            } else {
                health.status = HealthStatus::UNHEALTHY;
                health.message = "HSM not responding";
            }
        } catch (const std::exception& e) {
            health.status = HealthStatus::UNHEALTHY;
            health.message = std::string("HSM error: ") + e.what();
        }
        return health;
    };
}

HealthCheckCallback kmsCheck(std::function<bool()> ping_fn) {
    return [ping_fn]() -> ComponentHealth {
        ComponentHealth health;
        try {
            if (ping_fn()) {
                health.status = HealthStatus::HEALTHY;
                health.message = "KMS connected";
            } else {
                health.status = HealthStatus::DEGRADED;
                health.message = "KMS not responding, using cache";
            }
        } catch (const std::exception& e) {
            health.status = HealthStatus::DEGRADED;
            health.message = std::string("KMS error: ") + e.what();
        }
        return health;
    };
}

HealthCheckCallback keyStoreCheck(std::function<bool()> ping_fn) {
    return [ping_fn]() -> ComponentHealth {
        ComponentHealth health;
        try {
            if (ping_fn()) {
                health.status = HealthStatus::HEALTHY;
                health.message = "Key store operational";
            } else {
                health.status = HealthStatus::UNHEALTHY;
                health.message = "Key store unavailable";
            }
        } catch (const std::exception& e) {
            health.status = HealthStatus::UNHEALTHY;
            health.message = std::string("Key store error: ") + e.what();
        }
        return health;
    };
}

HealthCheckCallback auditLoggerCheck(std::function<bool()> ping_fn) {
    return [ping_fn]() -> ComponentHealth {
        ComponentHealth health;
        try {
            if (ping_fn()) {
                health.status = HealthStatus::HEALTHY;
                health.message = "Audit logger operational";
            } else {
                health.status = HealthStatus::DEGRADED;
                health.message = "Audit logger degraded";
            }
        } catch (const std::exception& e) {
            health.status = HealthStatus::DEGRADED;
            health.message = std::string("Audit logger error: ") + e.what();
        }
        return health;
    };
}

} // namespace HealthChecks

} // namespace crypto
