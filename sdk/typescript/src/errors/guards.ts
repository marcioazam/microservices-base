/**
 * Type guards for error classes
 */

import { AuthPlatformError } from './base.js';
import { TokenExpiredError, TokenRefreshError, TokenInvalidError } from './token.js';
import { NetworkError, TimeoutError, RateLimitError } from './network.js';
import { InvalidConfigError, MissingRequiredFieldError } from './config.js';
import {
  PasskeyError,
  PasskeyNotSupportedError,
  PasskeyCancelledError,
  PasskeyRegistrationError,
  PasskeyAuthError,
} from './passkey.js';
import { CaepConnectionError, CaepParseError } from './caep.js';

/** Type guard for AuthPlatformError */
export function isAuthPlatformError(error: unknown): error is AuthPlatformError {
  return error instanceof AuthPlatformError;
}

/** Type guard for TokenExpiredError */
export function isTokenExpiredError(error: unknown): error is TokenExpiredError {
  return error instanceof TokenExpiredError;
}

/** Type guard for TokenRefreshError */
export function isTokenRefreshError(error: unknown): error is TokenRefreshError {
  return error instanceof TokenRefreshError;
}

/** Type guard for TokenInvalidError */
export function isTokenInvalidError(error: unknown): error is TokenInvalidError {
  return error instanceof TokenInvalidError;
}

/** Type guard for NetworkError */
export function isNetworkError(error: unknown): error is NetworkError {
  return error instanceof NetworkError;
}

/** Type guard for TimeoutError */
export function isTimeoutError(error: unknown): error is TimeoutError {
  return error instanceof TimeoutError;
}

/** Type guard for RateLimitError */
export function isRateLimitError(error: unknown): error is RateLimitError {
  return error instanceof RateLimitError;
}

/** Type guard for InvalidConfigError */
export function isInvalidConfigError(error: unknown): error is InvalidConfigError {
  return error instanceof InvalidConfigError;
}

/** Type guard for MissingRequiredFieldError */
export function isMissingRequiredFieldError(error: unknown): error is MissingRequiredFieldError {
  return error instanceof MissingRequiredFieldError;
}

/** Type guard for PasskeyError (base class) */
export function isPasskeyError(error: unknown): error is PasskeyError {
  return error instanceof PasskeyError;
}

/** Type guard for PasskeyNotSupportedError */
export function isPasskeyNotSupportedError(error: unknown): error is PasskeyNotSupportedError {
  return error instanceof PasskeyNotSupportedError;
}

/** Type guard for PasskeyCancelledError */
export function isPasskeyCancelledError(error: unknown): error is PasskeyCancelledError {
  return error instanceof PasskeyCancelledError;
}

/** Type guard for PasskeyRegistrationError */
export function isPasskeyRegistrationError(error: unknown): error is PasskeyRegistrationError {
  return error instanceof PasskeyRegistrationError;
}

/** Type guard for PasskeyAuthError */
export function isPasskeyAuthError(error: unknown): error is PasskeyAuthError {
  return error instanceof PasskeyAuthError;
}

/** Type guard for CaepConnectionError */
export function isCaepConnectionError(error: unknown): error is CaepConnectionError {
  return error instanceof CaepConnectionError;
}

/** Type guard for CaepParseError */
export function isCaepParseError(error: unknown): error is CaepParseError {
  return error instanceof CaepParseError;
}
