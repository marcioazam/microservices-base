package com.authplatform.usersvc.infra.outbox;

import com.authplatform.usersvc.domain.model.OutboxEvent;
import com.authplatform.usersvc.infra.persistence.OutboxEventRepository;
import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.databind.ObjectMapper;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Component;
import java.util.UUID;

@Component
@RequiredArgsConstructor
@Slf4j
public class OutboxPublisher {

    private final OutboxEventRepository outboxEventRepository;
    private final ObjectMapper objectMapper;

    public void publish(String aggregateType, UUID aggregateId, String eventType, Object payload) {
        try {
            String payloadJson = objectMapper.writeValueAsString(payload);
            
            OutboxEvent event = OutboxEvent.builder()
                    .aggregateType(aggregateType)
                    .aggregateId(aggregateId)
                    .eventType(eventType)
                    .payloadJson(payloadJson)
                    .build();
            
            outboxEventRepository.save(event);
            log.debug("Published outbox event: type={}, aggregateId={}", eventType, aggregateId);
        } catch (JsonProcessingException e) {
            log.error("Failed to serialize event payload: {}", e.getMessage());
            throw new RuntimeException("Failed to serialize event payload", e);
        }
    }

    public void publishUserRegistered(UUID userId, String email, String displayName) {
        var payload = new UserRegisteredPayload(userId, email, displayName, java.time.Instant.now());
        publish("User", userId, "UserRegistered", payload);
    }

    public void publishEmailVerificationRequested(UUID userId, String email, String verificationLink) {
        var payload = new EmailVerificationRequestedPayload(userId, email, verificationLink, "email-verification", "en");
        publish("User", userId, "EmailVerificationRequested", payload);
    }

    public void publishUserEmailVerified(UUID userId, String email) {
        var payload = new UserEmailVerifiedPayload(userId, email, java.time.Instant.now());
        publish("User", userId, "UserEmailVerified", payload);
    }

    public record UserRegisteredPayload(UUID userId, String email, String displayName, java.time.Instant registeredAt) {}
    public record EmailVerificationRequestedPayload(UUID userId, String email, String verificationLink, String templateId, String locale) {}
    public record UserEmailVerifiedPayload(UUID userId, String email, java.time.Instant verifiedAt) {}
}
