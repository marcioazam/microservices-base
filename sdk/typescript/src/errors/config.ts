/**
 * Configuration-related error classes
 */

import { AuthPlatformError, type AuthPlatformErrorOptions } from './base.js';
import { ErrorCode } from './codes.js';

/**
 * Error thrown when configuration is invalid
 */
export class InvalidConfigError extends AuthPlatformError {
  constructor(
    message: string,
    options?: AuthPlatformErrorOptions
  ) {
    super(message, ErrorCode.INVALID_CONFIG, options);
    this.name = 'InvalidConfigError';
  }
}

/**
 * Error thrown when a required field is missing
 */
export class MissingRequiredFieldError extends AuthPlatformError {
  readonly fieldName: string;

  constructor(
    fieldName: string,
    options?: AuthPlatformErrorOptions
  ) {
    super(`Missing required field: ${fieldName}`, ErrorCode.MISSING_REQUIRED_FIELD, options);
    this.name = 'MissingRequiredFieldError';
    this.fieldName = fieldName;
  }

  override toJSON(): Record<string, unknown> {
    return {
      ...super.toJSON(),
      fieldName: this.fieldName,
    };
  }
}
