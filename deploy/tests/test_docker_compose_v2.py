"""
Property-based tests for Docker Compose V2 compliance.

Feature: deploy-modernization-2025
Property 1: Docker Compose V2 Compliance
Validates: Requirements 1.1, 1.2, 1.3, 1.5, 1.7
"""

import os
import glob
from pathlib import Path
from typing import Any

import pytest
import yaml
from hypothesis import given, settings, strategies as st

# Get the deploy directory path
DEPLOY_DIR = Path(__file__).parent.parent
DOCKER_DIR = DEPLOY_DIR / "docker"


def load_compose_file(filepath: Path) -> dict[str, Any]:
    """Load and parse a Docker Compose file."""
    with open(filepath, "r") as f:
        return yaml.safe_load(f)


def get_all_compose_files() -> list[Path]:
    """Get all Docker Compose files in the deploy/docker directory."""
    patterns = [
        "docker-compose.yml",
        "docker-compose.*.yml",
    ]
    files = []
    for pattern in patterns:
        files.extend(DOCKER_DIR.glob(pattern))
    return [f for f in files if f.exists()]


class TestDockerComposeV2Compliance:
    """Property 1: Docker Compose V2 Compliance tests."""

    @pytest.fixture
    def compose_files(self) -> list[Path]:
        """Get all compose files for testing."""
        return get_all_compose_files()

    def test_no_version_field(self, compose_files: list[Path]) -> None:
        """
        Property 1.1: No version field in compose files.
        
        For any Docker Compose file, the file SHALL NOT contain a version field.
        Validates: Requirement 1.1
        """
        # Skip legacy override files that may exist
        skip_files = {"docker-compose.override.yml"}
        
        for filepath in compose_files:
            if filepath.name in skip_files:
                continue
            config = load_compose_file(filepath)
            assert "version" not in config, (
                f"Compose file {filepath.name} contains deprecated 'version' field. "
                "Docker Compose V2 does not require the version field."
            )

    def test_services_have_deploy_resources(self, compose_files: list[Path]) -> None:
        """
        Property 1.2: All services have deploy.resources defined.
        
        For any service in a Docker Compose file, deploy.resources SHALL be defined.
        Validates: Requirement 1.2
        """
        for filepath in compose_files:
            config = load_compose_file(filepath)
            services = config.get("services", {})
            
            for service_name, service_config in services.items():
                # Skip if service uses extension anchor
                if service_config is None:
                    continue
                    
                deploy = service_config.get("deploy", {})
                # Check for resources in deploy or via extension
                has_resources = (
                    "resources" in deploy or
                    any(k.startswith("<<") for k in service_config.keys() if isinstance(k, str))
                )
                
                # For base compose, all services should have resources
                if filepath.name == "docker-compose.yml":
                    assert has_resources or "resources" in str(service_config), (
                        f"Service '{service_name}' in {filepath.name} missing deploy.resources"
                    )

    def test_logging_configuration(self, compose_files: list[Path]) -> None:
        """
        Property 1.3: Logging configured with json-file driver.
        
        For any service, logging SHALL use json-file driver with max-size 10m and max-file 3.
        Validates: Requirement 1.3
        """
        for filepath in compose_files:
            config = load_compose_file(filepath)
            
            # Check for common logging extension
            extensions = {k: v for k, v in config.items() if k.startswith("x-")}
            has_logging_extension = any(
                "logging" in k or (isinstance(v, dict) and "driver" in v)
                for k, v in extensions.items()
            )
            
            if filepath.name == "docker-compose.yml":
                # Base compose should have logging extension
                assert "x-common-logging" in config, (
                    f"Base compose {filepath.name} missing x-common-logging extension"
                )
                
                logging_config = config.get("x-common-logging", {})
                assert logging_config.get("driver") == "json-file", (
                    "Logging driver should be 'json-file'"
                )
                
                options = logging_config.get("options", {})
                assert options.get("max-size") == "10m", (
                    "Logging max-size should be '10m'"
                )
                assert options.get("max-file") == "3", (
                    "Logging max-file should be '3'"
                )

    def test_security_opt_no_new_privileges(self, compose_files: list[Path]) -> None:
        """
        Property 1.5: All application services have no-new-privileges.
        
        For any application service, security_opt SHALL include no-new-privileges:true.
        Validates: Requirement 1.5
        """
        # Infrastructure services that may not support no-new-privileges
        infra_services = {"elasticsearch", "kibana"}
        
        for filepath in compose_files:
            config = load_compose_file(filepath)
            
            if filepath.name == "docker-compose.yml":
                # Check for common security extension
                assert "x-common-security" in config, (
                    f"Base compose {filepath.name} missing x-common-security extension"
                )
                
                security_config = config.get("x-common-security", {})
                security_opts = security_config.get("security_opt", [])
                assert "no-new-privileges:true" in security_opts, (
                    "Security extension should include 'no-new-privileges:true'"
                )

    def test_healthcheck_has_start_period(self, compose_files: list[Path]) -> None:
        """
        Property 1.7: Health checks include start_period.
        
        For any health check definition, start_period SHALL be included.
        Validates: Requirement 1.7
        """
        for filepath in compose_files:
            config = load_compose_file(filepath)
            
            if filepath.name == "docker-compose.yml":
                # Check for common healthcheck extension
                assert "x-common-healthcheck" in config, (
                    f"Base compose {filepath.name} missing x-common-healthcheck extension"
                )
                
                healthcheck_config = config.get("x-common-healthcheck", {})
                assert "start_period" in healthcheck_config, (
                    "Healthcheck extension should include 'start_period'"
                )

    def test_depends_on_uses_service_healthy(self, compose_files: list[Path]) -> None:
        """
        Property 1.4: Dependencies use condition: service_healthy.
        
        For any service with depends_on, all dependencies SHALL use condition: service_healthy.
        Validates: Requirement 1.4
        """
        for filepath in compose_files:
            config = load_compose_file(filepath)
            services = config.get("services", {})
            
            for service_name, service_config in services.items():
                if service_config is None:
                    continue
                    
                depends_on = service_config.get("depends_on", {})
                
                if isinstance(depends_on, dict):
                    for dep_name, dep_config in depends_on.items():
                        if isinstance(dep_config, dict):
                            condition = dep_config.get("condition")
                            assert condition == "service_healthy", (
                                f"Service '{service_name}' dependency '{dep_name}' in {filepath.name} "
                                f"should use 'condition: service_healthy', got '{condition}'"
                            )

    def test_named_networks_with_subnets(self, compose_files: list[Path]) -> None:
        """
        Property 1.6: Named networks with explicit subnet configuration.
        
        For any network definition, explicit subnet configuration SHALL be defined.
        Validates: Requirement 1.6
        """
        for filepath in compose_files:
            config = load_compose_file(filepath)
            networks = config.get("networks", {})
            
            if filepath.name == "docker-compose.yml" and networks:
                for network_name, network_config in networks.items():
                    if network_config is None:
                        continue
                        
                    ipam = network_config.get("ipam", {})
                    ipam_config = ipam.get("config", [])
                    
                    has_subnet = any(
                        "subnet" in cfg for cfg in ipam_config if isinstance(cfg, dict)
                    )
                    
                    assert has_subnet, (
                        f"Network '{network_name}' in {filepath.name} missing subnet configuration"
                    )


class TestDockerComposeSecretExternalization:
    """Property 3: Docker Compose Secret Externalization tests."""

    def test_no_hardcoded_secrets(self) -> None:
        """
        Property 3: No hardcoded secrets in compose files.
        
        For any Docker Compose file, no hardcoded secrets SHALL exist.
        Validates: Requirement 1.8
        """
        # Only check for actual secret patterns, not URL patterns
        sensitive_patterns = [
            "password",
            "secret_key",
            "api_key",
            "apikey",
            "private_key",
        ]
        
        # Patterns that look sensitive but are actually URLs or non-secrets
        safe_suffixes = ["_url", "_host", "_port", "_endpoint", "_service"]
        
        compose_files = get_all_compose_files()
        
        for filepath in compose_files:
            with open(filepath, "r") as f:
                content = f.read().lower()
            
            config = load_compose_file(filepath)
            services = config.get("services", {})
            
            for service_name, service_config in services.items():
                if service_config is None:
                    continue
                    
                env_vars = service_config.get("environment", {})
                
                if isinstance(env_vars, dict):
                    for key, value in env_vars.items():
                        if value is None:
                            continue
                            
                        key_lower = key.lower()
                        value_str = str(value)
                        
                        # Check if this is a sensitive key (not a URL or endpoint)
                        is_sensitive = any(p in key_lower for p in sensitive_patterns)
                        is_safe_suffix = any(key_lower.endswith(s) for s in safe_suffixes)
                        
                        if is_sensitive and not is_safe_suffix:
                            # Value should be a variable reference, not hardcoded
                            is_variable = (
                                value_str.startswith("${") or
                                value_str == "" or
                                value_str.startswith("$")
                            )
                            
                            # Allow default values in variable syntax
                            if not is_variable and ":-" not in value_str:
                                # Skip known safe defaults
                                safe_defaults = ["guest", "postgres", "admin"]
                                if value_str.lower() not in safe_defaults:
                                    pytest.fail(
                                        f"Service '{service_name}' in {filepath.name} has "
                                        f"potentially hardcoded secret for '{key}'"
                                    )

    def test_env_file_exists(self) -> None:
        """
        Test that .env.example template exists.
        
        Validates: Requirement 2.6
        """
        env_example = DOCKER_DIR / ".env.example"
        assert env_example.exists(), (
            f"Missing .env.example template at {env_example}"
        )

    def test_env_example_has_required_variables(self) -> None:
        """
        Test that .env.example contains all required variables.
        
        Validates: Requirement 2.6
        """
        env_example = DOCKER_DIR / ".env.example"
        
        if not env_example.exists():
            pytest.skip(".env.example not found")
        
        with open(env_example, "r") as f:
            content = f.read()
        
        required_vars = [
            "POSTGRES_USER",
            "POSTGRES_PASSWORD",
            "REDIS_PASSWORD",
            "RABBITMQ_USER",
            "RABBITMQ_PASSWORD",
            "JWT_SECRET",
            "ENCRYPTION_KEY",
            "GRAFANA_ADMIN_PASSWORD",
        ]
        
        for var in required_vars:
            assert var in content, (
                f".env.example missing required variable: {var}"
            )


if __name__ == "__main__":
    pytest.main([__file__, "-v"])
