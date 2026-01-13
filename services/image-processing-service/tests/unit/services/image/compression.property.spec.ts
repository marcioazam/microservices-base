import * as fc from 'fast-check';
import { ImageService } from '../../../../src/services/image/image.service';
import { generateTestImage } from './test-helpers';

describe('ImageService Compression Property Tests', () => {
  let imageService: ImageService;

  beforeAll(() => {
    imageService = new ImageService();
  });

  // Feature: image-processing-service, Property 18: Lossy Compression Effectiveness
  // Validates: Requirements 6.1
  describe('Property 18: Lossy Compression Effectiveness', () => {
    it('should produce smaller or equal file size with lossy compression', async () => {
      await fc.assert(
        fc.asyncProperty(
          fc.integer({ min: 10, max: 90 }),
          async (quality) => {
            const input = await generateTestImage(200, 200, 'jpeg', false);
            const originalSize = input.length;

            const result = await imageService.compress(input, {
              mode: 'lossy',
              quality,
              format: 'jpeg',
            });

            expect(result.metadata.size).toBeLessThanOrEqual(originalSize);
            expect(result.compressionStats).toBeDefined();
            expect(result.compressionStats!.originalSize).toBe(originalSize);
          }
        ),
        { numRuns: 100 }
      );
    });
  });

  // Feature: image-processing-service, Property 19: Lossless Compression Integrity
  // Validates: Requirements 6.2
  describe('Property 19: Lossless Compression Integrity', () => {
    it('should maintain image dimensions with lossless compression', async () => {
      const input = await generateTestImage(100, 100, 'png');

      const result = await imageService.compress(input, {
        mode: 'lossless',
        format: 'png',
      });

      expect(result.metadata.width).toBe(100);
      expect(result.metadata.height).toBe(100);
    });
  });

  // Feature: image-processing-service, Property 20: Compression Statistics Accuracy
  // Validates: Requirements 6.3
  describe('Property 20: Compression Statistics Accuracy', () => {
    it('should return accurate compression statistics', async () => {
      await fc.assert(
        fc.asyncProperty(
          fc.integer({ min: 20, max: 80 }),
          async (quality) => {
            const input = await generateTestImage(150, 150, 'jpeg', false);

            const result = await imageService.compress(input, {
              mode: 'lossy',
              quality,
              format: 'jpeg',
            });

            const stats = result.compressionStats!;

            // Verify ratio calculation
            const expectedRatio = stats.originalSize / stats.newSize;
            expect(Math.abs(stats.ratio - expectedRatio)).toBeLessThan(0.01);

            // Verify saved bytes
            expect(stats.savedBytes).toBe(stats.originalSize - stats.newSize);

            // Verify saved percent
            const expectedPercent = ((stats.originalSize - stats.newSize) / stats.originalSize) * 100;
            expect(Math.abs(stats.savedPercent - expectedPercent)).toBeLessThan(0.01);
          }
        ),
        { numRuns: 100 }
      );
    });
  });
});
