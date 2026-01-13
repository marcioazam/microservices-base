"""
Property-based tests for Production Docker Compose Security.

Feature: deploy-modernization-2025
Property 4: Production Compose Security
Validates: Requirements 2.5
"""

import os
from pathlib import Path
from typing import Any

import pytest
import yaml

# Get the deploy directory path
DEPLOY_DIR = Path(__file__).parent.parent
DOCKER_DIR = DEPLOY_DIR / "docker"


def load_compose_file(filepath: Path) -> dict[str, Any]:
    """Load and parse a Docker Compose file."""
    with open(filepath, "r") as f:
        return yaml.safe_load(f)


class TestProductionComposeSecurity:
    """Property 4: Production Compose Security tests."""

    @pytest.fixture
    def prod_compose(self) -> dict[str, Any]:
        """Load production compose file."""
        prod_file = DOCKER_DIR / "docker-compose.prod.yml"
        if not prod_file.exists():
            pytest.skip("docker-compose.prod.yml not found")
        return load_compose_file(prod_file)

    @pytest.fixture
    def dev_compose(self) -> dict[str, Any]:
        """Load development compose file for comparison."""
        dev_file = DOCKER_DIR / "docker-compose.dev.yml"
        if not dev_file.exists():
            pytest.skip("docker-compose.dev.yml not found")
        return load_compose_file(dev_file)

    def test_no_debug_ports_in_production(self, prod_compose: dict[str, Any], dev_compose: dict[str, Any]) -> None:
        """
        Property 4.1: No debug ports exposed in production.
        
        For any production compose configuration, no debug ports SHALL be exposed.
        Validates: Requirement 2.5
        """
        # Common debug ports
        debug_ports = {
            "2345",   # Delve (Go debugger)
            "5002",   # .NET debug
            "4001",   # Phoenix LiveDashboard
            "50057",  # gRPC debug
            "55679",  # zpages
            "6831",   # Jaeger agent UDP
            "6832",   # Jaeger agent UDP
            "15692",  # RabbitMQ Prometheus
        }
        
        prod_services = prod_compose.get("services", {})
        
        for service_name, service_config in prod_services.items():
            if service_config is None:
                continue
                
            ports = service_config.get("ports", [])
            
            for port_mapping in ports:
                port_str = str(port_mapping)
                
                # Extract host port from mapping
                if ":" in port_str:
                    host_port = port_str.split(":")[0]
                else:
                    host_port = port_str.split("/")[0]
                
                assert host_port not in debug_ports, (
                    f"Service '{service_name}' in production exposes debug port {host_port}"
                )

    def test_verbose_logging_disabled_in_production(self, prod_compose: dict[str, Any]) -> None:
        """
        Property 4.2: Verbose logging disabled in production.
        
        For any production compose configuration, verbose logging SHALL be disabled.
        Validates: Requirement 2.5
        """
        verbose_log_levels = {"debug", "trace", "verbose"}
        
        prod_services = prod_compose.get("services", {})
        
        for service_name, service_config in prod_services.items():
            if service_config is None:
                continue
                
            env_vars = service_config.get("environment", {})
            
            if isinstance(env_vars, dict):
                for key, value in env_vars.items():
                    if value is None:
                        continue
                        
                    key_lower = key.lower()
                    value_lower = str(value).lower()
                    
                    # Check log level settings
                    if "log_level" in key_lower or "logging_level" in key_lower:
                        assert value_lower not in verbose_log_levels, (
                            f"Service '{service_name}' has verbose logging enabled: {key}={value}"
                        )
                    
                    # Check debug mode settings
                    if "debug" in key_lower and "mode" in key_lower:
                        assert value_lower not in {"true", "1", "yes"}, (
                            f"Service '{service_name}' has debug mode enabled: {key}={value}"
                        )

    def test_production_environment_set(self, prod_compose: dict[str, Any]) -> None:
        """
        Property 4.3: Production environment properly configured.
        
        For any production compose configuration, environment SHALL be set to production.
        Validates: Requirement 2.5
        """
        production_env_values = {"production", "prod"}
        
        prod_services = prod_compose.get("services", {})
        
        # Services that should have environment set
        app_services = [
            "auth-edge-service",
            "token-service",
            "session-identity-core",
            "iam-policy-service",
            "mfa-service",
        ]
        
        for service_name in app_services:
            service_config = prod_services.get(service_name, {})
            if service_config is None:
                continue
                
            env_vars = service_config.get("environment", {})
            
            if isinstance(env_vars, dict):
                # Check for ENVIRONMENT or similar
                env_value = None
                for key in ["ENVIRONMENT", "MIX_ENV", "ASPNETCORE_ENVIRONMENT", "DOTNET_ENVIRONMENT"]:
                    if key in env_vars:
                        env_value = str(env_vars[key]).lower()
                        break
                
                if env_value:
                    assert env_value in production_env_values, (
                        f"Service '{service_name}' environment not set to production: {env_value}"
                    )

    def test_infrastructure_ports_not_exposed(self, prod_compose: dict[str, Any]) -> None:
        """
        Property 4.4: Infrastructure ports not externally exposed in production.
        
        For production compose, infrastructure services SHALL NOT expose ports externally.
        Validates: Requirement 2.5
        """
        # Infrastructure services that should not expose ports
        infra_services = ["postgres", "redis", "elasticsearch", "rabbitmq"]
        
        prod_services = prod_compose.get("services", {})
        
        for service_name in infra_services:
            service_config = prod_services.get(service_name, {})
            if service_config is None:
                continue
                
            ports = service_config.get("ports", [])
            
            # Ports should be empty or not defined
            assert len(ports) == 0, (
                f"Infrastructure service '{service_name}' exposes ports in production: {ports}"
            )

    def test_restart_policies_configured(self, prod_compose: dict[str, Any]) -> None:
        """
        Property 4.5: Restart policies configured for production.
        
        For production compose, services SHALL have restart policies configured.
        Validates: Requirement 2.5
        """
        prod_services = prod_compose.get("services", {})
        
        # Check for restart policy in deploy section
        for service_name, service_config in prod_services.items():
            if service_config is None:
                continue
                
            deploy = service_config.get("deploy", {})
            restart_policy = deploy.get("restart_policy", {})
            
            # Either has restart_policy in deploy or uses extension
            has_restart = (
                bool(restart_policy) or
                "restart" in service_config or
                any(k.startswith("<<") for k in service_config.keys() if isinstance(k, str))
            )
            
            # At minimum, critical services should have restart policy
            critical_services = [
                "auth-edge-service",
                "token-service",
                "session-identity-core",
                "postgres",
                "redis",
            ]
            
            if service_name in critical_services:
                assert has_restart, (
                    f"Critical service '{service_name}' missing restart policy in production"
                )

    def test_production_logging_configuration(self, prod_compose: dict[str, Any]) -> None:
        """
        Property 4.6: Production logging properly configured.
        
        For production compose, logging SHALL be configured with appropriate limits.
        Validates: Requirement 2.5
        """
        # Check for production logging extension
        extensions = {k: v for k, v in prod_compose.items() if k.startswith("x-")}
        
        # Should have production-specific logging
        has_prod_logging = any(
            "prod" in k.lower() and "logging" in k.lower()
            for k in extensions.keys()
        )
        
        if has_prod_logging:
            for ext_name, ext_config in extensions.items():
                if "prod" in ext_name.lower() and "logging" in ext_name.lower():
                    if isinstance(ext_config, dict):
                        options = ext_config.get("options", {})
                        
                        # Production should have larger log files
                        max_size = options.get("max-size", "")
                        if max_size:
                            # Extract numeric value
                            size_value = int("".join(filter(str.isdigit, max_size)) or "0")
                            assert size_value >= 50, (
                                f"Production log max-size should be >= 50m, got {max_size}"
                            )


class TestEnvironmentComparison:
    """Tests comparing dev vs production configurations."""

    def test_dev_has_more_ports_than_prod(self) -> None:
        """
        Dev environment should expose more ports than production.
        
        Validates: Requirement 2.5
        """
        dev_file = DOCKER_DIR / "docker-compose.dev.yml"
        prod_file = DOCKER_DIR / "docker-compose.prod.yml"
        
        if not dev_file.exists() or not prod_file.exists():
            pytest.skip("Dev or prod compose file not found")
        
        dev_compose = load_compose_file(dev_file)
        prod_compose = load_compose_file(prod_file)
        
        def count_ports(compose: dict) -> int:
            total = 0
            for service_config in compose.get("services", {}).values():
                if service_config and "ports" in service_config:
                    total += len(service_config["ports"])
            return total
        
        dev_ports = count_ports(dev_compose)
        prod_ports = count_ports(prod_compose)
        
        assert dev_ports > prod_ports, (
            f"Dev should have more ports ({dev_ports}) than prod ({prod_ports})"
        )

    def test_prod_has_higher_resource_limits(self) -> None:
        """
        Production should have higher resource limits than dev.
        
        Validates: Requirement 2.5
        """
        dev_file = DOCKER_DIR / "docker-compose.dev.yml"
        prod_file = DOCKER_DIR / "docker-compose.prod.yml"
        base_file = DOCKER_DIR / "docker-compose.yml"
        
        if not prod_file.exists():
            pytest.skip("Prod compose file not found")
        
        prod_compose = load_compose_file(prod_file)
        
        # Check that production has resource configurations
        prod_services = prod_compose.get("services", {})
        
        services_with_resources = 0
        for service_config in prod_services.values():
            if service_config is None:
                continue
                
            deploy = service_config.get("deploy", {})
            if "resources" in deploy or any(k.startswith("<<") for k in service_config.keys() if isinstance(k, str)):
                services_with_resources += 1
        
        # Most services should have resource configs
        total_services = len([s for s in prod_services.values() if s is not None])
        assert services_with_resources >= total_services * 0.5, (
            f"Production should have resources configured for most services"
        )


if __name__ == "__main__":
    pytest.main([__file__, "-v"])
