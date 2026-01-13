/**
 * Cryptographic utilities
 */

import { base64UrlEncode } from './base64url.js';

/**
 * Generate cryptographically secure random bytes
 */
export function getRandomBytes(length: number): Uint8Array {
  const array = new Uint8Array(length);
  crypto.getRandomValues(array);
  return array;
}

/**
 * Generate a random string of specified length using base64url alphabet
 */
export function generateRandomString(length: number): string {
  // Generate enough bytes to produce the desired length after encoding
  const bytesNeeded = Math.ceil((length * 3) / 4);
  const bytes = getRandomBytes(bytesNeeded);
  return base64UrlEncode(bytes).slice(0, length);
}

/**
 * Compute SHA-256 hash of data
 */
export async function sha256(data: string): Promise<Uint8Array> {
  const encoder = new TextEncoder();
  const encoded = encoder.encode(data);
  const digest = await crypto.subtle.digest('SHA-256', encoded);
  return new Uint8Array(digest);
}

/**
 * Timing-safe string comparison
 * Prevents timing attacks by always comparing all characters
 */
export function timingSafeEqual(a: string, b: string): boolean {
  if (a.length !== b.length) {
    return false;
  }

  let result = 0;
  for (let i = 0; i < a.length; i++) {
    result |= a.charCodeAt(i) ^ b.charCodeAt(i);
  }

  return result === 0;
}
