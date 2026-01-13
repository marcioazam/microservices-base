import { ImageService } from '../../../../src/services/image/image.service';
import { generateTestImage, pixelsEqual } from './test-helpers';

describe('ImageService Watermark Property Tests', () => {
  let imageService: ImageService;

  beforeAll(() => {
    imageService = new ImageService();
  });

  // Feature: image-processing-service, Property 16: Watermark Presence
  // Validates: Requirements 5.1, 5.2
  describe('Property 16: Watermark Presence', () => {
    it('should produce different output when watermark is applied', async () => {
      const input = await generateTestImage(200, 200, 'png');

      const result = await imageService.watermark(input, {
        type: 'text',
        content: 'Test Watermark',
        position: 'center',
        opacity: 100,
        font: {
          size: 24,
          color: '#ff0000',
        },
      });

      // The output should be different from input
      const areEqual = await pixelsEqual(input, result.buffer, 0);
      expect(areEqual).toBe(false);
    });
  });

  // Feature: image-processing-service, Property 17: Watermark Opacity Zero
  // Validates: Requirements 5.3
  describe('Property 17: Watermark Opacity Zero', () => {
    it('should produce identical output when opacity is 0', async () => {
      const input = await generateTestImage(100, 100, 'png');

      const result = await imageService.watermark(input, {
        type: 'text',
        content: 'Test Watermark',
        position: 'center',
        opacity: 0,
      });

      const areEqual = await pixelsEqual(input, result.buffer, 0);
      expect(areEqual).toBe(true);
    });
  });
});
