//! IAM Service Contract Tests
//!
//! Consumer contracts for iam-policy-service provider.

use pact_consumer::prelude::*;
use pact_consumer::mock_server::StartMockServerAsync;
use serde_json::json;

/// Contract: Authorize user action
#[tokio::test]
async fn contract_authorize() {
    let pact = PactBuilder::new("auth-edge-service", "iam-policy-service")
        .interaction("authorize user action", "", |mut i| async move {
            i.given("user has admin role");
            i.request
                .method("POST")
                .path("/auth.iam.v1.IAMService/Authorize")
                .header("Content-Type", "application/grpc")
                .body_matching(
                    "application/grpc",
                    json!({
                        "subject": "user-123",
                        "resource": "documents/doc-456",
                        "action": "read",
                        "context": {}
                    }),
                    None,
                );
            i.response
                .status(200)
                .header("Content-Type", "application/grpc")
                .body_matching(
                    "application/grpc",
                    json!({
                        "allowed": true,
                        "reason": "RBAC: user has admin role"
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

/// Contract: CheckPermission for user
#[tokio::test]
async fn contract_check_permission() {
    let pact = PactBuilder::new("auth-edge-service", "iam-policy-service")
        .interaction("check user permission", "", |mut i| async move {
            i.given("permission policy exists");
            i.request
                .method("POST")
                .path("/auth.iam.v1.IAMService/CheckPermission")
                .header("Content-Type", "application/grpc")
                .body_matching(
                    "application/grpc",
                    json!({
                        "user_id": "user-123",
                        "permission": "documents:read"
                    }),
                    None,
                );
            i.response
                .status(200)
                .header("Content-Type", "application/grpc")
                .body_matching(
                    "application/grpc",
                    json!({ "has_permission": true }),
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
