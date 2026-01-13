#pragma once

#include "crypto/common/result.h"
#include <string>
#include <vector>
#include <chrono>
#include <optional>
#include <memory>

namespace crypto {

// JWT claims extracted from token
struct JWTClaims {
    std::string subject;           // sub - user/service ID
    std::string issuer;            // iss - token issuer
    std::string audience;          // aud - intended audience
    std::vector<std::string> roles;
    std::string service_name;      // Custom claim for service identity
    std::string namespace_prefix;  // Custom claim for key namespace access
    std::chrono::system_clock::time_point issued_at;
    std::chrono::system_clock::time_point expires_at;
    std::optional<std::string> jti;  // JWT ID for tracking
};

// JWT validation configuration
struct JWTValidatorConfig {
    std::string public_key_path;   // RSA/ECDSA public key for verification
    std::string jwks_url;          // JWKS endpoint for key rotation
    std::string expected_issuer;
    std::string expected_audience;
    std::chrono::seconds clock_skew{60};  // Allowed clock skew
    bool require_exp = true;
    bool require_iat = true;
};

// JWT validation result
struct JWTValidationResult {
    bool valid;
    std::optional<JWTClaims> claims;
    std::optional<std::string> error;
};

// JWT Validator interface
class IJWTValidator {
public:
    virtual ~IJWTValidator() = default;
    virtual JWTValidationResult validate(const std::string& token) = 0;
    virtual Result<void> refreshKeys() = 0;
};

// JWT Validator implementation
class JWTValidator : public IJWTValidator {
public:
    explicit JWTValidator(const JWTValidatorConfig& config);
    ~JWTValidator() override = default;
    
    JWTValidationResult validate(const std::string& token) override;
    Result<void> refreshKeys() override;

private:
    JWTValidatorConfig config_;
    std::string public_key_;
    
    Result<void> loadPublicKey();
    Result<JWTClaims> parseToken(const std::string& token);
    bool verifySignature(const std::string& token);
    bool validateClaims(const JWTClaims& claims);
};

// Extract JWT from Authorization header
std::optional<std::string> extractBearerToken(const std::string& auth_header);

} // namespace crypto
