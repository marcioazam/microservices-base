"""
Property-based tests for OpenTelemetry Collector Configuration.

Feature: deploy-modernization-2025
Property 5: OTel Collector Security Filtering
Property 6: OTel Collector Exporter Retry
Validates: Requirements 3.2, 3.8
"""

from pathlib import Path
from typing import Any

import pytest
import yaml

# Get the deploy directory path
DEPLOY_DIR = Path(__file__).parent.parent
OTEL_CONFIG_PATH = DEPLOY_DIR / "docker" / "observability" / "otel-collector-config.yaml"


def load_otel_config() -> dict[str, Any]:
    """Load and parse the OTel Collector configuration."""
    if not OTEL_CONFIG_PATH.exists():
        pytest.skip(f"OTel config not found at {OTEL_CONFIG_PATH}")
    
    with open(OTEL_CONFIG_PATH, "r") as f:
        return yaml.safe_load(f)


class TestOTelCollectorSecurityFiltering:
    """Property 5: OTel Collector Security Filtering tests."""

    @pytest.fixture
    def otel_config(self) -> dict[str, Any]:
        """Load OTel configuration."""
        return load_otel_config()

    def test_sensitive_attribute_filtering_configured(self, otel_config: dict[str, Any]) -> None:
        """
        Property 5.1: Sensitive data filtering is configured.
        
        For any OTel Collector configuration, sensitive attribute filtering SHALL be configured.
        Validates: Requirement 3.2
        """
        processors = otel_config.get("processors", {})
        
        # Find attribute filtering processor
        filter_processors = [
            name for name in processors.keys()
            if "filter" in name.lower() or "attributes" in name.lower()
        ]
        
        assert len(filter_processors) > 0, (
            "OTel Collector missing attribute filtering processor"
        )

    def test_password_attributes_deleted(self, otel_config: dict[str, Any]) -> None:
        """
        Property 5.2: Password attributes are deleted.
        
        For any OTel Collector configuration, password attributes SHALL be deleted.
        Validates: Requirement 3.2
        """
        processors = otel_config.get("processors", {})
        
        # Find attribute filtering processor
        filter_config = None
        for name, config in processors.items():
            if "filter" in name.lower() or "attributes" in name.lower():
                filter_config = config
                break
        
        assert filter_config is not None, "No attribute filter processor found"
        
        actions = filter_config.get("actions", [])
        deleted_keys = [
            action.get("key") for action in actions
            if action.get("action") == "delete"
        ]
        
        # Must delete password-related attributes
        password_keys = ["password", "passwd", "secret"]
        for key in password_keys:
            assert key in deleted_keys, (
                f"OTel Collector should delete '{key}' attribute"
            )

    def test_token_attributes_deleted(self, otel_config: dict[str, Any]) -> None:
        """
        Property 5.3: Token attributes are deleted.
        
        For any OTel Collector configuration, token attributes SHALL be deleted.
        Validates: Requirement 3.2
        """
        processors = otel_config.get("processors", {})
        
        filter_config = None
        for name, config in processors.items():
            if "filter" in name.lower() or "attributes" in name.lower():
                filter_config = config
                break
        
        assert filter_config is not None, "No attribute filter processor found"
        
        actions = filter_config.get("actions", [])
        deleted_keys = [
            action.get("key") for action in actions
            if action.get("action") == "delete"
        ]
        
        # Must delete token-related attributes
        token_keys = ["token", "authorization", "api_key", "bearer"]
        for key in token_keys:
            assert key in deleted_keys, (
                f"OTel Collector should delete '{key}' attribute"
            )

    def test_authorization_header_deleted(self, otel_config: dict[str, Any]) -> None:
        """
        Property 5.4: Authorization headers are deleted.
        
        For any OTel Collector configuration, authorization headers SHALL be deleted.
        Validates: Requirement 3.2
        """
        processors = otel_config.get("processors", {})
        
        filter_config = None
        for name, config in processors.items():
            if "filter" in name.lower() or "attributes" in name.lower():
                filter_config = config
                break
        
        assert filter_config is not None, "No attribute filter processor found"
        
        actions = filter_config.get("actions", [])
        deleted_keys = [
            action.get("key") for action in actions
            if action.get("action") == "delete"
        ]
        
        # Must delete HTTP authorization header
        auth_header_keys = [
            "http.request.header.authorization",
            "http.request.header.cookie",
        ]
        
        for key in auth_header_keys:
            assert key in deleted_keys, (
                f"OTel Collector should delete '{key}' attribute"
            )


class TestOTelCollectorExporterRetry:
    """Property 6: OTel Collector Exporter Retry tests."""

    @pytest.fixture
    def otel_config(self) -> dict[str, Any]:
        """Load OTel configuration."""
        return load_otel_config()

    def test_all_exporters_have_retry(self, otel_config: dict[str, Any]) -> None:
        """
        Property 6.1: All exporters have retry on failure configured.
        
        For any exporter in the OTel Collector, retry on failure SHALL be enabled.
        Validates: Requirement 3.8
        """
        exporters = otel_config.get("exporters", {})
        
        # Exporters that should have retry (exclude prometheus which doesn't support it)
        retry_required_exporters = [
            name for name in exporters.keys()
            if name not in ["prometheus", "debug", "logging"]
        ]
        
        for exporter_name in retry_required_exporters:
            exporter_config = exporters.get(exporter_name, {})
            
            if isinstance(exporter_config, dict):
                retry_config = exporter_config.get("retry_on_failure", {})
                
                assert retry_config.get("enabled", False) is True, (
                    f"Exporter '{exporter_name}' should have retry_on_failure enabled"
                )

    def test_retry_has_initial_interval(self, otel_config: dict[str, Any]) -> None:
        """
        Property 6.2: Retry configuration has initial interval.
        
        For any exporter with retry, initial_interval SHALL be configured.
        Validates: Requirement 3.8
        """
        exporters = otel_config.get("exporters", {})
        
        for exporter_name, exporter_config in exporters.items():
            if not isinstance(exporter_config, dict):
                continue
                
            retry_config = exporter_config.get("retry_on_failure", {})
            
            if retry_config.get("enabled", False):
                assert "initial_interval" in retry_config, (
                    f"Exporter '{exporter_name}' retry missing initial_interval"
                )

    def test_retry_has_max_interval(self, otel_config: dict[str, Any]) -> None:
        """
        Property 6.3: Retry configuration has max interval.
        
        For any exporter with retry, max_interval SHALL be configured.
        Validates: Requirement 3.8
        """
        exporters = otel_config.get("exporters", {})
        
        for exporter_name, exporter_config in exporters.items():
            if not isinstance(exporter_config, dict):
                continue
                
            retry_config = exporter_config.get("retry_on_failure", {})
            
            if retry_config.get("enabled", False):
                assert "max_interval" in retry_config, (
                    f"Exporter '{exporter_name}' retry missing max_interval"
                )

    def test_retry_has_max_elapsed_time(self, otel_config: dict[str, Any]) -> None:
        """
        Property 6.4: Retry configuration has max elapsed time.
        
        For any exporter with retry, max_elapsed_time SHALL be configured.
        Validates: Requirement 3.8
        """
        exporters = otel_config.get("exporters", {})
        
        for exporter_name, exporter_config in exporters.items():
            if not isinstance(exporter_config, dict):
                continue
                
            retry_config = exporter_config.get("retry_on_failure", {})
            
            if retry_config.get("enabled", False):
                assert "max_elapsed_time" in retry_config, (
                    f"Exporter '{exporter_name}' retry missing max_elapsed_time"
                )


class TestOTelCollectorConfiguration:
    """Additional OTel Collector configuration tests."""

    @pytest.fixture
    def otel_config(self) -> dict[str, Any]:
        """Load OTel configuration."""
        return load_otel_config()

    def test_resource_detection_configured(self, otel_config: dict[str, Any]) -> None:
        """
        Test that resource detection processor is configured.
        
        Validates: Requirement 3.1
        """
        processors = otel_config.get("processors", {})
        
        assert "resourcedetection" in processors, (
            "OTel Collector missing resourcedetection processor"
        )
        
        resource_config = processors.get("resourcedetection", {})
        detectors = resource_config.get("detectors", [])
        
        # Should have container and system detectors
        assert "docker" in detectors or "container" in detectors, (
            "Resource detection should include docker/container detector"
        )
        assert "system" in detectors, (
            "Resource detection should include system detector"
        )

    def test_tail_sampling_configured(self, otel_config: dict[str, Any]) -> None:
        """
        Test that tail-based sampling is configured.
        
        Validates: Requirement 3.3
        """
        processors = otel_config.get("processors", {})
        
        assert "tail_sampling" in processors, (
            "OTel Collector missing tail_sampling processor"
        )
        
        sampling_config = processors.get("tail_sampling", {})
        
        assert "decision_wait" in sampling_config, (
            "Tail sampling missing decision_wait"
        )
        assert "policies" in sampling_config, (
            "Tail sampling missing policies"
        )

    def test_prometheus_exporter_configured(self, otel_config: dict[str, Any]) -> None:
        """
        Test that Prometheus exporter is properly configured.
        
        Validates: Requirement 3.4
        """
        exporters = otel_config.get("exporters", {})
        
        assert "prometheus" in exporters, (
            "OTel Collector missing prometheus exporter"
        )
        
        prom_config = exporters.get("prometheus", {})
        
        assert "endpoint" in prom_config, (
            "Prometheus exporter missing endpoint"
        )
        
        # Should have resource to telemetry conversion
        r2t = prom_config.get("resource_to_telemetry_conversion", {})
        assert r2t.get("enabled", False) is True, (
            "Prometheus exporter should have resource_to_telemetry_conversion enabled"
        )

    def test_health_check_extension_configured(self, otel_config: dict[str, Any]) -> None:
        """
        Test that health check extension is configured on port 13133.
        
        Validates: Requirement 3.5
        """
        extensions = otel_config.get("extensions", {})
        
        assert "health_check" in extensions, (
            "OTel Collector missing health_check extension"
        )
        
        health_config = extensions.get("health_check", {})
        endpoint = health_config.get("endpoint", "")
        
        assert "13133" in endpoint, (
            f"Health check should be on port 13133, got endpoint: {endpoint}"
        )

    def test_memory_limiter_configured(self, otel_config: dict[str, Any]) -> None:
        """
        Test that memory limiter is properly configured.
        
        Validates: Requirement 3.7
        """
        processors = otel_config.get("processors", {})
        
        assert "memory_limiter" in processors, (
            "OTel Collector missing memory_limiter processor"
        )
        
        limiter_config = processors.get("memory_limiter", {})
        
        assert "limit_mib" in limiter_config or "limit_percentage" in limiter_config, (
            "Memory limiter missing limit configuration"
        )
        assert "spike_limit_mib" in limiter_config or "spike_limit_percentage" in limiter_config, (
            "Memory limiter missing spike limit configuration"
        )

    def test_log_transform_configured(self, otel_config: dict[str, Any]) -> None:
        """
        Test that log transformation for JSON parsing is configured.
        
        Validates: Requirement 3.6
        """
        processors = otel_config.get("processors", {})
        
        # Find transform processor for logs
        transform_processors = [
            name for name in processors.keys()
            if "transform" in name.lower()
        ]
        
        assert len(transform_processors) > 0, (
            "OTel Collector missing transform processor for logs"
        )
        
        # Check that it has log statements
        for proc_name in transform_processors:
            proc_config = processors.get(proc_name, {})
            if "log_statements" in proc_config:
                statements = proc_config.get("log_statements", [])
                assert len(statements) > 0, (
                    "Transform processor should have log statements"
                )
                break
        else:
            pytest.fail("No transform processor with log_statements found")


if __name__ == "__main__":
    pytest.main([__file__, "-v"])
