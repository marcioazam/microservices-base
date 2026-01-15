#!/usr/bin/env python3
"""
Example: Using HashiCorp Vault for Secrets Management

This example demonstrates how to use Vault in the SMS service.

Prerequisites:
1. Vault must be running: docker-compose -f deploy/docker/vault/docker-compose.vault.yml up -d
2. Vault must be initialized: ./deploy/docker/vault/scripts/init-vault.sh
3. Install dependencies: pip install -r requirements-vault.txt
4. Load configuration: source deploy/docker/vault/.env.vault
"""

import os
import sys
from pathlib import Path

# Add parent directory to path for imports
sys.path.insert(0, str(Path(__file__).parent.parent))


def example_1_basic_vault_client():
    """Example 1: Basic Vault client usage."""
    print("\n" + "=" * 60)
    print("Example 1: Basic Vault Client Usage")
    print("=" * 60)

    from src.shared.vault_client import VaultClient

    # Initialize client (reads from VAULT_ADDR and VAULT_TOKEN env vars)
    vault = VaultClient()

    # Check authentication
    if vault.is_authenticated():
        print("✓ Successfully authenticated with Vault")
    else:
        print("✗ Authentication failed")
        return

    # Read a single secret
    jwt_secret = vault.get_secret("sms-service", "jwt_secret_key")
    print(f"✓ JWT Secret: {jwt_secret[:10]}... (length: {len(jwt_secret)})")

    # Read all secrets from a path
    all_secrets = vault.get_secret("sms-service")
    print(f"✓ Retrieved {len(all_secrets)} secrets from sms-service path")
    print(f"  Keys: {list(all_secrets.keys())}")


def example_2_batch_secret_retrieval():
    """Example 2: Batch secret retrieval."""
    print("\n" + "=" * 60)
    print("Example 2: Batch Secret Retrieval")
    print("=" * 60)

    from src.shared.vault_client import VaultClient

    vault = VaultClient()

    # Retrieve multiple secrets at once
    secrets = vault.get_secrets_batch(
        "sms-service",
        ["jwt_secret_key", "twilio_auth_token", "messagebird_api_key"],
    )

    print(f"✓ Retrieved {len(secrets)} secrets in batch:")
    for key in secrets:
        value_preview = secrets[key][:10] if secrets[key] else "None"
        print(f"  - {key}: {value_preview}...")


def example_3_settings_integration():
    """Example 3: Settings integration with Vault."""
    print("\n" + "=" * 60)
    print("Example 3: Settings Integration")
    print("=" * 60)

    from src.config.vault_settings import get_settings_with_vault

    # This automatically loads secrets from Vault
    settings = get_settings_with_vault()

    print("✓ Settings loaded with Vault integration")
    print(f"  - App Name: {settings.app_name}")
    print(f"  - Environment: {settings.environment}")
    print(f"  - JWT Secret Length: {len(settings.jwt_secret_key)} characters")
    print(f"  - JWT Algorithm: {settings.jwt_algorithm}")


def example_4_write_and_delete():
    """Example 4: Write and delete secrets."""
    print("\n" + "=" * 60)
    print("Example 4: Write and Delete Secrets")
    print("=" * 60)

    from src.shared.vault_client import VaultClient
    from hvac.exceptions import InvalidPath

    vault = VaultClient()

    test_path = "sms-service/temp-example"

    # Write a temporary secret
    print("✓ Writing temporary secret...")
    vault.put_secret(test_path, {"example_key": "example_value", "timestamp": "2026-01-14"})

    # Read it back
    print("✓ Reading temporary secret...")
    retrieved = vault.get_secret(test_path)
    print(f"  Retrieved: {retrieved}")

    # Delete it
    print("✓ Deleting temporary secret...")
    vault.delete_secret(test_path)

    # Verify deletion
    try:
        vault.get_secret(test_path)
        print("✗ Secret still exists (should have been deleted)")
    except InvalidPath:
        print("✓ Secret successfully deleted")


def example_5_token_management():
    """Example 5: Token management."""
    print("\n" + "=" * 60)
    print("Example 5: Token Management")
    print("=" * 60)

    from src.shared.vault_client import VaultClient

    vault = VaultClient()

    # Get token TTL
    ttl = vault.get_token_ttl()
    hours = ttl // 3600
    minutes = (ttl % 3600) // 60
    print(f"✓ Token TTL: {hours}h {minutes}m ({ttl} seconds)")

    # Renew token
    print("✓ Renewing token...")
    vault.renew_token()
    print("  Token renewed successfully")

    # Check new TTL
    new_ttl = vault.get_token_ttl()
    new_hours = new_ttl // 3600
    print(f"✓ New Token TTL: {new_hours}h ({new_ttl} seconds)")


def example_6_error_handling():
    """Example 6: Error handling."""
    print("\n" + "=" * 60)
    print("Example 6: Error Handling")
    print("=" * 60)

    from src.shared.vault_client import VaultClient
    from hvac.exceptions import InvalidPath

    vault = VaultClient()

    # Try to read non-existent path
    try:
        vault.get_secret("non-existent-path")
        print("✗ Should have raised InvalidPath error")
    except InvalidPath:
        print("✓ Correctly handled non-existent path")

    # Try to read non-existent key
    try:
        vault.get_secret("sms-service", "non_existent_key")
        print("✗ Should have raised KeyError")
    except KeyError as e:
        print(f"✓ Correctly handled non-existent key: {e}")


def example_7_caching():
    """Example 7: Client caching."""
    print("\n" + "=" * 60)
    print("Example 7: Client Caching")
    print("=" * 60)

    from src.shared.vault_client import get_vault_client

    # Get client (will be cached)
    client1 = get_vault_client()
    print(f"✓ First client: {id(client1)}")

    # Get client again (returns cached instance)
    client2 = get_vault_client()
    print(f"✓ Second client: {id(client2)}")

    if client1 is client2:
        print("✓ Clients are the same instance (cached)")
    else:
        print("✗ Clients are different (caching not working)")

    # Clear cache
    get_vault_client.cache_clear()
    print("✓ Cache cleared")

    # Get new client
    client3 = get_vault_client()
    print(f"✓ Third client: {id(client3)}")

    if client1 is not client3:
        print("✓ New client instance created after cache clear")


def main():
    """Run all examples."""
    print("\n" + "#" * 60)
    print("# HashiCorp Vault Usage Examples")
    print("#" * 60)

    # Check prerequisites
    if not os.getenv("VAULT_ADDR") or not os.getenv("VAULT_TOKEN"):
        print("\n❌ ERROR: Vault not configured!")
        print("\nPlease run:")
        print("  1. docker-compose -f deploy/docker/vault/docker-compose.vault.yml up -d")
        print("  2. ./deploy/docker/vault/scripts/init-vault.sh")
        print("  3. source deploy/docker/vault/.env.vault")
        return

    try:
        # Run all examples
        example_1_basic_vault_client()
        example_2_batch_secret_retrieval()
        example_3_settings_integration()
        example_4_write_and_delete()
        example_5_token_management()
        example_6_error_handling()
        example_7_caching()

        print("\n" + "=" * 60)
        print("✓ All examples completed successfully!")
        print("=" * 60 + "\n")

    except Exception as e:
        print(f"\n❌ Error running examples: {e}")
        import traceback

        traceback.print_exc()


if __name__ == "__main__":
    main()
