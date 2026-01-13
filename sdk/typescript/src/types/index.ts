/**
 * Type exports
 */

// Branded types
export type {
  AccessToken,
  RefreshToken,
  CodeVerifier,
  CodeChallenge,
  CredentialId,
  OAuthState,
  CorrelationId,
} from './branded.js';

export {
  createAccessToken,
  createRefreshToken,
  createCodeVerifier,
  createCodeChallenge,
  createCredentialId,
  createOAuthState,
  generateCorrelationId,
} from './branded.js';

// Config types
export type { AuthPlatformConfig } from './config.js';
export { validateConfig, getDefaultConfig } from './config.js';

// Token types
export type {
  TokenStorage,
  TokenData,
  TokenResponse,
  SerializedTokenData,
} from './tokens.js';

export {
  isValidTokenData,
  serializeTokenData,
  deserializeTokenData,
} from './tokens.js';

// Passkey types
export type {
  PasskeyCredential,
  PasskeyRegistrationOptions,
  PasskeyAuthenticationOptions,
  PasskeyAuthResult,
} from './passkeys.js';

// CAEP types
export { CaepEventType } from './caep.js';
export type {
  CaepEvent,
  CaepEventHandler,
  SubjectIdentifier,
  SubjectFormat,
  Unsubscribe,
  CaepSubscriberOptions,
} from './caep.js';

// OAuth types
export type { AuthorizeOptions, AuthorizationResult } from './oauth.js';
