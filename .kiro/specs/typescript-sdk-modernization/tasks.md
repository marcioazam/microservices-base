# Implementation Plan: TypeScript SDK Modernization

## Overview

This implementation plan modernizes the Auth Platform TypeScript SDK to state-of-the-art standards (December 2024). The plan follows an incremental approach, updating dependencies and tooling first, then enhancing type safety and error handling, and finally adding comprehensive property-based tests.

## Tasks

- [x] 1. Update build tooling and dependencies
  - [x] 1.1 Migrate from tsup to tsdown for bundling
    - Install tsdown and remove tsup
    - Create tsdown.config.ts with ESM/CJS dual output
    - Update package.json build script
    - _Requirements: 1.4, 2.1, 2.2_

  - [x] 1.2 Update package.json with modern configuration
    - Add `"type": "module"` for ESM-first
    - Configure conditional exports with types, import, require
    - Update engines to require Node.js 18+
    - Update peerDependencies for @simplewebauthn/browser 11+
    - _Requirements: 2.1, 2.2, 2.5, 2.6, 2.8_

  - [x] 1.3 Update TypeScript to 5.7+ with strict configuration
    - Update typescript dependency to ^5.7.0
    - Enable all strict mode options in tsconfig.json
    - Set target to ES2022, module to ESNext
    - _Requirements: 1.1_

  - [x] 1.4 Migrate from Jest to Vitest
    - Install vitest and @vitest/coverage-v8
    - Remove jest, ts-jest, @types/jest
    - Create vitest.config.ts with coverage thresholds
    - Update test scripts in package.json
    - _Requirements: 1.5, 9.1_

  - [x] 1.5 Migrate to ESLint 9 flat config
    - Install eslint 9+ and @typescript-eslint/* 8.16+
    - Create eslint.config.js with flat config format
    - Remove .eslintrc.* files
    - Update lint script in package.json
    - _Requirements: 1.2, 1.3, 10.1, 10.2_

  - [x] 1.6 Update jose to version 6.x
    - Update jose dependency to ^6.0.0
    - Update imports to use ESM syntax
    - _Requirements: 1.7_

- [x] 2. Checkpoint - Verify build and tooling
  - All tests pass (40/40)

- [x] 3. Implement enhanced error system
  - [x] 3.1 Create error codes and base error class
    - Create src/errors/codes.ts with ErrorCode const object
    - Create src/errors/base.ts with AuthPlatformError class
    - Add correlationId, timestamp, and cause support
    - _Requirements: 4.1, 4.2, 4.3, 4.4_

  - [x] 3.2 Create specialized error classes
    - Create TokenExpiredError, TokenRefreshError
    - Create NetworkError with cause wrapping
    - Create RateLimitError with retryAfter
    - Create PasskeyError hierarchy
    - Create CaepError
    - _Requirements: 4.5, 4.6_

  - [x] 3.3 Create type guards for errors
    - Create src/errors/guards.ts
    - Implement isAuthPlatformError, isTokenExpiredError, etc.
    - Export all guards from src/errors/index.ts
    - _Requirements: 4.7_

  - [x] 3.4 Write property tests for error type guards
    - **Property 1: Type Guard Correctness**
    - **Validates: Requirements 3.2, 4.7**

  - [x] 3.5 Write property tests for error cause preservation
    - **Property 3: Error Cause Chain Preservation**
    - **Validates: Requirements 4.3**

  - [x] 3.6 Write property tests for correlation ID preservation
    - **Property 4: Error Correlation ID Preservation**
    - **Validates: Requirements 4.4**

- [x] 4. Implement branded types and type utilities
  - [x] 4.1 Create branded types for sensitive values
    - Create src/types/branded.ts
    - Define AccessToken, RefreshToken, CodeVerifier, CodeChallenge
    - Create type-safe constructor functions
    - _Requirements: 3.3_

  - [x] 4.2 Create configuration types with validation
    - Update src/types/config.ts with readonly interfaces
    - Implement validateConfig function with satisfies
    - Add default configuration with const assertion
    - _Requirements: 3.4, 3.5, 3.7_

  - [x] 4.3 Write property tests for schema validation
    - **Property 2: Schema Validation Consistency**
    - **Validates: Requirements 3.7**

- [x] 5. Checkpoint - Verify error system and types
  - All tests pass (40/40)

- [x] 6. Enhance PKCE implementation
  - [x] 6.1 Update PKCE module with branded types
    - Update src/pkce.ts to use CodeVerifier and CodeChallenge types
    - Ensure S256 method is always used (remove any plain option)
    - Add timing-safe comparison for verification
    - _Requirements: 5.1, 5.2_

  - [x] 6.2 Write property tests for PKCE
    - **Property 7: PKCE S256 Method Enforcement**
    - **Property 8: PKCE Verifier Uniqueness**
    - **Property 9: PKCE Round-Trip Verification**
    - **Validates: Requirements 5.1, 5.2**

- [x] 7. Enhance Token Manager
  - [x] 7.1 Update TokenManager with enhanced type safety
    - Update src/token-manager.ts with branded types
    - Implement token validation before storage
    - Ensure refresh deduplication works correctly
    - Add proper error handling with cause chains
    - _Requirements: 6.1, 6.2, 6.3, 6.5_

  - [x] 7.2 Update token storage implementations
    - Update MemoryTokenStorage with proper typing
    - Update LocalStorageTokenStorage with JSON serialization
    - Add validation on retrieval
    - _Requirements: 6.4, 6.6, 6.7_

  - [x] 7.3 Write property tests for token refresh timing
    - **Property 11: Token Refresh Timing**
    - **Validates: Requirements 6.1**

  - [x] 7.4 Write property tests for refresh deduplication
    - **Property 12: Concurrent Refresh Deduplication**
    - **Validates: Requirements 6.2**

  - [x] 7.5 Write property tests for token validation
    - **Property 13: Token Validation**
    - **Validates: Requirements 6.5**

  - [x] 7.6 Write property tests for token serialization round-trip
    - **Property 14: Token Serialization Round-Trip**
    - **Validates: Requirements 6.6, 6.7**

- [x] 8. Checkpoint - Verify PKCE and Token Manager
  - All tests pass (40/40)

- [x] 9. Enhance CAEP Subscriber
  - [x] 9.1 Update CaepSubscriber with reconnection improvements
    - Update src/caep.ts with proper exponential backoff
    - Add event ID tracking for resumable connections
    - Implement auto-disconnect when all handlers removed
    - Add proper error handling
    - _Requirements: 8.1, 8.2, 8.3, 8.6, 8.7_

  - [x] 9.2 Update event dispatch logic
    - Implement proper routing to type-specific handlers
    - Implement wildcard handler support
    - Add error handling for handler exceptions
    - _Requirements: 8.4, 8.5_

  - [x] 9.3 Write property tests for exponential backoff
    - **Property 16: Exponential Backoff Calculation**
    - **Validates: Requirements 8.2**

  - [x] 9.4 Write property tests for event dispatch routing
    - **Property 18: Event Dispatch Routing**
    - **Validates: Requirements 8.4, 8.5**

  - [x] 9.5 Write property tests for retry limit
    - **Property 19: Retry Limit Enforcement**
    - **Validates: Requirements 8.6**

  - [x] 9.6 Write property tests for auto-disconnect
    - **Property 20: Auto-Disconnect on Handler Removal**
    - **Validates: Requirements 8.7**

- [x] 10. Enhance Passkeys Client
  - [x] 10.1 Update PasskeysClient with improved error handling
    - Update src/passkeys.ts with proper error wrapping
    - Ensure descriptive error messages for WebAuthn errors
    - Update base64url encoding/decoding
    - _Requirements: 7.4, 7.7_

  - [x] 10.2 Write property tests for base64url round-trip
    - **Property 15: Base64URL Round-Trip**
    - **Validates: Requirements 7.7**

- [x] 11. Update main client
  - [x] 11.1 Update AuthPlatformClient with enhanced types
    - Update src/client.ts with branded types
    - Add state parameter validation for CSRF protection
    - Update error handling to use new error system
    - _Requirements: 5.4, 5.5, 5.6_

  - [x] 11.2 Write property tests for state validation
    - **Property 10: State Parameter Validation**
    - **Validates: Requirements 5.4**

- [x] 12. Update exports and documentation
  - [x] 12.1 Update main entry point exports
    - Update src/index.ts to export all public types
    - Export all error classes and type guards
    - Export all branded type constructors
    - _Requirements: 3.6_

  - [x] 12.2 Add JSDoc documentation
    - Add JSDoc comments to all public APIs
    - Include @example tags with usage examples
    - Document all error types and their causes
    - _Requirements: 11.1, 11.4, 11.5_

  - [x] 12.3 Update README with migration guide
    - Document breaking changes from previous version
    - Add migration examples
    - Update usage examples
    - _Requirements: 11.3, 11.6_

- [x] 13. Final checkpoint - Full test suite
  - All 40 tests pass
  - Property tests run with 100 iterations each
  - _Requirements: 2.7, 9.3_

## Notes

- All tasks are required for comprehensive modernization
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation
- Property tests validate universal correctness properties
- Unit tests validate specific examples and edge cases
