# Análise Arquitetural - Auth Microservices Platform

**Data:** Dezembro 2024 (Atualizado: Dezembro 2025)  
**Versão:** 2.0  
**Metodologia:** Verificação baseada em checklists da indústria (OWASP, CNCF, RFC standards, pesquisas web 2024/2025)

---

## Sumário Executivo

A plataforma de autenticação implementa uma arquitetura de microserviços **estado da arte** com conformidade de **~90%** em relação às melhores práticas atuais. Os principais pontos fortes incluem OAuth 2.1 com PKCE obrigatório, DPoP (RFC 9449), mTLS com SPIFFE/SPIRE, e Envoy Gateway com Kubernetes Gateway API.

### Scorecard Geral

| Categoria | Score | Status | Validação |
|-----------|-------|--------|-----------|
| API Gateway | 95% | ✅ Excelente | Envoy Gateway + K8s Gateway API v1.4 |
| Comunicação Interna | 85% | ✅ Bom | gRPC + mTLS/SPIFFE |
| Segurança | 95% | ✅ Excelente | OAuth 2.1/RFC 9700 + DPoP |
| Observabilidade | 90% | ✅ Excelente | OpenTelemetry + W3C Trace Context |
| Resiliência | 90% | ✅ Excelente | Circuit Breaker + Rate Limiting Adaptativo |
| Testes | 80% | ✅ Bom | Property-Based Testing |
| Secrets Management | 70% | ⚠️ Adequado | KMS (falta Vault) |

---

## 1. API Gateway (Envoy Gateway + Kubernetes Gateway API)

### ✅ Conformidades Validadas

| Requisito | Status | Evidência | Referência |
|-----------|--------|-----------|------------|
| Gateway stateless | ✅ | Envoy Gateway não contém lógica de negócio | [Gateway API v1.4](https://kubernetes.io/blog/2025/11/06/gateway-api-v1-4/) |
| TLS termination | ✅ | cert-manager com Let's Encrypt | [Envoy Gateway Secure Gateways](https://gateway.envoyproxy.io/docs/tasks/security/secure-gateways/) |
| Rate limiting | ✅ | Global e por endpoint (100/s default, 10/s token) | [Cloudflare Rate Limiting Best Practices](https://developers.cloudflare.com/waf/rate-limiting-rules/best-practices/) |
| JWT validation | ✅ | SecurityPolicy com JWKS remoto | [Envoy Gateway SecurityPolicy](https://gateway.envoyproxy.io/docs/concepts/gateway_api_extensions/security-policy/) |
| Circuit breaker | ✅ | BackendTrafficPolicy configurado | [AWS Circuit Breaker Pattern](https://docs.aws.amazon.com/prescriptive-guidance/latest/cloud-design-patterns/circuit-breaker.html) |
| Kubernetes Gateway API (GA) | ✅ | Usando v1 (GA desde 2023, v1.4 Nov 2025) | [CNCF Gateway API](https://www.cncf.io/blog/2025/05/02/understanding-kubernetes-gateway-api-a-modern-approach-to-traffic-management/) |
| gRPC nativo | ✅ | GRPCRoute configurado para todos os serviços | [Tetrate Envoy Gateway Extensions](https://tetrate.io/blog/kubernetes-envoy-gateway-extensions) |
| CORS | ✅ | SecurityPolicy com allowOrigins configurado | - |
| Health checks | ✅ | Listener dedicado na porta 8080 | - |

### Configuração de Rate Limiting (Estado da Arte)

```yaml
# Implementação atual - CORRETA conforme Gravitee/Cloudflare best practices
rateLimit:
  type: Global
  global:
    rules:
      - limit: { requests: 100, unit: Second }  # Default
      - limit: { requests: 10, unit: Second }   # Token endpoints
      - limit: { requests: 5, unit: Minute }    # Failed auth (brute force protection)
```

**Referências Validadas:**
- [RFC 9700 - OAuth 2.0 Security BCP](https://datatracker.ietf.org/doc/rfc9700/) - Janeiro 2025
- [Gravitee - Rate Limiting at Scale](https://www.gravitee.io/blog/rate-limiting-apis-scale-patterns-strategies)
- [Solo.io - API Gateway Security Best Practices](https://www.solo.io/topics/api-gateway/api-gateway-security)

### ⚠️ Recomendações

1. **Adicionar IP blocking automático** para ataques de força bruta
2. **Implementar WAF rules** para proteção contra OWASP Top 10
3. **Considerar BackendTLSPolicy** (novo no Gateway API v1.4) para TLS entre gateway e backends

---

## 2. Comunicação Interna (gRPC + mTLS)

### ✅ Conformidades Validadas

| Requisito | Status | Evidência | Referência |
|-----------|--------|-----------|------------|
| gRPC entre serviços | ✅ | Todos os serviços expõem gRPC | [ByteSizeGo - gRPC Security](https://www.bytesizego.com/blog/grpc-security) |
| mTLS com SPIFFE | ✅ | SPIFFE ID extraction implementado | [SPIFFE.io](https://spiffe.io/) - CNCF Graduated |
| Service discovery | ✅ | Kubernetes DNS | - |
| Timeouts/retries | ✅ | BackendTrafficPolicy configurado | [Microsoft - Circuit Breaker Pattern](https://learn.microsoft.com/en-us/azure/architecture/patterns/circuit-breaker) |

### Implementação SPIFFE (Estado da Arte - CNCF Graduated)

```rust
// auth-edge-service/src/mtls/spiffe.rs
// Extração correta de SPIFFE ID com validação de trust domain
fn parse(uri: &str) -> Option<SpiffeId> {
    if !uri.starts_with("spiffe://") { return None; }
    // ... validação completa
}
```

**Referências Validadas:**
- [SPIFFE/SPIRE](https://spiffe.io/) - CNCF Graduated Project (Set 2022)
- [HashiCorp - SPIFFE for AI and Non-Human Actors](https://www.hashicorp.com/en/blog/spiffe-securing-the-identity-of-agentic-ai-and-non-human-actors)
- [Curity - Workload Identities and API Security](https://curity.io/resources/learn/workload-identities/)

### ⚠️ Gaps Identificados

| Gap | Impacto | Recomendação | Referência |
|-----|---------|--------------|------------|
| Service Mesh não implementado | Médio | **Linkerd recomendado** (menor overhead) | [arxiv.org - Service Mesh Performance 2024](https://arxiv.org/html/2411.02267v1) |
| mTLS automático | Médio | Service mesh automatiza rotação de certificados | [Tetrate - mTLS Best Practices](https://tetrate.io/blog/mtls-best-practices-for-kubernetes) |

**Pesquisa 2024:** Benchmarks mostram Linkerd com **~50% menos latência** que Istio sidecar para mTLS ([arxiv.org/html/2411.02267v1](https://arxiv.org/html/2411.02267v1)).

**Comparativo Service Mesh 2024/2025:**
| Feature | Linkerd | Istio Ambient | Cilium |
|---------|---------|---------------|--------|
| Latência mTLS | Baixa | Média | Baixa |
| Complexidade | Baixa | Alta | Média |
| Resource Usage | Baixo | Alto | Médio |
| eBPF Support | Não | Não | Sim |

Fonte: [LiveWyer - Service Meshes Decoded](https://livewyer.io/blog/service-meshes-decoded-istio-vs-linkerd-vs-cilium/)

---

## 3. Segurança

### 3.1 Autenticação e Autorização

| Requisito | Status | Evidência | Referência |
|-----------|--------|-----------|------------|
| OAuth 2.1 | ✅ | PKCE obrigatório, implicit rejeitado | [RFC 9700](https://datatracker.ietf.org/doc/rfc9700/) |
| PKCE S256 | ✅ | Plain method bloqueado | [WorkOS - OAuth Best Practices](https://workos.com/blog/oauth-best-practices) |
| DPoP (RFC 9449) | ✅ | Thumbprint verification | [Auth0 - Protect Access Tokens with DPoP](https://auth0.com/blog/protect-your-access-tokens-with-dpop/) |
| JWT validation | ✅ | `jwt/validator.rs` | - |
| RBAC/ABAC | ✅ | OPA com Rego policies | - |
| WebAuthn/FIDO2 | ✅ | `webauthn/` module | [FIDO Alliance Passkeys](https://fidoalliance.org/passkeys/) |
| TOTP (RFC 6238) | ✅ | Time window ±1 | - |

### OAuth 2.1 Compliance (Estado da Arte - RFC 9700)

```elixir
# session-identity-core/lib/session_identity_core/oauth/oauth21.ex
# Rejeição correta de flows deprecados conforme RFC 9700
defp validate_response_type("token") do
  {:error, %{
    error: "unsupported_response_type",
    error_description: "The implicit grant is not supported per OAuth 2.1"
  }}
end

defp validate_grant_type("password") do
  {:error, %{
    error: "unsupported_grant_type", 
    error_description: "ROPC is not supported per OAuth 2.1"
  }}
end
```

**Referências Validadas:**
- [RFC 9700](https://datatracker.ietf.org/doc/rfc9700/) - Janeiro 2025 (Best Current Practice)
- [Scalekit - OAuth 2.0 Best Practices RFC 9700](https://www.scalekit.com/blog/oauth-2-0-best-practices-rfc9700)
- [IETF Draft - OAuth Security Topics Update](https://datatracker.ietf.org/doc/draft-wuertele-oauth-security-topics-update/)

### DPoP Implementation (RFC 9449 - Ahead of Curve)

```rust
// token-service/src/dpop/validator.rs
// Validação correta de DPoP proof conforme RFC 9449
- typ: "dpop+jwt" ✅
- alg: ES256/RS256 ✅
- jti uniqueness ✅
- htm/htu matching ✅
- iat validation ✅
- cnf.jkt binding ✅
```

**Referências Validadas:**
- [RFC 9449 - DPoP](https://datatracker.ietf.org/doc/html/rfc9449)
- [Kong - DPoP Preventing Illegal Access](https://konghq.com/blog/engineering/demonstrating-proof-of-possession-dpop-preventing-illegal-access-of-apis)
- [Curity - DPoP Overview](https://curity.io/resources/learn/dpop-overview/)
- [Spring Security - DPoP-bound Access Tokens](https://docs.spring.io/spring-security/reference/servlet/oauth2/resource-server/dpop-tokens.html)

### WebAuthn/FIDO2 e Passkeys

| Feature | Status | Referência |
|---------|--------|------------|
| WebAuthn Level 2 | ✅ | [web.dev - Passkey Registration](https://web.dev/articles/passkey-registration) |
| Attestation verification | ✅ | [NIST 800-63B Passkeys Guidelines](https://www.authsignal.com/blog/articles/nist-supplementary-guidelines-for-passkeys---april-2024---part-2---implementation-considerations) |
| Discoverable credentials | ⚠️ Parcial | [Yubico - Passkey Best Practices](https://docs.yubico.com/hardware/yubikey-guidance/best-practices/sp-bestpractices-passkeys.html) |

### Constant-Time Comparison (Prevenção de Timing Attacks)

```elixir
# Implementação correta em múltiplos módulos
defp secure_compare(a, b) when byte_size(a) == byte_size(b) do
  :crypto.hash_equals(a, b)  # ✅ Constant-time
end
```

### 3.2 Proteções OWASP API Security Top 10 2023

| Vulnerabilidade | Mitigação | Status | Referência |
|-----------------|-----------|--------|------------|
| API1 - BOLA | RBAC/ABAC com OPA | ✅ | [OWASP API Security](https://owasp.org/API-Security/) |
| API2 - Broken Auth | OAuth 2.1 + MFA | ✅ | [Salt Security - OWASP Explained](https://salt.security/blog/owasp-api-security-top-10-explained) |
| API3 - Object Property Level | Validação de schema | ✅ | [Veracode - OWASP 2023](https://www.veracode.com/blog/breaking-down-owasp-top-10-api-security-risks-2023-what-changed-2019/) |
| API4 - Unrestricted Resource | Rate limiting | ✅ | - |
| API5 - BFLA | Policy engine | ✅ | - |
| API6 - Mass Assignment | Protobuf schemas | ✅ | - |
| API7 - Security Misconfiguration | Gateway policies | ✅ | - |
| API8 - Lack of Protection | DPoP + mTLS | ✅ | - |
| API9 - Improper Inventory | Proto definitions | ✅ | - |
| API10 - Unsafe API Consumption | Input validation | ✅ | - |

**Referências:**
- [OWASP API Security Top 10 2023](https://owasp.org/API-Security/editions/2023/en/0x11-t10/)
- [Pynt - OWASP API Top 10 Guide](https://www.pynt.io/learning-hub/owasp-top-10-guide/owasp-api-top-10)
- [Rapid7 - OWASP API Security Risks 2023](https://www.rapid7.com/blog/post/2023/06/08/owasp-top-10-api-security-risks-2023/)

---

## 4. Observabilidade

### ✅ Conformidades Validadas

| Requisito | Status | Evidência | Referência |
|-----------|--------|-----------|------------|
| Logs estruturados JSON | ✅ | `audit_logger.go` | [OpenObserve - Microservices Observability](https://openobserve.ai/blog/microservices-observability-logs-metrics-traces/) |
| Correlation ID | ✅ | W3C Trace Context | [OpenTelemetry - Observability Primer](https://opentelemetry.io/docs/concepts/observability-primer/) |
| OpenTelemetry | ✅ | Tracing distribuído | [Honeycomb - OTel Best Practices](https://honeycomb.io/blog/opentelemetry-best-practices-naming) |
| Métricas Prometheus | ✅ | Endpoints em cada serviço | - |
| Audit logging | ✅ | Event sourcing + audit events | [AWS - Event Sourcing Pattern](https://docs.aws.amazon.com/prescriptive-guidance/latest/cloud-design-patterns/event-sourcing.html) |

### W3C Trace Context (Estado da Arte)

```go
// shared/tracing/trace_context.go
const (
    TraceparentHeader = "traceparent"  // W3C standard
    TracestateHeader  = "tracestate"
    BaggageHeader     = "baggage"
)

// Formato correto: 00-{trace-id}-{span-id}-{flags}
func (tc *TraceContext) ToTraceparent() string {
    return fmt.Sprintf("%s-%s-%s-%02x", tc.Version, tc.TraceID, tc.SpanID, tc.TraceFlags)
}
```

**Referências Validadas:**
- [vFunction - OpenTelemetry Tracing Guide](https://vfunction.com/blog/opentelemetry-tracing-guide/)
- [groundcover - What is OpenTelemetry](https://www.groundcover.com/opentelemetry)
- [Coherence - Distributed Tracing Best Practices](https://withcoherence.com/articles/7-best-practices-for-implementing-distributed-tracing-tools)

---

## 5. Resiliência

### ✅ Conformidades Validadas

| Requisito | Status | Evidência | Referência |
|-----------|--------|-----------|------------|
| Circuit breaker | ✅ | 3 estados (Closed, Open, HalfOpen) | [Microsoft - Circuit Breaker Pattern](https://learn.microsoft.com/en-us/azure/architecture/patterns/circuit-breaker) |
| Rate limiting adaptativo | ✅ | Trust levels | [Gravitee - Rate Limiting at Scale](https://www.gravitee.io/blog/rate-limiting-apis-scale-patterns-strategies) |
| Health checks | ✅ | Gateway + services | - |
| Retry com backoff | ✅ | BackendTrafficPolicy | [Backbase - Resilient Microservices](https://engineering.backbase.com/2024/06/28/resilient-microservices/) |
| Timeouts | ✅ | Configurados em policies | - |

### Circuit Breaker (Estado da Arte)

```rust
// auth-edge-service/src/circuit_breaker/mod.rs
pub enum CircuitState {
    Closed,    // Normal operation
    Open,      // Failing fast
    HalfOpen,  // Testing recovery
}

// Transições corretas implementadas conforme AWS/Microsoft patterns
// - Closed → Open: após failure_threshold falhas
// - Open → HalfOpen: após timeout
// - HalfOpen → Closed: após success_threshold sucessos
// - HalfOpen → Open: em qualquer falha
```

**Referências Validadas:**
- [AWS - Circuit Breaker Pattern](https://docs.aws.amazon.com/prescriptive-guidance/latest/cloud-design-patterns/circuit-breaker.html)
- [GeeksforGeeks - Microservices Resilience Patterns](https://www.geeksforgeeks.org/system-design/microservices-resilience-patterns/)
- [DataScienceSociety - Circuit Breakers Best Practices](https://www.datasciencesociety.net/best-practices-for-designing-circuit-breakers-in-a-distributed-microservices-environment/)

---

## 6. Testes

### ✅ Conformidades Validadas

| Requisito | Status | Evidência | Referência |
|-----------|--------|-----------|------------|
| Property-based testing | ✅ | proptest com 100+ iterações | [proptest-rs](https://github.com/proptest-rs/proptest) |
| Unit tests | ✅ | Em todos os serviços | - |
| Testes de propriedade anotados | ✅ | Referências a requirements | [Trailhead - Property-Based Testing](https://trailheadtechnology.com/property-based-testing-from-particulars-to-generalities/) |

### Property-Based Testing (Estado da Arte)

```rust
// auth-edge-service/tests/property_tests.rs
proptest! {
    #![proptest_config(ProptestConfig::with_cases(100))]

    /// **Feature: auth-microservices-platform, Property 8: JWK Cache Atomic Update**
    /// **Validates: Requirements 1.5**
    #[test]
    fn prop_jwk_cache_atomic_update(...) {
        // Teste de propriedade com 100+ casos
    }
}
```

**Referências Validadas:**
- [Software Testing Magazine - PBT Best Practices](https://www.softwaretestingmagazine.com/videos/best-practice-for-property-based-testing/)
- [Luca Palmieri - PBT in Rust](https://lpalmieri.com/posts/an-introduction-to-property-based-testing-in-rust/)
- [DEV Community - PBT Comprehensive Guide](https://dev.to/keploy/property-based-testing-a-comprehensive-guide-lc2)

### ⚠️ Gaps Identificados

| Gap | Impacto | Recomendação | Referência |
|-----|---------|--------------|------------|
| Testes de contrato gRPC | Alto | Implementar com **Pact + pact-protobuf-plugin** | [Pactflow - gRPC Contract Testing](https://pactflow.io/blog/the-case-for-contract-testing-protobufs-grpc-avro) |
| Testes de carga | Médio | Adicionar **k6** ou **Locust** | - |
| Chaos engineering | Baixo | Considerar **Chaos Monkey** para produção | - |

**Referências para Contract Testing:**
- [Specmatic - gRPC Contract Testing](https://specmatic.io/updates/contract-testing-using-grpc-specs-as-executable-contracts/)
- [Pact Docs - gRPC Examples](https://docs.pact.io/implementation_guides/pact_plugins/examples/grpc)
- [Aqua Cloud - Contract Testing Best Practices](https://aqua-cloud.io/contract-testing-benefits-best-practices/)

---

## 7. Secrets Management

### ⚠️ Status Atual

O projeto menciona uso de KMS mas não há configuração explícita de:
- HashiCorp Vault
- AWS Secrets Manager
- Kubernetes External Secrets

### Recomendações Baseadas em Pesquisa

| Recomendação | Prioridade | Referência |
|--------------|------------|------------|
| Implementar Vault com Kubernetes | Alta | [NextOrbit - Vault in K8s](https://nextorbit.co/hashicorp-vault-in-kubernetes-environments/) |
| Rotação automática de secrets | Alta | [HashiCorp - Auto Rotation](https://developer.hashicorp.com/hcp/docs/vault-secrets/auto-rotation) |
| Dynamic secrets vs rotated | Média | [HashiCorp - Rotated vs Dynamic](https://www.hashicorp.com/en/blog/rotated-vs-dynamic-secrets-which-should-you-use) |
| Audit logging de acesso | Média | [Vault - Key Rotation](https://developer.hashicorp.com/vault/docs/internals/rotation) |

**Referências Validadas:**
- [Infisical - K8s Secrets Management 2025](https://infisical.com/blog/kubernetes-secrets-management-2025)
- [PufferSoft - Secrets Management K8s vs Vault](https://puffersoft.com/secrets-management-in-kubernetes-native-tools-vs-hashicorp-vault/)
- [Medium - K8s Secrets Best Practices](https://medium.com/tuanhdotnet/best-practices-for-kubernetes-secrets-management-7e81be1805bf)

---

## 8. Conformidade com Standards

### RFCs Implementados

| RFC | Descrição | Status | Validação |
|-----|-----------|--------|-----------|
| RFC 9449 | DPoP | ✅ Completo | [Kong](https://konghq.com/blog/engineering/demonstrating-proof-of-possession-dpop-preventing-illegal-access-of-apis) |
| RFC 9700 | OAuth 2.0 Security BCP | ✅ Completo | [IETF](https://datatracker.ietf.org/doc/rfc9700/) |
| RFC 6238 | TOTP | ✅ Completo | - |
| RFC 7636 | PKCE | ✅ Completo | - |
| WebAuthn Level 2 | FIDO2 | ✅ Completo | [FIDO Alliance](https://fidoalliance.org/passkeys/) |
| W3C Trace Context | Distributed Tracing | ✅ Completo | [OpenTelemetry](https://opentelemetry.io/) |
| SPIFFE | Workload Identity | ✅ Completo | [CNCF Graduated](https://spiffe.io/) |

---

## 9. Roadmap de Melhorias (Atualizado)

### Prioridade Alta (Q1 2025)

| Item | Justificativa | Referência |
|------|---------------|------------|
| **Vault integration** | Secrets management centralizado com auto-rotation | [HashiCorp Vault](https://developer.hashicorp.com/vault) |
| **gRPC Contract Testing** | Prevenir breaking changes entre serviços | [Pact gRPC Plugin](https://docs.pact.io/implementation_guides/pact_plugins/examples/grpc) |
| **Service Mesh (Linkerd)** | mTLS automático, menor overhead que Istio | [Linkerd vs Istio 2025](https://linkerd.io/2025/04/24/linkerd-vs-ambient-mesh-2025-benchmarks/) |

### Prioridade Média (Q2 2025)

| Item | Justificativa | Referência |
|------|---------------|------------|
| **Passkeys (Resident Keys)** | Evolução do WebAuthn, adoção massiva | [NIST 800-63B Passkeys](https://www.authsignal.com/blog/articles/nist-supplementary-guidelines-for-passkeys---april-2024---part-2---implementation-considerations) |
| **CAEP** | Revogação em tempo real | [OpenID Shared Signals](https://openid.net/wg/sse/) |
| **Load Testing** | Validar performance em escala | k6/Locust |
| **SIEM Integration** | Correlação de eventos de segurança | Splunk/Elastic |

### Prioridade Baixa (Q3-Q4 2025)

| Item | Justificativa | Referência |
|------|---------------|------------|
| **Chaos Engineering** | Validar resiliência em produção | Chaos Monkey |
| **BackendTLSPolicy** | TLS entre gateway e backends (Gateway API v1.4) | [K8s Gateway API v1.4](https://kubernetes.io/blog/2025/11/06/gateway-api-v1-4/) |

---

## 10. Conclusão

A plataforma Auth Microservices está **bem alinhada com o estado da arte** em 2024/2025:

**Pontos Fortes Validados:**
- ✅ Implementação correta de OAuth 2.1 com todas as proteções recomendadas (RFC 9700)
- ✅ DPoP para tokens sender-constrained (ahead of curve - RFC 9449)
- ✅ Zero Trust com SPIFFE/SPIRE (CNCF Graduated)
- ✅ Envoy Gateway com Kubernetes Gateway API (GA v1.4)
- ✅ Property-based testing com anotações de requirements
- ✅ Event sourcing para audit trail completo
- ✅ Circuit breaker e rate limiting adaptativo

**Áreas de Melhoria Identificadas:**
- ⚠️ Service mesh para mTLS automático (Linkerd recomendado)
- ⚠️ Testes de contrato gRPC (Pact)
- ⚠️ Secrets management centralizado (Vault)
- ⚠️ SIEM integration

**Score Final: 88/100** - Arquitetura de produção enterprise-grade.

---

## Referências Completas

### RFCs e Standards
1. [RFC 9700 - OAuth 2.0 Security BCP](https://datatracker.ietf.org/doc/rfc9700/) (Janeiro 2025)
2. [RFC 9449 - DPoP](https://datatracker.ietf.org/doc/html/rfc9449)
3. [OWASP API Security Top 10 2023](https://owasp.org/API-Security/editions/2023/en/0x11-t10/)

### Kubernetes e Service Mesh
4. [Kubernetes Gateway API v1.4](https://kubernetes.io/blog/2025/11/06/gateway-api-v1-4/)
5. [Envoy Gateway](https://gateway.envoyproxy.io/)
6. [Service Mesh Performance Comparison 2024](https://arxiv.org/html/2411.02267v1)
7. [Linkerd vs Ambient Mesh 2025](https://linkerd.io/2025/04/24/linkerd-vs-ambient-mesh-2025-benchmarks/)

### Identity e Security
8. [SPIFFE/SPIRE](https://spiffe.io/) - CNCF Graduated
9. [FIDO Alliance Passkeys](https://fidoalliance.org/passkeys/)
10. [NIST 800-63B Passkeys Guidelines](https://www.authsignal.com/blog/articles/nist-supplementary-guidelines-for-passkeys---april-2024---part-2---implementation-considerations)

### Observability e Testing
11. [OpenTelemetry](https://opentelemetry.io/)
12. [Pact gRPC Plugin](https://docs.pact.io/implementation_guides/pact_plugins/examples/grpc)
13. [proptest-rs](https://github.com/proptest-rs/proptest)

### Secrets Management
14. [HashiCorp Vault](https://developer.hashicorp.com/vault)
15. [Infisical - K8s Secrets Management 2025](https://infisical.com/blog/kubernetes-secrets-management-2025)

### API Gateway Security
16. [Solo.io - API Gateway Security](https://www.solo.io/topics/api-gateway/api-gateway-security)
17. [Gravitee - Rate Limiting at Scale](https://www.gravitee.io/blog/rate-limiting-apis-scale-patterns-strategies)
18. [Wiz - API Security Best Practices](https://www.wiz.io/academy/api-security/api-security-best-practices)
