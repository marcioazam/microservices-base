#include "crypto/auth/rbac_engine.h"

namespace crypto {

RBACEngine::RBACEngine(const RBACConfig& config)
    : config_(config) {}

AuthorizationResult RBACEngine::authorize(const AuthorizationRequest& request) {
    AuthorizationResult result;
    result.authorized = false;
    
    // Check each role the subject has
    for (const auto& role_name : request.roles) {
        auto it = config_.roles.find(role_name);
        if (it == config_.roles.end()) {
            continue;
        }
        
        const auto& role = it->second;
        
        // Admin bypasses all checks
        if (role.is_admin) {
            result.authorized = true;
            return result;
        }
        
        // Check operation permission
        if (!hasOperation(role, request.operation)) {
            continue;
        }
        
        // Check namespace access
        if (config_.enable_namespace_isolation && !request.target_namespace.empty()) {
            if (!hasNamespaceAccess(role, request.target_namespace)) {
                continue;
            }
        }
        
        // All checks passed
        result.authorized = true;
        return result;
    }
    
    result.reason = "No role grants permission for " + 
                    operationToString(request.operation);
    return result;
}

AuthorizationResult RBACEngine::canAccessKey(
    const JWTClaims& claims,
    const KeyId& key_id,
    Operation operation) {
    
    AuthorizationRequest request;
    request.subject = claims.subject;
    request.roles = claims.roles;
    request.operation = operation;
    request.key_id = key_id;
    request.target_namespace = key_id.namespace_prefix;
    
    return authorize(request);
}

void RBACEngine::addRole(const Role& role) {
    config_.roles[role.name] = role;
}

void RBACEngine::removeRole(const std::string& role_name) {
    config_.roles.erase(role_name);
}

std::optional<Role> RBACEngine::getRole(const std::string& role_name) const {
    auto it = config_.roles.find(role_name);
    if (it != config_.roles.end()) {
        return it->second;
    }
    return std::nullopt;
}

bool RBACEngine::hasOperation(const Role& role, Operation op) const {
    return role.allowed_operations.count(op) > 0;
}

bool RBACEngine::hasNamespaceAccess(const Role& role, const std::string& ns) const {
    if (role.allowed_namespaces.empty()) {
        return true;  // Empty = all namespaces
    }
    
    for (const auto& allowed : role.allowed_namespaces) {
        if (allowed == ns || allowed == "*") {
            return true;
        }
        // Support prefix matching (e.g., "auth:*" matches "auth:users")
        if (allowed.back() == '*') {
            std::string prefix = allowed.substr(0, allowed.size() - 1);
            if (ns.substr(0, prefix.size()) == prefix) {
                return true;
            }
        }
    }
    
    return false;
}

namespace DefaultRoles {

Role admin() {
    Role role;
    role.name = "admin";
    role.is_admin = true;
    role.allowed_operations = {
        Operation::ENCRYPT, Operation::DECRYPT,
        Operation::SIGN, Operation::VERIFY,
        Operation::KEY_GENERATE, Operation::KEY_ROTATE,
        Operation::KEY_DELETE, Operation::KEY_READ,
        Operation::FILE_ENCRYPT, Operation::FILE_DECRYPT
    };
    return role;
}

Role keyManager() {
    Role role;
    role.name = "key-manager";
    role.allowed_operations = {
        Operation::KEY_GENERATE, Operation::KEY_ROTATE,
        Operation::KEY_DELETE, Operation::KEY_READ
    };
    return role;
}

Role encryptor() {
    Role role;
    role.name = "encryptor";
    role.allowed_operations = {
        Operation::ENCRYPT, Operation::DECRYPT,
        Operation::KEY_READ,
        Operation::FILE_ENCRYPT, Operation::FILE_DECRYPT
    };
    return role;
}

Role signer() {
    Role role;
    role.name = "signer";
    role.allowed_operations = {
        Operation::SIGN, Operation::VERIFY,
        Operation::KEY_READ
    };
    return role;
}

Role reader() {
    Role role;
    role.name = "reader";
    role.allowed_operations = {
        Operation::KEY_READ, Operation::VERIFY
    };
    return role;
}

} // namespace DefaultRoles

std::string operationToString(Operation op) {
    switch (op) {
        case Operation::ENCRYPT: return "ENCRYPT";
        case Operation::DECRYPT: return "DECRYPT";
        case Operation::SIGN: return "SIGN";
        case Operation::VERIFY: return "VERIFY";
        case Operation::KEY_GENERATE: return "KEY_GENERATE";
        case Operation::KEY_ROTATE: return "KEY_ROTATE";
        case Operation::KEY_DELETE: return "KEY_DELETE";
        case Operation::KEY_READ: return "KEY_READ";
        case Operation::FILE_ENCRYPT: return "FILE_ENCRYPT";
        case Operation::FILE_DECRYPT: return "FILE_DECRYPT";
        default: return "UNKNOWN";
    }
}

} // namespace crypto
