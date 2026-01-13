/**
 * PKCE (Proof Key for Code Exchange) implementation.
 * 
 * OAuth 2.1 compliant implementation that always uses the S256 challenge method.
 * The plain method is intentionally not supported as it provides no security benefit.
 * 
 * @see {@link https://datatracker.ietf.org/doc/html/rfc7636 | RFC 7636 - PKCE}
 * @packageDocumentation
 */

import type { CodeVerifier, CodeChallenge } from './types/branded.js';
import { createCodeVerifier, createCodeChallenge } from './types/branded.js';
import { base64UrlEncode } from './utils/base64url.js';
import { sha256, timingSafeEqual } from './utils/crypto.js';

/**
 * PKCE challenge pair containing verifier and challenge.
 * 
 * The verifier is kept secret and sent during token exchange.
 * The challenge is sent during authorization request.
 */
export interface PKCEChallenge {
  /** Secret code verifier (43-128 characters) */
  readonly codeVerifier: CodeVerifier;
  /** SHA-256 hash of verifier, base64url encoded */
  readonly codeChallenge: CodeChallenge;
  /** Challenge method (always 'S256' for security) */
  readonly codeChallengeMethod: 'S256';
}

/**
 * Generate a cryptographically random code verifier.
 * 
 * Per RFC 7636, the verifier must be 43-128 characters from the
 * unreserved character set [A-Z] / [a-z] / [0-9] / "-" / "." / "_" / "~".
 * 
 * @returns Branded CodeVerifier string
 * @internal
 */
function generateCodeVerifier(): CodeVerifier {
  const array = new Uint8Array(32);
  crypto.getRandomValues(array);
  const encoded = base64UrlEncode(array);
  return createCodeVerifier(encoded);
}

/**
 * Generate code challenge from verifier using SHA-256.
 * 
 * @param verifier - Code verifier to hash
 * @returns Base64url-encoded SHA-256 hash
 * @internal
 */
async function generateCodeChallengeFromVerifier(
  verifier: CodeVerifier
): Promise<CodeChallenge> {
  const digest = await sha256(verifier);
  const encoded = base64UrlEncode(digest);
  return createCodeChallenge(encoded);
}

/**
 * Generate a PKCE challenge pair for OAuth 2.1 authorization.
 * 
 * Always uses the S256 (SHA-256) challenge method as required by OAuth 2.1.
 * The plain method is not supported for security reasons.
 * 
 * @returns PKCE challenge containing verifier, challenge, and method
 * 
 * @example
 * ```typescript
 * const pkce = await generatePKCE();
 * 
 * // Use challenge in authorization request
 * const authUrl = new URL('https://auth.example.com/authorize');
 * authUrl.searchParams.set('code_challenge', pkce.codeChallenge);
 * authUrl.searchParams.set('code_challenge_method', pkce.codeChallengeMethod);
 * 
 * // Store verifier for token exchange
 * sessionStorage.setItem('pkce_verifier', pkce.codeVerifier);
 * ```
 */
export async function generatePKCE(): Promise<PKCEChallenge> {
  const codeVerifier = generateCodeVerifier();
  const codeChallenge = await generateCodeChallengeFromVerifier(codeVerifier);

  return {
    codeVerifier,
    codeChallenge,
    codeChallengeMethod: 'S256',
  };
}

/**
 * Verify that a code verifier matches a code challenge.
 * 
 * Uses timing-safe comparison to prevent timing attacks.
 * This is typically used server-side during token exchange.
 * 
 * @param codeVerifier - The original code verifier
 * @param codeChallenge - The challenge to verify against
 * @returns `true` if verifier matches challenge, `false` otherwise
 * 
 * @example
 * ```typescript
 * const isValid = await verifyPKCE(codeVerifier, codeChallenge);
 * if (!isValid) {
 *   throw new Error('PKCE verification failed');
 * }
 * ```
 */
export async function verifyPKCE(
  codeVerifier: CodeVerifier,
  codeChallenge: CodeChallenge
): Promise<boolean> {
  const computed = await generateCodeChallengeFromVerifier(codeVerifier);
  return timingSafeEqual(computed, codeChallenge);
}
