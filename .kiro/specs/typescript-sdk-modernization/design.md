# Design Document: TypeScript SDK Modernization

## Overview

This design document outlines the architectural approach for modernizing the Auth Platform TypeScript SDK to state-of-the-art standards as of December 2024. The modernization focuses on:

1. **Dependency Updates**: Migrating to latest stable versions (TypeScript 5.7+, ESLint 9, Vitest, tsdown, jose 6.x)
2. **Type Safety**: Enhanced type guards, branded types, and strict error handling
3. **Package Structure**: Modern ESM-first dual package with proper conditional exports
4. **Testing**: Vitest with fast-check property-based testing
5. **Code Quality**: ESLint 9 flat config with strict rules

## Architecture

### Module Structure

```
sdk/typescript/
├── src/
│   ├── index.ts              # Main entry point with all exports
│   ├── client.ts             # AuthPlatformClient main class
│   ├── token-manager.ts      # Token storage and refresh logic
│   ├── pkce.ts               # PKCE implementation
│   ├── passkeys.ts           # WebAuthn/Passkeys client
│   ├── caep.ts               # CAEP event subscriber
│   ├── errors/
│   │   ├── index.ts          # Error exports
│   │   ├── base.ts           # Base error class
│   │   ├── codes.ts          # Error codes enum
│   │   └── guards.ts         # Type guards for errors
│   ├── types/
│   │   ├── index.ts          # Type exports
│   │   ├── config.ts         # Configuration types
│   │   ├── tokens.ts         # Token-related types
│   │   ├── passkeys.ts       # Passkey types
│   │   ├── caep.ts           # CAEP event types
│   │   └── branded.ts        # Branded types for sensitive values
│   └── utils/
│       ├── base64url.ts      # Base64URL encoding/decoding
│       ├── crypto.ts         # Crypto utilities
│       └── http.ts           # HTTP client utilities
├── tests/
│   ├── unit/                 # Unit tests
│   ├── property/             # Property-based tests
│   └── integration/          # Integration tests
├── package.json
├── tsconfig.json
├── tsdown.config.ts          # Build configuration
├── eslint.config.js          # ESLint 9 flat config
├── vitest.config.ts          # Vitest configuration
└── README.md
```

### Component Diagram

```mermaid
graph TB
    subgraph SDK["@auth-platform/sdk"]
        Client[AuthPlatformClient]
        TokenMgr[TokenManager]
        PKCE[PKCE Module]
        Passkeys[PasskeysClient]
        CAEP[CaepSubscriber]
        Errors[Error System]
        Types[Type System]
    end
    
    subgraph External["External Dependencies"]
        Jose[jose 6.x]
        WebAuthn[@simplewebauthn/browser]
        FastCheck[fast-check]
    end
    
    Client --> TokenMgr
    Client --> PKCE
    Client --> Passkeys
    Client --> CAEP
    Client --> Errors
    
    TokenMgr --> Jose
    Passkeys --> WebAuthn
    
    subgraph Storage["Token Storage"]
        Memory[MemoryTokenStorage]
        LocalStorage[LocalStorageTokenStorage]
        Custom[Custom Implementation]
    end
    
    TokenMgr --> Storage
```

## Components and Interfaces

### 1. Error System

The error system provides a centralized, type-safe approach to error handling.

```typescript
// Error codes as const enum for tree-shaking
export const ErrorCode = {
  TOKEN_EXPIRED: 'TOKEN_EXPIRED',
  TOKEN_REFRESH_FAILED: 'TOKEN_REFRESH_FAILED',
  NETWORK_ERROR: 'NETWORK_ERROR',
  INVALID_CONFIG: 'INVALID_CONFIG',
  RATE_LIMITED: 'RATE_LIMITED',
  PASSKEY_NOT_SUPPORTED: 'PASSKEY_NOT_SUPPORTED',
  PASSKEY_CANCELLED: 'PASSKEY_CANCELLED',
  CAEP_CONNECTION_FAILED: 'CAEP_CONNECTION_FAILED',
} as const;

export type ErrorCode = typeof ErrorCode[keyof typeof ErrorCode];

// Base error with enhanced properties
export class AuthPlatformError extends Error {
  readonly code: ErrorCode;
  readonly statusCode?: number;
  readonly correlationId?: string;
  readonly timestamp: Date;
  
  constructor(
    message: string,
    code: ErrorCode,
    options?: {
      statusCode?: number;
      correlationId?: string;
      cause?: unknown;
    }
  ) {
    super(message, { cause: options?.cause });
    this.name = 'AuthPlatformError';
    this.code = code;
    this.statusCode = options?.statusCode;
    this.correlationId = options?.correlationId;
    this.timestamp = new Date();
  }
}

// Type guards for error handling
export function isAuthPlatformError(error: unknown): error is AuthPlatformError {
  return error instanceof AuthPlatformError;
}

export function isTokenExpiredError(error: unknown): error is TokenExpiredError {
  return error instanceof TokenExpiredError;
}
```

### 2. Branded Types

Branded types prevent accidental misuse of sensitive string values.

```typescript
// Branded type utility
declare const __brand: unique symbol;
type Brand<T, B> = T & { [__brand]: B };

// Branded types for sensitive values
export type AccessToken = Brand<string, 'AccessToken'>;
export type RefreshToken = Brand<string, 'RefreshToken'>;
export type CodeVerifier = Brand<string, 'CodeVerifier'>;
export type CodeChallenge = Brand<string, 'CodeChallenge'>;
export type CredentialId = Brand<string, 'CredentialId'>;

// Type-safe constructors
export function createAccessToken(value: string): AccessToken {
  if (!value || typeof value !== 'string') {
    throw new InvalidConfigError('Invalid access token');
  }
  return value as AccessToken;
}
```

### 3. Configuration Types with Validation

```typescript
// Configuration schema with satisfies for validation
export interface AuthPlatformConfig {
  readonly baseUrl: string;
  readonly clientId: string;
  readonly clientSecret?: string;
  readonly scopes?: readonly string[];
  readonly storage?: TokenStorage;
  readonly timeout?: number;
  readonly refreshBuffer?: number;
}

// Default configuration with satisfies
const DEFAULT_CONFIG = {
  timeout: 30_000,
  refreshBuffer: 60_000,
} as const satisfies Partial<AuthPlatformConfig>;

// Validation function
export function validateConfig(config: unknown): AuthPlatformConfig {
  if (!isPlainObject(config)) {
    throw new InvalidConfigError('Config must be an object');
  }
  if (typeof config.baseUrl !== 'string' || !config.baseUrl) {
    throw new InvalidConfigError('baseUrl is required');
  }
  if (typeof config.clientId !== 'string' || !config.clientId) {
    throw new InvalidConfigError('clientId is required');
  }
  return { ...DEFAULT_CONFIG, ...config } as AuthPlatformConfig;
}
```

### 4. Token Manager with Enhanced Type Safety

```typescript
export interface TokenData {
  readonly accessToken: AccessToken;
  readonly refreshToken?: RefreshToken;
  readonly expiresAt: number;
  readonly tokenType: 'Bearer';
  readonly scope?: string;
}

export interface TokenStorage {
  get(): Promise<TokenData | null>;
  set(tokens: TokenData): Promise<void>;
  clear(): Promise<void>;
}

export class TokenManager {
  private readonly storage: TokenStorage;
  private readonly baseUrl: string;
  private readonly clientId: string;
  private readonly clientSecret?: string;
  private readonly refreshBuffer: number;
  private refreshPromise: Promise<TokenData> | null = null;

  constructor(options: TokenManagerOptions) {
    this.storage = options.storage;
    this.baseUrl = options.baseUrl;
    this.clientId = options.clientId;
    this.clientSecret = options.clientSecret;
    this.refreshBuffer = options.refreshBuffer ?? 60_000;
  }

  async getAccessToken(): Promise<AccessToken> {
    const tokens = await this.storage.get();
    
    if (!tokens) {
      throw new TokenExpiredError('No tokens available');
    }

    if (this.shouldRefresh(tokens)) {
      const refreshed = await this.refreshTokens(tokens);
      return refreshed.accessToken;
    }

    return tokens.accessToken;
  }

  private shouldRefresh(tokens: TokenData): boolean {
    return Date.now() >= tokens.expiresAt - this.refreshBuffer;
  }

  async refreshTokens(tokens: TokenData): Promise<TokenData> {
    if (!tokens.refreshToken) {
      throw new TokenRefreshError('No refresh token available');
    }

    // Deduplicate concurrent refresh requests
    if (this.refreshPromise) {
      return this.refreshPromise;
    }

    this.refreshPromise = this.doRefresh(tokens.refreshToken);

    try {
      return await this.refreshPromise;
    } finally {
      this.refreshPromise = null;
    }
  }

  private async doRefresh(refreshToken: RefreshToken): Promise<TokenData> {
    // Implementation with proper error handling
  }
}
```

### 5. PKCE Implementation

```typescript
export interface PKCEChallenge {
  readonly codeVerifier: CodeVerifier;
  readonly codeChallenge: CodeChallenge;
  readonly codeChallengeMethod: 'S256';
}

export async function generatePKCE(): Promise<PKCEChallenge> {
  const codeVerifier = generateCodeVerifier();
  const codeChallenge = await generateCodeChallenge(codeVerifier);

  return {
    codeVerifier,
    codeChallenge,
    codeChallengeMethod: 'S256',
  };
}

function generateCodeVerifier(): CodeVerifier {
  const array = new Uint8Array(32);
  crypto.getRandomValues(array);
  return base64UrlEncode(array) as CodeVerifier;
}

async function generateCodeChallenge(verifier: CodeVerifier): Promise<CodeChallenge> {
  const encoder = new TextEncoder();
  const data = encoder.encode(verifier);
  const digest = await crypto.subtle.digest('SHA-256', data);
  return base64UrlEncode(new Uint8Array(digest)) as CodeChallenge;
}

export async function verifyPKCE(
  codeVerifier: CodeVerifier,
  codeChallenge: CodeChallenge
): Promise<boolean> {
  const computed = await generateCodeChallenge(codeVerifier);
  return timingSafeEqual(computed, codeChallenge);
}
```

### 6. CAEP Subscriber with Reconnection

```typescript
export interface CaepSubscriberOptions {
  readonly baseUrl: string;
  readonly getAccessToken: () => Promise<AccessToken>;
  readonly maxReconnectAttempts?: number;
  readonly initialReconnectDelay?: number;
}

export class CaepSubscriber {
  private readonly options: Required<CaepSubscriberOptions>;
  private readonly handlers = new Map<CaepEventType | '*', Set<EventHandler>>();
  private eventSource: EventSource | null = null;
  private reconnectAttempts = 0;
  private lastEventId: string | null = null;

  constructor(options: CaepSubscriberOptions) {
    this.options = {
      maxReconnectAttempts: 5,
      initialReconnectDelay: 1000,
      ...options,
    };
  }

  subscribe(handler: EventHandler, eventTypes?: CaepEventType[]): Unsubscribe {
    const types = eventTypes ?? ['*'];

    for (const type of types) {
      if (!this.handlers.has(type)) {
        this.handlers.set(type, new Set());
      }
      this.handlers.get(type)!.add(handler);
    }

    if (!this.eventSource) {
      this.connect();
    }

    return () => this.unsubscribe(handler, types);
  }

  private async connect(): Promise<void> {
    try {
      const token = await this.options.getAccessToken();
      const url = new URL(`${this.options.baseUrl}/caep/events`);
      url.searchParams.set('token', token);

      this.eventSource = new EventSource(url.toString());

      // Support resumable connections with Last-Event-ID
      if (this.lastEventId) {
        // Note: EventSource doesn't support custom headers
        // Server should check query param as fallback
        url.searchParams.set('lastEventId', this.lastEventId);
      }

      this.eventSource.onopen = () => {
        this.reconnectAttempts = 0;
      };

      this.eventSource.onmessage = (event) => {
        this.lastEventId = event.lastEventId;
        this.handleMessage(event.data);
      };

      this.eventSource.onerror = () => {
        this.handleError();
      };
    } catch (error) {
      this.handleError();
    }
  }

  private handleError(): void {
    this.disconnect();

    if (
      this.reconnectAttempts < this.options.maxReconnectAttempts &&
      this.getTotalHandlers() > 0
    ) {
      const delay = this.options.initialReconnectDelay * 
        Math.pow(2, this.reconnectAttempts);
      this.reconnectAttempts++;

      setTimeout(() => this.connect(), delay);
    }
  }
}
```

## Data Models

### Token Response (from server)

```typescript
export interface TokenResponse {
  readonly access_token: string;
  readonly token_type: 'Bearer';
  readonly expires_in: number;
  readonly refresh_token?: string;
  readonly scope?: string;
}

// Transform function with validation
export function tokenResponseToData(response: TokenResponse): TokenData {
  return {
    accessToken: createAccessToken(response.access_token),
    refreshToken: response.refresh_token 
      ? createRefreshToken(response.refresh_token) 
      : undefined,
    expiresAt: Date.now() + response.expires_in * 1000,
    tokenType: response.token_type,
    scope: response.scope,
  };
}
```

### CAEP Event Types

```typescript
export const CaepEventType = {
  SESSION_REVOKED: 'session-revoked',
  CREDENTIAL_CHANGE: 'credential-change',
  ASSURANCE_LEVEL_CHANGE: 'assurance-level-change',
  TOKEN_CLAIMS_CHANGE: 'token-claims-change',
} as const;

export type CaepEventType = typeof CaepEventType[keyof typeof CaepEventType];

export interface CaepEvent {
  readonly type: CaepEventType;
  readonly subject: SubjectIdentifier;
  readonly timestamp: Date;
  readonly reason?: string;
}

export interface SubjectIdentifier {
  readonly format: 'iss_sub' | 'email' | 'opaque';
  readonly iss?: string;
  readonly sub?: string;
  readonly email?: string;
  readonly id?: string;
}
```



## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system—essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*

### Property 1: Type Guard Correctness

*For any* error instance created by the SDK, the corresponding type guard function SHALL return `true`, and all other type guards SHALL return `false`.

**Validates: Requirements 3.2, 4.7**

### Property 2: Schema Validation Consistency

*For any* valid configuration object conforming to the schema, validation SHALL succeed. *For any* object missing required fields or with invalid field types, validation SHALL throw an InvalidConfigError.

**Validates: Requirements 3.7**

### Property 3: Error Cause Chain Preservation

*For any* error created with a `cause` option, the resulting error's `cause` property SHALL reference the original cause, preserving the full error chain.

**Validates: Requirements 4.3**

### Property 4: Error Correlation ID Preservation

*For any* error created with a `correlationId` option, the resulting error's `correlationId` property SHALL equal the provided value.

**Validates: Requirements 4.4**

### Property 5: Network Error Context Wrapping

*For any* network error caught during HTTP operations, the SDK SHALL wrap it in a NetworkError with the original error as the `cause`.

**Validates: Requirements 4.5**

### Property 6: Rate Limit Error Information

*For any* HTTP 429 response with a Retry-After header, the resulting RateLimitError SHALL contain the `retryAfter` value parsed from the header.

**Validates: Requirements 4.6**

### Property 7: PKCE S256 Method Enforcement

*For any* PKCE challenge generated by the SDK, the `codeChallengeMethod` SHALL always be `'S256'`, never `'plain'`.

**Validates: Requirements 5.1**

### Property 8: PKCE Verifier Uniqueness

*For any* two PKCE challenges generated by the SDK, the `codeVerifier` values SHALL be different (cryptographically unique).

**Validates: Requirements 5.2**

### Property 9: PKCE Round-Trip Verification

*For any* PKCE challenge generated by the SDK, verifying the `codeVerifier` against the `codeChallenge` SHALL return `true`.

**Validates: Requirements 5.1, 5.2**

### Property 10: State Parameter Validation

*For any* state parameter, validation SHALL return `true` only when the provided state exactly matches the expected state.

**Validates: Requirements 5.4**

### Property 11: Token Refresh Timing

*For any* token with `expiresAt` within the refresh buffer period, `shouldRefresh()` SHALL return `true`. *For any* token with `expiresAt` beyond the buffer, it SHALL return `false`.

**Validates: Requirements 6.1**

### Property 12: Concurrent Refresh Deduplication

*For any* number of concurrent calls to `refreshTokens()`, only one actual HTTP refresh request SHALL be made, and all callers SHALL receive the same result.

**Validates: Requirements 6.2**

### Property 13: Token Validation

*For any* valid TokenData object, validation SHALL succeed. *For any* object missing required fields (accessToken, expiresAt, tokenType), validation SHALL fail.

**Validates: Requirements 6.5**

### Property 14: Token Serialization Round-Trip

*For any* valid TokenData object, serializing to JSON and deserializing back SHALL produce an equivalent TokenData object.

**Validates: Requirements 6.6, 6.7**

### Property 15: Base64URL Round-Trip

*For any* Uint8Array, encoding to base64url and decoding back SHALL produce an equivalent Uint8Array.

**Validates: Requirements 7.7**

### Property 16: Exponential Backoff Calculation

*For any* reconnection attempt number `n`, the delay SHALL equal `initialDelay * 2^n` (exponential backoff formula).

**Validates: Requirements 8.2**

### Property 17: Event ID Tracking

*For any* SSE event received with a `lastEventId`, the subscriber SHALL store this ID for use in reconnection.

**Validates: Requirements 8.3**

### Property 18: Event Dispatch Routing

*For any* CAEP event, handlers registered for that specific event type AND wildcard handlers SHALL receive the event. Handlers registered for other event types SHALL NOT receive it.

**Validates: Requirements 8.4, 8.5**

### Property 19: Retry Limit Enforcement

*For any* sequence of connection failures, the number of retry attempts SHALL NOT exceed `maxReconnectAttempts`.

**Validates: Requirements 8.6**

### Property 20: Auto-Disconnect on Handler Removal

*For any* subscriber state where all handlers have been removed, the EventSource connection SHALL be closed.

**Validates: Requirements 8.7**

## Error Handling

### Error Hierarchy

```
AuthPlatformError (base)
├── TokenExpiredError
├── TokenRefreshError
├── NetworkError
├── InvalidConfigError
├── RateLimitError
├── PasskeyError
│   ├── PasskeyNotSupportedError
│   └── PasskeyCancelledError
└── CaepError
```

### Error Recovery Strategies

| Error Type | Recovery Strategy |
|------------|-------------------|
| TokenExpiredError | Trigger refresh flow or re-authenticate |
| TokenRefreshError | Clear tokens, redirect to login |
| NetworkError | Retry with exponential backoff |
| RateLimitError | Wait for `retryAfter` seconds, then retry |
| PasskeyNotSupportedError | Fall back to password authentication |
| PasskeyCancelledError | Show user-friendly message, allow retry |
| CaepError | Log error, attempt reconnection |

### Error Codes

```typescript
export const ErrorCode = {
  // Token errors
  TOKEN_EXPIRED: 'TOKEN_EXPIRED',
  TOKEN_REFRESH_FAILED: 'TOKEN_REFRESH_FAILED',
  TOKEN_INVALID: 'TOKEN_INVALID',
  
  // Network errors
  NETWORK_ERROR: 'NETWORK_ERROR',
  TIMEOUT: 'TIMEOUT',
  
  // Configuration errors
  INVALID_CONFIG: 'INVALID_CONFIG',
  MISSING_REQUIRED_FIELD: 'MISSING_REQUIRED_FIELD',
  
  // Rate limiting
  RATE_LIMITED: 'RATE_LIMITED',
  
  // Passkey errors
  PASSKEY_NOT_SUPPORTED: 'PASSKEY_NOT_SUPPORTED',
  PASSKEY_CANCELLED: 'PASSKEY_CANCELLED',
  PASSKEY_REGISTRATION_FAILED: 'PASSKEY_REGISTRATION_FAILED',
  PASSKEY_AUTH_FAILED: 'PASSKEY_AUTH_FAILED',
  
  // CAEP errors
  CAEP_CONNECTION_FAILED: 'CAEP_CONNECTION_FAILED',
  CAEP_PARSE_ERROR: 'CAEP_PARSE_ERROR',
} as const;
```

## Testing Strategy

### Dual Testing Approach

The SDK uses both unit tests and property-based tests for comprehensive coverage:

- **Unit tests**: Verify specific examples, edge cases, and error conditions
- **Property tests**: Verify universal properties across all valid inputs

### Testing Framework

- **Test Runner**: Vitest (fast, native ESM support, Jest-compatible API)
- **Property Testing**: fast-check 3.15+ (minimum 100 iterations per property)
- **Coverage Target**: 80%+ overall, 100% for cryptographic operations

### Property Test Configuration

```typescript
// vitest.config.ts
import { defineConfig } from 'vitest/config';

export default defineConfig({
  test: {
    globals: true,
    environment: 'node',
    coverage: {
      provider: 'v8',
      reporter: ['text', 'json', 'html'],
      thresholds: {
        global: {
          branches: 80,
          functions: 80,
          lines: 80,
          statements: 80,
        },
      },
    },
  },
});
```

### Property Test Annotation Format

Each property test must be annotated with:

```typescript
/**
 * **Feature: typescript-sdk-modernization, Property 14: Token Serialization Round-Trip**
 * **Validates: Requirements 6.6, 6.7**
 */
describe('Token Serialization', () => {
  it('round-trips token data correctly', () => {
    fc.assert(
      fc.property(validTokenDataArbitrary, (tokenData) => {
        const serialized = JSON.stringify(tokenData);
        const deserialized = JSON.parse(serialized);
        expect(deserialized).toEqual(tokenData);
      }),
      { numRuns: 100 }
    );
  });
});
```

### Test File Structure

```
tests/
├── unit/
│   ├── errors.test.ts
│   ├── config.test.ts
│   └── http.test.ts
├── property/
│   ├── pkce.property.test.ts
│   ├── token-manager.property.test.ts
│   ├── base64url.property.test.ts
│   ├── error-guards.property.test.ts
│   └── caep.property.test.ts
└── integration/
    ├── client.integration.test.ts
    └── passkeys.integration.test.ts
```

### Generators for Property Tests

```typescript
// Arbitrary generators for property tests
import * as fc from 'fast-check';

export const validTokenDataArbitrary = fc.record({
  accessToken: fc.string({ minLength: 10, maxLength: 2048 }),
  refreshToken: fc.option(fc.string({ minLength: 10, maxLength: 2048 })),
  expiresAt: fc.integer({ min: Date.now(), max: Date.now() + 86400000 }),
  tokenType: fc.constant('Bearer' as const),
  scope: fc.option(fc.string({ minLength: 1, maxLength: 256 })),
});

export const errorCodeArbitrary = fc.constantFrom(
  ...Object.values(ErrorCode)
);

export const caepEventTypeArbitrary = fc.constantFrom(
  ...Object.values(CaepEventType)
);
```
