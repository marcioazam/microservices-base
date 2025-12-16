/**
 * Passkeys Client - WebAuthn wrapper with platform-specific handling
 */

import {
  PasskeyCredential,
  PasskeyRegistrationOptions,
  PasskeyAuthenticationOptions,
} from './types';
import {
  PasskeyError,
  PasskeyNotSupportedError,
  PasskeyCancelledError,
} from './errors';

export class PasskeysClient {
  private baseUrl: string;
  private getAccessToken: () => Promise<string>;

  constructor(baseUrl: string, getAccessToken: () => Promise<string>) {
    this.baseUrl = baseUrl;
    this.getAccessToken = getAccessToken;
  }

  /**
   * Check if passkeys are supported on this device/browser
   */
  static isSupported(): boolean {
    return (
      typeof window !== 'undefined' &&
      window.PublicKeyCredential !== undefined &&
      typeof window.PublicKeyCredential === 'function'
    );
  }

  /**
   * Check if platform authenticator is available
   */
  static async isPlatformAuthenticatorAvailable(): Promise<boolean> {
    if (!this.isSupported()) return false;
    return PublicKeyCredential.isUserVerifyingPlatformAuthenticatorAvailable();
  }

  /**
   * Register a new passkey
   */
  async register(options: PasskeyRegistrationOptions = {}): Promise<PasskeyCredential> {
    if (!PasskeysClient.isSupported()) {
      throw new PasskeyNotSupportedError();
    }

    const token = await this.getAccessToken();

    // Get registration options from server
    const optionsResponse = await fetch(`${this.baseUrl}/passkeys/register/begin`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify({
        authenticatorAttachment: options.authenticatorAttachment,
      }),
    });

    if (!optionsResponse.ok) {
      throw new PasskeyError('Failed to get registration options', 'REGISTRATION_OPTIONS_FAILED');
    }

    const publicKeyOptions = await optionsResponse.json();

    // Create credential
    let credential: PublicKeyCredential;
    try {
      credential = (await navigator.credentials.create({
        publicKey: this.decodePublicKeyOptions(publicKeyOptions),
      })) as PublicKeyCredential;
    } catch (error) {
      if (error instanceof DOMException) {
        if (error.name === 'NotAllowedError') {
          throw new PasskeyCancelledError();
        }
        throw new PasskeyError(error.message, error.name);
      }
      throw error;
    }

    if (!credential) {
      throw new PasskeyError('No credential returned', 'NO_CREDENTIAL');
    }

    // Send attestation to server
    const attestationResponse = credential.response as AuthenticatorAttestationResponse;
    const verifyResponse = await fetch(`${this.baseUrl}/passkeys/register/finish`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify({
        id: credential.id,
        rawId: this.bufferToBase64Url(credential.rawId),
        type: credential.type,
        clientDataJSON: this.bufferToBase64Url(attestationResponse.clientDataJSON),
        attestationObject: this.bufferToBase64Url(attestationResponse.attestationObject),
        transports: attestationResponse.getTransports?.() || [],
        deviceName: options.deviceName,
      }),
    });

    if (!verifyResponse.ok) {
      throw new PasskeyError('Failed to verify registration', 'REGISTRATION_VERIFY_FAILED');
    }

    return verifyResponse.json();
  }

  /**
   * Authenticate with a passkey
   */
  async authenticate(options: PasskeyAuthenticationOptions = {}): Promise<{ token: string }> {
    if (!PasskeysClient.isSupported()) {
      throw new PasskeyNotSupportedError();
    }

    // Get authentication options from server
    const optionsResponse = await fetch(`${this.baseUrl}/passkeys/authenticate/begin`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ mediation: options.mediation }),
    });

    if (!optionsResponse.ok) {
      throw new PasskeyError('Failed to get authentication options', 'AUTH_OPTIONS_FAILED');
    }

    const publicKeyOptions = await optionsResponse.json();

    // Get credential
    let credential: PublicKeyCredential;
    try {
      credential = (await navigator.credentials.get({
        publicKey: this.decodePublicKeyOptions(publicKeyOptions),
        mediation: options.mediation as CredentialMediationRequirement,
      })) as PublicKeyCredential;
    } catch (error) {
      if (error instanceof DOMException) {
        if (error.name === 'NotAllowedError') {
          throw new PasskeyCancelledError();
        }
        throw new PasskeyError(error.message, error.name);
      }
      throw error;
    }

    if (!credential) {
      throw new PasskeyError('No credential returned', 'NO_CREDENTIAL');
    }

    // Send assertion to server
    const assertionResponse = credential.response as AuthenticatorAssertionResponse;
    const verifyResponse = await fetch(`${this.baseUrl}/passkeys/authenticate/finish`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        id: credential.id,
        rawId: this.bufferToBase64Url(credential.rawId),
        type: credential.type,
        clientDataJSON: this.bufferToBase64Url(assertionResponse.clientDataJSON),
        authenticatorData: this.bufferToBase64Url(assertionResponse.authenticatorData),
        signature: this.bufferToBase64Url(assertionResponse.signature),
        userHandle: assertionResponse.userHandle
          ? this.bufferToBase64Url(assertionResponse.userHandle)
          : null,
      }),
    });

    if (!verifyResponse.ok) {
      throw new PasskeyError('Failed to verify authentication', 'AUTH_VERIFY_FAILED');
    }

    return verifyResponse.json();
  }

  /**
   * List registered passkeys
   */
  async list(): Promise<PasskeyCredential[]> {
    const token = await this.getAccessToken();
    const response = await fetch(`${this.baseUrl}/passkeys`, {
      headers: { Authorization: `Bearer ${token}` },
    });

    if (!response.ok) {
      throw new PasskeyError('Failed to list passkeys', 'LIST_FAILED');
    }

    return response.json();
  }

  /**
   * Delete a passkey
   */
  async delete(passkeyId: string): Promise<void> {
    const token = await this.getAccessToken();
    const response = await fetch(`${this.baseUrl}/passkeys/${passkeyId}`, {
      method: 'DELETE',
      headers: { Authorization: `Bearer ${token}` },
    });

    if (!response.ok) {
      throw new PasskeyError('Failed to delete passkey', 'DELETE_FAILED');
    }
  }

  private decodePublicKeyOptions(options: any): PublicKeyCredentialCreationOptions {
    return {
      ...options,
      challenge: this.base64UrlToBuffer(options.challenge),
      user: options.user
        ? { ...options.user, id: this.base64UrlToBuffer(options.user.id) }
        : undefined,
      excludeCredentials: options.excludeCredentials?.map((c: any) => ({
        ...c,
        id: this.base64UrlToBuffer(c.id),
      })),
      allowCredentials: options.allowCredentials?.map((c: any) => ({
        ...c,
        id: this.base64UrlToBuffer(c.id),
      })),
    };
  }

  private bufferToBase64Url(buffer: ArrayBuffer): string {
    const bytes = new Uint8Array(buffer);
    let str = '';
    for (const byte of bytes) {
      str += String.fromCharCode(byte);
    }
    return btoa(str).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
  }

  private base64UrlToBuffer(base64url: string): ArrayBuffer {
    const base64 = base64url.replace(/-/g, '+').replace(/_/g, '/');
    const padding = '='.repeat((4 - (base64.length % 4)) % 4);
    const binary = atob(base64 + padding);
    const bytes = new Uint8Array(binary.length);
    for (let i = 0; i < binary.length; i++) {
      bytes[i] = binary.charCodeAt(i);
    }
    return bytes.buffer;
  }
}
