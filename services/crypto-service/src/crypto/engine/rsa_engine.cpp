/**
 * @file rsa_engine.cpp
 * @brief RSA encryption engine implementation using centralized utilities
 * 
 * Requirements: 4.3, 4.4, 4.5, 5.2
 */

#include "crypto/engine/rsa_engine.h"
#include "crypto/common/openssl_raii.h"
#include "crypto/common/hash_utils.h"
#include <openssl/evp.h>
#include <openssl/rsa.h>
#include <openssl/pem.h>

namespace crypto {

// RSAKeyPair implementation
RSAKeyPair::RSAKeyPair() : key_(nullptr) {}

RSAKeyPair::RSAKeyPair(EVP_PKEY* key) : key_(key) {}

RSAKeyPair::~RSAKeyPair() {
    if (key_) {
        EVP_PKEY_free(key_);
    }
}

RSAKeyPair::RSAKeyPair(RSAKeyPair&& other) noexcept : key_(other.key_) {
    other.key_ = nullptr;
}

RSAKeyPair& RSAKeyPair::operator=(RSAKeyPair&& other) noexcept {
    if (this != &other) {
        if (key_) EVP_PKEY_free(key_);
        key_ = other.key_;
        other.key_ = nullptr;
    }
    return *this;
}

[[nodiscard]] size_t RSAKeyPair::keySize() const noexcept {
    if (!key_) return 0;
    return static_cast<size_t>(EVP_PKEY_bits(key_));
}

[[nodiscard]] size_t RSAKeyPair::maxPlaintextSize(HashAlgorithm hash_algo) const noexcept {
    if (!key_) return 0;
    return get_rsa_oaep_max_plaintext(keySize(), hash_algo);
}

[[nodiscard]] Result<std::vector<uint8_t>> RSAKeyPair::exportPublicKeyDER() const {
    if (!key_) {
        return Err<std::vector<uint8_t>>(ErrorCode::INVALID_INPUT, "No key to export");
    }

    int len = i2d_PUBKEY(key_, nullptr);
    if (len <= 0) {
        return Err<std::vector<uint8_t>>(ErrorCode::CRYPTO_ERROR, "Failed to get DER length");
    }

    std::vector<uint8_t> der(static_cast<size_t>(len));
    unsigned char* ptr = der.data();
    if (i2d_PUBKEY(key_, &ptr) <= 0) {
        return Err<std::vector<uint8_t>>(ErrorCode::CRYPTO_ERROR, "Failed to export public key");
    }

    return Ok(std::move(der));
}

[[nodiscard]] Result<std::vector<uint8_t>> RSAKeyPair::exportPrivateKeyDER() const {
    if (!key_) {
        return Err<std::vector<uint8_t>>(ErrorCode::INVALID_INPUT, "No key to export");
    }

    int len = i2d_PrivateKey(key_, nullptr);
    if (len <= 0) {
        return Err<std::vector<uint8_t>>(ErrorCode::CRYPTO_ERROR, "Failed to get DER length");
    }

    std::vector<uint8_t> der(static_cast<size_t>(len));
    unsigned char* ptr = der.data();
    if (i2d_PrivateKey(key_, &ptr) <= 0) {
        return Err<std::vector<uint8_t>>(ErrorCode::CRYPTO_ERROR, "Failed to export private key");
    }

    return Ok(std::move(der));
}

[[nodiscard]] Result<std::string> RSAKeyPair::exportPublicKeyPEM() const {
    if (!key_) {
        return Err<std::string>(ErrorCode::INVALID_INPUT, "No key to export");
    }

    auto bio = openssl::make_bio_mem();
    if (!bio) {
        return Err<std::string>(ErrorCode::CRYPTO_ERROR, "Failed to create BIO");
    }

    if (PEM_write_bio_PUBKEY(bio.get(), key_) != 1) {
        return Err<std::string>(ErrorCode::CRYPTO_ERROR, "Failed to write PEM");
    }

    char* data = nullptr;
    long len = BIO_get_mem_data(bio.get(), &data);
    return Ok(std::string(data, static_cast<size_t>(len)));
}

[[nodiscard]] Result<std::string> RSAKeyPair::exportPrivateKeyPEM() const {
    if (!key_) {
        return Err<std::string>(ErrorCode::INVALID_INPUT, "No key to export");
    }

    auto bio = openssl::make_bio_mem();
    if (!bio) {
        return Err<std::string>(ErrorCode::CRYPTO_ERROR, "Failed to create BIO");
    }

    if (PEM_write_bio_PrivateKey(bio.get(), key_, nullptr, nullptr, 0, nullptr, nullptr) != 1) {
        return Err<std::string>(ErrorCode::CRYPTO_ERROR, "Failed to write PEM");
    }

    char* data = nullptr;
    long len = BIO_get_mem_data(bio.get(), &data);
    return Ok(std::string(data, static_cast<size_t>(len)));
}

[[nodiscard]] Result<RSAKeyPair> RSAKeyPair::importPublicKeyDER(std::span<const uint8_t> der) {
    const unsigned char* ptr = der.data();
    EVP_PKEY* key = d2i_PUBKEY(nullptr, &ptr, static_cast<long>(der.size()));
    if (!key) {
        return Err<RSAKeyPair>(ErrorCode::CRYPTO_ERROR, "Failed to import public key");
    }
    return Ok(RSAKeyPair(key));
}

[[nodiscard]] Result<RSAKeyPair> RSAKeyPair::importPrivateKeyDER(std::span<const uint8_t> der) {
    const unsigned char* ptr = der.data();
    EVP_PKEY* key = d2i_PrivateKey(EVP_PKEY_RSA, nullptr, &ptr, static_cast<long>(der.size()));
    if (!key) {
        return Err<RSAKeyPair>(ErrorCode::CRYPTO_ERROR, "Failed to import private key");
    }
    return Ok(RSAKeyPair(key));
}

[[nodiscard]] Result<RSAKeyPair> RSAKeyPair::importPublicKeyPEM(std::string_view pem) {
    auto bio = openssl::make_bio_mem_buf(pem.data(), static_cast<int>(pem.size()));
    if (!bio) {
        return Err<RSAKeyPair>(ErrorCode::CRYPTO_ERROR, "Failed to create BIO");
    }

    EVP_PKEY* key = PEM_read_bio_PUBKEY(bio.get(), nullptr, nullptr, nullptr);
    if (!key) {
        return Err<RSAKeyPair>(ErrorCode::CRYPTO_ERROR, "Failed to import public key");
    }
    return Ok(RSAKeyPair(key));
}

[[nodiscard]] Result<RSAKeyPair> RSAKeyPair::importPrivateKeyPEM(std::string_view pem) {
    auto bio = openssl::make_bio_mem_buf(pem.data(), static_cast<int>(pem.size()));
    if (!bio) {
        return Err<RSAKeyPair>(ErrorCode::CRYPTO_ERROR, "Failed to create BIO");
    }

    EVP_PKEY* key = PEM_read_bio_PrivateKey(bio.get(), nullptr, nullptr, nullptr);
    if (!key) {
        return Err<RSAKeyPair>(ErrorCode::CRYPTO_ERROR, "Failed to import private key");
    }
    return Ok(RSAKeyPair(key));
}

// RSAEngine implementation
[[nodiscard]] Result<RSAKeyPair> RSAEngine::generateKeyPair(RSAKeySize key_size) {
    auto ctx = openssl::make_pkey_ctx(EVP_PKEY_RSA);
    if (!ctx) {
        return Err<RSAKeyPair>(ErrorCode::KEY_GENERATION_FAILED, "Failed to create context");
    }

    if (EVP_PKEY_keygen_init(ctx.get()) <= 0) {
        return Err<RSAKeyPair>(ErrorCode::KEY_GENERATION_FAILED, "Failed to init keygen");
    }

    if (EVP_PKEY_CTX_set_rsa_keygen_bits(ctx.get(), static_cast<int>(key_size)) <= 0) {
        return Err<RSAKeyPair>(ErrorCode::KEY_GENERATION_FAILED, "Failed to set key size");
    }

    EVP_PKEY* key = nullptr;
    if (EVP_PKEY_keygen(ctx.get(), &key) <= 0) {
        return Err<RSAKeyPair>(ErrorCode::KEY_GENERATION_FAILED, "Failed to generate key");
    }

    return Ok(RSAKeyPair(key));
}

[[nodiscard]] Result<std::vector<uint8_t>> RSAEngine::encryptOAEP(
    std::span<const uint8_t> plaintext,
    const RSAKeyPair& public_key,
    HashAlgorithm hash_algo) {
    
    if (!public_key.isValid()) {
        return Err<std::vector<uint8_t>>(ErrorCode::INVALID_INPUT, "Invalid public key");
    }

    if (plaintext.size() > public_key.maxPlaintextSize(hash_algo)) {
        return Err<std::vector<uint8_t>>(ErrorCode::SIZE_LIMIT_EXCEEDED,
            "Plaintext exceeds maximum size for key");
    }

    auto ctx = openssl::make_pkey_ctx(public_key.get());
    if (!ctx) {
        return Err<std::vector<uint8_t>>(ErrorCode::CRYPTO_ERROR, "Failed to create context");
    }

    if (EVP_PKEY_encrypt_init(ctx.get()) <= 0) {
        return Err<std::vector<uint8_t>>(ErrorCode::CRYPTO_ERROR, "Failed to init encryption");
    }

    if (EVP_PKEY_CTX_set_rsa_padding(ctx.get(), RSA_PKCS1_OAEP_PADDING) <= 0) {
        return Err<std::vector<uint8_t>>(ErrorCode::CRYPTO_ERROR, "Failed to set padding");
    }

    const EVP_MD* md = get_evp_md(hash_algo);
    if (EVP_PKEY_CTX_set_rsa_oaep_md(ctx.get(), md) <= 0) {
        return Err<std::vector<uint8_t>>(ErrorCode::CRYPTO_ERROR, "Failed to set OAEP hash");
    }

    if (EVP_PKEY_CTX_set_rsa_mgf1_md(ctx.get(), md) <= 0) {
        return Err<std::vector<uint8_t>>(ErrorCode::CRYPTO_ERROR, "Failed to set MGF1 hash");
    }

    size_t outlen = 0;
    if (EVP_PKEY_encrypt(ctx.get(), nullptr, &outlen, plaintext.data(), plaintext.size()) <= 0) {
        return Err<std::vector<uint8_t>>(ErrorCode::CRYPTO_ERROR, "Failed to get output size");
    }

    std::vector<uint8_t> ciphertext(outlen);
    if (EVP_PKEY_encrypt(ctx.get(), ciphertext.data(), &outlen, 
                        plaintext.data(), plaintext.size()) <= 0) {
        return Err<std::vector<uint8_t>>(ErrorCode::ENCRYPTION_FAILED, "Encryption failed");
    }

    ciphertext.resize(outlen);
    return Ok(std::move(ciphertext));
}

[[nodiscard]] Result<std::vector<uint8_t>> RSAEngine::decryptOAEP(
    std::span<const uint8_t> ciphertext,
    const RSAKeyPair& private_key,
    HashAlgorithm hash_algo) {
    
    if (!private_key.isValid()) {
        return Err<std::vector<uint8_t>>(ErrorCode::INVALID_INPUT, "Invalid private key");
    }

    auto ctx = openssl::make_pkey_ctx(private_key.get());
    if (!ctx) {
        return Err<std::vector<uint8_t>>(ErrorCode::CRYPTO_ERROR, "Failed to create context");
    }

    if (EVP_PKEY_decrypt_init(ctx.get()) <= 0) {
        return Err<std::vector<uint8_t>>(ErrorCode::CRYPTO_ERROR, "Failed to init decryption");
    }

    if (EVP_PKEY_CTX_set_rsa_padding(ctx.get(), RSA_PKCS1_OAEP_PADDING) <= 0) {
        return Err<std::vector<uint8_t>>(ErrorCode::CRYPTO_ERROR, "Failed to set padding");
    }

    const EVP_MD* md = get_evp_md(hash_algo);
    if (EVP_PKEY_CTX_set_rsa_oaep_md(ctx.get(), md) <= 0) {
        return Err<std::vector<uint8_t>>(ErrorCode::CRYPTO_ERROR, "Failed to set OAEP hash");
    }

    if (EVP_PKEY_CTX_set_rsa_mgf1_md(ctx.get(), md) <= 0) {
        return Err<std::vector<uint8_t>>(ErrorCode::CRYPTO_ERROR, "Failed to set MGF1 hash");
    }

    size_t outlen = 0;
    if (EVP_PKEY_decrypt(ctx.get(), nullptr, &outlen, ciphertext.data(), ciphertext.size()) <= 0) {
        return Err<std::vector<uint8_t>>(ErrorCode::CRYPTO_ERROR, "Failed to get output size");
    }

    std::vector<uint8_t> plaintext(outlen);
    if (EVP_PKEY_decrypt(ctx.get(), plaintext.data(), &outlen,
                        ciphertext.data(), ciphertext.size()) <= 0) {
        return Err<std::vector<uint8_t>>(ErrorCode::DECRYPTION_FAILED, "Decryption failed");
    }

    plaintext.resize(outlen);
    return Ok(std::move(plaintext));
}

[[nodiscard]] Result<std::vector<uint8_t>> RSAEngine::signPSS(
    std::span<const uint8_t> data,
    const RSAKeyPair& private_key,
    HashAlgorithm hash_algo) {
    
    if (!private_key.isValid()) {
        return Err<std::vector<uint8_t>>(ErrorCode::INVALID_INPUT, "Invalid private key");
    }

    const EVP_MD* md = get_evp_md(hash_algo);

    auto md_ctx = openssl::make_md_ctx();
    if (!md_ctx) {
        return Err<std::vector<uint8_t>>(ErrorCode::CRYPTO_ERROR, "Failed to create MD context");
    }

    EVP_PKEY_CTX* pkey_ctx = nullptr;
    if (EVP_DigestSignInit(md_ctx.get(), &pkey_ctx, md, nullptr, private_key.get()) <= 0) {
        return Err<std::vector<uint8_t>>(ErrorCode::CRYPTO_ERROR, "Failed to init signing");
    }

    if (EVP_PKEY_CTX_set_rsa_padding(pkey_ctx, RSA_PKCS1_PSS_PADDING) <= 0) {
        return Err<std::vector<uint8_t>>(ErrorCode::CRYPTO_ERROR, "Failed to set PSS padding");
    }

    if (EVP_PKEY_CTX_set_rsa_pss_saltlen(pkey_ctx, RSA_PSS_SALTLEN_DIGEST) <= 0) {
        return Err<std::vector<uint8_t>>(ErrorCode::CRYPTO_ERROR, "Failed to set salt length");
    }

    if (EVP_DigestSignUpdate(md_ctx.get(), data.data(), data.size()) <= 0) {
        return Err<std::vector<uint8_t>>(ErrorCode::CRYPTO_ERROR, "Failed to update digest");
    }

    size_t sig_len = 0;
    if (EVP_DigestSignFinal(md_ctx.get(), nullptr, &sig_len) <= 0) {
        return Err<std::vector<uint8_t>>(ErrorCode::CRYPTO_ERROR, "Failed to get signature size");
    }

    std::vector<uint8_t> signature(sig_len);
    if (EVP_DigestSignFinal(md_ctx.get(), signature.data(), &sig_len) <= 0) {
        return Err<std::vector<uint8_t>>(ErrorCode::CRYPTO_ERROR, "Signing failed");
    }

    signature.resize(sig_len);
    return Ok(std::move(signature));
}

[[nodiscard]] Result<bool> RSAEngine::verifyPSS(
    std::span<const uint8_t> data,
    std::span<const uint8_t> signature,
    const RSAKeyPair& public_key,
    HashAlgorithm hash_algo) {
    
    if (!public_key.isValid()) {
        return Err<bool>(ErrorCode::INVALID_INPUT, "Invalid public key");
    }

    const EVP_MD* md = get_evp_md(hash_algo);

    auto md_ctx = openssl::make_md_ctx();
    if (!md_ctx) {
        return Err<bool>(ErrorCode::CRYPTO_ERROR, "Failed to create MD context");
    }

    EVP_PKEY_CTX* pkey_ctx = nullptr;
    if (EVP_DigestVerifyInit(md_ctx.get(), &pkey_ctx, md, nullptr, public_key.get()) <= 0) {
        return Err<bool>(ErrorCode::CRYPTO_ERROR, "Failed to init verification");
    }

    if (EVP_PKEY_CTX_set_rsa_padding(pkey_ctx, RSA_PKCS1_PSS_PADDING) <= 0) {
        return Err<bool>(ErrorCode::CRYPTO_ERROR, "Failed to set PSS padding");
    }

    if (EVP_PKEY_CTX_set_rsa_pss_saltlen(pkey_ctx, RSA_PSS_SALTLEN_DIGEST) <= 0) {
        return Err<bool>(ErrorCode::CRYPTO_ERROR, "Failed to set salt length");
    }

    if (EVP_DigestVerifyUpdate(md_ctx.get(), data.data(), data.size()) <= 0) {
        return Err<bool>(ErrorCode::CRYPTO_ERROR, "Failed to update digest");
    }

    int ret = EVP_DigestVerifyFinal(md_ctx.get(), signature.data(), signature.size());
    if (ret == 1) {
        return Ok(true);
    } else if (ret == 0) {
        return Ok(false);
    } else {
        return Err<bool>(ErrorCode::CRYPTO_ERROR, "Verification error");
    }
}

} // namespace crypto
