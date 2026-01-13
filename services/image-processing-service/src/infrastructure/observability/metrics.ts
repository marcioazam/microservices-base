import { Counter, Histogram, Registry } from 'prom-client';

const register = new Registry();

// HTTP request metrics
export const httpRequestsTotal = new Counter({
  name: 'http_requests_total',
  help: 'Total number of HTTP requests',
  labelNames: ['method', 'path', 'status'],
  registers: [register],
});

export const httpRequestDuration = new Histogram({
  name: 'http_request_duration_seconds',
  help: 'HTTP request duration in seconds',
  labelNames: ['method', 'path', 'status'],
  buckets: [0.01, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10],
  registers: [register],
});

// Image processing metrics
export const imageProcessingTotal = new Counter({
  name: 'image_processing_total',
  help: 'Total number of image processing operations',
  labelNames: ['operation', 'status'],
  registers: [register],
});

export const imageProcessingDuration = new Histogram({
  name: 'image_processing_duration_seconds',
  help: 'Image processing duration in seconds',
  labelNames: ['operation'],
  buckets: [0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30],
  registers: [register],
});

export const imageProcessingSize = new Histogram({
  name: 'image_processing_size_bytes',
  help: 'Size of processed images in bytes',
  labelNames: ['operation', 'type'],
  buckets: [1024, 10240, 102400, 1048576, 10485760, 52428800],
  registers: [register],
});

// Cache metrics
export const cacheOperationsTotal = new Counter({
  name: 'cache_operations_total',
  help: 'Total number of cache operations',
  labelNames: ['operation', 'result'],
  registers: [register],
});

// Job queue metrics
export const jobQueueSize = new Counter({
  name: 'job_queue_size_total',
  help: 'Total number of jobs added to queue',
  labelNames: ['status'],
  registers: [register],
});

export function recordHttpRequest(method: string, path: string, status: number, durationMs: number): void {
  httpRequestsTotal.inc({ method, path, status: String(status) });
  httpRequestDuration.observe({ method, path, status: String(status) }, durationMs / 1000);
}

export function recordImageProcessing(operation: string, success: boolean, durationMs: number): void {
  imageProcessingTotal.inc({ operation, status: success ? 'success' : 'error' });
  imageProcessingDuration.observe({ operation }, durationMs / 1000);
}

export function recordImageSize(operation: string, inputSize: number, outputSize: number): void {
  imageProcessingSize.observe({ operation, type: 'input' }, inputSize);
  imageProcessingSize.observe({ operation, type: 'output' }, outputSize);
}

export function recordCacheOperation(operation: 'get' | 'set' | 'delete', hit: boolean): void {
  cacheOperationsTotal.inc({ operation, result: hit ? 'hit' : 'miss' });
}

export async function getMetrics(): Promise<string> {
  return register.metrics();
}

export function getContentType(): string {
  return register.contentType;
}

export { register };
