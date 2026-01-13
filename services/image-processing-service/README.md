# Image Processing Service

A high-performance image processing microservice for the Auth Platform, providing image manipulation capabilities through a RESTful API with async job processing support.

## Overview

This service handles image operations including resizing, format conversion, adjustments (brightness, contrast, saturation), rotation, watermarking, and compression. It's designed for horizontal scalability with separate API and worker processes.

## Architecture

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   API Server    │────▶│   Redis/BullMQ  │◀────│     Worker      │
│  (src/index.ts) │     │     Queue       │     │ (src/worker.ts) │
└────────┬────────┘     └─────────────────┘     └────────┬────────┘
         │                                               │
         ▼                                               ▼
┌─────────────────┐                             ┌─────────────────┐
│   S3 Storage    │◀────────────────────────────│  Image Service  │
│    (MinIO)      │                             │    (Sharp)      │
└─────────────────┘                             └─────────────────┘
```

## Entry Points

| Entry Point | File | Purpose |
|-------------|------|---------|
| API Server | `src/index.ts` | HTTP API for image operations |
| Worker | `src/worker.ts` | Background job processor |

## Quick Start

```bash
# Install dependencies
npm install

# Development - API Server
npm run dev

# Development - Worker (separate terminal)
npm run dev:worker

# Production - API Server
npm run build
npm start

# Production - Worker
npm run start:worker
```

## Configuration

Environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `3000` | API server port |
| `HOST` | `0.0.0.0` | API server host |
| `REDIS_HOST` | `localhost` | Redis host |
| `REDIS_PORT` | `6379` | Redis port |
| `REDIS_PASSWORD` | - | Redis password |
| `S3_ENDPOINT` | - | S3/MinIO endpoint |
| `S3_BUCKET` | `image-processing` | S3 bucket name |
| `S3_ACCESS_KEY_ID` | - | S3 access key |
| `S3_SECRET_ACCESS_KEY` | - | S3 secret key |
| `JWT_SECRET` | - | JWT signing secret |
| `QUEUE_CONCURRENCY` | `5` | Worker concurrency |
| `LOG_LEVEL` | `info` | Logging level |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `http://localhost:4318/v1/traces` | OpenTelemetry OTLP endpoint |
| `LOGGING_SERVICE_ENDPOINT` | - | Platform logging service gRPC endpoint |
| `CACHE_SERVICE_ENDPOINT` | - | Platform cache service gRPC endpoint |

## API Endpoints

### Image Operations

- `POST /api/v1/images/resize` - Resize image
- `POST /api/v1/images/convert` - Convert format
- `POST /api/v1/images/adjust` - Adjust brightness/contrast/saturation
- `POST /api/v1/images/rotate` - Rotate image
- `POST /api/v1/images/flip` - Flip image
- `POST /api/v1/images/watermark` - Add watermark
- `POST /api/v1/images/compress` - Compress image

### Upload

- `POST /api/v1/upload` - Upload image file
- `POST /api/v1/upload/url` - Upload from URL
- `GET /api/v1/images/:id` - Get image by ID

### Jobs (Async Processing)

- `GET /api/v1/jobs/:id` - Get job status
- `GET /api/v1/jobs/:id/result` - Get job result
- `DELETE /api/v1/jobs/:id` - Cancel job

### Health

- `GET /health/live` - Liveness probe (basic service running check)
- `GET /health/ready` - Readiness probe (checks Redis, S3, Cache dependencies)
- `GET /health` - Detailed health status with latency metrics

Health check response includes latency metrics for each dependency:

```json
{
  "status": "healthy",
  "timestamp": "2025-01-09T12:00:00.000Z",
  "version": "2.0.0",
  "uptime": 3600,
  "checks": {
    "redis": { "status": "healthy", "latencyMs": 2 },
    "s3": { "status": "healthy", "latencyMs": 15 },
    "cache": { "status": "healthy", "latencyMs": 3 }
  }
}
```

## Docker

```bash
# Build
docker build -t image-processing-service .

# Run API
docker run -p 3000:3000 image-processing-service

# Run Worker
docker run image-processing-service npm run start:worker
```

## Testing

```bash
# Run all tests
npm test

# Run property-based tests
npm run test:property

# Run with coverage
npm run test:coverage
```

## Tech Stack

- **Runtime**: Node.js 22 LTS
- **Framework**: Fastify
- **Image Processing**: Sharp (libvips)
- **Queue**: BullMQ
- **Cache**: Platform Cache Service (Redis backend)
- **Storage**: AWS S3 / MinIO
- **Observability**: OpenTelemetry SDK (W3C Trace Context), Platform Logging Service
- **Validation**: Zod schemas

## Observability

The service integrates with OpenTelemetry for distributed tracing:

- **W3C Trace Context**: Automatic propagation of `traceparent`/`tracestate` headers
- **OTLP Exporter**: Sends traces to configured collector endpoint
- **Auto-instrumentation**: HTTP, filesystem, and other Node.js operations
- **Structured Logging**: JSON logs with correlation IDs (traceId, spanId, requestId)

```typescript
// Trace context is automatically included in logs
{
  "timestamp": "2025-01-09T12:00:00.000Z",
  "level": "info",
  "message": "Image processed",
  "traceId": "abc123...",
  "spanId": "def456...",
  "requestId": "req-789..."
}
```

## Related Documentation

- [Design Document](../../.kiro/specs/image-processing-service/design.md)
- [Requirements](../../.kiro/specs/image-processing-service/requirements.md)
- [Tasks](../../.kiro/specs/image-processing-service/tasks.md)
