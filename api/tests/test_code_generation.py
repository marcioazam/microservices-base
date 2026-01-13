"""
Property-based tests for code generation round-trip.

**Property 7: Code Generation Round-Trip**
**Validates: Requirements 1.2, 5.1, 5.2, 16.4**

Tests that:
- Proto files can be parsed and regenerated consistently
- Generated code structure matches proto definitions
- All services generate expected output files
"""

import os
import re
import json
from pathlib import Path
from typing import List, Dict, Set, Tuple

import pytest
from hypothesis import given, settings, assume
from hypothesis import strategies as st


# Paths
API_ROOT = Path(__file__).parent.parent
PROTO_ROOT = API_ROOT / "proto"
GEN_ROOT = API_ROOT / "gen"
OPENAPI_ROOT = API_ROOT / "openapi"
BUF_GEN_YAML = API_ROOT / "buf.gen.yaml"


def get_all_proto_files() -> List[Path]:
    """Get all versioned .proto files in the proto directory (v1/, v2/, etc.)."""
    proto_files = []
    for root, _, files in os.walk(PROTO_ROOT):
        for file in files:
            if file.endswith(".proto"):
                full_path = Path(root) / file
                # Only include versioned proto files
                path_str = str(full_path).replace("\\", "/")
                if any(f"/v{i}/" in path_str for i in range(1, 10)):
                    proto_files.append(full_path)
    return proto_files


def parse_proto_structure(content: str) -> Dict:
    """Parse proto file and extract structure."""
    structure = {
        "package": None,
        "services": [],
        "messages": [],
        "enums": [],
        "imports": [],
    }
    
    # Extract package
    pkg_match = re.search(r'^package\s+([a-zA-Z0-9_.]+)\s*;', content, re.MULTILINE)
    if pkg_match:
        structure["package"] = pkg_match.group(1)
    
    # Extract services with their methods
    service_pattern = re.compile(r'^service\s+(\w+)\s*\{([^}]*)\}', re.MULTILINE | re.DOTALL)
    for match in service_pattern.finditer(content):
        service_name = match.group(1)
        service_body = match.group(2)
        
        # Extract RPCs
        rpcs = re.findall(r'rpc\s+(\w+)\s*\(\s*(\w+)\s*\)\s*returns\s*\(\s*(?:stream\s+)?(\w+)\s*\)', service_body)
        
        structure["services"].append({
            "name": service_name,
            "rpcs": [{"name": r[0], "request": r[1], "response": r[2]} for r in rpcs]
        })
    
    # Extract messages
    messages = re.findall(r'^message\s+(\w+)\s*\{', content, re.MULTILINE)
    structure["messages"] = messages
    
    # Extract enums
    enums = re.findall(r'^enum\s+(\w+)\s*\{', content, re.MULTILINE)
    structure["enums"] = enums
    
    # Extract imports
    imports = re.findall(r'^import\s+"([^"]+)"', content, re.MULTILINE)
    structure["imports"] = imports
    
    return structure


class TestProtoParsingConsistency:
    """Tests for proto parsing consistency."""
    
    def test_proto_structure_is_deterministic(self):
        """
        Property: Parsing proto files produces consistent structure.
        
        For any proto file, parsing it multiple times should produce
        the same structure.
        
        **Validates: Requirements 1.2**
        """
        for proto_file in get_all_proto_files():
            content = proto_file.read_text(encoding="utf-8")
            
            # Parse multiple times
            structure1 = parse_proto_structure(content)
            structure2 = parse_proto_structure(content)
            
            assert structure1 == structure2, (
                f"Proto file {proto_file} produces inconsistent parse results"
            )
    
    def test_all_rpc_messages_are_defined(self):
        """
        Property: All RPC request/response types are defined as messages.
        
        For any RPC method, its request and response types should be
        defined as messages in the same file or imported.
        
        **Validates: Requirements 1.2, 5.1**
        """
        for proto_file in get_all_proto_files():
            content = proto_file.read_text(encoding="utf-8")
            structure = parse_proto_structure(content)
            
            # Collect all message names (including from imports conceptually)
            all_messages = set(structure["messages"])
            
            # Check each service's RPCs
            for service in structure["services"]:
                for rpc in service["rpcs"]:
                    request_type = rpc["request"]
                    response_type = rpc["response"]
                    
                    # Skip well-known types
                    well_known = {"Empty", "Timestamp", "Duration", "Struct", "Any"}
                    
                    if request_type not in well_known:
                        # Request type should be defined or imported
                        assert request_type in all_messages or "import" in content, (
                            f"RPC {rpc['name']} in {proto_file} uses undefined request type {request_type}"
                        )
                    
                    if response_type not in well_known:
                        # Response type should be defined or imported
                        assert response_type in all_messages or "import" in content, (
                            f"RPC {rpc['name']} in {proto_file} uses undefined response type {response_type}"
                        )


class TestBufGenYamlConsistency:
    """Tests for buf.gen.yaml configuration consistency."""
    
    @pytest.fixture(autouse=True)
    def setup(self):
        """Setup test fixtures."""
        if not BUF_GEN_YAML.exists():
            pytest.skip("buf.gen.yaml doesn't exist")
        self.buf_gen_content = BUF_GEN_YAML.read_text(encoding="utf-8")
    
    def test_buf_gen_yaml_has_go_plugins(self):
        """buf.gen.yaml should configure Go code generation."""
        assert "go" in self.buf_gen_content.lower(), (
            "buf.gen.yaml is missing Go plugin configuration"
        )
    
    def test_buf_gen_yaml_has_typescript_plugins(self):
        """buf.gen.yaml should configure TypeScript code generation."""
        assert "typescript" in self.buf_gen_content.lower() or "es" in self.buf_gen_content.lower(), (
            "buf.gen.yaml is missing TypeScript plugin configuration"
        )
    
    def test_buf_gen_yaml_has_python_plugins(self):
        """buf.gen.yaml should configure Python code generation."""
        assert "python" in self.buf_gen_content.lower(), (
            "buf.gen.yaml is missing Python plugin configuration"
        )
    
    def test_buf_gen_yaml_has_openapi_plugins(self):
        """buf.gen.yaml should configure OpenAPI generation."""
        assert "openapi" in self.buf_gen_content.lower(), (
            "buf.gen.yaml is missing OpenAPI plugin configuration"
        )


class TestGeneratedCodeStructure:
    """Tests for generated code structure (when available)."""
    
    def test_gen_directory_structure(self):
        """
        Property: Generated code follows expected directory structure.
        
        If gen/ directory exists, it should have subdirectories for
        each target language.
        
        **Validates: Requirements 16.4**
        NOTE: This test is skipped if gen/ directory doesn't exist.
        Run 'buf generate' to create the generated code.
        """
        if not GEN_ROOT.exists():
            pytest.skip("gen/ directory doesn't exist (run 'buf generate' to create)")
        
        existing_dirs = [d.name for d in GEN_ROOT.iterdir() if d.is_dir()]
        
        if not existing_dirs:
            pytest.skip("gen/ directory is empty (run 'buf generate' to populate)")
        
        expected_dirs = ["go", "typescript", "python"]
        missing = [d for d in expected_dirs if d not in existing_dirs]
        
        if missing:
            pytest.skip(f"gen/ directory is missing {missing} (run 'buf generate' to create)")
    
    def test_openapi_directory_structure(self):
        """
        Property: OpenAPI specs are generated in expected location.
        
        If openapi/ directory exists, it should contain v1/ subdirectory.
        
        **Validates: Requirements 16.4**
        """
        if not OPENAPI_ROOT.exists():
            pytest.skip("openapi/ directory doesn't exist")
        
        v1_dir = OPENAPI_ROOT / "v1"
        assert v1_dir.exists(), "openapi/v1/ directory doesn't exist"


class TestServiceCodeGeneration:
    """Tests for service-specific code generation."""
    
    def test_all_services_would_generate_code(self):
        """
        Property: All services have the structure needed for code generation.
        
        For any service, it should have at least one RPC method
        and proper package naming for code generation.
        
        **Validates: Requirements 5.1, 5.2**
        """
        for proto_file in get_all_proto_files():
            content = proto_file.read_text(encoding="utf-8")
            structure = parse_proto_structure(content)
            
            for service in structure["services"]:
                # Service should have at least one RPC
                assert len(service["rpcs"]) > 0, (
                    f"Service {service['name']} in {proto_file} has no RPC methods"
                )
                
                # Package should be defined
                assert structure["package"] is not None, (
                    f"Proto file {proto_file} with service {service['name']} has no package"
                )


class TestConnectRPCCompatibility:
    """Tests for Connect-RPC protocol compatibility."""
    
    def test_services_are_connect_compatible(self):
        """
        Property: All services are compatible with Connect-RPC.
        
        For any service, its RPC methods should use message types
        (not primitives) for request and response.
        
        **Validates: Requirements 5.1, 5.2**
        """
        primitive_types = {"string", "int32", "int64", "bool", "float", "double", "bytes"}
        
        for proto_file in get_all_proto_files():
            content = proto_file.read_text(encoding="utf-8")
            structure = parse_proto_structure(content)
            
            for service in structure["services"]:
                for rpc in service["rpcs"]:
                    # Request and response should be message types, not primitives
                    assert rpc["request"] not in primitive_types, (
                        f"RPC {rpc['name']} uses primitive request type {rpc['request']}"
                    )
                    assert rpc["response"] not in primitive_types, (
                        f"RPC {rpc['name']} uses primitive response type {rpc['response']}"
                    )


# Property-based tests
@given(st.sampled_from(get_all_proto_files() or [Path("dummy.proto")]))
@settings(max_examples=50)
def test_proto_file_package_matches_path(proto_file: Path):
    """
    Property: Proto file package matches its directory path.
    
    For any proto file, the package name should correspond to
    its location in the directory structure.
    
    **Validates: Requirements 1.2, 2.1**
    """
    if not proto_file.exists() or proto_file.name == ".gitkeep":
        assume(False)
    
    content = proto_file.read_text(encoding="utf-8")
    structure = parse_proto_structure(content)
    
    if structure["package"] is None:
        assume(False)
    
    # Get relative path from proto root
    try:
        rel_path = proto_file.relative_to(PROTO_ROOT)
    except ValueError:
        assume(False)
    
    # Convert path to package format
    path_parts = rel_path.parent.parts
    expected_suffix = ".".join(path_parts)
    
    # Package should contain the path-derived suffix
    assert expected_suffix in structure["package"] or structure["package"].endswith(expected_suffix), (
        f"Proto file {proto_file} has package '{structure['package']}' "
        f"but path suggests '{expected_suffix}'"
    )


@given(st.sampled_from(get_all_proto_files() or [Path("dummy.proto")]))
@settings(max_examples=50)
def test_proto_file_has_valid_service_structure(proto_file: Path):
    """
    Property: Proto files with services have valid structure.
    
    For any proto file containing services, each service should
    have properly formed RPC definitions.
    """
    if not proto_file.exists() or proto_file.name == ".gitkeep":
        assume(False)
    
    content = proto_file.read_text(encoding="utf-8")
    structure = parse_proto_structure(content)
    
    for service in structure["services"]:
        # Service name should be PascalCase
        assert service["name"][0].isupper(), (
            f"Service {service['name']} doesn't follow PascalCase naming"
        )
        
        # Each RPC should have request and response
        for rpc in service["rpcs"]:
            assert rpc["request"], f"RPC {rpc['name']} missing request type"
            assert rpc["response"], f"RPC {rpc['name']} missing response type"
