/**
 * Token Manager - handles token storage, refresh, and expiration
 */

import { TokenData, TokenStorage, TokenResponse } from './types';
import { TokenExpiredError, TokenRefreshError } from './errors';

const DEFAULT_REFRESH_BUFFER_MS = 60_000; // Refresh 1 minute before expiry

export class TokenManager {
  private storage: TokenStorage;
  private baseUrl: string;
  private clientId: string;
  private clientSecret?: string;
  private refreshBufferMs: number;
  private refreshPromise: Promise<TokenData> | null = null;

  constructor(
    storage: TokenStorage,
    baseUrl: string,
    clientId: string,
    clientSecret?: string,
    refreshBufferMs = DEFAULT_REFRESH_BUFFER_MS
  ) {
    this.storage = storage;
    this.baseUrl = baseUrl;
    this.clientId = clientId;
    this.clientSecret = clientSecret;
    this.refreshBufferMs = refreshBufferMs;
  }

  /**
   * Get a valid access token, refreshing if necessary
   */
  async getAccessToken(): Promise<string> {
    const tokens = await this.storage.get();

    if (!tokens) {
      throw new TokenExpiredError('No tokens available');
    }

    // Check if token needs refresh
    if (this.shouldRefresh(tokens)) {
      const refreshed = await this.refreshTokens(tokens);
      return refreshed.accessToken;
    }

    return tokens.accessToken;
  }

  /**
   * Check if token should be refreshed
   */
  private shouldRefresh(tokens: TokenData): boolean {
    const now = Date.now();
    return tokens.expiresAt - now < this.refreshBufferMs;
  }

  /**
   * Refresh tokens using refresh token
   * Uses a single promise to prevent concurrent refresh requests
   */
  async refreshTokens(tokens: TokenData): Promise<TokenData> {
    if (!tokens.refreshToken) {
      throw new TokenRefreshError('No refresh token available');
    }

    // Deduplicate concurrent refresh requests
    if (this.refreshPromise) {
      return this.refreshPromise;
    }

    this.refreshPromise = this.doRefresh(tokens.refreshToken);

    try {
      return await this.refreshPromise;
    } finally {
      this.refreshPromise = null;
    }
  }

  private async doRefresh(refreshToken: string): Promise<TokenData> {
    const body = new URLSearchParams({
      grant_type: 'refresh_token',
      refresh_token: refreshToken,
      client_id: this.clientId,
    });

    if (this.clientSecret) {
      body.append('client_secret', this.clientSecret);
    }

    const response = await fetch(`${this.baseUrl}/oauth/token`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/x-www-form-urlencoded',
      },
      body: body.toString(),
    });

    if (!response.ok) {
      await this.storage.clear();
      throw new TokenRefreshError(`Refresh failed: ${response.status}`);
    }

    const data: TokenResponse = await response.json();
    const tokens = this.tokenResponseToData(data);

    await this.storage.set(tokens);
    return tokens;
  }

  /**
   * Store tokens from a token response
   */
  async storeTokens(response: TokenResponse): Promise<TokenData> {
    const tokens = this.tokenResponseToData(response);
    await this.storage.set(tokens);
    return tokens;
  }

  /**
   * Clear stored tokens
   */
  async clearTokens(): Promise<void> {
    await this.storage.clear();
  }

  /**
   * Check if tokens are available
   */
  async hasTokens(): Promise<boolean> {
    const tokens = await this.storage.get();
    return tokens !== null;
  }

  private tokenResponseToData(response: TokenResponse): TokenData {
    return {
      accessToken: response.access_token,
      refreshToken: response.refresh_token,
      expiresAt: Date.now() + response.expires_in * 1000,
      tokenType: response.token_type,
      scope: response.scope,
    };
  }
}

/**
 * In-memory token storage (for testing or short-lived sessions)
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
 * LocalStorage-based token storage (for browser)
 */
export class LocalStorageTokenStorage implements TokenStorage {
  private key: string;

  constructor(key = 'auth_platform_tokens') {
    this.key = key;
  }

  async get(): Promise<TokenData | null> {
    const data = localStorage.getItem(this.key);
    return data ? JSON.parse(data) : null;
  }

  async set(tokens: TokenData): Promise<void> {
    localStorage.setItem(this.key, JSON.stringify(tokens));
  }

  async clear(): Promise<void> {
    localStorage.removeItem(this.key);
  }
}
