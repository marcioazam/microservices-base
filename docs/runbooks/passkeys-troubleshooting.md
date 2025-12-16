# Passkeys Troubleshooting Runbook

## Common Issues

### 1. Registration Failures

#### Symptom: NotAllowedError during registration

**Possible Causes:**
- User cancelled the operation
- Timeout exceeded
- Authenticator not available

**Resolution:**
1. Check browser console for detailed error
2. Verify RP ID matches the domain
3. Increase timeout in configuration
4. Check if platform authenticator is available

```bash
# Check MFA service logs
kubectl logs -l app=mfa-service -n auth | grep -i passkey
```

#### Symptom: InvalidStateError

**Cause:** Credential already exists for this user/authenticator combination.

**Resolution:**
1. Check existing credentials for user
2. Offer to manage existing passkeys
3. Use `excludeCredentials` in registration options

### 2. Authentication Failures

#### Symptom: No credentials available

**Possible Causes:**
- No passkeys registered for user
- Wrong RP ID
- Credentials on different device

**Resolution:**
1. Query user's registered passkeys
2. Verify RP ID configuration
3. Offer cross-device authentication

```sql
-- Check user's passkeys
SELECT * FROM passkey_credentials WHERE user_id = 'user-123';
```

#### Symptom: Sign count not increased

**Cause:** Possible cloned authenticator.

**Resolution:**
1. Alert security team
2. Consider revoking the credential
3. Require re-registration

### 3. Cross-Device Issues

#### Symptom: QR code scan fails

**Possible Causes:**
- QR code expired
- Network connectivity issues
- Incompatible authenticator

**Resolution:**
1. Generate new QR code
2. Check network connectivity
3. Verify hybrid transport support

#### Symptom: Hybrid transport timeout

**Resolution:**
1. Check BLE/network connectivity
2. Increase timeout
3. Offer fallback methods

### 4. Performance Issues

#### Symptom: High latency on registration

**Resolution:**
1. Check database query performance
2. Verify Redis/ETS cache is working
3. Check network latency to authenticator

```bash
# Check registration latency metrics
curl -s http://mfa-service:9090/metrics | grep passkey_registration
```

## Monitoring

### Key Metrics

| Metric | Alert Threshold |
|--------|-----------------|
| `passkey_registration_latency_p99` | > 200ms |
| `passkey_authentication_latency_p99` | > 100ms |
| `passkey_registration_error_rate` | > 5% |
| `passkey_authentication_error_rate` | > 1% |

### Dashboards

- Grafana: Auth Platform > Passkeys
- Datadog: auth.passkeys.*

## Recovery Procedures

### Credential Database Corruption

1. Stop MFA service
2. Restore from backup
3. Verify credential integrity
4. Restart service

```bash
# Backup current state
pg_dump -t passkey_credentials auth_db > passkeys_backup.sql

# Restore from backup
psql auth_db < passkeys_backup_good.sql
```

### Mass Credential Revocation

If compromised authenticator detected:

1. Identify affected credentials by AAGUID
2. Mark credentials as revoked
3. Notify affected users
4. Emit CAEP credential-change events

```sql
-- Revoke by AAGUID
UPDATE passkey_credentials 
SET status = 'revoked' 
WHERE aaguid = 'compromised-aaguid';
```

## Escalation

| Severity | Contact | Response Time |
|----------|---------|---------------|
| P1 - Auth down | On-call SRE | 15 min |
| P2 - High error rate | Auth team | 1 hour |
| P3 - Performance degradation | Auth team | 4 hours |
