/**
 * Auth Platform TypeScript SDK Client
 */

import {
  AuthPlatformConfig,
  AuthorizeOptions,
  TokenResponse,
  TokenData,
  Unsubscribe,
  CaepEvent,
  PasskeyCredential,
  PasskeyRegistrationOptions,
  PasskeyAuthenticationOptions,
} from './types';
import { TokenManager, MemoryTokenStorage } from './token-manager';
import { PasskeysClient } from './passkeys';
import { CaepSubscriber } from './caep';
import { generatePKCE, PKCEChallenge } from './pkce';
import {
  InvalidConfigError,
  NetworkError,
  RateLimitError,
  AuthPlatformError,
} from './errors';

export class AuthPlatformClient {
  private config: AuthPlatformConfig;
  private tokenManager: TokenManager;
  private passkeys: PasskeysClient;
  private caep: CaepSubscriber;
  private pendingPKCE: PKCEChallenge | null = null;

  constructor(config: AuthPlatformConfig) {
    this.validateConfig(config);
    this.config = config;

    const storage = config.storage || new MemoryTokenStorage();
    this.tokenManager = new TokenManager(
      storage,
      config.baseUrl,
      config.clientId,
      config.clientSecret
    );

    this.passkeys = new PasskeysClient(config.baseUrl, () => this.getAccessToken());
    this.caep = new CaepSubscriber(config.baseUrl, () => this.getAccessToken());
  }

  private validateConfig(config: AuthPlatformConfig): void {
    if (!config.baseUrl) {
      throw new InvalidConfigError('baseUrl is required');
    }
    if (!config.clientId) {
      throw new InvalidConfigError('clientId is required');
    }
  }

  /**
   * Start OAuth 2.1 authorization flow with PKCE
   * Returns the authorization URL to redirect the user to
   */
  async authorize(options: AuthorizeOptions = {}): Promise<string> {
    // Generate PKCE challenge (always S256 per OAuth 2.1)
    this.pendingPKCE = await generatePKCE();

    const params = new URLSearchParams({
      response_type: 'code',
      client_id: this.config.clientId,
      code_challenge: this.pendingPKCE.codeChallenge,
      code_challenge_method: this.pendingPKCE.codeChallengeMethod,
    });

    if (options.redirectUri) {
      params.set('redirect_uri', options.redirectUri);
    }

    if (options.scopes?.length || this.config.scopes?.length) {
      params.set('scope', (options.scopes || this.config.scopes || []).join(' '));
    }

    if (options.state) {
      params.set('state', options.state);
    }

    if (options.prompt) {
      params.set('prompt', options.prompt);
    }

    return `${this.config.baseUrl}/oauth/authorize?${params.toString()}`;
  }

  /**
   * Exchange authorization code for tokens
   */
  async handleCallback(code: string, redirectUri?: string): Promise<TokenData> {
    if (!this.pendingPKCE) {
      throw new InvalidConfigError('No pending PKCE challenge. Call authorize() first.');
    }

    const body = new URLSearchParams({
      grant_type: 'authorization_code',
      code,
      client_id: this.config.clientId,
      code_verifier: this.pendingPKCE.codeVerifier,
    });

    if (redirectUri) {
      body.set('redirect_uri', redirectUri);
    }

    if (this.config.clientSecret) {
      body.set('client_secret', this.config.clientSecret);
    }

    const response = await this.fetch('/oauth/token', {
      method: 'POST',
      headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
      body: body.toString(),
    });

    const tokenResponse: TokenResponse = await response.json();
    this.pendingPKCE = null;

    return this.tokenManager.storeTokens(tokenResponse);
  }

  /**
   * Get access token (refreshing if necessary)
   */
  async getAccessToken(): Promise<string> {
    return this.tokenManager.getAccessToken();
  }

  /**
   * Check if user is authenticated
   */
  async isAuthenticated(): Promise<boolean> {
    return this.tokenManager.hasTokens();
  }

  /**
   * Logout - clear tokens
   */
  async logout(): Promise<void> {
    await this.tokenManager.clearTokens();
    this.caep.disconnect();
  }

  // Passkeys methods

  /**
   * Register a new passkey
   */
  async registerPasskey(options?: PasskeyRegistrationOptions): Promise<PasskeyCredential> {
    return this.passkeys.register(options);
  }

  /**
   * Authenticate with passkey
   */
  async authenticateWithPasskey(
    options?: PasskeyAuthenticationOptions
  ): Promise<{ token: string }> {
    return this.passkeys.authenticate(options);
  }

  /**
   * List registered passkeys
   */
  async listPasskeys(): Promise<PasskeyCredential[]> {
    return this.passkeys.list();
  }

  /**
   * Delete a passkey
   */
  async deletePasskey(passkeyId: string): Promise<void> {
    return this.passkeys.delete(passkeyId);
  }

  /**
   * Check if passkeys are supported
   */
  static isPasskeysSupported(): boolean {
    return PasskeysClient.isSupported();
  }

  // CAEP methods

  /**
   * Subscribe to security events
   */
  onSecurityEvent(handler: (event: CaepEvent) => void): Unsubscribe {
    return this.caep.subscribe(handler);
  }

  // HTTP helper

  private async fetch(path: string, init?: RequestInit): Promise<Response> {
    const url = `${this.config.baseUrl}${path}`;
    const timeout = this.config.timeout || 30000;

    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), timeout);

    try {
      const response = await fetch(url, {
        ...init,
        signal: controller.signal,
      });

      if (response.status === 429) {
        const retryAfter = response.headers.get('Retry-After');
        throw new RateLimitError(
          'Rate limit exceeded',
          retryAfter ? parseInt(retryAfter, 10) : undefined
        );
      }

      if (!response.ok) {
        throw new AuthPlatformError(
          `Request failed: ${response.status}`,
          'REQUEST_FAILED',
          response.status
        );
      }

      return response;
    } catch (error) {
      if (error instanceof AuthPlatformError) {
        throw error;
      }
      if (error instanceof Error && error.name === 'AbortError') {
        throw new NetworkError('Request timeout');
      }
      throw new NetworkError('Network request failed');
    } finally {
      clearTimeout(timeoutId);
    }
  }
}
