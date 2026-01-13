/**
 * Base error class for Auth Platform SDK.
 * 
 * All SDK errors extend this class, providing consistent error handling
 * with error codes, correlation IDs, and cause chains.
 * 
 * @packageDocumentation
 */

import type { ErrorCode } from './codes.js';

/**
 * Options for creating an AuthPlatformError.
 */
export interface AuthPlatformErrorOptions {
  /** HTTP status code (if applicable) */
  readonly statusCode?: number | undefined;
  /** Correlation ID for distributed tracing */
  readonly correlationId?: string | undefined;
  /** Original error that caused this error */
  readonly cause?: unknown;
}

/**
 * Base error class with enhanced properties for debugging and tracing.
 * 
 * All SDK errors extend this class, providing:
 * - Error codes for programmatic error handling
 * - Correlation IDs for distributed tracing
 * - Cause chains for error context
 * - Timestamps for debugging
 * 
 * @example Catching and handling errors
 * ```typescript
 * try {
 *   await client.getAccessToken();
 * } catch (error) {
 *   if (error instanceof AuthPlatformError) {
 *     console.log('Error code:', error.code);
 *     console.log('Correlation ID:', error.correlationId);
 *     console.log('Timestamp:', error.timestamp);
 *     
 *     if (error.cause) {
 *       console.log('Caused by:', error.cause);
 *     }
 *   }
 * }
 * ```
 * 
 * @example Creating custom errors
 * ```typescript
 * throw new AuthPlatformError('Custom error', 'NETWORK_ERROR', {
 *   statusCode: 500,
 *   correlationId: 'abc-123',
 *   cause: originalError,
 * });
 * ```
 */
export class AuthPlatformError extends Error {
  /** Error code for programmatic handling */
  readonly code: ErrorCode;
  /** HTTP status code (if applicable) */
  readonly statusCode: number | undefined;
  /** Correlation ID for distributed tracing */
  readonly correlationId: string | undefined;
  /** Timestamp when the error was created */
  readonly timestamp: Date;

  /**
   * Create a new AuthPlatformError.
   * 
   * @param message - Human-readable error message
   * @param code - Error code from ErrorCode enum
   * @param options - Additional error options
   */
  constructor(
    message: string,
    code: ErrorCode,
    options?: AuthPlatformErrorOptions
  ) {
    super(message, { cause: options?.cause });
    this.name = 'AuthPlatformError';
    this.code = code;
    this.statusCode = options?.statusCode;
    this.correlationId = options?.correlationId;
    this.timestamp = new Date();

    // Maintains proper stack trace for where error was thrown (V8 engines)
    if ('captureStackTrace' in Error && typeof Error.captureStackTrace === 'function') {
      Error.captureStackTrace(this, this.constructor);
    }
  }

  /**
   * Create a JSON-serializable representation of the error.
   * 
   * Useful for logging and API responses.
   * 
   * @returns Plain object with error properties
   * 
   * @example
   * ```typescript
   * const error = new AuthPlatformError('Failed', 'NETWORK_ERROR');
   * console.log(JSON.stringify(error.toJSON()));
   * // {"name":"AuthPlatformError","message":"Failed","code":"NETWORK_ERROR",...}
   * ```
   */
  toJSON(): Record<string, unknown> {
    return {
      name: this.name,
      message: this.message,
      code: this.code,
      statusCode: this.statusCode,
      correlationId: this.correlationId,
      timestamp: this.timestamp.toISOString(),
    };
  }
}
