#pragma once

#include "crypto/common/result.h"
#include "crypto/common/uuid.h"
#include <string>
#include <vector>
#include <chrono>
#include <optional>
#include <cstdint>

namespace crypto {

// Key algorithms
enum class KeyAlgorithm {
    AES_128_GCM,
    AES_256_GCM,
    AES_128_CBC,
    AES_256_CBC,
    RSA_2048,
    RSA_3072,
    RSA_4096,
    ECDSA_P256,
    ECDSA_P384,
    ECDSA_P521
};

// Key types
enum class KeyType {
    SYMMETRIC,
    ASYMMETRIC_PUBLIC,
    ASYMMETRIC_PRIVATE
};

// Key states
enum class KeyState {
    PENDING_ACTIVATION,
    ACTIVE,
    DEPRECATED,
    PENDING_DESTRUCTION,
    DESTROYED
};

// Convert enums to strings
const char* keyAlgorithmToString(KeyAlgorithm algo);
const char* keyTypeToString(KeyType type);
const char* keyStateToString(KeyState state);

// Parse strings to enums
Result<KeyAlgorithm> parseKeyAlgorithm(std::string_view str);
Result<KeyType> parseKeyType(std::string_view str);
Result<KeyState> parseKeyState(std::string_view str);

// Key identifier with namespace support
struct KeyId {
    std::string namespace_prefix;  // Service namespace (e.g., "auth", "payment")
    std::string id;                // UUID v4
    uint32_t version;              // Key version for rotation

    KeyId() : version(1) {}
    KeyId(std::string ns, std::string uuid, uint32_t ver = 1)
        : namespace_prefix(std::move(ns)), id(std::move(uuid)), version(ver) {}

    // Generate new KeyId
    static KeyId generate(const std::string& namespace_prefix = "default");

    // Parse from string format: "namespace:uuid:version"
    static Result<KeyId> parse(std::string_view str);

    // Convert to string format
    std::string toString() const;

    // Comparison operators
    bool operator==(const KeyId& other) const;
    bool operator!=(const KeyId& other) const { return !(*this == other); }
    bool operator<(const KeyId& other) const;

    // Check if valid
    bool isValid() const;
};

// Key metadata stored alongside encrypted key
struct KeyMetadata {
    KeyId id;
    KeyAlgorithm algorithm;
    KeyType type;
    KeyState state;
    std::chrono::system_clock::time_point created_at;
    std::chrono::system_clock::time_point expires_at;
    std::optional<std::chrono::system_clock::time_point> rotated_at;
    std::optional<KeyId> previous_version;
    std::string owner_service;
    std::vector<std::string> allowed_operations;
    uint64_t usage_count;

    KeyMetadata();

    // Check if key is usable for encryption
    bool canEncrypt() const;
    
    // Check if key is usable for decryption
    bool canDecrypt() const;
    
    // Check if key is expired
    bool isExpired() const;
    
    // Check if key is active
    bool isActive() const;
};

// Encrypted key storage format
struct EncryptedKey {
    std::vector<uint8_t> encrypted_material;
    std::vector<uint8_t> iv;
    std::vector<uint8_t> tag;
    KeyId kek_id;  // Key Encryption Key used
    KeyMetadata metadata;
};

// Key generation parameters
struct KeyGenerationParams {
    std::string namespace_prefix;
    KeyAlgorithm algorithm;
    std::string owner_service;
    std::chrono::seconds validity_period;
    std::vector<std::string> allowed_operations;

    KeyGenerationParams()
        : namespace_prefix("default")
        , algorithm(KeyAlgorithm::AES_256_GCM)
        , validity_period(std::chrono::hours(24 * 365))  // 1 year default
        , allowed_operations({"encrypt", "decrypt"}) {}
};

// Get key size in bytes for algorithm
size_t getKeySize(KeyAlgorithm algo);

// Check if algorithm is symmetric
bool isSymmetricAlgorithm(KeyAlgorithm algo);

// Check if algorithm is asymmetric
bool isAsymmetricAlgorithm(KeyAlgorithm algo);

} // namespace crypto
