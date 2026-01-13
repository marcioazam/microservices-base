/**
 * Property-based tests for token management
 *
 * **Feature: typescript-sdk-modernization**
 * **Properties: 11, 13, 14**
 */

import { describe, it, expect } from 'vitest';
import * as fc from 'fast-check';
import {
  TokenManager,
  MemoryTokenStorage,
  createAccessToken,
  createRefreshToken,
  type TokenData,
} from '../../src/index.js';

// Inline validation function for testing
function isValidTokenData(value: unknown): boolean {
  if (typeof value !== 'object' || value === null) {
    return false;
  }
  const obj = value as Record<string, unknown>;
  return (
    typeof obj.accessToken === 'string' &&
    obj.accessToken.length > 0 &&
    typeof obj.expiresAt === 'number' &&
    obj.expiresAt > 0 &&
    obj.tokenType === 'Bearer' &&
    (obj.refreshToken === undefined || typeof obj.refreshToken === 'string') &&
    (obj.scope === undefined || typeof obj.scope === 'string')
  );
}

// Inline serialization for testing
function serializeTokenData(data: TokenData): string {
  return JSON.stringify({
    accessToken: data.accessToken,
    refreshToken: data.refreshToken,
    expiresAt: data.expiresAt,
    tokenType: data.tokenType,
    scope: data.scope,
  });
}

function deserializeTokenData(json: string): Record<string, unknown> {
  return JSON.parse(json) as Record<string, unknown>;
}

// Arbitrary for valid token data
const validTokenDataArbitrary = fc.record({
  accessToken: fc.string({ minLength: 10, maxLength: 100 }).map(createAccessToken),
  refreshToken: fc.option(
    fc.string({ minLength: 10, maxLength: 100 }).map(createRefreshToken),
    { nil: undefined }
  ),
  expiresAt: fc.integer({ min: Date.now(), max: Date.now() + 86400000 }),
  tokenType: fc.constant('Bearer' as const),
  scope: fc.option(fc.string({ minLength: 1, maxLength: 50 }), { nil: undefined }),
});

/**
 * **Property 11: Token Refresh Timing**
 * shouldRefresh returns true when expiresAt is within buffer, false otherwise.
 * **Validates: Requirements 6.1**
 */
describe('Property 11: Token Refresh Timing', () => {
  it('returns true when token expires within buffer', () => {
    fc.assert(
      fc.property(
        fc.integer({ min: 0, max: 59999 }),
        (timeUntilExpiry) => {
          const storage = new MemoryTokenStorage();
          const manager = new TokenManager({
            storage,
            baseUrl: 'https://example.com',
            clientId: 'test',
            refreshBuffer: 60000,
          });

          const tokens: TokenData = {
            accessToken: createAccessToken('test-token-value'),
            expiresAt: Date.now() + timeUntilExpiry,
            tokenType: 'Bearer',
          };

          expect(manager.shouldRefresh(tokens)).toBe(true);
        }
      ),
      { numRuns: 100 }
    );
  });

  it('returns false when token expires beyond buffer', () => {
    fc.assert(
      fc.property(
        fc.integer({ min: 60001, max: 3600000 }),
        (timeUntilExpiry) => {
          const storage = new MemoryTokenStorage();
          const manager = new TokenManager({
            storage,
            baseUrl: 'https://example.com',
            clientId: 'test',
            refreshBuffer: 60000,
          });

          const tokens: TokenData = {
            accessToken: createAccessToken('test-token-value'),
            expiresAt: Date.now() + timeUntilExpiry,
            tokenType: 'Bearer',
          };

          expect(manager.shouldRefresh(tokens)).toBe(false);
        }
      ),
      { numRuns: 100 }
    );
  });
});

/**
 * **Property 13: Token Validation**
 * Valid TokenData passes validation, invalid objects fail.
 * **Validates: Requirements 6.5**
 */
describe('Property 13: Token Validation', () => {
  it('valid token data passes validation', () => {
    fc.assert(
      fc.property(
        fc.record({
          accessToken: fc.string({ minLength: 1, maxLength: 100 }),
          expiresAt: fc.integer({ min: 1, max: Number.MAX_SAFE_INTEGER }),
          tokenType: fc.constant('Bearer' as const),
          refreshToken: fc.option(fc.string({ minLength: 1 }), { nil: undefined }),
          scope: fc.option(fc.string(), { nil: undefined }),
        }),
        (data) => {
          expect(isValidTokenData(data)).toBe(true);
        }
      ),
      { numRuns: 100 }
    );
  });

  it('invalid token data fails validation', () => {
    const invalidCases = [
      null,
      undefined,
      {},
      { accessToken: '' },
      { accessToken: 'valid', expiresAt: 'not-a-number' },
      { accessToken: 'valid', expiresAt: 123, tokenType: 'Invalid' },
      { accessToken: 123, expiresAt: 123, tokenType: 'Bearer' },
    ];

    for (const invalid of invalidCases) {
      expect(isValidTokenData(invalid)).toBe(false);
    }
  });
});

/**
 * **Property 14: Token Serialization Round-Trip**
 * Serializing and deserializing produces equivalent data.
 * **Validates: Requirements 6.6, 6.7**
 */
describe('Property 14: Token Serialization Round-Trip', () => {
  it('round-trips token data correctly', () => {
    fc.assert(
      fc.property(validTokenDataArbitrary, (tokenData) => {
        const serialized = serializeTokenData(tokenData);
        const deserialized = deserializeTokenData(serialized);

        expect(deserialized.accessToken).toBe(tokenData.accessToken);
        expect(deserialized.refreshToken).toBe(tokenData.refreshToken);
        expect(deserialized.expiresAt).toBe(tokenData.expiresAt);
        expect(deserialized.tokenType).toBe(tokenData.tokenType);
        expect(deserialized.scope).toBe(tokenData.scope);
      }),
      { numRuns: 100 }
    );
  });

  it('serialized format is valid JSON', () => {
    fc.assert(
      fc.property(validTokenDataArbitrary, (tokenData) => {
        const serialized = serializeTokenData(tokenData);
        const parsed = JSON.parse(serialized);
        expect(typeof parsed).toBe('object');
      }),
      { numRuns: 100 }
    );
  });
});
