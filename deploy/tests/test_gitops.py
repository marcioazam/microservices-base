"""
Property-based tests for GitOps Configuration.
Validates: Requirements 7.1-7.6

Property 12: GitOps Sync Policy
"""

import yaml
from pathlib import Path

OVERLAYS_PATH = Path(__file__).parent.parent / "kubernetes" / "overlays"
ARGOCD_PATH = Path(__file__).parent.parent / "kubernetes" / "argocd"


def load_yaml_file(filepath: Path) -> dict:
    """Load a single YAML file."""
    with open(filepath) as f:
        return yaml.safe_load(f)


def load_yaml_docs(filepath: Path) -> list[dict]:
    """Load all YAML documents from a file."""
    with open(filepath) as f:
        return list(yaml.safe_load_all(f))


class TestKustomizeOverlays:
    """Property tests for Kustomize overlays - Requirements 7.1, 7.2"""
    
    def test_dev_overlay_exists(self):
        """Development overlay must exist."""
        kustomization = OVERLAYS_PATH / "dev" / "kustomization.yaml"
        assert kustomization.exists(), "Dev overlay kustomization.yaml must exist"
    
    def test_staging_overlay_exists(self):
        """Staging overlay must exist."""
        kustomization = OVERLAYS_PATH / "staging" / "kustomization.yaml"
        assert kustomization.exists(), "Staging overlay kustomization.yaml must exist"
    
    def test_prod_overlay_exists(self):
        """Production overlay must exist."""
        kustomization = OVERLAYS_PATH / "prod" / "kustomization.yaml"
        assert kustomization.exists(), "Prod overlay kustomization.yaml must exist"
    
    def test_overlays_reference_base(self):
        """All overlays must reference the base directory."""
        for env in ["dev", "staging", "prod"]:
            kustomization = load_yaml_file(OVERLAYS_PATH / env / "kustomization.yaml")
            resources = kustomization.get("resources", [])
            base_ref = any("base" in r for r in resources)
            assert base_ref, f"{env} overlay must reference base directory"
    
    def test_overlays_have_namespace(self):
        """All overlays must define a namespace."""
        for env in ["dev", "staging", "prod"]:
            kustomization = load_yaml_file(OVERLAYS_PATH / env / "kustomization.yaml")
            assert "namespace" in kustomization, f"{env} overlay must define namespace"
    
    def test_prod_overlay_has_pdb(self):
        """Production overlay must include PodDisruptionBudgets."""
        kustomization = load_yaml_file(OVERLAYS_PATH / "prod" / "kustomization.yaml")
        resources = kustomization.get("resources", [])
        has_pdb = any("pdb" in r.lower() for r in resources)
        assert has_pdb, "Production overlay must include PDB resources"
    
    def test_prod_overlay_has_hpa(self):
        """Production overlay must include HorizontalPodAutoscalers."""
        kustomization = load_yaml_file(OVERLAYS_PATH / "prod" / "kustomization.yaml")
        resources = kustomization.get("resources", [])
        has_hpa = any("hpa" in r.lower() for r in resources)
        assert has_hpa, "Production overlay must include HPA resources"


class TestArgoCDApplications:
    """Property tests for ArgoCD Applications - Requirements 7.3, 7.4"""
    
    def test_dev_application_exists(self):
        """Development ArgoCD application must exist."""
        app_file = ARGOCD_PATH / "dev-application.yaml"
        assert app_file.exists(), "Dev ArgoCD application must exist"
    
    def test_staging_application_exists(self):
        """Staging ArgoCD application must exist."""
        app_file = ARGOCD_PATH / "staging-application.yaml"
        assert app_file.exists(), "Staging ArgoCD application must exist"
    
    def test_prod_application_exists(self):
        """Production ArgoCD application must exist."""
        app_file = ARGOCD_PATH / "prod-application.yaml"
        assert app_file.exists(), "Prod ArgoCD application must exist"


class TestGitOpsSyncPolicy:
    """Property 12: GitOps Sync Policy - Requirements 7.4, 7.6"""
    
    def test_applications_have_automated_sync(self):
        """All ArgoCD applications must have automated sync enabled."""
        for env in ["dev", "staging", "prod"]:
            app_file = ARGOCD_PATH / f"{env}-application.yaml"
            if not app_file.exists():
                continue
            
            app = load_yaml_file(app_file)
            sync_policy = app.get("spec", {}).get("syncPolicy", {})
            automated = sync_policy.get("automated", {})
            
            assert automated, f"{env} application must have automated sync"
    
    def test_applications_have_prune_enabled(self):
        """All ArgoCD applications must have prune enabled."""
        for env in ["dev", "staging", "prod"]:
            app_file = ARGOCD_PATH / f"{env}-application.yaml"
            if not app_file.exists():
                continue
            
            app = load_yaml_file(app_file)
            automated = app.get("spec", {}).get("syncPolicy", {}).get("automated", {})
            
            assert automated.get("prune") is True, (
                f"{env} application must have prune enabled"
            )
    
    def test_applications_have_self_heal_enabled(self):
        """All ArgoCD applications must have selfHeal enabled."""
        for env in ["dev", "staging", "prod"]:
            app_file = ARGOCD_PATH / f"{env}-application.yaml"
            if not app_file.exists():
                continue
            
            app = load_yaml_file(app_file)
            automated = app.get("spec", {}).get("syncPolicy", {}).get("automated", {})
            
            assert automated.get("selfHeal") is True, (
                f"{env} application must have selfHeal enabled"
            )
    
    def test_applications_have_retry_config(self):
        """All ArgoCD applications must have retry configuration."""
        for env in ["dev", "staging", "prod"]:
            app_file = ARGOCD_PATH / f"{env}-application.yaml"
            if not app_file.exists():
                continue
            
            app = load_yaml_file(app_file)
            retry = app.get("spec", {}).get("syncPolicy", {}).get("retry", {})
            
            assert retry, f"{env} application must have retry config"
            assert "limit" in retry, f"{env} application must have retry limit"
            assert "backoff" in retry, f"{env} application must have retry backoff"
    
    def test_applications_have_finalizers(self):
        """All ArgoCD applications must have resource finalizers."""
        for env in ["dev", "staging", "prod"]:
            app_file = ARGOCD_PATH / f"{env}-application.yaml"
            if not app_file.exists():
                continue
            
            app = load_yaml_file(app_file)
            finalizers = app.get("metadata", {}).get("finalizers", [])
            
            assert len(finalizers) > 0, f"{env} application must have finalizers"


class TestEnvironmentSeparation:
    """Property tests for environment separation."""
    
    def test_environments_have_different_namespaces(self):
        """Each environment must have a unique namespace."""
        namespaces = set()
        for env in ["dev", "staging", "prod"]:
            kustomization = load_yaml_file(OVERLAYS_PATH / env / "kustomization.yaml")
            ns = kustomization.get("namespace", "")
            assert ns not in namespaces, f"Namespace {ns} is duplicated"
            namespaces.add(ns)
    
    def test_environments_have_different_image_tags(self):
        """Each environment must use different image tags."""
        tags = {}
        for env in ["dev", "staging", "prod"]:
            kustomization = load_yaml_file(OVERLAYS_PATH / env / "kustomization.yaml")
            images = kustomization.get("images", [])
            for img in images:
                tag = img.get("newTag", "")
                if tag:
                    if img["name"] not in tags:
                        tags[img["name"]] = {}
                    tags[img["name"]][env] = tag
        
        for img_name, env_tags in tags.items():
            unique_tags = set(env_tags.values())
            assert len(unique_tags) == len(env_tags), (
                f"Image {img_name} should have different tags per environment"
            )


if __name__ == "__main__":
    import pytest
    pytest.main([__file__, "-v"])
