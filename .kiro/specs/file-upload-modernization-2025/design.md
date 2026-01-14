# Design Document: File Upload Service Modernization 2025

## Overview

This design modernizes the File Upload Service to state-of-the-art December 2025 standards, eliminating legacy patterns, integrating with platform services (logging-service, cache-service), and leveraging shared libraries (libs/go) for centralized, reusable code.

### Key Modernization Goals

1. **Go 1.24+** with modern dependencies (go-redis/v9, AWS SDK v2, OpenTelemetry 1.28+)
2. **Platform Integration** with logging-service (gRPC) and cache-service (distributed caching)
3. **Shared Library Adoption** from libs/go for domain, errors, validation, resilience, security
4. **Zero Redundancy** by removing internal implementations duplicated in libs
5. **Resilience Patterns** using circuit breakers and retry with exponential backoff
6. **Test Architecture** separation with property-based testing

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         File Upload Service (Go 1.24+)                       │
├─────────────────────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────────────────┐  │
│  │   REST API      │  │   Health API    │  │   Metrics API               │  │
│  │   /api/v1/*     │  │   /health/*     │  │   /metrics                  │  │
│  └────────┬────────┘  └────────┬────────┘  └─────────────┬───────────────┘  │
│           │                    │                         │                   │
│  ┌────────▼────────────────────▼─────────────────────────▼───────────────┐  │
│  │                        Middleware Layer                                │  │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────────┐ │  │
│  │  │ Auth     │ │ RateLimit│ │ Tracing  │ │ Logging  │ │ Correlation  │ │  │
│  │  │ (JWT)    │ │ (Cache)  │ │ (OTel)   │ │ (gRPC)   │ │ ID           │ │  │
│  │  └──────────┘ └──────────┘ └──────────┘ └──────────┘ └──────────────┘ │  │
│  └────────────────────────────────┬──────────────────────────────────────┘  │
│                                   │                                          │
│  ┌────────────────────────────────▼──────────────────────────────────────┐  │
│  │                        Handler Layer                                   │  │
│  │  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐ ┌──────────────┐  │  │
│  │  │ Upload       │ │ Chunk        │ │ Files        │ │ Download     │  │  │
│  │  │ Handler      │ │ Handler      │ │ Handler      │ │ Handler      │  │  │
│  │  └──────────────┘ └──────────────┘ └──────────────┘ └──────────────┘  │  │
│  └────────────────────────────────┬──────────────────────────────────────┘  │
│                                   │                                          │
│  ┌────────────────────────────────▼──────────────────────────────────────┐  │
│  │                        Service Layer                                   │  │
│  │  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐ ┌──────────────┐  │  │
│  │  │ Upload       │ │ Validation   │ │ Metadata     │ │ Async        │  │  │
│  │  │ Service      │ │ Service      │ │ Service      │ │ Processor    │  │  │
│  │  └──────────────┘ └──────────────┘ └──────────────┘ └──────────────┘  │  │
│  └────────────────────────────────┬──────────────────────────────────────┘  │
│                                   │                                          │
│  ┌────────────────────────────────▼──────────────────────────────────────┐  │
│  │                        Infrastructure Layer                            │  │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────────┐ │  │
│  │  │ Storage  │ │ Cache    │ │ Database │ │ Logging  │ │ Scanner      │ │  │
│  │  │ (S3)     │ │ Client   │ │ (PgSQL)  │ │ Client   │ │ (ClamAV)     │ │  │
│  │  └──────────┘ └──────────┘ └──────────┘ └──────────┘ └──────────────┘ │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
        ┌───────────────────────────┼───────────────────────────┐
        │                           │                           │
        ▼                           ▼                           ▼
┌───────────────┐         ┌───────────────┐         ┌───────────────┐
│ Cache Service │         │ Logging Svc   │         │ AWS S3        │
│ (gRPC:50051)  │         │ (gRPC:5001)   │         │               │
└───────────────┘         └───────────────┘         └───────────────┘
```

## Components and Interfaces

### 1. Storage Interface

```go
// Storage defines the contract for file storage providers
type Storage interface {
    Upload(ctx context.Context, req *UploadRequest) (*UploadResult, error)
    Download(ctx context.Context, path string) (io.ReadCloser, error)
    Delete(ctx context.Context, path string) error
    GeneratePresignedUploadURL(ctx context.Context, path string, expiry time.Duration) (string, error)
    GeneratePresignedDownloadURL(ctx context.Context, path string, expiry time.Duration) (string, error)
    Exists(ctx context.Context, path string) (bool, error)
}

// UploadRequest contains upload parameters
type UploadRequest struct {
    TenantID    string
    FileHash    string
    Filename    string
    Content     io.Reader
    ContentType string
    Size        int64
    Metadata    map[string]string
}

// UploadResult contains upload result
type UploadResult struct {
    Path      string
    URL       string
    ETag      string
    VersionID string
}
```

### 2. Cache Client Interface

```go
// CacheClient defines the contract for cache operations
type CacheClient interface {
    Get(ctx context.Context, key string) ([]byte, error)
    Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
    Delete(ctx context.Context, key string) error
    BatchGet(ctx context.Context, keys []string) (map[string][]byte, error)
    BatchSet(ctx context.Context, items map[string][]byte, ttl time.Duration) error
}

// CacheServiceClient wraps gRPC client with circuit breaker
type CacheServiceClient struct {
    client  cachepb.CacheServiceClient
    breaker *fault.CircuitBreaker[[]byte]
    ns      string // namespace: "file-upload"
}
```

### 3. Logging Client Interface

```go
// LoggingClient defines the contract for centralized logging
type LoggingClient interface {
    Log(ctx context.Context, entry *LogEntry) error
    LogBatch(ctx context.Context, entries []*LogEntry) error
    Close() error
}

// LoggingServiceClient wraps gRPC client with circuit breaker and fallback
type LoggingServiceClient struct {
    client   loggingpb.LoggingServiceClient
    breaker  *fault.CircuitBreaker[struct{}]
    fallback *slog.Logger // local fallback
    buffer   chan *LogEntry
}
```

### 4. Metadata Repository Interface

```go
// MetadataRepository defines the contract for file metadata storage
type MetadataRepository interface {
    Create(ctx context.Context, file *FileMetadata) error
    GetByID(ctx context.Context, id string) (*FileMetadata, error)
    GetByHash(ctx context.Context, tenantID, hash string) (*FileMetadata, error)
    List(ctx context.Context, req *ListRequest) (*ListResponse, error)
    Update(ctx context.Context, file *FileMetadata) error
    SoftDelete(ctx context.Context, id string) error
    UpdateScanStatus(ctx context.Context, id string, status ScanStatus) error
}
```

### 5. Validator Interface

```go
// Validator defines the contract for file validation
type Validator interface {
    Validate(ctx context.Context, req *ValidationRequest) (*ValidationResult, error)
    ValidateChunk(ctx context.Context, chunk *ChunkData) error
}

// ValidationRequest contains validation parameters
type ValidationRequest struct {
    Filename    string
    Content     io.Reader
    Size        int64
    TenantID    string
    IsChunked   bool
}

// ValidationResult contains validation result
type ValidationResult struct {
    Valid       bool
    MIMEType    string
    DetectedExt string
    Error       *errors.AppError
}
```

### 6. Async Processor Interface

```go
// AsyncProcessor defines the contract for async task processing
type AsyncProcessor interface {
    Submit(ctx context.Context, task Task) error
    Start() error
    Stop() error
    QueueDepth() int
}

// Task represents an async task
type Task struct {
    ID       string
    Type     TaskType
    Payload  []byte
    Priority int
    Retries  int
}

type TaskType string

const (
    TaskVirusScan       TaskType = "virus_scan"
    TaskThumbnail       TaskType = "thumbnail"
    TaskMetadataExtract TaskType = "metadata_extract"
)
```

## Data Models

### FileMetadata

```go
type FileMetadata struct {
    ID           string            `json:"id" db:"id"`
    TenantID     string            `json:"tenant_id" db:"tenant_id"`
    UserID       string            `json:"user_id" db:"user_id"`
    Filename     string            `json:"filename" db:"filename"`
    OriginalName string            `json:"original_name" db:"original_name"`
    MIMEType     string            `json:"mime_type" db:"mime_type"`
    Size         int64             `json:"size" db:"size"`
    Hash         string            `json:"hash" db:"hash"`
    StoragePath  string            `json:"storage_path" db:"storage_path"`
    StorageURL   string            `json:"storage_url,omitempty" db:"storage_url"`
    Status       FileStatus        `json:"status" db:"status"`
    ScanStatus   ScanStatus        `json:"scan_status" db:"scan_status"`
    Metadata     map[string]string `json:"metadata,omitempty" db:"metadata"`
    CreatedAt    time.Time         `json:"created_at" db:"created_at"`
    UpdatedAt    time.Time         `json:"updated_at" db:"updated_at"`
    DeletedAt    *time.Time        `json:"deleted_at,omitempty" db:"deleted_at"`
}

type FileStatus string

const (
    FileStatusPending    FileStatus = "pending"
    FileStatusUploaded   FileStatus = "uploaded"
    FileStatusProcessing FileStatus = "processing"
    FileStatusReady      FileStatus = "ready"
    FileStatusFailed     FileStatus = "failed"
    FileStatusDeleted    FileStatus = "deleted"
)

type ScanStatus string

const (
    ScanStatusPending  ScanStatus = "pending"
    ScanStatusScanning ScanStatus = "scanning"
    ScanStatusClean    ScanStatus = "clean"
    ScanStatusInfected ScanStatus = "infected"
    ScanStatusFailed   ScanStatus = "failed"
)
```

### ChunkSession

```go
type ChunkSession struct {
    ID             string    `json:"id"`
    TenantID       string    `json:"tenant_id"`
    UserID         string    `json:"user_id"`
    Filename       string    `json:"filename"`
    TotalSize      int64     `json:"total_size"`
    ChunkSize      int64     `json:"chunk_size"`
    TotalChunks    int       `json:"total_chunks"`
    UploadedChunks []int     `json:"uploaded_chunks"`
    Status         string    `json:"status"`
    CreatedAt      time.Time `json:"created_at"`
    ExpiresAt      time.Time `json:"expires_at"`
    CompletedAt    *time.Time `json:"completed_at,omitempty"`
}
```

### API Error Response (RFC 7807)

```go
type ProblemDetails struct {
    Type       string            `json:"type"`
    Title      string            `json:"title"`
    Status     int               `json:"status"`
    Detail     string            `json:"detail,omitempty"`
    Instance   string            `json:"instance,omitempty"`
    Extensions map[string]any    `json:"extensions,omitempty"`
}
```

## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system—essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*

### Property 1: Error Response Consistency

*For any* error returned by the service, the HTTP response SHALL include a valid RFC 7807 Problem Details body with type, title, status, and the X-Correlation-ID header SHALL be present.

**Validates: Requirements 4.2, 10.6, 11.1, 11.2, 11.3, 11.4**

### Property 2: Cache-Aside Pattern Correctness

*For any* file metadata operation:
- GET operations SHALL check cache before database
- CREATE/UPDATE operations SHALL invalidate cache after database write
- All cache keys SHALL have namespace prefix "file-upload:"

**Validates: Requirements 3.2, 3.3, 3.4**

### Property 3: Circuit Breaker Behavior

*For any* external dependency (S3, Cache, Database):
- After N consecutive failures (S3: 5, Cache: 3, DB: 3), circuit SHALL open
- While circuit is open, requests SHALL fail fast without calling dependency
- After timeout, circuit SHALL transition to half-open and allow probe requests

**Validates: Requirements 5.3, 5.4, 5.5, 2.5**

### Property 4: File Validation Completeness

*For any* uploaded file:
- MIME type SHALL be detected from content magic bytes
- File extension SHALL match detected MIME type
- File size SHALL not exceed tenant-specific limit
- MIME type SHALL be in tenant-specific allowlist
- Validation failure SHALL return specific error code

**Validates: Requirements 7.1, 7.2, 7.3, 7.5, 7.6**

### Property 5: Chunked Upload Integrity (Round-Trip)

*For any* chunked upload session:
- Each chunk checksum SHALL be verified using SHA-256
- Parallel chunk uploads SHALL not corrupt session state
- Assembled file hash SHALL match expected hash
- Expired sessions SHALL be automatically cleaned up

**Validates: Requirements 8.2, 8.3, 8.4, 8.5**

### Property 6: Security Enforcement

*For any* request:
- Filenames SHALL be sanitized to prevent path traversal (no "..", "/", "\")
- Tenant isolation SHALL be enforced (tenant A cannot access tenant B files)
- Sensitive data in logs SHALL be redacted (tokens, passwords)
- Token comparison SHALL use constant-time algorithm

**Validates: Requirements 13.1, 13.3, 13.5, 13.6, 4.5, 13.2**

### Property 7: Observability Completeness

*For any* request:
- Correlation ID SHALL be present in context and response header
- Trace span SHALL be created with W3C Trace Context
- Logs SHALL include correlation_id, tenant_id, user_id when available
- Upload metrics (latency, size, error rate) SHALL be recorded

**Validates: Requirements 2.2, 2.3, 12.1, 12.2, 12.4**

### Property 8: Graceful Shutdown Behavior

*For any* SIGTERM signal:
- New requests SHALL be rejected immediately
- In-flight requests SHALL complete within timeout
- After timeout, service SHALL force terminate
- All cleanup handlers SHALL be invoked

**Validates: Requirements 17.2, 17.3, 17.4, 17.5**

### Property 9: Rate Limiting Correctness

*For any* rate-limited request:
- HTTP 429 response SHALL include Retry-After header
- Rate limits SHALL be isolated per tenant
- Sliding window algorithm SHALL correctly count requests in window
- Cache unavailability SHALL trigger local fallback

**Validates: Requirements 6.2, 6.3, 6.4, 6.5**

### Property 10: Storage Path Tenant Isolation

*For any* generated storage path, the path SHALL contain the tenant ID as the first path segment, ensuring tenant-isolated hierarchical structure.

**Validates: Requirements 9.5**

### Property 11: Presigned URL Validity

*For any* presigned URL:
- Upload URLs SHALL allow PUT operations within expiry time
- Download URLs SHALL allow GET operations within expiry time
- Expired URLs SHALL return 403 Forbidden

**Validates: Requirements 9.3, 9.4**

### Property 12: Pagination Cursor Consistency

*For any* paginated list response:
- Cursor SHALL be decodable to valid pagination state
- Using cursor SHALL return next page of results
- Empty cursor on last page

**Validates: Requirements 4.7, 10.2**

### Property 13: Soft Delete Correctness

*For any* deleted file:
- deleted_at timestamp SHALL be set
- File SHALL not appear in list queries
- File SHALL be retrievable by ID with deleted status

**Validates: Requirements 10.3**

### Property 14: Database Transaction Atomicity

*For any* multi-step database operation:
- All steps SHALL succeed or all SHALL rollback
- Partial state SHALL never be visible

**Validates: Requirements 10.5**

### Property 15: Logging Service Fallback

*For any* logging operation when Logging_Service is unavailable:
- Logs SHALL be written to local structured JSON
- Circuit breaker SHALL prevent repeated failed calls
- Recovery SHALL resume remote logging

**Validates: Requirements 2.4, 2.5**

### Property 16: Configuration Validation

*For any* configuration:
- Missing required values SHALL cause startup failure
- Environment variables SHALL override config file values
- Invalid values SHALL produce clear error messages

**Validates: Requirements 15.2, 15.3, 15.4**

### Property 17: Health Check Accuracy

*For any* health check:
- Liveness SHALL return 200 when process is running
- Readiness SHALL return unhealthy when database is unavailable
- Readiness SHALL return degraded when cache is unavailable

**Validates: Requirements 16.3, 16.4**

### Property 18: Async Task Retry Behavior

*For any* failed async task:
- Retry delay SHALL increase exponentially
- Maximum retry count SHALL be respected
- Failed tasks SHALL be logged with error details

**Validates: Requirements 14.4, 5.2**

## Error Handling

### Error Code Mapping

| Error Code | HTTP Status | Description |
|------------|-------------|-------------|
| INVALID_FILE_TYPE | 400 | File type not in allowlist |
| FILE_TOO_LARGE | 400 | File exceeds size limit |
| EXTENSION_MISMATCH | 400 | Extension doesn't match content |
| INVALID_CHUNK | 400 | Invalid chunk index |
| DUPLICATE_CHUNK | 400 | Chunk already uploaded |
| CHECKSUM_MISMATCH | 400 | Chunk checksum verification failed |
| MISSING_TOKEN | 401 | No authentication token |
| INVALID_TOKEN | 401 | Token validation failed |
| TOKEN_EXPIRED | 401 | Token has expired |
| ACCESS_DENIED | 403 | Tenant isolation violation |
| FILE_NOT_FOUND | 404 | File does not exist |
| SESSION_NOT_FOUND | 404 | Upload session not found |
| SESSION_EXPIRED | 410 | Upload session has expired |
| RATE_LIMIT_EXCEEDED | 429 | Too many requests |
| STORAGE_ERROR | 500 | S3 operation failed |
| DATABASE_ERROR | 500 | PostgreSQL operation failed |
| SCANNER_ERROR | 500 | ClamAV unavailable |
| INTERNAL_ERROR | 500 | Unexpected error |

### Error Response Format

```json
{
  "type": "https://api.example.com/errors/INVALID_FILE_TYPE",
  "title": "Invalid File Type",
  "status": 400,
  "detail": "File type 'application/x-executable' is not allowed",
  "instance": "/api/v1/upload",
  "extensions": {
    "request_id": "req-123",
    "correlation_id": "corr-456",
    "allowed_types": ["image/jpeg", "image/png", "application/pdf"]
  }
}
```

## Testing Strategy

### Dual Testing Approach

The service uses both unit tests and property-based tests for comprehensive coverage:

- **Unit tests**: Verify specific examples, edge cases, and error conditions
- **Property tests**: Verify universal properties across all inputs using rapid library

### Property-Based Testing Configuration

- **Library**: pgregory.net/rapid (Go property-based testing)
- **Minimum iterations**: 100 per property test
- **Tag format**: `Feature: file-upload-modernization-2025, Property N: [property_text]`

### Test Organization

```
services/file-upload/
├── internal/           # Source code
│   ├── api/
│   ├── service/
│   ├── infrastructure/
│   └── domain/
└── tests/              # All tests (separate from source)
    ├── unit/
    │   ├── service/
    │   ├── infrastructure/
    │   └── domain/
    ├── property/
    │   ├── validation_property_test.go
    │   ├── cache_property_test.go
    │   ├── storage_property_test.go
    │   ├── security_property_test.go
    │   └── resilience_property_test.go
    ├── integration/
    │   ├── cache_integration_test.go
    │   ├── storage_integration_test.go
    │   └── logging_integration_test.go
    └── testutil/
        ├── generators.go
        ├── mocks.go
        └── fixtures.go
```

### Coverage Requirements

- Core business logic: 80%+ coverage
- Property tests: All 18 correctness properties
- Integration tests: All external dependencies
