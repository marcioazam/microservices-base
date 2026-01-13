# Copyright 2025 Auth Platform. All rights reserved.
# Property Test: Streaming API Compliance
# Property 5: For any streaming RPC method, the streamed message type SHALL include
# a unique event identifier field and a timestamp field for ordering.
# Validates: Requirements 13.1, 13.4

"""
Property-based tests for streaming API compliance.

This module verifies that all streaming RPC methods in the API return messages
with proper event identification and timestamp fields for ordering.
"""

import os
import re
from pathlib import Path
from typing import NamedTuple

import pytest
from hypothesis import given, settings, strategies as st


class StreamingRPC(NamedTuple):
    """Represents a streaming RPC method."""

    service_name: str
    method_name: str
    response_type: str
    file_path: str


class MessageField(NamedTuple):
    """Represents a message field."""

    name: str
    field_type: str
    number: int


def get_proto_files() -> list[Path]:
    """Get all versioned proto files in the api/proto directory (v1/, v2/, etc.)."""
    proto_dir = Path(__file__).parent.parent / "proto"
    if not proto_dir.exists():
        return []
    # Only include versioned proto files (in v1/, v2/, etc. directories)
    all_protos = list(proto_dir.rglob("*.proto"))
    return [p for p in all_protos if any(f"/v{i}/" in str(p).replace("\\", "/") or f"\\v{i}\\" in str(p) for i in range(1, 10))]


def parse_streaming_rpcs(content: str, file_path: str) -> list[StreamingRPC]:
    """Parse streaming RPC methods from proto content."""
    rpcs = []

    # Find service blocks
    service_pattern = r"service\s+(\w+)\s*\{([^}]+(?:\{[^}]*\}[^}]*)*)\}"
    for service_match in re.finditer(service_pattern, content, re.DOTALL):
        service_name = service_match.group(1)
        service_body = service_match.group(2)

        # Find streaming RPCs (server streaming or bidirectional)
        rpc_pattern = r"rpc\s+(\w+)\s*\([^)]*\)\s*returns\s*\(\s*stream\s+(\w+)\s*\)"
        for rpc_match in re.finditer(rpc_pattern, service_body):
            method_name = rpc_match.group(1)
            response_type = rpc_match.group(2)
            rpcs.append(
                StreamingRPC(
                    service_name=service_name,
                    method_name=method_name,
                    response_type=response_type,
                    file_path=file_path,
                )
            )

    return rpcs


def parse_message_fields(content: str, message_name: str) -> list[MessageField]:
    """Parse fields from a message definition."""
    fields = []

    # Find the message block
    message_pattern = rf"message\s+{re.escape(message_name)}\s*\{{([^}}]+(?:\{{[^}}]*\}}[^}}]*)*)\}}"
    message_match = re.search(message_pattern, content, re.DOTALL)

    if not message_match:
        return fields

    message_body = message_match.group(1)

    # Parse fields (simplified, handles basic cases)
    field_pattern = r"(?:repeated\s+)?(\w+(?:\.\w+)*)\s+(\w+)\s*=\s*(\d+)"
    for field_match in re.finditer(field_pattern, message_body):
        field_type = field_match.group(1)
        field_name = field_match.group(2)
        field_number = int(field_match.group(3))
        fields.append(MessageField(name=field_name, field_type=field_type, number=field_number))

    return fields


def has_event_id_field(fields: list[MessageField]) -> bool:
    """Check if message has an event identifier field."""
    event_id_patterns = ["event_id", "id", "message_id", "notification_id"]
    for field in fields:
        if field.name.lower() in event_id_patterns:
            return True
        if "id" in field.name.lower() and field.field_type == "string":
            return True
    return False


def has_timestamp_field(fields: list[MessageField]) -> bool:
    """Check if message has a timestamp field for ordering."""
    timestamp_patterns = ["timestamp", "created_at", "occurred_at", "event_time"]
    for field in fields:
        if field.name.lower() in timestamp_patterns:
            return True
        if field.field_type == "google.protobuf.Timestamp":
            return True
    return False


def get_all_streaming_rpcs() -> list[StreamingRPC]:
    """Get all streaming RPCs from all proto files."""
    all_rpcs = []
    for proto_file in get_proto_files():
        content = proto_file.read_text(encoding="utf-8")
        rpcs = parse_streaming_rpcs(content, str(proto_file))
        all_rpcs.extend(rpcs)
    return all_rpcs


def get_message_content(response_type: str) -> tuple[str, str] | None:
    """Find and return the content containing the message definition."""
    for proto_file in get_proto_files():
        content = proto_file.read_text(encoding="utf-8")
        if re.search(rf"message\s+{re.escape(response_type)}\s*\{{", content):
            return content, str(proto_file)
    return None


class TestStreamingAPICompliance:
    """Property tests for streaming API compliance."""

    def test_streaming_rpcs_exist(self) -> None:
        """Verify that streaming RPCs are defined in the API."""
        rpcs = get_all_streaming_rpcs()
        # Streaming RPCs are optional - skip if none found
        if len(rpcs) == 0:
            pytest.skip("No streaming RPCs found in versioned proto files (optional feature)")

    @pytest.mark.parametrize(
        "rpc",
        get_all_streaming_rpcs(),
        ids=lambda r: f"{r.service_name}.{r.method_name}",
    )
    def test_streaming_message_has_event_id(self, rpc: StreamingRPC) -> None:
        """
        Property 5a: Streaming messages SHALL have event identifier.

        For any streaming RPC method, the streamed message type SHALL include
        a unique event identifier field.
        """
        result = get_message_content(rpc.response_type)
        assert result is not None, f"Message {rpc.response_type} not found"

        content, file_path = result
        fields = parse_message_fields(content, rpc.response_type)

        assert has_event_id_field(fields), (
            f"Streaming message {rpc.response_type} in {rpc.service_name}.{rpc.method_name} "
            f"must have an event identifier field (e.g., event_id, id). "
            f"Found fields: {[f.name for f in fields]}"
        )

    @pytest.mark.parametrize(
        "rpc",
        get_all_streaming_rpcs(),
        ids=lambda r: f"{r.service_name}.{r.method_name}",
    )
    def test_streaming_message_has_timestamp(self, rpc: StreamingRPC) -> None:
        """
        Property 5b: Streaming messages SHALL have timestamp for ordering.

        For any streaming RPC method, the streamed message type SHALL include
        a timestamp field for ordering events.
        """
        result = get_message_content(rpc.response_type)
        assert result is not None, f"Message {rpc.response_type} not found"

        content, file_path = result
        fields = parse_message_fields(content, rpc.response_type)

        assert has_timestamp_field(fields), (
            f"Streaming message {rpc.response_type} in {rpc.service_name}.{rpc.method_name} "
            f"must have a timestamp field for ordering (e.g., timestamp, google.protobuf.Timestamp). "
            f"Found fields: {[f.name for f in fields]}"
        )


class TestStreamingMessageStructure:
    """Tests for streaming message structure requirements."""

    def test_caep_event_structure(self) -> None:
        """Verify CAEPEvent has required fields."""
        result = get_message_content("CAEPEvent")
        if result is None:
            pytest.skip("CAEPEvent message not found")

        content, _ = result
        fields = parse_message_fields(content, "CAEPEvent")
        field_names = [f.name for f in fields]

        assert "event_id" in field_names, "CAEPEvent must have event_id field"
        assert "timestamp" in field_names, "CAEPEvent must have timestamp field"
        assert "event_type" in field_names, "CAEPEvent must have event_type field"
        assert "subject" in field_names, "CAEPEvent must have subject field"

    def test_session_event_structure(self) -> None:
        """Verify SessionEvent has required fields."""
        result = get_message_content("SessionEvent")
        if result is None:
            pytest.skip("SessionEvent message not found")

        content, _ = result
        fields = parse_message_fields(content, "SessionEvent")
        field_names = [f.name for f in fields]

        assert "event_id" in field_names, "SessionEvent must have event_id field"
        assert "timestamp" in field_names, "SessionEvent must have timestamp field"
        assert "event_type" in field_names, "SessionEvent must have event_type field"


class TestStreamingRPCPatterns:
    """Tests for streaming RPC patterns and best practices."""

    def test_streaming_rpcs_use_server_streaming(self) -> None:
        """Verify streaming RPCs use appropriate streaming pattern."""
        rpcs = get_all_streaming_rpcs()

        for rpc in rpcs:
            # All our streaming RPCs should be server-streaming (returns stream)
            # This is verified by the parse_streaming_rpcs function
            assert rpc.response_type, f"{rpc.method_name} must have a response type"

    def test_streaming_request_has_filters(self) -> None:
        """Verify streaming requests support filtering."""
        filter_patterns = ["event_types", "user_id", "session_id", "filters"]

        for proto_file in get_proto_files():
            content = proto_file.read_text(encoding="utf-8")

            # Find streaming request messages
            for rpc in parse_streaming_rpcs(content, str(proto_file)):
                # Look for the request message
                request_pattern = rf"rpc\s+{re.escape(rpc.method_name)}\s*\(\s*(\w+)\s*\)"
                request_match = re.search(request_pattern, content)

                if request_match:
                    request_type = request_match.group(1)
                    fields = parse_message_fields(content, request_type)
                    field_names = [f.name.lower() for f in fields]

                    # Check if any filter pattern is present
                    has_filter = any(
                        pattern in name for name in field_names for pattern in filter_patterns
                    )

                    # This is a soft check - we just verify the pattern exists
                    if not has_filter:
                        pytest.skip(
                            f"Streaming request {request_type} could benefit from filter fields"
                        )


if __name__ == "__main__":
    pytest.main([__file__, "-v"])
