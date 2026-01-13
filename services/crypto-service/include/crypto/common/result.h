#pragma once

/**
 * @file result.h
 * @brief Modernized Result type using C++23 std::expected
 * 
 * This header provides a centralized error handling mechanism using
 * std::expected<T, Error> for type-safe, monadic error handling.
 * 
 * Requirements: 4.1, 4.2, 5.2, 5.6
 */

#include <expected>
#include <string>
#include <string_view>
#include <format>
#include <source_location>

namespace crypto {

// ============================================================================
// Error Codes - Centralized enumeration
// ============================================================================

/**
 * @brief Centralized error codes for all crypto operations
 */
enum class [[nodiscard]] ErrorCode {
    // Success
    OK = 0,
    
    // General errors (1-99)
    UNKNOWN_ERROR = 1,
    INVALID_INPUT = 2,
    INTERNAL_ERROR = 3,
    
    // Crypto errors (100-199)
    CRYPTO_ERROR = 100,
    INVALID_KEY_SIZE = 101,
    INVALID_IV_SIZE = 102,
    INVALID_TAG_SIZE = 103,
    INTEGRITY_ERROR = 104,
    PADDING_ERROR = 105,
    KEY_GENERATION_FAILED = 106,
    INVALID_KEY_TYPE = 107,
    SIZE_LIMIT_EXCEEDED = 108,
    SIGNATURE_INVALID = 109,
    ENCRYPTION_FAILED = 110,
    DECRYPTION_FAILED = 111,
    
    // Key management errors (200-299)
    KEY_NOT_FOUND = 200,
    KEY_DEPRECATED = 201,
    KEY_ROTATION_FAILED = 202,
    KEY_EXPIRED = 203,
    KEY_INVALID_STATE = 204,
    
    // Authentication/Authorization errors (300-399)
    AUTHENTICATION_FAILED = 300,
    AUTHORIZATION_FAILED = 301,
    PERMISSION_DENIED = 302,
    
    // Service errors (400-499)
    SERVICE_UNAVAILABLE = 400,
    TIMEOUT = 401,
    NOT_FOUND = 402,
    KMS_UNAVAILABLE = 403,
    
    // Cache errors (500-599)
    CACHE_MISS = 500,
    CACHE_ERROR = 501,
    CACHE_UNAVAILABLE = 502,
    
    // Logging errors (600-699)
    LOGGING_ERROR = 600,
    LOGGING_UNAVAILABLE = 601,
    
    // Configuration errors (700-799)
    CONFIG_ERROR = 700,
    CONFIG_MISSING = 701,
    CONFIG_INVALID = 702,
    
    // Audit errors (800-899)
    AUDIT_LOG_FAILED = 800
};

// ============================================================================
// Error Code Utilities
// ============================================================================

/**
 * @brief Convert error code to string representation
 * @param code Error code
 * @return String representation of the error code
 */
[[nodiscard]] constexpr std::string_view error_code_to_string(ErrorCode code) noexcept {
    switch (code) {
        case ErrorCode::OK: return "OK";
        case ErrorCode::UNKNOWN_ERROR: return "UNKNOWN_ERROR";
        case ErrorCode::INVALID_INPUT: return "INVALID_INPUT";
        case ErrorCode::INTERNAL_ERROR: return "INTERNAL_ERROR";
        case ErrorCode::CRYPTO_ERROR: return "CRYPTO_ERROR";
        case ErrorCode::INVALID_KEY_SIZE: return "INVALID_KEY_SIZE";
        case ErrorCode::INVALID_IV_SIZE: return "INVALID_IV_SIZE";
        case ErrorCode::INVALID_TAG_SIZE: return "INVALID_TAG_SIZE";
        case ErrorCode::INTEGRITY_ERROR: return "INTEGRITY_ERROR";
        case ErrorCode::PADDING_ERROR: return "PADDING_ERROR";
        case ErrorCode::KEY_GENERATION_FAILED: return "KEY_GENERATION_FAILED";
        case ErrorCode::INVALID_KEY_TYPE: return "INVALID_KEY_TYPE";
        case ErrorCode::SIZE_LIMIT_EXCEEDED: return "SIZE_LIMIT_EXCEEDED";
        case ErrorCode::SIGNATURE_INVALID: return "SIGNATURE_INVALID";
        case ErrorCode::ENCRYPTION_FAILED: return "ENCRYPTION_FAILED";
        case ErrorCode::DECRYPTION_FAILED: return "DECRYPTION_FAILED";
        case ErrorCode::KEY_NOT_FOUND: return "KEY_NOT_FOUND";
        case ErrorCode::KEY_DEPRECATED: return "KEY_DEPRECATED";
        case ErrorCode::KEY_ROTATION_FAILED: return "KEY_ROTATION_FAILED";
        case ErrorCode::KEY_EXPIRED: return "KEY_EXPIRED";
        case ErrorCode::KEY_INVALID_STATE: return "KEY_INVALID_STATE";
        case ErrorCode::AUTHENTICATION_FAILED: return "AUTHENTICATION_FAILED";
        case ErrorCode::AUTHORIZATION_FAILED: return "AUTHORIZATION_FAILED";
        case ErrorCode::PERMISSION_DENIED: return "PERMISSION_DENIED";
        case ErrorCode::SERVICE_UNAVAILABLE: return "SERVICE_UNAVAILABLE";
        case ErrorCode::TIMEOUT: return "TIMEOUT";
        case ErrorCode::NOT_FOUND: return "NOT_FOUND";
        case ErrorCode::KMS_UNAVAILABLE: return "KMS_UNAVAILABLE";
        case ErrorCode::CACHE_MISS: return "CACHE_MISS";
        case ErrorCode::CACHE_ERROR: return "CACHE_ERROR";
        case ErrorCode::CACHE_UNAVAILABLE: return "CACHE_UNAVAILABLE";
        case ErrorCode::LOGGING_ERROR: return "LOGGING_ERROR";
        case ErrorCode::LOGGING_UNAVAILABLE: return "LOGGING_UNAVAILABLE";
        case ErrorCode::CONFIG_ERROR: return "CONFIG_ERROR";
        case ErrorCode::CONFIG_MISSING: return "CONFIG_MISSING";
        case ErrorCode::CONFIG_INVALID: return "CONFIG_INVALID";
        case ErrorCode::AUDIT_LOG_FAILED: return "AUDIT_LOG_FAILED";
        default: return "UNKNOWN";
    }
}

/**
 * @brief Check if an error is retryable
 * @param code Error code
 * @return true if the operation can be retried
 */
[[nodiscard]] constexpr bool is_retryable(ErrorCode code) noexcept {
    switch (code) {
        case ErrorCode::SERVICE_UNAVAILABLE:
        case ErrorCode::TIMEOUT:
        case ErrorCode::KMS_UNAVAILABLE:
        case ErrorCode::CACHE_UNAVAILABLE:
        case ErrorCode::LOGGING_UNAVAILABLE:
            return true;
        default:
            return false;
    }
}

/**
 * @brief Check if an error is a client error (4xx equivalent)
 * @param code Error code
 * @return true if the error is due to client input
 */
[[nodiscard]] constexpr bool is_client_error(ErrorCode code) noexcept {
    switch (code) {
        case ErrorCode::INVALID_INPUT:
        case ErrorCode::INVALID_KEY_SIZE:
        case ErrorCode::INVALID_IV_SIZE:
        case ErrorCode::INVALID_TAG_SIZE:
        case ErrorCode::SIZE_LIMIT_EXCEEDED:
        case ErrorCode::AUTHENTICATION_FAILED:
        case ErrorCode::AUTHORIZATION_FAILED:
        case ErrorCode::PERMISSION_DENIED:
        case ErrorCode::NOT_FOUND:
        case ErrorCode::KEY_NOT_FOUND:
            return true;
        default:
            return false;
    }
}

// ============================================================================
// Error Structure
// ============================================================================

/**
 * @brief Error structure with code, message, and correlation ID
 */
struct Error {
    ErrorCode code;
    std::string message;
    std::string correlation_id;
    
    /**
     * @brief Construct an error with code and optional message
     */
    constexpr Error(ErrorCode c, std::string msg = "", std::string corr_id = "")
        : code(c)
        , message(std::move(msg))
        , correlation_id(std::move(corr_id)) {}
    
    /**
     * @brief Check if this error is retryable
     */
    [[nodiscard]] constexpr bool is_retryable() const noexcept {
        return crypto::is_retryable(code);
    }
    
    /**
     * @brief Check if this is a client error
     */
    [[nodiscard]] constexpr bool is_client_error() const noexcept {
        return crypto::is_client_error(code);
    }
    
    /**
     * @brief Get the error code as a string
     */
    [[nodiscard]] constexpr std::string_view code_string() const noexcept {
        return error_code_to_string(code);
    }
    
    /**
     * @brief Format error for logging (no sensitive data)
     */
    [[nodiscard]] std::string to_log_string() const {
        if (correlation_id.empty()) {
            return std::format("[{}] {}", code_string(), message);
        }
        return std::format("[{}] {} (correlation_id={})", 
                          code_string(), message, correlation_id);
    }
    
    /**
     * @brief Equality comparison
     */
    [[nodiscard]] bool operator==(const Error& other) const noexcept {
        return code == other.code;
    }
};

// ============================================================================
// Result Wrapper with backward-compatible methods
// ============================================================================

/**
 * @brief Wrapper class that adds member methods to std::expected
 * 
 * This allows test code to use result.is_error() and result.error_code()
 * syntax while still using std::expected internally.
 */
template<typename T>
class ResultWrapper : public std::expected<T, Error> {
public:
    using Base = std::expected<T, Error>;
    using Base::Base;
    
    // Implicit conversion from std::expected
    ResultWrapper(const Base& base) : Base(base) {}
    ResultWrapper(Base&& base) : Base(std::move(base)) {}
    
    // Implicit conversion from std::unexpected
    ResultWrapper(std::unexpected<Error> err) : Base(std::move(err)) {}
    
    /**
     * @brief Check if the result contains an error
     */
    [[nodiscard]] constexpr bool is_error() const noexcept {
        return !this->has_value();
    }
    
    /**
     * @brief Get the error code
     */
    [[nodiscard]] constexpr ErrorCode error_code() const noexcept {
        return this->error().code;
    }
};

// ============================================================================
// Result Type Alias
// ============================================================================

/**
 * @brief Result type using std::expected for modern error handling
 * 
 * Usage:
 *   Result<std::vector<uint8_t>> encrypt(std::span<const uint8_t> data);
 *   
 *   auto result = encrypt(data);
 *   if (result) {
 *       // Use result.value()
 *   } else {
 *       // Handle result.error()
 *   }
 *   
 * Backward-compatible methods:
 *   result.is_error()   - returns true if result contains an error
 *   result.error_code() - returns the ErrorCode from the error
 */
template<typename T>
using Result = ResultWrapper<T>;

// ============================================================================
// Helper Functions
// ============================================================================

/**
 * @brief Create a successful result
 * @param value The success value
 * @return Result containing the value
 */
template<typename T>
[[nodiscard]] constexpr Result<T> Ok(T value) {
    return Result<T>(std::move(value));
}

/**
 * @brief Create a successful void result
 * @return Result<void> indicating success
 */
[[nodiscard]] inline Result<void> Ok() {
    return Result<void>();
}

/**
 * @brief Create an error result
 * @param code Error code
 * @param message Error message (should not contain sensitive data)
 * @param correlation_id Optional correlation ID for tracing
 * @return Result containing the error
 */
template<typename T = void>
[[nodiscard]] constexpr Result<T> Err(
    ErrorCode code, 
    std::string message = "", 
    std::string correlation_id = "") {
    return std::unexpected(Error(code, std::move(message), std::move(correlation_id)));
}

/**
 * @brief Create an error result from an existing Error
 * @param error The error
 * @return Result containing the error
 */
template<typename T = void>
[[nodiscard]] constexpr Result<T> Err(Error error) {
    return std::unexpected(std::move(error));
}

/**
 * @brief Create an error result with source location for debugging
 * @param code Error code
 * @param message Error message
 * @param loc Source location (auto-captured)
 * @return Result containing the error with location info
 */
template<typename T = void>
[[nodiscard]] Result<T> ErrWithLocation(
    ErrorCode code,
    std::string message = "",
    const std::source_location& loc = std::source_location::current()) {
    
    auto full_message = std::format("{} (at {}:{})", 
                                    message, 
                                    loc.file_name(), 
                                    loc.line());
    return std::unexpected(Error(code, std::move(full_message)));
}

// ============================================================================
// Result Combinators
// ============================================================================

/**
 * @brief Transform a Result<T> to Result<U> using a function
 * @param result The input result
 * @param f Function T -> U
 * @return Result<U>
 */
template<typename T, typename F>
[[nodiscard]] auto transform(const Result<T>& result, F&& f) 
    -> Result<decltype(f(std::declval<T>()))> {
    
    using U = decltype(f(std::declval<T>()));
    if (result) {
        return Result<U>(f(*result));
    }
    return std::unexpected(result.error());
}

/**
 * @brief Chain Result operations (flatMap/bind)
 * @param result The input result
 * @param f Function T -> Result<U>
 * @return Result<U>
 */
template<typename T, typename F>
[[nodiscard]] auto and_then(const Result<T>& result, F&& f) 
    -> decltype(f(std::declval<T>())) {
    
    if (result) {
        return f(*result);
    }
    return std::unexpected(result.error());
}

/**
 * @brief Provide a fallback for error cases
 * @param result The input result
 * @param f Function Error -> Result<T>
 * @return Result<T>
 */
template<typename T, typename F>
[[nodiscard]] auto or_else(const Result<T>& result, F&& f) -> Result<T> {
    if (result) {
        return result;
    }
    return f(result.error());
}

// ============================================================================
// Backward Compatibility (deprecated)
// ============================================================================

// Keep old function name for migration period
[[deprecated("Use error_code_to_string instead")]]
inline const char* errorCodeToString(ErrorCode code) {
    return error_code_to_string(code).data();
}

} // namespace crypto
