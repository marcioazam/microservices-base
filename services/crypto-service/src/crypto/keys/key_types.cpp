#include "crypto/keys/key_types.h"
#include <sstream>
#include <algorithm>

namespace crypto {

const char* keyAlgorithmToString(KeyAlgorithm algo) {
    switch (algo) {
        case KeyAlgorithm::AES_128_GCM: return "AES_128_GCM";
        case KeyAlgorithm::AES_256_GCM: return "AES_256_GCM";
        case KeyAlgorithm::AES_128_CBC: return "AES_128_CBC";
        case KeyAlgorithm::AES_256_CBC: return "AES_256_CBC";
        case KeyAlgorithm::RSA_2048: return "RSA_2048";
        case KeyAlgorithm::RSA_3072: return "RSA_3072";
        case KeyAlgorithm::RSA_4096: return "RSA_4096";
        case KeyAlgorithm::ECDSA_P256: return "ECDSA_P256";
        case KeyAlgorithm::ECDSA_P384: return "ECDSA_P384";
        case KeyAlgorithm::ECDSA_P521: return "ECDSA_P521";
        default: return "UNKNOWN";
    }
}

const char* keyTypeToString(KeyType type) {
    switch (type) {
        case KeyType::SYMMETRIC: return "SYMMETRIC";
        case KeyType::ASYMMETRIC_PUBLIC: return "ASYMMETRIC_PUBLIC";
        case KeyType::ASYMMETRIC_PRIVATE: return "ASYMMETRIC_PRIVATE";
        default: return "UNKNOWN";
    }
}

const char* keyStateToString(KeyState state) {
    switch (state) {
        case KeyState::PENDING_ACTIVATION: return "PENDING_ACTIVATION";
        case KeyState::ACTIVE: return "ACTIVE";
        case KeyState::DEPRECATED: return "DEPRECATED";
        case KeyState::PENDING_DESTRUCTION: return "PENDING_DESTRUCTION";
        case KeyState::DESTROYED: return "DESTROYED";
        default: return "UNKNOWN";
    }
}

Result<KeyAlgorithm> parseKeyAlgorithm(std::string_view str) {
    if (str == "AES_128_GCM") return Ok(KeyAlgorithm::AES_128_GCM);
    if (str == "AES_256_GCM") return Ok(KeyAlgorithm::AES_256_GCM);
    if (str == "AES_128_CBC") return Ok(KeyAlgorithm::AES_128_CBC);
    if (str == "AES_256_CBC") return Ok(KeyAlgorithm::AES_256_CBC);
    if (str == "RSA_2048") return Ok(KeyAlgorithm::RSA_2048);
    if (str == "RSA_3072") return Ok(KeyAlgorithm::RSA_3072);
    if (str == "RSA_4096") return Ok(KeyAlgorithm::RSA_4096);
    if (str == "ECDSA_P256") return Ok(KeyAlgorithm::ECDSA_P256);
    if (str == "ECDSA_P384") return Ok(KeyAlgorithm::ECDSA_P384);
    if (str == "ECDSA_P521") return Ok(KeyAlgorithm::ECDSA_P521);
    return Err<KeyAlgorithm>(ErrorCode::INVALID_INPUT, "Unknown key algorithm");
}

Result<KeyType> parseKeyType(std::string_view str) {
    if (str == "SYMMETRIC") return Ok(KeyType::SYMMETRIC);
    if (str == "ASYMMETRIC_PUBLIC") return Ok(KeyType::ASYMMETRIC_PUBLIC);
    if (str == "ASYMMETRIC_PRIVATE") return Ok(KeyType::ASYMMETRIC_PRIVATE);
    return Err<KeyType>(ErrorCode::INVALID_INPUT, "Unknown key type");
}

Result<KeyState> parseKeyState(std::string_view str) {
    if (str == "PENDING_ACTIVATION") return Ok(KeyState::PENDING_ACTIVATION);
    if (str == "ACTIVE") return Ok(KeyState::ACTIVE);
    if (str == "DEPRECATED") return Ok(KeyState::DEPRECATED);
    if (str == "PENDING_DESTRUCTION") return Ok(KeyState::PENDING_DESTRUCTION);
    if (str == "DESTROYED") return Ok(KeyState::DESTROYED);
    return Err<KeyState>(ErrorCode::INVALID_INPUT, "Unknown key state");
}

size_t getKeySize(KeyAlgorithm algo) {
    switch (algo) {
        case KeyAlgorithm::AES_128_GCM:
        case KeyAlgorithm::AES_128_CBC:
            return 16;
        case KeyAlgorithm::AES_256_GCM:
        case KeyAlgorithm::AES_256_CBC:
            return 32;
        case KeyAlgorithm::RSA_2048:
            return 256;  // 2048 bits
        case KeyAlgorithm::RSA_3072:
            return 384;  // 3072 bits
        case KeyAlgorithm::RSA_4096:
            return 512;  // 4096 bits
        case KeyAlgorithm::ECDSA_P256:
            return 32;
        case KeyAlgorithm::ECDSA_P384:
            return 48;
        case KeyAlgorithm::ECDSA_P521:
            return 66;
        default:
            return 0;
    }
}

bool isSymmetricAlgorithm(KeyAlgorithm algo) {
    switch (algo) {
        case KeyAlgorithm::AES_128_GCM:
        case KeyAlgorithm::AES_256_GCM:
        case KeyAlgorithm::AES_128_CBC:
        case KeyAlgorithm::AES_256_CBC:
            return true;
        default:
            return false;
    }
}

bool isAsymmetricAlgorithm(KeyAlgorithm algo) {
    return !isSymmetricAlgorithm(algo);
}

// KeyId implementation
KeyId KeyId::generate(const std::string& namespace_prefix) {
    return KeyId(namespace_prefix, UUID::generate().to_string(), 1);
}

Result<KeyId> KeyId::parse(std::string_view str) {
    // Format: "namespace:uuid:version"
    std::string s(str);
    std::vector<std::string> parts;
    std::istringstream iss(s);
    std::string part;
    
    while (std::getline(iss, part, ':')) {
        parts.push_back(part);
    }

    if (parts.size() != 3) {
        return Err<KeyId>(ErrorCode::INVALID_INPUT, "Invalid KeyId format");
    }

    try {
        uint32_t version = std::stoul(parts[2]);
        return Ok(KeyId(parts[0], parts[1], version));
    } catch (...) {
        return Err<KeyId>(ErrorCode::INVALID_INPUT, "Invalid version number");
    }
}

std::string KeyId::toString() const {
    return namespace_prefix + ":" + id + ":" + std::to_string(version);
}

bool KeyId::operator==(const KeyId& other) const {
    return namespace_prefix == other.namespace_prefix &&
           id == other.id &&
           version == other.version;
}

bool KeyId::operator<(const KeyId& other) const {
    if (namespace_prefix != other.namespace_prefix)
        return namespace_prefix < other.namespace_prefix;
    if (id != other.id)
        return id < other.id;
    return version < other.version;
}

bool KeyId::isValid() const {
    return !namespace_prefix.empty() && !id.empty() && version > 0;
}

// KeyMetadata implementation
KeyMetadata::KeyMetadata()
    : algorithm(KeyAlgorithm::AES_256_GCM)
    , type(KeyType::SYMMETRIC)
    , state(KeyState::ACTIVE)
    , created_at(std::chrono::system_clock::now())
    , expires_at(std::chrono::system_clock::now() + std::chrono::hours(24 * 365))
    , usage_count(0) {}

bool KeyMetadata::canEncrypt() const {
    if (state != KeyState::ACTIVE) return false;
    if (isExpired()) return false;
    
    auto it = std::find(allowed_operations.begin(), allowed_operations.end(), "encrypt");
    return it != allowed_operations.end();
}

bool KeyMetadata::canDecrypt() const {
    // Deprecated keys can still decrypt (for grace period)
    if (state != KeyState::ACTIVE && state != KeyState::DEPRECATED) return false;
    if (isExpired()) return false;
    
    auto it = std::find(allowed_operations.begin(), allowed_operations.end(), "decrypt");
    return it != allowed_operations.end();
}

bool KeyMetadata::isExpired() const {
    return std::chrono::system_clock::now() > expires_at;
}

bool KeyMetadata::isActive() const {
    return state == KeyState::ACTIVE && !isExpired();
}

} // namespace crypto
