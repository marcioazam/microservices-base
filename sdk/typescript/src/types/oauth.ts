/**
 * OAuth-related types
 */

/**
 * Options for authorization request
 */
export interface AuthorizeOptions {
  readonly scopes?: readonly string[];
  readonly state?: string;
  readonly redirectUri?: string;
  readonly prompt?: 'none' | 'login' | 'consent';
}

/**
 * Result of authorization callback
 */
export interface AuthorizationResult {
  readonly code: string;
  readonly state?: string;
}
