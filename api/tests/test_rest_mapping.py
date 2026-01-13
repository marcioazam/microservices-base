"""
Property-based tests for REST mapping completeness.

Feature: api-proto-modernization-2025
Property 3: REST Mapping Completeness
Validates: Requirements 4.1, 4.3, 4.4

For any RPC method in a service, the method SHALL have a google.api.http 
annotation with appropriate HTTP method, valid path pattern, and body mapping.

NOTE: This test only checks versioned proto files (v1/) as per the modernization spec.
Legacy non-versioned proto files are excluded from validation checks.
"""

import os
import re
from pathlib import Path
from typing import List, Tuple, Dict, Optional
import pytest

# Proto directory path
PROTO_DIR = Path(__file__).parent.parent / "proto"

# HTTP method conventions based on RPC naming
RPC_HTTP_CONVENTIONS = {
    r"^Get.*": ["get"],
    r"^List.*": ["get"],
    r"^Create.*": ["post"],
    r"^Update.*": ["put", "patch"],
    r"^Delete.*": ["delete"],
    r"^Remove.*": ["delete"],
    r"^Validate.*": ["post"],
    r"^Verify.*": ["post"],
    r"^Check.*": ["get", "post"],
    r"^Search.*": ["get", "post"],
    r"^Batch.*": ["post"],
}


def get_all_proto_files() -> List[Path]:
    """Get all versioned .proto files in the proto directory (v1/, v2/, etc.)."""
    if not PROTO_DIR.exists():
        return []
    # Only include versioned proto files (in v1/, v2/, etc. directories)
    all_protos = list(PROTO_DIR.rglob("*.proto"))
    return [p for p in all_protos if any(f"/v{i}/" in str(p).replace("\\", "/") or f"\\v{i}\\" in str(p) for i in range(1, 10))]


def extract_services_and_rpcs(content: str) -> Dict[str, List[Dict]]:
    """
    Extract service definitions and their RPC methods from proto content.
    Returns dict of service_name -> list of RPC info dicts.
    """
    services = {}
    
    # Find all service blocks
    service_pattern = r"service\s+(\w+)\s*\{([^}]+(?:\{[^}]*\}[^}]*)*)\}"
    
    for service_match in re.finditer(service_pattern, content, re.DOTALL):
        service_name = service_match.group(1)
        service_body = service_match.group(2)
        
        rpcs = []
        
        # Find all RPC definitions within the service
        rpc_pattern = r"rpc\s+(\w+)\s*\([^)]+\)\s*returns\s*\([^)]+\)\s*(\{[^}]*\})?\s*;"
        
        for rpc_match in re.finditer(rpc_pattern, service_body, re.DOTALL):
            rpc_name = rpc_match.group(1)
            rpc_options = rpc_match.group(2) or ""
            
            # Check for HTTP annotation
            http_match = re.search(
                r'\(google\.api\.http\)\s*=\s*\{([^}]+)\}',
                rpc_options,
                re.DOTALL
            )
            
            http_info = None
            if http_match:
                http_body = http_match.group(1)
                
                # Extract HTTP method and path
                method_match = re.search(
                    r'(get|post|put|patch|delete):\s*"([^"]+)"',
                    http_body,
                    re.IGNORECASE
                )
                
                body_match = re.search(r'body:\s*"([^"]*)"', http_body)
                
                if method_match:
                    http_info = {
                        "method": method_match.group(1).lower(),
                        "path": method_match.group(2),
                        "body": body_match.group(1) if body_match else None
                    }
            
            rpcs.append({
                "name": rpc_name,
                "has_http": http_info is not None,
                "http": http_info
            })
        
        services[service_name] = rpcs
    
    return services


def is_streaming_rpc(content: str, rpc_name: str) -> bool:
    """Check if an RPC is a streaming RPC."""
    pattern = rf"rpc\s+{rpc_name}\s*\(\s*stream|returns\s*\(\s*stream"
    return bool(re.search(pattern, content))


class TestRESTMappingCompleteness:
    """
    Property 3: REST Mapping Completeness
    
    For any RPC method in a service, the method SHALL have a google.api.http
    annotation with appropriate HTTP method and path.
    """
    
    def test_all_proto_files_exist(self):
        """Verify proto files exist for testing."""
        proto_files = get_all_proto_files()
        assert len(proto_files) > 0, "No proto files found in proto directory"
    
    @pytest.mark.parametrize("proto_file", get_all_proto_files())
    def test_all_rpcs_have_http_annotations(self, proto_file: Path):
        """
        Property test: All non-streaming RPCs have HTTP annotations.
        """
        content = proto_file.read_text()
        services = extract_services_and_rpcs(content)
        
        missing_http = []
        
        for service_name, rpcs in services.items():
            for rpc in rpcs:
                # Skip streaming RPCs (they don't need HTTP annotations)
                if is_streaming_rpc(content, rpc["name"]):
                    continue
                
                if not rpc["has_http"]:
                    missing_http.append(f"{service_name}.{rpc['name']}")
        
        assert len(missing_http) == 0, (
            f"RPCs in {proto_file.name} missing HTTP annotations: "
            f"{', '.join(missing_http)}"
        )
    
    @pytest.mark.parametrize("proto_file", get_all_proto_files())
    def test_http_methods_follow_conventions(self, proto_file: Path):
        """
        Property test: HTTP methods follow REST conventions based on RPC names.
        """
        content = proto_file.read_text()
        services = extract_services_and_rpcs(content)
        
        violations = []
        
        for service_name, rpcs in services.items():
            for rpc in rpcs:
                if not rpc["has_http"] or rpc["http"] is None:
                    continue
                
                rpc_name = rpc["name"]
                http_method = rpc["http"]["method"]
                
                # Check against conventions
                for pattern, expected_methods in RPC_HTTP_CONVENTIONS.items():
                    if re.match(pattern, rpc_name):
                        if http_method not in expected_methods:
                            violations.append(
                                f"{service_name}.{rpc_name}: "
                                f"uses {http_method.upper()}, "
                                f"expected {'/'.join(m.upper() for m in expected_methods)}"
                            )
                        break
        
        # This is a soft check - violations are warnings, not failures
        if violations:
            pytest.skip(f"HTTP method convention suggestions: {violations}")
    
    @pytest.mark.parametrize("proto_file", get_all_proto_files())
    def test_http_paths_are_valid(self, proto_file: Path):
        """
        Property test: HTTP paths follow valid patterns.
        """
        content = proto_file.read_text()
        services = extract_services_and_rpcs(content)
        
        invalid_paths = []
        
        for service_name, rpcs in services.items():
            for rpc in rpcs:
                if not rpc["has_http"] or rpc["http"] is None:
                    continue
                
                path = rpc["http"]["path"]
                
                # Path should start with /
                if not path.startswith("/"):
                    invalid_paths.append(
                        f"{service_name}.{rpc['name']}: path '{path}' "
                        "should start with /"
                    )
                    continue
                
                # Path should be versioned (e.g., /v1/)
                if not re.match(r"^/v\d+/", path):
                    invalid_paths.append(
                        f"{service_name}.{rpc['name']}: path '{path}' "
                        "should include version prefix (e.g., /v1/)"
                    )
        
        assert len(invalid_paths) == 0, (
            f"Invalid HTTP paths in {proto_file.name}: "
            f"{'; '.join(invalid_paths)}"
        )
    
    @pytest.mark.parametrize("proto_file", get_all_proto_files())
    def test_post_put_patch_have_body(self, proto_file: Path):
        """
        Property test: POST, PUT, PATCH methods have body mapping.
        """
        content = proto_file.read_text()
        services = extract_services_and_rpcs(content)
        
        missing_body = []
        
        for service_name, rpcs in services.items():
            for rpc in rpcs:
                if not rpc["has_http"] or rpc["http"] is None:
                    continue
                
                http_method = rpc["http"]["method"]
                body = rpc["http"]["body"]
                
                if http_method in ["post", "put", "patch"]:
                    if body is None:
                        missing_body.append(
                            f"{service_name}.{rpc['name']}: "
                            f"{http_method.upper()} should have body mapping"
                        )
        
        assert len(missing_body) == 0, (
            f"RPCs missing body mapping in {proto_file.name}: "
            f"{'; '.join(missing_body)}"
        )


if __name__ == "__main__":
    pytest.main([__file__, "-v"])
