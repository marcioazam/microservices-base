# Build stage
FROM golang:1.22-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /app

# Copy go mod files
COPY platform/resilience-service/go.mod platform/resilience-service/go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY platform/resilience-service/ ./

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /resilience-service \
    ./cmd/server

# Final stage
FROM scratch

# Copy CA certificates for HTTPS
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy binary
COPY --from=builder /resilience-service /resilience-service

# Create non-root user
USER 1000:1000

# Expose gRPC port
EXPOSE 50056

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD ["/resilience-service", "-health-check"] || exit 1

# Run the service
ENTRYPOINT ["/resilience-service"]
