#pragma once

/**
 * @file file_encryption_service.h
 * @brief File encryption service with streaming and LoggingClient integration
 * 
 * Requirements: 1.2, 1.4
 */

#include "crypto/common/result.h"
#include "crypto/engine/aes_engine.h"
#include "crypto/keys/key_service.h"
#include "crypto/clients/logging_client.h"
#include <memory>
#include <string>
#include <span>
#include <fstream>
#include <functional>

namespace crypto {

// File encryption header format
struct FileEncryptionHeader {
    static constexpr uint32_t MAGIC = 0x43525950;  // "CRYP"
    static constexpr uint16_t VERSION = 1;
    
    uint32_t magic;
    uint16_t version;
    uint16_t algorithm;  // 1 = AES-256-GCM
    KeyId key_id;
    std::vector<uint8_t> wrapped_dek;  // DEK encrypted with KEK
    std::vector<uint8_t> iv;
    std::vector<uint8_t> tag;
    uint64_t original_size;
    uint32_t chunk_size;
    
    [[nodiscard]] std::vector<uint8_t> serialize() const;
    [[nodiscard]] static Result<FileEncryptionHeader> deserialize(std::span<const uint8_t> data);
};

// File encryption context
struct FileEncryptionContext {
    std::string correlation_id;
    std::string caller_identity;
    std::string caller_service;
    std::string source_ip;
};

// Progress callback
using ProgressCallback = std::function<void(uint64_t bytes_processed, uint64_t total_bytes)>;

// File encryption service with streaming support
class FileEncryptionService {
public:
    static constexpr size_t DEFAULT_CHUNK_SIZE = 64 * 1024;  // 64KB chunks
    
    FileEncryptionService(std::shared_ptr<KeyService> key_service,
                          std::shared_ptr<LoggingClient> logging_client);
    ~FileEncryptionService() = default;

    // Encrypt file using streaming
    [[nodiscard]] Result<void> encryptFile(
        const std::string& input_path,
        const std::string& output_path,
        const KeyId& kek_id,  // Key Encryption Key
        const FileEncryptionContext& ctx,
        ProgressCallback progress = nullptr);

    // Decrypt file using streaming
    [[nodiscard]] Result<void> decryptFile(
        const std::string& input_path,
        const std::string& output_path,
        const FileEncryptionContext& ctx,
        ProgressCallback progress = nullptr);

    // Encrypt data stream
    [[nodiscard]] Result<void> encryptStream(
        std::istream& input,
        std::ostream& output,
        const KeyId& kek_id,
        const FileEncryptionContext& ctx,
        uint64_t input_size = 0,
        ProgressCallback progress = nullptr);

    // Decrypt data stream
    [[nodiscard]] Result<void> decryptStream(
        std::istream& input,
        std::ostream& output,
        const FileEncryptionContext& ctx,
        ProgressCallback progress = nullptr);

    // Get file header without decrypting
    [[nodiscard]] Result<FileEncryptionHeader> readHeader(const std::string& file_path);

    // Set chunk size for streaming
    void setChunkSize(size_t size) { chunk_size_ = size; }

private:
    std::shared_ptr<KeyService> key_service_;
    std::shared_ptr<LoggingClient> logging_client_;
    AESEngine aes_engine_;
    size_t chunk_size_ = DEFAULT_CHUNK_SIZE;

    [[nodiscard]] Result<std::vector<uint8_t>> generateDEK();
    [[nodiscard]] Result<std::vector<uint8_t>> wrapDEK(std::span<const uint8_t> dek, const KeyId& kek_id);
    [[nodiscard]] Result<std::vector<uint8_t>> unwrapDEK(std::span<const uint8_t> wrapped, const KeyId& kek_id);
    
    void logOperation(std::string_view operation, const KeyId& key_id,
                      const FileEncryptionContext& ctx, bool success,
                      const std::optional<std::string>& error = std::nullopt);
};

} // namespace crypto
