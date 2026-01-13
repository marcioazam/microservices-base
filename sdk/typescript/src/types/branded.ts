/**
 * Branded types for sensitive values.
 * 
 * Branded types prevent accidental misuse of string values by adding
 * a compile-time "brand" that makes them incompatible with plain strings.
 * This provides type safety without runtime overhead.
 * 
 * @example
 * ```typescript
 * // These are incompatible at compile time:
 * const access: AccessToken = createAccessToken('token1');
 * const refresh: RefreshToken = createRefreshToken('token2');
 * 
 * // Error: Type 'AccessToken' is not assignable to type 'RefreshToken'
 * const wrong: RefreshToken = access;
 * ```
 * 
 * @packageDocumentation
 */

declare const __brand: unique symbol;

/**
 * Branded type utility.
 * Creates a nominal type by adding a unique brand property.
 * @typeParam T - Base type
 * @typeParam B - Brand identifier string
 */
type Brand<T, B extends string> = T & { readonly [__brand]: B };

/**
 * OAuth access token.
 * Used for authenticating API requests.
 */
export type AccessToken = Brand<string, 'AccessToken'>;

/**
 * OAuth refresh token.
 * Used for obtaining new access tokens.
 */
export type RefreshToken = Brand<string, 'RefreshToken'>;

/**
 * PKCE code verifier.
 * Secret value used during OAuth token exchange.
 * Must be 43-128 characters per RFC 7636.
 */
export type CodeVerifier = Brand<string, 'CodeVerifier'>;

/**
 * PKCE code challenge.
 * SHA-256 hash of the code verifier, base64url encoded.
 */
export type CodeChallenge = Brand<string, 'CodeChallenge'>;

/**
 * WebAuthn credential ID.
 * Unique identifier for a registered passkey.
 */
export type CredentialId = Brand<string, 'CredentialId'>;

/**
 * OAuth state parameter.
 * Used for CSRF protection during authorization flow.
 */
export type OAuthState = Brand<string, 'OAuthState'>;

/**
 * Correlation ID for error tracking.
 * Used to trace errors across distributed systems.
 */
export type CorrelationId = Brand<string, 'CorrelationId'>;

// Type-safe constructors with validation

/**
 * Create a branded AccessToken.
 * 
 * @param value - Raw token string
 * @returns Branded access token
 * @throws Error if value is invalid (empty or not a string)
 * 
 * @example
 * ```typescript
 * const token = createAccessToken('eyJhbGciOiJSUzI1NiIs...');
 * ```
 */
export function createAccessToken(value: string): AccessToken {
  if (typeof value !== 'string' || value.length === 0) {
    throw new Error('Invalid access token: must be a non-empty string');
  }
  return value as AccessToken;
}

/**
 * Create a branded RefreshToken.
 * 
 * @param value - Raw token string
 * @returns Branded refresh token
 * @throws Error if value is invalid (empty or not a string)
 * 
 * @example
 * ```typescript
 * const token = createRefreshToken('dGhpcyBpcyBhIHJlZnJlc2g...');
 * ```
 */
export function createRefreshToken(value: string): RefreshToken {
  if (typeof value !== 'string' || value.length === 0) {
    throw new Error('Invalid refresh token: must be a non-empty string');
  }
  return value as RefreshToken;
}

/**
 * Create a branded CodeVerifier.
 * 
 * PKCE verifiers must be 43-128 characters per RFC 7636.
 * 
 * @param value - Code verifier string
 * @returns Branded code verifier
 * @throws Error if value length is not 43-128 characters
 * 
 * @example
 * ```typescript
 * const verifier = createCodeVerifier('dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk');
 * ```
 */
export function createCodeVerifier(value: string): CodeVerifier {
  if (typeof value !== 'string' || value.length < 43 || value.length > 128) {
    throw new Error('Invalid code verifier: must be 43-128 characters');
  }
  return value as CodeVerifier;
}

/**
 * Create a branded CodeChallenge.
 * 
 * @param value - Code challenge string (base64url-encoded SHA-256 hash)
 * @returns Branded code challenge
 * @throws Error if value is empty
 * 
 * @example
 * ```typescript
 * const challenge = createCodeChallenge('E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM');
 * ```
 */
export function createCodeChallenge(value: string): CodeChallenge {
  if (typeof value !== 'string' || value.length === 0) {
    throw new Error('Invalid code challenge: must be a non-empty string');
  }
  return value as CodeChallenge;
}

/**
 * Create a branded CredentialId.
 * 
 * @param value - Credential ID string
 * @returns Branded credential ID
 * @throws Error if value is empty
 */
export function createCredentialId(value: string): CredentialId {
  if (typeof value !== 'string' || value.length === 0) {
    throw new Error('Invalid credential ID: must be a non-empty string');
  }
  return value as CredentialId;
}

/**
 * Create a branded OAuthState.
 * 
 * @param value - State parameter string
 * @returns Branded OAuth state
 * @throws Error if value is empty
 */
export function createOAuthState(value: string): OAuthState {
  if (typeof value !== 'string' || value.length === 0) {
    throw new Error('Invalid OAuth state: must be a non-empty string');
  }
  return value as OAuthState;
}

/**
 * Generate a random correlation ID.
 * 
 * Creates a 32-character hexadecimal string using cryptographically
 * secure random values.
 * 
 * @returns Branded correlation ID
 * 
 * @example
 * ```typescript
 * const correlationId = generateCorrelationId();
 * // e.g., 'a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6'
 * ```
 */
export function generateCorrelationId(): CorrelationId {
  const array = new Uint8Array(16);
  crypto.getRandomValues(array);
  const hex = Array.from(array, (b) => b.toString(16).padStart(2, '0')).join('');
  return hex as CorrelationId;
}
