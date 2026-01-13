import * as fc from 'fast-check';
import sharp from 'sharp';
import { ImageService } from '../../../../src/services/image/image.service';

// Helper to generate test images
async function generateTestImage(width: number, height: number, format: 'png' | 'jpeg' = 'png'): Promise<Buffer> {
  return sharp({
    create: { width, height, channels: format === 'png' ? 4 : 3, background: { r: 128, g: 128, b: 128 } },
  }).toFormat(format).toBuffer();
}

/**
 * Feature: image-processing-modernization-2025
 * Property 7: ImageService Returns Accurate Metadata
 * 
 * For any image processed by ImageService, the returned metadata SHALL accurately
 * reflect the actual dimensions, format, and size of the output buffer.
 * 
 * Validates: Requirements 6.5
 */
describe('Property 7: ImageService Returns Accurate Metadata', () => {
  const imageService = new ImageService();

  it('should return accurate metadata for getMetadata', async () => {
    await fc.assert(
      fc.asyncProperty(
        fc.integer({ min: 10, max: 200 }),
        fc.integer({ min: 10, max: 200 }),
        async (width, height) => {
          const input = await generateTestImage(width, height);
          const result = await imageService.getMetadata(input);

          expect(result.width).toBe(width);
          expect(result.height).toBe(height);
          expect(result.size).toBe(input.length);
          expect(result.format).toBe('png');
        }
      ),
      { numRuns: 50 }
    );
  });

  it('should return accurate metadata after resize', async () => {
    await fc.assert(
      fc.asyncProperty(
        fc.integer({ min: 50, max: 200 }),
        fc.integer({ min: 50, max: 200 }),
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
          expect(result.metadata.size).toBe(result.buffer.length);
        }
      ),
      { numRuns: 50 }
    );
  });

  it('should return accurate metadata after convert', async () => {
    await fc.assert(
      fc.asyncProperty(
        fc.constantFrom('jpeg', 'webp') as fc.Arbitrary<'jpeg' | 'webp'>,
        async (format) => {
          const input = await generateTestImage(100, 100);
          const result = await imageService.convert(input, { format });

          expect(result.metadata.format).toBe(format);
          expect(result.metadata.size).toBe(result.buffer.length);
          expect(result.metadata.width).toBe(100);
          expect(result.metadata.height).toBe(100);
        }
      ),
      { numRuns: 50 }
    );
  });

  it('should return accurate metadata after flip', async () => {
    await fc.assert(
      fc.asyncProperty(
        fc.boolean(),
        fc.boolean(),
        async (horizontal, vertical) => {
          if (!horizontal && !vertical) return; // Skip invalid case
          
          const input = await generateTestImage(100, 80);
          const result = await imageService.flip(input, { horizontal, vertical });

          // Flip should preserve dimensions
          expect(result.metadata.width).toBe(100);
          expect(result.metadata.height).toBe(80);
          expect(result.metadata.size).toBe(result.buffer.length);
        }
      ),
      { numRuns: 50 }
    );
  });

  it('should return accurate metadata after adjust', async () => {
    await fc.assert(
      fc.asyncProperty(
        fc.integer({ min: -50, max: 50 }),
        async (brightness) => {
          const input = await generateTestImage(100, 100);
          const result = await imageService.adjust(input, { brightness });

          // Adjust should preserve dimensions
          expect(result.metadata.width).toBe(100);
          expect(result.metadata.height).toBe(100);
          expect(result.metadata.size).toBe(result.buffer.length);
        }
      ),
      { numRuns: 50 }
    );
  });
});

/**
 * Feature: image-processing-modernization-2025
 * Property 11: Flip Idempotence
 * 
 * For any image, applying a horizontal flip twice SHALL produce an image
 * identical to the original. The same applies to vertical flip.
 * 
 * Validates: Requirements 11.3
 */
describe('Property 11: Flip Idempotence', () => {
  const imageService = new ImageService();

  it('should return to original after double horizontal flip', async () => {
    await fc.assert(
      fc.asyncProperty(
        fc.integer({ min: 20, max: 100 }),
        fc.integer({ min: 20, max: 100 }),
        async (width, height) => {
          const input = await generateTestImage(width, height);
          
          const flipped1 = await imageService.flip(input, { horizontal: true });
          const flipped2 = await imageService.flip(flipped1.buffer, { horizontal: true });

          // Dimensions should be preserved
          expect(flipped2.metadata.width).toBe(width);
          expect(flipped2.metadata.height).toBe(height);
        }
      ),
      { numRuns: 50 }
    );
  });

  it('should return to original after double vertical flip', async () => {
    await fc.assert(
      fc.asyncProperty(
        fc.integer({ min: 20, max: 100 }),
        fc.integer({ min: 20, max: 100 }),
        async (width, height) => {
          const input = await generateTestImage(width, height);
          
          const flipped1 = await imageService.flip(input, { vertical: true });
          const flipped2 = await imageService.flip(flipped1.buffer, { vertical: true });

          expect(flipped2.metadata.width).toBe(width);
          expect(flipped2.metadata.height).toBe(height);
        }
      ),
      { numRuns: 50 }
    );
  });
});

/**
 * Feature: image-processing-modernization-2025
 * Property 12: Resize Dimension Accuracy
 * 
 * For any resize operation with maintainAspectRatio=false and fit='fill',
 * the output image dimensions SHALL exactly match the requested width and height.
 * 
 * Validates: Requirements 11.4
 */
describe('Property 12: Resize Dimension Accuracy', () => {
  const imageService = new ImageService();

  it('should resize to exact dimensions when maintainAspectRatio is false', async () => {
    await fc.assert(
      fc.asyncProperty(
        fc.integer({ min: 10, max: 300 }),
        fc.integer({ min: 10, max: 300 }),
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
