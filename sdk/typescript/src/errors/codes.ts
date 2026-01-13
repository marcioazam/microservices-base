/**
 * Error codes for Auth Platform SDK.
 * 
 * Using const object instead of enum for better tree-shaking support.
 * Each code represents a specific error condition that can be handled
 * programmatically.
 * 
 * @packageDocumentation
 */

/**
 * Error codes for programmatic error handling.
 * 
 * @example Using error codes
 * ```typescript
 * import { ErrorCode, isAuthPlatformError } from '@auth-platform/sdk';
 * 
 * try {
 *   await client.getAccessToken();
 * } catch (error) {
 *   if (isAuthPlatformError(error)) {
 *     switch (error.code) {
 *       case ErrorCode.TOKEN_EXPIRED:
 *         // Redirect to login
 *         break;
 *       case ErrorCode.RATE_LIMITED:
 *         // Wait and retry
 *         break;
 *       default:
 *         // Handle other errors
 *     }
 *   }
 * }
 * ```
 */
export const ErrorCode = {
  // Token errors
  /** Access token has expired */
  TOKEN_EXPIRED: 'TOKEN_EXPIRED',
  /** Token refresh operation failed */
  TOKEN_REFRESH_FAILED: 'TOKEN_REFRESH_FAILED',
  /** Token is malformed or invalid */
  TOKEN_INVALID: 'TOKEN_INVALID',

  // Network errors
  /** Network request failed */
  NETWORK_ERROR: 'NETWORK_ERROR',
  /** Request timed out */
  TIMEOUT: 'TIMEOUT',

  // Configuration errors
  /** Configuration is invalid */
  INVALID_CONFIG: 'INVALID_CONFIG',
  /** Required configuration field is missing */
  MISSING_REQUIRED_FIELD: 'MISSING_REQUIRED_FIELD',

  // Rate limiting
  /** Request was rate limited (HTTP 429) */
  RATE_LIMITED: 'RATE_LIMITED',

  // Passkey errors
  /** WebAuthn is not supported on this device/browser */
  PASSKEY_NOT_SUPPORTED: 'PASSKEY_NOT_SUPPORTED',
  /** User cancelled the passkey operation */
  PASSKEY_CANCELLED: 'PASSKEY_CANCELLED',
  /** Passkey registration failed */
  PASSKEY_REGISTRATION_FAILED: 'PASSKEY_REGISTRATION_FAILED',
  /** Passkey authentication failed */
  PASSKEY_AUTH_FAILED: 'PASSKEY_AUTH_FAILED',

  // CAEP errors
  /** Failed to connect to CAEP event stream */
  CAEP_CONNECTION_FAILED: 'CAEP_CONNECTION_FAILED',
  /** Failed to parse CAEP event */
  CAEP_PARSE_ERROR: 'CAEP_PARSE_ERROR',
} as const;

/**
 * Union type of all error codes.
 */
export type ErrorCode = (typeof ErrorCode)[keyof typeof ErrorCode];
