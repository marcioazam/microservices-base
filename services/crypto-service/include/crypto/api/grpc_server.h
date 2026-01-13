#pragma once

#include "crypto/services/encryption_service.h"
#include "crypto/services/signature_service.h"
#include "crypto/services/file_encryption_service.h"
#include "crypto/keys/key_service.h"
#include "crypto/audit/audit_logger.h"
#include <grpcpp/grpcpp.h>
#include <memory>
#include <string>
#include <atomic>
#include <chrono>

namespace crypto {

// Server configuration
struct GrpcServerConfig {
    uint16_t port = 50051;
    std::string tls_cert_path;
    std::string tls_key_path;
    std::string tls_ca_path;
    size_t thread_pool_size = 4;
    bool enable_reflection = true;
};

// gRPC server implementation
class GrpcServer {
public:
    GrpcServer(const GrpcServerConfig& config,
               std::shared_ptr<EncryptionService> encryption_service,
               std::shared_ptr<SignatureService> signature_service,
               std::shared_ptr<FileEncryptionService> file_service,
               std::shared_ptr<KeyService> key_service,
               std::shared_ptr<IAuditLogger> audit_logger);
    
    ~GrpcServer();
    
    // Start the server (blocking)
    void run();
    
    // Start the server (non-blocking)
    void start();
    
    // Stop the server gracefully
    void shutdown();
    
    // Check if server is running
    bool isRunning() const { return running_.load(); }
    
    // Get server uptime
    std::chrono::seconds uptime() const;

private:
    GrpcServerConfig config_;
    std::shared_ptr<EncryptionService> encryption_service_;
    std::shared_ptr<SignatureService> signature_service_;
    std::shared_ptr<FileEncryptionService> file_service_;
    std::shared_ptr<KeyService> key_service_;
    std::shared_ptr<IAuditLogger> audit_logger_;
    
    std::unique_ptr<grpc::Server> server_;
    std::atomic<bool> running_{false};
    std::chrono::steady_clock::time_point start_time_;
    
    std::shared_ptr<grpc::ServerCredentials> createCredentials();
};

} // namespace crypto
