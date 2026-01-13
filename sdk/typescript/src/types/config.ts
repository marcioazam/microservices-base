/**
 * Configuration types with validation.
 * 
 * Provides type-safe configuration for the Auth Platform SDK
 * with runtime validation and sensible defaults.
 * 
 * @packageDocumentation
 */

import { InvalidConfigError, MissingRequiredFieldError } from '../errors/index.js';
import type { TokenStorage } from './tokens.js';

/**
 * Auth Platform SDK configuration options.
 * 
 * @example Minimal configuration
 * ```typescript
 * const config: AuthPlatformConfig = {
 *   baseUrl: 'https://auth.example.com',
 *   clientId: 'your-client-id',
 * };
 * ```
 * 
 * @example Full configuration
 * ```typescript
 * const config: AuthPlatformConfig = {
 *   baseUrl: 'https://auth.example.com',
 *   clientId: 'your-client-id',
 *   clientSecret: 'your-client-secret', // For confidential clients
 *   scopes: ['openid', 'profile', 'email'],
 *   storage: new LocalStorageTokenStorage(),
 *   timeout: 30000, // 30 seconds
 *   refreshBuffer: 60000, // Refresh 60s before expiration
 * };
 * ```
 */
export interface AuthPlatformConfig {
  /** Base URL of the auth server (e.g., 'https://auth.example.com') */
  readonly baseUrl: string;
  /** OAuth client ID */
  readonly clientId: string;
  /** OAuth client secret (optional, for confidential clients) */
  readonly clientSecret?: string;
  /** Default OAuth scopes to request */
  readonly scopes?: readonly string[];
  /** Token storage implementation (default: MemoryTokenStorage) */
  readonly storage?: TokenStorage;
  /** Request timeout in milliseconds (default: 30000) */
  readonly timeout?: number;
  /** Time in ms before expiration to trigger refresh (default: 60000) */
  readonly refreshBuffer?: number;
}

/**
 * Default configuration values.
 * Applied when optional fields are not provided.
 */
const DEFAULT_CONFIG = {
  /** Default request timeout: 30 seconds */
  timeout: 30_000,
  /** Default refresh buffer: 60 seconds before expiration */
  refreshBuffer: 60_000,
} as const satisfies Partial<AuthPlatformConfig>;

/**
 * Check if value is a plain object (not null, not array).
 * @internal
 */
function isPlainObject(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

/**
 * Check if value is a non-empty string.
 * @internal
 */
function isNonEmptyString(value: unknown): value is string {
  return typeof value === 'string' && value.length > 0;
}

/**
 * Validate and normalize configuration.
 * 
 * Validates required fields, checks types of optional fields,
 * and applies default values.
 * 
 * @param config - Raw configuration object
 * @returns Validated and normalized configuration
 * @throws {@link InvalidConfigError} If config is not an object or has invalid field types
 * @throws {@link MissingRequiredFieldError} If required fields (baseUrl, clientId) are missing
 * 
 * @example
 * ```typescript
 * const config = validateConfig({
 *   baseUrl: 'https://auth.example.com',
 *   clientId: 'my-client',
 * });
 * // config.timeout === 30000 (default applied)
 * ```
 */
export function validateConfig(config: unknown): AuthPlatformConfig {
  if (!isPlainObject(config)) {
    throw new InvalidConfigError('Config must be an object');
  }

  if (!isNonEmptyString(config.baseUrl)) {
    throw new MissingRequiredFieldError('baseUrl');
  }

  if (!isNonEmptyString(config.clientId)) {
    throw new MissingRequiredFieldError('clientId');
  }

  // Validate optional fields
  if (config.clientSecret !== undefined && typeof config.clientSecret !== 'string') {
    throw new InvalidConfigError('clientSecret must be a string');
  }

  if (config.scopes !== undefined && !Array.isArray(config.scopes)) {
    throw new InvalidConfigError('scopes must be an array');
  }

  if (config.timeout !== undefined && typeof config.timeout !== 'number') {
    throw new InvalidConfigError('timeout must be a number');
  }

  if (config.refreshBuffer !== undefined && typeof config.refreshBuffer !== 'number') {
    throw new InvalidConfigError('refreshBuffer must be a number');
  }

  const result: AuthPlatformConfig = {
    baseUrl: config.baseUrl,
    clientId: config.clientId,
    timeout: (config.timeout as number | undefined) ?? DEFAULT_CONFIG.timeout,
    refreshBuffer: (config.refreshBuffer as number | undefined) ?? DEFAULT_CONFIG.refreshBuffer,
  };

  // Only add optional properties if they are defined (exactOptionalPropertyTypes)
  if (typeof config.clientSecret === 'string') {
    (result as { clientSecret: string }).clientSecret = config.clientSecret;
  }
  if (Array.isArray(config.scopes)) {
    (result as { scopes: readonly string[] }).scopes = config.scopes as readonly string[];
  }
  if (config.storage !== undefined) {
    (result as { storage: TokenStorage }).storage = config.storage as TokenStorage;
  }

  return result;
}

/**
 * Get default configuration values.
 * 
 * @returns Object containing default timeout and refreshBuffer values
 */
export function getDefaultConfig(): typeof DEFAULT_CONFIG {
  return DEFAULT_CONFIG;
}
