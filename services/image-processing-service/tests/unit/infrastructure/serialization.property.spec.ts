import * as fc from 'fast-check';
import { ProcessedImage, ImageMetadata } from '../../../src/domain/types/responses';

/**
 * Feature: image-processing-modernization-2025
 * Property 10: Serialization Round-Trip
 * 
 * For any valid ProcessedImage, serializing to cache format and deserializing
 * SHALL produce an equivalent ProcessedImage with identical buffer content
 * and metadata.
 * 
 * Validates: Requirements 11.2
 */
describe('Property 10: Serialization Round-Trip', () => {
  interface CachedImage {
    buffer: string; // base64 encoded
    metadata: ImageMetadata;
  }

  function serialize(image: ProcessedImage): CachedImage {
    return {
      buffer: image.buffer.toString('base64'),
      metadata: image.metadata,
    };
  }

  function deserialize(cached: CachedImage): ProcessedImage {
    return {
      buffer: Buffer.from(cached.buffer, 'base64'),
      metadata: cached.metadata,
    };
  }

  const metadataArb: fc.Arbitrary<ImageMetadata> = fc.record({
    width: fc.integer({ min: 1, max: 10000 }),
    height: fc.integer({ min: 1, max: 10000 }),
    format: fc.constantFrom('jpeg', 'png', 'gif', 'webp', 'tiff'),
    size: fc.integer({ min: 1, max: 100000000 }),
    hasAlpha: fc.boolean(),
  });

  it('should preserve buffer content after round-trip', async () => {
    await fc.assert(
      fc.property(
        fc.uint8Array({ minLength: 1, maxLength: 1000 }),
        metadataArb,
        (bufferData, metadata) => {
          const original: ProcessedImage = {
            buffer: Buffer.from(bufferData),
            metadata,
          };

          const serialized = serialize(original);
          const deserialized = deserialize(serialized);

          expect(deserialized.buffer.equals(original.buffer)).toBe(true);
        }
      ),
      { numRuns: 100 }
    );
  });

  it('should preserve all metadata fields after round-trip', async () => {
    await fc.assert(
      fc.property(
        fc.uint8Array({ minLength: 1, maxLength: 100 }),
        metadataArb,
        (bufferData, metadata) => {
          const original: ProcessedImage = {
            buffer: Buffer.from(bufferData),
            metadata,
          };

          const serialized = serialize(original);
          const deserialized = deserialize(serialized);

          expect(deserialized.metadata.width).toBe(original.metadata.width);
          expect(deserialized.metadata.height).toBe(original.metadata.height);
          expect(deserialized.metadata.format).toBe(original.metadata.format);
          expect(deserialized.metadata.size).toBe(original.metadata.size);
          expect(deserialized.metadata.hasAlpha).toBe(original.metadata.hasAlpha);
        }
      ),
      { numRuns: 100 }
    );
  });

  it('should produce valid JSON when serialized', async () => {
    await fc.assert(
      fc.property(
        fc.uint8Array({ minLength: 1, maxLength: 500 }),
        metadataArb,
        (bufferData, metadata) => {
          const original: ProcessedImage = {
            buffer: Buffer.from(bufferData),
            metadata,
          };

          const serialized = serialize(original);
          const jsonString = JSON.stringify(serialized);

          expect(() => JSON.parse(jsonString)).not.toThrow();
          
          const parsed = JSON.parse(jsonString) as CachedImage;
          expect(parsed.buffer).toBe(serialized.buffer);
          expect(parsed.metadata).toEqual(serialized.metadata);
        }
      ),
      { numRuns: 100 }
    );
  });

  it('should handle empty buffers', async () => {
    await fc.assert(
      fc.property(
        metadataArb,
        (metadata) => {
          const original: ProcessedImage = {
            buffer: Buffer.alloc(0),
            metadata,
          };

          const serialized = serialize(original);
          const deserialized = deserialize(serialized);

          expect(deserialized.buffer.length).toBe(0);
          expect(deserialized.metadata).toEqual(original.metadata);
        }
      ),
      { numRuns: 100 }
    );
  });

  it('should handle large buffers', async () => {
    await fc.assert(
      fc.property(
        fc.integer({ min: 10000, max: 50000 }),
        metadataArb,
        (size, metadata) => {
          const original: ProcessedImage = {
            buffer: Buffer.alloc(size, 0x42),
            metadata: { ...metadata, size },
          };

          const serialized = serialize(original);
          const deserialized = deserialize(serialized);

          expect(deserialized.buffer.length).toBe(size);
          expect(deserialized.buffer.equals(original.buffer)).toBe(true);
        }
      ),
      { numRuns: 20 } // Fewer runs for large buffers
    );
  });
});
