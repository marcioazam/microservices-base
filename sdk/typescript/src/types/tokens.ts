/**
 * Token-related types and interfaces.
 * 
 * Defines the structure for OAuth tokens and storage implementations.
 * 
 * @packageDocumentation
 */

import type { AccessToken, RefreshToken } from './branded.js';

/**
 * Token data stored by the SDK.
 * 
 * Contains the access token, optional refresh token, and metadata
 * about token expiration and scope.
 */
export interface TokenData {
  /** OAuth access token (branded type) */
  readonly accessToken: AccessToken;
  /** OAuth refresh token (branded type, optional) */
  readonly refreshToken?: RefreshToken;
  /** Token expiration timestamp in milliseconds since epoch */
  readonly expiresAt: number;
  /** Token type (always 'Bearer') */
  readonly tokenType: 'Bearer';
  /** Space-separated list of granted scopes */
  readonly scope?: string;
}

/**
 * Token storage interface for custom implementations.
 * 
 * Implement this interface to provide custom token storage
 * (e.g., secure storage, IndexedDB, etc.).
 * 
 * @example Custom implementation
 * ```typescript
 * class SecureTokenStorage implements TokenStorage {
 *   async get(): Promise<TokenData | null> {
 *     const encrypted = await secureStore.get('tokens');
 *     return encrypted ? decrypt(encrypted) : null;
 *   }
 *   
 *   async set(tokens: TokenData): Promise<void> {
 *     await secureStore.set('tokens', encrypt(tokens));
 *   }
 *   
 *   async clear(): Promise<void> {
 *     await secureStore.delete('tokens');
 *   }
 * }
 * ```
 */
export interface TokenStorage {
  /**
   * Retrieve stored tokens.
   * @returns Token data or null if not stored
   */
  get(): Promise<TokenData | null>;
  /**
   * Store tokens.
   * @param tokens - Token data to store
   */
  set(tokens: TokenData): Promise<void>;
  /**
   * Clear stored tokens.
   */
  clear(): Promise<void>;
}

/**
 * OAuth token response from the authorization server.
 * 
 * This is the raw response format from the /oauth/token endpoint.
 * Use tokenResponseToData() to convert to TokenData.
 */
export interface TokenResponse {
  /** Access token string */
  readonly access_token: string;
  /** Token type (always 'Bearer') */
  readonly token_type: 'Bearer';
  /** Token lifetime in seconds */
  readonly expires_in: number;
  /** Refresh token string (optional) */
  readonly refresh_token?: string;
  /** Space-separated list of granted scopes */
  readonly scope?: string;
}
