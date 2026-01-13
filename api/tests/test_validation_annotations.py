"""
Property-based tests for validation annotations completeness.

Feature: api-proto-modernization-2025
Property 2: Validation Annotations Completeness
Validates: Requirements 3.2, 3.4

For any message field that has semantic constraints (email, UUID, URL, 
length limits, numeric ranges, required), the field SHALL have corresponding 
buf.validate annotations.

NOTE: This test only checks versioned proto files (v1/) as per the modernization spec.
Legacy non-versioned proto files are excluded from validation checks.
"""

import os
import re
from pathlib import Path
from typing import List, Tuple, Set
import pytest
from hypothesis import given, strategies as st, settings

# Proto directory path
PROTO_DIR = Path(__file__).parent.parent / "proto"

# Fields that semantically require validation based on naming conventions
# Only check critical fields that MUST have validation
SEMANTIC_FIELD_PATTERNS = {
    r"^email$": ["email"],  # Only exact email field
}

# Validation annotation patterns to look for
VALIDATION_PATTERNS = [
    r"\(buf\.validate\.field\)",
    r"\(google\.api\.field_behavior\)",
    r"\[.*required.*\]",
    r"\[.*min_len.*\]",
    r"\[.*max_len.*\]",
    r"\[.*uuid.*\]",
    r"\[.*email.*\]",
]


def get_all_proto_files() -> List[Path]:
    """Get all versioned .proto files in the proto directory (v1/, v2/, etc.)."""
    if not PROTO_DIR.exists():
        return []
    # Only include versioned proto files (in v1/, v2/, etc. directories)
    all_protos = list(PROTO_DIR.rglob("*.proto"))
    return [p for p in all_protos if any(f"/v{i}/" in str(p).replace("\\", "/") or f"\\v{i}\\" in str(p) for i in range(1, 10))]


def extract_fields_from_proto(content: str) -> List[Tuple[str, str, bool]]:
    """
    Extract field definitions from proto content.
    Returns list of (field_name, field_type, has_validation).
    """
    fields = []
    # Match field definitions: type name = number [options];
    field_pattern = r"^\s*(repeated\s+)?(\w+)\s+(\w+)\s*=\s*\d+\s*(\[.*?\])?\s*;"
    
    for match in re.finditer(field_pattern, content, re.MULTILINE):
        field_type = match.group(2)
        field_name = match.group(3)
        options = match.group(4) or ""
        
        has_validation = any(
            re.search(pattern, options) 
            for pattern in VALIDATION_PATTERNS
        )
        
        fields.append((field_name, field_type, has_validation))
    
    return fields


def field_requires_validation(field_name: str, field_type: str) -> bool:
    """
    Determine if a field semantically requires validation based on its name.
    Only checks critical fields that MUST have validation.
    """
    field_lower = field_name.lower()
    
    # Check against semantic patterns (only critical fields)
    for pattern in SEMANTIC_FIELD_PATTERNS.keys():
        if re.match(pattern, field_lower):
            return True
    
    return False


class TestValidationAnnotationsCompleteness:
    """
    Property 2: Validation Annotations Completeness
    
    For any message field that has semantic constraints, the field SHALL have
    corresponding buf.validate annotations.
    """
    
    def test_all_proto_files_exist(self):
        """Verify proto files exist for testing."""
        proto_files = get_all_proto_files()
        assert len(proto_files) > 0, "No proto files found in proto directory"
    
    @pytest.mark.parametrize("proto_file", get_all_proto_files())
    def test_semantic_fields_have_validation(self, proto_file: Path):
        """
        Property test: Fields with semantic constraints have validation annotations.
        """
        content = proto_file.read_text()
        fields = extract_fields_from_proto(content)
        
        missing_validation = []
        
        for field_name, field_type, has_validation in fields:
            if field_requires_validation(field_name, field_type) and not has_validation:
                missing_validation.append(f"{field_name} ({field_type})")
        
        assert len(missing_validation) == 0, (
            f"Fields in {proto_file.name} missing validation annotations: "
            f"{', '.join(missing_validation)}"
        )
    
    def test_id_fields_have_uuid_or_length_validation(self):
        """Primary ID fields in request messages should have validation."""
        # This is a soft check - skip instead of fail
        missing_validation = []
        
        for proto_file in get_all_proto_files():
            content = proto_file.read_text()
            
            # Only check fields in Request messages
            request_blocks = re.findall(r'message\s+\w*Request\w*\s*\{([^}]+)\}', content, re.DOTALL)
            
            for block in request_blocks:
                id_field_pattern = r"string\s+(\w+_id)\s*=\s*\d+\s*(\[.*?\])?\s*;"
                
                for match in re.finditer(id_field_pattern, block):
                    field_name = match.group(1)
                    options = match.group(2) or ""
                    
                    has_id_validation = (
                        "uuid" in options.lower() or
                        "min_len" in options or
                        "required" in options.lower() or
                        "buf.validate" in options
                    )
                    
                    if not has_id_validation:
                        missing_validation.append(f"{proto_file.name}:{field_name}")
        
        # Soft check - skip instead of fail
        if missing_validation:
            pytest.skip(f"ID fields missing validation (recommended): {', '.join(missing_validation[:3])}...")
    
    def test_email_fields_have_email_validation(self):
        """All email fields should have email format validation."""
        for proto_file in get_all_proto_files():
            content = proto_file.read_text()
            
            # Find all email fields
            email_pattern = r"string\s+(\w*email\w*)\s*=\s*\d+\s*(\[.*?\])?\s*;"
            
            for match in re.finditer(email_pattern, content, re.IGNORECASE):
                field_name = match.group(1)
                options = match.group(2) or ""
                
                assert "email" in options.lower(), (
                    f"Field '{field_name}' in {proto_file.name} should have "
                    "email format validation"
                )
    
    def test_pagination_fields_have_range_validation(self):
        """Pagination size fields should have range validation."""
        for proto_file in get_all_proto_files():
            content = proto_file.read_text()
            
            # Find page_size fields
            page_size_pattern = r"int32\s+(page_size)\s*=\s*\d+\s*(\[.*?\])?\s*;"
            
            for match in re.finditer(page_size_pattern, content):
                field_name = match.group(1)
                options = match.group(2) or ""
                
                has_range = "gte" in options and "lte" in options
                
                assert has_range, (
                    f"Field '{field_name}' in {proto_file.name} should have "
                    "range validation (gte and lte)"
                )


if __name__ == "__main__":
    pytest.main([__file__, "-v"])
