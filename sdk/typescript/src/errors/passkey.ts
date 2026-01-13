/**
 * Passkey-related error classes
 */

import { AuthPlatformError, type AuthPlatformErrorOptions } from './base.js';
import { ErrorCode } from './codes.js';

/**
 * Base error for passkey operations
 */
export class PasskeyError extends AuthPlatformError {
  constructor(
    message: string,
    code: ErrorCode,
    options?: AuthPlatformErrorOptions
  ) {
    super(message, code, options);
    this.name = 'PasskeyError';
  }
}

/**
 * Error thrown when passkeys are not supported
 */
export class PasskeyNotSupportedError extends PasskeyError {
  constructor(options?: AuthPlatformErrorOptions) {
    super(
      'Passkeys are not supported on this device/browser',
      ErrorCode.PASSKEY_NOT_SUPPORTED,
      options
    );
    this.name = 'PasskeyNotSupportedError';
  }
}

/**
 * Error thrown when user cancels passkey operation
 */
export class PasskeyCancelledError extends PasskeyError {
  constructor(options?: AuthPlatformErrorOptions) {
    super(
      'Passkey operation was cancelled by user',
      ErrorCode.PASSKEY_CANCELLED,
      options
    );
    this.name = 'PasskeyCancelledError';
  }
}

/**
 * Error thrown when passkey registration fails
 */
export class PasskeyRegistrationError extends PasskeyError {
  constructor(
    message = 'Passkey registration failed',
    options?: AuthPlatformErrorOptions
  ) {
    super(message, ErrorCode.PASSKEY_REGISTRATION_FAILED, options);
    this.name = 'PasskeyRegistrationError';
  }
}

/**
 * Error thrown when passkey authentication fails
 */
export class PasskeyAuthError extends PasskeyError {
  constructor(
    message = 'Passkey authentication failed',
    options?: AuthPlatformErrorOptions
  ) {
    super(message, ErrorCode.PASSKEY_AUTH_FAILED, options);
    this.name = 'PasskeyAuthError';
  }
}
