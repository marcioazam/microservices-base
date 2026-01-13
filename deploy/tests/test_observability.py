"""
Property-based tests for Observability Stack.
Validates: Requirements 8.1-8.7

Property 13: Observability Dashboard Provisioning
"""

import yaml
from pathlib import Path

OBSERVABILITY_PATH = Path(__file__).parent.parent / "kubernetes" / "observability"


def load_yaml_docs(filepath: Path) -> list[dict]:
    """Load all YAML documents from a file."""
    with open(filepath) as f:
        return list(yaml.safe_load_all(f))


class TestPrometheusConfiguration:
    """Property tests for Prometheus - Requirements 8.1, 8.5"""
    
    def test_prometheus_config_exists(self):
        """Prometheus configuration must exist."""
        config_file = OBSERVABILITY_PATH / "prometheus-config.yaml"
        assert config_file.exists(), "Prometheus config must exist"
    
    def test_prometheus_has_remote_write(self):
        """Prometheus must have remote_write configured."""
        docs = load_yaml_docs(OBSERVABILITY_PATH / "prometheus-config.yaml")
        configmaps = [d for d in docs if d and d.get("kind") == "ConfigMap"]
        
        for cm in configmaps:
            data = cm.get("data", {})
            prometheus_yml = data.get("prometheus.yml", "")
            if prometheus_yml:
                assert "remote_write" in prometheus_yml, (
                    "Prometheus must have remote_write configured"
                )
    
    def test_prometheus_has_alerting(self):
        """Prometheus must have alerting configured."""
        docs = load_yaml_docs(OBSERVABILITY_PATH / "prometheus-config.yaml")
        configmaps = [d for d in docs if d and d.get("kind") == "ConfigMap"]
        
        for cm in configmaps:
            data = cm.get("data", {})
            prometheus_yml = data.get("prometheus.yml", "")
            if prometheus_yml:
                assert "alerting" in prometheus_yml, (
                    "Prometheus must have alerting configured"
                )


class TestPrometheusRules:
    """Property tests for Prometheus alerting rules - Requirement 8.5"""
    
    def test_prometheus_rules_exist(self):
        """Prometheus alerting rules must exist."""
        rules_file = OBSERVABILITY_PATH / "prometheus-rules.yaml"
        assert rules_file.exists(), "Prometheus rules must exist"
    
    def test_slo_alerts_defined(self):
        """SLO-based alerts must be defined."""
        docs = load_yaml_docs(OBSERVABILITY_PATH / "prometheus-rules.yaml")
        configmaps = [d for d in docs if d and d.get("kind") == "ConfigMap"]
        
        slo_alerts_found = False
        for cm in configmaps:
            data = cm.get("data", {})
            for key, value in data.items():
                if "slo" in value.lower():
                    slo_alerts_found = True
                    break
        
        assert slo_alerts_found, "SLO-based alerts must be defined"


class TestGrafanaDashboards:
    """Property 13: Observability Dashboard Provisioning - Requirements 8.2, 8.7"""
    
    def test_grafana_dashboards_exist(self):
        """Grafana dashboard provisioning must exist."""
        dashboards_file = OBSERVABILITY_PATH / "grafana-dashboards.yaml"
        assert dashboards_file.exists(), "Grafana dashboards must exist"
    
    def test_grafana_has_datasources(self):
        """Grafana must have datasources configured."""
        docs = load_yaml_docs(OBSERVABILITY_PATH / "grafana-dashboards.yaml")
        configmaps = [d for d in docs if d and d.get("kind") == "ConfigMap"]
        
        datasources_found = False
        for cm in configmaps:
            if "datasources" in cm.get("metadata", {}).get("name", ""):
                datasources_found = True
                break
        
        assert datasources_found, "Grafana datasources must be configured"
    
    def test_grafana_has_prometheus_datasource(self):
        """Grafana must have Prometheus datasource."""
        docs = load_yaml_docs(OBSERVABILITY_PATH / "grafana-dashboards.yaml")
        configmaps = [d for d in docs if d and d.get("kind") == "ConfigMap"]
        
        prometheus_found = False
        for cm in configmaps:
            data = cm.get("data", {})
            datasources_yaml = data.get("datasources.yaml", "")
            if "prometheus" in datasources_yaml.lower():
                prometheus_found = True
                break
        
        assert prometheus_found, "Grafana must have Prometheus datasource"
    
    def test_grafana_has_loki_datasource(self):
        """Grafana must have Loki datasource."""
        docs = load_yaml_docs(OBSERVABILITY_PATH / "grafana-dashboards.yaml")
        configmaps = [d for d in docs if d and d.get("kind") == "ConfigMap"]
        
        loki_found = False
        for cm in configmaps:
            data = cm.get("data", {})
            datasources_yaml = data.get("datasources.yaml", "")
            if "loki" in datasources_yaml.lower():
                loki_found = True
                break
        
        assert loki_found, "Grafana must have Loki datasource"
    
    def test_grafana_has_tempo_datasource(self):
        """Grafana must have Tempo datasource."""
        docs = load_yaml_docs(OBSERVABILITY_PATH / "grafana-dashboards.yaml")
        configmaps = [d for d in docs if d and d.get("kind") == "ConfigMap"]
        
        tempo_found = False
        for cm in configmaps:
            data = cm.get("data", {})
            datasources_yaml = data.get("datasources.yaml", "")
            if "tempo" in datasources_yaml.lower():
                tempo_found = True
                break
        
        assert tempo_found, "Grafana must have Tempo datasource"


class TestLokiConfiguration:
    """Property tests for Loki - Requirement 8.3"""
    
    def test_loki_config_exists(self):
        """Loki configuration must exist."""
        config_file = OBSERVABILITY_PATH / "loki-config.yaml"
        assert config_file.exists(), "Loki config must exist"
    
    def test_loki_has_retention(self):
        """Loki must have retention configured."""
        docs = load_yaml_docs(OBSERVABILITY_PATH / "loki-config.yaml")
        configmaps = [d for d in docs if d and d.get("kind") == "ConfigMap"]
        
        retention_found = False
        for cm in configmaps:
            data = cm.get("data", {})
            loki_yaml = data.get("loki.yaml", "")
            if "retention" in loki_yaml:
                retention_found = True
                break
        
        assert retention_found, "Loki must have retention configured"


class TestTempoConfiguration:
    """Property tests for Tempo - Requirement 8.4"""
    
    def test_tempo_config_exists(self):
        """Tempo configuration must exist."""
        config_file = OBSERVABILITY_PATH / "tempo-config.yaml"
        assert config_file.exists(), "Tempo config must exist"
    
    def test_tempo_has_otlp_receiver(self):
        """Tempo must have OTLP receiver configured."""
        docs = load_yaml_docs(OBSERVABILITY_PATH / "tempo-config.yaml")
        configmaps = [d for d in docs if d and d.get("kind") == "ConfigMap"]
        
        otlp_found = False
        for cm in configmaps:
            data = cm.get("data", {})
            tempo_yaml = data.get("tempo.yaml", "")
            if "otlp" in tempo_yaml:
                otlp_found = True
                break
        
        assert otlp_found, "Tempo must have OTLP receiver"


class TestAlertmanagerConfiguration:
    """Property tests for Alertmanager - Requirement 8.6"""
    
    def test_alertmanager_config_exists(self):
        """Alertmanager configuration must exist."""
        config_file = OBSERVABILITY_PATH / "alertmanager-config.yaml"
        assert config_file.exists(), "Alertmanager config must exist"
    
    def test_alertmanager_has_receivers(self):
        """Alertmanager must have receivers configured."""
        docs = load_yaml_docs(OBSERVABILITY_PATH / "alertmanager-config.yaml")
        configmaps = [d for d in docs if d and d.get("kind") == "ConfigMap"]
        
        receivers_found = False
        for cm in configmaps:
            data = cm.get("data", {})
            alertmanager_yml = data.get("alertmanager.yml", "")
            if "receivers" in alertmanager_yml:
                receivers_found = True
                break
        
        assert receivers_found, "Alertmanager must have receivers"
    
    def test_alertmanager_has_pagerduty(self):
        """Alertmanager must have PagerDuty integration."""
        docs = load_yaml_docs(OBSERVABILITY_PATH / "alertmanager-config.yaml")
        configmaps = [d for d in docs if d and d.get("kind") == "ConfigMap"]
        
        pagerduty_found = False
        for cm in configmaps:
            data = cm.get("data", {})
            alertmanager_yml = data.get("alertmanager.yml", "")
            if "pagerduty" in alertmanager_yml.lower():
                pagerduty_found = True
                break
        
        assert pagerduty_found, "Alertmanager must have PagerDuty integration"
    
    def test_alertmanager_has_slack(self):
        """Alertmanager must have Slack integration."""
        docs = load_yaml_docs(OBSERVABILITY_PATH / "alertmanager-config.yaml")
        configmaps = [d for d in docs if d and d.get("kind") == "ConfigMap"]
        
        slack_found = False
        for cm in configmaps:
            data = cm.get("data", {})
            alertmanager_yml = data.get("alertmanager.yml", "")
            if "slack" in alertmanager_yml.lower():
                slack_found = True
                break
        
        assert slack_found, "Alertmanager must have Slack integration"


if __name__ == "__main__":
    import pytest
    pytest.main([__file__, "-v"])
