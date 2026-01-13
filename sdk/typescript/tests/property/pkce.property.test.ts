/**
 * Property-based tests for PKCE implementation
 *
 * **Feature: typescript-sdk-modernization**
 * **Properties: 7, 8, 9**
 */

import { describe, it, expect } from 'vitest';
import * as fc from 'fast-check';
import { generatePKCE, verifyPKCE } from '../../src/pkce.js';

/**
 * **Property 7: PKCE S256 Method Enforcement**
 * For any PKCE challenge, codeChallengeMethod is always 'S256'.
 * **Validates: Requirements 5.1**
 */
describe('Property 7: PKCE S256 Method Enforcement', () => {
  it('always uses S256 method', async () => {
    await fc.assert(
      fc.asyncProperty(fc.constant(null), async () => {
        const challenge = await generatePKCE();
        expect(challenge.codeChallengeMethod).toBe('S256');
      }),
      { numRuns: 100 }
    );
  });
});

/**
 * **Property 8: PKCE Verifier Uniqueness**
 * For any two PKCE challenges, codeVerifier values are different.
 * **Validates: Requirements 5.2**
 */
describe('Property 8: PKCE Verifier Uniqueness', () => {
  it('generates unique verifiers', async () => {
    const verifiers = new Set<string>();
    const iterations = 100;

    for (let i = 0; i < iterations; i++) {
      const challenge = await generatePKCE();
      verifiers.add(challenge.codeVerifier);
    }

    // All verifiers should be unique
    expect(verifiers.size).toBe(iterations);
  });

  it('verifier length is within RFC 7636 bounds (43-128 chars)', async () => {
    await fc.assert(
      fc.asyncProperty(fc.constant(null), async () => {
        const challenge = await generatePKCE();
        expect(challenge.codeVerifier.length).toBeGreaterThanOrEqual(43);
        expect(challenge.codeVerifier.length).toBeLessThanOrEqual(128);
      }),
      { numRuns: 100 }
    );
  });

  it('verifier uses only valid base64url characters', async () => {
    const base64urlRegex = /^[A-Za-z0-9_-]+$/;

    await fc.assert(
      fc.asyncProperty(fc.constant(null), async () => {
        const challenge = await generatePKCE();
        expect(base64urlRegex.test(challenge.codeVerifier)).toBe(true);
      }),
      { numRuns: 100 }
    );
  });
});

/**
 * **Property 9: PKCE Round-Trip Verification**
 * For any PKCE challenge, verifying verifier against challenge returns true.
 * **Validates: Requirements 5.1, 5.2**
 */
describe('Property 9: PKCE Round-Trip Verification', () => {
  it('verifier matches its own challenge', async () => {
    await fc.assert(
      fc.asyncProperty(fc.constant(null), async () => {
        const challenge = await generatePKCE();
        const isValid = await verifyPKCE(
          challenge.codeVerifier,
          challenge.codeChallenge
        );
        expect(isValid).toBe(true);
      }),
      { numRuns: 100 }
    );
  });

  it('verifier does not match different challenge', async () => {
    await fc.assert(
      fc.asyncProperty(fc.constant(null), async () => {
        const challenge1 = await generatePKCE();
        const challenge2 = await generatePKCE();

        // Different verifiers should not match each other's challenges
        if (challenge1.codeVerifier !== challenge2.codeVerifier) {
          const isValid = await verifyPKCE(
            challenge1.codeVerifier,
            challenge2.codeChallenge
          );
          expect(isValid).toBe(false);
        }
      }),
      { numRuns: 100 }
    );
  });
});
