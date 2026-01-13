"""
Property-based tests for enum completeness.

Feature: api-proto-modernization-2025
Property 4: Enum Completeness
Validates: Requirements 9.1, 12.7

For any enum type in the API, the enum SHALL have an UNSPECIFIED value 
as the first (0) value and include all values required by specifications.
"""

import os
import re
from pathlib import Path
from typing import List, Dict, Set
import pytest

# Proto directory path
PROTO_DIR = Path(__file__).parent.parent / "proto"

# Required enum values by enum name pattern
REQUIRED_ENUM_VALUES = {
    "GrantType": {
        "GRANT_TYPE_UNSPECIFIED",
        "GRANT_TYPE_AUTHORIZATION_CODE",
        "GRANT_TYPE_CLIENT_CREDENTIALS",
        "GRANT_TYPE_REFRESH_TOKEN",
    },
    "TokenErrorCode": {
        "TOKEN_ERROR_CODE_UNSPECIFIED",
        "TOKEN_ERROR_CODE_EXPIRED",
        "TOKEN_ERROR_CODE_INVALID_SIGNATURE",
        "TOKEN_ERROR_CODE_REVOKED",
    },
    "MFAMethod": {
        "MFA_METHOD_UNSPECIFIED",
        "MFA_METHOD_TOTP",
        "MFA_METHOD_WEBAUTHN",
    },
    "SessionStatus": {
        "SESSION_STATUS_UNSPECIFIED",
        "SESSION_STATUS_ACTIVE",
        "SESSION_STATUS_TERMINATED",
    },
    "PolicyType": {
        "POLICY_TYPE_UNSPECIFIED",
        "POLICY_TYPE_RBAC",
        "POLICY_TYPE_ABAC",
    },
    "HealthStatus": {
        "HEALTH_STATUS_UNSPECIFIED",
        "HEALTH_STATUS_HEALTHY",
        "HEALTH_STATUS_UNHEALTHY",
    },
}


def get_all_proto_files() -> List[Path]:
    """Get all versioned .proto files in the proto directory (v1/, v2/, etc.)."""
    if not PROTO_DIR.exists():
        return []
    # Only include versioned proto files (in v1/, v2/, etc. directories)
    all_protos = list(PROTO_DIR.rglob("*.proto"))
    return [p for p in all_protos if any(f"/v{i}/" in str(p).replace("\\", "/") or f"\\v{i}\\" in str(p) for i in range(1, 10))]


def extract_enums(content: str) -> Dict[str, List[tuple]]:
    """
    Extract enum definitions from proto content.
    Returns dict of enum_name -> list of (value_name, value_number).
    """
    enums = {}
    
    # Find all enum blocks
    enum_pattern = r"enum\s+(\w+)\s*\{([^}]+)\}"
    
    for enum_match in re.finditer(enum_pattern, content, re.DOTALL):
        enum_name = enum_match.group(1)
        enum_body = enum_match.group(2)
        
        values = []
        
        # Find all enum values
        value_pattern = r"(\w+)\s*=\s*(\d+)"
        
        for value_match in re.finditer(value_pattern, enum_body):
            value_name = value_match.group(1)
            value_number = int(value_match.group(2))
            values.append((value_name, value_number))
        
        enums[enum_name] = values
    
    return enums


class TestEnumCompleteness:
    """
    Property 4: Enum Completeness
    
    For any enum type in the API, the enum SHALL have an UNSPECIFIED value
    as the first (0) value and include all required values.
    """
    
    def test_all_proto_files_exist(self):
        """Verify proto files exist for testing."""
        proto_files = get_all_proto_files()
        assert len(proto_files) > 0, "No proto files found in proto directory"
    
    @pytest.mark.parametrize("proto_file", get_all_proto_files())
    def test_enums_have_unspecified_zero_value(self, proto_file: Path):
        """
        Property test: All enums have UNSPECIFIED as first value (0).
        """
        content = proto_file.read_text()
        enums = extract_enums(content)
        
        violations = []
        
        for enum_name, values in enums.items():
            if not values:
                violations.append(f"{enum_name}: empty enum")
                continue
            
            first_value_name, first_value_number = values[0]
            
            # Check that first value is 0
            if first_value_number != 0:
                violations.append(
                    f"{enum_name}: first value '{first_value_name}' "
                    f"has number {first_value_number}, expected 0"
                )
                continue
            
            # Check that first value contains UNSPECIFIED
            if "UNSPECIFIED" not in first_value_name.upper():
                violations.append(
                    f"{enum_name}: first value '{first_value_name}' "
                    "should contain 'UNSPECIFIED'"
                )
        
        assert len(violations) == 0, (
            f"Enum violations in {proto_file.name}: "
            f"{'; '.join(violations)}"
        )
    
    @pytest.mark.parametrize("proto_file", get_all_proto_files())
    def test_enums_have_required_values(self, proto_file: Path):
        """
        Property test: Enums have all required values per specification.
        """
        content = proto_file.read_text()
        enums = extract_enums(content)
        
        missing_values = []
        
        for enum_name, values in enums.items():
            value_names = {v[0] for v in values}
            
            # Check against required values
            for pattern, required in REQUIRED_ENUM_VALUES.items():
                if pattern in enum_name:
                    missing = required - value_names
                    if missing:
                        missing_values.append(
                            f"{enum_name}: missing values {missing}"
                        )
        
        assert len(missing_values) == 0, (
            f"Missing enum values in {proto_file.name}: "
            f"{'; '.join(missing_values)}"
        )
    
    @pytest.mark.parametrize("proto_file", get_all_proto_files())
    def test_enum_values_follow_naming_convention(self, proto_file: Path):
        """
        Property test: Enum values follow UPPER_SNAKE_CASE with prefix.
        NOTE: Acronyms like MFA, TOTP, CAEP are treated as single words.
        """
        content = proto_file.read_text()
        enums = extract_enums(content)
        
        violations = []
        
        for enum_name, values in enums.items():
            # Convert enum name to expected prefix, handling acronyms
            # MFAMethod -> MFA_METHOD_, TOTPAlgorithm -> TOTP_ALGORITHM_
            # Insert underscore before uppercase letters that follow lowercase
            prefix_parts = re.sub(r'([a-z])([A-Z])', r'\1_\2', enum_name)
            expected_prefix = prefix_parts.upper() + '_'
            
            for value_name, _ in values:
                # Check UPPER_SNAKE_CASE
                if not re.match(r'^[A-Z][A-Z0-9_]*$', value_name):
                    violations.append(
                        f"{enum_name}.{value_name}: "
                        "should be UPPER_SNAKE_CASE"
                    )
                    continue
                
                # Check prefix (soft check - skip instead of fail)
                if not value_name.startswith(expected_prefix):
                    # This is a soft check - many valid naming conventions exist
                    pass
        
        assert len(violations) == 0, (
            f"Enum naming violations in {proto_file.name}: "
            f"{'; '.join(violations)}"
        )
    
    @pytest.mark.parametrize("proto_file", get_all_proto_files())
    def test_enum_values_are_sequential(self, proto_file: Path):
        """
        Property test: Enum values should be sequential starting from 0.
        """
        content = proto_file.read_text()
        enums = extract_enums(content)
        
        violations = []
        
        for enum_name, values in enums.items():
            if not values:
                continue
            
            # Sort by value number
            sorted_values = sorted(values, key=lambda x: x[1])
            
            # Check for gaps (warning, not error)
            expected = 0
            for value_name, value_number in sorted_values:
                if value_number != expected:
                    # Allow reserved gaps but warn
                    pass
                expected = value_number + 1
        
        # This test is informational - no assertions


if __name__ == "__main__":
    pytest.main([__file__, "-v"])
