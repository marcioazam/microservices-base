# Multi-stage build for Crypto Security Service
# Stage 1: Build environment
FROM ubuntu:22.04 AS builder

ENV DEBIAN_FRONTEND=noninteractive

# Install build dependencies
RUN apt-get update && apt-get install -y \
    build-essential \
    cmake \
    ninja-build \
    git \
    pkg-config \
    libssl-dev \
    libgrpc++-dev \
    libprotobuf-dev \
    protobuf-compiler \
    protobuf-compiler-grpc \
    libcurl4-openssl-dev \
    libjsoncpp-dev \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /build

# Copy source code
COPY CMakeLists.txt ./
COPY include/ ./include/
COPY src/ ./src/
COPY proto/ ./proto/

# Build the service
RUN cmake -B build -G Ninja \
    -DCMAKE_BUILD_TYPE=Release \
    -DCMAKE_CXX_FLAGS="-O3 -DNDEBUG" \
    && cmake --build build --target crypto-service

# Stage 2: Runtime environment
FROM ubuntu:22.04 AS runtime

ENV DEBIAN_FRONTEND=noninteractive

# Install runtime dependencies only
RUN apt-get update && apt-get install -y --no-install-recommends \
    libssl3 \
    libgrpc++1 \
    libprotobuf23 \
    libcurl4 \
    libjsoncpp25 \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/* \
    && apt-get clean

# Create non-root user for security
RUN groupadd -r crypto && useradd -r -g crypto crypto

# Create required directories
RUN mkdir -p /var/lib/crypto-service/keys \
    && mkdir -p /var/log/crypto-service/audit \
    && mkdir -p /etc/crypto-service \
    && chown -R crypto:crypto /var/lib/crypto-service \
    && chown -R crypto:crypto /var/log/crypto-service \
    && chown -R crypto:crypto /etc/crypto-service

# Copy binary from builder
COPY --from=builder /build/build/crypto-service /usr/local/bin/crypto-service

# Set permissions
RUN chmod 755 /usr/local/bin/crypto-service

# Switch to non-root user
USER crypto

# Expose ports
EXPOSE 50051 8080 9090

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Default environment variables
ENV CRYPTO_GRPC_PORT=50051 \
    CRYPTO_REST_PORT=8080 \
    CRYPTO_LOG_LEVEL=INFO \
    CRYPTO_LOG_OUTPUT=stdout \
    CRYPTO_KMS_PROVIDER=local \
    CRYPTO_LOCAL_KEY_PATH=/var/lib/crypto-service/keys \
    CRYPTO_AUDIT_STORAGE_PATH=/var/log/crypto-service/audit \
    CRYPTO_AUDIT_ENCRYPT_LOGS=true \
    CRYPTO_THREAD_POOL_SIZE=4

ENTRYPOINT ["/usr/local/bin/crypto-service"]
