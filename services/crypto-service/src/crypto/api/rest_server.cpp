#include "crypto/api/rest_server.h"
#include <nlohmann/json.hpp>
#include <httplib.h>
#include <sstream>

namespace crypto {

using json = nlohmann::json;

namespace {

// Helper to create JSON error response
json errorResponse(const std::string& code, const std::string& message,
                   const std::string& correlation_id = "") {
    return {
        {"error", {
            {"code", code},
            {"message", message},
            {"correlation_id", correlation_id}
        }}
    };
}

// Helper to extract correlation ID from request
std::string getCorrelationId(const httplib::Request& req) {
    auto it = req.headers.find("X-Correlation-ID");
    if (it != req.headers.end()) {
        return it->second;
    }
    return "";
}

// Base64 encode/decode helpers
std::string base64Encode(const std::vector<uint8_t>& data) {
    static const char* chars = 
        "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
    
    std::string result;
    int val = 0, valb = -6;
    
    for (uint8_t c : data) {
        val = (val << 8) + c;
        valb += 8;
        while (valb >= 0) {
            result.push_back(chars[(val >> valb) & 0x3F]);
            valb -= 6;
        }
    }
    
    if (valb > -6) {
        result.push_back(chars[((val << 8) >> (valb + 8)) & 0x3F]);
    }
    
    while (result.size() % 4) {
        result.push_back('=');
    }
    
    return result;
}

std::vector<uint8_t> base64Decode(const std::string& input) {
    static const int lookup[] = {
        -1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,
        -1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,
        -1,-1,-1,-1,-1,-1,-1,-1,-1,-1,-1,62,-1,-1,-1,63,
        52,53,54,55,56,57,58,59,60,61,-1,-1,-1,-1,-1,-1,
        -1, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9,10,11,12,13,14,
        15,16,17,18,19,20,21,22,23,24,25,-1,-1,-1,-1,-1,
        -1,26,27,28,29,30,31,32,33,34,35,36,37,38,39,40,
        41,42,43,44,45,46,47,48,49,50,51,-1,-1,-1,-1,-1
    };
    
    std::vector<uint8_t> result;
    int val = 0, valb = -8;
    
    for (char c : input) {
        if (c == '=') break;
        if (lookup[static_cast<unsigned char>(c)] == -1) continue;
        
        val = (val << 6) + lookup[static_cast<unsigned char>(c)];
        valb += 6;
        
        if (valb >= 0) {
            result.push_back(static_cast<uint8_t>((val >> valb) & 0xFF));
            valb -= 8;
        }
    }
    
    return result;
}

} // anonymous namespace

RestServer::RestServer(
    const RestServerConfig& config,
    std::shared_ptr<EncryptionService> encryption_service,
    std::shared_ptr<SignatureService> signature_service,
    std::shared_ptr<FileEncryptionService> file_service,
    std::shared_ptr<KeyService> key_service,
    std::shared_ptr<IAuditLogger> audit_logger,
    std::shared_ptr<IJWTValidator> jwt_validator,
    std::shared_ptr<RBACEngine> rbac_engine)
    : config_(config)
    , encryption_service_(std::move(encryption_service))
    , signature_service_(std::move(signature_service))
    , file_service_(std::move(file_service))
    , key_service_(std::move(key_service))
    , audit_logger_(std::move(audit_logger))
    , jwt_validator_(std::move(jwt_validator))
    , rbac_engine_(std::move(rbac_engine))
    , health_checker_(std::make_unique<HealthChecker>()) {}

RestServer::~RestServer() {
    shutdown();
}

void RestServer::run() {
    runInternal();
}

void RestServer::start() {
    if (running_.load()) {
        return;
    }
    
    running_.store(true);
    server_thread_ = std::make_unique<std::thread>([this]() {
        runInternal();
    });
}

void RestServer::shutdown() {
    running_.store(false);
    if (server_thread_ && server_thread_->joinable()) {
        server_thread_->join();
    }
}

void RestServer::runInternal() {
    httplib::SSLServer* ssl_server = nullptr;
    httplib::Server* server = nullptr;
    
    std::unique_ptr<httplib::SSLServer> ssl_ptr;
    std::unique_ptr<httplib::Server> plain_ptr;
    
    if (!config_.tls_cert_path.empty() && !config_.tls_key_path.empty()) {
        ssl_ptr = std::make_unique<httplib::SSLServer>(
            config_.tls_cert_path.c_str(),
            config_.tls_key_path.c_str());
        ssl_server = ssl_ptr.get();
        server = ssl_server;
    } else {
        plain_ptr = std::make_unique<httplib::Server>();
        server = plain_ptr.get();
    }
    
    // Health endpoint
    server->Get("/health", [this](const httplib::Request&, httplib::Response& res) {
        auto health = health_checker_->check();
        json response = {
            {"status", health.status == HealthStatus::HEALTHY ? "SERVING" : "NOT_SERVING"},
            {"hsm_connected", health.hsm_connected},
            {"kms_connected", health.kms_connected},
            {"version", health.version},
            {"uptime_seconds", health.uptime.count()}
        };
        res.set_content(response.dump(), "application/json");
    });
    
    // Metrics endpoint (placeholder)
    server->Get("/metrics", [](const httplib::Request&, httplib::Response& res) {
        res.set_content("# Prometheus metrics\n", "text/plain");
    });

    // Encrypt endpoint
    server->Post("/v1/encrypt", [this](const httplib::Request& req, 
                                        httplib::Response& res) {
        auto correlation_id = getCorrelationId(req);
        
        try {
            auto body = json::parse(req.body);
            
            auto plaintext = base64Decode(body["plaintext"].get<std::string>());
            auto key_id_result = KeyId::parse(body["key_id"].get<std::string>());
            
            if (!key_id_result) {
                res.status = 400;
                res.set_content(errorResponse("INVALID_KEY_ID", 
                    "Invalid key ID format", correlation_id).dump(), 
                    "application/json");
                return;
            }
            
            EncryptionContext ctx{
                .correlation_id = correlation_id,
                .caller_identity = "rest-client",
                .caller_service = "rest-api",
                .source_ip = req.remote_addr
            };
            
            auto result = encryption_service_->encrypt(plaintext, *key_id_result, ctx);
            
            if (!result) {
                res.status = 400;
                res.set_content(errorResponse(
                    std::to_string(static_cast<int>(result.error().code)),
                    result.error().message, correlation_id).dump(),
                    "application/json");
                return;
            }
            
            json response = {
                {"ciphertext", base64Encode(result->ciphertext)},
                {"iv", base64Encode(result->iv)},
                {"tag", base64Encode(result->tag)},
                {"key_id", result->key_id.toString()},
                {"algorithm", result->algorithm}
            };
            
            res.set_content(response.dump(), "application/json");
            
        } catch (const std::exception& e) {
            res.status = 400;
            res.set_content(errorResponse("INVALID_REQUEST", e.what(), 
                correlation_id).dump(), "application/json");
        }
    });
    
    // Decrypt endpoint
    server->Post("/v1/decrypt", [this](const httplib::Request& req,
                                        httplib::Response& res) {
        auto correlation_id = getCorrelationId(req);
        
        try {
            auto body = json::parse(req.body);
            
            auto ciphertext = base64Decode(body["ciphertext"].get<std::string>());
            auto iv = base64Decode(body["iv"].get<std::string>());
            auto tag = base64Decode(body["tag"].get<std::string>());
            auto key_id_result = KeyId::parse(body["key_id"].get<std::string>());
            
            if (!key_id_result) {
                res.status = 400;
                res.set_content(errorResponse("INVALID_KEY_ID",
                    "Invalid key ID format", correlation_id).dump(),
                    "application/json");
                return;
            }
            
            DecryptionRequest decrypt_req{
                .ciphertext = ciphertext,
                .iv = iv,
                .tag = tag,
                .key_id = *key_id_result
            };
            
            EncryptionContext ctx{
                .correlation_id = correlation_id,
                .caller_identity = "rest-client",
                .caller_service = "rest-api",
                .source_ip = req.remote_addr
            };
            
            auto result = encryption_service_->decrypt(decrypt_req, ctx);
            
            if (!result) {
                res.status = 400;
                res.set_content(errorResponse("DECRYPTION_FAILED",
                    result.error().message, correlation_id).dump(),
                    "application/json");
                return;
            }
            
            json response = {
                {"plaintext", base64Encode(*result)}
            };
            
            res.set_content(response.dump(), "application/json");
            
        } catch (const std::exception& e) {
            res.status = 400;
            res.set_content(errorResponse("INVALID_REQUEST", e.what(),
                correlation_id).dump(), "application/json");
        }
    });

    // Sign endpoint
    server->Post("/v1/sign", [this](const httplib::Request& req,
                                     httplib::Response& res) {
        auto correlation_id = getCorrelationId(req);
        
        try {
            auto body = json::parse(req.body);
            
            auto data = base64Decode(body["data"].get<std::string>());
            auto key_id_result = KeyId::parse(body["key_id"].get<std::string>());
            
            if (!key_id_result) {
                res.status = 400;
                res.set_content(errorResponse("INVALID_KEY_ID",
                    "Invalid key ID format", correlation_id).dump(),
                    "application/json");
                return;
            }
            
            SignatureContext ctx{
                .correlation_id = correlation_id,
                .caller_identity = "rest-client",
                .caller_service = "rest-api",
                .source_ip = req.remote_addr
            };
            
            auto result = signature_service_->sign(data, *key_id_result, ctx);
            
            if (!result) {
                res.status = 400;
                res.set_content(errorResponse("SIGN_FAILED",
                    result.error().message, correlation_id).dump(),
                    "application/json");
                return;
            }
            
            json response = {
                {"signature", base64Encode(result->signature)},
                {"key_id", result->key_id.toString()},
                {"algorithm", result->algorithm}
            };
            
            res.set_content(response.dump(), "application/json");
            
        } catch (const std::exception& e) {
            res.status = 400;
            res.set_content(errorResponse("INVALID_REQUEST", e.what(),
                correlation_id).dump(), "application/json");
        }
    });
    
    // Verify endpoint
    server->Post("/v1/verify", [this](const httplib::Request& req,
                                       httplib::Response& res) {
        auto correlation_id = getCorrelationId(req);
        
        try {
            auto body = json::parse(req.body);
            
            auto data = base64Decode(body["data"].get<std::string>());
            auto signature = base64Decode(body["signature"].get<std::string>());
            auto key_id_result = KeyId::parse(body["key_id"].get<std::string>());
            
            if (!key_id_result) {
                res.status = 400;
                res.set_content(errorResponse("INVALID_KEY_ID",
                    "Invalid key ID format", correlation_id).dump(),
                    "application/json");
                return;
            }
            
            SignatureContext ctx{
                .correlation_id = correlation_id,
                .caller_identity = "rest-client",
                .caller_service = "rest-api",
                .source_ip = req.remote_addr
            };
            
            auto result = signature_service_->verify(data, signature, 
                                                      *key_id_result, ctx);
            
            if (!result) {
                res.status = 400;
                res.set_content(errorResponse("VERIFY_FAILED",
                    result.error().message, correlation_id).dump(),
                    "application/json");
                return;
            }
            
            json response = {
                {"valid", result->valid},
                {"key_id", result->key_id.toString()}
            };
            
            res.set_content(response.dump(), "application/json");
            
        } catch (const std::exception& e) {
            res.status = 400;
            res.set_content(errorResponse("INVALID_REQUEST", e.what(),
                correlation_id).dump(), "application/json");
        }
    });

    // Generate key endpoint
    server->Post("/v1/keys", [this](const httplib::Request& req,
                                     httplib::Response& res) {
        auto correlation_id = getCorrelationId(req);
        
        try {
            auto body = json::parse(req.body);
            
            auto algorithm = body["algorithm"].get<std::string>();
            auto ns = body.value("namespace", "default");
            
            KeyType key_type;
            int key_size = 256;
            
            if (algorithm == "AES-128") {
                key_type = KeyType::AES;
                key_size = 128;
            } else if (algorithm == "AES-256") {
                key_type = KeyType::AES;
                key_size = 256;
            } else if (algorithm.find("RSA") == 0) {
                key_type = KeyType::RSA;
                key_size = std::stoi(algorithm.substr(4));
            } else if (algorithm.find("ECDSA") == 0) {
                key_type = KeyType::ECDSA;
            } else {
                res.status = 400;
                res.set_content(errorResponse("INVALID_ALGORITHM",
                    "Unsupported algorithm", correlation_id).dump(),
                    "application/json");
                return;
            }
            
            auto result = key_service_->generateKey(key_type, key_size, ns);
            
            if (!result) {
                res.status = 400;
                res.set_content(errorResponse("KEY_GENERATION_FAILED",
                    result.error().message, correlation_id).dump(),
                    "application/json");
                return;
            }
            
            json response = {
                {"key_id", result->toString()}
            };
            
            res.set_content(response.dump(), "application/json");
            
        } catch (const std::exception& e) {
            res.status = 400;
            res.set_content(errorResponse("INVALID_REQUEST", e.what(),
                correlation_id).dump(), "application/json");
        }
    });
    
    // Get key metadata endpoint
    server->Get(R"(/v1/keys/(.+))", [this](const httplib::Request& req,
                                            httplib::Response& res) {
        auto correlation_id = getCorrelationId(req);
        auto key_id_str = req.matches[1].str();
        
        auto key_id_result = KeyId::parse(key_id_str);
        if (!key_id_result) {
            res.status = 400;
            res.set_content(errorResponse("INVALID_KEY_ID",
                "Invalid key ID format", correlation_id).dump(),
                "application/json");
            return;
        }
        
        auto result = key_service_->getKeyMetadata(*key_id_result);
        
        if (!result) {
            res.status = 404;
            res.set_content(errorResponse("KEY_NOT_FOUND",
                "Key not found", correlation_id).dump(),
                "application/json");
            return;
        }
        
        json response = {
            {"key_id", result->id.toString()},
            {"algorithm", static_cast<int>(result->algorithm)},
            {"state", static_cast<int>(result->state)},
            {"created_at", std::chrono::system_clock::to_time_t(result->created_at)},
            {"owner_service", result->owner_service}
        };
        
        res.set_content(response.dump(), "application/json");
    });
    
    // Rotate key endpoint
    server->Post(R"(/v1/keys/(.+)/rotate)", [this](const httplib::Request& req,
                                                    httplib::Response& res) {
        auto correlation_id = getCorrelationId(req);
        auto key_id_str = req.matches[1].str();
        
        auto key_id_result = KeyId::parse(key_id_str);
        if (!key_id_result) {
            res.status = 400;
            res.set_content(errorResponse("INVALID_KEY_ID",
                "Invalid key ID format", correlation_id).dump(),
                "application/json");
            return;
        }
        
        auto result = key_service_->rotateKey(*key_id_result);
        
        if (!result) {
            res.status = 400;
            res.set_content(errorResponse("ROTATION_FAILED",
                result.error().message, correlation_id).dump(),
                "application/json");
            return;
        }
        
        json response = {
            {"new_key_id", result->toString()},
            {"old_key_id", key_id_str}
        };
        
        res.set_content(response.dump(), "application/json");
    });
    
    // Start server
    server->listen("0.0.0.0", config_.port);
}

} // namespace crypto
