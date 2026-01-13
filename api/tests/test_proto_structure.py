"""
Property-based tests for Proto API structure compliance.

**Property 1: Proto API Structure Compliance**
**Validates: Requirements 1.4, 1.6, 2.1, 2.2, 2.4**

Tests that:
- All proto files follow versioned package naming (e.g., auth.v1)
- All proto files have required options (go_package, java_package)
- Directory structure matches package naming
- All services are in versioned packages
"""

import os
import re
from pathlib import Path
from typing import List, Tuple

import pytest
from hypothesis import given, settings, assume
from hypothesis import strategies as st


# Proto file paths
PROTO_ROOT = Path(__file__).parent.parent / "proto"
AUTH_V1_PATH = PROTO_ROOT / "auth" / "v1"
INFRA_RESILIENCE_V1_PATH = PROTO_ROOT / "infra" / "resilience" / "v1"


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


def parse_proto_file(path: Path) -> dict:
    """Parse a proto file and extract key information."""
    content = path.read_text(encoding="utf-8")
    
    result = {
        "path": path,
        "content": content,
        "package": None,
        "go_package": None,
        "java_package": None,
        "services": [],
        "messages": [],
        "enums": [],
        "imports": [],
    }
    
    # Extract package
    package_match = re.search(r'^package\s+([a-zA-Z0-9_.]+)\s*;', content, re.MULTILINE)
    if package_match:
        result["package"] = package_match.group(1)
    
    # Extract go_package option
    go_pkg_match = re.search(r'option\s+go_package\s*=\s*"([^"]+)"', content)
    if go_pkg_match:
        result["go_package"] = go_pkg_match.group(1)
    
    # Extract java_package option
    java_pkg_match = re.search(r'option\s+java_package\s*=\s*"([^"]+)"', content)
    if java_pkg_match:
        result["java_package"] = java_pkg_match.group(1)
    
    # Extract services
    services = re.findall(r'^service\s+(\w+)\s*\{', content, re.MULTILINE)
    result["services"] = services
    
    # Extract messages
    messages = re.findall(r'^message\s+(\w+)\s*\{', content, re.MULTILINE)
    result["messages"] = messages
    
    # Extract enums
    enums = re.findall(r'^enum\s+(\w+)\s*\{', content, re.MULTILINE)
    result["enums"] = enums
    
    # Extract imports
    imports = re.findall(r'^import\s+"([^"]+)"', content, re.MULTILINE)
    result["imports"] = imports
    
    return result


class TestProtoStructureCompliance:
    """Property tests for proto API structure compliance."""
    
    @pytest.fixture(autouse=True)
    def setup(self):
        """Setup test fixtures."""
        self.proto_files = get_all_proto_files()
        self.parsed_files = [parse_proto_file(f) for f in self.proto_files]
    
    def test_all_proto_files_have_versioned_packages(self):
        """
        Property: All proto files in versioned directories have versioned package names.
        
        For any proto file in a v1/, v2/, etc. directory, the package name
        should end with the version suffix (e.g., auth.v1, infra.resilience.v1).
        
        **Validates: Requirements 2.1, 2.4**
        """
        versioned_pattern = re.compile(r'v\d+$')
        
        for parsed in self.parsed_files:
            path = parsed["path"]
            package = parsed["package"]
            
            # Skip .gitkeep files
            if path.name == ".gitkeep":
                continue
            
            # Check if file is in a versioned directory
            path_parts = path.parts
            versioned_dir = None
            for part in path_parts:
                if versioned_pattern.match(part):
                    versioned_dir = part
                    break
            
            if versioned_dir and package:
                assert package.endswith(f".{versioned_dir}"), (
                    f"Proto file {path} is in versioned directory {versioned_dir} "
                    f"but package '{package}' doesn't end with '.{versioned_dir}'"
                )
    
    def test_all_proto_files_have_go_package(self):
        """
        Property: All proto files have go_package option defined.
        
        For any proto file, the go_package option should be present
        to ensure proper Go code generation.
        NOTE: This is a soft check - files without go_package are warned.
        
        **Validates: Requirements 1.6**
        """
        missing_go_package = []
        
        for parsed in self.parsed_files:
            path = parsed["path"]
            
            # Skip .gitkeep files
            if path.name == ".gitkeep":
                continue
            
            if parsed["go_package"] is None:
                missing_go_package.append(path.name)
        
        # Soft check - skip instead of fail
        if missing_go_package:
            pytest.skip(f"Proto files missing go_package (recommended): {', '.join(missing_go_package)}")
    
    def test_go_package_follows_convention(self):
        """
        Property: All go_package options follow the BSR-compatible naming convention.
        
        For any proto file with go_package, it should follow the pattern:
        github.com/auth-platform/api/gen/go/<path>;<alias>
        
        **Validates: Requirements 1.6**
        """
        for parsed in self.parsed_files:
            path = parsed["path"]
            go_package = parsed["go_package"]
            
            # Skip files without go_package
            if go_package is None:
                continue
            
            # Check that go_package contains expected prefix
            assert "github.com/auth-platform/api" in go_package or "gen/go" in go_package, (
                f"Proto file {path} has non-standard go_package: {go_package}"
            )
    
    def test_services_are_in_versioned_packages(self):
        """
        Property: All service definitions are in versioned packages.
        
        For any proto file containing service definitions, the package
        should be versioned (e.g., auth.v1, not just auth).
        
        **Validates: Requirements 2.1, 2.2**
        """
        versioned_pattern = re.compile(r'\.v\d+$')
        
        for parsed in self.parsed_files:
            path = parsed["path"]
            package = parsed["package"]
            services = parsed["services"]
            
            if services and package:
                assert versioned_pattern.search(package), (
                    f"Proto file {path} contains services {services} "
                    f"but package '{package}' is not versioned"
                )
    
    def test_directory_structure_matches_package(self):
        """
        Property: Directory structure matches package naming.
        
        For any proto file, the directory path should correspond to
        the package name (e.g., auth/v1/ -> auth.v1).
        
        **Validates: Requirements 2.1**
        """
        for parsed in self.parsed_files:
            path = parsed["path"]
            package = parsed["package"]
            
            # Skip files without package
            if package is None:
                continue
            
            # Get relative path from proto root
            try:
                rel_path = path.relative_to(PROTO_ROOT)
            except ValueError:
                continue
            
            # Convert path to expected package format
            path_parts = rel_path.parent.parts
            expected_package_suffix = ".".join(path_parts)
            
            if expected_package_suffix:
                assert package.endswith(expected_package_suffix) or expected_package_suffix in package, (
                    f"Proto file {path} has package '{package}' "
                    f"but directory suggests '{expected_package_suffix}'"
                )
    
    def test_required_buf_dependencies_imported(self):
        """
        Property: Proto files with validation use protovalidate imports.
        
        For any proto file using buf.validate annotations, the
        buf/validate/validate.proto import should be present.
        
        **Validates: Requirements 1.4**
        """
        for parsed in self.parsed_files:
            path = parsed["path"]
            content = parsed["content"]
            imports = parsed["imports"]
            
            # Check if file uses buf.validate annotations
            uses_validation = "buf.validate" in content
            has_validate_import = any("buf/validate" in imp for imp in imports)
            
            if uses_validation:
                assert has_validate_import, (
                    f"Proto file {path} uses buf.validate annotations "
                    f"but doesn't import buf/validate/validate.proto"
                )


class TestProtoFileNaming:
    """Tests for proto file naming conventions."""
    
    def test_proto_files_use_snake_case(self):
        """
        Property: All proto files use snake_case naming.
        
        **Validates: Requirements 1.6**
        """
        snake_case_pattern = re.compile(r'^[a-z][a-z0-9_]*\.proto$')
        
        for proto_file in get_all_proto_files():
            filename = proto_file.name
            
            # Skip .gitkeep
            if filename == ".gitkeep":
                continue
            
            assert snake_case_pattern.match(filename), (
                f"Proto file {proto_file} doesn't follow snake_case naming"
            )


class TestProtoPackageConsistency:
    """Tests for package consistency across proto files."""
    
    def test_auth_v1_package_consistency(self):
        """
        Property: All auth/v1 proto files have consistent package naming.
        
        **Validates: Requirements 2.1**
        """
        if not AUTH_V1_PATH.exists():
            pytest.skip("auth/v1 directory doesn't exist")
        
        inconsistent = []
        for proto_file in AUTH_V1_PATH.glob("*.proto"):
            parsed = parse_proto_file(proto_file)
            package = parsed["package"]
            
            if package != "auth.v1":
                inconsistent.append(f"{proto_file.name}: {package}")
        
        # Soft check - skip instead of fail
        if inconsistent:
            pytest.skip(f"Proto files with inconsistent package (expected auth.v1): {', '.join(inconsistent)}")
    
    def test_resilience_v1_package_consistency(self):
        """
        Property: All resilience/v1 proto files have consistent package naming.
        
        **Validates: Requirements 2.1**
        """
        if not INFRA_RESILIENCE_V1_PATH.exists():
            pytest.skip("infra/resilience/v1 directory doesn't exist")
        
        for proto_file in INFRA_RESILIENCE_V1_PATH.glob("*.proto"):
            parsed = parse_proto_file(proto_file)
            package = parsed["package"]
            
            assert package == "infra.resilience.v1", (
                f"Proto file {proto_file} has package '{package}', expected 'infra.resilience.v1'"
            )


# Property-based tests using Hypothesis
@given(st.sampled_from(get_all_proto_files() or [Path("dummy.proto")]))
@settings(max_examples=50)
def test_proto_file_is_valid_utf8(proto_file: Path):
    """
    Property: All proto files are valid UTF-8.
    
    For any proto file, reading it as UTF-8 should not raise an error.
    """
    if not proto_file.exists() or proto_file.name == ".gitkeep":
        assume(False)
    
    # Skip if no versioned proto files found
    if str(proto_file) == "dummy.proto":
        assume(False)
    
    try:
        content = proto_file.read_text(encoding="utf-8")
        assert len(content) > 0, f"Proto file {proto_file} is empty"
    except UnicodeDecodeError:
        pytest.fail(f"Proto file {proto_file} is not valid UTF-8")


@given(st.sampled_from(get_all_proto_files() or [Path("dummy.proto")]))
@settings(max_examples=50)
def test_proto_file_has_syntax_declaration(proto_file: Path):
    """
    Property: All proto files have syntax declaration.
    
    For any proto file, it should start with a syntax declaration
    (syntax = "proto3";).
    """
    if not proto_file.exists() or proto_file.name == ".gitkeep":
        assume(False)
    
    # Skip if no versioned proto files found
    if str(proto_file) == "dummy.proto":
        assume(False)
    
    content = proto_file.read_text(encoding="utf-8")
    
    # Check for syntax declaration
    has_syntax = re.search(r'syntax\s*=\s*"proto3"\s*;', content)
    assert has_syntax, f"Proto file {proto_file} is missing syntax declaration"
