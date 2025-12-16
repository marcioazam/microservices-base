# CAEP (Continuous Access Evaluation Protocol)

CAEP enables real-time security event sharing between identity providers and relying parties.

## Overview

CAEP is part of the OpenID Shared Signals Framework (SSF) and provides:
- Real-time session revocation
- Credential change notifications
- Risk level updates
- Token claims changes

## Event Types

| Event Type | Description | Trigger |
|------------|-------------|---------|
| `session-revoked` | Session terminated | Logout, admin action, security event |
| `credential-change` | Credential modified | Password change, passkey added/removed |
| `assurance-level-change` | Risk level changed | Step-up auth, risk detection |
| `token-claims-change` | Token claims updated | Role change, permission update |

## Architecture

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│  Auth Platform  │────▶│ CAEP Transmitter│────▶│    Receiver     │
│   (Issuer)      │     │   (SET Signer)  │     │ (Relying Party) │
└─────────────────┘     └─────────────────┘     └─────────────────┘
                               │
                               ▼
                        ┌─────────────┐
                        │   Stream    │
                        │  (Webhook)  │
                        └─────────────┘
```

## Security Event Token (SET)

SETs are signed JWTs containing security events:

```json
{
  "iss": "https://auth.example.com",
  "iat": 1734307200,
  "jti": "unique-event-id",
  "aud": "https://receiver.example.com",
  "events": {
    "https://schemas.openid.net/secevent/caep/event-type/session-revoked": {
      "subject": {
        "format": "iss_sub",
        "iss": "https://auth.example.com",
        "sub": "user-123"
      },
      "event_timestamp": 1734307200,
      "reason_admin": {
        "en": "Session revoked due to security policy"
      }
    }
  }
}
```

## Stream Configuration

### Create Stream
```
POST /caep/streams
Authorization: Bearer <admin-token>

{
  "audience": "https://receiver.example.com",
  "delivery": {
    "method": "push",
    "endpoint_url": "https://receiver.example.com/caep/events"
  },
  "events_requested": [
    "session-revoked",
    "credential-change"
  ],
  "format": "iss_sub"
}

Response:
{
  "stream_id": "stream-123",
  "status": "active"
}
```

### List Streams
```
GET /caep/streams
Authorization: Bearer <admin-token>

Response:
[
  {
    "stream_id": "stream-123",
    "audience": "https://receiver.example.com",
    "status": "active",
    "events_delivered": 1234,
    "events_failed": 5,
    "last_delivery_at": "2025-01-15T12:00:00Z"
  }
]
```

### Delete Stream
```
DELETE /caep/streams/:id
Authorization: Bearer <admin-token>
```

## SSF Discovery

```
GET /.well-known/ssf-configuration

Response:
{
  "issuer": "https://auth.example.com",
  "jwks_uri": "https://auth.example.com/.well-known/jwks.json",
  "delivery_methods_supported": ["push", "poll"],
  "events_supported": [
    "https://schemas.openid.net/secevent/caep/event-type/session-revoked",
    "https://schemas.openid.net/secevent/caep/event-type/credential-change",
    "https://schemas.openid.net/secevent/caep/event-type/assurance-level-change"
  ]
}
```

## Receiving Events

### Webhook Endpoint
```
POST /caep/events
Content-Type: application/secevent+jwt

<signed-set-jwt>
```

### Validation Steps
1. Decode JWT header to get `kid`
2. Fetch JWKS from transmitter
3. Verify signature with ES256
4. Validate claims (iss, aud, iat, jti)
5. Process event

### Example Handler
```python
@app.post("/caep/events")
async def handle_caep_event(request: Request):
    set_jwt = await request.body()
    
    # Validate signature
    claims = validate_set(set_jwt, jwks_uri)
    
    # Process events
    for event_uri, event_data in claims["events"].items():
        if "session-revoked" in event_uri:
            await terminate_session(event_data["subject"]["sub"])
        elif "credential-change" in event_uri:
            await invalidate_credential_cache(event_data["subject"]["sub"])
    
    return {"status": "ok"}
```

## Integration Guide

### Session Identity Core
```elixir
# On logout
Caep.Emitter.emit_session_revoked_logout(user_id, session_id)

# On admin termination
Caep.Emitter.emit_session_revoked_admin(user_id, session_id, admin_id)
```

### MFA Service
```elixir
# On passkey added
Caep.Emitter.emit_passkey_added(user_id, passkey_id)

# On passkey removed
Caep.Emitter.emit_passkey_removed(user_id, passkey_id)
```

### IAM Policy Service
```go
// On role change
emitter.EmitRoleChange(ctx, userID, oldRole, newRole)

// On permission change
emitter.EmitPermissionChange(ctx, userID, added, removed)
```

## Health Monitoring

Stream health metrics:
- `events_delivered` - Total successful deliveries
- `events_failed` - Total failed deliveries
- `avg_latency_ms` - Average delivery latency
- `last_delivery_at` - Last successful delivery time
- `last_error` - Most recent error message

## Error Handling

| Error | Handling | Recovery |
|-------|----------|----------|
| Invalid signature | Reject, log | Refresh JWKS |
| Unknown event type | Log warning | Ignore |
| Delivery failed | Retry with backoff | Alert after 3 failures |
| Subject not found | Log, no action | Ignore stale events |

## Performance

- Event emission: < 100ms p99
- Signature validation: < 10ms
- Stream health update: < 1ms
