import * as fc from 'fast-check';
import { 
  validate, validateSafe, 
  resizeSchema, convertSchema, adjustSchema, rotateSchema, flipSchema, compressSchema 
} from '../../../src/api/validators/schemas';
import { AppError } from '../../../src/domain/errors';

/**
 * Feature: image-processing-modernization-2025
 * Property 4: Validation Errors Contain Field-Level Details
 * 
 * For any validation failure, the error response SHALL contain structured
 * field-level details including the field path, error message, and error code.
 * 
 * Validates: Requirements 3.4
 */
describe('Property 4: Validation Errors Contain Field-Level Details', () => {
  it('should include field path in validation errors for resize', async () => {
    await fc.assert(
      fc.property(
        fc.integer({ min: -1000, max: 0 }), // Invalid width
        (invalidWidth) => {
          try {
            validate(resizeSchema, { width: invalidWidth });
            fail('Should have thrown');
          } catch (error) {
            expect(error).toBeInstanceOf(AppError);
            const appError = error as AppError;
            expect(appError.fields).toBeDefined();
            expect(appError.fields!.length).toBeGreaterThan(0);
            expect(appError.fields![0]).toHaveProperty('field');
            expect(appError.fields![0]).toHaveProperty('message');
            expect(appError.fields![0]).toHaveProperty('code');
          }
        }
      ),
      { numRuns: 100 }
    );
  });

  it('should include field path in validation errors for convert', async () => {
    await fc.assert(
      fc.property(
        fc.string({ minLength: 1, maxLength: 10 }).filter(s => !['jpeg', 'png', 'gif', 'webp', 'tiff'].includes(s)),
        (invalidFormat) => {
          try {
            validate(convertSchema, { format: invalidFormat });
            fail('Should have thrown');
          } catch (error) {
            expect(error).toBeInstanceOf(AppError);
            const appError = error as AppError;
            expect(appError.fields).toBeDefined();
            expect(appError.fields!.length).toBeGreaterThan(0);
            
            const formatError = appError.fields!.find(f => f.field === 'format');
            expect(formatError).toBeDefined();
            expect(formatError!.message).toBeDefined();
            expect(formatError!.code).toBeDefined();
          }
        }
      ),
      { numRuns: 100 }
    );
  });

  it('should include field path in validation errors for adjust', async () => {
    await fc.assert(
      fc.property(
        fc.integer({ min: 101, max: 1000 }), // Invalid brightness > 100
        (invalidBrightness) => {
          try {
            validate(adjustSchema, { brightness: invalidBrightness });
            fail('Should have thrown');
          } catch (error) {
            expect(error).toBeInstanceOf(AppError);
            const appError = error as AppError;
            expect(appError.fields).toBeDefined();
            expect(appError.fields!.length).toBeGreaterThan(0);
            expect(appError.fields![0].field).toBe('brightness');
          }
        }
      ),
      { numRuns: 100 }
    );
  });

  it('should include field path in validation errors for rotate', async () => {
    await fc.assert(
      fc.property(
        fc.integer({ min: 361, max: 1000 }), // Invalid angle > 360
        (invalidAngle) => {
          try {
            validate(rotateSchema, { angle: invalidAngle });
            fail('Should have thrown');
          } catch (error) {
            expect(error).toBeInstanceOf(AppError);
            const appError = error as AppError;
            expect(appError.fields).toBeDefined();
            expect(appError.fields!.length).toBeGreaterThan(0);
            expect(appError.fields![0].field).toBe('angle');
          }
        }
      ),
      { numRuns: 100 }
    );
  });

  it('should return structured errors with validateSafe', async () => {
    await fc.assert(
      fc.property(
        fc.integer({ min: -1000, max: 0 }),
        (invalidWidth) => {
          const result = validateSafe(resizeSchema, { width: invalidWidth });
          
          expect(result.success).toBe(false);
          if (!result.success) {
            expect(result.errors.length).toBeGreaterThan(0);
            expect(result.errors[0]).toHaveProperty('field');
            expect(result.errors[0]).toHaveProperty('message');
            expect(result.errors[0]).toHaveProperty('code');
          }
        }
      ),
      { numRuns: 100 }
    );
  });

  it('should validate successfully for valid inputs', async () => {
    await fc.assert(
      fc.property(
        fc.integer({ min: 1, max: 10000 }),
        fc.integer({ min: 1, max: 10000 }),
        (width, height) => {
          const result = validateSafe(resizeSchema, { width, height });
          expect(result.success).toBe(true);
          if (result.success) {
            expect(result.data.width).toBe(width);
            expect(result.data.height).toBe(height);
          }
        }
      ),
      { numRuns: 100 }
    );
  });

  it('should include error code from Zod in field errors', async () => {
    await fc.assert(
      fc.property(
        fc.constant({}), // Empty object - missing required fields
        () => {
          const result = validateSafe(convertSchema, {});
          
          expect(result.success).toBe(false);
          if (!result.success) {
            expect(result.errors.length).toBeGreaterThan(0);
            // Zod error codes like 'invalid_type', 'invalid_enum_value', etc.
            expect(typeof result.errors[0].code).toBe('string');
            expect(result.errors[0].code.length).toBeGreaterThan(0);
          }
        }
      ),
      { numRuns: 100 }
    );
  });
});
