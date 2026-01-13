/**
 * Property-based tests for error system
 *
 * **Feature: typescript-sdk-modernization**
 * **Properties: 1, 3, 4**
 */

import { describe, it, expect } from 'vitest';
import * as fc from 'fast-check';
import {
  AuthPlatformError,
  TokenExpiredError,
  TokenRefreshError,
  TokenInvalidError,
  NetworkError,
  TimeoutError,
  RateLimitError,
  InvalidConfigError,
  MissingRequiredFieldError,
  PasskeyError,
  PasskeyNotSupportedError,
  PasskeyCancelledError,
  PasskeyRegistrationError,
  PasskeyAuthError,
  CaepConnectionError,
  CaepParseError,
  ErrorCode,
  isAuthPlatformError,
  isTokenExpiredError,
  isTokenRefreshError,
  isTokenInvalidError,
  isNetworkError,
  isTimeoutError,
  isRateLimitError,
  isInvalidConfigError,
  isMissingRequiredFieldError,
  isPasskeyError,
  isPasskeyNotSupportedError,
  isPasskeyCancelledError,
  isPasskeyRegistrationError,
  isPasskeyAuthError,
  isCaepConnectionError,
  isCaepParseError,
} from '../../src/index.js';

/**
 * **Property 1: Type Guard Correctness**
 * For any error instance, the corresponding type guard returns true,
 * and all other type guards return false.
 * **Validates: Requirements 3.2, 4.7**
 */
describe('Property 1: Type Guard Correctness', () => {
  const errorFactories = [
    { create: () => new TokenExpiredError(), guard: isTokenExpiredError, name: 'TokenExpiredError' },
    { create: () => new TokenRefreshError(), guard: isTokenRefreshError, name: 'TokenRefreshError' },
    { create: () => new TokenInvalidError(), guard: isTokenInvalidError, name: 'TokenInvalidError' },
    { create: () => new NetworkError(), guard: isNetworkError, name: 'NetworkError' },
    { create: () => new TimeoutError(), guard: isTimeoutError, name: 'TimeoutError' },
    { create: () => new RateLimitError(), guard: isRateLimitError, name: 'RateLimitError' },
    { create: () => new InvalidConfigError('test'), guard: isInvalidConfigError, name: 'InvalidConfigError' },
    { create: () => new MissingRequiredFieldError('field'), guard: isMissingRequiredFieldError, name: 'MissingRequiredFieldError' },
    { create: () => new PasskeyNotSupportedError(), guard: isPasskeyNotSupportedError, name: 'PasskeyNotSupportedError' },
    { create: () => new PasskeyCancelledError(), guard: isPasskeyCancelledError, name: 'PasskeyCancelledError' },
    { create: () => new PasskeyRegistrationError(), guard: isPasskeyRegistrationError, name: 'PasskeyRegistrationError' },
    { create: () => new PasskeyAuthError(), guard: isPasskeyAuthError, name: 'PasskeyAuthError' },
    { create: () => new CaepConnectionError(), guard: isCaepConnectionError, name: 'CaepConnectionError' },
    { create: () => new CaepParseError(), guard: isCaepParseError, name: 'CaepParseError' },
  ];

  it('each error type guard returns true only for its own type', () => {
    fc.assert(
      fc.property(fc.integer({ min: 0, max: errorFactories.length - 1 }), (index) => {
        const factory = errorFactories[index]!;
        const error = factory.create();

        // Own guard should return true
        expect(factory.guard(error)).toBe(true);

        // All errors should pass isAuthPlatformError
        expect(isAuthPlatformError(error)).toBe(true);

        // Other specific guards should return false (except parent classes)
        for (const other of errorFactories) {
          if (other.name !== factory.name) {
            // Skip passkey parent class check
            if (factory.name.startsWith('Passkey') && other.name === 'PasskeyError') {
              continue;
            }
            if (other.name.startsWith('Passkey') && factory.name.startsWith('Passkey')) {
              continue;
            }
            expect(other.guard(error)).toBe(false);
          }
        }
      }),
      { numRuns: 100 }
    );
  });

  it('type guards return false for non-error values', () => {
    const nonErrors = [null, undefined, 'string', 42, {}, [], new Error('plain')];
    const allGuards = [
      isAuthPlatformError,
      isTokenExpiredError,
      isTokenRefreshError,
      isNetworkError,
      isRateLimitError,
    ];

    for (const value of nonErrors) {
      for (const guard of allGuards) {
        expect(guard(value)).toBe(false);
      }
    }
  });
});

/**
 * **Property 3: Error Cause Chain Preservation**
 * For any error created with a cause, the cause property references the original.
 * **Validates: Requirements 4.3**
 */
describe('Property 3: Error Cause Chain Preservation', () => {
  it('preserves cause chain for any error with cause', () => {
    fc.assert(
      fc.property(fc.string(), fc.string(), (message, causeMessage) => {
        const originalCause = new Error(causeMessage);
        const error = new NetworkError(message, { cause: originalCause });

        expect(error.cause).toBe(originalCause);
        expect((error.cause as Error).message).toBe(causeMessage);
      }),
      { numRuns: 100 }
    );
  });

  it('preserves nested cause chains', () => {
    fc.assert(
      fc.property(
        fc.array(fc.string(), { minLength: 2, maxLength: 5 }),
        (messages) => {
          let currentCause: Error | undefined;

          for (const msg of messages) {
            currentCause = new NetworkError(msg, { cause: currentCause });
          }

          // Walk the chain and verify
          let depth = 0;
          let current: Error | undefined = currentCause;
          while (current !== undefined) {
            depth++;
            current = current.cause as Error | undefined;
          }

          expect(depth).toBe(messages.length);
        }
      ),
      { numRuns: 100 }
    );
  });
});

/**
 * **Property 4: Error Correlation ID Preservation**
 * For any error with correlationId, the property equals the provided value.
 * **Validates: Requirements 4.4**
 */
describe('Property 4: Error Correlation ID Preservation', () => {
  it('preserves correlation ID for any error', () => {
    fc.assert(
      fc.property(
        fc.string({ minLength: 1, maxLength: 64 }),
        fc.string(),
        (correlationId, message) => {
          const error = new NetworkError(message, { correlationId });
          expect(error.correlationId).toBe(correlationId);
        }
      ),
      { numRuns: 100 }
    );
  });

  it('correlation ID is undefined when not provided', () => {
    fc.assert(
      fc.property(fc.string(), (message) => {
        const error = new NetworkError(message);
        expect(error.correlationId).toBeUndefined();
      }),
      { numRuns: 100 }
    );
  });
});
