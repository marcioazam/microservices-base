import * as fc from 'fast-check';
import { ImageService } from '../../../../src/services/image/image.service';
import {
  generateTestImage,
  generateTransparentImage,
  getImageDimensions,
  getImageFormat,
  pixelsEqual,
  calculateAspectRatio,
} from './test-helpers';

describe('ImageService Property Tests', () => {
  let imageService: ImageService;

  beforeAll(() => {
    imageService = new ImageService();
  });

  // Feature: image-processing-service, Property 1: Resize Dimensions Accuracy
  // Validates: Requirements 1.1
  describe('Property 1: Resize Dimensions Accuracy', () => {
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

            const dimensions = await getImageDimensions(result.buffer);
            expect(dimensions.width).toBe(targetWidth);
            expect(dimensions.height).toBe(targetHeight);
            expect(result.metadata.width).toBe(targetWidth);
            expect(result.metadata.height).toBe(targetHeight);
          }
        ),
        { numRuns: 100 }
      );
    });
  });

  // Feature: image-processing-service, Property 2: Aspect Ratio Preservation
  // Validates: Requirements 1.2
  describe('Property 2: Aspect Ratio Preservation', () => {
    it('should preserve aspect ratio when maintainAspectRatio is true', async () => {
      await fc.assert(
        fc.asyncProperty(
          fc.integer({ min: 50, max: 300 }),
          fc.integer({ min: 50, max: 300 }),
          fc.integer({ min: 50, max: 500 }),
          async (inputWidth, inputHeight, targetSize) => {
            const input = await generateTestImage(inputWidth, inputHeight);
            const originalRatio = calculateAspectRatio(inputWidth, inputHeight);

            const result = await imageService.resize(input, {
              width: targetSize,
              height: targetSize,
              maintainAspectRatio: true,
              fit: 'inside',
            });

            const dimensions = await getImageDimensions(result.buffer);
            const newRatio = calculateAspectRatio(dimensions.width, dimensions.height);

            // Allow 1% tolerance for aspect ratio
            expect(Math.abs(originalRatio - newRatio)).toBeLessThan(0.01);
          }
        ),
        { numRuns: 100 }
      );
    });
  });

  // Feature: image-processing-service, Property 3: Quality-Size Relationship
  // Validates: Requirements 1.3, 2.2
  describe('Property 3: Quality-Size Relationship', () => {
    it('should produce smaller files with lower quality', async () => {
      await fc.assert(
        fc.asyncProperty(
          fc.integer({ min: 10, max: 50 }),
          fc.integer({ min: 60, max: 100 }),
          async (lowQuality, highQuality) => {
            fc.pre(lowQuality < highQuality);

            const input = await generateTestImage(200, 200, 'jpeg', false);

            const lowQualityResult = await imageService.convert(input, {
              format: 'jpeg',
              quality: lowQuality,
            });

            const highQualityResult = await imageService.convert(input, {
              format: 'jpeg',
              quality: highQuality,
            });

            expect(lowQualityResult.metadata.size).toBeLessThanOrEqual(
              highQualityResult.metadata.size
            );
          }
        ),
        { numRuns: 100 }
      );
    });
  });

  // Feature: image-processing-service, Property 7: Format Conversion Correctness
  // Validates: Requirements 2.1
  describe('Property 7: Format Conversion Correctness', () => {
    it('should convert to the specified format', async () => {
      const formats = ['jpeg', 'png', 'webp'] as const;

      await fc.assert(
        fc.asyncProperty(
          fc.constantFrom(...formats),
          async (targetFormat) => {
            const input = await generateTestImage(100, 100, 'png', false);

            const result = await imageService.convert(input, {
              format: targetFormat,
              quality: 80,
            });

            const actualFormat = await getImageFormat(result.buffer);
            expect(actualFormat).toBe(targetFormat);
            expect(result.metadata.format).toBe(targetFormat);
          }
        ),
        { numRuns: 100 }
      );
    });
  });

  // Feature: image-processing-service, Property 8: Transparency Background Handling
  // Validates: Requirements 2.3
  describe('Property 8: Transparency Background Handling', () => {
    it('should apply background color when converting transparent to JPEG', async () => {
      const input = await generateTransparentImage(100, 100);

      const result = await imageService.convert(input, {
        format: 'jpeg',
        backgroundColor: '#ffffff',
      });

      const format = await getImageFormat(result.buffer);
      expect(format).toBe('jpeg');
      expect(result.metadata.hasAlpha).toBe(false);
    });
  });

  // Feature: image-processing-service, Property 10: Brightness Adjustment Identity
  // Validates: Requirements 3.1
  describe('Property 10: Brightness Adjustment Identity', () => {
    it('should produce identical output when brightness is 0', async () => {
      const input = await generateTestImage(50, 50, 'png');

      const result = await imageService.adjust(input, { brightness: 0 });

      const areEqual = await pixelsEqual(input, result.buffer, 1);
      expect(areEqual).toBe(true);
    });
  });

  // Feature: image-processing-service, Property 11: Contrast Adjustment Identity
  // Validates: Requirements 3.2
  describe('Property 11: Contrast Adjustment Identity', () => {
    it('should produce identical output when contrast is 0', async () => {
      const input = await generateTestImage(50, 50, 'png');

      const result = await imageService.adjust(input, { contrast: 0 });

      const areEqual = await pixelsEqual(input, result.buffer, 1);
      expect(areEqual).toBe(true);
    });
  });

  // Feature: image-processing-service, Property 14: Rotation Identity
  // Validates: Requirements 4.1
  describe('Property 14: Rotation Identity', () => {
    it('should produce identical output when rotating by 0 or 360 degrees', async () => {
      await fc.assert(
        fc.asyncProperty(fc.constantFrom(0, 360), async (angle) => {
          const input = await generateTestImage(50, 50, 'png');

          const result = await imageService.rotate(input, { angle });

          const dimensions = await getImageDimensions(result.buffer);
          const inputDimensions = await getImageDimensions(input);

          expect(dimensions.width).toBe(inputDimensions.width);
          expect(dimensions.height).toBe(inputDimensions.height);
        }),
        { numRuns: 100 }
      );
    });
  });

  // Feature: image-processing-service, Property 15: Flip Idempotence
  // Validates: Requirements 4.2, 4.3
  describe('Property 15: Flip Idempotence', () => {
    it('should return to original after double horizontal flip', async () => {
      const input = await generateTestImage(50, 50, 'png');

      const once = await imageService.flip(input, { horizontal: true });
      const twice = await imageService.flip(once.buffer, { horizontal: true });

      const areEqual = await pixelsEqual(input, twice.buffer, 0);
      expect(areEqual).toBe(true);
    });

    it('should return to original after double vertical flip', async () => {
      const input = await generateTestImage(50, 50, 'png');

      const once = await imageService.flip(input, { vertical: true });
      const twice = await imageService.flip(once.buffer, { vertical: true });

      const areEqual = await pixelsEqual(input, twice.buffer, 0);
      expect(areEqual).toBe(true);
    });
  });
});
