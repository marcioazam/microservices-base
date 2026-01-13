//! Pact Broker matrix types for can-i-deploy.

use serde::{Deserialize, Serialize};

/// Result of can-i-deploy check.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CanIDeployResult {
    /// Whether deployment is allowed
    pub ok: bool,
    /// Reason for result
    pub reason: String,
    /// Verification matrix
    pub matrix: Vec<MatrixEntry>,
}

impl CanIDeployResult {
    /// Create from matrix entries.
    #[must_use]
    pub fn from_matrix(matrix: Vec<MatrixEntry>) -> Self {
        let ok = matrix.iter().all(|e| e.success);
        let reason = if ok {
            "All contracts verified".to_string()
        } else {
            let failed: Vec<_> = matrix
                .iter()
                .filter(|e| !e.success)
                .map(|e| format!("{} -> {}", e.consumer, e.provider))
                .collect();
            format!("Verification failed: {}", failed.join(", "))
        };

        Self { ok, reason, matrix }
    }

    /// Check if deployment is allowed.
    #[must_use]
    pub const fn can_deploy(&self) -> bool {
        self.ok
    }
}

/// Entry in the verification matrix.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MatrixEntry {
    /// Consumer name
    pub consumer: String,
    /// Consumer version
    pub consumer_version: String,
    /// Provider name
    pub provider: String,
    /// Provider version
    pub provider_version: String,
    /// Whether verification succeeded
    pub success: bool,
}

impl MatrixEntry {
    /// Create a new matrix entry.
    #[must_use]
    pub fn new(
        consumer: impl Into<String>,
        consumer_version: impl Into<String>,
        provider: impl Into<String>,
        provider_version: impl Into<String>,
        success: bool,
    ) -> Self {
        Self {
            consumer: consumer.into(),
            consumer_version: consumer_version.into(),
            provider: provider.into(),
            provider_version: provider_version.into(),
            success,
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_can_deploy_all_success() {
        let matrix = vec![
            MatrixEntry::new("auth-edge", "1.0.0", "token-service", "2.0.0", true),
            MatrixEntry::new("session-core", "1.0.0", "token-service", "2.0.0", true),
        ];

        let result = CanIDeployResult::from_matrix(matrix);
        assert!(result.can_deploy());
        assert!(result.reason.contains("All contracts verified"));
    }

    #[test]
    fn test_can_deploy_with_failure() {
        let matrix = vec![
            MatrixEntry::new("auth-edge", "1.0.0", "token-service", "2.0.0", true),
            MatrixEntry::new("session-core", "1.0.0", "token-service", "2.0.0", false),
        ];

        let result = CanIDeployResult::from_matrix(matrix);
        assert!(!result.can_deploy());
        assert!(result.reason.contains("Verification failed"));
    }

    #[test]
    fn test_empty_matrix() {
        let result = CanIDeployResult::from_matrix(vec![]);
        assert!(result.can_deploy());
    }
}
