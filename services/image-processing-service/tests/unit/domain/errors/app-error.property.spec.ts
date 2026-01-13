import * as fc from 'fast-check';
import { AppError } from '../../../../src/domain/errors/app-error';
import { ErrorCode } from '../../../../src/domain/errors/error-codes';

/**
 * Feature: image-processing-modernization-2025
 * Property 5: Error Serialization Is Consistent
 * 
 * For any AppError instance, serializing to JSON and deserializing SHALL produce
 * an equivalent error with the same code, message, and HTTP status.
 * 
 * Validates: Requirements 4.3
 */
describe('Property 5: Error Serialization Is Consistent', () => {
  const errorCodeArb = fc.constantFrom(
    ErrorCode.VALIDATION_ERROR,
    ErrorCode.INVALID_DIMENSIONS,
    ErrorCode.INVALID_FORMAT,
    ErrorCode.INVALID_QUALITY,
    ErrorCode.PROCESSING_ERROR,
    ErrorCode.IMAGE_NOT_FOUND
  );

  it('should preserve code and message after JSON serialization', async () => {
    await fc.assert(
      fc.property(
        errorCodeArb,
        fc.string({ minLength: 1, maxLength: 200 }),
        (code, message) => {
          const error = new AppError(code, message);
          const json = error.toJSON();
          
          expect(json.code).toBe(code);
          expect(json.message).toBe(message);
        }
      ),
      { numRuns: 100 }
    );
  });

  it('should preserve details after JSON serialization', async () => {
    await fc.assert(
      fc.property(
        errorCodeArb,
        fc.string({ minLength: 1, maxLength: 100 }),
        fc.dictionary(fc.string({ minLength: 1, maxLength: 20 }), fc.string()),
        (code, message, details) => {
          const error = new AppError(code, message, details);
          const json = error.toJSON();
          
          expect(json.details).toEqual(details);
        }
      ),
      { numRuns: 100 }
    );
  });

  it('should preserve field errors in validation errors', async () => {
    await fc.assert(
      fc.property(
        fc.string({ minLength: 1, maxLength: 100 }),
        fc.array(
          fc.record({
            field: fc.string({ minLength: 1, maxLength: 50 }),
            message: fc.string({ minLength: 1, maxLength: 100 }),
            code: fc.string({ minLength: 1, maxLength: 30 }),
          }),
          { minLength: 1, maxLength: 5 }
        ),
        (message, fields) => {
          const error = AppError.validationError(message, fields);
          const json = error.toJSON();
          
          expect(json.code).toBe(ErrorCode.VALIDATION_ERROR);
          expect(json.message).toBe(message);
          expect(json.fields).toEqual(fields);
        }
      ),
      { numRuns: 100 }
    );
  });

  it('should produce valid JSON string', async () => {
    await fc.assert(
      fc.property(
        errorCodeArb,
        fc.string({ minLength: 1, maxLength: 100 }),
        (code, message) => {
          const error = new AppError(code, message);
          const jsonString = JSON.stringify(error.toJSON());
          
          expect(() => JSON.parse(jsonString)).not.toThrow();
          const parsed = JSON.parse(jsonString);
          expect(parsed.code).toBe(code);
          expect(parsed.message).toBe(message);
        }
      ),
      { numRuns: 100 }
    );
  });

  it('should have correct HTTP status for each error code', async () => {
    await fc.assert(
      fc.property(
        errorCodeArb,
        fc.string({ minLength: 1, maxLength: 100 }),
        (code, message) => {
          const error = new AppError(code, message);
          
          expect(error.httpStatus).toBeGreaterThanOrEqual(400);
          expect(error.httpStatus).toBeLessThan(600);
          
          // Validation errors should be 400
          if (code === ErrorCode.VALIDATION_ERROR || 
              code === ErrorCode.INVALID_DIMENSIONS ||
              code === ErrorCode.INVALID_FORMAT ||
              code === ErrorCode.INVALID_QUALITY) {
            expect(error.httpStatus).toBe(400);
          }
          
          // Not found errors should be 404
          if (code === ErrorCode.IMAGE_NOT_FOUND) {
            expect(error.httpStatus).toBe(404);
          }
          
          // Server errors should be 500
          if (code === ErrorCode.PROCESSING_ERROR) {
            expect(error.httpStatus).toBe(500);
          }
        }
      ),
      { numRuns: 100 }
    );
  });

  it('should be instanceof Error', async () => {
    await fc.assert(
      fc.property(
        errorCodeArb,
        fc.string({ minLength: 1, maxLength: 100 }),
        (code, message) => {
          const error = new AppError(code, message);
          
          expect(error).toBeInstanceOf(Error);
          expect(error).toBeInstanceOf(AppError);
          expect(error.name).toBe('AppError');
        }
      ),
      { numRuns: 100 }
    );
  });
});
