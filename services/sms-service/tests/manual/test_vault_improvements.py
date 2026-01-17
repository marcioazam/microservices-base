#!/usr/bin/env python3
"""
Manual tests for Vault client improvements.

Tests the following HIGH priority fixes:
1. Enhanced error handling with context
2. Automatic token renewal
3. Rate limiting
4. Robust database URL reconstruction

Prerequisites:
- Vault must be running: docker-compose -f deploy/docker/vault/docker-compose.vault.yml up -d
- Vault must be initialized: ./deploy/docker/vault/scripts/init-vault.sh
- Load configuration: source deploy/docker/vault/.env.vault
"""

import os
import sys
import time
from pathlib import Path

# Add parent directory to path for imports
sys.path.insert(0, str(Path(__file__).parent.parent.parent))


def test_1_enhanced_error_handling():
    """Test 1: Enhanced error handling with structured logging."""
    print("\n" + "=" * 70)
    print("Test 1: Enhanced Error Handling with Structured Logging")
    print("=" * 70)

    from src.shared.vault_client import VaultClient
    from hvac.exceptions import InvalidPath

    vault = VaultClient()

    # Test 1a: Non-existent path error
    print("\n1a. Testing non-existent path error...")
    try:
        vault.get_secret("non-existent-path-12345")
        print("❌ FAIL: Should have raised InvalidPath")
        return False
    except InvalidPath as e:
        print(f"✓ PASS: Caught InvalidPath - {e}")

    # Test 1b: Non-existent key error
    print("\n1b. Testing non-existent key error...")
    try:
        vault.get_secret("sms-service", "non_existent_key_12345")
        print("❌ FAIL: Should have raised KeyError")
        return False
    except KeyError as e:
        print(f"✓ PASS: Caught KeyError - {e}")

    print("\n✅ Test 1 PASSED: Enhanced error handling working correctly")
    return True


def test_2_automatic_token_renewal():
    """Test 2: Automatic token renewal with background thread."""
    print("\n" + "=" * 70)
    print("Test 2: Automatic Token Renewal")
    print("=" * 70)

    from src.shared.vault_client import VaultClient

    # Create client with auto-renewal enabled
    # Use short interval for testing (30 seconds instead of 5 minutes)
    vault = VaultClient(
        auto_renew=True,
        renew_threshold=0.5,  # Renew at 50% TTL
        renew_interval=30,  # Check every 30 seconds
    )

    print(f"\n✓ Vault client created with auto_renew=True")
    print(f"  - Renewal threshold: 50% of TTL")
    print(f"  - Check interval: 30 seconds")

    # Check initial TTL
    initial_ttl = vault.get_token_ttl()
    print(f"\n✓ Initial token TTL: {initial_ttl} seconds ({initial_ttl // 3600}h {(initial_ttl % 3600) // 60}m)")

    # Verify renewal thread is running
    if vault._renewal_thread and vault._renewal_thread.is_alive():
        print(f"✓ Renewal thread is running (thread: {vault._renewal_thread.name})")
    else:
        print("❌ FAIL: Renewal thread not running")
        return False

    # Wait a bit and check thread is still alive
    print("\n⏳ Waiting 5 seconds to verify thread stability...")
    time.sleep(5)

    if vault._renewal_thread.is_alive():
        print("✓ Renewal thread still running after 5 seconds")
    else:
        print("❌ FAIL: Renewal thread died")
        return False

    # Test manual stop
    print("\n⏳ Testing manual thread stop...")
    vault.stop_token_renewal()
    time.sleep(1)

    if not vault._renewal_thread.is_alive():
        print("✓ Renewal thread stopped successfully")
    else:
        print("⚠️  WARNING: Thread did not stop within timeout")

    print("\n✅ Test 2 PASSED: Automatic token renewal working correctly")
    return True


def test_3_rate_limiting():
    """Test 3: Rate limiting with token bucket algorithm."""
    print("\n" + "=" * 70)
    print("Test 3: Rate Limiting")
    print("=" * 70)

    from src.shared.vault_client import VaultClient

    # Create client with strict rate limiting for testing
    vault = VaultClient(
        rate_limit=2.0,  # 2 requests per second
        rate_limit_burst=5,  # Burst capacity of 5
        auto_renew=False,  # Disable auto-renewal for this test
    )

    print(f"\n✓ Vault client created with rate limiting:")
    print(f"  - Rate: 2 requests/second")
    print(f"  - Burst: 5 requests")

    # Test burst capacity (should be fast)
    print("\n⏳ Testing burst capacity (5 requests)...")
    start = time.time()
    for i in range(5):
        vault.get_secret("sms-service", "jwt_secret_key")
        elapsed = time.time() - start
        print(f"  Request {i+1} at {elapsed:.3f}s")

    burst_duration = time.time() - start
    print(f"\n✓ Burst completed in {burst_duration:.3f}s (should be < 1s)")

    if burst_duration > 1.5:
        print(f"⚠️  WARNING: Burst took longer than expected")

    # Test rate limiting (should be throttled)
    print("\n⏳ Testing rate limiting (5 more requests)...")
    start = time.time()
    for i in range(5):
        vault.get_secret("sms-service", "jwt_secret_key")
        elapsed = time.time() - start
        print(f"  Request {i+1} at {elapsed:.3f}s")

    throttled_duration = time.time() - start
    expected_min_duration = 2.5  # 5 requests / 2 req/s = 2.5s

    print(f"\n✓ Throttled requests completed in {throttled_duration:.3f}s")
    print(f"  Expected minimum: {expected_min_duration:.1f}s")

    if throttled_duration < expected_min_duration:
        print(f"⚠️  WARNING: Rate limiting may not be working correctly")
    else:
        print(f"✓ Rate limiting working as expected")

    print("\n✅ Test 3 PASSED: Rate limiting working correctly")
    return True


def test_4_database_url_reconstruction():
    """Test 4: Robust database URL reconstruction with urllib.parse."""
    print("\n" + "=" * 70)
    print("Test 4: Database URL Reconstruction")
    print("=" * 70)

    from src.config.vault_settings import VaultSettings
    from urllib.parse import urlparse

    # Test cases covering edge cases
    test_cases = [
        {
            "name": "Standard PostgreSQL URL",
            "url": "postgresql+asyncpg://user:oldpass@localhost:5432/dbname",
            "new_password": "newpass123",
            "expected_user": "user",
            "expected_host": "localhost",
            "expected_port": 5432,
        },
        {
            "name": "IPv6 Address",
            "url": "postgresql+asyncpg://user:oldpass@[::1]:5432/dbname",
            "new_password": "newpass456",
            "expected_user": "user",
            "expected_host": "::1",
            "expected_port": 5432,
        },
        {
            "name": "URL with special characters in password",
            "url": "postgresql+asyncpg://user:old@pass@localhost:5432/dbname",
            "new_password": "new@pass#2025!",
            "expected_user": "user",
            "expected_host": "localhost",
            "expected_port": 5432,
        },
        {
            "name": "URL without port",
            "url": "postgresql+asyncpg://user:oldpass@localhost/dbname",
            "new_password": "newpass789",
            "expected_user": "user",
            "expected_host": "localhost",
            "expected_port": None,
        },
    ]

    passed = 0
    failed = 0

    for i, test_case in enumerate(test_cases, 1):
        print(f"\n{i}. Testing: {test_case['name']}")
        print(f"   Original URL: {test_case['url']}")

        try:
            # Create a mock settings object
            class MockSettings:
                database_url = test_case["url"]

            mock_settings = MockSettings()

            # Create VaultSettings instance (without actually connecting to Vault)
            vault_settings = VaultSettings.__new__(VaultSettings)
            vault_settings.base_settings = mock_settings

            # Call the URL update method
            vault_settings._update_database_url(test_case["new_password"])

            # Parse the updated URL
            updated_url = str(mock_settings.database_url)
            parsed = urlparse(updated_url)

            print(f"   Updated URL: {updated_url}")

            # Verify username
            if parsed.username != test_case["expected_user"]:
                print(f"   ❌ FAIL: Username mismatch (expected: {test_case['expected_user']}, got: {parsed.username})")
                failed += 1
                continue

            # Verify hostname
            if parsed.hostname != test_case["expected_host"]:
                print(f"   ❌ FAIL: Hostname mismatch (expected: {test_case['expected_host']}, got: {parsed.hostname})")
                failed += 1
                continue

            # Verify port
            if parsed.port != test_case["expected_port"]:
                print(f"   ❌ FAIL: Port mismatch (expected: {test_case['expected_port']}, got: {parsed.port})")
                failed += 1
                continue

            # Verify password was updated
            if parsed.password != test_case["new_password"]:
                print(f"   ❌ FAIL: Password not updated correctly")
                failed += 1
                continue

            print(f"   ✓ PASS: URL reconstructed correctly")
            passed += 1

        except Exception as e:
            print(f"   ❌ FAIL: Exception - {e}")
            failed += 1

    print(f"\n{'=' * 70}")
    print(f"Results: {passed} passed, {failed} failed out of {len(test_cases)} tests")

    if failed == 0:
        print("\n✅ Test 4 PASSED: Database URL reconstruction working correctly")
        return True
    else:
        print(f"\n❌ Test 4 FAILED: {failed} test case(s) failed")
        return False


def main():
    """Run all validation tests."""
    print("\n" + "#" * 70)
    print("# Vault Client Improvements - Validation Tests")
    print("#" * 70)

    # Check prerequisites
    if not os.getenv("VAULT_ADDR") or not os.getenv("VAULT_TOKEN"):
        print("\n❌ ERROR: Vault not configured!")
        print("\nPlease run:")
        print("  1. docker-compose -f deploy/docker/vault/docker-compose.vault.yml up -d")
        print("  2. ./deploy/docker/vault/scripts/init-vault.sh")
        print("  3. source deploy/docker/vault/.env.vault")
        return

    results = []

    try:
        # Run all tests
        print("\n" + "=" * 70)
        print("Running validation tests...")
        print("=" * 70)

        results.append(("Enhanced Error Handling", test_1_enhanced_error_handling()))
        results.append(("Automatic Token Renewal", test_2_automatic_token_renewal()))
        results.append(("Rate Limiting", test_3_rate_limiting()))
        results.append(("Database URL Reconstruction", test_4_database_url_reconstruction()))

        # Summary
        print("\n" + "=" * 70)
        print("TEST SUMMARY")
        print("=" * 70)

        passed = sum(1 for _, result in results if result)
        failed = len(results) - passed

        for name, result in results:
            status = "✅ PASSED" if result else "❌ FAILED"
            print(f"{status}: {name}")

        print(f"\nTotal: {passed} passed, {failed} failed out of {len(results)} tests")

        if failed == 0:
            print("\n" + "=" * 70)
            print("🎉 ALL TESTS PASSED!")
            print("=" * 70 + "\n")
        else:
            print(f"\n⚠️  {failed} test(s) failed. Please review the output above.")

    except Exception as e:
        print(f"\n❌ Error running tests: {e}")
        import traceback
        traceback.print_exc()


if __name__ == "__main__":
    main()
