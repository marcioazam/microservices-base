/**
 * Auth Platform SDK Types
 */

export interface AuthPlatformConfig {
  baseUrl: string;
  clientId: string;
  clientSecret?: string;
  scopes?: string[];
  storage?: TokenStorage;
  timeout?: number;
}

export interface TokenStorage {
  get(): Promise<TokenData | null>;
  set(tokens: TokenData): Promise<void>;
  clear(): Promise<void>;
}

export interface TokenData {
  accessToken: string;
  refreshToken?: string;
  expiresAt: number;
  tokenType: string;
  scope?: string;
}

export interface AuthorizeOptions {
  scopes?: string[];
  state?: string;
  redirectUri?: string;
  prompt?: 'none' | 'login' | 'consent';
}

export interface AuthorizationResult {
  code: string;
  state?: string;
}

export interface TokenResponse {
  access_token: string;
  token_type: string;
  expires_in: number;
  refresh_token?: string;
  scope?: string;
}

export interface PasskeyCredential {
  id: string;
  credentialId: string;
  deviceName: string;
  createdAt: Date;
  lastUsedAt?: Date;
  backedUp: boolean;
  transports: string[];
}

export interface PasskeyRegistrationOptions {
  deviceName?: string;
  authenticatorAttachment?: 'platform' | 'cross-platform';
}

export interface PasskeyAuthenticationOptions {
  mediation?: 'optional' | 'required' | 'conditional';
}

export interface CaepEvent {
  type: CaepEventType;
  subject: SubjectIdentifier;
  timestamp: Date;
  reason?: string;
}

export type CaepEventType =
  | 'session-revoked'
  | 'credential-change'
  | 'assurance-level-change'
  | 'token-claims-change';

export interface SubjectIdentifier {
  format: 'iss_sub' | 'email' | 'opaque';
  iss?: string;
  sub?: string;
  email?: string;
  id?: string;
}

export type Unsubscribe = () => void;
