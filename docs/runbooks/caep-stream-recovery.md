# CAEP Stream Recovery Runbook

## Common Issues

### 1. Stream Delivery Failures

#### Symptom: Events not being delivered

**Possible Causes:**
- Receiver endpoint down
- Network connectivity issues
- Authentication failure
- Rate limiting

**Resolution:**
1. Check stream status
2. Verify receiver endpoint health
3. Check authentication credentials
4. Review error logs

```bash
# Check stream status
curl -H "Authorization: Bearer $TOKEN" \
  https://auth.example.com/caep/streams

# Check CAEP service logs
kubectl logs -l app=caep-service -n auth | grep -i error
```

#### Symptom: High failure rate

**Resolution:**
1. Check receiver endpoint response times
2. Verify TLS certificate validity
3. Check for rate limiting responses
4. Review retry configuration

### 2. Signature Validation Failures

#### Symptom: Receiver rejecting SETs

**Possible Causes:**
- JWKS not refreshed
- Key rotation in progress
- Clock skew

**Resolution:**
1. Verify JWKS endpoint is accessible
2. Check key rotation status
3. Verify clock synchronization

```bash
# Check JWKS endpoint
curl https://auth.example.com/.well-known/jwks.json | jq .

# Check key IDs
curl https://auth.example.com/.well-known/jwks.json | jq '.keys[].kid'
```

### 3. Stream Configuration Issues

#### Symptom: Stream stuck in "failed" status

**Resolution:**
1. Check last error message
2. Verify endpoint URL is correct
3. Test endpoint connectivity
4. Reset stream status

```sql
-- Check stream details
SELECT * FROM caep_streams WHERE status = 'failed';

-- Reset stream status
UPDATE caep_streams 
SET status = 'active', 
    events_failed = 0,
    last_error = NULL 
WHERE id = 'stream-id';
```

### 4. Event Processing Delays

#### Symptom: Events delivered but not processed

**Possible Causes:**
- Receiver processing backlog
- Database contention
- Handler errors

**Resolution:**
1. Check receiver logs
2. Verify event handler health
3. Check database performance

## Monitoring

### Key Metrics

| Metric | Alert Threshold |
|--------|-----------------|
| `caep_event_delivery_latency_p99` | > 100ms |
| `caep_stream_failure_rate` | > 5% |
| `caep_event_queue_depth` | > 1000 |
| `caep_stream_consecutive_failures` | > 5 |

### Health Checks

```bash
# Check stream health
curl -H "Authorization: Bearer $TOKEN" \
  https://auth.example.com/caep/streams | jq '.[] | {id, status, success_rate: (.events_delivered / (.events_delivered + .events_failed))}'
```

## Recovery Procedures

### Stream Reset

When a stream is stuck:

```bash
# 1. Pause the stream
curl -X PATCH -H "Authorization: Bearer $TOKEN" \
  -d '{"status": "paused"}' \
  https://auth.example.com/caep/streams/stream-id

# 2. Clear error state
curl -X POST -H "Authorization: Bearer $TOKEN" \
  https://auth.example.com/caep/streams/stream-id/reset

# 3. Resume the stream
curl -X PATCH -H "Authorization: Bearer $TOKEN" \
  -d '{"status": "active"}' \
  https://auth.example.com/caep/streams/stream-id
```

### Event Replay

If events were lost:

```sql
-- Find undelivered events
SELECT * FROM caep_events 
WHERE stream_id = 'stream-id' 
AND delivery_status = 'failed'
ORDER BY event_timestamp;

-- Mark for retry
UPDATE caep_events 
SET delivery_status = 'pending',
    delivery_attempts = 0
WHERE stream_id = 'stream-id' 
AND delivery_status = 'failed';
```

### Key Rotation Recovery

If key rotation caused issues:

1. Ensure old key is still in JWKS
2. Notify receivers to refresh JWKS
3. Wait for cache TTL to expire
4. Remove old key after grace period

```bash
# Check both keys are present
curl https://auth.example.com/.well-known/jwks.json | jq '.keys | length'
```

### Full Stream Recreation

If stream is unrecoverable:

```bash
# 1. Delete old stream
curl -X DELETE -H "Authorization: Bearer $TOKEN" \
  https://auth.example.com/caep/streams/stream-id

# 2. Create new stream
curl -X POST -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "audience": "https://receiver.example.com",
    "delivery": {
      "method": "push",
      "endpoint_url": "https://receiver.example.com/caep/events"
    },
    "events_requested": ["session-revoked", "credential-change"]
  }' \
  https://auth.example.com/caep/streams
```

## Escalation

| Severity | Contact | Response Time |
|----------|---------|---------------|
| P1 - All streams down | On-call SRE | 15 min |
| P2 - Single stream failing | Auth team | 1 hour |
| P3 - Elevated latency | Auth team | 4 hours |

## Post-Incident

After resolving issues:

1. Review event delivery logs
2. Identify root cause
3. Update monitoring thresholds if needed
4. Document lessons learned
