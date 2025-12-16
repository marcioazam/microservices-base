# Pact Contract Testing

## Overview

Pact provides consumer-driven contract testing for Auth Platform gRPC services:
- Consumer tests generate contracts from proto definitions
- Provider verification ensures backward compatibility
- CI/CD integration blocks breaking changes
- Pact Broker stores and versions contracts

## Contract Writing Guide

### Consumer Test Structure (Rust)

```rust
use pact_consumer::prelude::*;

#[tokio::test]
async fn test_token_service_contract() {
    let pact = PactBuilder::new("auth-edge-service", "token-service")
        .interaction("issue token pair", "", |mut i| async move {
            i.given("user exists");
            i.request
                .method("POST")
                .path("/auth.token.v1.TokenService/IssueTokenPair")
                .header("Content-Type", "application/grpc");
            i.response
                .status(200)
                .header("Content-Type", "application/grpc");
            i
        })
        .await
        .build();

    let mock_server = pact.start_mock_server_async(None).await;
    // Test consumer against mock server
    
    pact.write_pact(Some("./target/pacts"), false).unwrap();
}
```

### Provider Verification (Rust)

```rust
use pact_verifier::*;

#[tokio::test]
async fn verify_contracts() {
    let provider = ProviderInfo {
        name: "token-service".to_string(),
        host: "localhost".to_string(),
        port: Some(8081),
        ..Default::default()
    };

    let pact_source = PactSource::BrokerWithDynamicConfiguration {
        provider_name: "token-service".to_string(),
        broker_url: "http://pact-broker:9292".to_string(),
        enable_pending: true,
        ..Default::default()
    };

    verify_provider_async(provider, vec![pact_source], options).await.unwrap();
}
```

## CI/CD Configuration

### GitHub Actions Workflow

```yaml
# Consumer tests
- name: Run consumer tests
  run: cargo test --test pact_consumer_tests

- name: Publish contracts
  run: |
    pact-broker publish ./target/pacts \
      --broker-base-url $PACT_BROKER_URL \
      --consumer-app-version $GITHUB_SHA \
      --branch $GITHUB_REF_NAME

# Can-I-Deploy check
- name: Can I Deploy?
  run: |
    pact-broker can-i-deploy \
      --pacticipant auth-edge-service \
      --version $GITHUB_SHA \
      --to-environment production
```

## Pact Broker Usage

### Publishing Contracts

```bash
pact-broker publish ./target/pacts \
  --broker-base-url https://pact-broker.auth-platform.local \
  --consumer-app-version $(git rev-parse HEAD) \
  --branch $(git branch --show-current) \
  --tag $(git branch --show-current)
```

### Can-I-Deploy Check

```bash
pact-broker can-i-deploy \
  --pacticipant auth-edge-service \
  --version $(git rev-parse HEAD) \
  --to-environment production
```

### Recording Deployment

```bash
pact-broker record-deployment \
  --pacticipant auth-edge-service \
  --version $(git rev-parse HEAD) \
  --environment production
```

## Service Contracts

### auth-edge-service → token-service
- IssueTokenPair
- RefreshTokens
- GetJWKS

### auth-edge-service → session-identity-core
- CreateSession
- GetSession

### auth-edge-service → iam-policy-service
- Authorize
- CheckPermission

## Troubleshooting

### Contract Verification Failed

```bash
# View verification details
pact-broker describe-version \
  --pacticipant token-service \
  --version $(git rev-parse HEAD)

# Check specific interaction
pact-broker list-latest-pact-versions \
  --broker-base-url $PACT_BROKER_URL
```

### Can-I-Deploy Returns False

```bash
# View deployment matrix
pact-broker matrix \
  --pacticipant auth-edge-service \
  --version $(git rev-parse HEAD) \
  --to-environment production
```

### Webhook Not Triggering

```bash
# Check webhook configuration
curl -u admin:password $PACT_BROKER_URL/webhooks

# Test webhook manually
curl -X POST $PACT_BROKER_URL/webhooks/<id>/execute
```
