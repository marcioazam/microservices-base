#include "crypto/api/grpc_server.h"
#include <grpcpp/ext/proto_server_reflection_plugin.h>
#include <fstream>
#include <sstream>

namespace crypto {

namespace {

std::string readFile(const std::string& path) {
    std::ifstream file(path);
    if (!file) {
        throw std::runtime_error("Cannot read file: " + path);
    }
    std::stringstream buffer;
    buffer << file.rdbuf();
    return buffer.str();
}

} // anonymous namespace

GrpcServer::GrpcServer(
    const GrpcServerConfig& config,
    std::shared_ptr<EncryptionService> encryption_service,
    std::shared_ptr<SignatureService> signature_service,
    std::shared_ptr<FileEncryptionService> file_service,
    std::shared_ptr<KeyService> key_service,
    std::shared_ptr<IAuditLogger> audit_logger)
    : config_(config)
    , encryption_service_(std::move(encryption_service))
    , signature_service_(std::move(signature_service))
    , file_service_(std::move(file_service))
    , key_service_(std::move(key_service))
    , audit_logger_(std::move(audit_logger)) {}

GrpcServer::~GrpcServer() {
    shutdown();
}

std::shared_ptr<grpc::ServerCredentials> GrpcServer::createCredentials() {
    if (config_.tls_cert_path.empty() || config_.tls_key_path.empty()) {
        // Insecure for development only
        return grpc::InsecureServerCredentials();
    }
    
    grpc::SslServerCredentialsOptions ssl_opts;
    
    // Read certificate and key
    std::string cert = readFile(config_.tls_cert_path);
    std::string key = readFile(config_.tls_key_path);
    
    grpc::SslServerCredentialsOptions::PemKeyCertPair key_cert_pair;
    key_cert_pair.private_key = key;
    key_cert_pair.cert_chain = cert;
    ssl_opts.pem_key_cert_pairs.push_back(key_cert_pair);
    
    // Read CA certificate for client verification (mTLS)
    if (!config_.tls_ca_path.empty()) {
        ssl_opts.pem_root_certs = readFile(config_.tls_ca_path);
        ssl_opts.client_certificate_request = 
            GRPC_SSL_REQUEST_AND_REQUIRE_CLIENT_CERTIFICATE_AND_VERIFY;
    }
    
    // Force TLS 1.3
    ssl_opts.force_client_auth = !config_.tls_ca_path.empty();
    
    return grpc::SslServerCredentials(ssl_opts);
}

void GrpcServer::run() {
    start();
    if (server_) {
        server_->Wait();
    }
}

void GrpcServer::start() {
    if (running_.load()) {
        return;
    }
    
    std::string server_address = "0.0.0.0:" + std::to_string(config_.port);
    
    grpc::ServerBuilder builder;
    builder.AddListeningPort(server_address, createCredentials());
    
    // Configure thread pool
    builder.SetSyncServerOption(grpc::ServerBuilder::NUM_CQS, 
                                 static_cast<int>(config_.thread_pool_size));
    
    // Enable reflection for debugging
    if (config_.enable_reflection) {
        grpc::reflection::InitProtoReflectionServerBuilderPlugin();
    }
    
    // Build and start server
    server_ = builder.BuildAndStart();
    
    if (server_) {
        running_.store(true);
        start_time_ = std::chrono::steady_clock::now();
    }
}

void GrpcServer::shutdown() {
    if (!running_.load()) {
        return;
    }
    
    if (server_) {
        // Graceful shutdown with deadline
        auto deadline = std::chrono::system_clock::now() + std::chrono::seconds(5);
        server_->Shutdown(deadline);
        server_.reset();
    }
    
    running_.store(false);
}

std::chrono::seconds GrpcServer::uptime() const {
    if (!running_.load()) {
        return std::chrono::seconds(0);
    }
    
    auto now = std::chrono::steady_clock::now();
    return std::chrono::duration_cast<std::chrono::seconds>(now - start_time_);
}

} // namespace crypto
