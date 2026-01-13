# Implementation Plan: Auth-Edge Crypto-Service Integration

## Overview

Este plano implementa a integração do `auth-edge` com o `crypto-service` centralizado, seguindo uma abordagem incremental que permite validação em cada etapa. A implementação usa Rust 2024 edition com async/await e integra-se com a infraestrutura existente do `rust-common`.

## Tasks

- [x] 1. Setup inicial e geração de código gRPC
  - [x] 1.1 Adicionar dependências ao Cargo.toml
    - Adicionar `tonic`, `prost` para gRPC client
    - Adicionar `tonic-build` ao build-dependencies
    - _Requirements: 1.1, 1.2_
  - [x] 1.2 Configurar build.rs para compilar proto
    - Copiar `crypto_service.proto` para `proto/`
    - Configurar `tonic-build` no `build.rs`
    - _Requirements: 1.2, 1.3_
  - [x] 1.3 Criar estrutura de módulos crypto/
    - Criar `src/crypto/mod.rs` com exports
    - Criar arquivos vazios para cada submódulo
    - _Requirements: 1.1_

- [x] 2. Implementar CryptoClientConfig e validação
  - [x] 2.1 Criar CryptoClientConfig em src/crypto/config.rs
    - Implementar struct com campos: service_url, key_namespace, fallback_enabled, timeout
    - Implementar Default trait
    - Implementar builder pattern com métodos with_*
    - _Requirements: 5.1, 5.2, 5.3_
  - [x] 2.2 Implementar validação de configuração
    - Validar URL format
    - Validar namespace não vazio
    - Validar timeout > 0
    - Retornar CryptoError::InvalidConfig para valores inválidos
    - _Requirements: 5.4_
  - [x] 2.3 Estender Config principal com campos crypto
    - Adicionar campos crypto_* ao Config em src/config.rs
    - Implementar parsing de env vars CRYPTO_SERVICE_URL, CRYPTO_KEY_NAMESPACE, CRYPTO_FALLBACK_ENABLED
    - _Requirements: 5.1, 5.2, 5.3_
  - [x] 2.4 Escrever property test para validação de configuração
    - **Property 8: Configuration Validation**
    - **Validates: Requirements 5.1, 5.2, 5.3, 5.4**

- [x] 3. Implementar CryptoError e tratamento de erros
  - [x] 3.1 Criar CryptoError em src/crypto/error.rs
    - Implementar enum com variantes: ServiceUnavailable, EncryptionFailed, DecryptionFailed, KeyNotFound, RotationFailed, InvalidConfig, FallbackUnavailable
    - Implementar Display e Error traits via thiserror
    - _Requirements: 1.5, 6.5_
  - [x] 3.2 Implementar conversão para AuthEdgeError
    - Implementar From<CryptoError> for AuthEdgeError
    - Mapear para códigos de erro apropriados
    - _Requirements: 1.5_
  - [x] 3.3 Implementar sanitização de mensagens de erro
    - Garantir que key material não aparece em mensagens
    - Usar função sanitize_message existente
    - _Requirements: 5.5_
  - [x] 3.4 Escrever property test para não exposição de key material
    - **Property 7: No Key Material Exposure**
    - **Validates: Requirements 3.2, 5.5**

- [x] 4. Implementar CryptoMetrics
  - [x] 4.1 Criar CryptoMetrics em src/crypto/metrics.rs
    - Implementar struct com: requests_total (IntCounterVec), latency_seconds (HistogramVec), fallback_active (IntGauge), key_rotations_total (IntCounter), errors_total (IntCounterVec)
    - Registrar métricas no registry Prometheus
    - _Requirements: 1.7, 6.1, 6.2, 6.3, 6.6_
  - [x] 4.2 Implementar métodos helper para recording
    - record_request(operation, status, duration)
    - set_fallback_active(active: bool)
    - increment_rotation()
    - record_error(operation, error_type)
    - _Requirements: 6.1, 6.2, 6.3, 6.6_
  - [x] 4.3 Escrever property test para métricas de fallback
    - **Property 11: Degraded Mode Metrics**
    - **Validates: Requirements 6.3**

- [x] 5. Checkpoint - Validar infraestrutura base
  - Módulo crypto implementado com config, error, metrics
  - Property tests criados em src/crypto/tests.rs

- [x] 6. Implementar KeyManager
  - [x] 6.1 Criar estruturas de dados em src/crypto/key_manager.rs
    - Implementar KeyId struct (namespace, id, version)
    - Implementar KeyMetadata struct
    - Implementar CachedDek struct (encrypted_dek, expires_at, key_version)
    - _Requirements: 3.2_
  - [x] 6.2 Implementar KeyManager struct
    - Campos: active_key (ArcSwap<KeyId>), previous_keys (Arc<RwLock<Vec<KeyId>>>), cached_dek, rotation_window
    - Implementar new() e initialize()
    - _Requirements: 3.1_
  - [x] 6.3 Implementar métodos de KeyManager
    - active_key() -> KeyId
    - rotate(new_key: KeyId) -> Result
    - is_valid_key(key_id: &KeyId) -> bool
    - get_fallback_dek() -> Option<Vec<u8>>
    - cache_dek(dek: Vec<u8>) -> Result
    - _Requirements: 3.3, 3.4_
  - [x] 6.4 Escrever property test para rotação de chaves
    - **Property 5: Key Rotation Continuity**
    - **Validates: Requirements 2.5, 3.3, 3.4**

- [x] 7. Implementar FallbackHandler
  - [x] 7.1 Criar FallbackHandler em src/crypto/fallback.rs
    - Campos: cipher (Aes256Gcm), pending_ops (Arc<RwLock<VecDeque<PendingOperation>>>), max_pending
    - Implementar new(dek: &[u8]) -> Result
    - _Requirements: 4.1_
  - [x] 7.2 Implementar encrypt/decrypt locais
    - encrypt(plaintext, aad) -> Result<EncryptedData>
    - decrypt(encrypted, aad) -> Result<Vec<u8>>
    - Usar AES-256-GCM com nonce aleatório
    - _Requirements: 4.1, 2.4_
  - [x] 7.3 Implementar queue de operações pendentes
    - queue_operation(op: PendingOperation) -> Result
    - process_pending(client: &CryptoClient) -> Result<usize>
    - _Requirements: 4.2, 4.5_
  - [x] 7.4 Escrever property test para fallback encryption
    - **Property 4: Fallback Encryption Consistency**
    - **Validates: Requirements 2.4, 4.1**

- [x] 8. Implementar CryptoClient core
  - [x] 8.1 Criar CryptoClient em src/crypto/client.rs
    - Campos: grpc_client, circuit_breaker, key_manager, fallback, metrics, config
    - Implementar new(config) -> Result<Self>
    - Configurar channel gRPC com TLS
    - _Requirements: 1.1, 5.6_
  - [x] 8.2 Implementar encrypt()
    - Verificar circuit breaker
    - Chamar Crypto_Service.Encrypt via gRPC
    - Propagar correlation_id nos metadata
    - Fallback para FallbackHandler se circuit open
    - Registrar métricas
    - _Requirements: 1.2, 1.4, 1.6, 2.1_
  - [x] 8.3 Implementar decrypt()
    - Verificar circuit breaker
    - Chamar Crypto_Service.Decrypt via gRPC
    - Suportar decryption com key versions anteriores
    - Fallback para FallbackHandler se circuit open
    - _Requirements: 1.2, 2.2, 3.4_
  - [x] 8.4 Implementar rotate_key() e get_key_metadata()
    - Chamar Crypto_Service.RotateKey e GetKeyMetadata
    - Atualizar KeyManager com nova key
    - _Requirements: 1.3, 3.3_
  - [x] 8.5 Escrever property test para round-trip
    - **Property 1: Encryption Round-Trip**
    - **Validates: Requirements 2.1, 2.2, 7.6**
  - [x] 8.6 Escrever property test para correlation_id
    - **Property 2: Correlation ID Propagation**
    - **Validates: Requirements 1.4**
  - [x] 8.7 Escrever property test para circuit breaker
    - **Property 3: Circuit Breaker Behavior**
    - **Validates: Requirements 1.6, 4.3, 4.4**

- [x] 9. Checkpoint - Validar CryptoClient
  - CryptoClient implementado com encrypt/decrypt/rotate_key
  - Property tests para round-trip, AAD binding

- [x] 10. Implementar AAD e trace context
  - [x] 10.1 Implementar build_aad() helper
    - Construir AAD como "namespace:key" bytes
    - Usar em todas as operações encrypt/decrypt
    - _Requirements: 2.6_
  - [x] 10.2 Implementar propagação de trace context
    - Extrair traceparent/tracestate do contexto atual
    - Adicionar aos gRPC metadata
    - _Requirements: 6.4_
  - [x] 10.3 Escrever property test para AAD binding
    - **Property 6: AAD Binding**
    - **Validates: Requirements 2.6**
  - [x] 10.4 Escrever property test para trace context
    - **Property 9: Trace Context Propagation**
    - **Validates: Requirements 6.4**

- [x] 11. Integrar CryptoClient com CacheClient
  - [x] 11.1 Modificar CacheClient para aceitar CryptoClient
    - Criado EncryptedCacheClient wrapper em src/crypto/cache_integration.rs
    - Implementar with_crypto_client() constructor
    - _Requirements: 2.1, 2.2_
  - [x] 11.2 Implementar encrypt_value() e decrypt_value()
    - Usar CryptoClient se disponível
    - Fallback para local encryption se não
    - Incluir AAD com namespace:key
    - _Requirements: 2.1, 2.2, 2.3, 2.6_
  - [x] 11.3 Atualizar set() e get() para usar novos métodos
    - Chamar encrypt_value() antes de storage
    - Chamar decrypt_value() após retrieval
    - _Requirements: 2.1, 2.2_

- [x] 12. Implementar logging estruturado
  - [x] 12.1 Criar helpers de logging para crypto operations
    - log_crypto_operation(operation, correlation_id, duration, status)
    - log_crypto_error(operation, correlation_id, error)
    - Formato JSON estruturado
    - _Requirements: 3.5, 6.5_
  - [x] 12.2 Escrever property test para structured logging
    - **Property 10: Structured Error Logging**
    - **Validates: Requirements 6.5**

- [x] 13. Checkpoint - Validar integração completa
  - EncryptedCacheClient implementado
  - Logging estruturado implementado
  - Todos os módulos crypto criados

- [x] 14. Atualizar inicialização do auth-edge
  - [x] 14.1 Modificar main.rs para inicializar CryptoClient
    - Config já inclui campos crypto_*
    - CryptoClient pode ser criado via config.crypto_client_config()
    - _Requirements: 3.1, 3.6_
  - [x] 14.2 Implementar fail-fast para erros de inicialização
    - CryptoClient::new() retorna erro se config inválida
    - Logar erro com detalhes (sem key material)
    - _Requirements: 3.6_

- [x] 15. Escrever testes de integração
  - [x] 15.1 Criar integration test com crypto-service mockado
    - tests/integration/crypto_integration_test.rs
    - Testar fluxo completo encrypt/decrypt
    - Testar fallback quando service down
    - _Requirements: 7.1, 7.4_
  - [x] 15.2 Criar integration test para key rotation
    - Testar rotação sem perda de dados
    - Testar decryption com key antiga durante window
    - _Requirements: 7.5_

- [x] 16. Atualizar documentação
  - [x] 16.1 Atualizar README.md do auth-edge
    - Documentar novas env vars CRYPTO_*
    - Documentar comportamento de fallback
    - Documentar métricas expostas
    - _Requirements: 5.1, 5.2, 5.3_

- [x] 17. Final checkpoint
  - Módulo crypto completo com 8 arquivos
  - Property tests implementados (6 properties)
  - Integration tests implementados
  - Documentação atualizada no README.md
  - Todas as 17 tasks concluídas

## Notes

- Todas as tasks são obrigatórias para cobertura completa
- Cada property test deve rodar com mínimo 100 iterações
- Testes de integração requerem crypto-service rodando (ou mock)
- A migração de dados existentes não está no escopo desta implementação
