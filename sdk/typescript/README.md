# Auth Platform TypeScript SDK

Official TypeScript SDK for the Auth Platform. Supports OAuth 2.1 with PKCE, Passkeys (WebAuthn), and CAEP event subscriptions.

## Installation

```bash
npm install @auth-platform/sdk
# or
yarn add @auth-platform/sdk
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

```typescript
import {
  AuthPlatformError,
  TokenExpiredError,
  PasskeyNotSupportedError,
  RateLimitError,
} from '@auth-platform/sdk';

try {
  await client.registerPasskey();
} catch (error) {
  if (error instanceof PasskeyNotSupportedError) {
    // Show fallback UI
  } else if (error instanceof TokenExpiredError) {
    // Re-authenticate
  } else if (error instanceof RateLimitError) {
    // Wait and retry
    await sleep(error.retryAfter * 1000);
  }
}
```

## API Reference

### AuthPlatformClient

| Method | Description |
|--------|-------------|
| `authorize(options?)` | Start OAuth authorization flow |
| `handleCallback(code, redirectUri?)` | Exchange code for tokens |
| `getAccessToken()` | Get valid access token |
| `isAuthenticated()` | Check if user is authenticated |
| `logout()` | Clear tokens and disconnect |
| `registerPasskey(options?)` | Register new passkey |
| `authenticateWithPasskey(options?)` | Authenticate with passkey |
| `listPasskeys()` | List registered passkeys |
| `deletePasskey(id)` | Delete a passkey |
| `onSecurityEvent(handler)` | Subscribe to CAEP events |

## License

MIT
