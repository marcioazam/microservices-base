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

**Error:** `ValidationError` / `ErrValidation`

**Cause:** Token signature invalid, expired, or wrong audience.

**Solution:**
1. Verify the token was issued by your auth server
2. Check the `aud` claim matches your client ID
3. Ensure JWKS is up to date

```go
// Go
claims, err := client.ValidateToken(ctx, token)
if authplatform.IsValidation(err) {
    // Token is invalid, request new one
}
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

### Network Error

**Error:** `NetworkError` / `ErrNetwork`

**Cause:** Network connectivity issues or server unavailable.

**Solution:**
1. Check network connectivity
2. Verify the base URL is correct
3. Check for firewall/proxy issues

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
