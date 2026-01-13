# Auth Edge Service - Rust Multi-Stage Build
# Optimized for production with minimal attack surface
# Build context: repository root (../../)

# ============================================
# Stage 1: Build Environment
# ============================================
FROM rust:1.83-alpine AS builder

# Install build dependencies
RUN apk add --no-cache \
    musl-dev \
    openssl-dev \
    openssl-libs-static \
    pkgconfig \
    protobuf-dev \
    upx

WORKDIR /build

# Copy manifests first for dependency caching
COPY services/auth-edge/Cargo.toml services/auth-edge/Cargo.lock ./
COPY api/proto/ ../proto/

# Create dummy main.rs for dependency compilation
RUN mkdir src && echo "fn main() {}" > src/main.rs

# Build dependencies only (cached layer)
RUN cargo build --release --locked && rm -rf src

# Copy actual source code
COPY services/auth-edge/src ./src

# Build the application
RUN touch src/main.rs && \
    RUSTFLAGS="-C target-feature=+crt-static" \
    cargo build --release --locked --target=x86_64-unknown-linux-musl && \
    strip /build/target/x86_64-unknown-linux-musl/release/auth-edge-service && \
    upx --best --lzma /build/target/x86_64-unknown-linux-musl/release/auth-edge-service

# ============================================
# Stage 2: Runtime Environment (Distroless)
# ============================================
FROM gcr.io/distroless/static-debian12:nonroot

LABEL org.opencontainers.image.title="Auth Edge Service"
LABEL org.opencontainers.image.description="Ultra-low latency JWT validation and mTLS gateway"
LABEL org.opencontainers.image.vendor="Auth Platform"

# Copy binary from builder
COPY --from=builder --chown=nonroot:nonroot \
    /build/target/x86_64-unknown-linux-musl/release/auth-edge-service /app/auth-edge-service

# Copy CA certificates for TLS
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

WORKDIR /app

# Expose gRPC and metrics ports
EXPOSE 8080 9090

# Health check endpoint
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/app/auth-edge-service", "health"]

# Run as non-root user
USER nonroot:nonroot

ENTRYPOINT ["/app/auth-edge-service"]
