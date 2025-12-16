use crate::config::Config;
use crate::error::TokenError;
use crate::jwt::{JwtBuilder, JwtSerializer};
use crate::jwks::{Jwk, JwksPublisher};
use crate::kms::MockKms;
use crate::refresh::RefreshTokenRotator;
use crate::storage::RedisStorage;
use crate::proto::common::Empty;
use crate::proto::token::token_service_server::TokenService;
use crate::proto::token::*;
use jsonwebtoken::Algorithm;
use std::sync::Arc;
use tonic::{Request, Response, Status};
use tracing::{info, error};

pub struct TokenServiceImpl {
    config: Config,
    storage: Arc<RedisStorage>,
    rotator: RefreshTokenRotator,
    jwks_publisher: JwksPublisher,
    serializer: JwtSerializer,
    kms: MockKms,
}

impl TokenServiceImpl {
    pub async fn new(config: Config) -> Result<Self, TokenError> {
        let storage = Arc::new(RedisStorage::new(&config.redis_url).await?);
        let rotator = RefreshTokenRotator::new(storage.clone());
        let jwks_publisher = JwksPublisher::new();
        let serializer = JwtSerializer::new(&config.jwt_algorithm);
        let kms = MockKms::new(config.kms_key_id.clone());

        // Initialize with a default key
        let initial_key = Jwk {
            kty: "oct".to_string(),
            kid: config.kms_key_id.clone(),
            key_use: "sig".to_string(),
            alg: "HS256".to_string(),
            n: None,
            e: None,
            x: None,
            y: None,
            crv: None,
        };
        jwks_publisher.add_key(initial_key).await;

        Ok(TokenServiceImpl {
            config,
            storage,
            rotator,
            jwks_publisher,
            serializer,
            kms,
        })
    }
}

#[tonic::async_trait]
impl TokenService for TokenServiceImpl {
    async fn issue_token_pair(
        &self,
        request: Request<IssueTokenRequest>,
    ) -> Result<Response<TokenPairResponse>, Status> {
        let req = request.into_inner();
        
        let access_ttl = if req.access_token_ttl_seconds > 0 {
            req.access_token_ttl_seconds as i64
        } else {
            self.config.access_token_ttl_seconds
        };

        let refresh_ttl = if req.refresh_token_ttl_seconds > 0 {
            req.refresh_token_ttl_seconds as i64
        } else {
            self.config.refresh_token_ttl_seconds
        };

        // Build access token claims
        let mut builder = JwtBuilder::new(self.config.jwt_issuer.clone())
            .subject(req.user_id.clone())
            .audience(vec!["api".to_string()])
            .ttl_seconds(access_ttl)
            .scopes(req.scopes.clone());

        if !req.session_id.is_empty() {
            builder = builder.session_id(req.session_id.clone());
        }

        for (key, value) in req.custom_claims {
            builder = builder.custom_claim(key, serde_json::Value::String(value));
        }

        let claims = builder.build().map_err(|e| Status::invalid_argument(e))?;

        // Serialize access token
        let encoding_key = self.kms.get_encoding_key()
            .map_err(|e| Status::internal(e.to_string()))?;
        
        let access_token = JwtSerializer { algorithm: Algorithm::HS256 }
            .serialize(&claims, &encoding_key, Some(&self.config.kms_key_id))
            .map_err(|e| Status::internal(e.to_string()))?;

        // Create refresh token family
        let (refresh_token, _family) = self.rotator
            .create_token_family(&req.user_id, &req.session_id, refresh_ttl)
            .await
            .map_err(|e| Status::internal(e.to_string()))?;

        let expires_at = chrono::Utc::now().timestamp() + access_ttl;

        info!(
            user_id = %req.user_id,
            session_id = %req.session_id,
            "Issued token pair"
        );

        Ok(Response::new(TokenPairResponse {
            access_token,
            refresh_token,
            id_token: String::new(), // ID token would be generated for OIDC flows
            expires_at,
            token_type: "Bearer".to_string(),
        }))
    }

    async fn refresh_tokens(
        &self,
        request: Request<RefreshRequest>,
    ) -> Result<Response<TokenPairResponse>, Status> {
        let req = request.into_inner();

        let (new_refresh_token, family) = self.rotator
            .rotate(&req.refresh_token, self.config.refresh_token_ttl_seconds)
            .await
            .map_err(|e| match e {
                TokenError::RefreshInvalid(_) => Status::unauthenticated("Invalid refresh token"),
                TokenError::RefreshReused => Status::unauthenticated("Token replay detected"),
                TokenError::FamilyRevoked => Status::unauthenticated("Token family revoked"),
                _ => Status::internal(e.to_string()),
            })?;

        // Build new access token
        let claims = JwtBuilder::new(self.config.jwt_issuer.clone())
            .subject(family.user_id.clone())
            .audience(vec!["api".to_string()])
            .ttl_seconds(self.config.access_token_ttl_seconds)
            .session_id(family.session_id.clone())
            .scopes(req.scopes)
            .build()
            .map_err(|e| Status::internal(e))?;

        let encoding_key = self.kms.get_encoding_key()
            .map_err(|e| Status::internal(e.to_string()))?;

        let access_token = JwtSerializer { algorithm: Algorithm::HS256 }
            .serialize(&claims, &encoding_key, Some(&self.config.kms_key_id))
            .map_err(|e| Status::internal(e.to_string()))?;

        let expires_at = chrono::Utc::now().timestamp() + self.config.access_token_ttl_seconds;

        info!(
            user_id = %family.user_id,
            rotation_count = %family.rotation_count,
            "Refreshed tokens"
        );

        Ok(Response::new(TokenPairResponse {
            access_token,
            refresh_token: new_refresh_token,
            id_token: String::new(),
            expires_at,
            token_type: "Bearer".to_string(),
        }))
    }

    async fn revoke_token(
        &self,
        request: Request<RevokeRequest>,
    ) -> Result<Response<RevokeResponse>, Status> {
        let req = request.into_inner();

        // For refresh tokens, revoke the family
        // For access tokens, add to revocation list
        if req.token_type_hint == "refresh_token" {
            // Find and revoke the family
            let token_hash = crate::refresh::RefreshTokenGenerator::hash(&req.token);
            if let Ok(Some(family)) = self.storage.find_family_by_token_hash(&token_hash).await {
                self.rotator.revoke_family(&family.family_id).await
                    .map_err(|e| Status::internal(e.to_string()))?;
            }
        } else {
            // Add access token JTI to revocation list
            // In production, we'd decode the token to get the JTI
            self.storage.add_to_revocation_list(&req.token, self.config.access_token_ttl_seconds)
                .await
                .map_err(|e| Status::internal(e.to_string()))?;
        }

        info!("Revoked token");

        Ok(Response::new(RevokeResponse { success: true }))
    }

    async fn revoke_all_user_tokens(
        &self,
        request: Request<RevokeAllRequest>,
    ) -> Result<Response<RevokeResponse>, Status> {
        let req = request.into_inner();

        self.rotator.revoke_all_user_tokens(&req.user_id)
            .await
            .map_err(|e| Status::internal(e.to_string()))?;

        info!(user_id = %req.user_id, "Revoked all user tokens");

        Ok(Response::new(RevokeResponse { success: true }))
    }

    async fn get_jwks(
        &self,
        _request: Request<Empty>,
    ) -> Result<Response<JwksResponse>, Status> {
        let jwks = self.jwks_publisher.get_jwks().await;
        
        Ok(Response::new(JwksResponse {
            keys_json: jwks.to_json(),
        }))
    }

    async fn rotate_signing_key(
        &self,
        request: Request<RotateKeyRequest>,
    ) -> Result<Response<RotateKeyResponse>, Status> {
        let req = request.into_inner();

        let new_key = Jwk {
            kty: "oct".to_string(),
            kid: req.key_id.clone(),
            key_use: "sig".to_string(),
            alg: "HS256".to_string(),
            n: None,
            e: None,
            x: None,
            y: None,
            crv: None,
        };

        self.jwks_publisher.rotate_keys(new_key).await;

        info!(new_key_id = %req.key_id, "Rotated signing key");

        Ok(Response::new(RotateKeyResponse {
            success: true,
            new_key_id: req.key_id,
        }))
    }
}
