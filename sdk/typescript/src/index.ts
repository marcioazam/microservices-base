/**
 * Auth Platform TypeScript SDK
 *
 * @packageDocumentation
 */

export { AuthPlatformClient } from './client';
export { TokenManager, MemoryTokenStorage, LocalStorageTokenStorage } from './token-manager';
export { PasskeysClient } from './passkeys';
export { CaepSubscriber } from './caep';
export { generatePKCE, verifyPKCE } from './pkce';
export * from './types';
export * from './errors';
