/**
 * Auth Platform TypeScript SDK
 *
 * @packageDocumentation
 */

// Main client
export { AuthPlatformClient } from './client.js';

// Token management
export {
  TokenManager,
  MemoryTokenStorage,
  LocalStorageTokenStorage,
  type TokenManagerOptions,
} from './token-manager.js';

// Passkeys
export { PasskeysClient } from './passkeys.js';

// CAEP
export { CaepSubscriber, type CaepSubscriberConfig } from './caep.js';

// PKCE
export { generatePKCE, verifyPKCE, type PKCEChallenge } from './pkce.js';

// Types
export type {
  // Branded types
  AccessToken,
  RefreshToken,
  CodeVerifier,
  CodeChallenge,
  CredentialId,
  OAuthState,
  CorrelationId,
  // Config
  AuthPlatformConfig,
  // Tokens
  TokenStorage,
  TokenData,
  TokenResponse,
  // Passkeys
  PasskeyCredential,
  PasskeyRegistrationOptions,
  PasskeyAuthenticationOptions,
  PasskeyAuthResult,
  // CAEP
  CaepEvent,
  CaepEventHandler,
  SubjectIdentifier,
  Unsubscribe,
  CaepSubscriberOptions,
  // OAuth
  AuthorizeOptions,
  AuthorizationResult,
} from './types/index.js';

export {
  // Branded type constructors
  createAccessToken,
  createRefreshToken,
  createCodeVerifier,
  createCodeChallenge,
  createCredentialId,
  createOAuthState,
  generateCorrelationId,
  // Config validation
  validateConfig,
  getDefaultConfig,
  // Token utilities
  isValidTokenData,
  serializeTokenData,
  deserializeTokenData,
  // CAEP event types
  CaepEventType,
} from './types/index.js';

// Errors
export {
  ErrorCode,
  AuthPlatformError,
  TokenExpiredError,
  TokenRefreshError,
  TokenInvalidError,
  NetworkError,
  TimeoutError,
  RateLimitError,
  InvalidConfigError,
  MissingRequiredFieldError,
  PasskeyError,
  PasskeyNotSupportedError,
  PasskeyCancelledError,
  PasskeyRegistrationError,
  PasskeyAuthError,
  CaepConnectionError,
  CaepParseError,
  type AuthPlatformErrorOptions,
} from './errors/index.js';

// Error type guards
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
} from './errors/index.js';

// Utilities
export {
  base64UrlEncode,
  base64UrlDecode,
  bufferToBase64Url,
  base64UrlToBuffer,
  timingSafeEqual,
} from './utils/index.js';
