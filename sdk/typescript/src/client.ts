/**
 * Auth Platform TypeScript SDK Client
 * 
 * Main entry point for the Auth Platform SDK. Provides OAuth 2.1 authentication
 * with PKCE, token management, passkey support, and CAEP event subscriptions.
 * 
 * @packageDocumentation
 * @module @auth-platform/sdk
 */

import type { AccessToken } from './types/branded.js';
import type { AuthPlatformConfig } from './types/config.js';
import { validateConfig } from './types/config.js';
import type { TokenData, TokenResponse } from './types/tokens.js';
import type { AuthorizeOptions } from './types/oauth.js';
import type {
  PasskeyCredential,
  PasskeyRegistrationOptions,
  PasskeyAuthenticationOptions,
  PasskeyAuthResult,
} from './types/passkeys.js';
import type { CaepEvent, CaepEventHandler, Unsubscribe } from './types/caep.js';
import { TokenManager, MemoryTokenStorage } from './token-manager.js';
import { PasskeysClient } from './passkeys.js';
import { CaepSubscriber } from './caep.js';
import { generatePKCE, type PKCEChallenge } from './pkce.js';
import {
  InvalidConfigError,
  NetworkError,
  RateLimitError,
  AuthPlatformError,
  TimeoutError,
} from './errors/index.js';
import { timingSafeEqual } from './utils/crypto.js';

/**
 * Main client for Auth Platform SDK.
 * 
 * Provides a unified interface for:
 * - OAuth 2.1 authorization with PKCE
 * - Token management with automatic refresh
 * - Passkey (WebAuthn) authentication
 * - CAEP security event subscriptions
 * 
 * @example Basic usage
 * ```typescript
 * import { AuthPlatformClient } from '@auth-platform/sdk';
 * 
 * const client = new AuthPlatformClient({
 *   baseUrl: 'https://auth.example.com',
 *   clientId: 'your-client-id',
 * });
 * 
 * // Start OAuth flow
 * const authUrl = await client.authorize({
 *   redirectUri: 'https://app.example.com/callback',
 *   scopes: ['openid', 'profile'],
 * });
 * 
 * // Handle callback
 * const tokens = await client.handleCallback(code, redirectUri);
 * ```
 * 
 * @example With passkeys
 * ```typescript
 * // Check if passkeys are supported
 * if (AuthPlatformClient.isPasskeysSupported()) {
 *   const credential = await client.registerPasskey({
 *     deviceName: 'My MacBook',
 *   });
 * }
 * ```
 * 
 * @example With CAEP events
 * ```typescript
 * const unsubscribe = client.onSecurityEvent((event) => {
 *   if (event.type === 'session-revoked') {
 *     // Handle session revocation
 *     client.logout();
 *   }
 * });
 * ```
 */
export class AuthPlatformClient {
  private readonly config: AuthPlatformConfig;
  private readonly tokenManager: TokenManager;
  private readonly passkeys: PasskeysClient;
  private readonly caep: CaepSubscriber;
  private pendingPKCE: PKCEChallenge | null = null;
  private pendingState: string | null = null;

  /**
   * Create a new AuthPlatformClient instance.
   * 
   * @param config - Client configuration options
   * @throws {@link InvalidConfigError} If configuration is invalid
   * @throws {@link MissingRequiredFieldError} If required fields are missing
   * 
   * @example
   * ```typescript
   * const client = new AuthPlatformClient({
   *   baseUrl: 'https://auth.example.com',
   *   clientId: 'your-client-id',
   *   scopes: ['openid', 'profile', 'email'],
   *   timeout: 30000,
   *   refreshBuffer: 60000,
   * });
   * ```
   */
  constructor(config: AuthPlatformConfig) {
    this.config = validateConfig(config);

    const storage = this.config.storage ?? new MemoryTokenStorage();
    this.tokenManager = new TokenManager({
      storage,
      baseUrl: this.config.baseUrl,
      clientId: this.config.clientId,
      clientSecret: this.config.clientSecret,
      refreshBuffer: this.config.refreshBuffer,
    });

    this.passkeys = new PasskeysClient(
      this.config.baseUrl,
      () => this.getAccessToken()
    );

    this.caep = new CaepSubscriber({
      baseUrl: this.config.baseUrl,
      getAccessToken: () => this.getAccessToken(),
    });
  }

  /**
   * Start OAuth 2.1 authorization flow with PKCE.
   * 
   * Generates a PKCE challenge and returns the authorization URL.
   * The user should be redirected to this URL to authenticate.
   * 
   * @param options - Authorization options
   * @returns Authorization URL to redirect the user to
   * 
   * @example
   * ```typescript
   * const authUrl = await client.authorize({
   *   redirectUri: 'https://app.example.com/callback',
   *   scopes: ['openid', 'profile'],
   *   state: 'random-state-value',
   *   prompt: 'consent',
   * });
   * 
   * // Redirect user to authUrl
   * window.location.href = authUrl;
   * ```
   */
  async authorize(options: AuthorizeOptions = {}): Promise<string> {
    this.pendingPKCE = await generatePKCE();
    this.pendingState = options.state ?? null;

    const params = new URLSearchParams({
      response_type: 'code',
      client_id: this.config.clientId,
      code_challenge: this.pendingPKCE.codeChallenge,
      code_challenge_method: this.pendingPKCE.codeChallengeMethod,
    });

    if (options.redirectUri !== undefined) {
      params.set('redirect_uri', options.redirectUri);
    }

    const scopes = options.scopes ?? this.config.scopes;
    if (scopes !== undefined && scopes.length > 0) {
      params.set('scope', scopes.join(' '));
    }

    if (options.state !== undefined) {
      params.set('state', options.state);
    }

    if (options.prompt !== undefined) {
      params.set('prompt', options.prompt);
    }

    return `${this.config.baseUrl}/oauth/authorize?${params.toString()}`;
  }

  /**
   * Validate state parameter to prevent CSRF attacks.
   * 
   * Uses timing-safe comparison to prevent timing attacks.
   * 
   * @param receivedState - State parameter received from callback
   * @returns `true` if state matches, `false` otherwise
   * 
   * @example
   * ```typescript
   * const isValid = client.validateState(receivedState);
   * if (!isValid) {
   *   throw new Error('CSRF attack detected');
   * }
   * ```
   */
  validateState(receivedState: string | null): boolean {
    if (this.pendingState === null && receivedState === null) {
      return true;
    }
    if (this.pendingState === null || receivedState === null) {
      return false;
    }
    return timingSafeEqual(this.pendingState, receivedState);
  }

  /**
   * Exchange authorization code for tokens.
   * 
   * Completes the OAuth 2.1 authorization flow by exchanging the
   * authorization code for access and refresh tokens.
   * 
   * @param code - Authorization code from callback
   * @param redirectUri - Redirect URI used in authorization request
   * @param state - State parameter from callback (optional)
   * @returns Token data including access token and optional refresh token
   * @throws {@link InvalidConfigError} If no pending PKCE challenge or state mismatch
   * @throws {@link NetworkError} If token exchange fails
   * @throws {@link RateLimitError} If rate limited
   * 
   * @example
   * ```typescript
   * // In your callback handler
   * const urlParams = new URLSearchParams(window.location.search);
   * const code = urlParams.get('code');
   * const state = urlParams.get('state');
   * 
   * const tokens = await client.handleCallback(
   *   code,
   *   'https://app.example.com/callback',
   *   state
   * );
   * ```
   */
  async handleCallback(
    code: string,
    redirectUri?: string,
    state?: string
  ): Promise<TokenData> {
    if (this.pendingPKCE === null) {
      throw new InvalidConfigError('No pending PKCE challenge. Call authorize() first.');
    }

    if (!this.validateState(state ?? null)) {
      this.clearPendingAuth();
      throw new InvalidConfigError('State parameter mismatch. Possible CSRF attack.');
    }

    const body = new URLSearchParams({
      grant_type: 'authorization_code',
      code,
      client_id: this.config.clientId,
      code_verifier: this.pendingPKCE.codeVerifier,
    });

    if (redirectUri !== undefined) {
      body.set('redirect_uri', redirectUri);
    }

    if (this.config.clientSecret !== undefined) {
      body.set('client_secret', this.config.clientSecret);
    }

    try {
      const response = await this.fetch('/oauth/token', {
        method: 'POST',
        headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
        body: body.toString(),
      });

      const tokenResponse = (await response.json()) as TokenResponse;
      this.clearPendingAuth();

      return this.tokenManager.storeTokens(tokenResponse);
    } catch (error) {
      this.clearPendingAuth();
      throw error;
    }
  }

  /**
   * Get access token, refreshing if necessary.
   * 
   * Automatically refreshes the token if it's about to expire
   * (within the configured refresh buffer).
   * 
   * @returns Valid access token
   * @throws {@link TokenExpiredError} If no tokens available
   * @throws {@link TokenRefreshError} If refresh fails
   * 
   * @example
   * ```typescript
   * const token = await client.getAccessToken();
   * 
   * // Use token in API requests
   * const response = await fetch('/api/data', {
   *   headers: { Authorization: `Bearer ${token}` },
   * });
   * ```
   */
  async getAccessToken(): Promise<AccessToken> {
    return this.tokenManager.getAccessToken();
  }

  /**
   * Check if user is authenticated.
   * 
   * @returns `true` if tokens are available, `false` otherwise
   * 
   * @example
   * ```typescript
   * if (await client.isAuthenticated()) {
   *   // User is logged in
   * } else {
   *   // Redirect to login
   * }
   * ```
   */
  async isAuthenticated(): Promise<boolean> {
    return this.tokenManager.hasTokens();
  }

  /**
   * Logout and clear all tokens.
   * 
   * Clears stored tokens and disconnects from CAEP event stream.
   * 
   * @example
   * ```typescript
   * await client.logout();
   * // Redirect to login page
   * window.location.href = '/login';
   * ```
   */
  async logout(): Promise<void> {
    await this.tokenManager.clearTokens();
    this.caep.disconnect();
  }

  // Passkeys methods

  /**
   * Register a new passkey for the current user.
   * 
   * @param options - Registration options
   * @returns Registered passkey credential
   * @throws {@link PasskeyNotSupportedError} If passkeys are not supported
   * @throws {@link PasskeyCancelledError} If user cancels the operation
   * @throws {@link PasskeyRegistrationError} If registration fails
   * 
   * @example
   * ```typescript
   * const credential = await client.registerPasskey({
   *   deviceName: 'My MacBook Pro',
   *   authenticatorAttachment: 'platform',
   * });
   * console.log('Registered passkey:', credential.id);
   * ```
   */
  async registerPasskey(options?: PasskeyRegistrationOptions): Promise<PasskeyCredential> {
    return this.passkeys.register(options);
  }

  /**
   * Authenticate using a passkey.
   * 
   * @param options - Authentication options
   * @returns Authentication result with tokens
   * @throws {@link PasskeyNotSupportedError} If passkeys are not supported
   * @throws {@link PasskeyCancelledError} If user cancels the operation
   * @throws {@link PasskeyAuthError} If authentication fails
   * 
   * @example
   * ```typescript
   * const result = await client.authenticateWithPasskey({
   *   mediation: 'optional',
   * });
   * console.log('Authenticated as:', result.userId);
   * ```
   */
  async authenticateWithPasskey(options?: PasskeyAuthenticationOptions): Promise<PasskeyAuthResult> {
    return this.passkeys.authenticate(options);
  }

  /**
   * List all registered passkeys for the current user.
   * 
   * @returns Array of registered passkey credentials
   * @throws {@link PasskeyAuthError} If listing fails
   * 
   * @example
   * ```typescript
   * const passkeys = await client.listPasskeys();
   * passkeys.forEach(pk => console.log(pk.deviceName, pk.createdAt));
   * ```
   */
  async listPasskeys(): Promise<PasskeyCredential[]> {
    return this.passkeys.list();
  }

  /**
   * Delete a registered passkey.
   * 
   * @param passkeyId - ID of the passkey to delete
   * @throws {@link PasskeyAuthError} If deletion fails
   * 
   * @example
   * ```typescript
   * await client.deletePasskey('passkey-id-123');
   * ```
   */
  async deletePasskey(passkeyId: string): Promise<void> {
    return this.passkeys.delete(passkeyId);
  }

  /**
   * Check if passkeys are supported on this device/browser.
   * 
   * @returns `true` if WebAuthn is available, `false` otherwise
   * 
   * @example
   * ```typescript
   * if (AuthPlatformClient.isPasskeysSupported()) {
   *   // Show passkey registration option
   * }
   * ```
   */
  static isPasskeysSupported(): boolean {
    return PasskeysClient.isSupported();
  }

  // CAEP methods

  /**
   * Subscribe to security events via CAEP.
   * 
   * Establishes an SSE connection to receive real-time security events
   * such as session revocations and credential changes.
   * 
   * @param handler - Event handler function
   * @returns Unsubscribe function to stop receiving events
   * 
   * @example
   * ```typescript
   * const unsubscribe = client.onSecurityEvent((event) => {
   *   switch (event.type) {
   *     case 'session-revoked':
   *       console.log('Session revoked:', event.reason);
   *       client.logout();
   *       break;
   *     case 'credential-change':
   *       console.log('Credentials changed');
   *       break;
   *   }
   * });
   * 
   * // Later, to stop receiving events:
   * unsubscribe();
   * ```
   */
  onSecurityEvent(handler: CaepEventHandler): Unsubscribe {
    return this.caep.subscribe(handler);
  }

  // Private helpers

  private clearPendingAuth(): void {
    this.pendingPKCE = null;
    this.pendingState = null;
  }

  private async fetch(path: string, init?: RequestInit): Promise<Response> {
    const url = `${this.config.baseUrl}${path}`;
    const timeout = this.config.timeout ?? 30_000;

    const controller = new AbortController();
    const timeoutId = setTimeout(() => { controller.abort(); }, timeout);

    try {
      const response = await fetch(url, { ...init, signal: controller.signal });

      if (response.status === 429) {
        const retryAfter = response.headers.get('Retry-After');
        throw new RateLimitError(
          'Rate limit exceeded',
          retryAfter !== null ? parseInt(retryAfter, 10) : undefined
        );
      }

      if (!response.ok) {
        throw new AuthPlatformError(
          `Request failed: ${response.status.toString()}`,
          'NETWORK_ERROR',
          { statusCode: response.status }
        );
      }

      return response;
    } catch (error) {
      if (error instanceof AuthPlatformError) {
        throw error;
      }
      if (error instanceof Error && error.name === 'AbortError') {
        throw new TimeoutError('Request timeout');
      }
      throw NetworkError.fromError(error);
    } finally {
      clearTimeout(timeoutId);
    }
  }
}
