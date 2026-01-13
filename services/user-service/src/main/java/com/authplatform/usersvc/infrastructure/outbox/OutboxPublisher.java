package com.authplatform.usersvc.infrastructure.outbox;

import com.authplatform.usersvc.domain.model.OutboxEvent;
import com.authplatform.usersvc.infra.persistence.OutboxEventRepository;
import com.fasterxml.jackson.databind.ObjectMapper;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Component;
import org.springframework.transaction.annotation.Propagation;
import org.springframework.transaction.annotation.Transactional;

import java.util.Map;
import java.util.UUID;

/**
 * Publishes domain events to the outbox table within the same transaction.
 */
@Component
@RequiredArgsConstructor
@Slf4j
public class OutboxPublisher {

    private final OutboxEventRepository outboxRepository;
    private final ObjectMapper objectMapper;

    @Transactional(propagation = Propagation.MANDATORY)
    public void publish(String aggregateType, UUID aggregateId, String eventType, Map<String, String> payload) {
        try {
            String payloadJson = objectMapper.writeValueAsString(payload);
            
            OutboxEvent event = OutboxEvent.builder()
                    .aggregateType(aggregateType)
                    .aggregateId(aggregateId)
                    .eventType(eventType)
                    .payloadJson(payloadJson)
                    .build();
            
            outboxRepository.save(event);
            log.debug("Published outbox event: type={}, aggregateId={}", eventType, aggregateId);
        } catch (Exception e) {
            log.error("Failed to publish outbox event: type={}, aggregateId={}", eventType, aggregateId, e);
            throw new RuntimeException("Failed to publish outbox event", e);
        }
    }
}
