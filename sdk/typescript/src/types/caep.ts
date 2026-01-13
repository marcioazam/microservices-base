/**
 * CAEP (Continuous Access Evaluation Protocol) types
 */

/**
 * CAEP event types as const object
 */
export const CaepEventType = {
  SESSION_REVOKED: 'session-revoked',
  CREDENTIAL_CHANGE: 'credential-change',
  ASSURANCE_LEVEL_CHANGE: 'assurance-level-change',
  TOKEN_CLAIMS_CHANGE: 'token-claims-change',
} as const;

export type CaepEventType = (typeof CaepEventType)[keyof typeof CaepEventType];

/**
 * Subject identifier formats
 */
export type SubjectFormat = 'iss_sub' | 'email' | 'opaque';

/**
 * Subject identifier for CAEP events
 */
export interface SubjectIdentifier {
  readonly format: SubjectFormat;
  readonly iss?: string;
  readonly sub?: string;
  readonly email?: string;
  readonly id?: string;
}

/**
 * CAEP security event
 */
export interface CaepEvent {
  readonly type: CaepEventType;
  readonly subject: SubjectIdentifier;
  readonly timestamp: Date;
  readonly reason?: string;
}

/**
 * Event handler function type
 */
export type CaepEventHandler = (event: CaepEvent) => void;

/**
 * Unsubscribe function type
 */
export type Unsubscribe = () => void;

/**
 * CAEP subscriber options
 */
export interface CaepSubscriberOptions {
  readonly maxReconnectAttempts?: number;
  readonly initialReconnectDelay?: number;
}
