//! Contract verification types.

use serde::{Deserialize, Serialize};

/// Contract version with git metadata.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ContractVersion {
    /// Consumer name
    pub consumer: String,
    /// Provider name
    pub provider: String,
    /// Semantic version
    pub version: String,
    /// Git commit SHA (40 hex chars)
    pub git_commit: String,
    /// Git branch
    pub branch: String,
    /// Version tags
    pub tags: Vec<String>,
}

impl ContractVersion {
    /// Check if git commit is valid (40 hex characters).
    #[must_use]
    pub fn has_valid_git_commit(&self) -> bool {
        self.git_commit.len() == 40 && self.git_commit.chars().all(|c| c.is_ascii_hexdigit())
    }

    /// Check if tags include git commit.
    #[must_use]
    pub fn tags_include_commit(&self) -> bool {
        self.tags.iter().any(|t| t == &self.git_commit)
    }
}

/// Verification result from Pact Broker.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VerificationResult {
    /// Whether verification succeeded
    pub success: bool,
    /// Provider name
    pub provider: String,
    /// Consumer name
    pub consumer: String,
    /// Consumer version
    pub consumer_version: String,
    /// Provider version
    pub provider_version: String,
    /// Verification timestamp
    pub verified_at: String,
}

impl VerificationResult {
    /// Check if deployment should be allowed.
    #[must_use]
    pub const fn can_deploy(&self) -> bool {
        self.success
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_valid_git_commit() {
        let version = ContractVersion {
            consumer: "auth-edge".to_string(),
            provider: "token-service".to_string(),
            version: "1.0.0".to_string(),
            git_commit: "abc123def456789012345678901234567890abcd".to_string(),
            branch: "main".to_string(),
            tags: vec!["main".to_string(), "abc123def456789012345678901234567890abcd".to_string()],
        };

        assert!(version.has_valid_git_commit());
        assert!(version.tags_include_commit());
    }

    #[test]
    fn test_invalid_git_commit() {
        let version = ContractVersion {
            consumer: "auth-edge".to_string(),
            provider: "token-service".to_string(),
            version: "1.0.0".to_string(),
            git_commit: "short".to_string(),
            branch: "main".to_string(),
            tags: vec![],
        };

        assert!(!version.has_valid_git_commit());
    }

    #[test]
    fn test_verification_result() {
        let success = VerificationResult {
            success: true,
            provider: "token-service".to_string(),
            consumer: "auth-edge".to_string(),
            consumer_version: "1.0.0".to_string(),
            provider_version: "2.0.0".to_string(),
            verified_at: "2025-01-15T00:00:00Z".to_string(),
        };

        assert!(success.can_deploy());

        let failure = VerificationResult {
            success: false,
            ..success
        };

        assert!(!failure.can_deploy());
    }
}
