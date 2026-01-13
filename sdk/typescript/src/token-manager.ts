/**
 * Token Manager - handles token storage, refresh, and expiration
 * 
 * Provides automatic token refresh with deduplication of concurrent requests.
 * Supports custom storage implementations for different environments.
 * 
 * @packageDocumentation
 */

import type { AccessToken, RefreshToken } from './types/branded.js';
import { createAccessToken, createRefreshToken } from './types/branded.js';
import type { TokenData, TokenStorage, TokenResponse } from './types/tokens.js';
import { isValidTokenData, serializeTokenData, deserializeTokenData } from './types/tokens.js';
import { TokenExpiredError, TokenRefreshError, NetworkError } from './errors/index.js';

/** Default refresh buffer in milliseconds (60 seconds) */
const DEFAULT_REFRESH_BUFFER_MS = 60_000;

/**
 * Options for creating a TokenManager instance.
 */
export interface TokenManagerOptions {
  /** Token storage implementation */
  readonly storage: TokenStorage;
  /** Base URL of the auth server */
  readonly baseUrl: string;
  /** OAuth client ID */
  readonly clientId: string;
  /** OAuth client secret (optional, for confidential clients) */
  readonly clientSecret?: string;
  /** Time in ms before expiration to trigger refresh (default: 60000) */
  readonly refreshBuffer?: number;
}

/**
 * Manages OAuth tokens with automatic refresh and secure storage.
 * 
 * Features:
 * - Automatic token refresh before expiration
 * - Deduplication of concurrent refresh requests
 * - Support for custom storage implementations
 * - Proper error handling with cause chains
 * 
 * @example
 * ```typescript
 * const tokenManager = new TokenManager({
 *   storage: new LocalStorageTokenStorage(),
 *   baseUrl: 'https://auth.example.com',
 *   clientId: 'your-client-id',
 *   refreshBuffer: 60000, // Refresh 60s before expiration
 * });
 * 
 * // Get a valid access token (auto-refreshes if needed)
 * const token = await tokenManager.getAccessToken();
 * ```
 */
export class TokenManager {
  private readonly storage: TokenStorage;
  private readonly baseUrl: string;
  private readonly clientId: string;
  private readonly clientSecret?: string;
  private readonly refreshBufferMs: number;
  private refreshPromise: Promise<TokenData> | null = null;

  /**
   * Create a new TokenManager instance.
   * 
   * @param options - Token manager configuration
   */
  constructor(options: TokenManagerOptions) {
    this.storage = options.storage;
    this.baseUrl = options.baseUrl;
    this.clientId = options.clientId;
    this.clientSecret = options.clientSecret;
    this.refreshBufferMs = options.refreshBuffer ?? DEFAULT_REFRESH_BUFFER_MS;
  }

  /**
   * Get a valid access token, refreshing if necessary.
   * 
   * If the token is about to expire (within the refresh buffer),
   * it will be automatically refreshed before being returned.
   * 
   * @returns Valid access token
   * @throws {@link TokenExpiredError} If no tokens are available
   * @throws {@link TokenRefreshError} If refresh fails
   * 
   * @example
   * ```typescript
   * try {
   *   const token = await tokenManager.getAccessToken();
   *   // Use token for API requests
   * } catch (error) {
   *   if (isTokenExpiredError(error)) {
   *     // Redirect to login
   *   }
   * }
   * ```
   */
  async getAccessToken(): Promise<AccessToken> {
    const tokens = await this.storage.get();

    if (tokens === null) {
      throw new TokenExpiredError('No tokens available');
    }

    if (this.shouldRefresh(tokens)) {
      const refreshed = await this.refreshTokens(tokens);
      return refreshed.accessToken;
    }

    return tokens.accessToken;
  }

  /**
   * Check if token should be refreshed.
   * 
   * Returns true if the token will expire within the refresh buffer period.
   * 
   * @param tokens - Current token data
   * @returns `true` if token should be refreshed
   */
  shouldRefresh(tokens: TokenData): boolean {
    const now = Date.now();
    return tokens.expiresAt - now < this.refreshBufferMs;
  }

  /**
   * Refresh tokens using the refresh token.
   * 
   * Deduplicates concurrent refresh requests - if multiple calls are made
   * while a refresh is in progress, they will all receive the same result.
   * 
   * @param tokens - Current token data with refresh token
   * @returns New token data
   * @throws {@link TokenRefreshError} If no refresh token or refresh fails
   * @throws {@link NetworkError} If network request fails
   */
  async refreshTokens(tokens: TokenData): Promise<TokenData> {
    if (tokens.refreshToken === undefined) {
      throw new TokenRefreshError('No refresh token available');
    }

    if (this.refreshPromise !== null) {
      return this.refreshPromise;
    }

    this.refreshPromise = this.doRefresh(tokens.refreshToken);

    try {
      return await this.refreshPromise;
    } finally {
      this.refreshPromise = null;
    }
  }

  private async doRefresh(refreshToken: RefreshToken): Promise<TokenData> {
    const body = new URLSearchParams({
      grant_type: 'refresh_token',
      refresh_token: refreshToken,
      client_id: this.clientId,
    });

    if (this.clientSecret !== undefined) {
      body.append('client_secret', this.clientSecret);
    }

    let response: Response;
    try {
      response = await fetch(`${this.baseUrl}/oauth/token`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
        body: body.toString(),
      });
    } catch (error) {
      await this.storage.clear();
      throw NetworkError.fromError(error);
    }

    if (!response.ok) {
      await this.storage.clear();
      throw new TokenRefreshError(`Refresh failed: ${response.status.toString()}`);
    }

    const data = (await response.json()) as TokenResponse;
    const tokens = this.tokenResponseToData(data);

    await this.storage.set(tokens);
    return tokens;
  }

  /**
   * Store tokens from a token response.
   * 
   * Converts the OAuth token response to internal format and stores it.
   * 
   * @param response - OAuth token response from server
   * @returns Stored token data
   */
  async storeTokens(response: TokenResponse): Promise<TokenData> {
    const tokens = this.tokenResponseToData(response);
    await this.storage.set(tokens);
    return tokens;
  }

  /**
   * Clear all stored tokens.
   * 
   * Should be called on logout or when tokens are invalidated.
   */
  async clearTokens(): Promise<void> {
    await this.storage.clear();
  }

  /**
   * Check if tokens are available in storage.
   * 
   * @returns `true` if tokens exist, `false` otherwise
   */
  async hasTokens(): Promise<boolean> {
    const tokens = await this.storage.get();
    return tokens !== null;
  }

  private tokenResponseToData(response: TokenResponse): TokenData {
    return {
      accessToken: createAccessToken(response.access_token),
      refreshToken:
        response.refresh_token !== undefined
          ? createRefreshToken(response.refresh_token)
          : undefined,
      expiresAt: Date.now() + response.expires_in * 1000,
      tokenType: 'Bearer',
      scope: response.scope,
    };
  }
}

/**
 * In-memory token storage implementation.
 * 
 * Suitable for testing or short-lived sessions where persistence
 * is not required. Tokens are lost when the page is refreshed.
 * 
 * @example
 * ```typescript
 * const storage = new MemoryTokenStorage();
 * const tokenManager = new TokenManager({
 *   storage,
 *   baseUrl: 'https://auth.example.com',
 *   clientId: 'your-client-id',
 * });
 * ```
 */
export class MemoryTokenStorage implements TokenStorage {
  private tokens: TokenData | null = null;

  async get(): Promise<TokenData | null> {
    return this.tokens;
  }

  async set(tokens: TokenData): Promise<void> {
    this.tokens = tokens;
  }

  async clear(): Promise<void> {
    this.tokens = null;
  }
}

/**
 * LocalStorage-based token storage for browser environments.
 * 
 * Persists tokens across page refreshes and browser sessions.
 * Automatically validates tokens on retrieval.
 * 
 * @example
 * ```typescript
 * const storage = new LocalStorageTokenStorage('my_app_tokens');
 * const tokenManager = new TokenManager({
 *   storage,
 *   baseUrl: 'https://auth.example.com',
 *   clientId: 'your-client-id',
 * });
 * ```
 */
export class LocalStorageTokenStorage implements TokenStorage {
  private readonly key: string;

  /**
   * Create a new LocalStorageTokenStorage instance.
   * 
   * @param key - LocalStorage key to use (default: 'auth_platform_tokens')
   */
  constructor(key = 'auth_platform_tokens') {
    this.key = key;
  }

  async get(): Promise<TokenData | null> {
    const data = localStorage.getItem(this.key);
    if (data === null) {
      return null;
    }

    try {
      const parsed = deserializeTokenData(data);
      if (!isValidTokenData(parsed)) {
        return null;
      }
      return {
        accessToken: createAccessToken(parsed.accessToken),
        refreshToken:
          parsed.refreshToken !== undefined
            ? createRefreshToken(parsed.refreshToken)
            : undefined,
        expiresAt: parsed.expiresAt,
        tokenType: parsed.tokenType,
        scope: parsed.scope,
      };
    } catch {
      return null;
    }
  }

  async set(tokens: TokenData): Promise<void> {
    localStorage.setItem(this.key, serializeTokenData(tokens));
  }

  async clear(): Promise<void> {
    localStorage.removeItem(this.key);
  }
}
