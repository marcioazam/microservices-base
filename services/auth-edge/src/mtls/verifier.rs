use crate::error::AuthEdgeError;
use x509_parser::prelude::*;
use std::time::SystemTime;
use std::io::Cursor;

pub struct CertificateVerifier {
    trust_domain: String,
}

impl CertificateVerifier {
    pub fn new(trust_domain: String) -> Self {
        CertificateVerifier { trust_domain }
    }

    pub fn verify_certificate(&self, cert_pem: &str) -> Result<(), AuthEdgeError> {
        // Parse PEM-encoded certificate using rustls-pemfile
        let mut cursor = Cursor::new(cert_pem.as_bytes());
        let cert_der = rustls_pemfile::certs(&mut cursor)
            .next()
            .ok_or_else(|| AuthEdgeError::CertificateError { reason: "No PEM certificate found".to_string() })?
            .map_err(|e| AuthEdgeError::CertificateError { reason: format!("Failed to parse PEM: {}", e) })?;

        let (_, cert) = X509Certificate::from_der(&cert_der)
            .map_err(|e| AuthEdgeError::CertificateError { reason: format!("Failed to parse certificate: {}", e) })?;

        // Check validity period
        self.check_validity(&cert)?;

        // Verify trust domain in SPIFFE ID
        self.verify_trust_domain(&cert)?;

        Ok(())
    }

    fn check_validity(&self, cert: &X509Certificate) -> Result<(), AuthEdgeError> {
        let now = SystemTime::now()
            .duration_since(SystemTime::UNIX_EPOCH)
            .unwrap()
            .as_secs() as i64;

        let not_before = cert.validity().not_before.timestamp();
        let not_after = cert.validity().not_after.timestamp();

        if now < not_before {
            return Err(AuthEdgeError::CertificateError { reason: "Certificate not yet valid".to_string() });
        }

        if now > not_after {
            return Err(AuthEdgeError::CertificateError { reason: "Certificate expired".to_string() });
        }

        Ok(())
    }

    fn verify_trust_domain(&self, cert: &X509Certificate) -> Result<(), AuthEdgeError> {
        for ext in cert.extensions() {
            if let ParsedExtension::SubjectAlternativeName(san) = ext.parsed_extension() {
                for name in &san.general_names {
                    if let GeneralName::URI(uri) = name {
                        if uri.starts_with("spiffe://") {
                            let expected_prefix = format!("spiffe://{}/", self.trust_domain);
                            if uri.starts_with(&expected_prefix) {
                                return Ok(());
                            }
                        }
                    }
                }
            }
        }

        Err(AuthEdgeError::CertificateError {
            reason: format!("Certificate not from trusted domain: {}", self.trust_domain)
        })
    }
}
