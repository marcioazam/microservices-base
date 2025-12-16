use serde::{Deserialize, Serialize};
use chrono::{DateTime, Utc};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TokenFamily {
    pub family_id: String,
    pub user_id: String,
    pub session_id: String,
    pub current_token_hash: String,
    pub rotation_count: u32,
    pub created_at: DateTime<Utc>,
    pub revoked: bool,
    pub revoked_at: Option<DateTime<Utc>>,
}

impl TokenFamily {
    pub fn new(family_id: String, user_id: String, session_id: String, token_hash: String) -> Self {
        TokenFamily {
            family_id,
            user_id,
            session_id,
            current_token_hash: token_hash,
            rotation_count: 0,
            created_at: Utc::now(),
            revoked: false,
            revoked_at: None,
        }
    }

    pub fn rotate(&mut self, new_token_hash: String) {
        self.current_token_hash = new_token_hash;
        self.rotation_count += 1;
    }

    pub fn revoke(&mut self) {
        self.revoked = true;
        self.revoked_at = Some(Utc::now());
    }

    pub fn is_valid_token(&self, token_hash: &str) -> bool {
        !self.revoked && self.current_token_hash == token_hash
    }

    pub fn is_replay_attack(&self, token_hash: &str) -> bool {
        !self.revoked && self.current_token_hash != token_hash
    }
}

#[cfg(test)]
mod tests {
    use super::*;

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
    fn test_token_rotation() {
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
        assert!(family.is_replay_attack("hash-1"));
    }

    #[test]
    fn test_token_revocation() {
        let mut family = TokenFamily::new(
            "family-1".to_string(),
            "user-1".to_string(),
            "session-1".to_string(),
            "hash-1".to_string(),
        );

        family.revoke();

        assert!(family.revoked);
        assert!(!family.is_valid_token("hash-1"));
    }
}
