# Elasticsearch Security Fix (2026-01-09)

## Problem

Elasticsearch was running **without authentication**, allowing anyone with network access to:
- Read all logs (potentially containing PII/secrets)
- Write arbitrary data
- Delete indices
- Modify cluster settings

**Vulnerability**: `xpack.security.enabled=false` in `docker-compose.yml`

## Solution

Elasticsearch security (X-Pack) is now **ENABLED by default**:

```yaml
environment:
  - xpack.security.enabled=true
  - ELASTIC_PASSWORD=${ELASTICSEARCH_PASSWORD:-changeme}
```

## Setup Instructions

### 1. Set Elasticsearch Password

**Option A: Via .env file** (Recommended)

```bash
cd deploy/docker
cp .env.example .env

# Generate a strong password
ELASTIC_PASSWORD=$(openssl rand -base64 32)

# Add to .env file
echo "ELASTICSEARCH_PASSWORD=${ELASTIC_PASSWORD}" >> .env
```

**Option B: Export environment variable**

```bash
export ELASTICSEARCH_PASSWORD=$(openssl rand -base64 32)
docker-compose up -d
```

### 2. Restart Services

```bash
docker-compose down
docker-compose up -d elasticsearch kibana
```

### 3. Verify Security is Enabled

```bash
# This should fail (401 Unauthorized)
curl http://localhost:9200/_cluster/health

# This should succeed
curl -u elastic:${ELASTICSEARCH_PASSWORD} http://localhost:9200/_cluster/health
```

### 4. Access Kibana

1. Navigate to `http://localhost:5601`
2. Login with:
   - Username: `elastic`
   - Password: (value from `ELASTICSEARCH_PASSWORD`)

## Updating Application Configuration

Services that connect to Elasticsearch need to be updated with credentials:

### Logging Service

```yaml
# docker-compose.yml or service configuration
environment:
  - ELASTICSEARCH_URL=http://elasticsearch:9200
  - ELASTICSEARCH_USERNAME=elastic
  - ELASTICSEARCH_PASSWORD=${ELASTICSEARCH_PASSWORD}
```

### Fluentd/Fluent Bit

```ini
# fluent.conf
<match **>
  @type elasticsearch
  host elasticsearch
  port 9200
  user elastic
  password ${ELASTICSEARCH_PASSWORD}
  # ...
</match>
```

### OpenTelemetry Collector

```yaml
# otel-collector-config.yaml
exporters:
  elasticsearch:
    endpoints: [http://elasticsearch:9200]
    auth:
      authenticator: basicauth/elasticsearch

extensions:
  basicauth/elasticsearch:
    client_auth:
      username: elastic
      password: ${ELASTICSEARCH_PASSWORD}
```

## Production Considerations

### 1. Use Strong Passwords

❌ **NEVER use the default `changeme` password in production!**

Generate strong passwords:
```bash
openssl rand -base64 32
```

### 2. Create Service-Specific Users

Instead of using the `elastic` superuser everywhere, create users with minimal privileges:

```bash
# Connect to Elasticsearch
curl -u elastic:${ELASTICSEARCH_PASSWORD} -X POST http://localhost:9200/_security/user/logging_service \
  -H 'Content-Type: application/json' \
  -d '{
    "password": "'"$(openssl rand -base64 32)"'",
    "roles": ["logs_writer"],
    "full_name": "Logging Service User"
  }'

# Create custom role for logging
curl -u elastic:${ELASTICSEARCH_PASSWORD} -X POST http://localhost:9200/_security/role/logs_writer \
  -H 'Content-Type: application/json' \
  -d '{
    "cluster": ["monitor"],
    "indices": [
      {
        "names": ["logs-*"],
        "privileges": ["create_index", "write", "read"]
      }
    ]
  }'
```

### 3. Enable TLS/SSL

For production, enable TLS for Elasticsearch:

```yaml
elasticsearch:
  environment:
    - xpack.security.http.ssl.enabled=true
    - xpack.security.http.ssl.certificate=/usr/share/elasticsearch/config/certs/elasticsearch.crt
    - xpack.security.http.ssl.key=/usr/share/elasticsearch/config/certs/elasticsearch.key
  volumes:
    - ./certs:/usr/share/elasticsearch/config/certs:ro
```

### 4. Restrict Network Access

Don't expose Elasticsearch port externally in production:

```yaml
elasticsearch:
  ports:
    # ❌ Don't do this in production:
    # - "9200:9200"

    # ✅ Access only via service name in Docker network
    # (no ports exposed to host)
```

### 5. Regular Password Rotation

Rotate passwords every 90 days:

```bash
curl -u elastic:${OLD_PASSWORD} -X POST http://localhost:9200/_security/user/elastic/_password \
  -H 'Content-Type: application/json' \
  -d '{
    "password": "'"${NEW_PASSWORD}"'"
  }'
```

## Troubleshooting

### "Connection refused" or "401 Unauthorized"

Check that:
1. Elasticsearch is running: `docker-compose ps elasticsearch`
2. Password is set: `echo $ELASTICSEARCH_PASSWORD`
3. Credentials are correct in applications

### Kibana "Unable to connect to Elasticsearch"

Check Kibana logs:
```bash
docker-compose logs kibana
```

Common issues:
- `ELASTICSEARCH_PASSWORD` not set in Kibana environment
- Mismatched passwords between Elasticsearch and Kibana

### Services can't write logs to Elasticsearch

Update service configuration with credentials:
```yaml
environment:
  - ELASTICSEARCH_USERNAME=elastic
  - ELASTICSEARCH_PASSWORD=${ELASTICSEARCH_PASSWORD}
```

## Migration from Insecure Setup

If you have existing data in Elasticsearch without security:

1. **Backup data** before enabling security:
   ```bash
   docker-compose exec elasticsearch elasticsearch-dump \
     --input=http://localhost:9200/my-index \
     --output=/tmp/my-index.json
   ```

2. **Enable security** (as described above)

3. **Restore data** with credentials:
   ```bash
   docker-compose exec elasticsearch elasticsearch-dump \
     --input=/tmp/my-index.json \
     --output=http://elastic:${ELASTICSEARCH_PASSWORD}@localhost:9200/my-index
   ```

## Security Checklist

Before deploying to production:

- [ ] Set strong `ELASTICSEARCH_PASSWORD` (32+ characters)
- [ ] Create service-specific users (not using `elastic` superuser)
- [ ] Enable TLS/SSL for Elasticsearch
- [ ] Don't expose Elasticsearch port externally
- [ ] Configure audit logging
- [ ] Set up password rotation schedule
- [ ] Update all services with credentials
- [ ] Test authentication is working
- [ ] Monitor for authentication failures
- [ ] Document credential storage location (Vault/AWS Secrets Manager)

## References

- Elasticsearch Security: https://www.elastic.co/guide/en/elasticsearch/reference/current/secure-cluster.html
- X-Pack Security: https://www.elastic.co/guide/en/elasticsearch/reference/current/security-settings.html
- OWASP: https://owasp.org/www-community/vulnerabilities/Insecure_Elasticsearch_Configuration
