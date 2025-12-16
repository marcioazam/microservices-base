//! Token Service Contract Tests
//!
//! Consumer contracts for token-service provider.

use pact_consumer::prelude::*;
use pact_consumer::mock_server::StartMockServerAsync;
use serde_json::json;

/// Contract: IssueTokenPair for valid user
#[tokio::test]
async fn contract_issue_token_pair() {
    let pact = PactBuilder::new("auth-edge-service", "token-service")
        .interaction("issue token pair for valid user", "", |mut i| async move {
            i.given("user exists with id user-123");
            i.request
                .method("POST")
                .path("/auth.token.v1.TokenService/IssueTokenPair")
                .header("Content-Type", "application/grpc")
                .header("grpc-accept-encoding", "identity")
                .body_matching(
                    "application/grpc",
                    json!({
                        "user_id": "user-123",
                        "client_id": "client-abc",
                        "scopes": ["read", "write"],
                        "device_id": "device-xyz"
                    }),
                    None,
                );
            i.response
                .status(200)
                .header("Content-Type", "application/grpc")
                .body_matching(
                    "application/grpc",
                    json!({
                        "access_token": "eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9...",
                        "refresh_token": "rt_abc123...",
                        "token_type": "Bearer",
                        "expires_in": 900,
                        "scope": "read write"
                    }),
                    None,
                );
            i
        })
        .await
        .build();

    let mock_server = pact.start_mock_server_async(None).await;
    assert!(mock_server.url().starts_with("http://"));
    
    pact.write_pact(Some("./target/pacts"), false)
        .expect("Failed to write pact file");
}

/// Contract: RefreshTokens with valid refresh token
#[tokio::test]
async fn contract_refresh_tokens() {
    let pact = PactBuilder::new("auth-edge-service", "token-service")
        .interaction("refresh tokens with valid refresh token", "", |mut i| async move {
            i.given("valid refresh token exists");
            i.request
                .method("POST")
                .path("/auth.token.v1.TokenService/RefreshTokens")
                .header("Content-Type", "application/grpc")
                .body_matching(
                    "application/grpc",
                    json!({
                        "refresh_token": "rt_valid_token",
                        "client_id": "client-abc"
                    }),
                    None,
                );
            i.response
                .status(200)
                .header("Content-Type", "application/grpc")
                .body_matching(
                    "application/grpc",
                    json!({
                        "access_token": "eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9...",
                        "refresh_token": "rt_new_token...",
                        "token_type": "Bearer",
                        "expires_in": 900
                    }),
                    None,
                );
            i
        })
        .await
        .build();

    let mock_server = pact.start_mock_server_async(None).await;
    assert!(mock_server.url().starts_with("http://"));
    
    pact.write_pact(Some("./target/pacts"), false)
        .expect("Failed to write pact file");
}

/// Contract: GetJWKS for token validation
#[tokio::test]
async fn contract_get_jwks() {
    let pact = PactBuilder::new("auth-edge-service", "token-service")
        .interaction("get JWKS for token validation", "", |mut i| async move {
            i.given("signing keys are configured");
            i.request
                .method("POST")
                .path("/auth.token.v1.TokenService/GetJWKS")
                .header("Content-Type", "application/grpc");
            i.response
                .status(200)
                .header("Content-Type", "application/grpc")
                .body_matching(
                    "application/grpc",
                    json!({
                        "keys": [{
                            "kty": "EC",
                            "crv": "P-256",
                            "kid": "key-2025-01",
                            "use": "sig",
                            "alg": "ES256",
                            "x": "...",
                            "y": "..."
                        }]
                    }),
                    None,
                );
            i
        })
        .await
        .build();

    let mock_server = pact.start_mock_server_async(None).await;
    assert!(mock_server.url().starts_with("http://"));
    
    pact.write_pact(Some("./target/pacts"), false)
        .expect("Failed to write pact file");
}
