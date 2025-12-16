# IAM/Policy Service - Go Multi-Stage Build
# OPA integration for RBAC/ABAC policy evaluation

# ============================================
# Stage 1: Build Environment
# ============================================
FROM golang:1.23-alpine AS builder

RUN apk add --no-cache \
    git \
    ca-certificates \
    tzdata \
    upx \
    protobuf-dev

WORKDIR /build

# Copy go mod files for dependency caching
COPY auth/iam-policy-service/go.mod auth/iam-policy-service/go.sum ./
COPY proto/ ../proto/

# Download dependencies
RUN go mod download && go mod verify

# Copy source code
COPY auth/iam-policy-service/ ./

# Build with optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -extldflags '-static'" \
    -trimpath \
    -o /app/iam-policy-service \
    ./cmd/server && \
    upx --best --lzma /app/iam-policy-service

# ============================================
# Stage 2: Runtime Environment
# ============================================
FROM gcr.io/distroless/static-debian12:nonroot

LABEL org.opencontainers.image.title="IAM Policy Service"
LABEL org.opencontainers.image.description="RBAC/ABAC policy evaluation with OPA"

# Copy binary and policies
COPY --from=builder --chown=nonroot:nonroot /app/iam-policy-service /app/iam-policy-service
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy OPA policies
COPY auth/iam-policy-service/policies /app/policies

WORKDIR /app
EXPOSE 8083 9093

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/app/iam-policy-service", "health"]

USER nonroot:nonroot
ENTRYPOINT ["/app/iam-policy-service"]
