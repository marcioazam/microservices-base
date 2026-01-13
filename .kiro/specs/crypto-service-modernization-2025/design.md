# Design Document

## Overview

This design document describes the modernization of the `crypto-service` to state-of-the-art December 2025 standards. The modernization eliminates redundant code, integrates with platform services (logging-service, cache-service), removes local resilience implementations in favor of Service Mesh, and upgrades to modern C++23 and OpenSSL 3.3+ standards.

### Key Changes

1. **Remove Local Implementations**: Delete `crypto/logging/json_logger.h/.cpp`, `crypto/resilience/circuit_breaker.h/.cpp`, `crypto/resilience/retry.h/.cpp`, `crypto/keys/key_cache.h/.cpp`
2. **Add Platform Clients**: Integrate gRPC clients for `logging-service` and `cache-service`
3. **Centralize Cross-Cutting Concerns**: Single location for RAII wrappers, error codes, hash algorithms
4. **Upgrade Language/Library**: C++23 with `std::expected`, OpenSSL 3.3+ with modern APIs
5. **Reorganize Tests**: Separate `tests/unit/`, `tests/property/`, `tests/integration/`

## Architecture

### Current Architecture (Before)

```
┌─────────────────────────────────────────────────────────────┐
│                    crypto-service                            │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐  │
│  │ JsonLogger  │  │KeyCache     │  │ CircuitBreaker      │  │
│  │ (LOCAL)     │  │(LOCAL)      │  │ (LOCAL)             │  │
│  └─────────────┘  └─────────────┘  └─────────────────────┘  │
│                                                              │
│  ┌─────────────────────────────────────────────────────────┐│
│  │              Crypto Engines (AES, RSA, ECDSA)           ││
│  └─────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────┘
```

### Target Architecture (After)

```
┌─────────────────────────────────────────────────────────────┐
│                    crypto-service                            │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐  ┌─────────────────┐                   │
│  │ LoggingClient   │  │ CacheClient     │                   │
│  │ (gRPC to        │  │ (gRPC to        │                   │
│  │  logging-svc)   │  │  cache-svc)     │                   │
│  └────────┬────────┘  └────────┬────────┘                   │
│           │                    │                             │
│  ┌────────▼────────────────────▼────────────────────────┐   │
│  │              Core Services Layer                      │   │
│  │  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐  │   │
│  │  │ Encryption   │ │  Signature   │ │  Key         │  │   │
│  │  │ Service      │ │  Service     │ │  Service     │  │   │
│  │  └──────────────┘ └──────────────┘ └──────────────┘  │   │
│  └──────────────────────────────────────────────────────┘   │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐   │
│  │              Crypto Engines (Centralized)             │   │
│  │  ┌────────┐ ┌────────┐ ┌────────┐ ┌────────────────┐ │   │
│  │  │  AES   │ │  RSA   │ │ ECDSA  │ │ Hybrid         │ │   │
│  │  └────────┘ └────────┘ └────────┘ └────────────────┘ │   │
│  └──────────────────────────────────────────────────────┘   │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐   │
│  │              Common Layer (Centralized)               │   │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ │   │
│  │  │ Result   │ │ OpenSSL  │ │ Config   │ │ Metrics  │ │   │
│  │  │ <T,E>    │ │ RAII     │ │ Loader   │ │ Exporter │ │   │
│  │  └──────────┘ └──────────┘ └──────────┘ └──────────┘ │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
           │                              │
           ▼                              ▼
┌──────────────────────┐    ┌──────────────────────┐
│   logging-service    │    │    cache-service     │
│   (gRPC :5001)       │    │    (gRPC :50051)     │
└──────────────────────┘    └──────────────────────┘
           │                              │
           └──────────────┬───────────────┘
                          ▼
              ┌──────────────────────┐
              │   Linkerd Proxy      │
              │   (Service Mesh)     │
              │   - Circuit Breaker  │
              │   - Retry            │
              │   - mTLS             │
              └──────────────────────┘
```

## Components and Interfaces

### 1. LoggingClient (New)

gRPC client for centralized logging-service integration.

```cpp
// include/crypto/clients/logging_client.h
namespace crypto {

struct LoggingClientConfig {
    std::string address = "localhost:5001";
    std::string service_id = "crypto-service";
    size_t batch_size = 100;
    std::chrono::milliseconds flush_interval{5000};
    size_t buffer_size = 10000;
};

class LoggingClient {
public:
    explicit LoggingClient(const LoggingClientConfig& config);
    ~LoggingClient();

    // Async logging methods
    void debug(std::string_view message, 
               const std::map<std::string, std::string>& fields = {});
    void info(std::string_view message,
              const std::map<std::string, std::string>& fields = {});
    void warn(std::string_view message,
              const std::map<std::string, std::string>& fields = {});
    void error(std::string_view message,
               const std::map<std::string, std::string>& fields = {});

    // Structured logging with context
    void log(LogLevel level, std::string_view message,
             std::string_view correlation_id,
             const std::map<std::string, std::string>& fields = {});

    // Flush buffered logs
    void flush();

    // Health check
    bool is_connected() const;

private:
    struct Impl;
    std::unique_ptr<Impl> impl_;
};

} // namespace crypto
```

### 2. CacheClient (New)

gRPC client for centralized cache-service integration.

```cpp
// include/crypto/clients/cache_client.h
namespace crypto {

struct CacheClientConfig {
    std::string address = "localhost:50051";
    std::string namespace_prefix = "crypto";
    std::chrono::seconds default_ttl{300};
    std::optional<std::array<uint8_t, 32>> encryption_key;
    bool local_fallback_enabled = true;
    size_t local_cache_size = 1000;
};

class CacheClient {
public:
    explicit CacheClient(const CacheClientConfig& config);
    ~CacheClient();

    // Cache operations
    std::expected<std::vector<uint8_t>, CacheError> get(std::string_view key);
    std::expected<void, CacheError> set(std::string_view key, 
                                         std::span<const uint8_t> value,
                                         std::optional<std::chrono::seconds> ttl = std::nullopt);
    std::expected<void, CacheError> del(std::string_view key);

    // Batch operations
    std::expected<std::map<std::string, std::vector<uint8_t>>, CacheError> 
        batch_get(const std::vector<std::string>& keys);

    // Health check
    bool is_connected() const;

private:
    struct Impl;
    std::unique_ptr<Impl> impl_;
};

} // namespace crypto
```

### 3. Centralized OpenSSL RAII Wrappers (Refactored)

Single header for all OpenSSL resource management.

```cpp
// include/crypto/common/openssl_raii.h
namespace crypto::openssl {

// EVP_CIPHER_CTX wrapper
struct CipherCtxDeleter {
    void operator()(EVP_CIPHER_CTX* ctx) const noexcept {
        if (ctx) EVP_CIPHER_CTX_free(ctx);
    }
};
using CipherCtx = std::unique_ptr<EVP_CIPHER_CTX, CipherCtxDeleter>;

inline CipherCtx make_cipher_ctx() {
    return CipherCtx(EVP_CIPHER_CTX_new());
}

// EVP_PKEY wrapper
struct PKeyDeleter {
    void operator()(EVP_PKEY* key) const noexcept {
        if (key) EVP_PKEY_free(key);
    }
};
using PKey = std::unique_ptr<EVP_PKEY, PKeyDeleter>;

// EVP_PKEY_CTX wrapper
struct PKeyCtxDeleter {
    void operator()(EVP_PKEY_CTX* ctx) const noexcept {
        if (ctx) EVP_PKEY_CTX_free(ctx);
    }
};
using PKeyCtx = std::unique_ptr<EVP_PKEY_CTX, PKeyCtxDeleter>;

// EVP_MD_CTX wrapper
struct MDCtxDeleter {
    void operator()(EVP_MD_CTX* ctx) const noexcept {
        if (ctx) EVP_MD_CTX_free(ctx);
    }
};
using MDCtx = std::unique_ptr<EVP_MD_CTX, MDCtxDeleter>;

inline MDCtx make_md_ctx() {
    return MDCtx(EVP_MD_CTX_new());
}

// BIO wrapper
struct BIODeleter {
    void operator()(BIO* bio) const noexcept {
        if (bio) BIO_free(bio);
    }
};
using BIO_ptr = std::unique_ptr<BIO, BIODeleter>;

// OSSL_PARAM_BLD wrapper (OpenSSL 3.x)
struct ParamBldDeleter {
    void operator()(OSSL_PARAM_BLD* bld) const noexcept {
        if (bld) OSSL_PARAM_BLD_free(bld);
    }
};
using ParamBld = std::unique_ptr<OSSL_PARAM_BLD, ParamBldDeleter>;

// OSSL_PARAM wrapper
struct ParamDeleter {
    void operator()(OSSL_PARAM* params) const noexcept {
        if (params) OSSL_PARAM_free(params);
    }
};
using Params = std::unique_ptr<OSSL_PARAM, ParamDeleter>;

// EVP_MAC wrapper (OpenSSL 3.x)
struct MACDeleter {
    void operator()(EVP_MAC* mac) const noexcept {
        if (mac) EVP_MAC_free(mac);
    }
};
using MAC = std::unique_ptr<EVP_MAC, MACDeleter>;

// EVP_MAC_CTX wrapper
struct MACCtxDeleter {
    void operator()(EVP_MAC_CTX* ctx) const noexcept {
        if (ctx) EVP_MAC_CTX_free(ctx);
    }
};
using MACCtx = std::unique_ptr<EVP_MAC_CTX, MACCtxDeleter>;

} // namespace crypto::openssl
```

### 4. Centralized Hash Algorithm Selection (Refactored)

```cpp
// include/crypto/common/hash_utils.h
namespace crypto {

enum class HashAlgorithm {
    SHA256,
    SHA384,
    SHA512
};

// Centralized hash algorithm utilities
constexpr const EVP_MD* get_evp_md(HashAlgorithm algo) {
    switch (algo) {
        case HashAlgorithm::SHA256: return EVP_sha256();
        case HashAlgorithm::SHA384: return EVP_sha384();
        case HashAlgorithm::SHA512: return EVP_sha512();
    }
    std::unreachable();
}

constexpr size_t get_hash_size(HashAlgorithm algo) {
    switch (algo) {
        case HashAlgorithm::SHA256: return 32;
        case HashAlgorithm::SHA384: return 48;
        case HashAlgorithm::SHA512: return 64;
    }
    std::unreachable();
}

constexpr std::string_view get_hash_name(HashAlgorithm algo) {
    switch (algo) {
        case HashAlgorithm::SHA256: return "SHA256";
        case HashAlgorithm::SHA384: return "SHA384";
        case HashAlgorithm::SHA512: return "SHA512";
    }
    std::unreachable();
}

// Get appropriate hash for EC curve
constexpr HashAlgorithm get_hash_for_curve(ECCurve curve) {
    switch (curve) {
        case ECCurve::P256: return HashAlgorithm::SHA256;
        case ECCurve::P384: return HashAlgorithm::SHA384;
        case ECCurve::P521: return HashAlgorithm::SHA512;
    }
    std::unreachable();
}

} // namespace crypto
```

### 5. Modernized Result Type (C++23)

```cpp
// include/crypto/common/result.h
namespace crypto {

// Error codes - single centralized enumeration
enum class [[nodiscard]] ErrorCode {
    // General errors
    OK = 0,
    UNKNOWN_ERROR,
    INVALID_INPUT,
    
    // Crypto errors
    CRYPTO_ERROR,
    INVALID_KEY_SIZE,
    INVALID_IV_SIZE,
    INVALID_TAG_SIZE,
    INTEGRITY_ERROR,
    PADDING_ERROR,
    KEY_GENERATION_FAILED,
    INVALID_KEY_TYPE,
    SIZE_LIMIT_EXCEEDED,
    
    // Service errors
    SERVICE_UNAVAILABLE,
    TIMEOUT,
    NOT_FOUND,
    PERMISSION_DENIED,
    
    // Cache errors
    CACHE_MISS,
    CACHE_ERROR,
    
    // Logging errors
    LOGGING_ERROR
};

// Error with code and message
struct Error {
    ErrorCode code;
    std::string message;
    
    constexpr bool is_retryable() const noexcept {
        return code == ErrorCode::SERVICE_UNAVAILABLE || 
               code == ErrorCode::TIMEOUT;
    }
};

// Use std::expected for modern error handling
template<typename T>
using Result = std::expected<T, Error>;

// Helper functions
template<typename T>
[[nodiscard]] constexpr Result<T> Ok(T&& value) {
    return Result<T>(std::forward<T>(value));
}

template<typename T>
[[nodiscard]] constexpr Result<T> Err(ErrorCode code, std::string message = "") {
    return std::unexpected(Error{code, std::move(message)});
}

} // namespace crypto
```

## Data Models

### Configuration Model

```cpp
// include/crypto/config/service_config.h
namespace crypto {

struct ServiceConfig {
    // Server configuration
    uint16_t grpc_port = 50051;
    uint16_t rest_port = 8080;
    std::chrono::seconds graceful_shutdown_timeout{30};
    
    // TLS configuration
    std::optional<std::string> tls_cert_path;
    std::optional<std::string> tls_key_path;
    bool require_tls = true;
    
    // Logging service configuration
    LoggingClientConfig logging;
    
    // Cache service configuration
    CacheClientConfig cache;
    
    // Key management configuration
    std::string kms_provider = "local";
    std::optional<std::string> hsm_slot_id;
    std::optional<std::string> aws_kms_key_arn;
    std::chrono::seconds key_cache_ttl{300};
    
    // Audit configuration
    std::string audit_log_path = "/var/log/crypto";
    
    // FIPS mode
    bool fips_mode = false;
    
    // Load from environment
    static Result<ServiceConfig> from_env();
    
    // Validate configuration
    Result<void> validate() const;
};

} // namespace crypto
```

### Key Metadata Model (Unchanged)

```cpp
// include/crypto/keys/key_types.h
namespace crypto {

struct KeyId {
    std::string namespace_id;
    std::string id;
    uint32_t version = 1;
    
    std::string to_string() const {
        return std::format("{}:{}:v{}", namespace_id, id, version);
    }
    
    std::string cache_key() const {
        return std::format("key:{}:{}:{}", namespace_id, id, version);
    }
};

enum class KeyType {
    AES,
    RSA,
    ECDSA
};

enum class KeyState {
    PENDING_ACTIVATION,
    ACTIVE,
    DEPRECATED,
    PENDING_DESTRUCTION,
    DESTROYED
};

struct KeyMetadata {
    KeyId id;
    KeyType type;
    KeyState state;
    std::chrono::system_clock::time_point created_at;
    std::optional<std::chrono::system_clock::time_point> expires_at;
    std::optional<std::chrono::system_clock::time_point> rotated_at;
    std::optional<KeyId> previous_version;
    std::string owner_service;
    std::vector<std::string> allowed_operations;
    uint64_t usage_count = 0;
};

} // namespace crypto
```

## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system—essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*

### Property 1: Log Entry Structure Completeness

*For any* cryptographic operation that generates a log entry, the log entry SHALL contain all required fields: correlation_id, trace_context, service_id, operation type, timestamp, and result status.

**Validates: Requirements 1.2, 1.4**

### Property 2: Key Caching Lifecycle Correctness

*For any* key operation sequence (load, use, rotate, delete), the cache state SHALL be consistent with the key store state:
- After load: cache contains key
- After rotate: old key invalidated, new key cached
- After delete: key removed from cache

**Validates: Requirements 2.2, 2.3, 2.4**

### Property 3: Trace Context Propagation

*For any* incoming request with W3C Trace Context headers, the Crypto_Service SHALL propagate the trace context to all outgoing requests (logging, cache) and include it in all generated spans.

**Validates: Requirements 3.6, 9.3**

### Property 4: Observability Metadata Completeness

*For any* operation that generates traces or logs, the metadata SHALL include correlation_id linking traces and logs for the same request.

**Validates: Requirements 9.2, 9.4**

### Property 5: Error Metric Emission

*For any* operation that fails with an error, the Crypto_Service SHALL emit a Prometheus counter metric with the error_code label set to the specific error code.

**Validates: Requirements 9.5**

### Property 6: Input Validation and Error Safety

*For any* input to a cryptographic operation, the Crypto_Service SHALL:
- Validate input sizes before processing
- Return errors that do not leak sensitive information (key material, plaintext)

**Validates: Requirements 10.5, 10.6**

### Property 7: Configuration Validation

*For any* configuration value provided via environment variables, the Crypto_Service SHALL validate the value at startup and fail fast with a descriptive error if invalid.

**Validates: Requirements 8.3**

## Error Handling

### Error Categories

1. **Crypto Errors**: Invalid key size, integrity failures, padding errors
   - Return specific error code
   - Log error with correlation_id (no sensitive data)
   - Emit error metric

2. **Service Errors**: Logging/cache service unavailable
   - Use fallback (local logging, local cache)
   - Log degraded mode
   - Emit circuit breaker metric

3. **Configuration Errors**: Missing/invalid configuration
   - Fail fast at startup
   - Log descriptive error message
   - Exit with non-zero code

4. **Input Validation Errors**: Oversized input, invalid format
   - Return INVALID_INPUT error
   - Do not process input
   - Log validation failure

### Error Response Format

```cpp
// gRPC error mapping
grpc::Status to_grpc_status(const Error& error) {
    switch (error.code) {
        case ErrorCode::INVALID_INPUT:
        case ErrorCode::INVALID_KEY_SIZE:
        case ErrorCode::INVALID_IV_SIZE:
            return grpc::Status(grpc::INVALID_ARGUMENT, error.message);
        case ErrorCode::NOT_FOUND:
            return grpc::Status(grpc::NOT_FOUND, error.message);
        case ErrorCode::PERMISSION_DENIED:
            return grpc::Status(grpc::PERMISSION_DENIED, error.message);
        case ErrorCode::SERVICE_UNAVAILABLE:
            return grpc::Status(grpc::UNAVAILABLE, error.message);
        case ErrorCode::INTEGRITY_ERROR:
            return grpc::Status(grpc::DATA_LOSS, "Integrity verification failed");
        default:
            return grpc::Status(grpc::INTERNAL, "Internal error");
    }
}
```

## Testing Strategy

### Test Organization

```
tests/
├── unit/
│   ├── crypto/
│   │   ├── engine/
│   │   │   ├── aes_engine_test.cpp
│   │   │   ├── rsa_engine_test.cpp
│   │   │   ├── ecdsa_engine_test.cpp
│   │   │   └── hybrid_encryption_test.cpp
│   │   ├── keys/
│   │   │   ├── key_service_test.cpp
│   │   │   └── key_store_test.cpp
│   │   ├── services/
│   │   │   ├── encryption_service_test.cpp
│   │   │   └── signature_service_test.cpp
│   │   └── clients/
│   │       ├── logging_client_test.cpp
│   │       └── cache_client_test.cpp
│   └── common/
│       ├── result_test.cpp
│       └── config_test.cpp
├── property/
│   ├── aes_properties_test.cpp
│   ├── rsa_properties_test.cpp
│   ├── signature_properties_test.cpp
│   ├── key_properties_test.cpp
│   ├── file_encryption_properties_test.cpp
│   ├── audit_properties_test.cpp
│   ├── logging_properties_test.cpp      # NEW
│   ├── cache_properties_test.cpp        # NEW
│   └── observability_properties_test.cpp # NEW
└── integration/
    ├── logging_service_integration_test.cpp
    ├── cache_service_integration_test.cpp
    └── end_to_end_test.cpp
```

### Property-Based Testing Configuration

- Framework: RapidCheck
- Minimum iterations: 100 per property
- Tag format: `Feature: crypto-service-modernization-2025, Property N: [description]`

### Unit Test Focus

- Specific examples demonstrating correct behavior
- Edge cases (empty input, max size, boundary values)
- Error conditions and error message validation
- Mock external dependencies (logging, cache clients)

### Integration Test Focus

- Real service communication (Testcontainers)
- End-to-end cryptographic operations
- Failover and fallback behavior
- Performance under load

## Files to Delete (Redundancy Elimination)

```
# Local logging (replaced by logging-service client)
include/crypto/logging/json_logger.h
src/crypto/logging/json_logger.cpp

# Local resilience (replaced by Service Mesh)
include/crypto/resilience/circuit_breaker.h
include/crypto/resilience/retry.h
src/crypto/resilience/circuit_breaker.cpp
src/crypto/resilience/retry.cpp

# Local key cache (replaced by cache-service client)
include/crypto/keys/key_cache.h
src/crypto/keys/key_cache.cpp
```

## Files to Create

```
# Platform clients
include/crypto/clients/logging_client.h
include/crypto/clients/cache_client.h
src/crypto/clients/logging_client.cpp
src/crypto/clients/cache_client.cpp

# Centralized utilities
include/crypto/common/openssl_raii.h
include/crypto/common/hash_utils.h

# New tests
tests/property/logging_properties_test.cpp
tests/property/cache_properties_test.cpp
tests/property/observability_properties_test.cpp
tests/integration/logging_service_integration_test.cpp
tests/integration/cache_service_integration_test.cpp

# Kubernetes manifests
deploy/kubernetes/service-mesh/crypto-service/resilience-policy.yaml
```

## Migration Strategy

1. **Phase 1**: Add new clients (LoggingClient, CacheClient) alongside existing implementations
2. **Phase 2**: Update services to use new clients with feature flags
3. **Phase 3**: Remove old implementations after validation
4. **Phase 4**: Update tests and documentation
