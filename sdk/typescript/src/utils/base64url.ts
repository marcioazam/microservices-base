/**
 * Base64URL encoding/decoding utilities
 */

/**
 * Encode Uint8Array to base64url string (no padding)
 */
export function base64UrlEncode(buffer: Uint8Array): string {
  const base64 = btoa(String.fromCharCode(...buffer));
  return base64.replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
}

/**
 * Decode base64url string to Uint8Array
 */
export function base64UrlDecode(base64url: string): Uint8Array {
  const base64 = base64url.replace(/-/g, '+').replace(/_/g, '/');
  const padding = '='.repeat((4 - (base64.length % 4)) % 4);
  const binary = atob(base64 + padding);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) {
    bytes[i] = binary.charCodeAt(i);
  }
  return bytes;
}

/**
 * Encode ArrayBuffer to base64url string
 */
export function bufferToBase64Url(buffer: ArrayBuffer): string {
  return base64UrlEncode(new Uint8Array(buffer));
}

/**
 * Decode base64url string to ArrayBuffer
 */
export function base64UrlToBuffer(base64url: string): ArrayBuffer {
  return base64UrlDecode(base64url).buffer;
}
