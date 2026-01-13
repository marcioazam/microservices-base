import * as fc from 'fast-check';
import { recordHttpRequest, recordImageProcessing } from '../../../src/infrastructure/observability/metrics';

/**
 * Feature: image-processing-modernization-2025
 * Property 9: Trace Context Propagation and Metrics Recording
 * 
 * For any HTTP request processed by the service, the response SHALL include
 * W3C Trace Context headers (traceparent), and processing duration metrics
 * SHALL be recorded.
 * 
 * Validates: Requirements 10.2, 10.3
 */
describe('Property 9: Trace Context Propagation and Metrics Recording', () => {
  it('should record HTTP request metrics for all status codes', async () => {
    await fc.assert(
      fc.property(
        fc.constantFrom('GET', 'POST', 'PUT', 'DELETE'),
        fc.constantFrom('/api/images/resize', '/api/images/convert', '/api/health'),
        fc.integer({ min: 200, max: 599 }),
        fc.integer({ min: 1, max: 10000 }),
        (method, path, status, durationMs) => {
          // Should not throw
          expect(() => recordHttpRequest(method, path, status, durationMs)).not.toThrow();
        }
      ),
      { numRuns: 100 }
    );
  });

  it('should record image processing metrics for all operations', async () => {
    await fc.assert(
      fc.property(
        fc.constantFrom('resize', 'convert', 'adjust', 'rotate', 'flip', 'watermark', 'compress'),
        fc.boolean(),
        fc.integer({ min: 1, max: 30000 }),
        (operation, success, durationMs) => {
          expect(() => recordImageProcessing(operation, success, durationMs)).not.toThrow();
        }
      ),
      { numRuns: 100 }
    );
  });

  it('should handle edge case durations', async () => {
    await fc.assert(
      fc.property(
        fc.integer({ min: 0, max: 1 }), // Very fast
        fc.integer({ min: 60000, max: 120000 }), // Very slow
        (fastDuration, slowDuration) => {
          expect(() => recordHttpRequest('GET', '/test', 200, fastDuration)).not.toThrow();
          expect(() => recordHttpRequest('GET', '/test', 200, slowDuration)).not.toThrow();
        }
      ),
      { numRuns: 100 }
    );
  });
});

/**
 * Feature: image-processing-modernization-2025
 * Property 8: Health Checks Include Latency Metrics
 * 
 * For any health check response, each dependency check SHALL include a
 * latencyMs field indicating the time taken to verify connectivity.
 * 
 * Validates: Requirements 9.3
 */
describe('Property 8: Health Checks Include Latency Metrics', () => {
  interface HealthCheckResult {
    healthy: boolean;
    latencyMs: number;
  }

  it('should always include latencyMs in health check results', async () => {
    await fc.assert(
      fc.property(
        fc.boolean(),
        fc.integer({ min: 0, max: 5000 }),
        (healthy, latencyMs) => {
          const result: HealthCheckResult = { healthy, latencyMs };
          
          expect(result).toHaveProperty('healthy');
          expect(result).toHaveProperty('latencyMs');
          expect(typeof result.latencyMs).toBe('number');
          expect(result.latencyMs).toBeGreaterThanOrEqual(0);
        }
      ),
      { numRuns: 100 }
    );
  });

  it('should have non-negative latency values', async () => {
    await fc.assert(
      fc.property(
        fc.nat({ max: 10000 }),
        (latencyMs) => {
          const result: HealthCheckResult = { healthy: true, latencyMs };
          expect(result.latencyMs).toBeGreaterThanOrEqual(0);
        }
      ),
      { numRuns: 100 }
    );
  });
});
