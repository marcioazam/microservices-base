# Passkeys (WebAuthn Discoverable Credentials)

Passkeys provide passwordless authentication using FIDO2/WebAuthn discoverable credentials.

## Overview

Passkeys are cryptographic credentials stored on user devices (phones, laptops, security keys) that enable:
- Passwordless authentication
- Phishing-resistant security
- Cross-device authentication via QR codes
- Biometric verification (Touch ID, Face ID, Windows Hello)

## Architecture

```
┌─────────────┐     ┌──────────────┐     ┌─────────────────┐
│   Client    │────▶│  Auth Edge   │────▶│   MFA Service   │
│  (Browser)  │     │   Service    │     │   (Passkeys)    │
└─────────────┘     └──────────────┘     └─────────────────┘
       │                                         │
       │                                         ▼
       │                                 ┌───────────────┐
       └────────────────────────────────▶│  Authenticator │
                                         │ (Platform/Key) │
                                         └───────────────┘
```

## Registration Flow

1. **Begin Registration**
   ```
   POST /passkeys/register/begin
   Authorization: Bearer <token>
   
   Response:
   {
     "challenge": "base64url-encoded",
     "rp": { "id": "example.com", "name": "Example" },
     "user": { "id": "...", "name": "user@example.com" },
     "pubKeyCredParams": [...],
     "authenticatorSelection": {
       "residentKey": "required",
       "userVerification": "required"
     }
   }
   ```

2. **Create Credential** (Browser)
   ```javascript
   const credential = await navigator.credentials.create({
     publicKey: options
   });
   ```

3. **Finish Registration**
   ```
   POST /passkeys/register/finish
   Authorization: Bearer <token>
   
   {
     "id": "credential-id",
     "rawId": "base64url-encoded",
     "type": "public-key",
     "clientDataJSON": "base64url-encoded",
     "attestationObject": "base64url-encoded",
     "transports": ["internal", "hybrid"]
   }
   ```

## Authentication Flow

1. **Begin Authentication**
   ```
   POST /passkeys/authenticate/begin
   
   Response:
   {
     "challenge": "base64url-encoded",
     "rpId": "example.com",
     "userVerification": "required",
     "allowCredentials": []  // Empty for discoverable
   }
   ```

2. **Get Credential** (Browser)
   ```javascript
   const credential = await navigator.credentials.get({
     publicKey: options,
     mediation: "conditional"  // For autofill UI
   });
   ```

3. **Finish Authentication**
   ```
   POST /passkeys/authenticate/finish
   
   {
     "id": "credential-id",
     "rawId": "base64url-encoded",
     "type": "public-key",
     "clientDataJSON": "base64url-encoded",
     "authenticatorData": "base64url-encoded",
     "signature": "base64url-encoded",
     "userHandle": "base64url-encoded"
   }
   ```

## Cross-Device Authentication

For authenticating on devices without registered passkeys:

1. **Generate QR Code**
   ```
   POST /passkeys/cross-device/begin
   
   Response:
   {
     "qr_code": "FIDO://...",
     "session_id": "...",
     "expires_at": "2025-01-15T12:00:00Z"
   }
   ```

2. **Scan with Phone** - User scans QR with phone containing passkey

3. **Complete Authentication** - Phone authenticates via hybrid transport

4. **Offer Local Registration** - Optionally register passkey on current device

## Management API

### List Passkeys
```
GET /passkeys
Authorization: Bearer <token>

Response:
[
  {
    "id": "uuid",
    "device_name": "MacBook Pro Touch ID",
    "created_at": "2025-01-15T00:00:00Z",
    "last_used_at": "2025-01-15T12:00:00Z",
    "backed_up": true,
    "transports": ["internal", "hybrid"]
  }
]
```

### Rename Passkey
```
PATCH /passkeys/:id
Authorization: Bearer <token>

{ "device_name": "Work Laptop" }
```

### Delete Passkey
```
DELETE /passkeys/:id
Authorization: Bearer <token>

Requires recent re-authentication (within 5 minutes)
```

## Security Considerations

1. **User Verification** - Always required for passkeys
2. **Attestation** - Validates authenticator origin
3. **Sign Count** - Detects cloned authenticators
4. **Backup Status** - Indicates if credential is synced

## Browser Support

| Browser | Platform Auth | Security Keys | Conditional UI |
|---------|--------------|---------------|----------------|
| Chrome 108+ | ✅ | ✅ | ✅ |
| Safari 16+ | ✅ | ✅ | ✅ |
| Firefox 119+ | ✅ | ✅ | ✅ |
| Edge 108+ | ✅ | ✅ | ✅ |

## Error Handling

| Error | Cause | Solution |
|-------|-------|----------|
| NotAllowedError | User cancelled | Retry or fallback |
| InvalidStateError | Credential exists | Manage existing |
| NotSupportedError | Not supported | Fallback to TOTP |
| SecurityError | Origin mismatch | Check RP ID |
