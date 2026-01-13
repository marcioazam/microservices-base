---
inclusion: always
---

# Arquitetura de Resiliência - Auth Platform

## Visão Geral
O projeto utiliza Service Mesh (Linkerd 2.16+) com um Kubernetes Operator customizado 
para gerenciar resiliência de forma declarativa. NÃO requer alterações de código nos 
microserviços - apenas configuração YAML.

## Componentes Principais

### 1. Resilience Operator (`platform/resilience-operator/`)
Kubernetes Operator que:
- Observa CRDs `ResiliencePolicy`
- Aplica annotations Linkerd nos Services
- Cria HTTPRoutes para retry/timeout
- Gerencia status e métricas

### 2. ResiliencePolicy CRD
Recurso customizado que define políticas de resiliência:
- Circuit Breaker: Isolamento de falhas
- Retry: Retentativas automáticas
- Timeout: Limites de tempo
- Rate Limit: (futuro)

## Fluxo de Operação

1. DevOps cria `ResiliencePolicy` YAML
2. Operator detecta via watch
3. Annotation Mapper gera annotations Linkerd
4. Controller aplica no Service alvo
5. Linkerd proxy lê annotations e aplica config
6. Status Manager atualiza condições

## Exemplo de Uso

```yaml
apiVersion: resilience.auth-platform.github.com/v1
kind: ResiliencePolicy
metadata:
  name: iam-policy-resilience
spec:
  targetRef:
    name: iam-policy-service
  circuitBreaker:
    enabled: true
    failureThreshold: 5
  retry:
    enabled: true
    maxAttempts: 3
    retryableStatusCodes: "5xx,429"
  timeout:
    enabled: true
    requestTimeout: "30s"
```

## Mapeamento Linkerd

| Feature | Annotation Linkerd |
|---------|-------------------|
| Circuit Breaker | `config.linkerd.io/failure-accrual-consecutive-failures` |
| Retry | `retry.linkerd.io/http`, `retry.linkerd.io/http-status-codes` |
| Timeout | `timeout.linkerd.io/request`, `timeout.linkerd.io/response` |

## Arquivos Chave

- `platform/resilience-operator/` - Código do operator
- `platform/resilience-operator/api/v1/resiliencepolicy_types.go` - CRD types
- `platform/resilience-operator/internal/controller/` - Reconciliation logic
- `platform/resilience-operator/internal/linkerd/annotations.go` - Mapper
- `deploy/kubernetes/service-mesh/examples/` - Exemplos de políticas
- `docs/service-mesh-architecture.md` - Documentação arquitetural
- `docs/runbooks/resilience-operator-runbook.md` - Operações

## Vantagens

✅ Zero código nos microserviços
✅ Configuração declarativa (GitOps)
✅ mTLS automático via Linkerd
✅ Observabilidade integrada
✅ Rollback simples (kubectl delete)
✅ Validação CEL no CRD
✅ Métricas Prometheus nativas

## Comandos Úteis

```bash
# Ver políticas
kubectl get resiliencepolicy -A

# Status detalhado
kubectl describe respol <nome>

# Logs do operator
kubectl logs -n resilience-system -l app.kubernetes.io/name=resilience-operator

# Verificar annotations aplicadas
kubectl get svc <service> -o jsonpath='{.metadata.annotations}'
```


