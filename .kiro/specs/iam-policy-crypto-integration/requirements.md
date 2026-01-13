# Requirements Document

## Introduction

Este documento define os requisitos para integrar o serviço `iam-policy` com o `crypto-service` centralizado da plataforma. A integração visa proteger dados sensíveis de autorização através de criptografia e garantir integridade das decisões através de assinaturas digitais.

O `iam-policy` é um Policy Decision Point (PDP) que processa requisições de autorização usando OPA (Open Policy Agent). Atualmente, decisões de autorização são armazenadas em cache sem criptografia e não possuem garantia de integridade criptográfica.

## Glossary

- **IAM_Policy_Service**: Serviço de decisão de políticas que avalia requisições de autorização usando RBAC e ABAC
- **Crypto_Service**: Microserviço centralizado que fornece operações criptográficas (AES-256-GCM, RSA, ECDSA, assinaturas digitais)
- **Crypto_Client**: Cliente gRPC em Go que se comunica com o Crypto_Service
- **Decision_Cache**: Componente que armazena decisões de autorização em cache distribuído
- **Authorization_Decision**: Resultado de uma avaliação de política contendo allowed, reason, policy_name
- **Encrypted_Decision**: Decisão de autorização criptografada com AES-256-GCM
- **Signed_Decision**: Decisão de autorização com assinatura digital ECDSA para verificação de integridade
- **Key_ID**: Identificador único de chave criptográfica no formato namespace/id/version
- **AAD**: Additional Authenticated Data - dados adicionais autenticados mas não criptografados

## Requirements

### Requirement 1: Cliente gRPC do Crypto Service

**User Story:** Como desenvolvedor do iam-policy, quero um cliente gRPC para comunicar com o crypto-service, para que eu possa usar operações criptográficas centralizadas.

#### Acceptance Criteria

1. THE Crypto_Client SHALL conectar ao Crypto_Service via gRPC com timeout configurável
2. WHEN a conexão com Crypto_Service falhar, THEN THE Crypto_Client SHALL retornar erro descritivo com correlation_id
3. THE Crypto_Client SHALL propagar W3C Trace Context em todas as requisições gRPC
4. WHEN o Crypto_Service estiver indisponível, THEN THE IAM_Policy_Service SHALL continuar operando em modo degradado sem criptografia
5. THE Crypto_Client SHALL expor métricas de latência e erros para Prometheus

### Requirement 2: Criptografia de Decisões em Cache

**User Story:** Como arquiteto de segurança, quero que decisões de autorização em cache sejam criptografadas, para que dados sensíveis não fiquem expostos no cache distribuído.

#### Acceptance Criteria

1. WHEN uma decisão é armazenada em cache, THEN THE Decision_Cache SHALL criptografar o payload usando AES-256-GCM via Crypto_Service
2. WHEN uma decisão é recuperada do cache, THEN THE Decision_Cache SHALL descriptografar o payload usando Crypto_Service
3. THE Decision_Cache SHALL usar o subject_id e resource_id como AAD (Additional Authenticated Data) para binding contextual
4. WHEN a descriptografia falhar por AAD inválido, THEN THE Decision_Cache SHALL invalidar a entrada e retornar cache miss
5. WHEN a criptografia estiver desabilitada, THEN THE Decision_Cache SHALL armazenar decisões em texto plano como fallback
6. FOR ALL decisões criptografadas, descriptografar e re-criptografar SHALL produzir payload equivalente ao original (round-trip)

### Requirement 3: Assinatura de Decisões de Autorização

**User Story:** Como auditor de compliance, quero que decisões de autorização sejam assinadas digitalmente, para que eu possa verificar integridade e não-repúdio.

#### Acceptance Criteria

1. WHEN uma decisão de autorização é emitida, THEN THE IAM_Policy_Service SHALL assinar a decisão usando ECDSA P-256 via Crypto_Service
2. THE Signed_Decision SHALL incluir timestamp, decision_id, subject_id, resource_id, action, allowed, e policy_name no payload assinado
3. WHEN um cliente solicitar verificação, THEN THE IAM_Policy_Service SHALL verificar a assinatura via Crypto_Service
4. IF a verificação de assinatura falhar, THEN THE IAM_Policy_Service SHALL retornar erro SIGNATURE_INVALID com detalhes
5. THE Signed_Decision SHALL ser retornada opcionalmente via campo signature na resposta gRPC
6. WHEN assinatura estiver desabilitada, THEN THE IAM_Policy_Service SHALL omitir o campo signature na resposta

### Requirement 4: Gerenciamento de Chaves

**User Story:** Como operador de plataforma, quero que chaves criptográficas sejam gerenciadas centralmente, para que eu possa rotacionar chaves sem downtime.

#### Acceptance Criteria

1. THE IAM_Policy_Service SHALL usar Key_ID configurável para operações de criptografia de cache
2. THE IAM_Policy_Service SHALL usar Key_ID configurável para operações de assinatura
3. WHEN uma chave é rotacionada no Crypto_Service, THEN THE IAM_Policy_Service SHALL aceitar decisões assinadas com versões anteriores da chave
4. THE IAM_Policy_Service SHALL cachear metadados de chaves localmente com TTL configurável
5. WHEN o cache de metadados expirar, THEN THE Crypto_Client SHALL buscar metadados atualizados do Crypto_Service

### Requirement 5: Configuração e Feature Flags

**User Story:** Como DevOps, quero controlar funcionalidades de criptografia via configuração, para que eu possa habilitar/desabilitar features sem deploy.

#### Acceptance Criteria

1. THE IAM_Policy_Service SHALL suportar variável IAM_POLICY_CRYPTO_ENABLED para habilitar/desabilitar integração
2. THE IAM_Policy_Service SHALL suportar variável IAM_POLICY_CRYPTO_ADDRESS para endereço do Crypto_Service
3. THE IAM_Policy_Service SHALL suportar variável IAM_POLICY_CRYPTO_CACHE_ENCRYPTION para habilitar criptografia de cache
4. THE IAM_Policy_Service SHALL suportar variável IAM_POLICY_CRYPTO_SIGNING para habilitar assinatura de decisões
5. THE IAM_Policy_Service SHALL suportar variável IAM_POLICY_CRYPTO_ENCRYPTION_KEY_ID para Key_ID de criptografia
6. THE IAM_Policy_Service SHALL suportar variável IAM_POLICY_CRYPTO_SIGNING_KEY_ID para Key_ID de assinatura
7. WHEN configuração inválida for detectada, THEN THE IAM_Policy_Service SHALL falhar na inicialização com mensagem clara

### Requirement 6: Observabilidade e Métricas

**User Story:** Como SRE, quero métricas detalhadas das operações criptográficas, para que eu possa monitorar performance e detectar problemas.

#### Acceptance Criteria

1. THE IAM_Policy_Service SHALL expor métrica iam_crypto_encrypt_total com labels operation e status
2. THE IAM_Policy_Service SHALL expor métrica iam_crypto_decrypt_total com labels operation e status
3. THE IAM_Policy_Service SHALL expor métrica iam_crypto_sign_total com labels status
4. THE IAM_Policy_Service SHALL expor métrica iam_crypto_verify_total com labels status
5. THE IAM_Policy_Service SHALL expor métrica iam_crypto_latency_seconds como histograma com labels operation
6. THE IAM_Policy_Service SHALL expor métrica iam_crypto_errors_total com labels error_code
7. WHEN Crypto_Service estiver indisponível, THEN THE IAM_Policy_Service SHALL incrementar iam_crypto_fallback_total

### Requirement 7: Health Check Integration

**User Story:** Como operador Kubernetes, quero que o health check reflita status do crypto-service, para que pods degradados sejam identificados.

#### Acceptance Criteria

1. THE IAM_Policy_Service readiness check SHALL verificar conectividade com Crypto_Service
2. WHEN Crypto_Service estiver indisponível, THEN THE readiness check SHALL retornar status DEGRADED (não UNHEALTHY)
3. THE health response SHALL incluir campo crypto_service_connected com status booleano
4. THE health response SHALL incluir campo crypto_service_latency_ms com latência do último health check
