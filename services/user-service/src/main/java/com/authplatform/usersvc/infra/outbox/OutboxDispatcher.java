package com.authplatform.usersvc.infra.outbox;

import com.authplatform.usersvc.domain.model.OutboxEvent;
import com.authplatform.usersvc.infra.persistence.OutboxEventRepository;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.kafka.core.KafkaTemplate;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Component;
import org.springframework.transaction.annotation.Transactional;
import java.time.Instant;
import java.util.List;

@Component
@RequiredArgsConstructor
@Slf4j
public class OutboxDispatcher {

    private final OutboxEventRepository outboxEventRepository;
    private final KafkaTemplate<String, String> kafkaTemplate;

    @Value("${app.outbox.batch-size:100}")
    private int batchSize;

    @Value("${app.outbox.topic-prefix:user-service}")
    private String topicPrefix;

    @Scheduled(fixedDelayString = "${app.outbox.poll-interval-ms:1000}")
    @Transactional
    public void dispatchPendingEvents() {
        List<OutboxEvent> pendingEvents = outboxEventRepository
                .findByProcessedAtIsNullOrderByCreatedAtAsc()
                .stream()
                .limit(batchSize)
                .toList();

        if (pendingEvents.isEmpty()) {
            return;
        }

        log.debug("Processing {} pending outbox events", pendingEvents.size());

        for (OutboxEvent event : pendingEvents) {
            try {
                dispatchEvent(event);
                markAsProcessed(event);
            } catch (Exception e) {
                log.error("Failed to dispatch event {}: {}", event.getId(), e.getMessage());
                // Event will be retried on next poll
            }
        }
    }

    private void dispatchEvent(OutboxEvent event) {
        String topic = buildTopicName(event.getEventType());
        String key = event.getAggregateId().toString();
        
        kafkaTemplate.send(topic, key, event.getPayloadJson())
                .whenComplete((result, ex) -> {
                    if (ex != null) {
                        log.error("Kafka send failed for event {}: {}", event.getId(), ex.getMessage());
                    } else {
                        log.debug("Event {} sent to topic {}", event.getId(), topic);
                    }
                });
    }

    private void markAsProcessed(OutboxEvent event) {
        event.setProcessedAt(Instant.now());
        outboxEventRepository.save(event);
    }

    private String buildTopicName(String eventType) {
        return topicPrefix + "." + camelToKebab(eventType);
    }

    private String camelToKebab(String camelCase) {
        return camelCase.replaceAll("([a-z])([A-Z])", "$1-$2").toLowerCase();
    }
}
