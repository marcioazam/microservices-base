"""
Property-based tests for Kubernetes Security Hardening.

Feature: deploy-modernization-2025
Property 7: Kubernetes Pod Security Standards
Property 8: Kubernetes Resource Constraints
Validates: Requirements 4.1, 4.3, 4.4, 4.5, 4.6
"""

import glob
from pathlib import Path
from typing import Any

import pytest
import yaml

# Get the deploy directory path
DEPLOY_DIR = Path(__file__).parent.parent
K8S_BASE_DIR = DEPLOY_DIR / "kubernetes" / "base"


def load_yaml_file(filepath: Path) -> list[dict[str, Any]]:
    """Load and parse a YAML file, handling multi-document files."""
    if not filepath.exists():
        return []
    
    with open(filepath, "r") as f:
        content = f.read()
    
    docs = []
    for doc in yaml.safe_load_all(content):
        if doc:
            docs.append(doc)
    return docs


def get_all_deployments() -> list[tuple[Path, dict[str, Any]]]:
    """Get all Deployment manifests from the base directory."""
    deployments = []
    
    for yaml_file in K8S_BASE_DIR.rglob("*.yaml"):
        docs = load_yaml_file(yaml_file)
        for doc in docs:
            if doc.get("kind") == "Deployment":
                deployments.append((yaml_file, doc))
    
    return deployments


class TestKubernetesPodSecurityStandards:
    """Property 7: Kubernetes Pod Security Standards tests."""

    @pytest.fixture
    def deployments(self) -> list[tuple[Path, dict[str, Any]]]:
        """Get all deployments for testing."""
        deps = get_all_deployments()
        if not deps:
            pytest.skip("No deployments found in base directory")
        return deps

    def test_run_as_non_root(self, deployments: list[tuple[Path, dict[str, Any]]]) -> None:
        """
        Property 7.1: All pods run as non-root.
        
        For any Kubernetes Deployment, pods SHALL have runAsNonRoot: true.
        Validates: Requirement 4.1
        """
        for filepath, deployment in deployments:
            name = deployment.get("metadata", {}).get("name", "unknown")
            pod_spec = deployment.get("spec", {}).get("template", {}).get("spec", {})
            security_context = pod_spec.get("securityContext", {})
            
            assert security_context.get("runAsNonRoot") is True, (
                f"Deployment '{name}' in {filepath.name} missing runAsNonRoot: true"
            )

    def test_seccomp_profile_runtime_default(self, deployments: list[tuple[Path, dict[str, Any]]]) -> None:
        """
        Property 7.2: All pods use RuntimeDefault seccomp profile.
        
        For any Kubernetes Deployment, pods SHALL have seccompProfile.type: RuntimeDefault.
        Validates: Requirement 4.6
        """
        for filepath, deployment in deployments:
            name = deployment.get("metadata", {}).get("name", "unknown")
            pod_spec = deployment.get("spec", {}).get("template", {}).get("spec", {})
            security_context = pod_spec.get("securityContext", {})
            seccomp = security_context.get("seccompProfile", {})
            
            assert seccomp.get("type") == "RuntimeDefault", (
                f"Deployment '{name}' in {filepath.name} missing seccompProfile.type: RuntimeDefault"
            )

    def test_allow_privilege_escalation_false(self, deployments: list[tuple[Path, dict[str, Any]]]) -> None:
        """
        Property 7.3: All containers have allowPrivilegeEscalation: false.
        
        For any container in a Kubernetes Deployment, allowPrivilegeEscalation SHALL be false.
        Validates: Requirement 4.4
        """
        for filepath, deployment in deployments:
            name = deployment.get("metadata", {}).get("name", "unknown")
            pod_spec = deployment.get("spec", {}).get("template", {}).get("spec", {})
            containers = pod_spec.get("containers", [])
            
            for container in containers:
                container_name = container.get("name", "unknown")
                security_context = container.get("securityContext", {})
                
                assert security_context.get("allowPrivilegeEscalation") is False, (
                    f"Container '{container_name}' in deployment '{name}' ({filepath.name}) "
                    "missing allowPrivilegeEscalation: false"
                )

    def test_capabilities_drop_all(self, deployments: list[tuple[Path, dict[str, Any]]]) -> None:
        """
        Property 7.4: All containers drop ALL capabilities.
        
        For any container in a Kubernetes Deployment, capabilities SHALL drop ALL.
        Validates: Requirement 4.3
        """
        for filepath, deployment in deployments:
            name = deployment.get("metadata", {}).get("name", "unknown")
            pod_spec = deployment.get("spec", {}).get("template", {}).get("spec", {})
            containers = pod_spec.get("containers", [])
            
            for container in containers:
                container_name = container.get("name", "unknown")
                security_context = container.get("securityContext", {})
                capabilities = security_context.get("capabilities", {})
                drop = capabilities.get("drop", [])
                
                assert "ALL" in drop, (
                    f"Container '{container_name}' in deployment '{name}' ({filepath.name}) "
                    "should drop ALL capabilities"
                )


class TestKubernetesResourceConstraints:
    """Property 8: Kubernetes Resource Constraints tests."""

    @pytest.fixture
    def deployments(self) -> list[tuple[Path, dict[str, Any]]]:
        """Get all deployments for testing."""
        deps = get_all_deployments()
        if not deps:
            pytest.skip("No deployments found in base directory")
        return deps

    def test_resource_requests_defined(self, deployments: list[tuple[Path, dict[str, Any]]]) -> None:
        """
        Property 8.1: All containers have resource requests defined.
        
        For any container in a Kubernetes Deployment, resource requests SHALL be defined.
        Validates: Requirement 4.5
        """
        for filepath, deployment in deployments:
            name = deployment.get("metadata", {}).get("name", "unknown")
            pod_spec = deployment.get("spec", {}).get("template", {}).get("spec", {})
            containers = pod_spec.get("containers", [])
            
            for container in containers:
                container_name = container.get("name", "unknown")
                resources = container.get("resources", {})
                requests = resources.get("requests", {})
                
                assert "cpu" in requests, (
                    f"Container '{container_name}' in deployment '{name}' ({filepath.name}) "
                    "missing CPU request"
                )
                assert "memory" in requests, (
                    f"Container '{container_name}' in deployment '{name}' ({filepath.name}) "
                    "missing memory request"
                )

    def test_resource_limits_defined(self, deployments: list[tuple[Path, dict[str, Any]]]) -> None:
        """
        Property 8.2: All containers have resource limits defined.
        
        For any container in a Kubernetes Deployment, resource limits SHALL be defined.
        Validates: Requirement 4.5
        """
        for filepath, deployment in deployments:
            name = deployment.get("metadata", {}).get("name", "unknown")
            pod_spec = deployment.get("spec", {}).get("template", {}).get("spec", {})
            containers = pod_spec.get("containers", [])
            
            for container in containers:
                container_name = container.get("name", "unknown")
                resources = container.get("resources", {})
                limits = resources.get("limits", {})
                
                assert "cpu" in limits, (
                    f"Container '{container_name}' in deployment '{name}' ({filepath.name}) "
                    "missing CPU limit"
                )
                assert "memory" in limits, (
                    f"Container '{container_name}' in deployment '{name}' ({filepath.name}) "
                    "missing memory limit"
                )

    def test_emptydir_has_size_limit(self, deployments: list[tuple[Path, dict[str, Any]]]) -> None:
        """
        Property 8.3: EmptyDir volumes have size limits.
        
        For any emptyDir volume in a Kubernetes Deployment, sizeLimit SHALL be defined.
        Validates: Requirement 4.7
        """
        for filepath, deployment in deployments:
            name = deployment.get("metadata", {}).get("name", "unknown")
            pod_spec = deployment.get("spec", {}).get("template", {}).get("spec", {})
            volumes = pod_spec.get("volumes", [])
            
            for volume in volumes:
                volume_name = volume.get("name", "unknown")
                empty_dir = volume.get("emptyDir")
                
                if empty_dir is not None:
                    assert "sizeLimit" in empty_dir, (
                        f"EmptyDir volume '{volume_name}' in deployment '{name}' ({filepath.name}) "
                        "missing sizeLimit"
                    )


class TestKubernetesAdditionalSecurity:
    """Additional Kubernetes security tests."""

    @pytest.fixture
    def deployments(self) -> list[tuple[Path, dict[str, Any]]]:
        """Get all deployments for testing."""
        deps = get_all_deployments()
        if not deps:
            pytest.skip("No deployments found in base directory")
        return deps

    def test_read_only_root_filesystem(self, deployments: list[tuple[Path, dict[str, Any]]]) -> None:
        """
        Test that containers have readOnlyRootFilesystem where applicable.
        
        Validates: Requirement 4.2
        """
        for filepath, deployment in deployments:
            name = deployment.get("metadata", {}).get("name", "unknown")
            pod_spec = deployment.get("spec", {}).get("template", {}).get("spec", {})
            containers = pod_spec.get("containers", [])
            
            for container in containers:
                container_name = container.get("name", "unknown")
                security_context = container.get("securityContext", {})
                
                # readOnlyRootFilesystem should be true for most containers
                read_only = security_context.get("readOnlyRootFilesystem")
                
                assert read_only is True, (
                    f"Container '{container_name}' in deployment '{name}' ({filepath.name}) "
                    "should have readOnlyRootFilesystem: true"
                )

    def test_service_account_configured(self, deployments: list[tuple[Path, dict[str, Any]]]) -> None:
        """
        Test that deployments have service accounts configured.
        """
        for filepath, deployment in deployments:
            name = deployment.get("metadata", {}).get("name", "unknown")
            pod_spec = deployment.get("spec", {}).get("template", {}).get("spec", {})
            
            service_account = pod_spec.get("serviceAccountName")
            
            assert service_account is not None, (
                f"Deployment '{name}' in {filepath.name} missing serviceAccountName"
            )

    def test_automount_service_account_token_disabled(self, deployments: list[tuple[Path, dict[str, Any]]]) -> None:
        """
        Test that automountServiceAccountToken is disabled.
        """
        for filepath, deployment in deployments:
            name = deployment.get("metadata", {}).get("name", "unknown")
            pod_spec = deployment.get("spec", {}).get("template", {}).get("spec", {})
            
            automount = pod_spec.get("automountServiceAccountToken")
            
            assert automount is False, (
                f"Deployment '{name}' in {filepath.name} should have automountServiceAccountToken: false"
            )

    def test_pod_anti_affinity_configured(self, deployments: list[tuple[Path, dict[str, Any]]]) -> None:
        """
        Test that deployments have pod anti-affinity configured.
        """
        for filepath, deployment in deployments:
            name = deployment.get("metadata", {}).get("name", "unknown")
            pod_spec = deployment.get("spec", {}).get("template", {}).get("spec", {})
            
            affinity = pod_spec.get("affinity", {})
            pod_anti_affinity = affinity.get("podAntiAffinity", {})
            
            # Should have either preferred or required anti-affinity
            has_anti_affinity = (
                "preferredDuringSchedulingIgnoredDuringExecution" in pod_anti_affinity or
                "requiredDuringSchedulingIgnoredDuringExecution" in pod_anti_affinity
            )
            
            assert has_anti_affinity, (
                f"Deployment '{name}' in {filepath.name} missing pod anti-affinity configuration"
            )


class TestNamespaceSecurityLabels:
    """Test namespace security labels."""

    def test_namespace_has_pss_labels(self) -> None:
        """
        Test that namespace has Pod Security Standards labels.
        """
        namespace_file = K8S_BASE_DIR / "namespace.yaml"
        
        if not namespace_file.exists():
            pytest.skip("namespace.yaml not found")
        
        docs = load_yaml_file(namespace_file)
        
        namespace = None
        for doc in docs:
            if doc.get("kind") == "Namespace":
                namespace = doc
                break
        
        assert namespace is not None, "No Namespace resource found"
        
        labels = namespace.get("metadata", {}).get("labels", {})
        
        # Check for PSS enforce label
        assert "pod-security.kubernetes.io/enforce" in labels, (
            "Namespace missing pod-security.kubernetes.io/enforce label"
        )
        
        enforce_level = labels.get("pod-security.kubernetes.io/enforce")
        assert enforce_level in ["restricted", "baseline"], (
            f"Namespace PSS enforce level should be 'restricted' or 'baseline', got '{enforce_level}'"
        )


if __name__ == "__main__":
    pytest.main([__file__, "-v"])
