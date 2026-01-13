#include "crypto/engine/ecdsa_engine.h"
#include "crypto/common/openssl_raii.h"
#include "crypto/common/hash_utils.h"
#include <openssl/evp.h>
#include <openssl/ec.h>
#include <openssl/pem.h>
#include <openssl/err.h>
#include <openssl/obj_mac.h>

namespace crypto {

// ECKeyPair implementation
ECKeyPair::ECKeyPair() : key_(nullptr), curve_(ECCurve::P256) {}

ECKeyPair::ECKeyPair(EVP_PKEY* key, ECCurve curve) : key_(key), curve_(curve) {}

ECKeyPair::~ECKeyPair() {
    if (key_) {
        EVP_PKEY_free(key_);
    }
}

ECKeyPair::ECKeyPair(ECKeyPair&& other) noexcept 
    : key_(other.key_), curve_(other.curve_) {
    other.key_ = nullptr;
}

ECKeyPair& ECKeyPair::operator=(ECKeyPair&& other) noexcept {
    if (this != &other) {
        if (key_) EVP_PKEY_free(key_);
        key_ = other.key_;
        curve_ = other.curve_;
        other.key_ = nullptr;
    }
    return *this;
}

Result<std::vector<uint8_t>> ECKeyPair::exportPublicKeyDER() const {
    if (!key_) {
        return Err<std::vector<uint8_t>>(ErrorCode::INVALID_INPUT, "No key to export");
    }

    int len = i2d_PUBKEY(key_, nullptr);
    if (len <= 0) {
        return Err<std::vector<uint8_t>>(ErrorCode::CRYPTO_ERROR, "Failed to get DER length");
    }

    std::vector<uint8_t> der(len);
    unsigned char* ptr = der.data();
    if (i2d_PUBKEY(key_, &ptr) <= 0) {
        return Err<std::vector<uint8_t>>(ErrorCode::CRYPTO_ERROR, "Failed to export public key");
    }

    return Ok(std::move(der));
}

Result<std::vector<uint8_t>> ECKeyPair::exportPrivateKeyDER() const {
    if (!key_) {
        return Err<std::vector<uint8_t>>(ErrorCode::INVALID_INPUT, "No key to export");
    }

    int len = i2d_PrivateKey(key_, nullptr);
    if (len <= 0) {
        return Err<std::vector<uint8_t>>(ErrorCode::CRYPTO_ERROR, "Failed to get DER length");
    }

    std::vector<uint8_t> der(len);
    unsigned char* ptr = der.data();
    if (i2d_PrivateKey(key_, &ptr) <= 0) {
        return Err<std::vector<uint8_t>>(ErrorCode::CRYPTO_ERROR, "Failed to export private key");
    }

    return Ok(std::move(der));
}

Result<std::string> ECKeyPair::exportPublicKeyPEM() const {
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
    return Ok(std::string(data, len));
}

Result<std::string> ECKeyPair::exportPrivateKeyPEM() const {
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
    return Ok(std::string(data, len));
}

Result<ECKeyPair> ECKeyPair::importPublicKeyDER(std::span<const uint8_t> der, ECCurve curve) {
    const unsigned char* ptr = der.data();
    EVP_PKEY* key = d2i_PUBKEY(nullptr, &ptr, der.size());
    if (!key) {
        return Err<ECKeyPair>(ErrorCode::CRYPTO_ERROR, "Failed to import public key");
    }
    return Ok(ECKeyPair(key, curve));
}

Result<ECKeyPair> ECKeyPair::importPrivateKeyDER(std::span<const uint8_t> der, ECCurve curve) {
    const unsigned char* ptr = der.data();
    EVP_PKEY* key = d2i_PrivateKey(EVP_PKEY_EC, nullptr, &ptr, der.size());
    if (!key) {
        return Err<ECKeyPair>(ErrorCode::CRYPTO_ERROR, "Failed to import private key");
    }
    return Ok(ECKeyPair(key, curve));
}

Result<ECKeyPair> ECKeyPair::importPublicKeyPEM(std::string_view pem, ECCurve curve) {
    auto bio = openssl::make_bio_mem_buf(pem.data(), static_cast<int>(pem.size()));
    if (!bio) {
        return Err<ECKeyPair>(ErrorCode::CRYPTO_ERROR, "Failed to create BIO");
    }

    EVP_PKEY* key = PEM_read_bio_PUBKEY(bio.get(), nullptr, nullptr, nullptr);
    if (!key) {
        return Err<ECKeyPair>(ErrorCode::CRYPTO_ERROR, "Failed to import public key");
    }
    return Ok(ECKeyPair(key, curve));
}

Result<ECKeyPair> ECKeyPair::importPrivateKeyPEM(std::string_view pem, ECCurve curve) {
    auto bio = openssl::make_bio_mem_buf(pem.data(), static_cast<int>(pem.size()));
    if (!bio) {
        return Err<ECKeyPair>(ErrorCode::CRYPTO_ERROR, "Failed to create BIO");
    }

    EVP_PKEY* key = PEM_read_bio_PrivateKey(bio.get(), nullptr, nullptr, nullptr);
    if (!key) {
        return Err<ECKeyPair>(ErrorCode::CRYPTO_ERROR, "Failed to import private key");
    }
    return Ok(ECKeyPair(key, curve));
}

// ECDSAEngine implementation
int ECDSAEngine::curveNID(ECCurve curve) {
    return get_curve_nid(curve);
}

const char* ECDSAEngine::curveName(ECCurve curve) {
    // Return C-string for backward compatibility
    switch (curve) {
        case ECCurve::P256: return "P-256";
        case ECCurve::P384: return "P-384";
        case ECCurve::P521: return "P-521";
        default: return "unknown";
    }
}

Result<ECKeyPair> ECDSAEngine::generateKeyPair(ECCurve curve) {
    int nid = get_curve_nid(curve);
    if (nid == 0) {
        return Err<ECKeyPair>(ErrorCode::INVALID_INPUT, "Invalid curve");
    }

    auto ctx = openssl::make_pkey_ctx(EVP_PKEY_EC);
    if (!ctx) {
        return Err<ECKeyPair>(ErrorCode::KEY_GENERATION_FAILED, "Failed to create context");
    }

    if (EVP_PKEY_keygen_init(ctx.get()) <= 0) {
        return Err<ECKeyPair>(ErrorCode::KEY_GENERATION_FAILED, "Failed to init keygen");
    }

    if (EVP_PKEY_CTX_set_ec_paramgen_curve_nid(ctx.get(), nid) <= 0) {
        return Err<ECKeyPair>(ErrorCode::KEY_GENERATION_FAILED, "Failed to set curve");
    }

    EVP_PKEY* key = nullptr;
    if (EVP_PKEY_keygen(ctx.get(), &key) <= 0) {
        return Err<ECKeyPair>(ErrorCode::KEY_GENERATION_FAILED, "Failed to generate key");
    }

    return Ok(ECKeyPair(key, curve));
}

Result<std::vector<uint8_t>> ECDSAEngine::sign(
    std::span<const uint8_t> data,
    const ECKeyPair& private_key) {
    
    if (!private_key.isValid()) {
        return Err<std::vector<uint8_t>>(ErrorCode::INVALID_INPUT, "Invalid private key");
    }

    const EVP_MD* md = get_evp_md_for_curve(private_key.curve());
    if (!md) {
        return Err<std::vector<uint8_t>>(ErrorCode::INVALID_INPUT, "Invalid curve");
    }

    auto md_ctx = openssl::make_md_ctx();
    if (!md_ctx) {
        return Err<std::vector<uint8_t>>(ErrorCode::CRYPTO_ERROR, "Failed to create MD context");
    }

    if (EVP_DigestSignInit(md_ctx.get(), nullptr, md, nullptr, private_key.get()) <= 0) {
        return Err<std::vector<uint8_t>>(ErrorCode::CRYPTO_ERROR, "Failed to init signing");
    }

    if (EVP_DigestSignUpdate(md_ctx.get(), data.data(), data.size()) <= 0) {
        return Err<std::vector<uint8_t>>(ErrorCode::CRYPTO_ERROR, "Failed to update digest");
    }

    // Get signature size
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

Result<bool> ECDSAEngine::verify(
    std::span<const uint8_t> data,
    std::span<const uint8_t> signature,
    const ECKeyPair& public_key) {
    
    if (!public_key.isValid()) {
        return Err<bool>(ErrorCode::INVALID_INPUT, "Invalid public key");
    }

    const EVP_MD* md = get_evp_md_for_curve(public_key.curve());
    if (!md) {
        return Err<bool>(ErrorCode::INVALID_INPUT, "Invalid curve");
    }

    auto md_ctx = openssl::make_md_ctx();
    if (!md_ctx) {
        return Err<bool>(ErrorCode::CRYPTO_ERROR, "Failed to create MD context");
    }

    if (EVP_DigestVerifyInit(md_ctx.get(), nullptr, md, nullptr, public_key.get()) <= 0) {
        return Err<bool>(ErrorCode::CRYPTO_ERROR, "Failed to init verification");
    }

    if (EVP_DigestVerifyUpdate(md_ctx.get(), data.data(), data.size()) <= 0) {
        return Err<bool>(ErrorCode::CRYPTO_ERROR, "Failed to update digest");
    }

    int ret = EVP_DigestVerifyFinal(md_ctx.get(), signature.data(), signature.size());
    if (ret == 1) {
        return Ok(true);
    } else if (ret == 0) {
        return Ok(false);  // Signature invalid
    } else {
        return Err<bool>(ErrorCode::CRYPTO_ERROR, "Verification error");
    }
}

} // namespace crypto
