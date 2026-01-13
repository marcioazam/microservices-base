# Design Document: Image Processing Service Modernization 2025

## Overview

This design document describes the architectural modernization of the Image Processing Service to achieve state-of-the-art 2025 standards. The modernization eliminates redundancy, integrates with platform services (Logging Service, Cache Service), centralizes shared logic, and ensures production-ready quality with comprehensive property-based testing.

The service processes images using Sharp (v0.33.5) with operations including resize, convert, adjust, rotate, flip, watermark, and compress. It supports both synchronous processing and async job queues via BullMQ.

## Architecture

### Current Architecture Issues

1. **Redundant Logging**: Direct pino instantiation in multiple files (server.ts, logger.ts, worker.ts)
2. **Local Cache Implementation**: Custom cache.service.ts duplicates platform Cache Service functionality
3. **Scattered Validation**: Validation logic exists in both validators and ImageService methods
4. **Duplicate Error Handling**: Error handling repeated across controllers
5. **Custom Tracing**: In-memory span storage instead of OpenTelemetry SDK
6. **CachedImageService Wrapper**: Unnecessary abstraction layer

### Target Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                        API Layer (Fastify)                          │
├─────────────────────────────────────────────────────────────────────┤
│  Routes → Controllers → Validators → Services → Response Utils      │
└─────────────────────────────────────────────────────────────────────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        ▼                     ▼                     ▼
┌───────────────┐    ┌───────────────┐    ┌───────────────┐
│ ImageService  │    │ StorageService│    │  JobService   │
│   (Sharp)     │    │    (S3)       │    │  (BullMQ)     │
└───────────────┘    └───────────────┘    └───────────────┘
        │                     │                     │
        └─────────────────────┼─────────────────────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        ▼                     ▼                     ▼
┌───────────────┐    ┌───────────────┐    ┌───────────────┐
│ Logging Client│    │ Cache Client  │    │ OpenTelemetry │
│ (platform)    │    │ (platform)    │    │    SDK        │
└───────────────┘    └───────────────┘    └───────────────┘
        │                     │                     │
        ▼                     ▼                     ▼
┌───────────────┐    ┌───────────────┐    ┌───────────────┐
│Logging Service│    │ Cache Service │    │ OTLP Exporter │
│  (gRPC)       │    │   (gRPC)      │    │               │
└───────────────┘    └───────────────┘    └───────────────┘
```

### Directory Structure (Modernized)

```
services/image-processing-service/
├── src/
│   ├── api/
│   │   ├── controllers/
│   │   │   ├── image.controller.ts      # Consolidated image operations
│   │   │   ├── upload.controller.ts     # Upload operations
│   │   │   ├── job.controller.ts        # Job management
│   │   │   └── index.ts
│   │   ├── middlewares/
│   │   │   ├── auth.middleware.ts
│   │   │   ├── rate-limit.middleware.ts
│   │   │   └── index.ts
│   │   ├── routes/
│   │   │   └── index.ts
│   │   └── validators/
│   │       ├── schemas.ts               # All Zod schemas centralized
│   │       └── index.ts
│   ├── config/
│   │   └── index.ts                     # Centralized config with Zod
│   ├── domain/
│   │   ├── errors/
│   │   │   ├── app-error.ts
│   │   │   ├── error-codes.ts
│   │   │   └── index.ts
│   │   └── types/
│   │       ├── common.ts
│   │       ├── operations.ts            # Discriminated union types
│   │       ├── responses.ts
│   │       └── index.ts
│   ├── infrastructure/
│   │   ├── cache/
│   │   │   └── client.ts                # Platform Cache Service client
│   │   ├── logging/
│   │   │   └── client.ts                # Platform Logging Service client
│   │   ├── observability/
│   │   │   ├── metrics.ts               # Prometheus metrics
│   │   │   └── tracing.ts               # OpenTelemetry SDK
│   │   ├── health.ts
│   │   └── server.ts
│   ├── services/
│   │   ├── image/
│   │   │   ├── image.service.ts         # Pure image processing
│   │   │   └── index.ts
│   │   ├── storage/
│   │   │   ├── storage.service.ts
│   │   │   └── index.ts
│   │   └── job/
│   │       ├── job.service.ts
│   │       └── index.ts
│   ├── shared/
│   │   └── utils/
│   │       └── response.ts              # Centralized response utilities
│   ├── index.ts
│   └── worker.ts
└── tests/
    ├── unit/
    │   ├── services/
    │   │   └── image/
    │   │       └── image.property.spec.ts
    │   └── validators/
    │       └── validation.property.spec.ts
    ├── integration/
    │   └── api/
    │       └── endpoints.spec.ts
    └── setup.ts
```

## Components and Interfaces

### Platform Logging Client

```typescript
// src/infrastructure/logging/client.ts
import { LogLevel } from './types';

interface LogContext {
  requestId?: string;
  traceId?: string;
  spanId?: string;
  userId?: string;
  operation?: string;
  [key: string]: unknown;
}

interface LoggingClient {
  log(level: LogLevel, message: string, context?: LogContext): void;
  info(message: string, context?: LogContext): void;
  warn(message: string, context?: LogContext): void;
  error(message: string, error?: Error, context?: LogContext): void;
  debug(message: string, context?: LogContext): void;
  child(context: LogContext): LoggingClient;
}

class PlatformLoggingClient implements LoggingClient {
  private serviceName = 'image-processing-service';
  private fallbackLogger: Console;
  
  constructor(private endpoint: string) {
    this.fallbackLogger = console;
  }
  
  // Implementation sends to Logging Service via gRPC
  // Falls back to local console on connection failure
}

export const logger = new PlatformLoggingClient(config.logging.endpoint);
```

### Platform Cache Client

```typescript
// src/infrastructure/cache/client.ts
interface CacheClient {
  get<T>(key: string): Promise<T | null>;
  set<T>(key: string, value: T, ttlSeconds?: number): Promise<void>;
  delete(key: string): Promise<boolean>;
  exists(key: string): Promise<boolean>;
  invalidatePattern(pattern: string): Promise<number>;
}

interface ImageCacheClient extends CacheClient {
  getImage(key: string): Promise<ProcessedImage | null>;
  setImage(key: string, image: ProcessedImage, ttlSeconds?: number): Promise<void>;
  generateKey(inputHash: string, operation: ImageOperation): string;
}

class PlatformCacheClient implements ImageCacheClient {
  private readonly prefix = 'img:';
  
  constructor(private endpoint: string) {}
  
  generateKey(inputHash: string, operation: ImageOperation): string {
    const operationHash = crypto
      .createHash('sha256')
      .update(`${inputHash}:${JSON.stringify(operation)}`)
      .digest('hex');
    return `${this.prefix}${operationHash}`;
  }
  
  // Implementation communicates with Cache Service via gRPC
  // Graceful degradation on connection failure
}

export const cacheClient = new PlatformCacheClient(config.cache.endpoint);
```

### Centralized Validators

```typescript
// src/api/validators/schemas.ts
import { z } from 'zod';
import { SUPPORTED_FORMATS } from '@domain/types';

// Base schemas
const dimensionSchema = z.number().int().positive().max(10000);
const qualitySchema = z.number().int().min(1).max(100);
const hexColorSchema = z.string().regex(/^#[0-9A-Fa-f]{6}$/);
const formatSchema = z.enum(['jpeg', 'png', 'gif', 'webp', 'tiff']);

// Operation schemas
export const resizeSchema = z.object({
  width: dimensionSchema.optional(),
  height: dimensionSchema.optional(),
  maintainAspectRatio: z.boolean().default(true),
  fit: z.enum(['cover', 'contain', 'fill', 'inside', 'outside']).default('inside'),
  quality: qualitySchema.optional(),
}).refine(
  (data) => data.width !== undefined || data.height !== undefined,
  { message: 'At least one of width or height must be provided' }
);

export const convertSchema = z.object({
  format: formatSchema,
  quality: qualitySchema.optional(),
  backgroundColor: hexColorSchema.optional(),
});

export const adjustSchema = z.object({
  brightness: z.number().min(-100).max(100).optional(),
  contrast: z.number().min(-100).max(100).optional(),
  saturation: z.number().min(-100).max(100).optional(),
});

export const rotateSchema = z.object({
  angle: z.number().min(0).max(360),
  backgroundColor: hexColorSchema.optional(),
});

export const flipSchema = z.object({
  horizontal: z.boolean().optional(),
  vertical: z.boolean().optional(),
});

export const watermarkSchema = z.object({
  type: z.enum(['text', 'image']),
  content: z.string().min(1),
  position: z.union([
    z.enum(['top-left', 'top-center', 'top-right', 'center-left', 'center', 
            'center-right', 'bottom-left', 'bottom-center', 'bottom-right']),
    z.object({ x: z.number(), y: z.number() }),
  ]),
  opacity: z.number().min(0).max(100).optional(),
  font: z.object({
    family: z.string().optional(),
    size: z.number().positive().optional(),
    color: hexColorSchema.optional(),
    weight: z.enum(['normal', 'bold']).optional(),
  }).optional(),
});

export const compressSchema = z.object({
  mode: z.enum(['lossy', 'lossless']),
  quality: qualitySchema.optional(),
  format: formatSchema.optional(),
});

// Validation functions with structured errors
export function validate<T>(schema: z.ZodSchema<T>, input: unknown): T {
  const result = schema.safeParse(input);
  if (!result.success) {
    throw AppError.validationError(
      'Validation failed',
      result.error.errors.map(e => ({
        field: e.path.join('.'),
        message: e.message,
        code: e.code,
      }))
    );
  }
  return result.data;
}
```

### Simplified ImageService

```typescript
// src/services/image/image.service.ts
import sharp from 'sharp';
import type { 
  ResizeOptions, ConvertOptions, AdjustOptions, 
  RotateOptions, FlipOptions, WatermarkOptions, CompressOptions,
  ProcessedImage, ProcessedImageWithStats 
} from '@domain/types';

export class ImageService {
  async resize(input: Buffer, options: ResizeOptions): Promise<ProcessedImage> {
    const sharpInstance = sharp(input).resize({
      width: options.width,
      height: options.height,
      fit: options.maintainAspectRatio !== false ? (options.fit || 'inside') : 'fill',
      withoutEnlargement: false,
    });
    
    const buffer = await this.applyQuality(sharpInstance, options.quality).toBuffer();
    return this.buildResult(buffer);
  }

  async convert(input: Buffer, options: ConvertOptions): Promise<ProcessedImage> {
    let sharpInstance = sharp(input);
    const metadata = await sharpInstance.metadata();
    
    if (options.format === 'jpeg' && metadata.hasAlpha) {
      sharpInstance = sharpInstance.flatten({ 
        background: options.backgroundColor || '#ffffff' 
      });
    }
    
    const buffer = await sharpInstance
      .toFormat(options.format, { quality: options.quality || 80 })
      .toBuffer();
    return this.buildResult(buffer);
  }

  async adjust(input: Buffer, options: AdjustOptions): Promise<ProcessedImage> {
    let sharpInstance = sharp(input);
    
    if (options.brightness !== undefined && options.brightness !== 0) {
      sharpInstance = sharpInstance.modulate({ 
        brightness: 1 + (options.brightness / 100) 
      });
    }
    
    if (options.saturation !== undefined && options.saturation !== 0) {
      sharpInstance = sharpInstance.modulate({ 
        saturation: 1 + (options.saturation / 100) 
      });
    }
    
    if (options.contrast !== undefined && options.contrast !== 0) {
      const factor = 1 + (options.contrast / 100);
      sharpInstance = sharpInstance.linear(factor, -(128 * factor) + 128);
    }
    
    const buffer = await sharpInstance.toBuffer();
    return this.buildResult(buffer);
  }

  async rotate(input: Buffer, options: RotateOptions): Promise<ProcessedImage> {
    const buffer = await sharp(input)
      .rotate(options.angle, { background: options.backgroundColor })
      .toBuffer();
    return this.buildResult(buffer);
  }

  async flip(input: Buffer, options: FlipOptions): Promise<ProcessedImage> {
    let sharpInstance = sharp(input);
    if (options.vertical) sharpInstance = sharpInstance.flip();
    if (options.horizontal) sharpInstance = sharpInstance.flop();
    const buffer = await sharpInstance.toBuffer();
    return this.buildResult(buffer);
  }

  async watermark(input: Buffer, options: WatermarkOptions): Promise<ProcessedImage> {
    // Implementation unchanged - pure image processing
  }

  async compress(input: Buffer, options: CompressOptions): Promise<ProcessedImageWithStats> {
    // Implementation unchanged - pure image processing
  }

  async getMetadata(input: Buffer): Promise<ImageMetadata> {
    const metadata = await sharp(input).metadata();
    return {
      width: metadata.width || 0,
      height: metadata.height || 0,
      format: metadata.format || 'unknown',
      size: input.length,
      hasAlpha: metadata.hasAlpha || false,
    };
  }

  private async buildResult(buffer: Buffer): Promise<ProcessedImage> {
    const metadata = await sharp(buffer).metadata();
    return {
      buffer,
      metadata: {
        width: metadata.width || 0,
        height: metadata.height || 0,
        format: metadata.format || 'unknown',
        size: buffer.length,
        hasAlpha: metadata.hasAlpha || false,
      },
    };
  }

  private applyQuality(instance: sharp.Sharp, quality?: number): sharp.Sharp {
    if (!quality) return instance;
    return instance.jpeg({ quality }).png({ quality }).webp({ quality });
  }
}

export const imageService = new ImageService();
```

### OpenTelemetry Integration

```typescript
// src/infrastructure/observability/tracing.ts
import { NodeSDK } from '@opentelemetry/sdk-node';
import { getNodeAutoInstrumentations } from '@opentelemetry/auto-instrumentations-node';
import { OTLPTraceExporter } from '@opentelemetry/exporter-trace-otlp-http';
import { Resource } from '@opentelemetry/resources';
import { ATTR_SERVICE_NAME, ATTR_SERVICE_VERSION } from '@opentelemetry/semantic-conventions';
import { W3CTraceContextPropagator } from '@opentelemetry/core';
import { config } from '@config/index';

let sdk: NodeSDK | null = null;

export function initTracing(): void {
  if (sdk) return;
  
  const exporter = new OTLPTraceExporter({
    url: config.tracing.endpoint,
  });
  
  sdk = new NodeSDK({
    resource: new Resource({
      [ATTR_SERVICE_NAME]: 'image-processing-service',
      [ATTR_SERVICE_VERSION]: process.env.npm_package_version || '1.0.0',
    }),
    traceExporter: exporter,
    textMapPropagator: new W3CTraceContextPropagator(),
    instrumentations: [getNodeAutoInstrumentations()],
  });
  
  sdk.start();
}

export function shutdownTracing(): Promise<void> {
  return sdk?.shutdown() ?? Promise.resolve();
}
```

## Data Models

### Operation Types (Discriminated Union)

```typescript
// src/domain/types/operations.ts
export type ImageOperation =
  | { type: 'resize'; options: ResizeOptions }
  | { type: 'convert'; options: ConvertOptions }
  | { type: 'adjust'; options: AdjustOptions }
  | { type: 'rotate'; options: RotateOptions }
  | { type: 'flip'; options: FlipOptions }
  | { type: 'watermark'; options: WatermarkOptions }
  | { type: 'compress'; options: CompressOptions };

export type OperationType = ImageOperation['type'];
```

### Response Types

```typescript
// src/domain/types/responses.ts
export interface ImageMetadata {
  width: number;
  height: number;
  format: string;
  size: number;
  hasAlpha: boolean;
}

export interface ProcessedImage {
  buffer: Buffer;
  metadata: ImageMetadata;
}

export interface CompressionStats {
  originalSize: number;
  newSize: number;
  ratio: number;
  savedBytes: number;
  savedPercent: number;
}

export interface ProcessedImageWithStats extends ProcessedImage {
  compressionStats: CompressionStats;
}

export interface ApiResponse<T = unknown> {
  success: boolean;
  requestId: string;
  data?: T;
  error?: ErrorDetails;
}
```

## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system—essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*

### Property 1: Log Entries Contain Required Context

*For any* log entry generated by the service, the entry SHALL contain both a correlation ID (requestId) and trace context (traceId, spanId) when available.

**Validates: Requirements 1.2, 1.4**

### Property 2: Cache Keys Use Namespaced Prefix

*For any* cache operation performed by the service, the cache key SHALL start with the prefix "img:" to ensure namespace isolation.

**Validates: Requirements 2.2**

### Property 3: Cache Round-Trip Preserves Image Metadata

*For any* ProcessedImage cached and then retrieved, the retrieved image SHALL have identical metadata (width, height, format, size, hasAlpha) to the original.

**Validates: Requirements 2.5**

### Property 4: Validation Errors Contain Field-Level Details

*For any* validation failure, the error response SHALL contain structured field-level details including the field path, error message, and error code.

**Validates: Requirements 3.4**

### Property 5: Error Serialization Is Consistent

*For any* AppError instance, serializing to JSON and deserializing SHALL produce an equivalent error with the same code, message, and HTTP status.

**Validates: Requirements 4.3**

### Property 6: API Responses Include Required Headers

*For any* API response, the response SHALL include the X-Request-Id header. For image responses, the response SHALL also include Content-Type, X-Image-Width, X-Image-Height, and X-Image-Size headers.

**Validates: Requirements 5.3, 5.4**

### Property 7: ImageService Returns Accurate Metadata

*For any* image processed by ImageService, the returned metadata SHALL accurately reflect the actual dimensions, format, and size of the output buffer.

**Validates: Requirements 6.5**

### Property 8: Health Checks Include Latency Metrics

*For any* health check response, each dependency check SHALL include a latencyMs field indicating the time taken to verify connectivity.

**Validates: Requirements 9.3**

### Property 9: Trace Context Propagation and Metrics Recording

*For any* HTTP request processed by the service, the response SHALL include W3C Trace Context headers (traceparent), and processing duration metrics SHALL be recorded.

**Validates: Requirements 10.2, 10.3**

### Property 10: Serialization Round-Trip

*For any* valid ProcessedImage, serializing to cache format and deserializing SHALL produce an equivalent ProcessedImage with identical buffer content and metadata.

**Validates: Requirements 11.2**

### Property 11: Flip Idempotence

*For any* image, applying a horizontal flip twice SHALL produce an image identical to the original. The same applies to vertical flip.

**Validates: Requirements 11.3**

### Property 12: Resize Dimension Accuracy

*For any* resize operation with maintainAspectRatio=false and fit='fill', the output image dimensions SHALL exactly match the requested width and height.

**Validates: Requirements 11.4**

## Error Handling

### Centralized Error Factory

```typescript
// src/domain/errors/app-error.ts
export class AppError extends Error {
  constructor(
    public readonly code: ErrorCode,
    message: string,
    public readonly details?: Record<string, unknown>,
    public readonly httpStatus: number = ERROR_HTTP_STATUS[code]
  ) {
    super(message);
    this.name = 'AppError';
    Error.captureStackTrace(this, this.constructor);
  }

  toJSON(): ErrorDetails {
    return {
      code: this.code,
      message: this.message,
      details: this.details,
    };
  }

  // Factory methods for all error types
  static validationError(message: string, fields: FieldError[]): AppError {
    return new AppError(ErrorCode.VALIDATION_ERROR, message, { fields });
  }

  static invalidDimensions(message: string, details?: Record<string, unknown>): AppError {
    return new AppError(ErrorCode.INVALID_DIMENSIONS, message, details);
  }

  // ... other factory methods
}
```

### Global Error Handler

```typescript
// src/infrastructure/server.ts
server.setErrorHandler(async (error, request, reply) => {
  const requestId = request.requestId;
  const traceContext = getTraceContext(request);
  
  if (error instanceof AppError) {
    logger.warn('Application error', { 
      requestId, 
      ...traceContext,
      error: error.toJSON() 
    });
    return reply.status(error.httpStatus).send({
      success: false,
      requestId,
      error: error.toJSON(),
    });
  }

  // Log unexpected errors with full context
  logger.error('Unexpected error', error, { 
    requestId, 
    ...traceContext,
    url: request.url,
    method: request.method,
  });

  // Return sanitized response
  return reply.status(500).send({
    success: false,
    requestId,
    error: {
      code: 'INTERNAL_ERROR',
      message: 'An unexpected error occurred',
    },
  });
});
```

## Testing Strategy

### Dual Testing Approach

The test suite uses both unit tests and property-based tests:

- **Unit tests**: Verify specific examples, edge cases, and error conditions
- **Property tests**: Verify universal properties across all valid inputs using fast-check

### Property-Based Testing Configuration

- **Library**: fast-check v3.22.0+
- **Minimum iterations**: 100 per property test
- **Tag format**: `Feature: image-processing-modernization-2025, Property N: [property text]`

### Test File Structure

```
tests/
├── unit/
│   ├── services/
│   │   └── image/
│   │       ├── image.service.spec.ts      # Unit tests
│   │       └── image.property.spec.ts     # Property tests
│   ├── validators/
│   │   └── validation.property.spec.ts
│   └── infrastructure/
│       ├── cache.property.spec.ts
│       └── logging.property.spec.ts
├── integration/
│   └── api/
│       └── endpoints.spec.ts
└── setup.ts
```

### Example Property Test

```typescript
// Feature: image-processing-modernization-2025, Property 12: Resize Dimension Accuracy
describe('Property 12: Resize Dimension Accuracy', () => {
  it('should resize to exact dimensions when maintainAspectRatio is false', async () => {
    await fc.assert(
      fc.asyncProperty(
        fc.integer({ min: 10, max: 500 }),
        fc.integer({ min: 10, max: 500 }),
        async (targetWidth, targetHeight) => {
          const input = await generateTestImage(100, 100);
          const result = await imageService.resize(input, {
            width: targetWidth,
            height: targetHeight,
            maintainAspectRatio: false,
            fit: 'fill',
          });

          expect(result.metadata.width).toBe(targetWidth);
          expect(result.metadata.height).toBe(targetHeight);
        }
      ),
      { numRuns: 100 }
    );
  });
});
```

### Coverage Requirements

- Minimum 80% code coverage across all modules
- 100% coverage for error handling paths
- Property tests for all image operations
- Integration tests for all API endpoints
