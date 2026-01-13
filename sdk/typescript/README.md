# Auth Platform TypeScript SDK

Official TypeScript SDK for the Auth Platform. Supports OAuth 2.1 with PKCE, Passkeys (WebAuthn), and CAEP event subscriptions.

## Features

- **OAuth 2.1 with PKCE** - Secure authorization with S256 challenge method
- **Passkeys (WebAuthn)** - Biometric and security key authentication
- **CAEP Events** - Real-time security event subscriptions via SSE
- **Type Safety** - Branded types, strict mode, comprehensive type guards
- **Modern Stack** - ESM-first, TypeScript 5.7+, Vitest, fast-check

## Requirements

- Node.js 18+ (LTS)
- TypeScript 5.0+ (5.7+ recommended)

## Installation

```bash
npm install @auth-platform/sdk
# or
yarn add @auth-platform/sdk
# or
pnpm add @auth-platform/sdk
```

For Passkeys support, also install:
```bash
npm install @simplewebauthn/browser
```

## Quick Start

```typescript
import { AuthPlatformClient, LocalStorageTokenStorage } from '@auth-platform/sdk';

const client = new AuthPlatformClient({
  baseUrl: 'https://auth.example.com',
  clientId: 'your-client-id',
  storage: new LocalStorageTokenStorage(),
});

// Start OAuth flow
const authUrl = await client.authorize({
  redirectUri: 'https://yourapp.com/callback',
  scopes: ['openid', 'profile'],
});
window.location.href = authUrl;

// Handle callback
const tokens = await client.handleCallback(code, redirectUri);
```

## OAuth 2.1 with PKCE

The SDK automatically uses PKCE with S256 challenge method (required by OAuth 2.1):

```typescript
// Authorization
const authUrl = await client.authorize({
  scopes: ['openid', 'profile', 'email'],
  state: 'random-state-value',
  prompt: 'consent',
});

// Token exchange (after redirect)
const tokens = await client.handleCallback(code, redirectUri);

// Get access token (auto-refreshes if expired)
const accessToken = await client.getAccessToken();
```

## Passkeys (WebAuthn)

```typescript
// Check support
if (AuthPlatformClient.isPasskeysSupported()) {
  // Register a new passkey
  const credential = await client.registerPasskey({
    deviceName: 'My MacBook',
    authenticatorAttachment: 'platform', // or 'cross-platform'
  });

  // Authenticate with passkey
  const result = await client.authenticateWithPasskey({
    mediation: 'conditional', // For autofill UI
  });

  // List passkeys
  const passkeys = await client.listPasskeys();

  // Delete passkey
  await client.deletePasskey(passkeyId);
}
```

## CAEP Event Subscriptions

Subscribe to real-time security events:

```typescript
// Subscribe to all events
const unsubscribe = client.onSecurityEvent((event) => {
  console.log('Security event:', event.type, event.subject);
  
  switch (event.type) {
    case 'session-revoked':
      // Handle session revocation
      client.logout();
      break;
    case 'credential-change':
      // Handle credential change
      break;
  }
});

// Unsubscribe when done
unsubscribe();
```

## Token Management

```typescript
import { TokenManager, MemoryTokenStorage } from '@auth-platform/sdk';

// Custom token storage
class SecureTokenStorage implements TokenStorage {
  async get() { /* ... */ }
  async set(tokens) { /* ... */ }
  async clear() { /* ... */ }
}

// Check authentication status
const isAuth = await client.isAuthenticated();

// Logout
await client.logout();
```

## Error Handling

The SDK provides typed errors with type guards for safe error handling:

```typescript
import {
  AuthPlatformError,
  TokenExpiredError,
  PasskeyNotSupportedError,
  RateLimitError,
  // Type guards
  isAuthPlatformError,
  isTokenExpiredError,
  isPasskeyNotSupportedError,
  isPasskeyCancelledError,
  isRateLimitError,
} from '@auth-platform/sdk';

try {
  await client.registerPasskey();
} catch (error) {
  if (isPasskeyNotSupportedError(error)) {
    // Show fallback UI
  } else if (isPasskeyCancelledError(error)) {
    // User cancelled - allow retry
  } else if (isRateLimitError(error)) {
    // Wait and retry
    const retryAfter = error.retryAfter ?? 60;
    await new Promise(r => setTimeout(r, retryAfter * 1000));
  } else if (isAuthPlatformError(error)) {
    // Handle other SDK errors
    console.error('Error:', error.code, error.message);
    console.error('Correlation ID:', error.correlationId);
  }
}
```

All errors include:
- `code`: Error code for programmatic handling
- `correlationId`: For debugging and support
- `timestamp`: When the error occurred
- `cause`: Original error (if wrapped)


## Configuration Options

```typescript
interface AuthPlatformConfig {
  // Required
  baseUrl: string;      // Auth server URL
  clientId: string;     // OAuth client ID
  
  // Optional
  clientSecret?: string;        // For confidential clients
  scopes?: string[];            // Default scopes
  storage?: TokenStorage;       // Token storage (default: MemoryTokenStorage)
  timeout?: number;             // Request timeout in ms (default: 30000)
  refreshBuffer?: number;       // Refresh before expiry in ms (default: 60000)
}
```

## API Reference

### AuthPlatformClient

| Method | Description |
|--------|-------------|
| `authorize(options?)` | Start OAuth authorization flow |
| `handleCallback(code, redirectUri?, state?)` | Exchange code for tokens |
| `validateState(state)` | Validate state parameter (CSRF protection) |
| `getAccessToken()` | Get valid access token (auto-refreshes) |
| `isAuthenticated()` | Check if user is authenticated |
| `logout()` | Clear tokens and disconnect |
| `registerPasskey(options?)` | Register new passkey |
| `authenticateWithPasskey(options?)` | Authenticate with passkey |
| `listPasskeys()` | List registered passkeys |
| `deletePasskey(id)` | Delete a passkey |
| `onSecurityEvent(handler)` | Subscribe to CAEP events |
| `static isPasskeysSupported()` | Check WebAuthn support |

### Error Types

| Error | Code | Description |
|-------|------|-------------|
| `TokenExpiredError` | `TOKEN_EXPIRED` | Access token has expired |
| `TokenRefreshError` | `TOKEN_REFRESH_FAILED` | Token refresh failed |
| `NetworkError` | `NETWORK_ERROR` | Network request failed |
| `TimeoutError` | `TIMEOUT` | Request timed out |
| `InvalidConfigError` | `INVALID_CONFIG` | Invalid configuration |
| `RateLimitError` | `RATE_LIMITED` | Rate limited (429) |
| `PasskeyNotSupportedError` | `PASSKEY_NOT_SUPPORTED` | WebAuthn not available |
| `PasskeyCancelledError` | `PASSKEY_CANCELLED` | User cancelled operation |
| `PasskeyRegistrationError` | `PASSKEY_REGISTRATION_FAILED` | Registration failed |
| `PasskeyAuthError` | `PASSKEY_AUTH_FAILED` | Authentication failed |
| `CaepConnectionError` | `CAEP_CONNECTION_FAILED` | SSE connection failed |

---

## Migration Guide (v0.1.x → v0.2.x)

### Breaking Changes

#### 1. ESM-First Package

The SDK is now ESM-first with `"type": "module"`. CommonJS is still supported via conditional exports.

```typescript
// ESM (recommended)
import { AuthPlatformClient } from '@auth-platform/sdk';

// CommonJS (still works)
const { AuthPlatformClient } = require('@auth-platform/sdk');
```

#### 2. Node.js 18+ Required

The SDK now requires Node.js 18 or later (LTS versions).

#### 3. Error Handling Changes

Errors now use a consistent hierarchy with error codes:

```typescript
// Before (v0.1.x)
try {
  await client.getAccessToken();
} catch (error) {
  if (error.message.includes('expired')) {
    // Handle expired token
  }
}

// After (v0.2.x)
import { isTokenExpiredError, ErrorCode } from '@auth-platform/sdk';

try {
  await client.getAccessToken();
} catch (error) {
  if (isTokenExpiredError(error)) {
    // Type-safe error handling
    console.log(error.code); // 'TOKEN_EXPIRED'
  }
}
```

#### 4. Branded Types for Tokens

Tokens are now branded types for type safety:

```typescript
// Before (v0.1.x)
const token: string = await client.getAccessToken();

// After (v0.2.x)
import type { AccessToken } from '@auth-platform/sdk';
const token: AccessToken = await client.getAccessToken();
// token is still a string at runtime, but type-checked at compile time
```

#### 5. State Validation

State parameter validation is now explicit:

```typescript
// Before (v0.1.x)
await client.handleCallback(code, redirectUri);

// After (v0.2.x) - state validation is automatic but can be explicit
const isValid = client.validateState(receivedState);
if (!isValid) {
  throw new Error('CSRF attack detected');
}
await client.handleCallback(code, redirectUri, receivedState);
```

### New Features in v0.2.x

- **Type Guards**: `isTokenExpiredError()`, `isNetworkError()`, etc.
- **Correlation IDs**: All errors include `correlationId` for debugging
- **Error Cause Chains**: Original errors preserved in `error.cause`
- **Branded Types**: Type-safe tokens, verifiers, and credentials
- **PKCE Verification**: `verifyPKCE()` function for server-side validation
- **Exponential Backoff**: CAEP reconnection with configurable backoff

### Dependency Updates

| Package | v0.1.x | v0.2.x |
|---------|--------|--------|
| TypeScript | 5.0+ | 5.7+ |
| jose | 5.x | 6.x |
| @simplewebauthn/browser | 9.x | 11.x |
| ESLint | 8.x | 9.x |
| Test Runner | Jest | Vitest |

## License

MIT
