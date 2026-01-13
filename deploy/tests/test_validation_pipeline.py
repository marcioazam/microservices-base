"""
Property-based tests for Infrastructure Validation Pipeline.
Validates: Requirement 9.7

Property 14: Validation Pipeline Blocking
"""

import yaml
from pathlib import Path

WORKFLOW_PATH = Path(__file__).parent.parent.parent / ".github" / "workflows"


def load_yaml_file(filepath: Path) -> dict:
    """Load a single YAML file."""
    with open(filepath) as f:
        return yaml.safe_load(f)


class TestValidationPipelineBlocking:
    """Property 14: Validation Pipeline Blocking - Requirement 9.7"""
    
    def test_infrastructure_workflow_exists(self):
        """Infrastructure validation workflow must exist."""
        workflow_file = WORKFLOW_PATH / "infrastructure-validation.yaml"
        assert workflow_file.exists(), "Infrastructure validation workflow must exist"
    
    def test_workflow_has_validation_gate(self):
        """Workflow must have a validation gate job."""
        workflow = load_yaml_file(WORKFLOW_PATH / "infrastructure-validation.yaml")
        jobs = workflow.get("jobs", {})
        
        assert "validation-gate" in jobs, "Workflow must have validation-gate job"
    
    def test_validation_gate_depends_on_all_jobs(self):
        """Validation gate must depend on all validation jobs."""
        workflow = load_yaml_file(WORKFLOW_PATH / "infrastructure-validation.yaml")
        jobs = workflow.get("jobs", {})
        gate_job = jobs.get("validation-gate", {})
        
        needs = gate_job.get("needs", [])
        required_jobs = [
            "docker-compose-validation",
            "helm-validation",
            "kubernetes-validation",
            "security-scanning",
            "property-tests"
        ]
        
        for job in required_jobs:
            assert job in needs, f"Validation gate must depend on {job}"
    
    def test_validation_gate_blocks_on_failure(self):
        """Validation gate must block deployment on any failure."""
        workflow = load_yaml_file(WORKFLOW_PATH / "infrastructure-validation.yaml")
        jobs = workflow.get("jobs", {})
        gate_job = jobs.get("validation-gate", {})
        
        steps = gate_job.get("steps", [])
        has_failure_check = False
        
        for step in steps:
            run_cmd = step.get("run", "")
            if "exit 1" in run_cmd and "failed" in run_cmd.lower():
                has_failure_check = True
                break
        
        assert has_failure_check, "Validation gate must exit 1 on failures"
    
    def test_workflow_has_docker_compose_validation(self):
        """Workflow must validate Docker Compose files."""
        workflow = load_yaml_file(WORKFLOW_PATH / "infrastructure-validation.yaml")
        jobs = workflow.get("jobs", {})
        
        assert "docker-compose-validation" in jobs
    
    def test_workflow_has_helm_validation(self):
        """Workflow must validate Helm charts."""
        workflow = load_yaml_file(WORKFLOW_PATH / "infrastructure-validation.yaml")
        jobs = workflow.get("jobs", {})
        
        assert "helm-validation" in jobs
    
    def test_workflow_has_kubernetes_validation(self):
        """Workflow must validate Kubernetes manifests."""
        workflow = load_yaml_file(WORKFLOW_PATH / "infrastructure-validation.yaml")
        jobs = workflow.get("jobs", {})
        
        assert "kubernetes-validation" in jobs
    
    def test_workflow_has_security_scanning(self):
        """Workflow must include security scanning."""
        workflow = load_yaml_file(WORKFLOW_PATH / "infrastructure-validation.yaml")
        jobs = workflow.get("jobs", {})
        
        assert "security-scanning" in jobs
    
    def test_workflow_has_property_tests(self):
        """Workflow must run property-based tests."""
        workflow = load_yaml_file(WORKFLOW_PATH / "infrastructure-validation.yaml")
        jobs = workflow.get("jobs", {})
        
        assert "property-tests" in jobs


class TestWorkflowTriggers:
    """Tests for workflow trigger configuration."""
    
    def test_workflow_triggers_on_deploy_changes(self):
        """Workflow must trigger on deploy directory changes."""
        workflow = load_yaml_file(WORKFLOW_PATH / "infrastructure-validation.yaml")
        
        # 'on' is a reserved word in YAML, may be parsed as True
        on_config = workflow.get("on", workflow.get(True, {}))
        if isinstance(on_config, dict):
            push_config = on_config.get("push", {})
            pr_config = on_config.get("pull_request", {})
            
            push_paths = push_config.get("paths", []) if isinstance(push_config, dict) else []
            pr_paths = pr_config.get("paths", []) if isinstance(pr_config, dict) else []
            
            deploy_in_push = any("deploy" in p for p in push_paths)
            deploy_in_pr = any("deploy" in p for p in pr_paths)
            
            assert deploy_in_push or deploy_in_pr, (
                "Workflow must trigger on deploy directory changes"
            )
        else:
            # If on_config is not a dict, the workflow file structure is different
            pass  # Skip this check


if __name__ == "__main__":
    import pytest
    pytest.main([__file__, "-v"])
