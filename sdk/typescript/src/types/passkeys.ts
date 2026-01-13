/**
 * Passkey-related types
 */

import type { CredentialId } from './branded.js';

/**
 * Registered passkey credential
 */
export interface PasskeyCredential {
  readonly id: string;
  readonly credentialId: CredentialId;
  readonly deviceName: string;
  readonly createdAt: Date;
  readonly lastUsedAt?: Date;
  readonly backedUp: boolean;
  readonly transports: readonly string[];
}

/**
 * Options for passkey registration
 */
export interface PasskeyRegistrationOptions {
  readonly deviceName?: string;
  readonly authenticatorAttachment?: 'platform' | 'cross-platform';
}

/**
 * Options for passkey authentication
 */
export interface PasskeyAuthenticationOptions {
  readonly mediation?: 'optional' | 'required' | 'conditional';
}

/**
 * Result of passkey authentication
 */
export interface PasskeyAuthResult {
  readonly token: string;
}
