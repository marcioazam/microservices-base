//! mTLS connection types for Linkerd service mesh.

use serde::{Deserialize, Serialize};

/// mTLS connection information from Linkerd proxy.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct MtlsConnection {
    /// Source workload SPIFFE identity
    pub source_identity: String,
    /// Destination workload SPIFFE identity
    pub dest_identity: String,
    /// TLS version (should be TLSv1.3)
    pub tls_version: String,
    /// Cipher suite used
    pub cipher_suite: String,
    /// Whether certificate is valid
    pub cert_valid: bool,
}

impl MtlsConnection {
    /// Create a new mTLS connection.
    #[must_use]
    pub fn new(
        source_identity: impl Into<String>,
        dest_identity: impl Into<String>,
    ) -> Self {
        Self {
            source_identity: source_identity.into(),
            dest_identity: dest_identity.into(),
            tls_version: "TLSv1.3".to_string(),
            cipher_suite: "TLS_AES_256_GCM_SHA384".to_string(),
            cert_valid: true,
        }
    }

    /// Check if connection uses SPIFFE identities.
    #[must_use]
    pub fn has_spiffe_identities(&self) -> bool {
        self.source_identity.starts_with("spiffe://")
            && self.dest_identity.starts_with("spiffe://")
    }

    /// Check if connection uses TLS 1.3.
    #[must_use]
    pub fn is_tls_1_3(&self) -> bool {
        self.tls_version == "TLSv1.3"
    }

    /// Check if connection is fully valid.
    #[must_use]
    pub fn is_valid(&self) -> bool {
        self.cert_valid && self.has_spiffe_identities() && self.is_tls_1_3()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_mtls_connection_new() {
        let conn = MtlsConnection::new(
            "spiffe://cluster.local/ns/default/sa/client",
            "spiffe://cluster.local/ns/default/sa/server",
        );

        assert!(conn.is_valid());
        assert!(conn.has_spiffe_identities());
        assert!(conn.is_tls_1_3());
    }

    #[test]
    fn test_invalid_identity() {
        let conn = MtlsConnection {
            source_identity: "not-spiffe".to_string(),
            dest_identity: "spiffe://cluster.local/ns/default/sa/server".to_string(),
            tls_version: "TLSv1.3".to_string(),
            cipher_suite: "TLS_AES_256_GCM_SHA384".to_string(),
            cert_valid: true,
        };

        assert!(!conn.has_spiffe_identities());
        assert!(!conn.is_valid());
    }
}
