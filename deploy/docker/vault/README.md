# HashiCorp Vault - Quick Reference

## 🚀 Quick Start (5 minutes)

```bash
# 1. Start Vault
docker-compose -f docker-compose.vault.yml up -d

# 2. Initialize with secrets
chmod +x scripts/*.sh
./scripts/init-vault.sh

# 3. Test connection
./scripts/test-vault.sh

# 4. Load configuration
source .env.vault
```

**That's it! Vault is ready to use.** 🎉

---

## 📝 Essential Commands

### View Secrets
```bash
# All secrets in SMS service
vault kv get secret/sms-service

# Specific secret
vault kv get -field=jwt_secret_key secret/sms-service

# JSON format
vault kv get -format=json secret/sms-service | jq .
```

### Add/Update Secrets
```bash
# Single secret
vault kv put secret/sms-service new_api_key="xyz123"

# Multiple secrets
vault kv put secret/sms-service \
    jwt_secret_key="$(python3 -c 'import secrets; print(secrets.token_urlsafe(48))')" \
    api_key="another_secret"
```

### Delete Secrets
```bash
# Delete latest version
vault kv delete secret/sms-service

# Permanently delete all versions
vault kv metadata delete secret/sms-service
```

### Vault Status
```bash
# Check status
vault status

# Health check
curl http://localhost:8200/v1/sys/health

# Token info
vault token lookup
```

---

## 🌐 Web UI

Open in browser: **http://localhost:8200/ui**

**Login Token:** `root` (development) or your service token

---

## 🐍 Python Usage

```python
from src.config.vault_settings import get_settings_with_vault

# Automatically loads secrets from Vault
settings = get_settings_with_vault()
jwt_secret = settings.jwt_secret_key
```

---

## 🔧 Configuration Files

- `docker-compose.vault.yml` - Vault server configuration
- `config/vault.hcl` - Production settings
- `policies/sms-service-policy.hcl` - Access control
- `scripts/init-vault.sh` - Setup automation
- `scripts/test-vault.sh` - Connection testing

---

## 🆘 Troubleshooting

### Vault not starting?
```bash
# Check logs
docker logs vault-server

# Restart
docker-compose -f docker-compose.vault.yml restart
```

### Can't authenticate?
```bash
# Verify token
echo $VAULT_TOKEN

# Re-load config
source .env.vault

# Check connection
vault status
```

### Secrets not found?
```bash
# List all secrets
vault kv list secret/

# Re-initialize
./scripts/init-vault.sh
```

---

## 📚 Full Documentation

See [VAULT_SETUP_GUIDE.md](./VAULT_SETUP_GUIDE.md) for complete documentation.

---

## 🔒 Security Notes

⚠️ **IMPORTANT:**
- Never commit `.env.vault` to version control
- Use limited-privilege tokens for services (not root)
- Enable TLS in production
- Rotate secrets regularly
- Enable audit logging

---

## 📞 Support

- **Documentation:** [VAULT_SETUP_GUIDE.md](./VAULT_SETUP_GUIDE.md)
- **Official Docs:** https://www.vaultproject.io/docs
- **Python Client:** https://hvac.readthedocs.io/

---

**Version:** 1.0.0 | **Last Updated:** 2026-01-14
