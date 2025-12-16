use crate::error::TokenError;
use crate::refresh::family::TokenFamily;
use redis::aio::ConnectionManager;
use redis::AsyncCommands;
use std::sync::Arc;
use tokio::sync::RwLock;

pub struct RedisStorage {
    conn: Arc<RwLock<ConnectionManager>>,
}

impl RedisStorage {
    pub async fn new(redis_url: &str) -> Result<Self, TokenError> {
        let client = redis::Client::open(redis_url)
            .map_err(|e| TokenError::RedisError(e.to_string()))?;
        
        let conn = ConnectionManager::new(client)
            .await
            .map_err(|e| TokenError::RedisError(e.to_string()))?;

        Ok(RedisStorage {
            conn: Arc::new(RwLock::new(conn)),
        })
    }

    pub async fn store_token_family(&self, family: &TokenFamily, ttl_seconds: i64) -> Result<(), TokenError> {
        let mut conn = self.conn.write().await;
        let key = format!("token_family:{}", family.family_id);
        let value = serde_json::to_string(family)
            .map_err(|e| TokenError::Internal(e.to_string()))?;

        conn.set_ex::<_, _, ()>(&key, &value, ttl_seconds as u64)
            .await
            .map_err(|e| TokenError::RedisError(e.to_string()))?;

        // Index by token hash for lookup
        let hash_key = format!("token_hash:{}", family.current_token_hash);
        conn.set_ex::<_, _, ()>(&hash_key, &family.family_id, ttl_seconds as u64)
            .await
            .map_err(|e| TokenError::RedisError(e.to_string()))?;

        // Index by user for revocation
        let user_key = format!("user_families:{}", family.user_id);
        conn.sadd::<_, _, ()>(&user_key, &family.family_id)
            .await
            .map_err(|e| TokenError::RedisError(e.to_string()))?;

        Ok(())
    }

    pub async fn get_token_family(&self, family_id: &str) -> Result<Option<TokenFamily>, TokenError> {
        let mut conn = self.conn.write().await;
        let key = format!("token_family:{}", family_id);

        let value: Option<String> = conn.get(&key)
            .await
            .map_err(|e| TokenError::RedisError(e.to_string()))?;

        match value {
            Some(v) => {
                let family: TokenFamily = serde_json::from_str(&v)
                    .map_err(|e| TokenError::Internal(e.to_string()))?;
                Ok(Some(family))
            }
            None => Ok(None),
        }
    }

    pub async fn find_family_by_token_hash(&self, token_hash: &str) -> Result<Option<TokenFamily>, TokenError> {
        let mut conn = self.conn.write().await;
        let hash_key = format!("token_hash:{}", token_hash);

        let family_id: Option<String> = conn.get(&hash_key)
            .await
            .map_err(|e| TokenError::RedisError(e.to_string()))?;

        match family_id {
            Some(id) => self.get_token_family(&id).await,
            None => Ok(None),
        }
    }

    pub async fn get_user_token_families(&self, user_id: &str) -> Result<Vec<TokenFamily>, TokenError> {
        let mut conn = self.conn.write().await;
        let user_key = format!("user_families:{}", user_id);

        let family_ids: Vec<String> = conn.smembers(&user_key)
            .await
            .map_err(|e| TokenError::RedisError(e.to_string()))?;

        let mut families = Vec::new();
        for id in family_ids {
            if let Some(family) = self.get_token_family(&id).await? {
                families.push(family);
            }
        }

        Ok(families)
    }

    pub async fn add_to_revocation_list(&self, jti: &str, ttl_seconds: i64) -> Result<(), TokenError> {
        let mut conn = self.conn.write().await;
        let key = format!("revoked:{}", jti);

        conn.set_ex::<_, _, ()>(&key, "1", ttl_seconds as u64)
            .await
            .map_err(|e| TokenError::RedisError(e.to_string()))?;

        Ok(())
    }

    pub async fn is_token_revoked(&self, jti: &str) -> Result<bool, TokenError> {
        let mut conn = self.conn.write().await;
        let key = format!("revoked:{}", jti);

        let exists: bool = conn.exists(&key)
            .await
            .map_err(|e| TokenError::RedisError(e.to_string()))?;

        Ok(exists)
    }
}
