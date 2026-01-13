package com.authplatform.usersvc.infrastructure.outbox;

import com.authplatform.usersvc.domain.model.OutboxEvent;
import com.authplatform.usersvc.infra.persistence.OutboxEventRepository;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.kafka.core.KafkaTemplate;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Component;
import org.springframework.transaction.annotation.Transactional;

import java.util.List;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;

/**
 * Dispatches outbox events to Kafka using virtual threads.
 */
@Component
@RequiredArgsConstructor
@Slf4j
public class OutboxDispatcher {

    private static final int BATCH_SIZE = 100;
    private static final String TOPIC_PREFIX = "user-service.";
    
    private final OutboxEventRepository outboxRepository;
    private final KafkaTemplate<String, String> kafkaTemplate;
    
    // Virtual thread executor for high-concurrency I/O
    private final ExecutorService virtualExecutor = Executors.newVirtualThreadPerTaskExecutor();

    @Scheduled(fixedDelayString = "${app.outbox.poll-interval-ms:1000}")
    @Transactional
    public void dispatchEvents() {
        List<OutboxEvent> events = outboxRepository.findUnprocessedEvents(BATCH_SIZE);
        
        if (events.isEmpty()) {
            return;
        }
        
        log.debug("Dispatching {} outbox events", events.size());
        
        for (OutboxEvent event : events) {
            virtualExecutor.submit(() -> processEvent(event));
        }
    }

    private void processEvent(OutboxEvent event) {
        try {
            String topic = TOPIC_PREFIX + event.getEventType().toLowerCase().replace("_", "-");
            String key = event.getAggregateId().toString();
            
            kafkaTemplate.send(topic, key, event.getPayloadJson())
                    .whenComplete((result, ex) -> {
                        if (ex != null) {
                            log.error("Failed to send event to Kafka: eventId={}, error={}", 
                                    event.getId(), ex.getMessage());
                            markEventFailed(event, ex.getMessage());
                        } else {
                            log.debug("Event sent to Kafka: eventId={}, topic={}", event.getId(), topic);
                            markEventProcessed(event);
                        }
                    });
        } catch (Exception e) {
            log.error("Error processing outbox event: eventId={}", event.getId(), e);
            markEventFailed(event, e.getMessage());
        }
    }

    @Transactional
    protected void markEventProcessed(OutboxEvent event) {
        event.markAsProcessed();
        outboxRepository.save(event);
    }

    @Transactional
    protected void markEventFailed(OutboxEvent event, String error) {
        event.recordFailure(error);
        outboxRepository.save(event);
    }
}
