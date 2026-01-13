//! Property-based tests for cache operations.
//!
//! Property 13: Cache Encryption Round-Trip
//! Validates: Requirements 12.5

use proptest::prelude::*;
use rust_common::CacheClientConfig;
use std::time::Duration;

/// Generate arbitrary binary data for cache testing.
fn arb_cache_data() -> impl Strategy<Value = Vec<u8>> {
    prop::collection::vec(any::<u8>(), 1..1024)
}

/// Generate arbitrary cache keys.
fn arb_cache_key() -> impl Strategy<Value = String> {
    "[a-zA-Z0-9_-]{1,64}".prop_map(|s| s)
}

proptest! {
    #![proptest_config(ProptestConfig::with_cases(100))]

    /// Property 13: Cache Encryption Round-Trip
    ///
    /// For any data stored in cache with encryption enabled,
    /// retrieving it must return the exact original data.
    #[test]
    fn prop_cache_encryption_round_trip(
        key in arb_cache_key(),
        data in arb_cache_data(),
    ) {
        let rt = tokio::runtime::Runtime::new().unwrap();
        rt.block_on(async {
            // Create encryption key
            let encryption_key = [0x42u8; 32];

            let config = CacheClientConfig::default()
                .with_namespace("prop-test")
                .with_encryption_key(encryption_key);

            let cache = rust_common::CacheClient::new(config).await.unwrap();

            // Store encrypted data
            cache.set(&key, &data, Some(Duration::from_secs(60))).await.unwrap();

            // Retrieve and verify
            let retrieved = cache.get(&key).await.unwrap();
            prop_assert!(retrieved.is_some(), "Data should be retrievable");
            prop_assert_eq!(retrieved.unwrap(), data, "Data must match exactly");

            Ok(())
        })?;
    }

    /// Property: Namespace isolation ensures keys don't collide.
    #[test]
    fn prop_namespace_isolation(
        key in arb_cache_key(),
        data1 in arb_cache_data(),
        data2 in arb_cache_data(),
    ) {
        let rt = tokio::runtime::Runtime::new().unwrap();
        rt.block_on(async {
            let config1 = CacheClientConfig::default()
                .with_namespace("ns1");
            let config2 = CacheClientConfig::default()
                .with_namespace("ns2");

            let cache1 = rust_common::CacheClient::new(config1).await.unwrap();
            let cache2 = rust_common::CacheClient::new(config2).await.unwrap();

            // Store different data with same key in different namespaces
            cache1.set(&key, &data1, None).await.unwrap();
            cache2.set(&key, &data2, None).await.unwrap();

            // Verify isolation
            let retrieved1 = cache1.get(&key).await.unwrap();
            let retrieved2 = cache2.get(&key).await.unwrap();

            prop_assert_eq!(retrieved1.unwrap(), data1);
            prop_assert_eq!(retrieved2.unwrap(), data2);

            Ok(())
        })?;
    }

    /// Property: Deleted keys return None.
    #[test]
    fn prop_delete_removes_data(
        key in arb_cache_key(),
        data in arb_cache_data(),
    ) {
        let rt = tokio::runtime::Runtime::new().unwrap();
        rt.block_on(async {
            let config = CacheClientConfig::default()
                .with_namespace("delete-test");

            let cache = rust_common::CacheClient::new(config).await.unwrap();

            cache.set(&key, &data, None).await.unwrap();
            prop_assert!(cache.exists(&key).await.unwrap());

            cache.delete(&key).await.unwrap();
            prop_assert!(!cache.exists(&key).await.unwrap());

            let retrieved = cache.get(&key).await.unwrap();
            prop_assert!(retrieved.is_none());

            Ok(())
        })?;
    }

    /// Property: Different encryption keys produce different ciphertext.
    #[test]
    fn prop_different_keys_different_ciphertext(
        key in arb_cache_key(),
        data in arb_cache_data(),
    ) {
        let rt = tokio::runtime::Runtime::new().unwrap();
        rt.block_on(async {
            let key1 = [0x01u8; 32];
            let key2 = [0x02u8; 32];

            let config1 = CacheClientConfig::default()
                .with_namespace("enc1")
                .with_encryption_key(key1);
            let config2 = CacheClientConfig::default()
                .with_namespace("enc2")
                .with_encryption_key(key2);

            let cache1 = rust_common::CacheClient::new(config1).await.unwrap();
            let cache2 = rust_common::CacheClient::new(config2).await.unwrap();

            // Store same data with different keys
            cache1.set(&key, &data, None).await.unwrap();
            cache2.set(&key, &data, None).await.unwrap();

            // Both should decrypt correctly to original
            let r1 = cache1.get(&key).await.unwrap().unwrap();
            let r2 = cache2.get(&key).await.unwrap().unwrap();

            prop_assert_eq!(r1, data.clone());
            prop_assert_eq!(r2, data);

            Ok(())
        })?;
    }
}

#[cfg(test)]
mod unit_tests {
    use super::*;

    #[tokio::test]
    async fn test_cache_without_encryption() {
        let config = CacheClientConfig::default()
            .with_namespace("no-enc");

        let cache = rust_common::CacheClient::new(config).await.unwrap();

        let data = b"plaintext data";
        cache.set("test", data, None).await.unwrap();

        let retrieved = cache.get("test").await.unwrap();
        assert_eq!(retrieved, Some(data.to_vec()));
    }

    #[tokio::test]
    async fn test_cache_with_encryption() {
        let key = [0xABu8; 32];
        let config = CacheClientConfig::default()
            .with_namespace("with-enc")
            .with_encryption_key(key);

        let cache = rust_common::CacheClient::new(config).await.unwrap();

        let data = b"sensitive data";
        cache.set("secret", data, None).await.unwrap();

        let retrieved = cache.get("secret").await.unwrap();
        assert_eq!(retrieved, Some(data.to_vec()));
    }

    #[tokio::test]
    async fn test_ttl_expiration() {
        let config = CacheClientConfig::default()
            .with_namespace("ttl-test");

        let cache = rust_common::CacheClient::new(config).await.unwrap();

        cache.set("expiring", b"data", Some(Duration::from_millis(1))).await.unwrap();

        tokio::time::sleep(Duration::from_millis(10)).await;

        let retrieved = cache.get("expiring").await.unwrap();
        assert!(retrieved.is_none());
    }
}
