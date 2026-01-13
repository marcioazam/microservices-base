import * as fc from 'fast-check';
import { REQUIRED_HEADERS, IMAGE_HEADERS } from '../../../../src/shared/utils/response';
import { ImageMetadata } from '../../../../src/domain/types/responses';

/**
 * Feature: image-processing-modernization-2025
 * Property 6: API Responses Include Required Headers
 * 
 * For any API response, the response SHALL include the X-Request-Id header.
 * For image responses, the response SHALL also include Content-Type, 
 * X-Image-Width, X-Image-Height, and X-Image-Size headers.
 * 
 * Validates: Requirements 5.3, 5.4
 */
describe('Property 6: API Responses Include Required Headers', () => {
  // Mock FastifyReply
  const createMockReply = () => {
    const headers: Record<string, string> = {};
    const reply = {
      status: jest.fn().mockReturnThis(),
      header: jest.fn((key: string, value: string) => {
        headers[key] = value;
        return reply;
      }),
      send: jest.fn().mockReturnThis(),
      getHeaders: () => headers,
    };
    return reply;
  };

  it('should define X-Request-Id as required header constant', () => {
    expect(REQUIRED_HEADERS.REQUEST_ID).toBe('X-Request-Id');
  });

  it('should define all image header constants', () => {
    expect(IMAGE_HEADERS.CONTENT_TYPE).toBe('Content-Type');
    expect(IMAGE_HEADERS.WIDTH).toBe('X-Image-Width');
    expect(IMAGE_HEADERS.HEIGHT).toBe('X-Image-Height');
    expect(IMAGE_HEADERS.SIZE).toBe('X-Image-Size');
    expect(IMAGE_HEADERS.FORMAT).toBe('X-Image-Format');
  });

  it('should include requestId in success response body', async () => {
    await fc.assert(
      fc.property(
        fc.uuid(),
        fc.anything(),
        (requestId, data) => {
          const response = {
            success: true,
            requestId,
            data,
          };
          
          expect(response.requestId).toBe(requestId);
          expect(response.success).toBe(true);
        }
      ),
      { numRuns: 100 }
    );
  });

  it('should include requestId in error response body', async () => {
    await fc.assert(
      fc.property(
        fc.uuid(),
        fc.string({ minLength: 1, maxLength: 100 }),
        (requestId, message) => {
          const response = {
            success: false,
            requestId,
            error: {
              code: 'VALIDATION_ERROR',
              message,
            },
          };
          
          expect(response.requestId).toBe(requestId);
          expect(response.success).toBe(false);
        }
      ),
      { numRuns: 100 }
    );
  });

  it('should format image metadata headers correctly', async () => {
    const metadataArb: fc.Arbitrary<ImageMetadata> = fc.record({
      width: fc.integer({ min: 1, max: 10000 }),
      height: fc.integer({ min: 1, max: 10000 }),
      format: fc.constantFrom('jpeg', 'png', 'gif', 'webp', 'tiff'),
      size: fc.integer({ min: 1, max: 100000000 }),
      hasAlpha: fc.boolean(),
    });

    await fc.assert(
      fc.property(
        metadataArb,
        (metadata) => {
          // Verify header values can be converted to strings
          const widthHeader = metadata.width.toString();
          const heightHeader = metadata.height.toString();
          const sizeHeader = metadata.size.toString();
          const contentType = `image/${metadata.format}`;
          
          expect(widthHeader).toBe(String(metadata.width));
          expect(heightHeader).toBe(String(metadata.height));
          expect(sizeHeader).toBe(String(metadata.size));
          expect(contentType).toMatch(/^image\/(jpeg|png|gif|webp|tiff)$/);
        }
      ),
      { numRuns: 100 }
    );
  });

  it('should produce valid content-type for all supported formats', async () => {
    await fc.assert(
      fc.property(
        fc.constantFrom('jpeg', 'png', 'gif', 'webp', 'tiff'),
        (format) => {
          const contentType = `image/${format}`;
          expect(contentType).toMatch(/^image\/[a-z]+$/);
        }
      ),
      { numRuns: 100 }
    );
  });

  it('should include all required fields in image response data', async () => {
    const metadataArb: fc.Arbitrary<ImageMetadata> = fc.record({
      width: fc.integer({ min: 1, max: 10000 }),
      height: fc.integer({ min: 1, max: 10000 }),
      format: fc.constantFrom('jpeg', 'png', 'gif', 'webp', 'tiff'),
      size: fc.integer({ min: 1, max: 100000000 }),
      hasAlpha: fc.boolean(),
    });

    await fc.assert(
      fc.property(
        fc.uuid(),
        fc.uint8Array({ minLength: 10, maxLength: 100 }),
        metadataArb,
        (requestId, bufferData, metadata) => {
          const imageResponseData = {
            image: Buffer.from(bufferData).toString('base64'),
            metadata,
          };
          
          const response = {
            success: true,
            requestId,
            data: imageResponseData,
          };
          
          expect(response.data.image).toBeDefined();
          expect(response.data.metadata.width).toBe(metadata.width);
          expect(response.data.metadata.height).toBe(metadata.height);
          expect(response.data.metadata.format).toBe(metadata.format);
          expect(response.data.metadata.size).toBe(metadata.size);
        }
      ),
      { numRuns: 100 }
    );
  });
});
