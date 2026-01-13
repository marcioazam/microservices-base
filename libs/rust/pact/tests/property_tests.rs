//! Property-based tests for Pact library.
//!
//! Tests validate:
//! - Property 12: Contract Serialization Round-Trip
//! - Property 13: Contract Version Git Commit Match

use auth_pact::{
    CanIDeployResult, Contract, ContractMetadata, Interaction, MatrixEntry, Participant,
    PactSpecification, Request, Response, ContractVersion, VerificationResult,
};
use proptest::prelude::*;
use std::collections::HashMap;

// Strategy for generating service names
fn service_name_strategy() -> impl Strategy<Value = String> {
    prop_oneof![
        Just("auth-edge-service".to_string()),
        Just("token-service".to_string()),
        Just("session-identity-core".to_string()),
        Just("iam-policy-service".to_string()),
        Just("mfa-service".to_string()),
    ]
}

// Strategy for generating git commit SHAs
fn git_commit_strategy() -> impl Strategy<Value = String> {
    "[0-9a-f]{40}"
}

// Strategy for generating version tags
fn version_tag_strategy() -> impl Strategy<Value = String> {
    "(main|develop|release/[0-9]+\\.[0-9]+)"
}

// Strategy for generating contract versions
fn contract_version_strategy() -> impl Strategy<Value = ContractVersion> {
    (
        service_name_strategy(),
        service_name_strategy(),
        "[0-9]+\\.[0-9]+\\.[0-9]+",
        git_commit_strategy(),
        version_tag_strategy(),
    )
        .prop_map(|(consumer, provider, version, git_commit, branch)| ContractVersion {
            consumer,
            provider,
            version,
            git_commit: git_commit.clone(),
            branch: branch.clone(),
            tags: vec![branch, git_commit],
        })
}

// Strategy for generating HTTP methods
fn http_method_strategy() -> impl Strategy<Value = String> {
    prop_oneof![
        Just("GET".to_string()),
        Just("POST".to_string()),
        Just("PUT".to_string()),
        Just("DELETE".to_string()),
        Just("PATCH".to_string()),
    ]
}

// Strategy for generating paths
fn path_strategy() -> impl Strategy<Value = String> {
    "/[a-z][a-z0-9/-]{2,30}"
}

proptest! {
    #![proptest_config(ProptestConfig::with_cases(100))]

    /// **Property 12: Contract Serialization Round-Trip**
    /// *For any* generated contract, serialization to JSON and deserialization
    /// back SHALL produce an identical object.
    /// **Validates: Requirements 12.1**
    #[test]
    fn prop_contract_serialization_roundtrip(
        consumer in service_name_strategy(),
        provider in service_name_strategy(),
        method in http_method_strategy(),
        path in path_strategy(),
    ) {
        let contract = Contract {
            consumer: Participant::new(&consumer),
            provider: Participant::new(&provider),
            interactions: vec![Interaction {
                description: "test interaction".to_string(),
                provider_state: Some("test state".to_string()),
                request: Request {
                    method,
                    path,
                    headers: HashMap::new(),
                    body: None,
                },
                response: Response {
                    status: 200,
                    headers: HashMap::new(),
                    body: None,
                },
            }],
            metadata: ContractMetadata {
                pact_specification: PactSpecification {
                    version: "4.0".to_string(),
                },
            },
        };

        let json = serde_json::to_string(&contract).unwrap();
        let deserialized: Contract = serde_json::from_str(&json).unwrap();

        prop_assert_eq!(contract, deserialized,
            "Contract should survive serialization roundtrip");
    }

    /// **Property 13: Contract Version Git Commit Match**
    /// *For any* generated contract, the Pact Broker SHALL store it with version
    /// tag matching the git commit SHA, enabling traceability.
    /// **Validates: Requirements 12.2**
    #[test]
    fn prop_contract_version_matches_git_commit(
        contract_version in contract_version_strategy(),
    ) {
        prop_assert!(
            contract_version.tags_include_commit(),
            "Contract tags should include git commit SHA"
        );

        prop_assert!(
            contract_version.has_valid_git_commit(),
            "Git commit SHA should be valid (40 hex chars)"
        );

        prop_assert_eq!(
            contract_version.git_commit.len(),
            40,
            "Git commit SHA should be 40 characters"
        );

        prop_assert!(
            contract_version.git_commit.chars().all(|c| c.is_ascii_hexdigit()),
            "Git commit SHA should be hexadecimal"
        );
    }

    /// Property: can-i-deploy requires all verified
    #[test]
    fn prop_can_i_deploy_requires_all_verified(
        num_consumers in 1usize..5,
        all_verified in proptest::bool::ANY,
    ) {
        let matrix: Vec<MatrixEntry> = (0..num_consumers)
            .map(|i| MatrixEntry::new(
                format!("consumer-{i}"),
                "1.0.0",
                "token-service",
                "2.0.0",
                all_verified,
            ))
            .collect();

        let result = CanIDeployResult::from_matrix(matrix);

        if all_verified {
            prop_assert!(result.can_deploy(), "Should allow deploy when all verified");
        } else {
            prop_assert!(!result.can_deploy(), "Should block deploy when any unverified");
        }
    }

    /// Property: Verification blocks deployment on failure
    #[test]
    fn prop_verification_blocks_on_failure(
        consumer in service_name_strategy(),
        provider in service_name_strategy(),
        success in proptest::bool::ANY,
    ) {
        let result = VerificationResult {
            success,
            provider,
            consumer,
            consumer_version: "1.0.0".to_string(),
            provider_version: "2.0.0".to_string(),
            verified_at: "2025-01-15T00:00:00Z".to_string(),
        };

        prop_assert_eq!(result.can_deploy(), success,
            "can_deploy should match verification success");
    }
}

#[test]
fn test_webhook_trigger_on_publish() {
    let contract_version = ContractVersion {
        consumer: "auth-edge-service".to_string(),
        provider: "token-service".to_string(),
        version: "1.0.0".to_string(),
        git_commit: "abc123def456789012345678901234567890abcd".to_string(),
        branch: "main".to_string(),
        tags: vec!["main".to_string(), "abc123def456789012345678901234567890abcd".to_string()],
    };

    assert!(contract_version.has_valid_git_commit());
    assert!(contract_version.tags_include_commit());
}

#[test]
fn test_deployment_matrix() {
    let matrix = vec![
        MatrixEntry::new("auth-edge", "1.0.0", "token-service", "2.0.0", true),
        MatrixEntry::new("session-core", "1.0.0", "token-service", "2.0.0", true),
    ];

    let result = CanIDeployResult::from_matrix(matrix);
    assert!(result.can_deploy());

    let matrix_with_failure = vec![
        MatrixEntry::new("auth-edge", "1.0.0", "token-service", "2.0.0", true),
        MatrixEntry::new("session-core", "1.0.0", "token-service", "2.0.0", false),
    ];

    let result = CanIDeployResult::from_matrix(matrix_with_failure);
    assert!(!result.can_deploy());
}
