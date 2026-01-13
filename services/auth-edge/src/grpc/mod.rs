//! gRPC Service Implementation
//!
//! Implements the AuthEdgeService with type-state JWT validation,
//! Tower middleware stack, and proper error handling with correlation IDs.

use crate::config::Config;
use crate::error::{AuthEdgeError, ErrorResponse, ErrorCode as AuthErrorCode};
use crate::jwt::{JwkCache, JwtValidator};
use crate::mtls::SpiffeValidator;
use crate::observability::AuthEdgeLogger;
use crate::proto::auth::v1::auth_edge_service_server::AuthEdgeService;
use crate::proto::auth::v1::*;
use prost_types::Struct as ProtoStruct;
use prost_types::value::Kind;
use prost_types::Value as ProtoValue;
use rust_common::{CircuitBreaker, CircuitBreakerConfig};
use std::collections::HashMap;
use std::sync::Arc;
use std::time::Duration;
use tonic::{Request, Response, Status};
use tracing::{error, info, instrument};
use uuid::Uuid;

/// Auth Edge Service implementation with modern patterns.
pub struct AuthEdgeServiceImpl {
    config: Config,
    jwt_validator: JwtValidator,
    token_service_cb: Arc<CircuitBreaker>,
    iam_service_cb: Arc<CircuitBreaker>,
    spiffe_validator: SpiffeValidator,
    logger: Arc<AuthEdgeLogger>,
}

impl AuthEdgeServiceImpl {
    /// Creates a new AuthEdgeServiceImpl with all dependencies.
    pub async fn new(config: Config) -> Result<Self, AuthEdgeError> {
        let jwk_cache = Arc::new(
            JwkCache::new(&config).await?
        );

        let jwt_validator = JwtValidator::new(jwk_cache);

        let cb_config = CircuitBreakerConfig::default()
            .with_failure_threshold(config.circuit_breaker_failure_threshold)
            .with_timeout(Duration::from_secs(config.circuit_breaker_timeout_seconds));

        let token_service_cb = Arc::new(CircuitBreaker::new(cb_config.clone()));
        let iam_service_cb = Arc::new(CircuitBreaker::new(cb_config));

        let spiffe_validator = SpiffeValidator::new(config.allowed_spiffe_domains.clone());
        let logger = Arc::new(AuthEdgeLogger::new(&config).await?);

        Ok(Self {
            config,
            jwt_validator,
            token_service_cb,
            iam_service_cb,
            spiffe_validator,
            logger,
        })
    }

    /// Generates a new correlation ID for request tracing.
    fn generate_correlation_id() -> Uuid {
        Uuid::new_v4()
    }

    /// Converts ErrorCode to proto TokenErrorCode
    fn error_code_to_proto(code: AuthErrorCode) -> i32 {
        match code {
            AuthErrorCode::TokenMissing => 6,       // MISSING_CLAIMS
            AuthErrorCode::TokenInvalid => 3,       // INVALID_SIGNATURE
            AuthErrorCode::TokenExpired => 1,       // EXPIRED
            AuthErrorCode::TokenMalformed => 9,     // MALFORMED
            AuthErrorCode::ClaimsInvalid => 6,      // MISSING_CLAIMS
            AuthErrorCode::SpiffeError => 4,        // INVALID_ISSUER
            AuthErrorCode::CertificateError => 3,   // INVALID_SIGNATURE
            _ => 0,                                 // UNSPECIFIED
        }
    }

    /// Converts HashMap to proto Struct
    fn hashmap_to_proto_struct(map: HashMap<String, String>) -> Option<ProtoStruct> {
        let mut fields = HashMap::new();
        for (key, value) in map {
            fields.insert(key, ProtoValue {
                kind: Some(Kind::StringValue(value)),
            });
        }
        Some(ProtoStruct { fields })
    }

    /// Converts an AuthEdgeError to a ValidateTokenResponse with proper sanitization.
    fn error_to_response(err: &AuthEdgeError, correlation_id: Uuid) -> ValidateTokenResponse {
        let response = ErrorResponse::from_error(err, correlation_id);

        ValidateTokenResponse {
            valid: false,
            subject: String::new(),
            issuer: String::new(),
            audiences: vec![],
            scopes: vec![],
            expires_at: None,
            issued_at: None,
            not_before: None,
            jwt_id: String::new(),
            claims: None,
            error: Some(TokenValidationError {
                code: Self::error_code_to_proto(response.code),
                message: format!("{} [correlation_id: {}]", response.message, correlation_id),
                details: HashMap::new(),
            }),
            binding: None,
            acr: String::new(),
            amr: vec![],
            authorized_party: String::new(),
        }
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
            self.logger
                .log_validation_failure(&err, &correlation_id.to_string())
                .await;
            return Ok(Response::new(Self::error_to_response(&err, correlation_id)));
        }

        // Use type-state JWT validation
        let required_refs: Vec<&str> = req.required_claims.iter().map(|s| s.as_str()).collect();

        match self
            .jwt_validator
            .validate_token(&req.token, &required_refs)
            .await
        {
            Ok(validated_token) => {
                let claims = validated_token.claims();

                info!(
                    subject = %claims.sub,
                    correlation_id = %correlation_id,
                    "Token validated successfully"
                );
                self.logger
                    .log_validation_success(&claims.sub, &correlation_id.to_string())
                    .await;

                Ok(Response::new(ValidateTokenResponse {
                    valid: true,
                    subject: claims.sub.clone(),
                    issuer: claims.iss.clone(),
                    audiences: claims.aud.clone(),
                    scopes: claims.scopes.clone().unwrap_or_default(),
                    expires_at: None, // TODO: Convert from timestamp
                    issued_at: None,  // TODO: Convert from timestamp
                    not_before: None, // TODO: Convert from timestamp
                    jwt_id: claims.jti.clone(),
                    claims: Self::hashmap_to_proto_struct(claims.to_map()),
                    error: None,
                    binding: None,
                    acr: String::new(),
                    amr: vec![],
                    authorized_party: String::new(),
                }))
            }
            Err(err) => {
                error!(
                    error = %err,
                    correlation_id = %correlation_id,
                    error_type = ?err.code(),
                    "Token validation failed"
                );
                self.logger
                    .log_validation_failure(&err, &correlation_id.to_string())
                    .await;

                Ok(Response::new(Self::error_to_response(&err, correlation_id)))
            }
        }
    }

    #[instrument(skip(self, request))]
    async fn introspect_token(
        &self,
        request: Request<IntrospectTokenRequest>,
    ) -> Result<Response<IntrospectTokenResponse>, Status> {
        let correlation_id = Self::generate_correlation_id();
        let req = request.into_inner();

        // For introspection, we validate without required claims
        match self.jwt_validator.validate_token(&req.token, &[]).await {
            Ok(validated_token) => {
                let claims = validated_token.claims();

                Ok(Response::new(IntrospectTokenResponse {
                    active: !claims.is_expired(),
                    sub: Some(claims.sub.clone()),
                    client_id: claims
                        .custom
                        .get("client_id")
                        .and_then(|v| v.as_str())
                        .map(|s| s.to_string()),
                    scope: claims.scopes.as_ref().map(|scopes| scopes.join(" ")),
                    exp: Some(claims.exp as i64),
                    iat: Some(claims.iat as i64),
                    token_type: Some("Bearer".to_string()),
                    ..Default::default()
                }))
            }
            Err(_err) => {
                info!(
                    correlation_id = %correlation_id,
                    "Token introspection: token inactive"
                );

                Ok(Response::new(IntrospectTokenResponse {
                    active: false,
                    ..Default::default()
                }))
            }
        }
    }

    #[instrument(skip(self, request))]
    async fn get_service_identity(
        &self,
        request: Request<GetServiceIdentityRequest>,
    ) -> Result<Response<GetServiceIdentityResponse>, Status> {
        let correlation_id = Self::generate_correlation_id();
        let req = request.into_inner();

        match self
            .spiffe_validator
            .extract_from_certificate(&req.certificate_pem)
        {
            Ok(spiffe_id) => {
                let service_name =
                    SpiffeValidator::extract_service_name(&spiffe_id).unwrap_or_default();

                info!(
                    spiffe_id = %spiffe_id.to_uri(),
                    service_name = %service_name,
                    correlation_id = %correlation_id,
                    "Service identity extracted"
                );

                Ok(Response::new(GetServiceIdentityResponse {
                    valid: true,
                    spiffe_id: spiffe_id.to_uri(),
                    service_name,
                    error_message: String::new(),
                    ..Default::default()
                }))
            }
            Err(err) => {
                error!(
                    error = %err,
                    correlation_id = %correlation_id,
                    "Service identity extraction failed"
                );

                Ok(Response::new(GetServiceIdentityResponse {
                    valid: false,
                    spiffe_id: String::new(),
                    service_name: String::new(),
                    error_message: format!("{err} [correlation_id: {correlation_id}]"),
                    ..Default::default()
                }))
            }
        }
    }

    #[instrument(skip(self, request))]
    async fn validate_d_po_p(
        &self,
        request: Request<ValidateDPoPRequest>,
    ) -> Result<Response<ValidateDPoPResponse>, Status> {
        let correlation_id = Self::generate_correlation_id();
        let _req = request.into_inner();

        // TODO: Implement DPoP validation logic
        info!(
            correlation_id = %correlation_id,
            "DPoP validation requested - not yet implemented"
        );

        Ok(Response::new(ValidateDPoPResponse {
            valid: false,
            jwk_thumbprint: String::new(),
            jti: String::new(),
            error: Some(TokenValidationError {
                code: 0, // UNSPECIFIED
                message: "DPoP validation not yet implemented".to_string(),
                details: HashMap::new(),
            }),
            nonce: None,
        }))
    }

    #[instrument(skip(self, request))]
    async fn check_revocation(
        &self,
        request: Request<CheckRevocationRequest>,
    ) -> Result<Response<CheckRevocationResponse>, Status> {
        let correlation_id = Self::generate_correlation_id();
        let _req = request.into_inner();

        // TODO: Implement revocation checking logic
        info!(
            correlation_id = %correlation_id,
            "Revocation check requested - not yet implemented"
        );

        Ok(Response::new(CheckRevocationResponse {
            revoked: false,
            revoked_at: None,
            reason: String::new(),
        }))
    }
}
