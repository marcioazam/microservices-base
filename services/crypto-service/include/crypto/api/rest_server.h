#pragma once

#include "crypto/services/encryption_service.h"
#include "crypto/services/signature_service.h"
#include "crypto/services/file_encryption_service.h"
#include "crypto/keys/key_service.h"
#include "crypto/audit/audit_logger.h"
#include "crypto/auth/jwt_validator.h"
#include "crypto/auth/rbac_engine.h"
#include "crypto/api/health_check.h"
#include <memory>
#include <string>
#include <atomic>
#include <thread>

namespace crypto {

// REST server configuration
struct RestServerConfig {
    uint16_t port = 8080;
    std::string tls_cert_path;
    std::string tls_key_path;
    std::string tls_ca_path;
    size_t thread_pool_size = 4;
    size_t max_request_size = 10 * 1024 * 1024;  // 10MB
};

// REST server implementation
class RestServer {
public:
    RestServer(const RestServerConfig& config,
               std::shared_ptr<EncryptionService> encryption_service,
               std::shared_ptr<SignatureService> signature_service,
               std::shared_ptr<FileEncryptionService> file_service,
               std::shared_ptr<KeyService> key_service,
               std::shared_ptr<IAuditLogger> audit_logger,
               std::shared_ptr<IJWTValidator> jwt_validator,
               std::shared_ptr<RBACEngine> rbac_engine);
    
    ~RestServer();
    
    // Start the server (blocking)
    void run();
    
    // Start the server (non-blocking)
    void start();
    
    // Stop the server gracefully
    void shutdown();
    
    // Check if server is running
    bool isRunning() const { return running_.load(); }

private:
    RestServerConfig config_;
    std::shared_ptr<EncryptionService> encryption_service_;
    std::shared_ptr<SignatureService> signature_service_;
    std::shared_ptr<FileEncryptionService> file_service_;
    std::shared_ptr<KeyService> key_service_;
    std::shared_ptr<IAuditLogger> audit_logger_;
    std::shared_ptr<IJWTValidator> jwt_validator_;
    std::shared_ptr<RBACEngine> rbac_engine_;
    std::unique_ptr<HealthChecker> health_checker_;
    
    std::atomic<bool> running_{false};
    std::unique_ptr<std::thread> server_thread_;
    
    void setupRoutes();
    void runInternal();
};

} // namespace crypto
