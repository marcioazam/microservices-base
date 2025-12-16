//! gRPC Service Implementation
//!
//! Implements the AuthEdgeService with type-state JWT validation,
//! Tower middleware stack, and proper error handling with correlation IDs.

use crate::config::Config;
use crate::error::{AuthEdgeError, ErrorResponse, ErrorCode};
use crate::jwt::{JwkCache, JwtValidator, Token, Validated};
use crate::mtls::SpiffeExtractor;
use crate::circuit_breaker::StandaloneCircuitBreaker;
use crate::proto::edge::auth_edge_service_server::AuthEdgeService;
use crate::proto::edge::*;
use std::sync::Arc;
use tonic::{Request, Response, Status};
use tracing::{info, error, instrument, Span};
use uuid::Uuid;

/// Auth Edge Service implementation with modern patterns
pub struct AuthEdgeServiceImpl {
    config: Config,
    jwt_validator: JwtValidator,
    token_service_cb: StandaloneCircuitBreaker,
    iam_service_cb: StandaloneCircuitBreaker,
}

impl AuthEdgeServiceImpl {
    /// Creates a new AuthEdgeServiceImpl with all dependencies
    pub async fn new(config: Config) -> Result<Self, AuthEdgeError> {
        let jwk_cache = Arc::new(JwkCache::new(
            config.jwks_url.clone(),
            config.jwks_cache_ttl_seconds,
        ));

        let jwt_validator = JwtValidator::new(jwk_cache);

        let token_service_cb = StandaloneCircuitBreaker::from_config(
            "token-service",
            config.circuit_breaker_failure_threshold,
            config.circuit_breaker_timeout_seconds,
        );

        let iam_service_cb = StandaloneCircuitBreaker::from_config(
            "iam-service",
            config.circuit_breaker_failure_threshold,
            config.circuit_breaker_timeout_seconds,
        );

        Ok(AuthEdgeServiceImpl {
            config,
            jwt_validator,
            token_service_cb,
            iam_service_cb,
        })
    }

    /// Generates a new correlation ID for request tracing
    fn generate_correlation_id() -> Uuid {
        Uuid::new_v4()
    }

    /// Converts an AuthEdgeError to a ValidateTokenResponse with proper sanitization
    fn error_to_response(err: &AuthEdgeError, correlation_id: Uuid) -> ValidateTokenResponse {
        let response = ErrorResponse::from_error(err, correlation_id);
        
        ValidateTokenResponse {
            valid: false,
            subject: String::new(),
            claims: std::collections::HashMap::new(),
            error_code: response.code.as_str().to_string(),
            error_message: format!("{} [correlation_id: {}]", response.message, correlation_id),
        }
    }

    /// Converts an AuthEdgeError to gRPC Status with correlation ID
    fn error_to_status(err: &AuthEdgeError, correlation_id: Uuid) -> Status {
        err.to_status(correlation_id)
    }
}

#[tonic::async_trait]
impl AuthEdgeService for AuthEdgeServiceImpl {
    #[instrument(
        skip(self, request),
        fields(correlation_id = %Self::generate_correlation_id())
    )]
    async fn validate_token(
        &self,
        request: Request<ValidateTokenRequest>,
    ) -> Result<Response<ValidateTokenResponse>, Status> {
        let correlation_id = Self::generate_correlation_id();
        let req = request.into_inner();

        // Check for missing token
        if req.token.is_empty() {
            let err = AuthEdgeError::TokenMissing;
            error!(
                correlation_id = %correlation_id,
                error_type = "TokenMissing",
                "Token validation failed: token missing"
            );
            return Ok(Response::new(Self::error_to_response(&err, correlation_id)));
        }

        // Use type-state JWT validation
        let required_refs: Vec<&str> = req.required_claims.iter().map(|s| s.as_str()).collect();
        
        match self.jwt_validator.validate_token(&req.token, &required_refs).await {
            Ok(validated_token) => {
                let claims = validated_token.claims();
                
                info!(
                    subject = %claims.sub,
                    correlation_id = %correlation_id,
                    "Token validated successfully"
                );

                Ok(Response::new(ValidateTokenResponse {
                    valid: true,
                    subject: claims.sub.clone(),
                    claims: claims.to_map(),
                    error_code: String::new(),
                    error_message: String::new(),
                }))
            }
            Err(err) => {
                error!(
                    error = %err,
                    correlation_id = %correlation_id,
                    error_type = ?err.code(),
                    "Token validation failed"
                );

                Ok(Response::new(Self::error_to_response(&err, correlation_id)))
            }
        }
    }

    #[instrument(skip(self, request))]
    async fn introspect_token(
        &self,
        request: Request<IntrospectRequest>,
    ) -> Result<Response<IntrospectResponse>, Status> {
        let correlation_id = Self::generate_correlation_id();
        let req = request.into_inner();

        // For introspection, we validate without required claims
        match self.jwt_validator.validate_token(&req.token, &[]).await {
            Ok(validated_token) => {
                let claims = validated_token.claims();
                
                Ok(Response::new(IntrospectResponse {
                    active: !claims.is_expired(),
                    subject: claims.sub.clone(),
                    client_id: claims.custom.get("client_id")
                        .and_then(|v| v.as_str())
                        .unwrap_or("")
                        .to_string(),
                    scopes: claims.scopes.clone().unwrap_or_default(),
                    expires_at: claims.exp,
                    issued_at: claims.iat,
                    token_type: "Bearer".to_string(),
                    claims: claims.to_map(),
                }))
            }
            Err(err) => {
                info!(
                    correlation_id = %correlation_id,
                    "Token introspection: token inactive"
                );
                
                Ok(Response::new(IntrospectResponse {
                    active: false,
                    subject: String::new(),
                    client_id: String::new(),
                    scopes: Vec::new(),
                    expires_at: 0,
                    issued_at: 0,
                    token_type: String::new(),
                    claims: std::collections::HashMap::new(),
                }))
            }
        }
    }

    #[instrument(skip(self, request))]
    async fn get_service_identity(
        &self,
        request: Request<IdentityRequest>,
    ) -> Result<Response<IdentityResponse>, Status> {
        let correlation_id = Self::generate_correlation_id();
        let req = request.into_inner();

        match SpiffeExtractor::extract_spiffe_id(&req.certificate_pem) {
            Ok(spiffe_id) => {
                let service_name = SpiffeExtractor::extract_service_name(&spiffe_id)
                    .unwrap_or_default();

                info!(
                    spiffe_id = %spiffe_id,
                    service_name = %service_name,
                    correlation_id = %correlation_id,
                    "Service identity extracted"
                );

                Ok(Response::new(IdentityResponse {
                    spiffe_id,
                    service_name,
                    valid: true,
                    error_message: String::new(),
                }))
            }
            Err(err) => {
                error!(
                    error = %err,
                    correlation_id = %correlation_id,
                    "Service identity extraction failed"
                );
                
                Ok(Response::new(IdentityResponse {
                    spiffe_id: String::new(),
                    service_name: String::new(),
                    valid: false,
                    error_message: format!("{} [correlation_id: {}]", err, correlation_id),
                }))
            }
        }
    }
}
