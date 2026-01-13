/**
 * Passkeys Client - WebAuthn wrapper with platform-specific handling.
 * 
 * Provides a high-level API for passkey (WebAuthn) authentication,
 * supporting both platform authenticators (Touch ID, Face ID, Windows Hello)
 * and cross-platform authenticators (security keys).
 * 
 * @see {@link https://www.w3.org/TR/webauthn-2/ | WebAuthn Level 2}
 * @packageDocumentation
 */

import type { AccessToken } from './types/branded.js';
import type {
  PasskeyCredential,
  PasskeyRegistrationOptions,
  PasskeyAuthenticationOptions,
  PasskeyAuthResult,
} from './types/passkeys.js';
import {
  PasskeyNotSupportedError,
  PasskeyCancelledError,
  PasskeyRegistrationError,
  PasskeyAuthError,
} from './errors/index.js';
import { bufferToBase64Url, base64UrlToBuffer } from './utils/base64url.js';

/**
 * Client for passkey (WebAuthn) authentication.
 * 
 * Handles the complexity of WebAuthn credential creation and assertion,
 * including proper encoding/decoding of binary data and error handling.
 * 
 * @example Registration
 * ```typescript
 * const client = new PasskeysClient(
 *   'https://auth.example.com',
 *   () => tokenManager.getAccessToken()
 * );
 * 
 * if (PasskeysClient.isSupported()) {
 *   const credential = await client.register({
 *     deviceName: 'My MacBook',
 *     authenticatorAttachment: 'platform',
 *   });
 *   console.log('Registered:', credential.id);
 * }
 * ```
 * 
 * @example Authentication
 * ```typescript
 * const result = await client.authenticate({
 *   mediation: 'optional',
 * });
 * console.log('Authenticated:', result.userId);
 * ```
 */
export class PasskeysClient {
  private readonly baseUrl: string;
  private readonly getAccessToken: () => Promise<AccessToken>;

  /**
   * Create a new PasskeysClient instance.
   * 
   * @param baseUrl - Base URL of the auth server
   * @param getAccessToken - Function to get current access token
   */
  constructor(baseUrl: string, getAccessToken: () => Promise<AccessToken>) {
    this.baseUrl = baseUrl;
    this.getAccessToken = getAccessToken;
  }

  /**
   * Check if passkeys are supported on this device/browser.
   * 
   * Checks for the presence of the WebAuthn API (PublicKeyCredential).
   * 
   * @returns `true` if WebAuthn is available, `false` otherwise
   * 
   * @example
   * ```typescript
   * if (PasskeysClient.isSupported()) {
   *   // Show passkey options
   * } else {
   *   // Fall back to password authentication
   * }
   * ```
   */
  static isSupported(): boolean {
    return (
      typeof window !== 'undefined' &&
      window.PublicKeyCredential !== undefined &&
      typeof window.PublicKeyCredential === 'function'
    );
  }

  /**
   * Check if a platform authenticator is available.
   * 
   * Platform authenticators include Touch ID, Face ID, Windows Hello, etc.
   * 
   * @returns `true` if platform authenticator is available
   * 
   * @example
   * ```typescript
   * if (await PasskeysClient.isPlatformAuthenticatorAvailable()) {
   *   // Offer biometric authentication
   * }
   * ```
   */
  static async isPlatformAuthenticatorAvailable(): Promise<boolean> {
    if (!this.isSupported()) return false;
    return PublicKeyCredential.isUserVerifyingPlatformAuthenticatorAvailable();
  }

  /**
   * Register a new passkey for the current user.
   * 
   * Creates a new WebAuthn credential and registers it with the server.
   * 
   * @param options - Registration options
   * @returns Registered passkey credential information
   * @throws {@link PasskeyNotSupportedError} If WebAuthn is not supported
   * @throws {@link PasskeyCancelledError} If user cancels the operation
   * @throws {@link PasskeyRegistrationError} If registration fails
   * 
   * @example
   * ```typescript
   * const credential = await client.register({
   *   deviceName: 'My MacBook Pro',
   *   authenticatorAttachment: 'platform', // or 'cross-platform' for security keys
   * });
   * ```
   */
  async register(options: PasskeyRegistrationOptions = {}): Promise<PasskeyCredential> {
    if (!PasskeysClient.isSupported()) {
      throw new PasskeyNotSupportedError();
    }

    const token = await this.getAccessToken();
    const publicKeyOptions = await this.fetchRegistrationOptions(token, options);
    const credential = await this.createCredential(publicKeyOptions);
    return this.verifyRegistration(token, credential, options.deviceName);
  }

  /**
   * Authenticate using a passkey.
   * 
   * Prompts the user to select and use a registered passkey.
   * 
   * @param options - Authentication options
   * @returns Authentication result with tokens
   * @throws {@link PasskeyNotSupportedError} If WebAuthn is not supported
   * @throws {@link PasskeyCancelledError} If user cancels the operation
   * @throws {@link PasskeyAuthError} If authentication fails
   * 
   * @example
   * ```typescript
   * const result = await client.authenticate({
   *   mediation: 'optional', // 'silent', 'optional', 'required', or 'conditional'
   * });
   * ```
   */
  async authenticate(options: PasskeyAuthenticationOptions = {}): Promise<PasskeyAuthResult> {
    if (!PasskeysClient.isSupported()) {
      throw new PasskeyNotSupportedError();
    }

    const publicKeyOptions = await this.fetchAuthenticationOptions(options);
    const credential = await this.getCredential(publicKeyOptions, options.mediation);
    return this.verifyAuthentication(credential);
  }

  /**
   * List all registered passkeys for the current user.
   * 
   * @returns Array of registered passkey credentials
   * @throws {@link PasskeyAuthError} If the request fails
   * 
   * @example
   * ```typescript
   * const passkeys = await client.list();
   * passkeys.forEach(pk => {
   *   console.log(`${pk.deviceName} - Created: ${pk.createdAt}`);
   * });
   * ```
   */
  async list(): Promise<PasskeyCredential[]> {
    const token = await this.getAccessToken();
    const response = await fetch(`${this.baseUrl}/passkeys`, {
      headers: { Authorization: `Bearer ${token}` },
    });

    if (!response.ok) {
      throw new PasskeyAuthError('Failed to list passkeys');
    }

    return response.json() as Promise<PasskeyCredential[]>;
  }

  /**
   * Delete a registered passkey.
   * 
   * @param passkeyId - ID of the passkey to delete
   * @throws {@link PasskeyAuthError} If deletion fails
   * 
   * @example
   * ```typescript
   * await client.delete('passkey-id-123');
   * ```
   */
  async delete(passkeyId: string): Promise<void> {
    const token = await this.getAccessToken();
    const response = await fetch(`${this.baseUrl}/passkeys/${passkeyId}`, {
      method: 'DELETE',
      headers: { Authorization: `Bearer ${token}` },
    });

    if (!response.ok) {
      throw new PasskeyAuthError('Failed to delete passkey');
    }
  }

  private async fetchRegistrationOptions(
    token: AccessToken,
    options: PasskeyRegistrationOptions
  ): Promise<PublicKeyCredentialCreationOptions> {
    const response = await fetch(`${this.baseUrl}/passkeys/register/begin`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify({
        authenticatorAttachment: options.authenticatorAttachment,
      }),
    });

    if (!response.ok) {
      throw new PasskeyRegistrationError('Failed to get registration options');
    }

    const data = (await response.json()) as Record<string, unknown>;
    return this.decodePublicKeyOptions(data);
  }

  private async createCredential(
    options: PublicKeyCredentialCreationOptions
  ): Promise<PublicKeyCredential> {
    try {
      const credential = await navigator.credentials.create({ publicKey: options });
      if (credential === null) {
        throw new PasskeyRegistrationError('No credential returned');
      }
      return credential as PublicKeyCredential;
    } catch (error) {
      if (error instanceof DOMException && error.name === 'NotAllowedError') {
        throw new PasskeyCancelledError();
      }
      throw new PasskeyRegistrationError(
        error instanceof Error ? error.message : 'Registration failed',
        { cause: error }
      );
    }
  }

  private async verifyRegistration(
    token: AccessToken,
    credential: PublicKeyCredential,
    deviceName?: string
  ): Promise<PasskeyCredential> {
    const attestation = credential.response as AuthenticatorAttestationResponse;
    const response = await fetch(`${this.baseUrl}/passkeys/register/finish`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify({
        id: credential.id,
        rawId: bufferToBase64Url(credential.rawId),
        type: credential.type,
        clientDataJSON: bufferToBase64Url(attestation.clientDataJSON),
        attestationObject: bufferToBase64Url(attestation.attestationObject),
        transports: attestation.getTransports?.() ?? [],
        deviceName,
      }),
    });

    if (!response.ok) {
      throw new PasskeyRegistrationError('Failed to verify registration');
    }

    return response.json() as Promise<PasskeyCredential>;
  }

  private async fetchAuthenticationOptions(
    options: PasskeyAuthenticationOptions
  ): Promise<PublicKeyCredentialRequestOptions> {
    const response = await fetch(`${this.baseUrl}/passkeys/authenticate/begin`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ mediation: options.mediation }),
    });

    if (!response.ok) {
      throw new PasskeyAuthError('Failed to get authentication options');
    }

    const data = (await response.json()) as Record<string, unknown>;
    return this.decodePublicKeyOptions(data) as PublicKeyCredentialRequestOptions;
  }

  private async getCredential(
    options: PublicKeyCredentialRequestOptions,
    mediation?: string
  ): Promise<PublicKeyCredential> {
    try {
      const credential = await navigator.credentials.get({
        publicKey: options,
        mediation: mediation as CredentialMediationRequirement | undefined,
      });
      if (credential === null) {
        throw new PasskeyAuthError('No credential returned');
      }
      return credential as PublicKeyCredential;
    } catch (error) {
      if (error instanceof DOMException && error.name === 'NotAllowedError') {
        throw new PasskeyCancelledError();
      }
      throw new PasskeyAuthError(
        error instanceof Error ? error.message : 'Authentication failed',
        { cause: error }
      );
    }
  }

  private async verifyAuthentication(
    credential: PublicKeyCredential
  ): Promise<PasskeyAuthResult> {
    const assertion = credential.response as AuthenticatorAssertionResponse;
    const response = await fetch(`${this.baseUrl}/passkeys/authenticate/finish`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        id: credential.id,
        rawId: bufferToBase64Url(credential.rawId),
        type: credential.type,
        clientDataJSON: bufferToBase64Url(assertion.clientDataJSON),
        authenticatorData: bufferToBase64Url(assertion.authenticatorData),
        signature: bufferToBase64Url(assertion.signature),
        userHandle: assertion.userHandle !== null
          ? bufferToBase64Url(assertion.userHandle)
          : null,
      }),
    });

    if (!response.ok) {
      throw new PasskeyAuthError('Failed to verify authentication');
    }

    return response.json() as Promise<PasskeyAuthResult>;
  }

  private decodePublicKeyOptions(
    options: Record<string, unknown>
  ): PublicKeyCredentialCreationOptions {
    return {
      ...options,
      challenge: base64UrlToBuffer(options.challenge as string),
      user: options.user !== undefined
        ? {
            ...(options.user as Record<string, unknown>),
            id: base64UrlToBuffer((options.user as Record<string, unknown>).id as string),
          }
        : undefined,
      excludeCredentials: (options.excludeCredentials as Array<Record<string, unknown>> | undefined)?.map(
        (c) => ({ ...c, id: base64UrlToBuffer(c.id as string) })
      ),
      allowCredentials: (options.allowCredentials as Array<Record<string, unknown>> | undefined)?.map(
        (c) => ({ ...c, id: base64UrlToBuffer(c.id as string) })
      ),
    } as PublicKeyCredentialCreationOptions;
  }
}
