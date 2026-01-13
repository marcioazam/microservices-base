#include "crypto/auth/jwt_validator.h"
#include <openssl/evp.h>
#include <openssl/pem.h>
#include <openssl/rsa.h>
#include <openssl/err.h>
#include <fstream>
#include <sstream>
#include <algorithm>
#include <cstring>

namespace crypto {

namespace {

// Base64URL decode
std::vector<uint8_t> base64UrlDecode(const std::string& input) {
    std::string base64 = input;
    
    // Convert base64url to base64
    std::replace(base64.begin(), base64.end(), '-', '+');
    std::replace(base64.begin(), base64.end(), '_', '/');
    
    // Add padding
    while (base64.size() % 4 != 0) {
        base64 += '=';
    }
    
    // Decode
    std::vector<uint8_t> output(base64.size());
    int len = EVP_DecodeBlock(output.data(), 
                              reinterpret_cast<const uint8_t*>(base64.data()),
                              static_cast<int>(base64.size()));
    if (len < 0) {
        return {};
    }
    
    // Remove padding bytes
    while (!output.empty() && output.back() == 0) {
        output.pop_back();
    }
    
    return output;
}

std::vector<std::string> split(const std::string& str, char delimiter) {
    std::vector<std::string> parts;
    std::stringstream ss(str);
    std::string part;
    while (std::getline(ss, part, delimiter)) {
        parts.push_back(part);
    }
    return parts;
}

} // anonymous namespace

JWTValidator::JWTValidator(const JWTValidatorConfig& config)
    : config_(config) {
    if (!config_.public_key_path.empty()) {
        auto result = loadPublicKey();
        if (!result) {
            throw std::runtime_error("Failed to load public key: " + result.error().message);
        }
    }
}

Result<void> JWTValidator::loadPublicKey() {
    std::ifstream file(config_.public_key_path);
    if (!file) {
        return Err<void>(ErrorCode::FILE_NOT_FOUND, 
                         "Cannot open public key file");
    }
    
    std::stringstream buffer;
    buffer << file.rdbuf();
    public_key_ = buffer.str();
    
    return Ok();
}

Result<void> JWTValidator::refreshKeys() {
    if (!config_.jwks_url.empty()) {
        // TODO: Implement JWKS fetching
        return Err<void>(ErrorCode::NOT_IMPLEMENTED, "JWKS not implemented");
    }
    return loadPublicKey();
}

JWTValidationResult JWTValidator::validate(const std::string& token) {
    JWTValidationResult result;
    result.valid = false;
    
    // Parse token
    auto claims_result = parseToken(token);
    if (!claims_result) {
        result.error = claims_result.error().message;
        return result;
    }
    
    // Verify signature
    if (!verifySignature(token)) {
        result.error = "Invalid signature";
        return result;
    }
    
    // Validate claims
    if (!validateClaims(*claims_result)) {
        result.error = "Invalid claims";
        return result;
    }
    
    result.valid = true;
    result.claims = std::move(*claims_result);
    return result;
}

Result<JWTClaims> JWTValidator::parseToken(const std::string& token) {
    auto parts = split(token, '.');
    if (parts.size() != 3) {
        return Err<JWTClaims>(ErrorCode::INVALID_INPUT, "Invalid JWT format");
    }
    
    // Decode payload (second part)
    auto payload_bytes = base64UrlDecode(parts[1]);
    if (payload_bytes.empty()) {
        return Err<JWTClaims>(ErrorCode::INVALID_INPUT, "Invalid payload encoding");
    }
    
    std::string payload(payload_bytes.begin(), payload_bytes.end());
    
    // Simple JSON parsing (in production, use a proper JSON library)
    JWTClaims claims;
    
    // Extract subject
    auto sub_pos = payload.find("\"sub\"");
    if (sub_pos != std::string::npos) {
        auto start = payload.find('\"', sub_pos + 5) + 1;
        auto end = payload.find('\"', start);
        claims.subject = payload.substr(start, end - start);
    }
    
    // Extract issuer
    auto iss_pos = payload.find("\"iss\"");
    if (iss_pos != std::string::npos) {
        auto start = payload.find('\"', iss_pos + 5) + 1;
        auto end = payload.find('\"', start);
        claims.issuer = payload.substr(start, end - start);
    }
    
    // Extract expiration
    auto exp_pos = payload.find("\"exp\"");
    if (exp_pos != std::string::npos) {
        auto start = payload.find(':', exp_pos) + 1;
        auto end = payload.find_first_of(",}", start);
        int64_t exp = std::stoll(payload.substr(start, end - start));
        claims.expires_at = std::chrono::system_clock::from_time_t(exp);
    }
    
    // Extract issued at
    auto iat_pos = payload.find("\"iat\"");
    if (iat_pos != std::string::npos) {
        auto start = payload.find(':', iat_pos) + 1;
        auto end = payload.find_first_of(",}", start);
        int64_t iat = std::stoll(payload.substr(start, end - start));
        claims.issued_at = std::chrono::system_clock::from_time_t(iat);
    }
    
    // Extract service name (custom claim)
    auto svc_pos = payload.find("\"service\"");
    if (svc_pos != std::string::npos) {
        auto start = payload.find('\"', svc_pos + 9) + 1;
        auto end = payload.find('\"', start);
        claims.service_name = payload.substr(start, end - start);
    }
    
    return Ok(std::move(claims));
}

bool JWTValidator::verifySignature(const std::string& token) {
    if (public_key_.empty()) {
        return false;
    }
    
    auto parts = split(token, '.');
    if (parts.size() != 3) {
        return false;
    }
    
    std::string signed_data = parts[0] + "." + parts[1];
    auto signature = base64UrlDecode(parts[2]);
    
    // Load public key
    BIO* bio = BIO_new_mem_buf(public_key_.data(), 
                               static_cast<int>(public_key_.size()));
    if (!bio) {
        return false;
    }
    
    EVP_PKEY* pkey = PEM_read_bio_PUBKEY(bio, nullptr, nullptr, nullptr);
    BIO_free(bio);
    
    if (!pkey) {
        return false;
    }
    
    // Verify signature using RS256
    EVP_MD_CTX* ctx = EVP_MD_CTX_new();
    bool valid = false;
    
    if (ctx) {
        if (EVP_DigestVerifyInit(ctx, nullptr, EVP_sha256(), nullptr, pkey) == 1) {
            if (EVP_DigestVerifyUpdate(ctx, signed_data.data(), signed_data.size()) == 1) {
                valid = EVP_DigestVerifyFinal(ctx, signature.data(), 
                                              signature.size()) == 1;
            }
        }
        EVP_MD_CTX_free(ctx);
    }
    
    EVP_PKEY_free(pkey);
    return valid;
}

bool JWTValidator::validateClaims(const JWTClaims& claims) {
    auto now = std::chrono::system_clock::now();
    
    // Check expiration
    if (config_.require_exp) {
        if (claims.expires_at < now - config_.clock_skew) {
            return false;
        }
    }
    
    // Check issued at
    if (config_.require_iat) {
        if (claims.issued_at > now + config_.clock_skew) {
            return false;
        }
    }
    
    // Check issuer
    if (!config_.expected_issuer.empty()) {
        if (claims.issuer != config_.expected_issuer) {
            return false;
        }
    }
    
    return true;
}

std::optional<std::string> extractBearerToken(const std::string& auth_header) {
    const std::string prefix = "Bearer ";
    if (auth_header.size() <= prefix.size()) {
        return std::nullopt;
    }
    
    if (auth_header.substr(0, prefix.size()) != prefix) {
        return std::nullopt;
    }
    
    return auth_header.substr(prefix.size());
}

} // namespace crypto
