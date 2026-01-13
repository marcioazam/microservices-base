#pragma once

#include "crypto/common/result.h"
#include "crypto/auth/jwt_validator.h"
#include "crypto/keys/key_types.h"
#include <string>
#include <vector>
#include <unordered_map>
#include <unordered_set>

namespace crypto {

// Operations that can be authorized
enum class Operation {
    ENCRYPT,
    DECRYPT,
    SIGN,
    VERIFY,
    KEY_GENERATE,
    KEY_ROTATE,
    KEY_DELETE,
    KEY_READ,
    FILE_ENCRYPT,
    FILE_DECRYPT
};

// Role definition
struct Role {
    std::string name;
    std::unordered_set<Operation> allowed_operations;
    std::vector<std::string> allowed_namespaces;  // Empty = all namespaces
    bool is_admin = false;
};

// Authorization request
struct AuthorizationRequest {
    std::string subject;           // User/service ID
    std::vector<std::string> roles;
    Operation operation;
    std::optional<KeyId> key_id;   // For key-specific operations
    std::string target_namespace;  // For namespace-scoped operations
};

// Authorization result
struct AuthorizationResult {
    bool authorized;
    std::optional<std::string> reason;
};

// RBAC Engine configuration
struct RBACConfig {
    std::unordered_map<std::string, Role> roles;
    bool default_deny = true;
    bool enable_namespace_isolation = true;
};

// RBAC Engine
class RBACEngine {
public:
    explicit RBACEngine(const RBACConfig& config);
    ~RBACEngine() = default;
    
    // Check authorization
    AuthorizationResult authorize(const AuthorizationRequest& request);
    
    // Check if subject can access key
    AuthorizationResult canAccessKey(const JWTClaims& claims, 
                                     const KeyId& key_id,
                                     Operation operation);
    
    // Add/update role
    void addRole(const Role& role);
    
    // Remove role
    void removeRole(const std::string& role_name);
    
    // Get role
    std::optional<Role> getRole(const std::string& role_name) const;

private:
    RBACConfig config_;
    
    bool hasOperation(const Role& role, Operation op) const;
    bool hasNamespaceAccess(const Role& role, const std::string& ns) const;
};

// Default roles
namespace DefaultRoles {
    Role admin();
    Role keyManager();
    Role encryptor();
    Role signer();
    Role reader();
}

// Convert operation to string
std::string operationToString(Operation op);

} // namespace crypto
