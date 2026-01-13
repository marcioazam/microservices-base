#pragma once

/**
 * @file openssl_raii.h
 * @brief Centralized RAII wrappers for OpenSSL resources
 * 
 * This header provides type-safe, exception-safe wrappers for all OpenSSL
 * resources used in the crypto-service. All wrappers use unique_ptr with
 * custom deleters for automatic resource cleanup.
 * 
 * Requirements: 4.3, 4.4, 6.1
 */

#include <memory>
#include <openssl/evp.h>
#include <openssl/bio.h>
#include <openssl/err.h>
#include <openssl/rand.h>
#include <openssl/core_names.h>
#include <openssl/param_build.h>

namespace crypto::openssl {

// ============================================================================
// EVP_CIPHER_CTX - Symmetric cipher context
// ============================================================================

struct CipherCtxDeleter {
    void operator()(EVP_CIPHER_CTX* ctx) const noexcept {
        if (ctx) EVP_CIPHER_CTX_free(ctx);
    }
};

using CipherCtx = std::unique_ptr<EVP_CIPHER_CTX, CipherCtxDeleter>;

[[nodiscard]] inline CipherCtx make_cipher_ctx() {
    return CipherCtx(EVP_CIPHER_CTX_new());
}

// ============================================================================
// EVP_PKEY - Asymmetric key
// ============================================================================

struct PKeyDeleter {
    void operator()(EVP_PKEY* key) const noexcept {
        if (key) EVP_PKEY_free(key);
    }
};

using PKey = std::unique_ptr<EVP_PKEY, PKeyDeleter>;

// ============================================================================
// EVP_PKEY_CTX - Asymmetric key context
// ============================================================================

struct PKeyCtxDeleter {
    void operator()(EVP_PKEY_CTX* ctx) const noexcept {
        if (ctx) EVP_PKEY_CTX_free(ctx);
    }
};

using PKeyCtx = std::unique_ptr<EVP_PKEY_CTX, PKeyCtxDeleter>;

[[nodiscard]] inline PKeyCtx make_pkey_ctx(int id) {
    return PKeyCtx(EVP_PKEY_CTX_new_id(id, nullptr));
}

[[nodiscard]] inline PKeyCtx make_pkey_ctx(EVP_PKEY* pkey) {
    return PKeyCtx(EVP_PKEY_CTX_new(pkey, nullptr));
}

// ============================================================================
// EVP_MD_CTX - Message digest context
// ============================================================================

struct MDCtxDeleter {
    void operator()(EVP_MD_CTX* ctx) const noexcept {
        if (ctx) EVP_MD_CTX_free(ctx);
    }
};

using MDCtx = std::unique_ptr<EVP_MD_CTX, MDCtxDeleter>;

[[nodiscard]] inline MDCtx make_md_ctx() {
    return MDCtx(EVP_MD_CTX_new());
}

// ============================================================================
// BIO - Basic I/O abstraction
// ============================================================================

struct BIODeleter {
    void operator()(BIO* bio) const noexcept {
        if (bio) BIO_free(bio);
    }
};

using BIOPtr = std::unique_ptr<BIO, BIODeleter>;

[[nodiscard]] inline BIOPtr make_bio_mem() {
    return BIOPtr(BIO_new(BIO_s_mem()));
}

[[nodiscard]] inline BIOPtr make_bio_mem_buf(const void* data, int len) {
    return BIOPtr(BIO_new_mem_buf(data, len));
}

// ============================================================================
// OSSL_PARAM_BLD - Parameter builder (OpenSSL 3.x)
// ============================================================================

struct ParamBldDeleter {
    void operator()(OSSL_PARAM_BLD* bld) const noexcept {
        if (bld) OSSL_PARAM_BLD_free(bld);
    }
};

using ParamBld = std::unique_ptr<OSSL_PARAM_BLD, ParamBldDeleter>;

[[nodiscard]] inline ParamBld make_param_bld() {
    return ParamBld(OSSL_PARAM_BLD_new());
}

// ============================================================================
// OSSL_PARAM - Parameters (OpenSSL 3.x)
// ============================================================================

struct ParamDeleter {
    void operator()(OSSL_PARAM* params) const noexcept {
        if (params) OSSL_PARAM_free(params);
    }
};

using Params = std::unique_ptr<OSSL_PARAM, ParamDeleter>;

// ============================================================================
// EVP_MAC - Message Authentication Code (OpenSSL 3.x)
// ============================================================================

struct MACDeleter {
    void operator()(EVP_MAC* mac) const noexcept {
        if (mac) EVP_MAC_free(mac);
    }
};

using MAC = std::unique_ptr<EVP_MAC, MACDeleter>;

[[nodiscard]] inline MAC make_mac(const char* algorithm) {
    return MAC(EVP_MAC_fetch(nullptr, algorithm, nullptr));
}

// ============================================================================
// EVP_MAC_CTX - MAC context (OpenSSL 3.x)
// ============================================================================

struct MACCtxDeleter {
    void operator()(EVP_MAC_CTX* ctx) const noexcept {
        if (ctx) EVP_MAC_CTX_free(ctx);
    }
};

using MACCtx = std::unique_ptr<EVP_MAC_CTX, MACCtxDeleter>;

[[nodiscard]] inline MACCtx make_mac_ctx(EVP_MAC* mac) {
    return MACCtx(EVP_MAC_CTX_new(mac));
}

// ============================================================================
// EVP_KDF - Key Derivation Function (OpenSSL 3.x)
// ============================================================================

struct KDFDeleter {
    void operator()(EVP_KDF* kdf) const noexcept {
        if (kdf) EVP_KDF_free(kdf);
    }
};

using KDF = std::unique_ptr<EVP_KDF, KDFDeleter>;

[[nodiscard]] inline KDF make_kdf(const char* algorithm) {
    return KDF(EVP_KDF_fetch(nullptr, algorithm, nullptr));
}

// ============================================================================
// EVP_KDF_CTX - KDF context (OpenSSL 3.x)
// ============================================================================

struct KDFCtxDeleter {
    void operator()(EVP_KDF_CTX* ctx) const noexcept {
        if (ctx) EVP_KDF_CTX_free(ctx);
    }
};

using KDFCtx = std::unique_ptr<EVP_KDF_CTX, KDFCtxDeleter>;

[[nodiscard]] inline KDFCtx make_kdf_ctx(EVP_KDF* kdf) {
    return KDFCtx(EVP_KDF_CTX_new(kdf));
}

// ============================================================================
// BIGNUM - Arbitrary precision integer
// ============================================================================

struct BNDeleter {
    void operator()(BIGNUM* bn) const noexcept {
        if (bn) BN_free(bn);
    }
};

using BN = std::unique_ptr<BIGNUM, BNDeleter>;

[[nodiscard]] inline BN make_bn() {
    return BN(BN_new());
}

// ============================================================================
// BN_CTX - BIGNUM context for temporary variables
// ============================================================================

struct BNCtxDeleter {
    void operator()(BN_CTX* ctx) const noexcept {
        if (ctx) BN_CTX_free(ctx);
    }
};

using BNCtx = std::unique_ptr<BN_CTX, BNCtxDeleter>;

[[nodiscard]] inline BNCtx make_bn_ctx() {
    return BNCtx(BN_CTX_new());
}

// ============================================================================
// Utility functions
// ============================================================================

/**
 * @brief Get the last OpenSSL error as a string
 * @return Human-readable error message
 */
[[nodiscard]] inline std::string get_last_error() {
    char buf[256];
    ERR_error_string_n(ERR_get_error(), buf, sizeof(buf));
    return std::string(buf);
}

/**
 * @brief Clear all pending OpenSSL errors
 */
inline void clear_errors() noexcept {
    ERR_clear_error();
}

/**
 * @brief Generate cryptographically secure random bytes
 * @param buffer Output buffer
 * @param size Number of bytes to generate
 * @return true on success, false on failure
 */
[[nodiscard]] inline bool random_bytes(void* buffer, size_t size) noexcept {
    return RAND_bytes(static_cast<unsigned char*>(buffer), 
                      static_cast<int>(size)) == 1;
}

} // namespace crypto::openssl
