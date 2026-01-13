# Build stage
FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY cmd/ cmd/
COPY api/ api/
COPY internal/ internal/

# Build
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build \
    -ldflags="-s -w" \
    -o manager cmd/main.go

# Runtime stage - distroless for security
FROM gcr.io/distroless/static:nonroot

LABEL org.opencontainers.image.source="https://github.com/auth-platform/platform"
LABEL org.opencontainers.image.description="Resilience Operator for Kubernetes"
LABEL org.opencontainers.image.licenses="Apache-2.0"

WORKDIR /
COPY --from=builder /workspace/manager .

# Run as non-root user (65532 is the nonroot user in distroless)
USER 65532:65532

ENTRYPOINT ["/manager"]
