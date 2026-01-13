#pragma once

#include "crypto/common/result.h"
#include "crypto/keys/key_types.h"
#include <string>
#include <vector>
#include <chrono>
#include <map>
#include <memory>
#include <mutex>
#include <functional>

namespace crypto {

// Audit operation types
enum class AuditOperation {
    ENCRYPT,
    DECRYPT,
    RSA_ENCRYPT,
    RSA_DECRYPT,
    SIGN,
    VERIFY,
    KEY_GENERATE,
    KEY_ROTATE,
    KEY_DELETE,
    KEY_ACCESS,
    FILE_ENCRYPT,
    FILE_DECRYPT
};

const char* auditOperationToString(AuditOperation op);

// Audit log entry
struct AuditEntry {
    std::string correlation_id;
    std::chrono::system_clock::time_point timestamp;
    AuditOperation operation;
    KeyId key_id;
    std::string caller_identity;
    std::string caller_service;
    bool success;
    std::optional<std::string> error_code;
    std::string source_ip;
    std::map<std::string, std::string> metadata;

    // Serialize to JSON (for storage/export)
    std::string toJson() const;
    
    // Parse from JSON
    static Result<AuditEntry> fromJson(const std::string& json);
};

// Audit query parameters
struct AuditQuery {
    std::optional<std::chrono::system_clock::time_point> start_time;
    std::optional<std::chrono::system_clock::time_point> end_time;
    std::optional<AuditOperation> operation;
    std::optional<KeyId> key_id;
    std::optional<std::string> caller_identity;
    std::optional<bool> success;
    size_t limit = 100;
    size_t offset = 0;
};

// Audit logger interface
class IAuditLogger {
public:
    virtual ~IAuditLogger() = default;

    virtual void logOperation(const AuditEntry& entry) = 0;
    virtual Result<std::vector<AuditEntry>> query(const AuditQuery& query) = 0;
    virtual Result<void> exportToJson(const std::string& path, const AuditQuery& query) = 0;
};

// In-memory audit logger (for testing)
class InMemoryAuditLogger : public IAuditLogger {
public:
    InMemoryAuditLogger() = default;
    ~InMemoryAuditLogger() override = default;

    void logOperation(const AuditEntry& entry) override;
    Result<std::vector<AuditEntry>> query(const AuditQuery& query) override;
    Result<void> exportToJson(const std::string& path, const AuditQuery& query) override;

    // Get all entries (for testing)
    std::vector<AuditEntry> getAllEntries() const;
    void clear();

private:
    mutable std::mutex mutex_;
    std::vector<AuditEntry> entries_;
};

// File-based audit logger with encryption
class FileAuditLogger : public IAuditLogger {
public:
    FileAuditLogger(const std::string& log_path, 
                    std::span<const uint8_t> encryption_key,
                    bool encrypt_logs = true);
    ~FileAuditLogger() override = default;

    void logOperation(const AuditEntry& entry) override;
    Result<std::vector<AuditEntry>> query(const AuditQuery& query) override;
    Result<void> exportToJson(const std::string& path, const AuditQuery& query) override;

private:
    std::string log_path_;
    std::vector<uint8_t> encryption_key_;
    bool encrypt_logs_;
    mutable std::mutex mutex_;

    std::string getCurrentLogFile() const;
    Result<void> writeEntry(const AuditEntry& entry);
    Result<std::vector<AuditEntry>> readEntries(const std::string& file_path);
};

// Audit logger builder for creating entries
class AuditEntryBuilder {
public:
    AuditEntryBuilder& setCorrelationId(const std::string& id);
    AuditEntryBuilder& setOperation(AuditOperation op);
    AuditEntryBuilder& setKeyId(const KeyId& id);
    AuditEntryBuilder& setCallerIdentity(const std::string& identity);
    AuditEntryBuilder& setCallerService(const std::string& service);
    AuditEntryBuilder& setSuccess(bool success);
    AuditEntryBuilder& setErrorCode(const std::string& code);
    AuditEntryBuilder& setSourceIp(const std::string& ip);
    AuditEntryBuilder& addMetadata(const std::string& key, const std::string& value);

    AuditEntry build() const;

private:
    AuditEntry entry_;
};

// RAII helper for automatic audit logging
class ScopedAuditLog {
public:
    ScopedAuditLog(IAuditLogger& logger, AuditEntryBuilder builder);
    ~ScopedAuditLog();

    void setSuccess(bool success);
    void setErrorCode(const std::string& code);

private:
    IAuditLogger& logger_;
    AuditEntryBuilder builder_;
    bool success_ = false;
    std::optional<std::string> error_code_;
};

} // namespace crypto
