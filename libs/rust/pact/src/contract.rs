//! Pact contract types.

use serde::{Deserialize, Serialize};
use std::collections::HashMap;

/// A Pact contract between consumer and provider.
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct Contract {
    /// Consumer participant
    pub consumer: Participant,
    /// Provider participant
    pub provider: Participant,
    /// Contract interactions
    pub interactions: Vec<Interaction>,
    /// Contract metadata
    pub metadata: ContractMetadata,
}

/// A participant in a contract (consumer or provider).
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct Participant {
    /// Participant name
    pub name: String,
}

impl Participant {
    /// Create a new participant.
    #[must_use]
    pub fn new(name: impl Into<String>) -> Self {
        Self { name: name.into() }
    }
}

/// An interaction in a contract.
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct Interaction {
    /// Interaction description
    pub description: String,
    /// Provider state (precondition)
    #[serde(skip_serializing_if = "Option::is_none")]
    pub provider_state: Option<String>,
    /// Expected request
    pub request: Request,
    /// Expected response
    pub response: Response,
}

/// HTTP request in an interaction.
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct Request {
    /// HTTP method
    pub method: String,
    /// Request path
    pub path: String,
    /// Request headers
    #[serde(default)]
    pub headers: HashMap<String, String>,
    /// Request body
    #[serde(skip_serializing_if = "Option::is_none")]
    pub body: Option<serde_json::Value>,
}

/// HTTP response in an interaction.
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct Response {
    /// HTTP status code
    pub status: u16,
    /// Response headers
    #[serde(default)]
    pub headers: HashMap<String, String>,
    /// Response body
    #[serde(skip_serializing_if = "Option::is_none")]
    pub body: Option<serde_json::Value>,
}

/// Contract metadata.
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct ContractMetadata {
    /// Pact specification version
    #[serde(rename = "pactSpecification")]
    pub pact_specification: PactSpecification,
}

/// Pact specification version.
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct PactSpecification {
    /// Version string
    pub version: String,
}

impl Default for ContractMetadata {
    fn default() -> Self {
        Self {
            pact_specification: PactSpecification {
                version: "4.0".to_string(),
            },
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_contract_serialization() {
        let contract = Contract {
            consumer: Participant::new("auth-edge"),
            provider: Participant::new("token-service"),
            interactions: vec![Interaction {
                description: "get token".to_string(),
                provider_state: Some("user exists".to_string()),
                request: Request {
                    method: "POST".to_string(),
                    path: "/token".to_string(),
                    headers: HashMap::new(),
                    body: None,
                },
                response: Response {
                    status: 200,
                    headers: HashMap::new(),
                    body: None,
                },
            }],
            metadata: ContractMetadata::default(),
        };

        let json = serde_json::to_string(&contract).unwrap();
        let restored: Contract = serde_json::from_str(&json).unwrap();
        assert_eq!(contract, restored);
    }
}
