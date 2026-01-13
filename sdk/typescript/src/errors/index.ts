/**
 * Error module exports
 */

// Error codes
export { ErrorCode } from './codes.js';

// Base error
export { AuthPlatformError, type AuthPlatformErrorOptions } from './base.js';

// Token errors
export { TokenExpiredError, TokenRefreshError, TokenInvalidError } from './token.js';

// Network errors
export { NetworkError, TimeoutError, RateLimitError } from './network.js';

// Config errors
export { InvalidConfigError, MissingRequiredFieldError } from './config.js';

// Passkey errors
export {
  PasskeyError,
  PasskeyNotSupportedError,
  PasskeyCancelledError,
  PasskeyRegistrationError,
  PasskeyAuthError,
} from './passkey.js';

// CAEP errors
export { CaepConnectionError, CaepParseError } from './caep.js';

// Type guards
export {
  isAuthPlatformError,
  isTokenExpiredError,
  isTokenRefreshError,
  isTokenInvalidError,
  isNetworkError,
  isTimeoutError,
  isRateLimitError,
  isInvalidConfigError,
  isMissingRequiredFieldError,
  isPasskeyError,
  isPasskeyNotSupportedError,
  isPasskeyCancelledError,
  isPasskeyRegistrationError,
  isPasskeyAuthError,
  isCaepConnectionError,
  isCaepParseError,
} from './guards.js';
