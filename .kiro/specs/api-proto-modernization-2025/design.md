# Design Document: API Proto Modernization 2025

## Overview

This design document outlines the modernization of the `/api` directory to state-of-the-art standards as of December 2025. The modernization transforms the existing Protocol Buffer definitions into a production-ready, multi-protocol API platform supporting gRPC, gRPC-Web, Connect-RPC, and REST/HTTP with comprehensive validation, observability, and security features.

## Architecture

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              API Layer                                       │
├─────────────────────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐        │
│  │   gRPC      │  │  gRPC-Web   │  │ Connect-RPC │  │  REST/HTTP  │        │
│  │  (HTTP/2)   │  │  (HTTP/1.1) │  │  (HTTP/1+2) │  │  (OpenAPI)  │        │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘        │
│         │                │                │                │               │
│         └────────────────┴────────────────┴────────────────┘               │
│                                   │                                         │
│                    ┌──────────────▼──────────────┐                         │
│                    │     Protocol Buffers v3     │                         │
│                    │   (Buf Ecosystem Managed)   │                         │
│                    └──────────────┬──────────────┘                         │
│                                   │                                         │
│  ┌────────────────────────────────┼────────────────────────────────┐       │
│  │                                │                                │       │
│  ▼                                ▼                                ▼       │
│ ┌──────────────┐  ┌──────────────────────────┐  ┌──────────────────┐      │
│ │ Protovalidate│  │   OpenTelemetry Tracing  │  │ Security Metadata│      │
│ │ (CEL Rules)  │  │   (W3C Trace Context)    │  │ (Auth/Authz)     │      │
│ └──────────────┘  └──────────────────────────┘  └──────────────────┘      │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Directory Structure

```
api/
├── buf.yaml                    # Buf configuration
├── buf.gen.yaml                # Code generation config
├── buf.lock                    # Dependency lock file
├── Makefile                    # Build automation
├── README.md                   # API documentation
├── proto/
│   ├── auth/
│   │   └── v1/                 # Versioned auth services
│   │       ├── auth_edge.proto
│   │       ├── common.proto
│   │       ├── iam_policy.proto
│   │       ├── mfa_service.proto
│   │       ├── session_identity.proto
│   │       └── token_service.proto
│   └── infra/
│       └── resilience/
│           └── v1/
│               └── resilience.proto
├── openapi/                    # Generated OpenAPI specs
│   └── v1/
└── gen/                        # Generated code (gitignored)
    ├── go/
    ├── rust/
    ├── typescript/
    └── python/
```


## Components and Interfaces

### 1. Buf Configuration (buf.yaml)

```yaml
version: v2
name: buf.build/auth-platform/api
modules:
  - path: proto
deps:
  - buf.build/bufbuild/protovalidate:v0.13.0
  - buf.build/googleapis/googleapis
  - buf.build/grpc-ecosystem/grpc-gateway:v2.25.1
  - buf.build/envoyproxy/protoc-gen-validate
lint:
  use:
    - STANDARD
    - COMMENTS
  except:
    - PACKAGE_VERSION_SUFFIX
  disallow_comment_ignores: true
breaking:
  use:
    - FILE
    - PACKAGE
```

### 2. Code Generation (buf.gen.yaml)

```yaml
version: v2
inputs:
  - directory: proto
plugins:
  # Go generation
  - remote: buf.build/protocolbuffers/go:v1.36.5
    out: gen/go
    opt: paths=source_relative
  - remote: buf.build/grpc/go:v1.5.1
    out: gen/go
    opt: paths=source_relative
  - remote: buf.build/connectrpc/go:v1.17.0
    out: gen/go
    opt: paths=source_relative
  - remote: buf.build/grpc-ecosystem/gateway:v2.25.1
    out: gen/go
    opt: paths=source_relative
  
  # Rust generation
  - remote: buf.build/community/neoeinstein-prost:v0.4.0
    out: gen/rust
  - remote: buf.build/community/neoeinstein-tonic:v0.4.1
    out: gen/rust
  
  # TypeScript/Connect-ES generation
  - remote: buf.build/bufbuild/es:v2.2.3
    out: gen/typescript
  - remote: buf.build/connectrpc/es:v2.0.0
    out: gen/typescript
  
  # Python generation
  - remote: buf.build/protocolbuffers/python:v29.3
    out: gen/python
  - remote: buf.build/grpc/python:v1.70.0
    out: gen/python
  
  # OpenAPI generation
  - remote: buf.build/grpc-ecosystem/openapiv2:v2.25.1
    out: openapi/v1
    opt:
      - allow_merge=true
      - merge_file_name=api

managed:
  enabled: true
  override:
    - file_option: go_package_prefix
      value: github.com/auth-platform/api/gen/go
  disable:
    - file_option: go_package
      module: buf.build/bufbuild/protovalidate
```

### 3. Common Types (proto/auth/v1/common.proto)

```protobuf
syntax = "proto3";

package auth.v1;

option go_package = "github.com/auth-platform/api/gen/go/auth/v1;authv1";
option java_package = "com.authplatform.api.auth.v1";
option java_multiple_files = true;

import "buf/validate/validate.proto";
import "google/protobuf/timestamp.proto";
import "google/protobuf/duration.proto";
import "google/rpc/status.proto";
import "google/api/field_behavior.proto";

// RequestMetadata contains common request metadata for tracing and correlation.
message RequestMetadata {
  // Unique correlation ID for request tracing (UUID v7 format).
  string correlation_id = 1 [(buf.validate.field).string.uuid = true];
  
  // Client application identifier.
  string client_id = 2 [(buf.validate.field).string = {
    min_len: 1,
    max_len: 128
  }];
  
  // Request timestamp.
  google.protobuf.Timestamp timestamp = 3;
  
  // W3C Trace Context traceparent header value.
  string trace_parent = 4;
  
  // W3C Trace Context tracestate header value.
  string trace_state = 5;
}

// PaginationRequest defines pagination parameters.
message PaginationRequest {
  // Page size (1-100, default 20).
  int32 page_size = 1 [(buf.validate.field).int32 = {
    gte: 1,
    lte: 100
  }];
  
  // Opaque page token for cursor-based pagination.
  string page_token = 2 [(buf.validate.field).string.max_len = 512];
}

// PaginationResponse contains pagination metadata.
message PaginationResponse {
  // Token for the next page, empty if no more pages.
  string next_page_token = 1;
  
  // Total count of items (optional, may be expensive to compute).
  optional int32 total_count = 2;
}

// ErrorDetail provides structured error information.
message ErrorDetail {
  // Machine-readable error code.
  string code = 1 [(buf.validate.field).string = {
    min_len: 1,
    max_len: 64,
    pattern: "^[A-Z][A-Z0-9_]*$"
  }];
  
  // Human-readable error message.
  string message = 2 [(buf.validate.field).string.max_len = 1024];
  
  // Field path for validation errors (e.g., "user.email").
  string field = 3;
  
  // Additional error metadata.
  map<string, string> metadata = 4;
}
```


### 4. Auth Edge Service (proto/auth/v1/auth_edge.proto)

```protobuf
syntax = "proto3";

package auth.v1;

option go_package = "github.com/auth-platform/api/gen/go/auth/v1;authv1";

import "buf/validate/validate.proto";
import "google/api/annotations.proto";
import "google/api/field_behavior.proto";
import "google/protobuf/timestamp.proto";
import "google/protobuf/struct.proto";

// AuthEdgeService provides token validation and identity services at the edge.
service AuthEdgeService {
  // ValidateToken validates a JWT access token.
  rpc ValidateToken(ValidateTokenRequest) returns (ValidateTokenResponse) {
    option (google.api.http) = {
      post: "/v1/auth/validate"
      body: "*"
    };
  }
  
  // IntrospectToken performs RFC 7662 token introspection.
  rpc IntrospectToken(IntrospectTokenRequest) returns (IntrospectTokenResponse) {
    option (google.api.http) = {
      post: "/v1/auth/introspect"
      body: "*"
    };
  }
  
  // ValidateDPoP validates a DPoP proof JWT per RFC 9449.
  rpc ValidateDPoP(ValidateDPoPRequest) returns (ValidateDPoPResponse) {
    option (google.api.http) = {
      post: "/v1/auth/dpop/validate"
      body: "*"
    };
  }
  
  // GetServiceIdentity validates SPIFFE/mTLS identity.
  rpc GetServiceIdentity(GetServiceIdentityRequest) returns (GetServiceIdentityResponse) {
    option (google.api.http) = {
      post: "/v1/auth/identity/service"
      body: "*"
    };
  }
}

// ValidateTokenRequest contains the token to validate.
message ValidateTokenRequest {
  // The JWT access token to validate.
  string token = 1 [
    (buf.validate.field).required = true,
    (buf.validate.field).string.min_len = 10,
    (google.api.field_behavior) = REQUIRED
  ];
  
  // Required claims that must be present.
  repeated string required_claims = 2;
  
  // Required scopes (all must be present).
  repeated string required_scopes = 3;
  
  // Expected audience values (at least one must match).
  repeated string audiences = 4;
  
  // DPoP proof JWT for sender-constrained tokens.
  optional string dpop_proof = 5;
  
  // HTTP method for DPoP validation.
  optional string http_method = 6;
  
  // HTTP URI for DPoP validation.
  optional string http_uri = 7;
}

// ValidateTokenResponse contains validation results.
message ValidateTokenResponse {
  // Whether the token is valid.
  bool valid = 1;
  
  // Token subject (user ID or client ID).
  string subject = 2;
  
  // Token issuer.
  string issuer = 3;
  
  // Token audience.
  repeated string audiences = 4;
  
  // Token scopes.
  repeated string scopes = 5;
  
  // Token expiration time.
  google.protobuf.Timestamp expires_at = 6;
  
  // Token issued at time.
  google.protobuf.Timestamp issued_at = 7;
  
  // Custom claims as JSON.
  google.protobuf.Struct claims = 8;
  
  // Error details if validation failed.
  TokenValidationError error = 9;
  
  // Token binding confirmation (for DPoP/mTLS).
  TokenBinding binding = 10;
}

// TokenValidationError describes why token validation failed.
message TokenValidationError {
  TokenErrorCode code = 1;
  string message = 2;
}

// TokenErrorCode enumerates token validation error types.
enum TokenErrorCode {
  TOKEN_ERROR_CODE_UNSPECIFIED = 0;
  TOKEN_ERROR_CODE_EXPIRED = 1;
  TOKEN_ERROR_CODE_NOT_YET_VALID = 2;
  TOKEN_ERROR_CODE_INVALID_SIGNATURE = 3;
  TOKEN_ERROR_CODE_INVALID_ISSUER = 4;
  TOKEN_ERROR_CODE_INVALID_AUDIENCE = 5;
  TOKEN_ERROR_CODE_MISSING_CLAIMS = 6;
  TOKEN_ERROR_CODE_INSUFFICIENT_SCOPE = 7;
  TOKEN_ERROR_CODE_REVOKED = 8;
  TOKEN_ERROR_CODE_MALFORMED = 9;
  TOKEN_ERROR_CODE_DPOP_INVALID = 10;
  TOKEN_ERROR_CODE_BINDING_MISMATCH = 11;
}

// TokenBinding describes token sender constraints.
message TokenBinding {
  // Binding type (dpop, mtls).
  string type = 1;
  
  // JWK thumbprint for DPoP binding.
  string jwk_thumbprint = 2;
  
  // Certificate thumbprint for mTLS binding.
  string certificate_thumbprint = 3;
}

// IntrospectTokenRequest for RFC 7662 introspection.
message IntrospectTokenRequest {
  string token = 1 [(buf.validate.field).required = true];
  string token_type_hint = 2;
}

// IntrospectTokenResponse per RFC 7662.
message IntrospectTokenResponse {
  bool active = 1;
  optional string scope = 2;
  optional string client_id = 3;
  optional string username = 4;
  optional string token_type = 5;
  optional int64 exp = 6;
  optional int64 iat = 7;
  optional int64 nbf = 8;
  optional string sub = 9;
  optional string aud = 10;
  optional string iss = 11;
  optional string jti = 12;
  google.protobuf.Struct cnf = 13; // Confirmation claim for bound tokens
}

// ValidateDPoPRequest validates a DPoP proof.
message ValidateDPoPRequest {
  string dpop_proof = 1 [(buf.validate.field).required = true];
  string http_method = 2 [(buf.validate.field).required = true];
  string http_uri = 3 [(buf.validate.field).required = true];
  optional string access_token = 4;
}

// ValidateDPoPResponse contains DPoP validation results.
message ValidateDPoPResponse {
  bool valid = 1;
  string jwk_thumbprint = 2;
  string jti = 3;
  TokenValidationError error = 4;
}

// GetServiceIdentityRequest for SPIFFE/mTLS validation.
message GetServiceIdentityRequest {
  // PEM-encoded client certificate.
  string certificate_pem = 1 [(buf.validate.field).required = true];
  
  // Certificate chain (optional).
  repeated string certificate_chain = 2;
}

// GetServiceIdentityResponse contains service identity.
message GetServiceIdentityResponse {
  bool valid = 1;
  string spiffe_id = 2;
  string service_name = 3;
  string namespace = 4;
  string trust_domain = 5;
  google.protobuf.Timestamp not_before = 6;
  google.protobuf.Timestamp not_after = 7;
  string error_message = 8;
}
```


### 5. Token Service (proto/auth/v1/token_service.proto)

```protobuf
syntax = "proto3";

package auth.v1;

option go_package = "github.com/auth-platform/api/gen/go/auth/v1;authv1";

import "buf/validate/validate.proto";
import "google/api/annotations.proto";
import "google/api/field_behavior.proto";
import "google/protobuf/timestamp.proto";
import "google/protobuf/duration.proto";
import "google/protobuf/struct.proto";

// TokenService manages OAuth 2.1 token operations.
service TokenService {
  // IssueTokens issues access, refresh, and ID tokens.
  rpc IssueTokens(IssueTokensRequest) returns (TokenResponse) {
    option (google.api.http) = {
      post: "/v1/oauth/token"
      body: "*"
    };
  }
  
  // RefreshTokens exchanges a refresh token for new tokens.
  rpc RefreshTokens(RefreshTokensRequest) returns (TokenResponse) {
    option (google.api.http) = {
      post: "/v1/oauth/token/refresh"
      body: "*"
    };
  }
  
  // RevokeToken revokes an access or refresh token (RFC 7009).
  rpc RevokeToken(RevokeTokenRequest) returns (RevokeTokenResponse) {
    option (google.api.http) = {
      post: "/v1/oauth/revoke"
      body: "*"
    };
  }
  
  // ExchangeToken performs token exchange (RFC 8693).
  rpc ExchangeToken(ExchangeTokenRequest) returns (TokenResponse) {
    option (google.api.http) = {
      post: "/v1/oauth/token/exchange"
      body: "*"
    };
  }
  
  // GetJWKS returns the JSON Web Key Set.
  rpc GetJWKS(GetJWKSRequest) returns (JWKSResponse) {
    option (google.api.http) = {
      get: "/v1/.well-known/jwks.json"
    };
  }
  
  // RotateSigningKey rotates the token signing key.
  rpc RotateSigningKey(RotateSigningKeyRequest) returns (RotateSigningKeyResponse);
  
  // PushAuthorizationRequest handles PAR (RFC 9126).
  rpc PushAuthorizationRequest(PARRequest) returns (PARResponse) {
    option (google.api.http) = {
      post: "/v1/oauth/par"
      body: "*"
    };
  }
}

// GrantType enumerates OAuth 2.1 grant types.
enum GrantType {
  GRANT_TYPE_UNSPECIFIED = 0;
  GRANT_TYPE_AUTHORIZATION_CODE = 1;
  GRANT_TYPE_CLIENT_CREDENTIALS = 2;
  GRANT_TYPE_REFRESH_TOKEN = 3;
  GRANT_TYPE_DEVICE_CODE = 4;
  GRANT_TYPE_TOKEN_EXCHANGE = 5;
}

// IssueTokensRequest for token issuance.
message IssueTokensRequest {
  // Grant type.
  GrantType grant_type = 1 [(buf.validate.field).required = true];
  
  // User ID (for authorization_code grant).
  string user_id = 2 [(buf.validate.field).string.uuid = true];
  
  // Session ID to bind tokens to.
  string session_id = 3 [(buf.validate.field).string.uuid = true];
  
  // Client ID.
  string client_id = 4 [(buf.validate.field).required = true];
  
  // Requested scopes.
  repeated string scopes = 5;
  
  // Custom claims to include.
  google.protobuf.Struct custom_claims = 6;
  
  // Access token TTL override.
  google.protobuf.Duration access_token_ttl = 7;
  
  // Refresh token TTL override.
  google.protobuf.Duration refresh_token_ttl = 8;
  
  // Authorization code (for authorization_code grant).
  string code = 9;
  
  // PKCE code verifier.
  string code_verifier = 10;
  
  // Redirect URI for validation.
  string redirect_uri = 11;
  
  // DPoP JWK thumbprint for sender-constrained tokens.
  string dpop_jkt = 12;
  
  // Rich Authorization Request (RAR) details.
  repeated AuthorizationDetail authorization_details = 13;
}

// AuthorizationDetail for RAR (RFC 9396).
message AuthorizationDetail {
  string type = 1 [(buf.validate.field).required = true];
  repeated string locations = 2;
  repeated string actions = 3;
  repeated string datatypes = 4;
  string identifier = 5;
  google.protobuf.Struct privileges = 6;
}

// TokenResponse contains issued tokens.
message TokenResponse {
  string access_token = 1;
  string token_type = 2; // "Bearer" or "DPoP"
  int64 expires_in = 3;
  string refresh_token = 4;
  string id_token = 5;
  string scope = 6;
  repeated AuthorizationDetail authorization_details = 7;
}

// RefreshTokensRequest for token refresh.
message RefreshTokensRequest {
  string refresh_token = 1 [(buf.validate.field).required = true];
  string client_id = 2 [(buf.validate.field).required = true];
  string client_secret = 3;
  repeated string scopes = 4;
  string dpop_jkt = 5;
}

// RevokeTokenRequest for token revocation (RFC 7009).
message RevokeTokenRequest {
  string token = 1 [(buf.validate.field).required = true];
  string token_type_hint = 2; // "access_token" or "refresh_token"
  string client_id = 3 [(buf.validate.field).required = true];
  string client_secret = 4;
}

// RevokeTokenResponse confirms revocation.
message RevokeTokenResponse {
  bool success = 1;
}

// ExchangeTokenRequest for token exchange (RFC 8693).
message ExchangeTokenRequest {
  string grant_type = 1 [(buf.validate.field).string.const = "urn:ietf:params:oauth:grant-type:token-exchange"];
  string subject_token = 2 [(buf.validate.field).required = true];
  string subject_token_type = 3 [(buf.validate.field).required = true];
  string actor_token = 4;
  string actor_token_type = 5;
  string requested_token_type = 6;
  string audience = 7;
  string scope = 8;
  string resource = 9;
}

// GetJWKSRequest for JWKS retrieval.
message GetJWKSRequest {}

// JWKSResponse contains the JSON Web Key Set.
message JWKSResponse {
  repeated JWK keys = 1;
}

// JWK represents a JSON Web Key.
message JWK {
  string kty = 1;  // Key type (RSA, EC, OKP)
  string use = 2;  // Key use (sig)
  string kid = 3;  // Key ID
  string alg = 4;  // Algorithm (RS256, ES256, EdDSA)
  string n = 5;    // RSA modulus
  string e = 6;    // RSA exponent
  string crv = 7;  // EC curve (P-256, P-384, Ed25519)
  string x = 8;    // EC x coordinate
  string y = 9;    // EC y coordinate
}

// RotateSigningKeyRequest for key rotation.
message RotateSigningKeyRequest {
  string algorithm = 1; // RS256, ES256, EdDSA
  bool immediate = 2;   // Skip grace period
}

// RotateSigningKeyResponse confirms rotation.
message RotateSigningKeyResponse {
  string new_key_id = 1;
  string old_key_id = 2;
  google.protobuf.Timestamp old_key_expires_at = 3;
}

// PARRequest for Pushed Authorization Request (RFC 9126).
message PARRequest {
  string client_id = 1 [(buf.validate.field).required = true];
  string client_secret = 2;
  string response_type = 3 [(buf.validate.field).required = true];
  string redirect_uri = 4 [(buf.validate.field).required = true];
  string scope = 5;
  string state = 6;
  string code_challenge = 7 [(buf.validate.field).required = true];
  string code_challenge_method = 8 [(buf.validate.field).string.const = "S256"];
  string nonce = 9;
  repeated AuthorizationDetail authorization_details = 10;
  string dpop_jkt = 11;
}

// PARResponse contains the request_uri.
message PARResponse {
  string request_uri = 1;
  int64 expires_in = 2; // Max 600 seconds per FAPI 2.0
}
```


### 6. MFA Service (proto/auth/v1/mfa_service.proto)

```protobuf
syntax = "proto3";

package auth.v1;

option go_package = "github.com/auth-platform/api/gen/go/auth/v1;authv1";

import "buf/validate/validate.proto";
import "google/api/annotations.proto";
import "google/protobuf/timestamp.proto";

// MFAService provides multi-factor authentication operations.
service MFAService {
  // TOTP Operations
  rpc EnrollTOTP(EnrollTOTPRequest) returns (EnrollTOTPResponse) {
    option (google.api.http) = {
      post: "/v1/mfa/totp/enroll"
      body: "*"
    };
  }
  
  rpc VerifyTOTP(VerifyTOTPRequest) returns (VerifyMFAResponse) {
    option (google.api.http) = {
      post: "/v1/mfa/totp/verify"
      body: "*"
    };
  }
  
  // WebAuthn/Passkey Operations
  rpc BeginWebAuthnRegistration(BeginWebAuthnRegistrationRequest) returns (WebAuthnRegistrationOptions) {
    option (google.api.http) = {
      post: "/v1/mfa/webauthn/register/begin"
      body: "*"
    };
  }
  
  rpc CompleteWebAuthnRegistration(CompleteWebAuthnRegistrationRequest) returns (WebAuthnCredential) {
    option (google.api.http) = {
      post: "/v1/mfa/webauthn/register/complete"
      body: "*"
    };
  }
  
  rpc BeginWebAuthnAuthentication(BeginWebAuthnAuthenticationRequest) returns (WebAuthnAuthenticationOptions) {
    option (google.api.http) = {
      post: "/v1/mfa/webauthn/authenticate/begin"
      body: "*"
    };
  }
  
  rpc CompleteWebAuthnAuthentication(CompleteWebAuthnAuthenticationRequest) returns (VerifyMFAResponse) {
    option (google.api.http) = {
      post: "/v1/mfa/webauthn/authenticate/complete"
      body: "*"
    };
  }
  
  // Push Notification MFA
  rpc SendPushChallenge(SendPushChallengeRequest) returns (PushChallengeResponse) {
    option (google.api.http) = {
      post: "/v1/mfa/push/send"
      body: "*"
    };
  }
  
  rpc CheckPushApproval(CheckPushApprovalRequest) returns (VerifyMFAResponse) {
    option (google.api.http) = {
      get: "/v1/mfa/push/{challenge_id}/status"
    };
  }
  
  // Backup Codes
  rpc GenerateBackupCodes(GenerateBackupCodesRequest) returns (BackupCodesResponse) {
    option (google.api.http) = {
      post: "/v1/mfa/backup-codes/generate"
      body: "*"
    };
  }
  
  rpc VerifyBackupCode(VerifyBackupCodeRequest) returns (VerifyMFAResponse) {
    option (google.api.http) = {
      post: "/v1/mfa/backup-codes/verify"
      body: "*"
    };
  }
  
  // MFA Status
  rpc GetMFAStatus(GetMFAStatusRequest) returns (MFAStatusResponse) {
    option (google.api.http) = {
      get: "/v1/mfa/status/{user_id}"
    };
  }
  
  // CAEP Event Streaming
  rpc StreamCAEPEvents(StreamCAEPEventsRequest) returns (stream CAEPEvent);
}

// TOTP Messages
message EnrollTOTPRequest {
  string user_id = 1 [(buf.validate.field).string.uuid = true];
  string issuer = 2 [(buf.validate.field).string.max_len = 64];
  TOTPAlgorithm algorithm = 3;
  int32 digits = 4 [(buf.validate.field).int32 = { gte: 6, lte: 8 }];
  int32 period_seconds = 5 [(buf.validate.field).int32 = { gte: 30, lte: 60 }];
}

enum TOTPAlgorithm {
  TOTP_ALGORITHM_UNSPECIFIED = 0;
  TOTP_ALGORITHM_SHA1 = 1;
  TOTP_ALGORITHM_SHA256 = 2;
  TOTP_ALGORITHM_SHA512 = 3;
}

message EnrollTOTPResponse {
  string secret = 1;
  string provisioning_uri = 2;
  string qr_code_base64 = 3;
  repeated string backup_codes = 4;
}

message VerifyTOTPRequest {
  string user_id = 1 [(buf.validate.field).string.uuid = true];
  string code = 2 [(buf.validate.field).string = { min_len: 6, max_len: 8, pattern: "^[0-9]+$" }];
}

message VerifyMFAResponse {
  bool valid = 1;
  MFAErrorCode error_code = 2;
  string error_message = 3;
  google.protobuf.Timestamp verified_at = 4;
}

enum MFAErrorCode {
  MFA_ERROR_CODE_UNSPECIFIED = 0;
  MFA_ERROR_CODE_INVALID_CODE = 1;
  MFA_ERROR_CODE_EXPIRED = 2;
  MFA_ERROR_CODE_ALREADY_USED = 3;
  MFA_ERROR_CODE_NOT_ENROLLED = 4;
  MFA_ERROR_CODE_RATE_LIMITED = 5;
  MFA_ERROR_CODE_CREDENTIAL_NOT_FOUND = 6;
}

// WebAuthn Messages (FIDO2/Passkeys)
message BeginWebAuthnRegistrationRequest {
  string user_id = 1 [(buf.validate.field).string.uuid = true];
  string user_name = 2 [(buf.validate.field).string.max_len = 64];
  string display_name = 3 [(buf.validate.field).string.max_len = 64];
  AuthenticatorSelection authenticator_selection = 4;
  AttestationConveyance attestation = 5;
}

message AuthenticatorSelection {
  AuthenticatorAttachment authenticator_attachment = 1;
  ResidentKeyRequirement resident_key = 2;
  UserVerificationRequirement user_verification = 3;
}

enum AuthenticatorAttachment {
  AUTHENTICATOR_ATTACHMENT_UNSPECIFIED = 0;
  AUTHENTICATOR_ATTACHMENT_PLATFORM = 1;
  AUTHENTICATOR_ATTACHMENT_CROSS_PLATFORM = 2;
}

enum ResidentKeyRequirement {
  RESIDENT_KEY_REQUIREMENT_UNSPECIFIED = 0;
  RESIDENT_KEY_REQUIREMENT_DISCOURAGED = 1;
  RESIDENT_KEY_REQUIREMENT_PREFERRED = 2;
  RESIDENT_KEY_REQUIREMENT_REQUIRED = 3;
}

enum UserVerificationRequirement {
  USER_VERIFICATION_REQUIREMENT_UNSPECIFIED = 0;
  USER_VERIFICATION_REQUIREMENT_REQUIRED = 1;
  USER_VERIFICATION_REQUIREMENT_PREFERRED = 2;
  USER_VERIFICATION_REQUIREMENT_DISCOURAGED = 3;
}

enum AttestationConveyance {
  ATTESTATION_CONVEYANCE_UNSPECIFIED = 0;
  ATTESTATION_CONVEYANCE_NONE = 1;
  ATTESTATION_CONVEYANCE_INDIRECT = 2;
  ATTESTATION_CONVEYANCE_DIRECT = 3;
  ATTESTATION_CONVEYANCE_ENTERPRISE = 4;
}

message WebAuthnRegistrationOptions {
  bytes challenge = 1;
  RelyingParty rp = 2;
  WebAuthnUser user = 3;
  repeated PublicKeyCredentialParameters pub_key_cred_params = 4;
  int64 timeout_ms = 5;
  repeated PublicKeyCredentialDescriptor exclude_credentials = 6;
  AuthenticatorSelection authenticator_selection = 7;
  AttestationConveyance attestation = 8;
  // Passkey-specific: hints for hybrid transport
  repeated string hints = 9;
}

message RelyingParty {
  string id = 1;
  string name = 2;
}

message WebAuthnUser {
  bytes id = 1;
  string name = 2;
  string display_name = 3;
}

message PublicKeyCredentialParameters {
  string type = 1; // "public-key"
  int32 alg = 2;   // COSE algorithm identifier
}

message PublicKeyCredentialDescriptor {
  string type = 1;
  bytes id = 2;
  repeated string transports = 3; // usb, nfc, ble, internal, hybrid
}

message CompleteWebAuthnRegistrationRequest {
  string user_id = 1 [(buf.validate.field).string.uuid = true];
  bytes credential_id = 2;
  bytes attestation_object = 3;
  bytes client_data_json = 4;
  string device_name = 5;
  repeated string transports = 6;
}

message WebAuthnCredential {
  string credential_id = 1;
  string device_name = 2;
  google.protobuf.Timestamp created_at = 3;
  google.protobuf.Timestamp last_used_at = 4;
  repeated string transports = 5;
  bool backup_eligible = 6;
  bool backup_state = 7;
}

message BeginWebAuthnAuthenticationRequest {
  string user_id = 1;
  bool conditional_ui = 2; // For passkey autofill
}

message WebAuthnAuthenticationOptions {
  bytes challenge = 1;
  int64 timeout_ms = 2;
  string rp_id = 3;
  repeated PublicKeyCredentialDescriptor allow_credentials = 4;
  UserVerificationRequirement user_verification = 5;
}

message CompleteWebAuthnAuthenticationRequest {
  string user_id = 1;
  bytes credential_id = 2;
  bytes authenticator_data = 3;
  bytes client_data_json = 4;
  bytes signature = 5;
  bytes user_handle = 6;
}

// Push MFA Messages
message SendPushChallengeRequest {
  string user_id = 1 [(buf.validate.field).string.uuid = true];
  string device_id = 2;
  string title = 3;
  string body = 4;
  PushContext context = 5;
  int32 timeout_seconds = 6 [(buf.validate.field).int32 = { gte: 30, lte: 300 }];
}

message PushContext {
  string ip_address = 1;
  string location = 2;
  string device_info = 3;
  string action = 4;
}

message PushChallengeResponse {
  string challenge_id = 1;
  bool sent = 2;
  string error_message = 3;
}

message CheckPushApprovalRequest {
  string challenge_id = 1 [(buf.validate.field).string.uuid = true];
}

// Backup Codes Messages
message GenerateBackupCodesRequest {
  string user_id = 1 [(buf.validate.field).string.uuid = true];
  int32 count = 2 [(buf.validate.field).int32 = { gte: 8, lte: 16 }];
}

message BackupCodesResponse {
  repeated string codes = 1;
  google.protobuf.Timestamp generated_at = 2;
}

message VerifyBackupCodeRequest {
  string user_id = 1 [(buf.validate.field).string.uuid = true];
  string code = 2 [(buf.validate.field).string = { min_len: 8, max_len: 16 }];
}

// MFA Status Messages
message GetMFAStatusRequest {
  string user_id = 1 [(buf.validate.field).string.uuid = true];
}

message MFAStatusResponse {
  bool mfa_enabled = 1;
  repeated MFAMethod enrolled_methods = 2;
  MFAMethod preferred_method = 3;
  int32 backup_codes_remaining = 4;
}

enum MFAMethod {
  MFA_METHOD_UNSPECIFIED = 0;
  MFA_METHOD_TOTP = 1;
  MFA_METHOD_WEBAUTHN = 2;
  MFA_METHOD_PUSH = 3;
  MFA_METHOD_SMS = 4;
  MFA_METHOD_EMAIL = 5;
  MFA_METHOD_BACKUP_CODE = 6;
}

// CAEP (Continuous Access Evaluation Protocol) Messages
message StreamCAEPEventsRequest {
  string user_id = 1;
  repeated string event_types = 2;
}

message CAEPEvent {
  string event_id = 1;
  CAEPEventType event_type = 2;
  string subject = 3;
  google.protobuf.Timestamp timestamp = 4;
  map<string, string> claims = 5;
}

enum CAEPEventType {
  CAEP_EVENT_TYPE_UNSPECIFIED = 0;
  CAEP_EVENT_TYPE_SESSION_REVOKED = 1;
  CAEP_EVENT_TYPE_TOKEN_CLAIMS_CHANGE = 2;
  CAEP_EVENT_TYPE_CREDENTIAL_CHANGE = 3;
  CAEP_EVENT_TYPE_ASSURANCE_LEVEL_CHANGE = 4;
  CAEP_EVENT_TYPE_DEVICE_COMPLIANCE_CHANGE = 5;
}
```


### 7. Session Identity Service (proto/auth/v1/session_identity.proto)

```protobuf
syntax = "proto3";

package auth.v1;

option go_package = "github.com/auth-platform/api/gen/go/auth/v1;authv1";

import "buf/validate/validate.proto";
import "google/api/annotations.proto";
import "google/protobuf/timestamp.proto";

// SessionIdentityService manages user sessions and OAuth flows.
service SessionIdentityService {
  // Session Management
  rpc CreateSession(CreateSessionRequest) returns (Session) {
    option (google.api.http) = {
      post: "/v1/sessions"
      body: "*"
    };
  }
  
  rpc GetSession(GetSessionRequest) returns (Session) {
    option (google.api.http) = {
      get: "/v1/sessions/{session_id}"
    };
  }
  
  rpc ListUserSessions(ListUserSessionsRequest) returns (ListUserSessionsResponse) {
    option (google.api.http) = {
      get: "/v1/users/{user_id}/sessions"
    };
  }
  
  rpc TerminateSession(TerminateSessionRequest) returns (TerminateSessionResponse) {
    option (google.api.http) = {
      delete: "/v1/sessions/{session_id}"
    };
  }
  
  rpc TerminateAllUserSessions(TerminateAllUserSessionsRequest) returns (TerminateAllUserSessionsResponse) {
    option (google.api.http) = {
      delete: "/v1/users/{user_id}/sessions"
    };
  }
  
  // Risk Assessment
  rpc UpdateRiskScore(UpdateRiskScoreRequest) returns (RiskAssessment) {
    option (google.api.http) = {
      post: "/v1/sessions/{session_id}/risk"
      body: "*"
    };
  }
  
  rpc GetRiskAssessment(GetRiskAssessmentRequest) returns (RiskAssessment) {
    option (google.api.http) = {
      get: "/v1/sessions/{session_id}/risk"
    };
  }
  
  // OAuth Authorization
  rpc Authorize(AuthorizeRequest) returns (AuthorizeResponse) {
    option (google.api.http) = {
      get: "/v1/oauth/authorize"
    };
  }
  
  // Authentication
  rpc Authenticate(AuthenticateRequest) returns (AuthenticateResponse) {
    option (google.api.http) = {
      post: "/v1/auth/authenticate"
      body: "*"
    };
  }
  
  // Session Events Stream
  rpc StreamSessionEvents(StreamSessionEventsRequest) returns (stream SessionEvent);
}

// Session represents a user session.
message Session {
  string session_id = 1;
  string user_id = 2;
  string device_id = 3;
  DeviceInfo device_info = 4;
  string ip_address = 5;
  GeoLocation geo_location = 6;
  SessionStatus status = 7;
  double risk_score = 8;
  bool mfa_verified = 9;
  repeated string mfa_methods_used = 10;
  google.protobuf.Timestamp created_at = 11;
  google.protobuf.Timestamp last_activity_at = 12;
  google.protobuf.Timestamp expires_at = 13;
  map<string, string> metadata = 14;
}

enum SessionStatus {
  SESSION_STATUS_UNSPECIFIED = 0;
  SESSION_STATUS_ACTIVE = 1;
  SESSION_STATUS_IDLE = 2;
  SESSION_STATUS_STEP_UP_REQUIRED = 3;
  SESSION_STATUS_TERMINATED = 4;
  SESSION_STATUS_EXPIRED = 5;
}

message DeviceInfo {
  string device_fingerprint = 1;
  string device_type = 2;
  string os = 3;
  string os_version = 4;
  string browser = 5;
  string browser_version = 6;
  bool is_mobile = 7;
  bool is_trusted = 8;
}

message GeoLocation {
  string country = 1;
  string region = 2;
  string city = 3;
  double latitude = 4;
  double longitude = 5;
  string timezone = 6;
}

// Session Management Messages
message CreateSessionRequest {
  string user_id = 1 [(buf.validate.field).string.uuid = true];
  DeviceInfo device_info = 2;
  string ip_address = 3 [(buf.validate.field).string.ip = true];
  map<string, string> metadata = 4;
  int32 max_idle_seconds = 5;
  int32 max_lifetime_seconds = 6;
}

message GetSessionRequest {
  string session_id = 1 [(buf.validate.field).string.uuid = true];
}

message ListUserSessionsRequest {
  string user_id = 1 [(buf.validate.field).string.uuid = true];
  PaginationRequest pagination = 2;
  bool active_only = 3;
}

message ListUserSessionsResponse {
  repeated Session sessions = 1;
  PaginationResponse pagination = 2;
}

message TerminateSessionRequest {
  string session_id = 1 [(buf.validate.field).string.uuid = true];
  string reason = 2;
}

message TerminateSessionResponse {
  bool success = 1;
}

message TerminateAllUserSessionsRequest {
  string user_id = 1 [(buf.validate.field).string.uuid = true];
  string except_session_id = 2; // Keep current session
  string reason = 3;
}

message TerminateAllUserSessionsResponse {
  int32 terminated_count = 1;
}

// Risk Assessment Messages
message UpdateRiskScoreRequest {
  string session_id = 1 [(buf.validate.field).string.uuid = true];
  repeated RiskFactor risk_factors = 2;
}

message RiskFactor {
  RiskFactorType type = 1;
  double score = 2 [(buf.validate.field).double = { gte: 0.0, lte: 1.0 }];
  string description = 3;
  map<string, string> details = 4;
}

enum RiskFactorType {
  RISK_FACTOR_TYPE_UNSPECIFIED = 0;
  RISK_FACTOR_TYPE_NEW_DEVICE = 1;
  RISK_FACTOR_TYPE_NEW_LOCATION = 2;
  RISK_FACTOR_TYPE_IMPOSSIBLE_TRAVEL = 3;
  RISK_FACTOR_TYPE_TOR_EXIT_NODE = 4;
  RISK_FACTOR_TYPE_VPN_DETECTED = 5;
  RISK_FACTOR_TYPE_BOT_DETECTED = 6;
  RISK_FACTOR_TYPE_CREDENTIAL_STUFFING = 7;
  RISK_FACTOR_TYPE_BRUTE_FORCE = 8;
  RISK_FACTOR_TYPE_COMPROMISED_CREDENTIALS = 9;
  RISK_FACTOR_TYPE_SUSPICIOUS_BEHAVIOR = 10;
}

message RiskAssessment {
  string session_id = 1;
  double overall_score = 2;
  RiskLevel risk_level = 3;
  repeated RiskFactor factors = 4;
  bool step_up_required = 5;
  repeated string required_mfa_methods = 6;
  google.protobuf.Timestamp assessed_at = 7;
}

enum RiskLevel {
  RISK_LEVEL_UNSPECIFIED = 0;
  RISK_LEVEL_LOW = 1;
  RISK_LEVEL_MEDIUM = 2;
  RISK_LEVEL_HIGH = 3;
  RISK_LEVEL_CRITICAL = 4;
}

message GetRiskAssessmentRequest {
  string session_id = 1 [(buf.validate.field).string.uuid = true];
}

// OAuth Authorization Messages
message AuthorizeRequest {
  string client_id = 1 [(buf.validate.field).required = true];
  string redirect_uri = 2 [(buf.validate.field).string.uri = true];
  string response_type = 3 [(buf.validate.field).required = true];
  string scope = 4;
  string state = 5 [(buf.validate.field).required = true];
  string code_challenge = 6 [(buf.validate.field).required = true];
  string code_challenge_method = 7 [(buf.validate.field).string.const = "S256"];
  string nonce = 8;
  string request_uri = 9; // For PAR
  string prompt = 10;
  string login_hint = 11;
  string acr_values = 12;
}

message AuthorizeResponse {
  oneof result {
    AuthorizeSuccess success = 1;
    AuthorizeError error = 2;
  }
}

message AuthorizeSuccess {
  string code = 1;
  string state = 2;
  string redirect_uri = 3;
}

message AuthorizeError {
  string error = 1;
  string error_description = 2;
  string error_uri = 3;
  string state = 4;
}

// Authentication Messages
message AuthenticateRequest {
  oneof credential {
    PasswordCredential password = 1;
    RefreshTokenCredential refresh_token = 2;
    SocialCredential social = 3;
  }
  DeviceInfo device_info = 4;
  string ip_address = 5;
}

message PasswordCredential {
  string email = 1 [(buf.validate.field).string.email = true];
  string password = 2 [(buf.validate.field).string.min_len = 8];
}

message RefreshTokenCredential {
  string refresh_token = 1;
}

message SocialCredential {
  string provider = 1;
  string id_token = 2;
  string access_token = 3;
}

message AuthenticateResponse {
  bool success = 1;
  string user_id = 2;
  string session_id = 3;
  bool mfa_required = 4;
  repeated MFAMethod available_mfa_methods = 5;
  AuthenticationError error = 6;
}

message AuthenticationError {
  AuthenticationErrorCode code = 1;
  string message = 2;
  int32 remaining_attempts = 3;
  google.protobuf.Timestamp lockout_until = 4;
}

enum AuthenticationErrorCode {
  AUTHENTICATION_ERROR_CODE_UNSPECIFIED = 0;
  AUTHENTICATION_ERROR_CODE_INVALID_CREDENTIALS = 1;
  AUTHENTICATION_ERROR_CODE_ACCOUNT_LOCKED = 2;
  AUTHENTICATION_ERROR_CODE_ACCOUNT_DISABLED = 3;
  AUTHENTICATION_ERROR_CODE_PASSWORD_EXPIRED = 4;
  AUTHENTICATION_ERROR_CODE_MFA_REQUIRED = 5;
  AUTHENTICATION_ERROR_CODE_RATE_LIMITED = 6;
}

// Session Events Stream
message StreamSessionEventsRequest {
  string user_id = 1;
  repeated SessionEventType event_types = 2;
}

message SessionEvent {
  string event_id = 1;
  SessionEventType event_type = 2;
  string session_id = 3;
  string user_id = 4;
  google.protobuf.Timestamp timestamp = 5;
  map<string, string> details = 6;
}

enum SessionEventType {
  SESSION_EVENT_TYPE_UNSPECIFIED = 0;
  SESSION_EVENT_TYPE_CREATED = 1;
  SESSION_EVENT_TYPE_AUTHENTICATED = 2;
  SESSION_EVENT_TYPE_MFA_COMPLETED = 3;
  SESSION_EVENT_TYPE_RISK_ELEVATED = 4;
  SESSION_EVENT_TYPE_STEP_UP_REQUIRED = 5;
  SESSION_EVENT_TYPE_TERMINATED = 6;
  SESSION_EVENT_TYPE_EXPIRED = 7;
}

// Import common types
import "auth/v1/common.proto";
```


### 8. IAM Policy Service (proto/auth/v1/iam_policy.proto)

```protobuf
syntax = "proto3";

package auth.v1;

option go_package = "github.com/auth-platform/api/gen/go/auth/v1;authv1";

import "buf/validate/validate.proto";
import "google/api/annotations.proto";
import "google/protobuf/timestamp.proto";
import "google/protobuf/struct.proto";

// IAMPolicyService provides authorization and policy management.
service IAMPolicyService {
  // Authorization
  rpc Authorize(AuthorizeRequest) returns (AuthorizeResponse) {
    option (google.api.http) = {
      post: "/v1/iam/authorize"
      body: "*"
    };
  }
  
  rpc BatchAuthorize(BatchAuthorizeRequest) returns (BatchAuthorizeResponse) {
    option (google.api.http) = {
      post: "/v1/iam/authorize/batch"
      body: "*"
    };
  }
  
  // Permissions
  rpc GetUserPermissions(GetUserPermissionsRequest) returns (GetUserPermissionsResponse) {
    option (google.api.http) = {
      get: "/v1/iam/users/{user_id}/permissions"
    };
  }
  
  rpc CheckPermission(CheckPermissionRequest) returns (CheckPermissionResponse) {
    option (google.api.http) = {
      post: "/v1/iam/permissions/check"
      body: "*"
    };
  }
  
  // Roles
  rpc GetUserRoles(GetUserRolesRequest) returns (GetUserRolesResponse) {
    option (google.api.http) = {
      get: "/v1/iam/users/{user_id}/roles"
    };
  }
  
  rpc AssignRole(AssignRoleRequest) returns (AssignRoleResponse) {
    option (google.api.http) = {
      post: "/v1/iam/users/{user_id}/roles"
      body: "*"
    };
  }
  
  rpc RevokeRole(RevokeRoleRequest) returns (RevokeRoleResponse) {
    option (google.api.http) = {
      delete: "/v1/iam/users/{user_id}/roles/{role_id}"
    };
  }
  
  // Policy Management
  rpc CreatePolicy(CreatePolicyRequest) returns (Policy) {
    option (google.api.http) = {
      post: "/v1/iam/policies"
      body: "*"
    };
  }
  
  rpc GetPolicy(GetPolicyRequest) returns (Policy) {
    option (google.api.http) = {
      get: "/v1/iam/policies/{policy_id}"
    };
  }
  
  rpc UpdatePolicy(UpdatePolicyRequest) returns (Policy) {
    option (google.api.http) = {
      put: "/v1/iam/policies/{policy_id}"
      body: "*"
    };
  }
  
  rpc DeletePolicy(DeletePolicyRequest) returns (DeletePolicyResponse) {
    option (google.api.http) = {
      delete: "/v1/iam/policies/{policy_id}"
    };
  }
  
  rpc ListPolicies(ListPoliciesRequest) returns (ListPoliciesResponse) {
    option (google.api.http) = {
      get: "/v1/iam/policies"
    };
  }
  
  // Policy Reload
  rpc ReloadPolicies(ReloadPoliciesRequest) returns (ReloadPoliciesResponse);
}

// Authorization Messages
message AuthorizeRequest {
  // Subject (user or service) requesting access
  Subject subject = 1 [(buf.validate.field).required = true];
  
  // Resource being accessed
  Resource resource = 2 [(buf.validate.field).required = true];
  
  // Action being performed
  string action = 3 [(buf.validate.field).string = { min_len: 1, max_len: 64 }];
  
  // Environment context
  Environment environment = 4;
}

message Subject {
  string id = 1 [(buf.validate.field).required = true];
  SubjectType type = 2;
  map<string, string> attributes = 3;
}

enum SubjectType {
  SUBJECT_TYPE_UNSPECIFIED = 0;
  SUBJECT_TYPE_USER = 1;
  SUBJECT_TYPE_SERVICE = 2;
  SUBJECT_TYPE_GROUP = 3;
}

message Resource {
  string type = 1 [(buf.validate.field).string = { min_len: 1, max_len: 64 }];
  string id = 2;
  map<string, string> attributes = 3;
  string owner_id = 4;
}

message Environment {
  string ip_address = 1;
  google.protobuf.Timestamp timestamp = 2;
  string device_type = 3;
  string location = 4;
  map<string, string> custom = 5;
}

message AuthorizeResponse {
  bool allowed = 1;
  Decision decision = 2;
}

message Decision {
  DecisionEffect effect = 1;
  string policy_id = 2;
  string policy_name = 3;
  repeated string matched_rules = 4;
  string reason = 5;
  repeated Obligation obligations = 6;
}

enum DecisionEffect {
  DECISION_EFFECT_UNSPECIFIED = 0;
  DECISION_EFFECT_ALLOW = 1;
  DECISION_EFFECT_DENY = 2;
  DECISION_EFFECT_NOT_APPLICABLE = 3;
  DECISION_EFFECT_INDETERMINATE = 4;
}

message Obligation {
  string id = 1;
  string action = 2;
  map<string, string> parameters = 3;
}

message BatchAuthorizeRequest {
  repeated AuthorizeRequest requests = 1 [(buf.validate.field).repeated = { min_items: 1, max_items: 100 }];
}

message BatchAuthorizeResponse {
  repeated AuthorizeResponse responses = 1;
}

// Permission Messages
message GetUserPermissionsRequest {
  string user_id = 1 [(buf.validate.field).string.uuid = true];
  string resource_type = 2;
  PaginationRequest pagination = 3;
}

message GetUserPermissionsResponse {
  repeated Permission permissions = 1;
  PaginationResponse pagination = 2;
}

message Permission {
  string resource_type = 1;
  string resource_id = 2;
  repeated string actions = 3;
  repeated string conditions = 4;
}

message CheckPermissionRequest {
  string user_id = 1 [(buf.validate.field).string.uuid = true];
  string permission = 2 [(buf.validate.field).required = true];
  string resource_type = 3;
  string resource_id = 4;
}

message CheckPermissionResponse {
  bool has_permission = 1;
  string granted_by = 2; // Role or policy that granted permission
}

// Role Messages
message GetUserRolesRequest {
  string user_id = 1 [(buf.validate.field).string.uuid = true];
}

message GetUserRolesResponse {
  repeated Role roles = 1;
}

message Role {
  string id = 1;
  string name = 2;
  string description = 3;
  repeated string permissions = 4;
  string parent_role_id = 5;
  google.protobuf.Timestamp created_at = 6;
  google.protobuf.Timestamp updated_at = 7;
}

message AssignRoleRequest {
  string user_id = 1 [(buf.validate.field).string.uuid = true];
  string role_id = 2 [(buf.validate.field).required = true];
  string scope = 3; // Optional: limit role to specific resource
  google.protobuf.Timestamp expires_at = 4;
}

message AssignRoleResponse {
  bool success = 1;
}

message RevokeRoleRequest {
  string user_id = 1 [(buf.validate.field).string.uuid = true];
  string role_id = 2 [(buf.validate.field).required = true];
}

message RevokeRoleResponse {
  bool success = 1;
}

// Policy Messages
message Policy {
  string id = 1;
  string name = 2;
  string description = 3;
  int32 version = 4;
  PolicyType type = 5;
  repeated PolicyRule rules = 6;
  PolicyStatus status = 7;
  google.protobuf.Timestamp created_at = 8;
  google.protobuf.Timestamp updated_at = 9;
}

enum PolicyType {
  POLICY_TYPE_UNSPECIFIED = 0;
  POLICY_TYPE_RBAC = 1;
  POLICY_TYPE_ABAC = 2;
  POLICY_TYPE_REBAC = 3;
}

enum PolicyStatus {
  POLICY_STATUS_UNSPECIFIED = 0;
  POLICY_STATUS_ACTIVE = 1;
  POLICY_STATUS_INACTIVE = 2;
  POLICY_STATUS_DEPRECATED = 3;
}

message PolicyRule {
  string id = 1;
  string name = 2;
  DecisionEffect effect = 3;
  repeated string subjects = 4;
  repeated string resources = 5;
  repeated string actions = 6;
  string condition = 7; // CEL expression
  int32 priority = 8;
}

message CreatePolicyRequest {
  string name = 1 [(buf.validate.field).string = { min_len: 1, max_len: 128 }];
  string description = 2;
  PolicyType type = 3;
  repeated PolicyRule rules = 4;
}

message GetPolicyRequest {
  string policy_id = 1 [(buf.validate.field).required = true];
}

message UpdatePolicyRequest {
  string policy_id = 1 [(buf.validate.field).required = true];
  string name = 2;
  string description = 3;
  repeated PolicyRule rules = 4;
  PolicyStatus status = 5;
}

message DeletePolicyRequest {
  string policy_id = 1 [(buf.validate.field).required = true];
}

message DeletePolicyResponse {
  bool success = 1;
}

message ListPoliciesRequest {
  PolicyType type = 1;
  PolicyStatus status = 2;
  PaginationRequest pagination = 3;
}

message ListPoliciesResponse {
  repeated Policy policies = 1;
  PaginationResponse pagination = 2;
}

message ReloadPoliciesRequest {
  bool force = 1;
}

message ReloadPoliciesResponse {
  bool success = 1;
  int32 policies_loaded = 2;
  string error_message = 3;
}

// Import common types
import "auth/v1/common.proto";
```


## Data Models

### Token Data Model

```
┌─────────────────────────────────────────────────────────────────┐
│                        Token Hierarchy                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────────┐                                           │
│  │  Access Token   │ ─── JWT (RS256/ES256/EdDSA)               │
│  │  (short-lived)  │     - sub: user_id                        │
│  │  15min default  │     - aud: client_id                      │
│  └────────┬────────┘     - scope: permissions                  │
│           │              - cnf: {jkt: dpop_thumbprint}         │
│           │                                                     │
│  ┌────────▼────────┐                                           │
│  │ Refresh Token   │ ─── Opaque or JWT                         │
│  │  (long-lived)   │     - Bound to session                    │
│  │  7 days default │     - Rotation on use                     │
│  └────────┬────────┘                                           │
│           │                                                     │
│  ┌────────▼────────┐                                           │
│  │    ID Token     │ ─── JWT (OIDC)                            │
│  │  (identity)     │     - Standard OIDC claims                │
│  │  Same as access │     - Custom claims                       │
│  └─────────────────┘                                           │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Session State Machine

```
┌─────────────────────────────────────────────────────────────────┐
│                    Session State Machine                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│                    ┌──────────────┐                            │
│                    │   CREATED    │                            │
│                    └──────┬───────┘                            │
│                           │ authenticate                        │
│                           ▼                                     │
│                    ┌──────────────┐                            │
│         ┌─────────│    ACTIVE    │◄────────┐                   │
│         │         └──────┬───────┘         │                   │
│         │                │                 │                   │
│    idle │    risk_elevated    │ step_up_complete               │
│         │                │                 │                   │
│         ▼                ▼                 │                   │
│  ┌──────────────┐ ┌──────────────┐        │                   │
│  │     IDLE     │ │  STEP_UP_    │────────┘                   │
│  └──────┬───────┘ │  REQUIRED    │                            │
│         │         └──────┬───────┘                            │
│         │                │ timeout/fail                        │
│         │                ▼                                     │
│         │         ┌──────────────┐                            │
│         └────────►│  TERMINATED  │◄─── explicit_terminate     │
│                   └──────────────┘                            │
│                          ▲                                     │
│                          │ ttl_expired                         │
│                   ┌──────────────┐                            │
│                   │   EXPIRED    │                            │
│                   └──────────────┘                            │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### WebAuthn Credential Model

```
┌─────────────────────────────────────────────────────────────────┐
│                  WebAuthn Credential Model                      │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  WebAuthnCredential {                                          │
│    credential_id: bytes      // Unique credential identifier   │
│    user_id: string           // Associated user                │
│    public_key: bytes         // COSE public key                │
│    sign_count: uint32        // Signature counter              │
│    transports: []string      // usb, nfc, ble, internal, hybrid│
│    aaguid: bytes             // Authenticator AAGUID           │
│    attestation_type: string  // none, basic, self, attca       │
│    device_name: string       // User-friendly name             │
│    created_at: timestamp                                       │
│    last_used_at: timestamp                                     │
│    backup_eligible: bool     // BE flag (passkey sync)         │
│    backup_state: bool        // BS flag (currently synced)     │
│  }                                                             │
│                                                                 │
│  Passkey Sync Support:                                         │
│  - backup_eligible=true: Credential can be synced              │
│  - backup_state=true: Credential is currently synced           │
│  - Hybrid transport: Cross-device authentication               │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```


## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system—essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*

Based on the prework analysis, the following correctness properties have been identified:

### Property 1: Proto API Structure Compliance

*For any* proto file in the `/api/proto` directory, the file SHALL:
- Pass all Buf lint rules without errors
- Be organized under a versioned package (e.g., `auth.v1`, `infra.resilience.v1`)
- Use BSR-compatible package naming conventions

**Validates: Requirements 1.4, 1.6, 2.1, 2.2, 2.4**

### Property 2: Validation Annotations Completeness

*For any* message field that has semantic constraints (email, UUID, URL, length limits, numeric ranges, required), the field SHALL have corresponding `buf.validate` annotations.

**Validates: Requirements 3.2, 3.4**

### Property 3: REST Mapping Completeness

*For any* RPC method in a service, the method SHALL have a `google.api.http` annotation with:
- Appropriate HTTP method (GET for reads, POST for creates, PUT/PATCH for updates, DELETE for deletes)
- Valid path pattern with parameter placeholders
- Body mapping for non-GET methods

**Validates: Requirements 4.1, 4.3, 4.4**

### Property 4: Enum Completeness

*For any* enum type in the API, the enum SHALL:
- Have an UNSPECIFIED value as the first (0) value
- Include all values required by the corresponding RFC or specification

**Validates: Requirements 9.1, 12.7**

### Property 5: Streaming API Compliance

*For any* streaming RPC method, the streamed message type SHALL include:
- A unique event identifier field
- A timestamp field for ordering

**Validates: Requirements 13.1, 13.4**

### Property 6: Documentation Completeness

*For any* service, RPC method, message, or enum in the API, the element SHALL have a non-empty documentation comment.

**Validates: Requirements 14.1**

### Property 7: Code Generation Round-Trip

*For any* valid proto file, running `buf generate` followed by compilation of generated code SHALL succeed without errors for all target languages (Go, Rust, TypeScript, Python).

**Validates: Requirements 1.2, 5.1, 5.2, 16.4**

## Error Handling

### gRPC Status Code Mapping

| Error Condition | gRPC Status | HTTP Status |
|----------------|-------------|-------------|
| Invalid token | UNAUTHENTICATED | 401 |
| Insufficient permissions | PERMISSION_DENIED | 403 |
| Resource not found | NOT_FOUND | 404 |
| Validation error | INVALID_ARGUMENT | 400 |
| Rate limited | RESOURCE_EXHAUSTED | 429 |
| Internal error | INTERNAL | 500 |
| Service unavailable | UNAVAILABLE | 503 |
| Timeout | DEADLINE_EXCEEDED | 504 |

### Error Response Structure

All errors follow the `google.rpc.Status` pattern with domain-specific details:

```protobuf
// Error response includes:
// - code: gRPC status code
// - message: Human-readable error message
// - details: Array of Any containing:
//   - google.rpc.ErrorInfo: Machine-readable error code and metadata
//   - google.rpc.BadRequest: Field-level validation errors
//   - google.rpc.RetryInfo: Retry delay for retryable errors
```

## Testing Strategy

### Dual Testing Approach

The API modernization requires both unit tests and property-based tests:

1. **Unit Tests**: Verify specific examples and edge cases
   - Proto file syntax validation
   - Individual field validation rules
   - HTTP annotation correctness
   - Generated code compilation

2. **Property-Based Tests**: Verify universal properties across all inputs
   - All proto files pass buf lint (Property 1)
   - All constrained fields have validation annotations (Property 2)
   - All RPCs have HTTP annotations (Property 3)
   - All enums have UNSPECIFIED first value (Property 4)
   - All streaming messages have event IDs (Property 5)
   - All elements have documentation (Property 6)
   - Code generation succeeds for all languages (Property 7)

### Testing Framework

- **Buf CLI**: Primary tool for linting, breaking change detection, and generation
- **pytest** (Python): Property-based tests using Hypothesis for proto file analysis
- **Go tests**: Compilation verification for generated Go code
- **TypeScript/Vitest**: Compilation verification for generated TypeScript code

### Test Configuration

```yaml
# Property-based test configuration
property_tests:
  iterations: 100  # Minimum iterations per property
  timeout: 30s     # Per-test timeout
  
# Test tags for traceability
tags:
  - "Feature: api-proto-modernization-2025"
  - "Property {N}: {property_text}"
```

### CI/CD Integration

```yaml
# .github/workflows/proto-ci.yml
name: Proto CI

on:
  push:
    paths:
      - 'api/**'
  pull_request:
    paths:
      - 'api/**'

jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: bufbuild/buf-setup-action@v1
      - run: buf lint api
      
  breaking:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: bufbuild/buf-setup-action@v1
      - run: buf breaking api --against '.git#branch=main'
      
  generate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: bufbuild/buf-setup-action@v1
      - run: buf generate api
      - name: Verify Go compilation
        run: cd api/gen/go && go build ./...
      - name: Verify TypeScript compilation
        run: cd api/gen/typescript && npm install && npm run build
```
