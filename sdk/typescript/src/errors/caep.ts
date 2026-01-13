/**
 * CAEP-related error classes
 */

import { AuthPlatformError, type AuthPlatformErrorOptions } from './base.js';
import { ErrorCode } from './codes.js';

/**
 * Error thrown when CAEP connection fails
 */
export class CaepConnectionError extends AuthPlatformError {
  constructor(
    message = 'CAEP connection failed',
    options?: AuthPlatformErrorOptions
  ) {
    super(message, ErrorCode.CAEP_CONNECTION_FAILED, options);
    this.name = 'CaepConnectionError';
  }
}

/**
 * Error thrown when CAEP event parsing fails
 */
export class CaepParseError extends AuthPlatformError {
  constructor(
    message = 'Failed to parse CAEP event',
    options?: AuthPlatformErrorOptions
  ) {
    super(message, ErrorCode.CAEP_PARSE_ERROR, options);
    this.name = 'CaepParseError';
  }
}
