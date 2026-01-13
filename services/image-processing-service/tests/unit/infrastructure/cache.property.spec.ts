import * as fc from 'fast-check';
import { PlatformCacheClient } from '../../../src/infrastructure/cache/client';
import { ImageOperation } from '../../../src/domain/types/requests';
import { ProcessedImage, ImageMetadata } from '../../../src/domain/types/responses';

/**
 * Feature: image-processing-modernization-2025
 * Property 2: Cache Keys Use Namespaced Prefix
 * 
 * For any cache operation performed by the service, the cache key SHALL start
 * with the prefix "img:" to ensure namespace isolation.
 * 
 * Validates: Requirements 2.2
 */
describe('Property 2: Cache Keys Use Namespaced Prefix', () => {
  it('should generate keys with img: prefix for all operations', async () => {
    const operationArb = fc.oneof(
      fc.record({ type: fc.constant('resize' as const), options: fc.record({ width: fc.integer({ min: 1, max: 1000 }) }) }),
      fc.record({ type: fc.constant('convert' as const), options: fc.record({ format: fc.constantFrom('jpeg', 'png', 'webp') }) }),
      fc.record({ type: fc.constant('rotate' as const), options: fc.record({ angle: fc.integer({ min: 0, max: 360 }) }) }),
      fc.record({ type: fc.constant('flip' as const), options: fc.record({ horizontal: fc.boolean() }) })
    ) as fc.Arbitrary<ImageOperation>;

    await fc.assert(
      fc.property(
        fc.hexaString({ minLength: 64, maxLength: 64 }),
        operationArb,
        (inputHash, operation) => {
          const client = new PlatformCacheClient('', 3600);
          const key = client.generateKey(inputHash, operation);
          
          expect(key.startsWith('img:')).toBe(true);
        }
      ),
      { numRuns: 100 }
    );
  });

  it('should generate unique keys for different operations on same input', async () => {
    await fc.assert(
      fc.property(
        fc.hexaString({ minLength: 64, maxLength: 64 }),
        fc.integer({ min: 1, max: 500 }),
        fc.integer({ min: 501, max: 1000 }),
        (inputHash, width1, width2) => {
          const client = new PlatformCacheClient('', 3600);
          const op1: ImageOperation = { type: 'resize', options: { width: width1 } };
          const op2: ImageOperation = { type: 'resize', options: { width: width2 } };
          
          const key1 = client.generateKey(inputHash, op1);
          const key2 = client.generateKey(inputHash, op2);
          
          expect(key1.startsWith('img:')).toBe(true);
          expect(key2.startsWith('img:')).toBe(true);
          expect(key1).not.toBe(key2);
        }
      ),
      { numRuns: 100 }
    );
  });

  it('should generate same key for identical input and operation', async () => {
    await fc.assert(
      fc.property(
        fc.hexaString({ minLength: 64, maxLength: 64 }),
        fc.integer({ min: 1, max: 1000 }),
        (inputHash, width) => {
          const client = new PlatformCacheClient('', 3600);
          const operation: ImageOperation = { type: 'resize', options: { width } };
          
          const key1 = client.generateKey(inputHash, operation);
          const key2 = client.generateKey(inputHash, operation);
          
          expect(key1).toBe(key2);
          expect(key1.startsWith('img:')).toBe(true);
        }
      ),
      { numRuns: 100 }
    );
  });
});

/**
 * Feature: image-processing-modernization-2025
 * Property 3: Cache Round-Trip Preserves Image Metadata
 * 
 * For any ProcessedImage cached and then retrieved, the retrieved image SHALL
 * have identical metadata (width, height, format, size, hasAlpha) to the original.
 * 
 * Validates: Requirements 2.5
 */
describe('Property 3: Cache Round-Trip Preserves Image Metadata', () => {
  const metadataArb: fc.Arbitrary<ImageMetadata> = fc.record({
    width: fc.integer({ min: 1, max: 10000 }),
    height: fc.integer({ min: 1, max: 10000 }),
    format: fc.constantFrom('jpeg', 'png', 'gif', 'webp', 'tiff'),
    size: fc.integer({ min: 1, max: 100000000 }),
    hasAlpha: fc.boolean(),
  });

  it('should preserve all metadata fields after cache round-trip', async () => {
    await fc.assert(
      fc.asyncProperty(
        metadataArb,
        fc.uint8Array({ minLength: 10, maxLength: 1000 }),
        async (metadata, bufferData) => {
          const client = new PlatformCacheClient('', 3600);
          const buffer = Buffer.from(bufferData);
          
          const original: ProcessedImage = { buffer, metadata };
          const key = `test-${Date.now()}-${Math.random()}`;
          
          await client.setImage(key, original);
          const retrieved = await client.getImage(key);
          
          expect(retrieved).not.toBeNull();
          expect(retrieved!.metadata.width).toBe(metadata.width);
          expect(retrieved!.metadata.height).toBe(metadata.height);
          expect(retrieved!.metadata.format).toBe(metadata.format);
          expect(retrieved!.metadata.size).toBe(metadata.size);
          expect(retrieved!.metadata.hasAlpha).toBe(metadata.hasAlpha);
        }
      ),
      { numRuns: 100 }
    );
  });

  it('should preserve buffer content after cache round-trip', async () => {
    await fc.assert(
      fc.asyncProperty(
        fc.uint8Array({ minLength: 10, maxLength: 1000 }),
        async (bufferData) => {
          const client = new PlatformCacheClient('', 3600);
          const buffer = Buffer.from(bufferData);
          
          const original: ProcessedImage = {
            buffer,
            metadata: { width: 100, height: 100, format: 'png', size: buffer.length, hasAlpha: false },
          };
          const key = `test-buffer-${Date.now()}-${Math.random()}`;
          
          await client.setImage(key, original);
          const retrieved = await client.getImage(key);
          
          expect(retrieved).not.toBeNull();
          expect(retrieved!.buffer.equals(buffer)).toBe(true);
        }
      ),
      { numRuns: 100 }
    );
  });

  it('should return null for non-existent keys', async () => {
    await fc.assert(
      fc.asyncProperty(
        fc.uuid(),
        async (randomKey) => {
          const client = new PlatformCacheClient('', 3600);
          const retrieved = await client.getImage(`nonexistent-${randomKey}`);
          expect(retrieved).toBeNull();
        }
      ),
      { numRuns: 100 }
    );
  });
});
