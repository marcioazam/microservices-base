//! Session Service Contract Tests
//!
//! Consumer contracts for session-identity-core provider.

use pact_consumer::prelude::*;
use pact_consumer::mock_server::StartMockServerAsync;
use serde_json::json;

/// Contract: CreateSession for authenticated user
#[tokio::test]
async fn contract_create_session() {
    let pact = PactBuilder::new("auth-edge-service", "session-identity-core")
        .interaction("create session for authenticated user", "", |mut i| async move {
            i.given("user is authenticated");
            i.request
                .method("POST")
                .path("/auth.session.v1.SessionService/CreateSession")
                .header("Content-Type", "application/grpc")
                .body_matching(
                    "application/grpc",
                    json!({
                        "user_id": "user-123",
                        "device_info": {
                            "device_id": "device-xyz",
                            "user_agent": "Mozilla/5.0...",
                            "ip_address": "192.168.1.1"
                        },
                        "metadata": {}
                    }),
                    None,
                );
            i.response
                .status(200)
                .header("Content-Type", "application/grpc")
                .body_matching(
                    "application/grpc",
                    json!({
                        "session_id": "sess_abc123",
                        "created_at": "2025-01-15T00:00:00Z",
                        "expires_at": "2025-01-16T00:00:00Z"
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

/// Contract: GetSession for existing session
#[tokio::test]
async fn contract_get_session() {
    let pact = PactBuilder::new("auth-edge-service", "session-identity-core")
        .interaction("get existing session", "", |mut i| async move {
            i.given("session exists with id sess_abc123");
            i.request
                .method("POST")
                .path("/auth.session.v1.SessionService/GetSession")
                .header("Content-Type", "application/grpc")
                .body_matching(
                    "application/grpc",
                    json!({ "session_id": "sess_abc123" }),
                    None,
                );
            i.response
                .status(200)
                .header("Content-Type", "application/grpc")
                .body_matching(
                    "application/grpc",
                    json!({
                        "session": {
                            "session_id": "sess_abc123",
                            "user_id": "user-123",
                            "created_at": "2025-01-15T00:00:00Z",
                            "expires_at": "2025-01-16T00:00:00Z",
                            "is_active": true
                        }
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
