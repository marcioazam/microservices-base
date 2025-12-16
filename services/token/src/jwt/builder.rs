use crate::jwt::claims::Claims;
use std::collections::HashMap;

pub struct JwtBuilder {
    issuer: String,
    subject: Option<String>,
    audience: Vec<String>,
    ttl_seconds: i64,
    nonce: Option<String>,
    session_id: Option<String>,
    scopes: Vec<String>,
    custom_claims: HashMap<String, serde_json::Value>,
}

impl JwtBuilder {
    pub fn new(issuer: String) -> Self {
        JwtBuilder {
            issuer,
            subject: None,
            audience: Vec::new(),
            ttl_seconds: 900, // 15 minutes default
            nonce: None,
            session_id: None,
            scopes: Vec::new(),
            custom_claims: HashMap::new(),
        }
    }

    pub fn subject(mut self, subject: String) -> Self {
        self.subject = Some(subject);
        self
    }

    pub fn audience(mut self, audience: Vec<String>) -> Self {
        self.audience = audience;
        self
    }

    pub fn ttl_seconds(mut self, ttl: i64) -> Self {
        self.ttl_seconds = ttl;
        self
    }

    pub fn nonce(mut self, nonce: String) -> Self {
        self.nonce = Some(nonce);
        self
    }

    pub fn session_id(mut self, session_id: String) -> Self {
        self.session_id = Some(session_id);
        self
    }

    pub fn scopes(mut self, scopes: Vec<String>) -> Self {
        self.scopes = scopes;
        self
    }

    pub fn custom_claim(mut self, key: String, value: serde_json::Value) -> Self {
        self.custom_claims.insert(key, value);
        self
    }

    pub fn build(self) -> Result<Claims, &'static str> {
        let subject = self.subject.ok_or("Subject is required")?;

        let mut claims = Claims::new(
            self.issuer,
            subject,
            self.audience,
            self.ttl_seconds,
        );

        if let Some(nonce) = self.nonce {
            claims = claims.with_nonce(nonce);
        }

        if let Some(session_id) = self.session_id {
            claims = claims.with_session_id(session_id);
        }

        if !self.scopes.is_empty() {
            claims = claims.with_scopes(self.scopes);
        }

        for (key, value) in self.custom_claims {
            claims = claims.with_custom_claim(key, value);
        }

        Ok(claims)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_builder_basic() {
        let claims = JwtBuilder::new("issuer".to_string())
            .subject("user-123".to_string())
            .audience(vec!["api".to_string()])
            .ttl_seconds(3600)
            .build()
            .unwrap();

        assert_eq!(claims.iss, "issuer");
        assert_eq!(claims.sub, "user-123");
    }

    #[test]
    fn test_builder_missing_subject() {
        let result = JwtBuilder::new("issuer".to_string()).build();
        assert!(result.is_err());
    }
}
