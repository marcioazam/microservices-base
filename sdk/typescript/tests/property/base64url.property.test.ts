/**
 * Property-based tests for base64url encoding
 *
 * **Feature: typescript-sdk-modernization**
 * **Property 15: Base64URL Round-Trip**
 */

import { describe, it, expect } from 'vitest';
import * as fc from 'fast-check';
import {
  base64UrlEncode,
  base64UrlDecode,
  bufferToBase64Url,
  base64UrlToBuffer,
} from '../../src/utils/base64url.js';

/**
 * **Property 15: Base64URL Round-Trip**
 * For any Uint8Array, encoding and decoding produces equivalent data.
 * **Validates: Requirements 7.7**
 */
describe('Property 15: Base64URL Round-Trip', () => {
  it('round-trips Uint8Array correctly', () => {
    fc.assert(
      fc.property(
        fc.uint8Array({ minLength: 0, maxLength: 1000 }),
        (original) => {
          const encoded = base64UrlEncode(original);
          const decoded = base64UrlDecode(encoded);

          expect(decoded.length).toBe(original.length);
          for (let i = 0; i < original.length; i++) {
            expect(decoded[i]).toBe(original[i]);
          }
        }
      ),
      { numRuns: 100 }
    );
  });

  it('round-trips ArrayBuffer correctly', () => {
    fc.assert(
      fc.property(
        fc.uint8Array({ minLength: 0, maxLength: 1000 }),
        (original) => {
          const buffer = original.buffer.slice(
            original.byteOffset,
            original.byteOffset + original.byteLength
          );
          const encoded = bufferToBase64Url(buffer);
          const decoded = base64UrlToBuffer(encoded);

          const originalView = new Uint8Array(buffer);
          const decodedView = new Uint8Array(decoded);

          expect(decodedView.length).toBe(originalView.length);
          for (let i = 0; i < originalView.length; i++) {
            expect(decodedView[i]).toBe(originalView[i]);
          }
        }
      ),
      { numRuns: 100 }
    );
  });

  it('produces valid base64url characters only', () => {
    const base64urlRegex = /^[A-Za-z0-9_-]*$/;

    fc.assert(
      fc.property(
        fc.uint8Array({ minLength: 1, maxLength: 100 }),
        (data) => {
          const encoded = base64UrlEncode(data);
          expect(base64urlRegex.test(encoded)).toBe(true);
        }
      ),
      { numRuns: 100 }
    );
  });

  it('does not include padding characters', () => {
    fc.assert(
      fc.property(
        fc.uint8Array({ minLength: 1, maxLength: 100 }),
        (data) => {
          const encoded = base64UrlEncode(data);
          expect(encoded).not.toContain('=');
        }
      ),
      { numRuns: 100 }
    );
  });

  it('handles empty input', () => {
    const empty = new Uint8Array(0);
    const encoded = base64UrlEncode(empty);
    const decoded = base64UrlDecode(encoded);

    expect(encoded).toBe('');
    expect(decoded.length).toBe(0);
  });
});
