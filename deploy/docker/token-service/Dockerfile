# Token Service - Rust Multi-Stage Build
# Cryptographic operations with KMS integration

# ============================================
# Stage 1: Build Environment
# ============================================
FROM rust:1.83-alpine AS builder

RUN apk add --no-cache \
    musl-dev \
    openssl-dev \
    openssl-libs-static \
    pkgconfig \
    protobuf-dev \
    upx

WORKDIR /build

COPY auth/token-service/Cargo.toml auth/token-service/Cargo.lock ./
COPY proto/ ../proto/

RUN mkdir src && echo "fn main() {}" > src/main.rs
RUN cargo build --release --locked && rm -rf src

COPY auth/token-service/src ./src

RUN touch src/main.rs && \
    RUSTFLAGS="-C target-feature=+crt-static" \
    cargo build --release --locked --target=x86_64-unknown-linux-musl && \
    strip /build/target/x86_64-unknown-linux-musl/release/token-service && \
    upx --best --lzma /build/target/x86_64-unknown-linux-musl/release/token-service

# ============================================
# Stage 2: Runtime Environment
# ============================================
FROM gcr.io/distroless/static-debian12:nonroot

LABEL org.opencontainers.image.title="Token Service"
LABEL org.opencontainers.image.description="JWT signing and refresh token management with KMS"

COPY --from=builder --chown=nonroot:nonroot \
    /build/target/x86_64-unknown-linux-musl/release/token-service /app/token-service
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

WORKDIR /app
EXPOSE 8081 9091

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/app/token-service", "health"]

USER nonroot:nonroot
ENTRYPOINT ["/app/token-service"]
