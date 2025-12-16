/**
 * Property-based tests for PKCE implementation
 *
 * **Feature: auth-platform-q2-2025-evolution, Property 10: SDK PKCE Enforcement**
 * **Validates: Requirements 8.2, 9.4, 10.1**
 */

import * as fc from 'fast-check';
import { generatePKCE, verifyPKCE } from '../pkce';

describe('PKCE Property Tests', () => {
  /**
   * Property 10: SDK PKCE Enforcement
   * For any OAuth authorization flow initiated through any SDK,
   * the flow SHALL use PKCE with S256 challenge method.
   */
  describe('Property 10: SDK PKCE Enforcement', () => {
    it('always generates S256 challenge method', async () => {
      await fc.assert(
        fc.asyncProperty(fc.integer({ min: 1, max: 100 }), async () => {
          const pkce = await generatePKCE();

          // Property: Challenge method must always be S256
          expect(pkce.codeChallengeMethod).toBe('S256');
        }),
        { numRuns: 100 }
      );
    });

    it('generates unique code verifiers', async () => {
      const verifiers = new Set<string>();

      await fc.assert(
        fc.asyncProperty(fc.integer({ min: 1, max: 100 }), async () => {
          const pkce = await generatePKCE();

          // Property: Each code verifier must be unique
          expect(verifiers.has(pkce.codeVerifier)).toBe(false);
          verifiers.add(pkce.codeVerifier);
        }),
        { numRuns: 100 }
      );
    });

    it('generates valid base64url encoded verifiers', async () => {
      await fc.assert(
        fc.asyncProperty(fc.integer({ min: 1, max: 100 }), async () => {
          const pkce = await generatePKCE();

          // Property: Code verifier must be valid base64url (no padding)
          expect(pkce.codeVerifier).toMatch(/^[A-Za-z0-9_-]+$/);
          expect(pkce.codeVerifier).not.toContain('=');
        }),
        { numRuns: 100 }
      );
    });

    it('generates valid base64url encoded challenges', async () => {
      await fc.assert(
        fc.asyncProperty(fc.integer({ min: 1, max: 100 }), async () => {
          const pkce = await generatePKCE();

          // Property: Code challenge must be valid base64url (no padding)
          expect(pkce.codeChallenge).toMatch(/^[A-Za-z0-9_-]+$/);
          expect(pkce.codeChallenge).not.toContain('=');
        }),
        { numRuns: 100 }
      );
    });

    it('verifier and challenge are different', async () => {
      await fc.assert(
        fc.asyncProperty(fc.integer({ min: 1, max: 100 }), async () => {
          const pkce = await generatePKCE();

          // Property: Verifier and challenge must be different (challenge is hash)
          expect(pkce.codeVerifier).not.toBe(pkce.codeChallenge);
        }),
        { numRuns: 100 }
      );
    });

    it('verifier can be verified against challenge', async () => {
      await fc.assert(
        fc.asyncProperty(fc.integer({ min: 1, max: 100 }), async () => {
          const pkce = await generatePKCE();

          // Property: Verifier must verify against its challenge
          const isValid = await verifyPKCE(pkce.codeVerifier, pkce.codeChallenge);
          expect(isValid).toBe(true);
        }),
        { numRuns: 100 }
      );
    });

    it('wrong verifier fails verification', async () => {
      await fc.assert(
        fc.asyncProperty(
          fc.string({ minLength: 43, maxLength: 128 }).filter((s) => /^[A-Za-z0-9_-]+$/.test(s)),
          async (wrongVerifier) => {
            const pkce = await generatePKCE();

            // Skip if by chance we generated the same verifier
            if (wrongVerifier === pkce.codeVerifier) return;

            // Property: Wrong verifier must fail verification
            const isValid = await verifyPKCE(wrongVerifier, pkce.codeChallenge);
            expect(isValid).toBe(false);
          }
        ),
        { numRuns: 100 }
      );
    });
  });
});
