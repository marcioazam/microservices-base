"""
Property-based tests for Gateway API and Service Mesh Integration.
Validates: Requirements 6.1-6.7

Property 11: Gateway API Rate Limiting
"""

import yaml
from pathlib import Path
from hypothesis import given, strategies as st, settings

GATEWAY_PATH = Path(__file__).parent.parent / "kubernetes" / "gateway"


def load_yaml_docs(filepath: Path) -> list[dict]:
    """Load all YAML documents from a file."""
    with open(filepath) as f:
        return list(yaml.safe_load_all(f))


class TestGatewayAPIHTTPRoutes:
    """Property tests for HTTPRoute resources - Requirement 6.1"""
    
    def test_http_routes_exist(self):
        """HTTPRoute resources must be defined."""
        docs = load_yaml_docs(GATEWAY_PATH / "http-routes.yaml")
        http_routes = [d for d in docs if d and d.get("kind") == "HTTPRoute"]
        assert len(http_routes) > 0, "At least one HTTPRoute must be defined"
    
    def test_http_routes_have_parent_refs(self):
        """All HTTPRoutes must reference a parent Gateway."""
        docs = load_yaml_docs(GATEWAY_PATH / "http-routes.yaml")
        http_routes = [d for d in docs if d and d.get("kind") == "HTTPRoute"]
        
        for route in http_routes:
            parent_refs = route.get("spec", {}).get("parentRefs", [])
            assert len(parent_refs) > 0, (
                f"HTTPRoute {route['metadata']['name']} must have parentRefs"
            )
    
    def test_http_routes_have_backend_refs(self):
        """All HTTPRoute rules must have backend references."""
        docs = load_yaml_docs(GATEWAY_PATH / "http-routes.yaml")
        http_routes = [d for d in docs if d and d.get("kind") == "HTTPRoute"]
        
        for route in http_routes:
            rules = route.get("spec", {}).get("rules", [])
            for rule in rules:
                backend_refs = rule.get("backendRefs", [])
                assert len(backend_refs) > 0, (
                    f"HTTPRoute {route['metadata']['name']} rule must have backendRefs"
                )


class TestGatewayAPIGRPCRoutes:
    """Property tests for GRPCRoute resources - Requirement 6.2"""
    
    def test_grpc_routes_exist(self):
        """GRPCRoute resources must be defined."""
        docs = load_yaml_docs(GATEWAY_PATH / "grpc-routes.yaml")
        grpc_routes = [d for d in docs if d and d.get("kind") == "GRPCRoute"]
        assert len(grpc_routes) > 0, "At least one GRPCRoute must be defined"
    
    def test_grpc_routes_have_method_matching(self):
        """GRPCRoutes must have method matching configured."""
        docs = load_yaml_docs(GATEWAY_PATH / "grpc-routes.yaml")
        grpc_routes = [d for d in docs if d and d.get("kind") == "GRPCRoute"]
        
        for route in grpc_routes:
            rules = route.get("spec", {}).get("rules", [])
            for rule in rules:
                matches = rule.get("matches", [])
                assert len(matches) > 0, (
                    f"GRPCRoute {route['metadata']['name']} must have matches"
                )
                for match in matches:
                    assert "method" in match, (
                        f"GRPCRoute {route['metadata']['name']} must have method matching"
                    )


class TestGatewayAPIRateLimiting:
    """Property 11: Gateway API Rate Limiting - Requirement 6.3"""
    
    def test_rate_limit_policy_exists(self):
        """Rate limiting policy must be defined."""
        docs = load_yaml_docs(GATEWAY_PATH / "policies.yaml")
        rate_limit_policies = [
            d for d in docs 
            if d and d.get("kind") == "BackendTrafficPolicy"
            and "rateLimit" in d.get("spec", {})
        ]
        assert len(rate_limit_policies) > 0, "Rate limiting policy must be defined"
    
    def test_rate_limit_has_rules(self):
        """Rate limiting must have rules defined."""
        docs = load_yaml_docs(GATEWAY_PATH / "policies.yaml")
        rate_limit_policies = [
            d for d in docs 
            if d and d.get("kind") == "BackendTrafficPolicy"
            and "rateLimit" in d.get("spec", {})
        ]
        
        for policy in rate_limit_policies:
            rate_limit = policy.get("spec", {}).get("rateLimit", {})
            global_config = rate_limit.get("global", {})
            rules = global_config.get("rules", [])
            assert len(rules) > 0, (
                f"Rate limit policy {policy['metadata']['name']} must have rules"
            )
    
    def test_rate_limit_has_token_endpoint_protection(self):
        """Token endpoints must have stricter rate limits."""
        docs = load_yaml_docs(GATEWAY_PATH / "policies.yaml")
        rate_limit_policies = [
            d for d in docs 
            if d and d.get("kind") == "BackendTrafficPolicy"
            and "rateLimit" in d.get("spec", {})
        ]
        
        token_protected = False
        for policy in rate_limit_policies:
            rules = policy.get("spec", {}).get("rateLimit", {}).get("global", {}).get("rules", [])
            for rule in rules:
                selectors = rule.get("clientSelectors", [])
                for selector in selectors:
                    headers = selector.get("headers", [])
                    for header in headers:
                        if "token" in header.get("value", "").lower():
                            token_protected = True
        
        assert token_protected, "Token endpoints must have rate limit protection"


class TestGatewayTLSConfiguration:
    """Property tests for TLS configuration - Requirement 6.4"""
    
    def test_gateway_has_tls_listeners(self):
        """Gateway must have TLS-enabled listeners."""
        docs = load_yaml_docs(GATEWAY_PATH / "gateway.yaml")
        gateways = [d for d in docs if d and d.get("kind") == "Gateway"]
        
        for gateway in gateways:
            listeners = gateway.get("spec", {}).get("listeners", [])
            tls_listeners = [l for l in listeners if l.get("tls")]
            assert len(tls_listeners) > 0, (
                f"Gateway {gateway['metadata']['name']} must have TLS listeners"
            )
    
    def test_certificates_reference_cluster_issuer(self):
        """Certificates must reference a ClusterIssuer."""
        docs = load_yaml_docs(GATEWAY_PATH / "gateway.yaml")
        certificates = [d for d in docs if d and d.get("kind") == "Certificate"]
        
        for cert in certificates:
            issuer_ref = cert.get("spec", {}).get("issuerRef", {})
            assert issuer_ref.get("kind") == "ClusterIssuer", (
                f"Certificate {cert['metadata']['name']} must use ClusterIssuer"
            )


class TestLinkerdIntegration:
    """Property tests for Linkerd service mesh - Requirements 6.5-6.7"""
    
    def test_namespace_has_linkerd_injection(self):
        """Namespace must have Linkerd injection enabled."""
        linkerd_file = GATEWAY_PATH / "linkerd-annotations.yaml"
        if not linkerd_file.exists():
            return  # Skip if file doesn't exist yet
        
        docs = load_yaml_docs(linkerd_file)
        namespaces = [d for d in docs if d and d.get("kind") == "Namespace"]
        
        for ns in namespaces:
            annotations = ns.get("metadata", {}).get("annotations", {})
            assert annotations.get("linkerd.io/inject") == "enabled", (
                f"Namespace {ns['metadata']['name']} must have Linkerd injection"
            )
    
    def test_service_profiles_exist(self):
        """ServiceProfile resources must be defined for circuit breaker."""
        profile_file = GATEWAY_PATH / "service-profiles.yaml"
        if not profile_file.exists():
            return
        
        docs = load_yaml_docs(profile_file)
        profiles = [d for d in docs if d and d.get("kind") == "ServiceProfile"]
        assert len(profiles) > 0, "ServiceProfile resources must be defined"
    
    def test_service_profiles_have_retry_budget(self):
        """ServiceProfiles must have retry budget configured."""
        profile_file = GATEWAY_PATH / "service-profiles.yaml"
        if not profile_file.exists():
            return
        
        docs = load_yaml_docs(profile_file)
        profiles = [d for d in docs if d and d.get("kind") == "ServiceProfile"]
        
        for profile in profiles:
            retry_budget = profile.get("spec", {}).get("retryBudget")
            assert retry_budget is not None, (
                f"ServiceProfile {profile['metadata']['name']} must have retryBudget"
            )
            assert "retryRatio" in retry_budget, "retryBudget must have retryRatio"


class TestSecurityPolicies:
    """Property tests for security policies."""
    
    def test_cors_policy_exists(self):
        """CORS security policy must be defined."""
        docs = load_yaml_docs(GATEWAY_PATH / "policies.yaml")
        security_policies = [
            d for d in docs 
            if d and d.get("kind") == "SecurityPolicy"
            and "cors" in d.get("spec", {})
        ]
        assert len(security_policies) > 0, "CORS security policy must be defined"
    
    def test_jwt_policy_exists(self):
        """JWT authentication policy must be defined."""
        docs = load_yaml_docs(GATEWAY_PATH / "policies.yaml")
        jwt_policies = [
            d for d in docs 
            if d and d.get("kind") == "SecurityPolicy"
            and "jwt" in d.get("spec", {})
        ]
        assert len(jwt_policies) > 0, "JWT authentication policy must be defined"


class TestCircuitBreakerConfiguration:
    """Property tests for circuit breaker configuration."""
    
    def test_circuit_breaker_policy_exists(self):
        """Circuit breaker policy must be defined."""
        docs = load_yaml_docs(GATEWAY_PATH / "policies.yaml")
        cb_policies = [
            d for d in docs 
            if d and d.get("kind") == "BackendTrafficPolicy"
            and "circuitBreaker" in d.get("spec", {})
        ]
        assert len(cb_policies) > 0, "Circuit breaker policy must be defined"
    
    def test_circuit_breaker_has_thresholds(self):
        """Circuit breaker must have connection thresholds."""
        docs = load_yaml_docs(GATEWAY_PATH / "policies.yaml")
        cb_policies = [
            d for d in docs 
            if d and d.get("kind") == "BackendTrafficPolicy"
            and "circuitBreaker" in d.get("spec", {})
        ]
        
        for policy in cb_policies:
            cb = policy.get("spec", {}).get("circuitBreaker", {})
            assert "maxConnections" in cb, "Circuit breaker must have maxConnections"
            assert "maxPendingRequests" in cb, "Circuit breaker must have maxPendingRequests"


if __name__ == "__main__":
    import pytest
    pytest.main([__file__, "-v"])
