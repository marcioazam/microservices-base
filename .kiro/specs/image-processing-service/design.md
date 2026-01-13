# Design Document: Image Processing Service

## Overview

The Image Processing Service is a Node.js microservice providing image manipulation capabilities through a RESTful API. It supports resizing, format conversion, adjustments (brightness, contrast, saturation), rotation, watermarking, and compression. The service is designed for horizontal scalability with asynchronous processing, caching, and integration with the Auth Platform ecosystem.

### Key Design Decisions

1. **Sharp as primary image library**: High-performance, libvips-based library for Node.js
2. **Fastify over Express**: Better performance and TypeScript support
3. **Bull/BullMQ for job queues**: Redis-backed, battle-tested job processing
4. **S3-compatible storage**: Flexible storage backend (AWS S3, MinIO, etc.)
5. **Redis for caching**: Fast, distributed cache with TTL support

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           API Gateway Layer                              │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐    │
│  │   Upload    │  │   Process   │  │    Jobs     │  │   Health    │    │
│  │  Controller │  │  Controller │  │  Controller │  │  Controller │    │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘  └─────────────┘    │
└─────────┼────────────────┼────────────────┼────────────────────────────┘
          │                │                │
          ▼                ▼                ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                          Service Layer                                   │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐         │
│  │  ImageService   │  │   JobService    │  │  CacheService   │         │
│  │  - resize()     │  │  - enqueue()    │  │  - get()        │         │
│  │  - convert()    │  │  - getStatus()  │  │  - set()        │         │
│  │  - adjust()     │  │  - getResult()  │  │  - invalidate() │         │
│  │  - rotate()     │  │                 │  │                 │         │
│  │  - watermark()  │  │                 │  │                 │         │
│  │  - compress()   │  │                 │  │                 │         │
│  └────────┬────────┘  └────────┬────────┘  └────────┬────────┘         │
└───────────┼────────────────────┼────────────────────┼───────────────────┘
            │                    │                    │
            ▼                    ▼                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                       Infrastructure Layer                               │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐    │
│  │    Sharp    │  │  BullMQ     │  │    Redis    │  │  S3 Storage │    │
│  │  (libvips)  │  │   Queue     │  │    Cache    │  │   (MinIO)   │    │
│  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘    │
└─────────────────────────────────────────────────────────────────────────┘
```

### Request Flow

```
┌──────┐     ┌─────────┐     ┌───────────┐     ┌─────────────┐
│Client│────▶│API Layer│────▶│Auth Check │────▶│Rate Limiter │
└──────┘     └─────────┘     └───────────┘     └──────┬──────┘
                                                       │
                    ┌──────────────────────────────────┘
                    ▼
             ┌─────────────┐
             │Cache Check  │
             └──────┬──────┘
                    │
        ┌───────────┴───────────┐
        │                       │
        ▼ (cache hit)           ▼ (cache miss)
┌───────────────┐       ┌───────────────┐
│ Return Cached │       │ Process Image │
└───────────────┘       └───────┬───────┘
                                │
                    ┌───────────┴───────────┐
                    │                       │
                    ▼ (sync)                ▼ (async)
            ┌───────────────┐       ┌───────────────┐
            │Return Result  │       │ Queue Job     │
            │+ Cache        │       │ Return JobID  │
            └───────────────┘       └───────────────┘
```

## Components and Interfaces

### API Controllers

```typescript
// src/api/controllers/image.controller.ts
interface ImageController {
  resize(req: ResizeRequest): Promise<ImageResponse>;
  convert(req: ConvertRequest): Promise<ImageResponse>;
  adjust(req: AdjustRequest): Promise<ImageResponse>;
  rotate(req: RotateRequest): Promise<ImageResponse>;
  watermark(req: WatermarkRequest): Promise<ImageResponse>;
  compress(req: CompressRequest): Promise<ImageResponse>;
}

// src/api/controllers/upload.controller.ts
interface UploadController {
  uploadFile(req: MultipartRequest): Promise<UploadResponse>;
  uploadFromUrl(req: UrlUploadRequest): Promise<UploadResponse>;
}

// src/api/controllers/job.controller.ts
interface JobController {
  getStatus(jobId: string): Promise<JobStatusResponse>;
  getResult(jobId: string): Promise<JobResultResponse>;
  cancel(jobId: string): Promise<void>;
}
```

### Service Layer

```typescript
// src/services/image/image.service.ts
interface ImageService {
  resize(input: Buffer, options: ResizeOptions): Promise<ProcessedImage>;
  convert(input: Buffer, options: ConvertOptions): Promise<ProcessedImage>;
  adjust(input: Buffer, options: AdjustOptions): Promise<ProcessedImage>;
  rotate(input: Buffer, options: RotateOptions): Promise<ProcessedImage>;
  flip(input: Buffer, options: FlipOptions): Promise<ProcessedImage>;
  watermark(input: Buffer, options: WatermarkOptions): Promise<ProcessedImage>;
  compress(input: Buffer, options: CompressOptions): Promise<ProcessedImage>;
}

// src/services/job/job.service.ts
interface JobService {
  enqueue(operation: ImageOperation): Promise<string>;
  getStatus(jobId: string): Promise<JobStatus>;
  getResult(jobId: string): Promise<ProcessedImage | null>;
  cancel(jobId: string): Promise<boolean>;
}

// src/services/cache/cache.service.ts
interface CacheService {
  generateKey(input: Buffer, operation: ImageOperation): string;
  get(key: string): Promise<ProcessedImage | null>;
  set(key: string, image: ProcessedImage, ttl?: number): Promise<void>;
  invalidate(pattern: string): Promise<number>;
  getStats(): Promise<CacheStats>;
}

// src/services/storage/storage.service.ts
interface StorageService {
  upload(image: Buffer, key: string): Promise<string>;
  download(key: string): Promise<Buffer>;
  getSignedUrl(key: string, expiresIn: number): Promise<string>;
  delete(key: string): Promise<void>;
}
```

### Request/Response Types

```typescript
// src/domain/types/requests.ts
interface ResizeOptions {
  width?: number;
  height?: number;
  maintainAspectRatio?: boolean;
  fit?: 'cover' | 'contain' | 'fill' | 'inside' | 'outside';
  quality?: number; // 1-100
}

interface ConvertOptions {
  format: 'jpeg' | 'png' | 'gif' | 'webp' | 'tiff';
  quality?: number; // 1-100, for lossy formats
  backgroundColor?: string; // hex color for transparency replacement
}

interface AdjustOptions {
  brightness?: number; // -100 to 100
  contrast?: number;   // -100 to 100
  saturation?: number; // -100 to 100
}

interface RotateOptions {
  angle: number; // 0-360
  backgroundColor?: string;
}

interface FlipOptions {
  horizontal?: boolean;
  vertical?: boolean;
}

interface WatermarkOptions {
  type: 'text' | 'image';
  content: string; // text or image URL/path
  position: WatermarkPosition;
  opacity?: number; // 0-100
  font?: FontOptions; // for text watermarks
}

interface CompressOptions {
  mode: 'lossy' | 'lossless';
  quality?: number; // 1-100, for lossy
  format?: 'jpeg' | 'png' | 'webp' | 'gif';
}

type WatermarkPosition = 
  | 'top-left' | 'top-center' | 'top-right'
  | 'center-left' | 'center' | 'center-right'
  | 'bottom-left' | 'bottom-center' | 'bottom-right'
  | { x: number; y: number };

interface FontOptions {
  family?: string;
  size?: number;
  color?: string;
  weight?: 'normal' | 'bold';
}

// src/domain/types/responses.ts
interface ProcessedImage {
  buffer: Buffer;
  metadata: ImageMetadata;
}

interface ImageMetadata {
  width: number;
  height: number;
  format: string;
  size: number;
  hasAlpha: boolean;
}

interface ImageResponse {
  success: boolean;
  requestId: string;
  data?: {
    image?: string; // base64 or URL
    metadata: ImageMetadata;
  };
  error?: ErrorDetails;
}

interface JobStatusResponse {
  success: boolean;
  requestId: string;
  data: {
    jobId: string;
    status: 'pending' | 'processing' | 'completed' | 'failed';
    progress?: number;
    createdAt: string;
    updatedAt: string;
    error?: string;
  };
}
```

## Data Models

### Job Entity

```typescript
// src/domain/entities/job.entity.ts
interface Job {
  id: string;
  userId: string;
  operation: ImageOperation;
  status: JobStatus;
  progress: number;
  inputKey: string;
  outputKey?: string;
  error?: string;
  createdAt: Date;
  updatedAt: Date;
  completedAt?: Date;
}

type JobStatus = 'pending' | 'processing' | 'completed' | 'failed' | 'cancelled';

interface ImageOperation {
  type: OperationType;
  options: ResizeOptions | ConvertOptions | AdjustOptions | 
           RotateOptions | WatermarkOptions | CompressOptions;
}

type OperationType = 'resize' | 'convert' | 'adjust' | 'rotate' | 
                     'flip' | 'watermark' | 'compress' | 'batch';
```

### Cache Entry

```typescript
// src/domain/entities/cache-entry.entity.ts
interface CacheEntry {
  key: string;
  imageBuffer: Buffer;
  metadata: ImageMetadata;
  createdAt: Date;
  expiresAt: Date;
  hits: number;
}
```

## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system—essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*

### Property 1: Resize Dimensions Accuracy

*For any* valid image and any valid positive dimensions (width, height), when resizing without maintainAspectRatio, the output image dimensions SHALL exactly match the requested dimensions.

**Validates: Requirements 1.1**

### Property 2: Aspect Ratio Preservation

*For any* valid image with aspect ratio R, when resizing with maintainAspectRatio=true, the output image aspect ratio SHALL equal R within a tolerance of 0.01.

**Validates: Requirements 1.2**

### Property 3: Quality-Size Relationship

*For any* valid image and two quality values Q1 < Q2, when processing with the same operation, the output file size for Q1 SHALL be less than or equal to the output file size for Q2.

**Validates: Requirements 1.3, 2.2**

### Property 4: Batch Processing Completeness

*For any* batch request containing N valid images, the response SHALL contain exactly N results, one for each input image.

**Validates: Requirements 1.4**

### Property 5: Invalid Dimension Rejection

*For any* resize request with width ≤ 0 or height ≤ 0, the API SHALL return HTTP 400 with an error message describing the invalid dimension.

**Validates: Requirements 1.5**

### Property 6: Metadata Accuracy

*For any* successfully processed image, the returned metadata dimensions SHALL exactly match the actual dimensions of the output image buffer.

**Validates: Requirements 1.6**

### Property 7: Format Conversion Correctness

*For any* valid image and any supported target format, the output image SHALL be decodable as the target format and the metadata format field SHALL match the target format.

**Validates: Requirements 2.1**

### Property 8: Transparency Background Handling

*For any* image with transparency converted to JPEG with a specified background color, the output image SHALL have no transparency and pixels that were transparent SHALL have the specified background color.

**Validates: Requirements 2.3**

### Property 9: Unsupported Format Rejection

*For any* conversion request with an unsupported format string, the API SHALL return HTTP 400 with an error listing supported formats.

**Validates: Requirements 2.4**

### Property 10: Brightness Adjustment Identity

*For any* valid image, applying brightness adjustment with value 0 SHALL produce an output image with identical pixel values to the input.

**Validates: Requirements 3.1**

### Property 11: Contrast Adjustment Identity

*For any* valid image, applying contrast adjustment with value 0 SHALL produce an output image with identical pixel values to the input.

**Validates: Requirements 3.2**

### Property 12: Saturation Desaturation

*For any* valid color image, applying saturation adjustment with value -100 SHALL produce a grayscale output where all pixels have equal R, G, B values.

**Validates: Requirements 3.3**

### Property 13: Adjustment Range Validation

*For any* adjustment request with brightness, contrast, or saturation value outside [-100, 100], the API SHALL return HTTP 400 with valid range information.

**Validates: Requirements 3.5**

### Property 14: Rotation Identity

*For any* valid image, rotating by 0 degrees or 360 degrees SHALL produce an output image with identical pixel values to the input.

**Validates: Requirements 4.1**

### Property 15: Flip Idempotence

*For any* valid image, applying the same flip operation (horizontal or vertical) twice SHALL produce an output image with identical pixel values to the input.

**Validates: Requirements 4.2, 4.3**

### Property 16: Watermark Presence

*For any* valid image and valid watermark configuration, the output image SHALL differ from the input image at pixels within the watermark region.

**Validates: Requirements 5.1, 5.2**

### Property 17: Watermark Opacity Zero

*For any* valid image with watermark opacity set to 0, the output image SHALL have identical pixel values to the input image.

**Validates: Requirements 5.3**

### Property 18: Lossy Compression Effectiveness

*For any* valid image with lossy compression at quality < 100, the output file size SHALL be less than or equal to the input file size.

**Validates: Requirements 6.1**

### Property 19: Lossless Compression Integrity

*For any* valid image with lossless compression, decoding the output SHALL produce pixel values identical to the input.

**Validates: Requirements 6.2**

### Property 20: Compression Statistics Accuracy

*For any* compression operation, the returned statistics SHALL satisfy: ratio = originalSize / newSize (within 0.01 tolerance).

**Validates: Requirements 6.3**

### Property 21: Upload Validation

*For any* valid image file uploaded via multipart/form-data, the upload SHALL succeed and return a valid storage reference.

**Validates: Requirements 7.1**

### Property 22: Invalid File Rejection

*For any* uploaded file that is not a valid image (random bytes, text file, etc.), the API SHALL return HTTP 400.

**Validates: Requirements 7.6**

### Property 23: Storage Round-Trip

*For any* successfully stored image, retrieving it by ID within the TTL SHALL return the exact same image buffer.

**Validates: Requirements 7.3, 7.4**

### Property 24: Async Job ID Return

*For any* request with async=true, the API SHALL return immediately (within 100ms) with a valid job ID.

**Validates: Requirements 8.1**

### Property 25: Job Status Validity

*For any* valid job ID, the status endpoint SHALL return one of: pending, processing, completed, or failed.

**Validates: Requirements 8.2, 8.3**

### Property 26: Completed Job Result Availability

*For any* job with status=completed, requesting the result SHALL return the processed image or a valid download URL.

**Validates: Requirements 8.4, 8.6**

### Property 27: Cache Key Determinism

*For any* image buffer and operation parameters, generating the cache key multiple times SHALL produce the same key.

**Validates: Requirements 9.1**

### Property 28: Cache Hit Consistency

*For any* cached operation, the cached result SHALL be byte-identical to the result of re-processing the same input with the same parameters.

**Validates: Requirements 9.2**

### Property 29: JWT Extraction

*For any* request with a valid JWT token, the extracted user ID and permissions SHALL match the token claims.

**Validates: Requirements 10.1**

### Property 30: Authentication Enforcement

*For any* request without a valid JWT token, the API SHALL return HTTP 401.

**Validates: Requirements 10.2**

### Property 31: Authorization Enforcement

*For any* request where the user lacks required permissions, the API SHALL return HTTP 403.

**Validates: Requirements 10.3**

### Property 32: Response Format Consistency

*For any* API response, it SHALL contain: success (boolean), requestId (string), and either data (on success) or error (on failure).

**Validates: Requirements 12.1, 12.2**

### Property 33: Request ID Presence

*For any* API response, the requestId field SHALL be a non-empty string.

**Validates: Requirements 12.3**

### Property 34: HTTP Status Code Correctness

*For any* successful operation, the HTTP status SHALL be 200 or 201. For validation errors, 400. For auth errors, 401 or 403. For server errors, 500.

**Validates: Requirements 12.4**

### Property 35: Response Format Options

*For any* image response, when Accept header is application/json, the image SHALL be base64-encoded. When Accept is image/*, the response SHALL be binary.

**Validates: Requirements 12.5**

## Error Handling

### Error Categories

```typescript
// src/domain/errors/errors.ts
enum ErrorCode {
  // Validation Errors (400)
  INVALID_DIMENSIONS = 'INVALID_DIMENSIONS',
  INVALID_FORMAT = 'INVALID_FORMAT',
  INVALID_QUALITY = 'INVALID_QUALITY',
  INVALID_ADJUSTMENT_VALUE = 'INVALID_ADJUSTMENT_VALUE',
  INVALID_IMAGE = 'INVALID_IMAGE',
  FILE_TOO_LARGE = 'FILE_TOO_LARGE',
  
  // Authentication Errors (401)
  MISSING_TOKEN = 'MISSING_TOKEN',
  INVALID_TOKEN = 'INVALID_TOKEN',
  EXPIRED_TOKEN = 'EXPIRED_TOKEN',
  
  // Authorization Errors (403)
  INSUFFICIENT_PERMISSIONS = 'INSUFFICIENT_PERMISSIONS',
  
  // Not Found Errors (404)
  IMAGE_NOT_FOUND = 'IMAGE_NOT_FOUND',
  JOB_NOT_FOUND = 'JOB_NOT_FOUND',
  
  // Rate Limit Errors (429)
  RATE_LIMIT_EXCEEDED = 'RATE_LIMIT_EXCEEDED',
  
  // Server Errors (500)
  PROCESSING_ERROR = 'PROCESSING_ERROR',
  STORAGE_ERROR = 'STORAGE_ERROR',
  QUEUE_ERROR = 'QUEUE_ERROR',
}

interface ErrorDetails {
  code: ErrorCode;
  message: string;
  details?: Record<string, unknown>;
}
```

### Error Response Format

```typescript
interface ErrorResponse {
  success: false;
  requestId: string;
  error: {
    code: string;
    message: string;
    details?: Record<string, unknown>;
  };
}
```

## Testing Strategy

### Property-Based Testing

The service will use **fast-check** for property-based testing in TypeScript/Node.js.

Configuration:
- Minimum 100 iterations per property test
- Each test tagged with property number and requirements reference

### Test Categories

1. **Unit Tests**: Individual service methods, validators, utilities
2. **Property Tests**: Universal properties from Correctness Properties section
3. **Integration Tests**: API endpoints with real dependencies
4. **E2E Tests**: Full request flows including async processing

### Test File Structure

```
tests/
├── unit/
│   ├── services/
│   │   ├── image/
│   │   │   ├── image.service.spec.ts
│   │   │   └── image.service.property.spec.ts
│   │   ├── cache/
│   │   │   └── cache.service.spec.ts
│   │   └── job/
│   │       └── job.service.spec.ts
│   └── validators/
│       └── request.validator.spec.ts
├── integration/
│   ├── api/
│   │   ├── image.controller.spec.ts
│   │   └── job.controller.spec.ts
│   └── storage/
│       └── storage.service.spec.ts
└── e2e/
    ├── resize-flow.spec.ts
    ├── async-processing-flow.spec.ts
    └── cache-flow.spec.ts
```

### Property Test Example

```typescript
import fc from 'fast-check';
import { ImageService } from '../../../src/services/image/image.service';

describe('ImageService Properties', () => {
  // Feature: image-processing-service, Property 1: Resize Dimensions Accuracy
  // Validates: Requirements 1.1
  it('should resize to exact dimensions when maintainAspectRatio is false', async () => {
    await fc.assert(
      fc.asyncProperty(
        fc.integer({ min: 1, max: 4096 }),
        fc.integer({ min: 1, max: 4096 }),
        async (width, height) => {
          const input = await generateTestImage();
          const result = await imageService.resize(input, {
            width,
            height,
            maintainAspectRatio: false,
          });
          
          expect(result.metadata.width).toBe(width);
          expect(result.metadata.height).toBe(height);
        }
      ),
      { numRuns: 100 }
    );
  });

  // Feature: image-processing-service, Property 15: Flip Idempotence
  // Validates: Requirements 4.2, 4.3
  it('should return to original after double flip', async () => {
    await fc.assert(
      fc.asyncProperty(
        fc.boolean(),
        async (horizontal) => {
          const input = await generateTestImage();
          const options = { horizontal, vertical: !horizontal };
          
          const once = await imageService.flip(input, options);
          const twice = await imageService.flip(once.buffer, options);
          
          expect(twice.buffer).toEqual(input);
        }
      ),
      { numRuns: 100 }
    );
  });
});
```
