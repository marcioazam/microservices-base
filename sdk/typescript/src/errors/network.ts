/**
 * Network-related error classes
 */

import { AuthPlatformError, type AuthPlatformErrorOptions } from './base.js';
import { ErrorCode } from './codes.js';

/**
 * Error thrown when a network request fails
 */
export class NetworkError extends AuthPlatformError {
  constructor(
    message = 'Network request failed',
    options?: AuthPlatformErrorOptions
  ) {
    super(message, ErrorCode.NETWORK_ERROR, options);
    this.name = 'NetworkError';
  }

  /**
   * Create a NetworkError from an unknown caught error
   */
  static fromError(error: unknown, correlationId?: string | undefined): NetworkError {
    const message = error instanceof Error ? error.message : 'Network request failed';
    return new NetworkError(message, {
      cause: error,
      ...(correlationId !== undefined && { correlationId }),
    });
  }
}

/**
 * Error thrown when a request times out
 */
export class TimeoutError extends AuthPlatformError {
  constructor(
    message = 'Request timeout',
    options?: AuthPlatformErrorOptions
  ) {
    super(message, ErrorCode.TIMEOUT, options);
    this.name = 'TimeoutError';
  }
}

/**
 * Error thrown when rate limit is exceeded
 */
export class RateLimitError extends AuthPlatformError {
  readonly retryAfter: number | undefined;

  constructor(
    message = 'Rate limit exceeded',
    retryAfter?: number | undefined,
    options?: Omit<AuthPlatformErrorOptions, 'statusCode'>
  ) {
    super(message, ErrorCode.RATE_LIMITED, { ...options, statusCode: 429 });
    this.name = 'RateLimitError';
    this.retryAfter = retryAfter;
  }

  override toJSON(): Record<string, unknown> {
    return {
      ...super.toJSON(),
      retryAfter: this.retryAfter,
    };
  }
}
