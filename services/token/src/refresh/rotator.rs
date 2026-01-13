//! Refresh token rotation with replay detection.
//!
//! Uses CacheStorage for persistence and LoggingClient for security events.

use crate::error::TokenError;
use crate::refresh::family::TokenFamily;
use crate::refresh::generator::RefreshTokenGenerator;
use crate::storage::CacheStorage;
use rust_common::{LogEntry, LogLevel, LoggingClient};
use std::sync::Arc;
use std::time::Duration;
use tracing::{info, warn};

/// Refresh token rotator with replay detection.
pub struct RefreshTokenRotator {
    storage: Arc<CacheStorage>,
    logger: Arc<LoggingClient>,
    default_ttl: Duration,
}

impl RefreshTokenRotator {
    /// Create a new rotator with cache storage and logging.
    pub fn new(
        storage: Arc<CacheStorage>,
        logger: Arc<LoggingClient>,
        default_ttl: Duration,
    ) -> Self {
        Self {
            storage,
            logger,
            default_ttl,
        }
    }

    /// Create a new token family for a user session.
    pub async fn create_token_family(
        &self,
        user_id: &str,
        session_id: &str,
        correlation_id: Option<&str>,
    ) -> Result<(String, TokenFamily), TokenError> {
        let token = RefreshTokenGenerator::generate();
        let token_hash = RefreshTokenGenerator::hash(&token);
        let family_id = RefreshTokenGenerator::generate_family_id();

        let family = TokenFamily::new(
            family_id.clone(),
            user_id.to_string(),
            session_id.to_string(),
            token_hash,
        );

        self.storage
            .store_token_family(&family, Some(self.default_ttl))
            .await?;

        info!(
            family_id = %family_id,
            user_id = %user_id,
            "Created new token family"
        );

        self.log_security_event(
            "TOKEN_FAMILY_CREATED",
            &family,
            correlation_id,
        ).await;

        Ok((token, family))
    }

    /// Rotate a refresh token, returning a new token.
    ///
    /// Detects replay attacks and revokes the entire family if detected.
    pub async fn rotate(
        &self,
        refresh_token: &str,
        correlation_id: Option<&str>,
    ) -> Result<(String, TokenFamily), TokenError> {
        let token_hash = RefreshTokenGenerator::hash(refresh_token);

        let mut family = self.storage
            .find_family_by_token_hash(&token_hash)
            .await?
            .ok_or(TokenError::RefreshInvalid)?;

        // Check if family is revoked
        if family.revoked {
            return Err(TokenError::FamilyRevoked);
        }

        // Check for replay attack
        if family.is_replay_attack(&token_hash) {
            warn!(
                family_id = %family.family_id,
                user_id = %family.user_id,
                "Replay attack detected - revoking token family"
            );

            family.revoke();
            self.storage
                .store_token_family(&family, Some(Duration::from_secs(86400)))
                .await?;

            self.log_security_event(
                "REPLAY_ATTACK_DETECTED",
                &family,
                correlation_id,
            ).await;

            return Err(TokenError::RefreshReplay);
        }

        // Generate new token and rotate
        let new_token = RefreshTokenGenerator::generate();
        let new_token_hash = RefreshTokenGenerator::hash(&new_token);

        family.rotate(new_token_hash);
        self.storage
            .store_token_family(&family, Some(self.default_ttl))
            .await?;

        info!(
            family_id = %family.family_id,
            rotation_count = %family.rotation_count,
            "Rotated refresh token"
        );

        self.log_security_event(
            "TOKEN_ROTATED",
            &family,
            correlation_id,
        ).await;

        Ok((new_token, family))
    }

    /// Revoke a token family by ID.
    pub async fn revoke_family(
        &self,
        family_id: &str,
        correlation_id: Option<&str>,
    ) -> Result<(), TokenError> {
        if let Some(mut family) = self.storage.get_token_family(family_id).await? {
            family.revoke();
            self.storage
                .store_token_family(&family, Some(Duration::from_secs(86400)))
                .await?;

            info!(family_id = %family_id, "Revoked token family");

            self.log_security_event(
                "TOKEN_FAMILY_REVOKED",
                &family,
                correlation_id,
            ).await;
        }
        Ok(())
    }

    /// Revoke all token families for a user.
    pub async fn revoke_all_user_tokens(
        &self,
        user_id: &str,
        correlation_id: Option<&str>,
    ) -> Result<u32, TokenError> {
        let families = self.storage.get_user_token_families(user_id).await?;
        let count = families.len() as u32;

        for mut family in families {
            family.revoke();
            self.storage
                .store_token_family(&family, Some(Duration::from_secs(86400)))
                .await?;

            self.log_security_event(
                "TOKEN_FAMILY_REVOKED",
                &family,
                correlation_id,
            ).await;
        }

        info!(user_id = %user_id, count = %count, "Revoked all user token families");
        Ok(count)
    }

    /// Log a security event to the centralized logging service.
    async fn log_security_event(
        &self,
        event_type: &str,
        family: &TokenFamily,
        correlation_id: Option<&str>,
    ) {
        let mut entry = LogEntry::new(
            LogLevel::Warn,
            format!("Security event: {}", event_type),
            "token-service",
        )
        .with_metadata("event_type", event_type)
        .with_metadata("family_id", &family.family_id)
        .with_metadata("user_id", &family.user_id)
        .with_metadata("session_id", &family.session_id)
        .with_metadata("rotation_count", family.rotation_count.to_string());

        if let Some(cid) = correlation_id {
            entry = entry.with_correlation_id(cid);
        }

        self.logger.log(entry).await;
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use rust_common::{CacheClientConfig, LoggingClientConfig};

    async fn create_test_rotator() -> RefreshTokenRotator {
        let cache_config = CacheClientConfig::default()
            .with_namespace("rotator-test");
        let storage = Arc::new(CacheStorage::new(cache_config).await.unwrap());

        let log_config = LoggingClientConfig::default()
            .with_service_id("token-service-test");
        let logger = Arc::new(LoggingClient::new(log_config).await.unwrap());

        RefreshTokenRotator::new(storage, logger, Duration::from_secs(604800))
    }

    #[tokio::test]
    async fn test_create_token_family() {
        let rotator = create_test_rotator().await;

        let (token, family) = rotator
            .create_token_family("user-1", "session-1", Some("corr-1"))
            .await
            .unwrap();

        assert!(!token.is_empty());
        assert_eq!(family.user_id, "user-1");
        assert_eq!(family.session_id, "session-1");
        assert_eq!(family.rotation_count, 0);
    }

    #[tokio::test]
    async fn test_rotate_token() {
        let rotator = create_test_rotator().await;

        let (token1, family1) = rotator
            .create_token_family("user-2", "session-2", None)
            .await
            .unwrap();

        let (token2, family2) = rotator.rotate(&token1, None).await.unwrap();

        assert_ne!(token1, token2);
        assert_eq!(family2.family_id, family1.family_id);
        assert_eq!(family2.rotation_count, 1);
    }

    #[tokio::test]
    async fn test_replay_detection() {
        let rotator = create_test_rotator().await;

        let (token1, _) = rotator
            .create_token_family("user-3", "session-3", None)
            .await
            .unwrap();

        // First rotation succeeds
        let (_, _) = rotator.rotate(&token1, None).await.unwrap();

        // Replay with old token fails
        let result = rotator.rotate(&token1, None).await;
        assert!(matches!(result, Err(TokenError::RefreshReplay)));
    }

    #[tokio::test]
    async fn test_revoke_family() {
        let rotator = create_test_rotator().await;

        let (token, family) = rotator
            .create_token_family("user-4", "session-4", None)
            .await
            .unwrap();

        rotator.revoke_family(&family.family_id, None).await.unwrap();

        let result = rotator.rotate(&token, None).await;
        assert!(matches!(result, Err(TokenError::FamilyRevoked)));
    }
}
