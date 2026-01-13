//! Property-based tests for refresh token module.
//!
//! Property 7: Refresh Token Rotation Invalidation
//! Property 8: Refresh Token Replay Detection
//! Property 9: Token Family Uniqueness
//! Property 10: Token Family Revocation

use proptest::prelude::*;
use rust_common::{CacheClientConfig, LoggingClientConfig};
use std::sync::Arc;
use std::time::Duration;

/// Generate arbitrary user IDs.
fn arb_user_id() -> impl Strategy<Value = String> {
    "[a-zA-Z0-9_-]{8,32}".prop_map(|s| s)
}

/// Generate arbitrary session IDs.
fn arb_session_id() -> impl Strategy<Value = String> {
    "[a-f0-9]{32}".prop_map(|s| s)
}

async fn create_test_rotator() -> token_service::refresh::RefreshTokenRotator {
    let cache_config = CacheClientConfig::default()
        .with_namespace(&format!("refresh-test-{}", uuid::Uuid::new_v4()));
    let storage = Arc::new(
        token_service::storage::CacheStorage::new(cache_config)
            .await
            .unwrap(),
    );

    let log_config = LoggingClientConfig::default().with_service_id("token-service-test");
    let logger = Arc::new(rust_common::LoggingClient::new(log_config).await.unwrap());

    token_service::refresh::RefreshTokenRotator::new(
        storage,
        logger,
        Duration::from_secs(604800),
    )
}

proptest! {
    #![proptest_config(ProptestConfig::with_cases(100))]

    /// Property 7: Refresh Token Rotation Invalidation
    ///
    /// After rotation, the old token must be invalid and
    /// only the new token should work.
    #[test]
    fn prop_refresh_token_rotation_invalidation(
        user_id in arb_user_id(),
        session_id in arb_session_id(),
    ) {
        let rt = tokio::runtime::Runtime::new().unwrap();
        rt.block_on(async {
            let rotator = create_test_rotator().await;

            // Create initial token
            let (token1, family1) = rotator
                .create_token_family(&user_id, &session_id, None)
                .await
                .unwrap();

            prop_assert!(!token1.is_empty(), "Token should not be empty");
            prop_assert_eq!(&family1.user_id, &user_id);
            prop_assert_eq!(family1.rotation_count, 0);

            // Rotate token
            let (token2, family2) = rotator.rotate(&token1, None).await.unwrap();

            prop_assert_ne!(&token1, &token2, "New token must be different");
            prop_assert_eq!(family2.family_id, family1.family_id, "Family ID preserved");
            prop_assert_eq!(family2.rotation_count, 1, "Rotation count incremented");

            // New token should work for another rotation
            let (token3, family3) = rotator.rotate(&token2, None).await.unwrap();
            prop_assert_ne!(&token2, &token3);
            prop_assert_eq!(family3.rotation_count, 2);

            // Old token (token1) should fail - this will revoke the family
            let old_result = rotator.rotate(&token1, None).await;
            prop_assert!(old_result.is_err(), "Old token must be invalid");

            Ok(())
        })?;
    }

    /// Property 8: Refresh Token Replay Detection
    ///
    /// Using an old token after rotation must be detected as replay
    /// and revoke the entire family.
    #[test]
    fn prop_refresh_token_replay_detection(
        user_id in arb_user_id(),
        session_id in arb_session_id(),
    ) {
        let rt = tokio::runtime::Runtime::new().unwrap();
        rt.block_on(async {
            let rotator = create_test_rotator().await;

            // Create and rotate
            let (token1, _) = rotator
                .create_token_family(&user_id, &session_id, None)
                .await
                .unwrap();

            let (token2, _) = rotator.rotate(&token1, None).await.unwrap();

            // Replay with old token should fail with RefreshReplay
            let replay_result = rotator.rotate(&token1, None).await;
            prop_assert!(
                matches!(replay_result, Err(token_service::error::TokenError::RefreshReplay)),
                "Replay must be detected"
            );

            // After replay detection, even the new token should fail (family revoked)
            let new_result = rotator.rotate(&token2, None).await;
            prop_assert!(
                matches!(new_result, Err(token_service::error::TokenError::FamilyRevoked)),
                "Family must be revoked after replay"
            );

            Ok(())
        })?;
    }

    /// Property 9: Token Family Uniqueness
    ///
    /// Each token family must have a unique family_id.
    #[test]
    fn prop_token_family_uniqueness(
        user_id in arb_user_id(),
        session_id in arb_session_id(),
    ) {
        let rt = tokio::runtime::Runtime::new().unwrap();
        rt.block_on(async {
            let rotator = create_test_rotator().await;

            let mut family_ids = std::collections::HashSet::new();

            // Create multiple families
            for i in 0..10 {
                let session = format!("{}-{}", session_id, i);
                let (_, family) = rotator
                    .create_token_family(&user_id, &session, None)
                    .await
                    .unwrap();

                prop_assert!(
                    family_ids.insert(family.family_id.clone()),
                    "Family ID must be unique: {}",
                    family.family_id
                );
            }

            Ok(())
        })?;
    }

    /// Property 10: Token Family Revocation
    ///
    /// After revocation, all tokens in the family must be invalid.
    #[test]
    fn prop_token_family_revocation(
        user_id in arb_user_id(),
        session_id in arb_session_id(),
    ) {
        let rt = tokio::runtime::Runtime::new().unwrap();
        rt.block_on(async {
            let rotator = create_test_rotator().await;

            // Create token family
            let (token, family) = rotator
                .create_token_family(&user_id, &session_id, None)
                .await
                .unwrap();

            // Revoke the family
            rotator.revoke_family(&family.family_id, None).await.unwrap();

            // Token should now fail with FamilyRevoked
            let result = rotator.rotate(&token, None).await;
            prop_assert!(
                matches!(result, Err(token_service::error::TokenError::FamilyRevoked)),
                "Revoked family tokens must fail"
            );

            Ok(())
        })?;
    }

    /// Property: Multiple rotations maintain family integrity.
    #[test]
    fn prop_multiple_rotations_integrity(
        user_id in arb_user_id(),
        session_id in arb_session_id(),
        rotation_count in 1usize..10,
    ) {
        let rt = tokio::runtime::Runtime::new().unwrap();
        rt.block_on(async {
            let rotator = create_test_rotator().await;

            let (mut current_token, initial_family) = rotator
                .create_token_family(&user_id, &session_id, None)
                .await
                .unwrap();

            let family_id = initial_family.family_id.clone();

            for i in 0..rotation_count {
                let (new_token, family) = rotator.rotate(&current_token, None).await.unwrap();

                prop_assert_eq!(
                    &family.family_id, &family_id,
                    "Family ID must be preserved"
                );
                prop_assert_eq!(
                    family.rotation_count as usize, i + 1,
                    "Rotation count must increment"
                );
                prop_assert_eq!(
                    &family.user_id, &user_id,
                    "User ID must be preserved"
                );

                current_token = new_token;
            }

            Ok(())
        })?;
    }
}

#[cfg(test)]
mod unit_tests {
    use super::*;
    use token_service::refresh::TokenFamily;

    #[test]
    fn test_token_family_creation() {
        let family = TokenFamily::new(
            "family-1".to_string(),
            "user-1".to_string(),
            "session-1".to_string(),
            "hash-1".to_string(),
        );

        assert_eq!(family.rotation_count, 0);
        assert!(!family.revoked);
        assert!(family.is_valid_token("hash-1"));
    }

    #[test]
    fn test_token_family_rotation() {
        let mut family = TokenFamily::new(
            "family-1".to_string(),
            "user-1".to_string(),
            "session-1".to_string(),
            "hash-1".to_string(),
        );

        family.rotate("hash-2".to_string());

        assert_eq!(family.rotation_count, 1);
        assert!(!family.is_valid_token("hash-1"));
        assert!(family.is_valid_token("hash-2"));
    }

    #[test]
    fn test_token_family_revocation() {
        let mut family = TokenFamily::new(
            "family-1".to_string(),
            "user-1".to_string(),
            "session-1".to_string(),
            "hash-1".to_string(),
        );

        family.revoke();

        assert!(family.revoked);
        assert!(family.revoked_at.is_some());
        assert!(!family.is_valid_token("hash-1"));
    }

    #[test]
    fn test_replay_detection() {
        let mut family = TokenFamily::new(
            "family-1".to_string(),
            "user-1".to_string(),
            "session-1".to_string(),
            "hash-1".to_string(),
        );

        // Before rotation, old hash is valid
        assert!(!family.is_replay_attack("hash-1"));

        // After rotation, old hash is replay
        family.rotate("hash-2".to_string());
        assert!(family.is_replay_attack("hash-1"));
        assert!(!family.is_replay_attack("hash-2"));
    }
}
