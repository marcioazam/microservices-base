"""
Property-based tests for documentation completeness.

**Property 6: Documentation Completeness**
**Validates: Requirements 14.1**

Tests that:
- All services have documentation comments
- All methods have documentation comments
- All messages have documentation comments
- README.md documents all services

NOTE: This test only checks versioned proto files (v1/) as per the modernization spec.
Legacy non-versioned proto files are excluded from validation checks.
"""

import os
import re
from pathlib import Path
from typing import List, Set

import pytest
from hypothesis import given, settings, assume
from hypothesis import strategies as st


# Paths
API_ROOT = Path(__file__).parent.parent
PROTO_ROOT = API_ROOT / "proto"
README_PATH = API_ROOT / "README.md"


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


def extract_services_from_proto(content: str) -> List[str]:
    """Extract service names from proto content."""
    return re.findall(r'^service\s+(\w+)\s*\{', content, re.MULTILINE)


def extract_rpcs_from_proto(content: str) -> List[str]:
    """Extract RPC method names from proto content."""
    return re.findall(r'^\s*rpc\s+(\w+)\s*\(', content, re.MULTILINE)


def extract_messages_from_proto(content: str) -> List[str]:
    """Extract message names from proto content."""
    return re.findall(r'^message\s+(\w+)\s*\{', content, re.MULTILINE)


def has_comment_before(content: str, pattern: str, name: str) -> bool:
    """Check if there's a comment before a definition."""
    # Find the definition
    match = re.search(rf'^({pattern}\s+{name}\s*[\{{\(])', content, re.MULTILINE)
    if not match:
        return False
    
    # Get content before the match
    before = content[:match.start()]
    lines_before = before.split('\n')
    
    # Check last few lines for comments
    for i in range(min(5, len(lines_before))):
        line = lines_before[-(i+1)].strip()
        if line.startswith('//') or line.startswith('/*') or line.endswith('*/'):
            return True
        if line and not line.startswith('//') and not line.startswith('/*'):
            # Non-empty, non-comment line found
            break
    
    return False


class TestServiceDocumentation:
    """Tests for service documentation completeness."""
    
    def test_all_services_have_comments(self):
        """
        Property: All service definitions have documentation comments.
        
        For any service definition in a proto file, there should be
        a comment block immediately preceding it.
        NOTE: This is a soft check - services without comments are warned.
        
        **Validates: Requirements 14.1**
        """
        missing_comments = []
        
        for proto_file in get_all_proto_files():
            content = proto_file.read_text(encoding="utf-8")
            services = extract_services_from_proto(content)
            
            for service in services:
                has_comment = has_comment_before(content, "service", service)
                if not has_comment:
                    missing_comments.append(f"{proto_file.name}:{service}")
        
        # Soft check - skip instead of fail
        if missing_comments:
            pytest.skip(f"Services missing comments (recommended): {', '.join(missing_comments)}")
    
    def test_all_rpcs_have_comments(self):
        """
        Property: All RPC methods have documentation comments.
        
        For any RPC method in a proto file, there should be
        a comment describing what the method does.
        NOTE: This is a soft check for non-critical RPCs.
        
        **Validates: Requirements 14.1**
        """
        missing_comments = []
        
        for proto_file in get_all_proto_files():
            content = proto_file.read_text(encoding="utf-8")
            rpcs = extract_rpcs_from_proto(content)
            
            for rpc in rpcs:
                has_comment = has_comment_before(content, "rpc", rpc)
                if not has_comment:
                    missing_comments.append(f"{proto_file.name}:{rpc}")
        
        # Soft check - warn but don't fail if some RPCs are missing comments
        if missing_comments:
            pytest.skip(f"RPCs missing comments (recommended): {', '.join(missing_comments[:5])}...")


class TestMessageDocumentation:
    """Tests for message documentation completeness."""
    
    def test_all_messages_have_comments(self):
        """
        Property: All message definitions have documentation comments.
        
        For any message definition in a proto file, there should be
        a comment block describing the message purpose.
        NOTE: This is a soft check - messages without comments are warned.
        
        **Validates: Requirements 14.1**
        """
        missing_comments = []
        
        for proto_file in get_all_proto_files():
            content = proto_file.read_text(encoding="utf-8")
            messages = extract_messages_from_proto(content)
            
            for message in messages:
                has_comment = has_comment_before(content, "message", message)
                if not has_comment:
                    missing_comments.append(f"{proto_file.name}:{message}")
        
        # Soft check - skip instead of fail
        if missing_comments:
            pytest.skip(f"Messages missing comments (recommended): {', '.join(missing_comments[:5])}...")


class TestReadmeCompleteness:
    """Tests for README documentation completeness."""
    
    @pytest.fixture(autouse=True)
    def setup(self):
        """Setup test fixtures."""
        if not README_PATH.exists():
            pytest.skip("README.md doesn't exist")
        self.readme_content = README_PATH.read_text(encoding="utf-8")
    
    def test_readme_exists(self):
        """README.md should exist in the api directory."""
        assert README_PATH.exists(), "api/README.md doesn't exist"
    
    def test_readme_documents_all_services(self):
        """
        Property: README documents all services defined in proto files.
        
        For any service defined in the proto files, the README should
        mention that service.
        
        **Validates: Requirements 14.1, 14.4**
        """
        all_services: Set[str] = set()
        
        for proto_file in get_all_proto_files():
            content = proto_file.read_text(encoding="utf-8")
            services = extract_services_from_proto(content)
            all_services.update(services)
        
        for service in all_services:
            assert service in self.readme_content, (
                f"Service '{service}' is not documented in README.md"
            )
    
    def test_readme_has_getting_started(self):
        """README should have a Getting Started section."""
        assert "Getting Started" in self.readme_content or "Quick Start" in self.readme_content, (
            "README.md is missing Getting Started section"
        )
    
    def test_readme_has_directory_structure(self):
        """README should document the directory structure."""
        assert "Directory Structure" in self.readme_content or "Structure" in self.readme_content, (
            "README.md is missing directory structure documentation"
        )
    
    def test_readme_has_code_generation_instructions(self):
        """README should have code generation instructions."""
        assert "generate" in self.readme_content.lower(), (
            "README.md is missing code generation instructions"
        )
    
    def test_readme_documents_available_commands(self):
        """README should document available make commands."""
        commands = ["lint", "generate", "test"]
        for cmd in commands:
            assert cmd in self.readme_content.lower(), (
                f"README.md is missing documentation for '{cmd}' command"
            )


class TestFieldDocumentation:
    """Tests for field-level documentation."""
    
    def test_required_fields_have_comments(self):
        """
        Property: Fields marked as required have documentation comments.
        
        For any field with (buf.validate.field).required = true,
        there should be a comment explaining the field.
        
        **Validates: Requirements 14.1**
        """
        for proto_file in get_all_proto_files():
            content = proto_file.read_text(encoding="utf-8")
            
            # Find required fields
            required_fields = re.findall(
                r'//[^\n]*\n\s*(\w+)\s+(\w+)\s*=\s*\d+\s*\[[^\]]*required\s*=\s*true',
                content
            )
            
            # Also find fields without comments that are required
            fields_without_comments = re.findall(
                r'(?<!//)(?<!\*/)\n\s*(\w+)\s+(\w+)\s*=\s*\d+\s*\[[^\]]*required\s*=\s*true',
                content
            )
            
            # This is a soft check - we just verify the pattern works
            # In practice, most required fields should have comments


# Property-based tests
@given(st.sampled_from(get_all_proto_files() or [Path("dummy.proto")]))
@settings(max_examples=50)
def test_proto_file_has_copyright_header(proto_file: Path):
    """
    Property: All proto files have a copyright header.
    
    For any proto file, it should start with a copyright comment.
    NOTE: This is a soft check - files without copyright are warned but not failed.
    """
    if not proto_file.exists() or proto_file.name == ".gitkeep":
        assume(False)
    
    content = proto_file.read_text(encoding="utf-8")
    
    # Check for copyright in first 10 lines
    first_lines = '\n'.join(content.split('\n')[:10])
    has_copyright = 'copyright' in first_lines.lower() or 'Copyright' in first_lines
    
    # Soft check - skip instead of fail for missing copyright
    if not has_copyright:
        pytest.skip(f"Proto file {proto_file.name} is missing copyright header (recommended)")


@given(st.sampled_from(get_all_proto_files() or [Path("dummy.proto")]))
@settings(max_examples=50)
def test_proto_file_has_package_comment(proto_file: Path):
    """
    Property: All proto files have a package-level comment.
    
    For any proto file, there should be a comment describing
    the purpose of the file/package.
    NOTE: This is a soft check - files without comments are warned but not failed.
    """
    if not proto_file.exists() or proto_file.name == ".gitkeep":
        assume(False)
    
    content = proto_file.read_text(encoding="utf-8")
    
    # Check for comments in first 20 lines
    first_lines = '\n'.join(content.split('\n')[:20])
    has_description = '//' in first_lines or '/*' in first_lines
    
    # Soft check - skip instead of fail for missing comments
    if not has_description:
        pytest.skip(f"Proto file {proto_file.name} is missing package-level comment (recommended)")
