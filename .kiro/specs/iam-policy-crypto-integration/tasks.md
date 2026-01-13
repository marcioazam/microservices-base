# Implementation Plan: IAM Policy Crypto Integration

## Overview

Este plano implementa a integração do `iam-policy` com o `crypto-service` em Go. A implementação segue uma abordagem incremental, começando pelo cliente gRPC, depois criptografia de cache, assinatura de decisões, e finalmente observabilidade.

## Tasks

- [x] 1. Configuração e Setup Inicial
  - [x] 1.1 Adicionar dependências ao go.mod
    - Adicionar `google.golang.org/grpc` para cliente gRPC
    - Adicionar proto gerado do crypto-service
    - _Requirements: 1.1_
  - [x] 1.2 Criar estrutura de diretórios
    - Criar `internal/crypto/` para componentes de criptografia
    - _Requirements: 1.1_
  - [x] 1.3 Estender configuração com CryptoConfig
    - Adicionar campos em `internal/config/config.go`
    - Implementar parsing de variáveis de ambiente IAM_POLICY_CRYPTO_*
    - Implementar validação de configuração
    - _Requirements: 5.1, 5.2, 5.3, 5.4, 5.5, 5.6, 5.7_
  - [x] 1.4 Escrever property test para validação de configuração
    - **Property 8: Configuration Validation**
    - **Validates: Requirements 5.7**

- [x] 2. Checkpoint - Validar configuração
  - Ensure all tests pass, ask the user if questions arise.

- [x] 3. Crypto Client Base
  - [x] 3.1 Implementar KeyID e tipos base
    - Criar `internal/crypto/types.go` com KeyID, EncryptResult, SignResult
    - Implementar parsing de KeyID do formato string "namespace/id/version"
    - _Requirements: 4.1, 4.2_
  - [x] 3.2 Implementar Crypto Client
    - Criar `internal/crypto/client.go`
    - Implementar conexão gRPC com timeout configurável
    - Implementar propagação de W3C Trace Context via metadata
    - Implementar métodos Encrypt, Decrypt, Sign, Verify
    - Implementar HealthCheck
    - _Requirements: 1.1, 1.2, 1.3, 1.4_
  - [x] 3.3 Escrever property test para propagação de trace context
    - **Property 9: Trace Context Propagation**
    - **Validates: Requirements 1.3**
  - [x] 3.4 Escrever property test para correlação de erros
    - **Property 10: Error Correlation**
    - **Validates: Requirements 1.2**

- [x] 4. Checkpoint - Validar crypto client
  - Ensure all tests pass, ask the user if questions arise.

- [x] 5. Key Metadata Cache
  - [x] 5.1 Implementar KeyMetadataCache
    - Criar `internal/crypto/key_cache.go`
    - Implementar cache com TTL configurável
    - Implementar Get, Set, Invalidate
    - _Requirements: 4.4, 4.5_
  - [x] 5.2 Escrever unit tests para key cache TTL
    - Testar expiração de cache
    - Testar invalidação manual
    - _Requirements: 4.4, 4.5_

- [x] 6. Encrypted Decision Cache
  - [x] 6.1 Implementar EncryptedCacheEntry
    - Criar `internal/cache/encrypted_entry.go`
    - Definir estrutura para armazenamento criptografado
    - _Requirements: 2.1_
  - [x] 6.2 Implementar EncryptedDecisionCache
    - Criar `internal/cache/encrypted_cache.go`
    - Implementar decorator sobre DecisionCache existente
    - Implementar generateAAD usando subject_id e resource_id
    - Implementar Get com descriptografia
    - Implementar Set com criptografia
    - Implementar fallback para texto plano quando desabilitado
    - _Requirements: 2.1, 2.2, 2.3, 2.4, 2.5_
  - [x] 6.3 Escrever property test para encryption round-trip
    - **Property 1: Encryption Round-Trip Consistency**
    - **Validates: Requirements 2.1, 2.2, 2.6**
  - [x] 6.4 Escrever property test para AAD binding
    - **Property 2: AAD Context Binding**
    - **Validates: Requirements 2.3, 2.4**
  - [x] 6.5 Escrever property test para graceful degradation
    - **Property 6: Graceful Degradation**
    - **Validates: Requirements 1.4, 2.5**

- [x] 7. Checkpoint - Validar encrypted cache
  - Ensure all tests pass, ask the user if questions arise.

- [x] 8. Decision Signer
  - [x] 8.1 Implementar SignedDecision
    - Criar `internal/crypto/signed_decision.go`
    - Definir estrutura com todos os campos obrigatórios
    - Implementar buildSignaturePayload com serialização canônica
    - _Requirements: 3.2_
  - [x] 8.2 Implementar DecisionSigner
    - Criar `internal/crypto/signer.go`
    - Implementar Sign usando ECDSA P-256 via crypto-service
    - Implementar Verify
    - Implementar fallback quando desabilitado
    - _Requirements: 3.1, 3.3, 3.5, 3.6_
  - [x] 8.3 Escrever property test para sign-then-verify
    - **Property 3: Sign-Then-Verify Consistency**
    - **Validates: Requirements 3.1, 3.3**
  - [x] 8.4 Escrever property test para signature payload completeness
    - **Property 4: Signature Payload Completeness**
    - **Validates: Requirements 3.2**
  - [x] 8.5 Escrever property test para invalid signature detection
    - **Property 5: Invalid Signature Detection**
    - **Validates: Requirements 3.4**
  - [x] 8.6 Escrever property test para key version backward compatibility
    - **Property 7: Key Version Backward Compatibility**
    - **Validates: Requirements 4.3**

- [x] 9. Checkpoint - Validar decision signer
  - Ensure all tests pass, ask the user if questions arise.

- [x] 10. Crypto Metrics
  - [x] 10.1 Implementar CryptoMetrics
    - Criar `internal/crypto/metrics.go`
    - Registrar métricas: encrypt_total, decrypt_total, sign_total, verify_total
    - Registrar histograma de latência
    - Registrar contador de erros por código
    - Registrar contador de fallback
    - _Requirements: 6.1, 6.2, 6.3, 6.4, 6.5, 6.6, 6.7_
  - [x] 10.2 Escrever unit tests para métricas
    - Verificar que métricas são incrementadas corretamente
    - _Requirements: 6.1, 6.2, 6.3, 6.4, 6.5, 6.6, 6.7_

- [x] 11. Health Check Integration
  - [x] 11.1 Estender HealthManager
    - Modificar `internal/health/manager.go`
    - Adicionar verificação de conectividade com crypto-service
    - Retornar DEGRADED (não UNHEALTHY) quando indisponível
    - Adicionar campos crypto_service_connected e crypto_service_latency_ms
    - _Requirements: 7.1, 7.2, 7.3, 7.4_
  - [x] 11.2 Escrever property test para health check degraded status
    - **Property 11: Health Check Degraded Status**
    - **Validates: Requirements 7.2**

- [x] 12. Integração com Authorization Service
  - [x] 12.1 Modificar AuthorizationService
    - Atualizar `internal/service/authorization.go`
    - Injetar DecisionSigner
    - Adicionar assinatura opcional nas respostas
    - _Requirements: 3.1, 3.5, 3.6_
  - [x] 12.2 Atualizar Policy Engine
    - Modificar `internal/policy/engine.go`
    - Usar EncryptedDecisionCache em vez de DecisionCache
    - _Requirements: 2.1, 2.2_
  - [x] 12.3 Atualizar main.go
    - Modificar `cmd/server/main.go`
    - Inicializar CryptoClient
    - Inicializar EncryptedDecisionCache
    - Inicializar DecisionSigner
    - Conectar componentes
    - _Requirements: 1.1, 5.1_

- [x] 13. Checkpoint Final
  - Ensure all tests pass, ask the user if questions arise.
  - Fixed compilation errors in property tests:
    - Renamed `intToString` to `cryptoIntToString` in crypto_config_test.go
    - Renamed `TestTraceContextPropagation` to `TestObservabilityTraceContextPropagation` in observability_test.go
    - Fixed type mismatch: cast int64 to time.Duration in crypto_config_test.go

- [x] 14. Documentação
  - [x] 14.1 Atualizar README.md
    - Documentar novas variáveis de ambiente
    - Documentar métricas expostas
    - Documentar comportamento de fallback
    - _Requirements: 5.1, 5.2, 5.3, 5.4, 5.5, 5.6, 6.1, 6.2, 6.3, 6.4, 6.5, 6.6, 6.7_

## Notes

- Tasks marcadas com `*` são opcionais e podem ser puladas para MVP mais rápido
- Cada task referencia requisitos específicos para rastreabilidade
- Checkpoints garantem validação incremental
- Property tests validam propriedades universais de correção
- Unit tests validam exemplos específicos e edge cases
