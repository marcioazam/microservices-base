/**
 * Token-related error classes
 */

import { AuthPlatformError, type AuthPlatformErrorOptions } from './base.js';
import { ErrorCode } from './codes.js';

/**
 * Error thrown when access token has expired
 */
export class TokenExpiredError extends AuthPlatformError {
  constructor(
    message = 'Access token has expired',
    options?: Omit<AuthPlatformErrorOptions, 'statusCode'>
  ) {
    super(message, ErrorCode.TOKEN_EXPIRED, { ...options, statusCode: 401 });
    this.name = 'TokenExpiredError';
  }
}

/**
 * Error thrown when token refresh fails
 */
export class TokenRefreshError extends AuthPlatformError {
  constructor(
    message = 'Failed to refresh token',
    options?: Omit<AuthPlatformErrorOptions, 'statusCode'>
  ) {
    super(message, ErrorCode.TOKEN_REFRESH_FAILED, { ...options, statusCode: 401 });
    this.name = 'TokenRefreshError';
  }
}

/**
 * Error thrown when token validation fails
 */
export class TokenInvalidError extends AuthPlatformError {
  constructor(
    message = 'Invalid token',
    options?: Omit<AuthPlatformErrorOptions, 'statusCode'>
  ) {
    super(message, ErrorCode.TOKEN_INVALID, { ...options, statusCode: 401 });
    this.name = 'TokenInvalidError';
  }
}
