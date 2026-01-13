#include "crypto/services/file_encryption_service.h"
#include <openssl/rand.h>
#include <cstring>
#include <filesystem>

namespace crypto {

// FileEncryptionHeader serialization
std::vector<uint8_t> FileEncryptionHeader::serialize() const {
    std::vector<uint8_t> result;
    
    auto write_u32 = [&](uint32_t v) {
        result.push_back(static_cast<uint8_t>(v & 0xFF));
        result.push_back(static_cast<uint8_t>((v >> 8) & 0xFF));
        result.push_back(static_cast<uint8_t>((v >> 16) & 0xFF));
        result.push_back(static_cast<uint8_t>((v >> 24) & 0xFF));
    };
    
    auto write_u16 = [&](uint16_t v) {
        result.push_back(static_cast<uint8_t>(v & 0xFF));
        result.push_back(static_cast<uint8_t>((v >> 8) & 0xFF));
    };
    
    auto write_u64 = [&](uint64_t v) {
        for (int i = 0; i < 8; ++i) {
            result.push_back(static_cast<uint8_t>((v >> (i * 8)) & 0xFF));
        }
    };
    
    auto write_bytes = [&](const std::vector<uint8_t>& data) {
        write_u32(static_cast<uint32_t>(data.size()));
        result.insert(result.end(), data.begin(), data.end());
    };
    
    write_u32(magic);
    write_u16(version);
    write_u16(algorithm);
    
    // Serialize KeyId
    auto key_str = key_id.toString();
    write_u32(static_cast<uint32_t>(key_str.size()));
    result.insert(result.end(), key_str.begin(), key_str.end());
    
    write_bytes(wrapped_dek);
    write_bytes(iv);
    write_bytes(tag);
    write_u64(original_size);
    write_u32(chunk_size);
    
    return result;
}

Result<FileEncryptionHeader> FileEncryptionHeader::deserialize(std::span<const uint8_t> data) {
    if (data.size() < 20) {
        return Err<FileEncryptionHeader>(ErrorCode::INVALID_INPUT, "Header too small");
    }
    
    size_t pos = 0;
    
    auto read_u32 = [&]() -> uint32_t {
        uint32_t v = data[pos] | (data[pos+1] << 8) | 
                     (data[pos+2] << 16) | (data[pos+3] << 24);
        pos += 4;
        return v;
    };
    
    auto read_u16 = [&]() -> uint16_t {
        uint16_t v = data[pos] | (data[pos+1] << 8);
        pos += 2;
        return v;
    };
    
    auto read_u64 = [&]() -> uint64_t {
        uint64_t v = 0;
        for (int i = 0; i < 8; ++i) {
            v |= static_cast<uint64_t>(data[pos + i]) << (i * 8);
        }
        pos += 8;
        return v;
    };
    
    auto read_bytes = [&]() -> std::vector<uint8_t> {
        uint32_t len = read_u32();
        std::vector<uint8_t> result(data.begin() + pos, data.begin() + pos + len);
        pos += len;
        return result;
    };
    
    FileEncryptionHeader header;
    header.magic = read_u32();
    
    if (header.magic != MAGIC) {
        return Err<FileEncryptionHeader>(ErrorCode::INVALID_INPUT, "Invalid magic number");
    }
    
    header.version = read_u16();
    header.algorithm = read_u16();
    
    uint32_t key_len = read_u32();
    std::string key_str(data.begin() + pos, data.begin() + pos + key_len);
    pos += key_len;
    
    auto key_result = KeyId::parse(key_str);
    if (!key_result) {
        return Err<FileEncryptionHeader>(ErrorCode::INVALID_INPUT, "Invalid key ID");
    }
    header.key_id = *key_result;
    
    header.wrapped_dek = read_bytes();
    header.iv = read_bytes();
    header.tag = read_bytes();
    header.original_size = read_u64();
    header.chunk_size = read_u32();
    
    return Ok(std::move(header));
}

FileEncryptionService::FileEncryptionService(
    std::shared_ptr<KeyService> key_service,
    std::shared_ptr<LoggingClient> logging_client)
    : key_service_(std::move(key_service))
    , logging_client_(std::move(logging_client)) {}

void FileEncryptionService::logOperation(
    std::string_view operation, const KeyId& key_id,
    const FileEncryptionContext& ctx, bool success,
    const std::optional<std::string>& error) {
    
    if (!logging_client_) return;
    
    std::map<std::string, std::string> fields = {
        {"operation", std::string(operation)},
        {"key_id", key_id.toString()},
        {"caller_identity", ctx.caller_identity},
        {"caller_service", ctx.caller_service},
        {"source_ip", ctx.source_ip},
        {"success", success ? "true" : "false"}
    };
    
    if (error) {
        fields["error_code"] = *error;
    }
    
    if (success) {
        logging_client_->log(LogLevel::INFO,
            std::string(operation) + " operation completed",
            ctx.correlation_id, fields);
    } else {
        logging_client_->log(LogLevel::ERROR,
            std::string(operation) + " operation failed",
            ctx.correlation_id, fields);
    }
}

Result<std::vector<uint8_t>> FileEncryptionService::generateDEK() {
    std::vector<uint8_t> dek(32);  // 256-bit DEK
    if (RAND_bytes(dek.data(), static_cast<int>(dek.size())) != 1) {
        return Err<std::vector<uint8_t>>(ErrorCode::CRYPTO_ERROR, "Failed to generate DEK");
    }
    return Ok(std::move(dek));
}

Result<std::vector<uint8_t>> FileEncryptionService::wrapDEK(
    std::span<const uint8_t> dek, const KeyId& kek_id) {
    
    auto kek_result = key_service_->getKey(kek_id);
    if (!kek_result) {
        return Err<std::vector<uint8_t>>(kek_result.error());
    }
    
    auto encrypt_result = aes_engine_.encryptGCM(dek, kek_result->key_material);
    if (!encrypt_result) {
        return Err<std::vector<uint8_t>>(encrypt_result.error());
    }
    
    // Combine IV + tag + ciphertext for wrapped DEK
    std::vector<uint8_t> wrapped;
    wrapped.insert(wrapped.end(), encrypt_result->iv.begin(), encrypt_result->iv.end());
    wrapped.insert(wrapped.end(), encrypt_result->tag.begin(), encrypt_result->tag.end());
    wrapped.insert(wrapped.end(), encrypt_result->ciphertext.begin(), 
                   encrypt_result->ciphertext.end());
    return Ok(std::move(wrapped));
}

Result<std::vector<uint8_t>> FileEncryptionService::unwrapDEK(
    std::span<const uint8_t> wrapped, const KeyId& kek_id) {
    
    if (wrapped.size() < AESEngine::GCM_IV_SIZE + AESEngine::GCM_TAG_SIZE) {
        return Err<std::vector<uint8_t>>(ErrorCode::INVALID_INPUT, "Wrapped DEK too small");
    }
    
    auto kek_result = key_service_->getKey(kek_id);
    if (!kek_result) {
        return Err<std::vector<uint8_t>>(kek_result.error());
    }
    
    std::span<const uint8_t> iv(wrapped.data(), AESEngine::GCM_IV_SIZE);
    std::span<const uint8_t> tag(wrapped.data() + AESEngine::GCM_IV_SIZE, 
                                  AESEngine::GCM_TAG_SIZE);
    std::span<const uint8_t> ciphertext(
        wrapped.data() + AESEngine::GCM_IV_SIZE + AESEngine::GCM_TAG_SIZE,
        wrapped.size() - AESEngine::GCM_IV_SIZE - AESEngine::GCM_TAG_SIZE);
    
    return aes_engine_.decryptGCM(ciphertext, kek_result->key_material, iv, tag);
}

Result<void> FileEncryptionService::encryptFile(
    const std::string& input_path,
    const std::string& output_path,
    const KeyId& kek_id,
    const FileEncryptionContext& ctx,
    ProgressCallback progress) {
    
    std::ifstream input(input_path, std::ios::binary);
    if (!input) {
        logOperation("encrypt_file", kek_id, ctx, false, "FILE_NOT_FOUND");
        return Err<void>(ErrorCode::FILE_NOT_FOUND, "Cannot open input file");
    }
    
    std::ofstream output(output_path, std::ios::binary);
    if (!output) {
        logOperation("encrypt_file", kek_id, ctx, false, "FILE_WRITE_ERROR");
        return Err<void>(ErrorCode::FILE_WRITE_ERROR, "Cannot create output file");
    }
    
    auto file_size = std::filesystem::file_size(input_path);
    return encryptStream(input, output, kek_id, ctx, file_size, progress);
}

Result<void> FileEncryptionService::decryptFile(
    const std::string& input_path,
    const std::string& output_path,
    const FileEncryptionContext& ctx,
    ProgressCallback progress) {
    
    std::ifstream input(input_path, std::ios::binary);
    if (!input) {
        return Err<void>(ErrorCode::FILE_NOT_FOUND, "Cannot open input file");
    }
    
    std::ofstream output(output_path, std::ios::binary);
    if (!output) {
        return Err<void>(ErrorCode::FILE_WRITE_ERROR, "Cannot create output file");
    }
    
    return decryptStream(input, output, ctx, progress);
}

Result<void> FileEncryptionService::encryptStream(
    std::istream& input,
    std::ostream& output,
    const KeyId& kek_id,
    const FileEncryptionContext& ctx,
    uint64_t input_size,
    ProgressCallback progress) {
    
    // Generate unique DEK for this file
    auto dek_result = generateDEK();
    if (!dek_result) {
        logOperation("encrypt_stream", kek_id, ctx, false, "DEK_GENERATION_FAILED");
        return Err<void>(dek_result.error());
    }
    
    // Wrap DEK with KEK
    auto wrapped_result = wrapDEK(*dek_result, kek_id);
    if (!wrapped_result) {
        logOperation("encrypt_stream", kek_id, ctx, false, "DEK_WRAP_FAILED");
        return Err<void>(wrapped_result.error());
    }
    
    // Read all input data (for streaming, we'd process in chunks)
    std::vector<uint8_t> plaintext;
    if (input_size > 0) {
        plaintext.resize(input_size);
        input.read(reinterpret_cast<char*>(plaintext.data()), input_size);
    } else {
        plaintext = std::vector<uint8_t>(
            std::istreambuf_iterator<char>(input),
            std::istreambuf_iterator<char>());
    }
    
    // Encrypt data
    auto encrypt_result = aes_engine_.encryptGCM(plaintext, *dek_result);
    if (!encrypt_result) {
        logOperation("encrypt_stream", kek_id, ctx, false, "ENCRYPTION_FAILED");
        return Err<void>(encrypt_result.error());
    }
    
    // Create header
    FileEncryptionHeader header;
    header.magic = FileEncryptionHeader::MAGIC;
    header.version = FileEncryptionHeader::VERSION;
    header.algorithm = 1;  // AES-256-GCM
    header.key_id = kek_id;
    header.wrapped_dek = std::move(*wrapped_result);
    header.iv = std::move(encrypt_result->iv);
    header.tag = std::move(encrypt_result->tag);
    header.original_size = plaintext.size();
    header.chunk_size = static_cast<uint32_t>(chunk_size_);
    
    // Write header
    auto header_data = header.serialize();
    uint32_t header_size = static_cast<uint32_t>(header_data.size());
    output.write(reinterpret_cast<const char*>(&header_size), sizeof(header_size));
    output.write(reinterpret_cast<const char*>(header_data.data()), header_data.size());
    
    // Write ciphertext
    output.write(reinterpret_cast<const char*>(encrypt_result->ciphertext.data()),
                 encrypt_result->ciphertext.size());
    
    if (progress) {
        progress(plaintext.size(), plaintext.size());
    }
    
    logOperation("encrypt_stream", kek_id, ctx, true);
    return Ok();
}

Result<void> FileEncryptionService::decryptStream(
    std::istream& input,
    std::ostream& output,
    const FileEncryptionContext& ctx,
    ProgressCallback progress) {
    
    // Read header size
    uint32_t header_size;
    input.read(reinterpret_cast<char*>(&header_size), sizeof(header_size));
    if (!input) {
        return Err<void>(ErrorCode::INVALID_INPUT, "Cannot read header size");
    }
    
    // Read header
    std::vector<uint8_t> header_data(header_size);
    input.read(reinterpret_cast<char*>(header_data.data()), header_size);
    if (!input) {
        return Err<void>(ErrorCode::INVALID_INPUT, "Cannot read header");
    }
    
    auto header_result = FileEncryptionHeader::deserialize(header_data);
    if (!header_result) {
        return Err<void>(header_result.error());
    }
    
    const auto& header = *header_result;
    
    // Unwrap DEK
    auto dek_result = unwrapDEK(header.wrapped_dek, header.key_id);
    if (!dek_result) {
        logOperation("decrypt_stream", header.key_id, ctx, false, "DEK_UNWRAP_FAILED");
        return Err<void>(dek_result.error());
    }
    
    // Read ciphertext
    std::vector<uint8_t> ciphertext(
        std::istreambuf_iterator<char>(input),
        std::istreambuf_iterator<char>());
    
    // Decrypt
    auto decrypt_result = aes_engine_.decryptGCM(
        ciphertext, *dek_result, header.iv, header.tag);
    if (!decrypt_result) {
        logOperation("decrypt_stream", header.key_id, ctx, false, "DECRYPTION_FAILED");
        return Err<void>(decrypt_result.error());
    }
    
    // Write plaintext
    output.write(reinterpret_cast<const char*>(decrypt_result->data()),
                 decrypt_result->size());
    
    if (progress) {
        progress(decrypt_result->size(), decrypt_result->size());
    }
    
    logOperation("decrypt_stream", header.key_id, ctx, true);
    return Ok();
}

Result<FileEncryptionHeader> FileEncryptionService::readHeader(const std::string& file_path) {
    std::ifstream input(file_path, std::ios::binary);
    if (!input) {
        return Err<FileEncryptionHeader>(ErrorCode::FILE_NOT_FOUND, "Cannot open file");
    }
    
    uint32_t header_size;
    input.read(reinterpret_cast<char*>(&header_size), sizeof(header_size));
    if (!input) {
        return Err<FileEncryptionHeader>(ErrorCode::INVALID_INPUT, "Cannot read header size");
    }
    
    std::vector<uint8_t> header_data(header_size);
    input.read(reinterpret_cast<char*>(header_data.data()), header_size);
    if (!input) {
        return Err<FileEncryptionHeader>(ErrorCode::INVALID_INPUT, "Cannot read header");
    }
    
    return FileEncryptionHeader::deserialize(header_data);
}

} // namespace crypto
