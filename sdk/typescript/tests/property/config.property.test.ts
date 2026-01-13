/**
 * Property-based tests for configuration validation
 *
 * **Feature: typescript-sdk-modernization**
 * **Properties: 2, 10**
 */

import { describe, it, expect } from 'vitest';
import * as fc from 'fast-check';
import {
  validateConfig,
  InvalidConfigError,
  MissingRequiredFieldError,
} from '../../src/index.js';
import { timingSafeEqual } from '../../src/utils/crypto.js';

// Arbitrary for valid config
const validConfigArbitrary = fc.record({
  baseUrl: fc.webUrl(),
  clientId: fc.string({ minLength: 1, maxLength: 100 }),
  clientSecret: fc.option(fc.string({ minLength: 1 }), { nil: undefined }),
  scopes: fc.option(fc.array(fc.string({ minLength: 1 })), { nil: undefined }),
  timeout: fc.option(fc.integer({ min: 1000, max: 60000 }), { nil: undefined }),
  refreshBuffer: fc.option(fc.integer({ min: 1000, max: 120000 }), { nil: undefined }),
});

/**
 * **Property 2: Schema Validation Consistency**
 * Valid configs pass validation, invalid configs throw appropriate errors.
 * **Validates: Requirements 3.7**
 */
describe('Property 2: Schema Validation Consistency', () => {
  it('valid configuration passes validation', () => {
    fc.assert(
      fc.property(validConfigArbitrary, (config) => {
        const validated = validateConfig(config);

        expect(validated.baseUrl).toBe(config.baseUrl);
        expect(validated.clientId).toBe(config.clientId);
        expect(validated.clientSecret).toBe(config.clientSecret);
      }),
      { numRuns: 100 }
    );
  });

  it('missing baseUrl throws MissingRequiredFieldError', () => {
    fc.assert(
      fc.property(
        fc.string({ minLength: 1 }),
        (clientId) => {
          expect(() => validateConfig({ clientId })).toThrow(MissingRequiredFieldError);
        }
      ),
      { numRuns: 100 }
    );
  });

  it('missing clientId throws MissingRequiredFieldError', () => {
    fc.assert(
      fc.property(fc.webUrl(), (baseUrl) => {
        expect(() => validateConfig({ baseUrl })).toThrow(MissingRequiredFieldError);
      }),
      { numRuns: 100 }
    );
  });

  it('empty baseUrl throws MissingRequiredFieldError', () => {
    fc.assert(
      fc.property(fc.string({ minLength: 1 }), (clientId) => {
        expect(() => validateConfig({ baseUrl: '', clientId })).toThrow(
          MissingRequiredFieldError
        );
      }),
      { numRuns: 100 }
    );
  });

  it('empty clientId throws MissingRequiredFieldError', () => {
    fc.assert(
      fc.property(fc.webUrl(), (baseUrl) => {
        expect(() => validateConfig({ baseUrl, clientId: '' })).toThrow(
          MissingRequiredFieldError
        );
      }),
      { numRuns: 100 }
    );
  });

  it('non-object config throws InvalidConfigError', () => {
    const invalidValues = ['string', 42, true, []];

    for (const value of invalidValues) {
      expect(() => validateConfig(value)).toThrow(InvalidConfigError);
    }

    // null and undefined also throw InvalidConfigError
    expect(() => validateConfig(null)).toThrow(InvalidConfigError);
    expect(() => validateConfig(undefined)).toThrow(InvalidConfigError);
  });

  it('applies default values for optional fields', () => {
    fc.assert(
      fc.property(fc.webUrl(), fc.string({ minLength: 1 }), (baseUrl, clientId) => {
        const validated = validateConfig({ baseUrl, clientId });

        expect(validated.timeout).toBe(30_000);
        expect(validated.refreshBuffer).toBe(60_000);
      }),
      { numRuns: 100 }
    );
  });

  it('preserves provided optional values over defaults', () => {
    fc.assert(
      fc.property(
        fc.webUrl(),
        fc.string({ minLength: 1 }),
        fc.integer({ min: 1000, max: 60000 }),
        fc.integer({ min: 1000, max: 120000 }),
        (baseUrl, clientId, timeout, refreshBuffer) => {
          const validated = validateConfig({ baseUrl, clientId, timeout, refreshBuffer });

          expect(validated.timeout).toBe(timeout);
          expect(validated.refreshBuffer).toBe(refreshBuffer);
        }
      ),
      { numRuns: 100 }
    );
  });
});

/**
 * **Property 10: State Parameter Validation**
 * State validation returns true only for exact matches.
 * **Validates: Requirements 5.4**
 */
describe('Property 10: State Parameter Validation', () => {
  it('returns true for identical strings', () => {
    fc.assert(
      fc.property(fc.string({ minLength: 1, maxLength: 100 }), (state) => {
        expect(timingSafeEqual(state, state)).toBe(true);
      }),
      { numRuns: 100 }
    );
  });

  it('returns false for different strings', () => {
    fc.assert(
      fc.property(
        fc.string({ minLength: 1, maxLength: 100 }),
        fc.string({ minLength: 1, maxLength: 100 }),
        (a, b) => {
          if (a !== b) {
            expect(timingSafeEqual(a, b)).toBe(false);
          }
        }
      ),
      { numRuns: 100 }
    );
  });

  it('returns false for strings of different lengths', () => {
    fc.assert(
      fc.property(
        fc.string({ minLength: 1, maxLength: 50 }),
        fc.string({ minLength: 51, maxLength: 100 }),
        (short, long) => {
          expect(timingSafeEqual(short, long)).toBe(false);
        }
      ),
      { numRuns: 100 }
    );
  });

  it('handles empty strings correctly', () => {
    expect(timingSafeEqual('', '')).toBe(true);
    expect(timingSafeEqual('', 'a')).toBe(false);
    expect(timingSafeEqual('a', '')).toBe(false);
  });
});
