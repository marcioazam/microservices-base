use crate::error::TokenError;
use crate::refresh::family::TokenFamily;
use crate::refresh::generator::RefreshTokenGenerator;
use crate::storage::redis::RedisStorage;
use std::sync::Arc;
use tracing::{info, warn};

pub struct RefreshTokenRotator {
    storage: Arc<RedisStorage>,
}

impl RefreshTokenRotator {
    pub fn new(storage: Arc<RedisStorage>) -> Self {
        RefreshTokenRotator { storage }
    }

    pub async fn create_token_family(
        &self,
        user_id: &str,
        session_id: &str,
        ttl_seconds: i64,
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

        self.storage.store_token_family(&family, ttl_seconds).await?;

        info!(
            family_id = %family_id,
            user_id = %user_id,
            "Created new token family"
        );

        Ok((token, family))
    }

    pub async fn rotate(
        &self,
        refresh_token: &str,
        ttl_seconds: i64,
    ) -> Result<(String, TokenFamily), TokenError> {
        let token_hash = RefreshTokenGenerator::hash(refresh_token);

        // Find the family by token hash
        let mut family = self.storage
            .find_family_by_token_hash(&token_hash)
            .await?
            .ok_or_else(|| TokenError::RefreshInvalid("Token not found".to_string()))?;

        // Check if family is revoked
        if family.revoked {
            return Err(TokenError::FamilyRevoked);
        }

        // Check if this is a replay attack (token already rotated)
        if family.is_replay_attack(&token_hash) {
            warn!(
                family_id = %family.family_id,
                user_id = %family.user_id,
                "Replay attack detected - revoking token family"
            );
            
            family.revoke();
            self.storage.store_token_family(&family, ttl_seconds).await?;
            
            // Log security event
            self.log_security_event(&family, "REPLAY_ATTACK_DETECTED").await;
            
            return Err(TokenError::RefreshReused);
        }

        // Generate new token and rotate
        let new_token = RefreshTokenGenerator::generate();
        let new_token_hash = RefreshTokenGenerator::hash(&new_token);

        family.rotate(new_token_hash);
        self.storage.store_token_family(&family, ttl_seconds).await?;

        info!(
            family_id = %family.family_id,
            rotation_count = %family.rotation_count,
            "Rotated refresh token"
        );

        Ok((new_token, family))
    }

    pub async fn revoke_family(&self, family_id: &str) -> Result<(), TokenError> {
        if let Some(mut family) = self.storage.get_token_family(family_id).await? {
            family.revoke();
            self.storage.store_token_family(&family, 86400).await?; // Keep for 24h for audit
            
            info!(family_id = %family_id, "Revoked token family");
        }
        Ok(())
    }

    pub async fn revoke_all_user_tokens(&self, user_id: &str) -> Result<u32, TokenError> {
        let families = self.storage.get_user_token_families(user_id).await?;
        let count = families.len() as u32;

        for mut family in families {
            family.revoke();
            self.storage.store_token_family(&family, 86400).await?;
        }

        info!(user_id = %user_id, count = %count, "Revoked all user token families");
        Ok(count)
    }

    async fn log_security_event(&self, family: &TokenFamily, event_type: &str) {
        // In production, this would emit to Kafka/NATS
        warn!(
            event_type = %event_type,
            family_id = %family.family_id,
            user_id = %family.user_id,
            session_id = %family.session_id,
            "Security event"
        );
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    // Integration tests would go here with mock storage
}
