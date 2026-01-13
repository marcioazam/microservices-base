# SDK Troubleshooting Guide

Common errors and solutions for Auth Platform SDKs.

## Common Errors

### Token Expired

**Error:** `TokenExpiredError` / `ErrTokenExpired`

**Cause:** Access token has expired and no refresh token is available.

**Solution:**
1. Ensure you're storing the refresh token
2. Re-authenticate the user
3. Check token expiration times in your configuration

```typescript
// TypeScript
try {
  const token = await client.getAccessToken();
} catch (error) {
  if (error instanceof TokenExpiredError) {
    // Redirect to login
    window.location.href = await client.authorize();
  }
}
```

### Rate Limited

**Error:** `RateLimitError` / `ErrRateLimited`

**Cause:** Too many requests in a short period.

**Solution:**
1. Implement exponential backoff
2. Cache tokens and validation results
3. Use the `retryAfter` value from the error

```python
# Python
try:
    claims = client.validate_token(token)
except RateLimitError as e:
    time.sleep(e.retry_after or 60)
    # Retry
```

### Invalid Token

**Error:** `ValidationError` (VAL_2001) / `TokenInvalidError` (AUTH_1002) / `ErrValidation`

**Cause:** Token signature invalid, expired, or wrong audience.

**Solution:**
1. Verify the token was issued by your auth server
2. Check the `aud` claim matches your client ID
3. Ensure JWKS is up to date
4. Check the `correlation_id` in error details for debugging

```go
// Go
claims, err := client.ValidateToken(ctx, token)
if authplatform.IsValidation(err) {
    // Token is invalid, request new one
}
```

```python
# Python - check error details
try:
    claims = client.validate_token(token)
except TokenInvalidError as e:
    print(f"Error code: {e.code}")  # AUTH_1002
    print(f"Correlation ID: {e.correlation_id}")
    print(f"Details: {e.details}")
```

### Passkey Not Supported

**Error:** `PasskeyNotSupportedError`

**Cause:** Browser or device doesn't support WebAuthn.

**Solution:**
1. Check support before showing passkey UI
2. Provide fallback authentication methods

```typescript
if (!AuthPlatformClient.isPasskeysSupported()) {
  // Show password/TOTP login instead
}
```

### Passkey Registration Failed

**Error:** `PasskeyRegistrationError`

**Cause:** WebAuthn credential creation failed.

**Solution:**
1. Check the error cause for underlying WebAuthn error
2. Verify authenticator attachment setting matches available authenticators
3. Ensure RP ID matches your domain

```typescript
import { isPasskeyRegistrationError, isPasskeyCancelledError } from '@auth-platform/sdk';

try {
  await client.registerPasskey({ deviceName: 'My Device' });
} catch (error) {
  if (isPasskeyCancelledError(error)) {
    // User cancelled - show retry option
  } else if (isPasskeyRegistrationError(error)) {
    console.error('Registration failed:', error.message, error.cause);
  }
}
```

### Passkey Authentication Failed

**Error:** `PasskeyAuthError`

**Cause:** WebAuthn assertion failed or server verification failed.

**Solution:**
1. Verify user has registered passkeys
2. Check for credential availability on current device
3. Offer cross-device authentication option

```typescript
import { isPasskeyAuthError, isPasskeyCancelledError } from '@auth-platform/sdk';

try {
  await client.authenticateWithPasskey();
} catch (error) {
  if (isPasskeyCancelledError(error)) {
    // User cancelled
  } else if (isPasskeyAuthError(error)) {
    // Show fallback authentication
  }
}
```

### Network Error

**Error:** `NetworkError` (NET_3001) / `TimeoutError` (NET_3002) / `ErrNetwork`

**Cause:** Network connectivity issues or server unavailable.

**Solution:**
1. Check network connectivity
2. Verify the base URL is correct
3. Check for firewall/proxy issues
4. For timeouts, consider increasing timeout configuration

```python
# Python - handle network errors with cause chain
try:
    claims = client.validate_token(token)
except TimeoutError as e:
    print(f"Timed out after {e.details.get('timeout_seconds')}s")
except NetworkError as e:
    print(f"Network error: {e.message}")
    if e.__cause__:
        print(f"Underlying cause: {e.__cause__}")
```

### DPoP Errors

**Error:** `DPoPError` (DPOP_6001, DPOP_6002, DPOP_6003)

**Cause:** DPoP proof missing, invalid, or nonce required.

**Solution:**
1. Ensure DPoP proof is included in requests
2. Handle nonce requirements by retrying with server-provided nonce
3. Verify DPoP key binding matches token

```python
# Python - handle DPoP nonce requirement
try:
    tokens = client.token_request(dpop_proof=proof)
except DPoPError as e:
    if e.code == "DPOP_6003" and e.dpop_nonce:
        # Retry with new nonce
        new_proof = create_dpop_proof(nonce=e.dpop_nonce)
        tokens = client.token_request(dpop_proof=new_proof)
```

### PKCE Errors

**Error:** `PKCEError` (PKCE_7001, PKCE_7002)

**Cause:** PKCE challenge/verifier missing or invalid.

**Solution:**
1. Ensure code_challenge is sent with authorization request
2. Ensure code_verifier is sent with token request
3. Verify challenge method matches (S256 recommended)

## JWKS Cache Issues

### Stale Keys

**Symptom:** Token validation fails after key rotation.

**Solution:**
1. Invalidate JWKS cache
2. Reduce cache TTL
3. Handle validation errors by refreshing cache

```python
# Python
cache.invalidate()
# Retry validation
```

### Cache Not Refreshing

**Symptom:** Old keys still being used.

**Solution:**
1. Check cache TTL configuration
2. Verify JWKS endpoint is accessible
3. Check for clock skew

## Passkey Issues

### Registration Fails

**Symptom:** `NotAllowedError` during registration.

**Causes:**
- User cancelled the operation
- Timeout exceeded
- Authenticator not available

**Solution:**
1. Increase timeout
2. Check authenticator attachment setting
3. Verify RP ID matches domain

### Authentication Fails

**Symptom:** No credentials available.

**Causes:**
- No passkeys registered for user
- Wrong RP ID
- Credentials on different device

**Solution:**
1. Check if user has registered passkeys
2. Offer cross-device authentication
3. Provide fallback methods

## CAEP Issues

### Events Not Received

**Symptom:** Security events not triggering handlers.

**Causes:**
- SSE connection dropped
- Wrong event types subscribed
- Network issues

**Solution:**
1. Check connection status
2. Verify event type subscription
3. Check server-side stream configuration

### Connection Drops

**Symptom:** Frequent reconnections.

**Solution:**
1. Check network stability
2. Verify token is valid
3. Check server-side timeout settings

## Migration Guide

### v0.x to v1.0

1. Update import paths
2. Replace deprecated methods
3. Update error handling

```typescript
// Old
import { Client } from '@auth-platform/sdk';

// New
import { AuthPlatformClient } from '@auth-platform/sdk';
```

## Getting Help

1. Check the [API documentation](../api/)
2. Search [GitHub issues](https://github.com/auth-platform/sdk/issues)
3. Contact support
