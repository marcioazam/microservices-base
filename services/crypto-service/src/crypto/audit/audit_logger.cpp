#include "crypto/audit/audit_logger.h"
#include "crypto/engine/aes_engine.h"
#include <fstream>
#include <sstream>
#include <iomanip>
#include <filesystem>
#include <algorithm>

namespace crypto {

const char* auditOperationToString(AuditOperation op) {
    switch (op) {
        case AuditOperation::ENCRYPT: return "ENCRYPT";
        case AuditOperation::DECRYPT: return "DECRYPT";
        case AuditOperation::RSA_ENCRYPT: return "RSA_ENCRYPT";
        case AuditOperation::RSA_DECRYPT: return "RSA_DECRYPT";
        case AuditOperation::SIGN: return "SIGN";
        case AuditOperation::VERIFY: return "VERIFY";
        case AuditOperation::KEY_GENERATE: return "KEY_GENERATE";
        case AuditOperation::KEY_ROTATE: return "KEY_ROTATE";
        case AuditOperation::KEY_DELETE: return "KEY_DELETE";
        case AuditOperation::KEY_ACCESS: return "KEY_ACCESS";
        case AuditOperation::FILE_ENCRYPT: return "FILE_ENCRYPT";
        case AuditOperation::FILE_DECRYPT: return "FILE_DECRYPT";
        default: return "UNKNOWN";
    }
}

namespace {

std::string escapeJson(const std::string& str) {
    std::ostringstream oss;
    for (char c : str) {
        switch (c) {
            case '"': oss << "\\\""; break;
            case '\\': oss << "\\\\"; break;
            case '\b': oss << "\\b"; break;
            case '\f': oss << "\\f"; break;
            case '\n': oss << "\\n"; break;
            case '\r': oss << "\\r"; break;
            case '\t': oss << "\\t"; break;
            default:
                if (c < 0x20) {
                    oss << "\\u" << std::hex << std::setw(4) << std::setfill('0') << static_cast<int>(c);
                } else {
                    oss << c;
                }
        }
    }
    return oss.str();
}

std::string timePointToIso8601(std::chrono::system_clock::time_point tp) {
    auto time_t = std::chrono::system_clock::to_time_t(tp);
    std::tm tm = *std::gmtime(&time_t);
    std::ostringstream oss;
    oss << std::put_time(&tm, "%Y-%m-%dT%H:%M:%SZ");
    return oss.str();
}

} // anonymous namespace

std::string AuditEntry::toJson() const {
    std::ostringstream oss;
    oss << "{";
    oss << "\"correlation_id\":\"" << escapeJson(correlation_id) << "\",";
    oss << "\"timestamp\":\"" << timePointToIso8601(timestamp) << "\",";
    oss << "\"operation\":\"" << auditOperationToString(operation) << "\",";
    oss << "\"key_id\":\"" << escapeJson(key_id.toString()) << "\",";
    oss << "\"caller_identity\":\"" << escapeJson(caller_identity) << "\",";
    oss << "\"caller_service\":\"" << escapeJson(caller_service) << "\",";
    oss << "\"success\":" << (success ? "true" : "false") << ",";
    
    if (error_code) {
        oss << "\"error_code\":\"" << escapeJson(*error_code) << "\",";
    }
    
    oss << "\"source_ip\":\"" << escapeJson(source_ip) << "\",";
    
    oss << "\"metadata\":{";
    bool first = true;
    for (const auto& [key, value] : metadata) {
        if (!first) oss << ",";
        oss << "\"" << escapeJson(key) << "\":\"" << escapeJson(value) << "\"";
        first = false;
    }
    oss << "}";
    
    oss << "}";
    return oss.str();
}

Result<AuditEntry> AuditEntry::fromJson(const std::string& /*json*/) {
    // Simplified implementation - in production, use a proper JSON parser
    return Err<AuditEntry>(ErrorCode::INTERNAL_ERROR, "JSON parsing not implemented");
}

// InMemoryAuditLogger implementation
void InMemoryAuditLogger::logOperation(const AuditEntry& entry) {
    std::lock_guard<std::mutex> lock(mutex_);
    entries_.push_back(entry);
}

Result<std::vector<AuditEntry>> InMemoryAuditLogger::query(const AuditQuery& query) {
    std::lock_guard<std::mutex> lock(mutex_);
    
    std::vector<AuditEntry> results;
    
    for (const auto& entry : entries_) {
        // Apply filters
        if (query.start_time && entry.timestamp < *query.start_time) continue;
        if (query.end_time && entry.timestamp > *query.end_time) continue;
        if (query.operation && entry.operation != *query.operation) continue;
        if (query.key_id && entry.key_id != *query.key_id) continue;
        if (query.caller_identity && entry.caller_identity != *query.caller_identity) continue;
        if (query.success && entry.success != *query.success) continue;
        
        results.push_back(entry);
    }
    
    // Apply pagination
    if (query.offset < results.size()) {
        auto start = results.begin() + query.offset;
        auto end = (query.offset + query.limit < results.size()) 
            ? start + query.limit 
            : results.end();
        results = std::vector<AuditEntry>(start, end);
    } else {
        results.clear();
    }
    
    return Ok(std::move(results));
}

Result<void> InMemoryAuditLogger::exportToJson(const std::string& path, const AuditQuery& query) {
    auto entries_result = this->query(query);
    if (!entries_result) {
        return Err<void>(entries_result.error());
    }
    
    std::ofstream file(path);
    if (!file) {
        return Err<void>(ErrorCode::INTERNAL_ERROR, "Failed to open file for writing");
    }
    
    file << "[\n";
    bool first = true;
    for (const auto& entry : *entries_result) {
        if (!first) file << ",\n";
        file << "  " << entry.toJson();
        first = false;
    }
    file << "\n]";
    
    return Ok();
}

std::vector<AuditEntry> InMemoryAuditLogger::getAllEntries() const {
    std::lock_guard<std::mutex> lock(mutex_);
    return entries_;
}

void InMemoryAuditLogger::clear() {
    std::lock_guard<std::mutex> lock(mutex_);
    entries_.clear();
}

// FileAuditLogger implementation
FileAuditLogger::FileAuditLogger(const std::string& log_path,
                                 std::span<const uint8_t> encryption_key,
                                 bool encrypt_logs)
    : log_path_(log_path)
    , encryption_key_(encryption_key.begin(), encryption_key.end())
    , encrypt_logs_(encrypt_logs) {
    std::filesystem::create_directories(log_path_);
}

std::string FileAuditLogger::getCurrentLogFile() const {
    auto now = std::chrono::system_clock::now();
    auto time_t = std::chrono::system_clock::to_time_t(now);
    std::tm tm = *std::gmtime(&time_t);
    
    std::ostringstream oss;
    oss << log_path_ << "/audit_" 
        << std::put_time(&tm, "%Y%m%d") << ".log";
    return oss.str();
}

void FileAuditLogger::logOperation(const AuditEntry& entry) {
    std::lock_guard<std::mutex> lock(mutex_);
    writeEntry(entry);
}

Result<void> FileAuditLogger::writeEntry(const AuditEntry& entry) {
    std::string json = entry.toJson();
    
    std::string data_to_write;
    if (encrypt_logs_ && !encryption_key_.empty()) {
        AESEngine aes;
        auto encrypt_result = aes.encryptGCM(
            std::span<const uint8_t>(
                reinterpret_cast<const uint8_t*>(json.data()), 
                json.size()
            ),
            encryption_key_
        );
        if (!encrypt_result) {
            return Err<void>(encrypt_result.error());
        }
        
        // Format: [iv_len][iv][tag_len][tag][data_len][data]
        std::ostringstream oss;
        uint32_t iv_len = static_cast<uint32_t>(encrypt_result->iv.size());
        uint32_t tag_len = static_cast<uint32_t>(encrypt_result->tag.size());
        uint32_t data_len = static_cast<uint32_t>(encrypt_result->ciphertext.size());
        
        oss.write(reinterpret_cast<const char*>(&iv_len), sizeof(iv_len));
        oss.write(reinterpret_cast<const char*>(encrypt_result->iv.data()), iv_len);
        oss.write(reinterpret_cast<const char*>(&tag_len), sizeof(tag_len));
        oss.write(reinterpret_cast<const char*>(encrypt_result->tag.data()), tag_len);
        oss.write(reinterpret_cast<const char*>(&data_len), sizeof(data_len));
        oss.write(reinterpret_cast<const char*>(encrypt_result->ciphertext.data()), data_len);
        
        data_to_write = oss.str();
    } else {
        data_to_write = json + "\n";
    }
    
    std::ofstream file(getCurrentLogFile(), std::ios::app | std::ios::binary);
    if (!file) {
        return Err<void>(ErrorCode::AUDIT_LOG_FAILED, "Failed to open audit log file");
    }
    
    file.write(data_to_write.data(), data_to_write.size());
    if (!file) {
        return Err<void>(ErrorCode::AUDIT_LOG_FAILED, "Failed to write audit log entry");
    }
    
    return Ok();
}

Result<std::vector<AuditEntry>> FileAuditLogger::query(const AuditQuery& /*query*/) {
    // Simplified implementation - in production, implement proper querying
    return Ok(std::vector<AuditEntry>());
}

Result<void> FileAuditLogger::exportToJson(const std::string& path, const AuditQuery& query) {
    auto entries_result = this->query(query);
    if (!entries_result) {
        return Err<void>(entries_result.error());
    }
    
    std::ofstream file(path);
    if (!file) {
        return Err<void>(ErrorCode::INTERNAL_ERROR, "Failed to open file for writing");
    }
    
    file << "[\n";
    bool first = true;
    for (const auto& entry : *entries_result) {
        if (!first) file << ",\n";
        file << "  " << entry.toJson();
        first = false;
    }
    file << "\n]";
    
    return Ok();
}

// AuditEntryBuilder implementation
AuditEntryBuilder& AuditEntryBuilder::setCorrelationId(const std::string& id) {
    entry_.correlation_id = id;
    return *this;
}

AuditEntryBuilder& AuditEntryBuilder::setOperation(AuditOperation op) {
    entry_.operation = op;
    return *this;
}

AuditEntryBuilder& AuditEntryBuilder::setKeyId(const KeyId& id) {
    entry_.key_id = id;
    return *this;
}

AuditEntryBuilder& AuditEntryBuilder::setCallerIdentity(const std::string& identity) {
    entry_.caller_identity = identity;
    return *this;
}

AuditEntryBuilder& AuditEntryBuilder::setCallerService(const std::string& service) {
    entry_.caller_service = service;
    return *this;
}

AuditEntryBuilder& AuditEntryBuilder::setSuccess(bool success) {
    entry_.success = success;
    return *this;
}

AuditEntryBuilder& AuditEntryBuilder::setErrorCode(const std::string& code) {
    entry_.error_code = code;
    return *this;
}

AuditEntryBuilder& AuditEntryBuilder::setSourceIp(const std::string& ip) {
    entry_.source_ip = ip;
    return *this;
}

AuditEntryBuilder& AuditEntryBuilder::addMetadata(const std::string& key, const std::string& value) {
    entry_.metadata[key] = value;
    return *this;
}

AuditEntry AuditEntryBuilder::build() const {
    AuditEntry result = entry_;
    result.timestamp = std::chrono::system_clock::now();
    return result;
}

// ScopedAuditLog implementation
ScopedAuditLog::ScopedAuditLog(IAuditLogger& logger, AuditEntryBuilder builder)
    : logger_(logger), builder_(std::move(builder)) {}

ScopedAuditLog::~ScopedAuditLog() {
    builder_.setSuccess(success_);
    if (error_code_) {
        builder_.setErrorCode(*error_code_);
    }
    logger_.logOperation(builder_.build());
}

void ScopedAuditLog::setSuccess(bool success) {
    success_ = success;
}

void ScopedAuditLog::setErrorCode(const std::string& code) {
    error_code_ = code;
}

} // namespace crypto
