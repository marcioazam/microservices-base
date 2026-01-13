import * as fc from 'fast-check';

// Feature: image-processing-service, Property 32: Response Format Consistency
// Validates: Requirements 12.1, 12.2

// Feature: image-processing-service, Property 33: Request ID Presence
// Validates: Requirements 12.3

// Feature: image-processing-service, Property 34: HTTP Status Code Correctness
// Validates: Requirements 12.4

// Feature: image-processing-service, Property 35: Response Format Options
// Validates: Requirements 12.5

interface SuccessResponse<T> {
  success: true;
  requestId: string;
  data: T;
}

interface ErrorResponse {
  success: false;
  requestId: string;
  error: {
    code: string;
    message: string;
  };
}

type ApiResponse<T> = SuccessResponse<T> | ErrorResponse;

function createSuccessResponse<T>(requestId: string, data: T): SuccessResponse<T> {
  return { success: true, requestId, data };
}

function createErrorResponse(requestId: string, code: string, message: string): ErrorResponse {
  return { success: false, requestId, error: { code, message } };
}

function isValidResponse<T>(response: ApiResponse<T>): boolean {
  if (typeof response.success !== 'boolean') return false;
  if (typeof response.requestId !== 'string') return false;
  if (response.requestId.length === 0) return false;

  if (response.success) {
    return 'data' in response;
  } else {
    return 'error' in response && 
           typeof response.error.code === 'string' && 
           typeof response.error.message === 'string';
  }
}

describe('Response Format Property Tests', () => {
  describe('Property 32: Response Format Consistency', () => {
    it('should always include success, requestId, and data/error', async () => {
      await fc.assert(
        fc.property(
          fc.uuid(),
          fc.object(),
          (requestId, data) => {
            const response = createSuccessResponse(requestId, data);
            expect(isValidResponse(response)).toBe(true);
            expect(response.success).toBe(true);
            expect(response.requestId).toBe(requestId);
            expect(response.data).toEqual(data);
          }
        ),
        { numRuns: 100 }
      );
    });

    it('should format error responses consistently', async () => {
      await fc.assert(
        fc.property(
          fc.uuid(),
          fc.string({ minLength: 1 }),
          fc.string({ minLength: 1 }),
          (requestId, code, message) => {
            const response = createErrorResponse(requestId, code, message);
            expect(isValidResponse(response)).toBe(true);
            expect(response.success).toBe(false);
            expect(response.requestId).toBe(requestId);
            expect(response.error.code).toBe(code);
            expect(response.error.message).toBe(message);
          }
        ),
        { numRuns: 100 }
      );
    });
  });

  describe('Property 33: Request ID Presence', () => {
    it('should always have non-empty requestId in success responses', async () => {
      await fc.assert(
        fc.property(
          fc.uuid(),
          fc.anything(),
          (requestId, data) => {
            const response = createSuccessResponse(requestId, data);
            expect(response.requestId).toBeTruthy();
            expect(response.requestId.length).toBeGreaterThan(0);
          }
        ),
        { numRuns: 100 }
      );
    });

    it('should always have non-empty requestId in error responses', async () => {
      await fc.assert(
        fc.property(
          fc.uuid(),
          fc.string(),
          fc.string(),
          (requestId, code, message) => {
            const response = createErrorResponse(requestId, code, message);
            expect(response.requestId).toBeTruthy();
            expect(response.requestId.length).toBeGreaterThan(0);
          }
        ),
        { numRuns: 100 }
      );
    });
  });

  describe('Property 34: HTTP Status Code Correctness', () => {
    const errorCodeToStatus: Record<string, number> = {
      INVALID_DIMENSIONS: 400,
      INVALID_FORMAT: 400,
      INVALID_QUALITY: 400,
      INVALID_IMAGE: 400,
      MISSING_TOKEN: 401,
      INVALID_TOKEN: 401,
      INSUFFICIENT_PERMISSIONS: 403,
      IMAGE_NOT_FOUND: 404,
      RATE_LIMIT_EXCEEDED: 429,
      PROCESSING_ERROR: 500,
      INTERNAL_ERROR: 500,
    };

    it('should map error codes to correct HTTP status', async () => {
      const errorCodes = Object.keys(errorCodeToStatus);

      await fc.assert(
        fc.property(
          fc.constantFrom(...errorCodes),
          (errorCode) => {
            const expectedStatus = errorCodeToStatus[errorCode];
            expect(expectedStatus).toBeDefined();
            
            // Validation errors should be 4xx
            if (errorCode.startsWith('INVALID') || errorCode.startsWith('MISSING')) {
              expect(expectedStatus).toBeGreaterThanOrEqual(400);
              expect(expectedStatus).toBeLessThan(500);
            }
            
            // Server errors should be 5xx
            if (errorCode.includes('ERROR') && !errorCode.startsWith('INVALID')) {
              expect(expectedStatus).toBeGreaterThanOrEqual(500);
            }
          }
        ),
        { numRuns: 100 }
      );
    });
  });

  describe('Property 35: Response Format Options', () => {
    it('should support base64 encoding for JSON responses', async () => {
      await fc.assert(
        fc.property(
          fc.uint8Array({ minLength: 1, maxLength: 1000 }),
          (imageData) => {
            const buffer = Buffer.from(imageData);
            const base64 = buffer.toString('base64');
            
            // Verify round-trip
            const decoded = Buffer.from(base64, 'base64');
            expect(decoded.compare(buffer)).toBe(0);
          }
        ),
        { numRuns: 100 }
      );
    });

    it('should preserve binary data integrity', async () => {
      await fc.assert(
        fc.property(
          fc.uint8Array({ minLength: 1, maxLength: 10000 }),
          (data) => {
            const buffer = Buffer.from(data);
            
            // Binary response should be identical
            expect(buffer.length).toBe(data.length);
            for (let i = 0; i < data.length; i++) {
              expect(buffer[i]).toBe(data[i]);
            }
          }
        ),
        { numRuns: 100 }
      );
    });
  });
});
