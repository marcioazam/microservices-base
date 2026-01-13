package com.authplatform.usersvc.property;

import com.authplatform.usersvc.domain.model.OutboxEvent;
import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import net.jqwik.api.*;
import net.jqwik.api.constraints.AlphaChars;
import net.jqwik.api.constraints.StringLength;
import java.time.Instant;
import java.util.UUID;
import static org.assertj.core.api.Assertions.assertThat;

/**
 * Property 6: Outbox Event Completeness
 * Validates: Requirements 1.9, 2.8, 7.1, 7.2
 * 
 * Ensures that outbox events contain all required fields
 * and maintain data integrity for reliable event delivery.
 */
class OutboxEventPropertyTest {

    private final ObjectMapper objectMapper = new ObjectMapper();

    @Property(tries = 100)
    void outboxEventHasAllRequiredFields(
            @ForAll("validAggregateType") String aggregateType,
            @ForAll("validEventType") String eventType,
            @ForAll @AlphaChars @StringLength(min = 5, max = 50) String email) {
        
        UUID aggregateId = UUID.randomUUID();
        String payloadJson = createPayloadJson(aggregateId, email);
        
        OutboxEvent event = OutboxEvent.builder()
                .aggregateType(aggregateType)
                .aggregateId(aggregateId)
                .eventType(eventType)
                .payloadJson(payloadJson)
                .build();
        
        // All required fields must be present
        assertThat(event.getId()).isNotNull();
        assertThat(event.getAggregateType()).isNotBlank();
        assertThat(event.getAggregateId()).isNotNull();
        assertThat(event.getEventType()).isNotBlank();
        assertThat(event.getPayloadJson()).isNotBlank();
        assertThat(event.getCreatedAt()).isNotNull();
        
        // processedAt should be null for new events
        assertThat(event.getProcessedAt()).isNull();
    }

    @Property(tries = 100)
    void outboxEventPayloadIsValidJson(
            @ForAll @AlphaChars @StringLength(min = 5, max = 50) String email,
            @ForAll @AlphaChars @StringLength(min = 2, max = 30) String displayName) {
        
        UUID userId = UUID.randomUUID();
        String payloadJson = String.format(
            "{\"userId\":\"%s\",\"email\":\"%s@example.com\",\"displayName\":\"%s\"}",
            userId, email.toLowerCase(), displayName
        );
        
        OutboxEvent event = OutboxEvent.builder()
                .aggregateType("User")
                .aggregateId(userId)
                .eventType("UserRegistered")
                .payloadJson(payloadJson)
                .build();
        
        // Payload must be valid JSON
        assertThat(isValidJson(event.getPayloadJson())).isTrue();
        
        // Payload must contain userId
        assertThat(event.getPayloadJson()).contains(userId.toString());
    }

    @Property(tries = 100)
    void processedEventHasProcessedTimestamp(
            @ForAll("validEventType") String eventType) {
        
        UUID aggregateId = UUID.randomUUID();
        OutboxEvent event = OutboxEvent.builder()
                .aggregateType("User")
                .aggregateId(aggregateId)
                .eventType(eventType)
                .payloadJson("{\"userId\":\"" + aggregateId + "\"}")
                .build();
        
        // Simulate processing
        Instant processedAt = Instant.now();
        event.setProcessedAt(processedAt);
        
        // Processed event must have timestamp
        assertThat(event.getProcessedAt()).isNotNull();
        assertThat(event.getProcessedAt()).isAfterOrEqualTo(event.getCreatedAt());
    }

    @Property(tries = 100)
    void eventTypeMatchesAggregateType(
            @ForAll("validAggregateType") String aggregateType,
            @ForAll("validEventType") String eventType) {
        
        UUID aggregateId = UUID.randomUUID();
        OutboxEvent event = OutboxEvent.builder()
                .aggregateType(aggregateType)
                .aggregateId(aggregateId)
                .eventType(eventType)
                .payloadJson("{}")
                .build();
        
        // Event type should relate to aggregate type
        if (aggregateType.equals("User")) {
            assertThat(eventType).startsWith("User");
        }
    }

    @Provide
    Arbitrary<String> validAggregateType() {
        return Arbitraries.of("User");
    }

    @Provide
    Arbitrary<String> validEventType() {
        return Arbitraries.of(
            "UserRegistered",
            "UserEmailVerified",
            "EmailVerificationRequested"
        );
    }

    private String createPayloadJson(UUID userId, String email) {
        return String.format(
            "{\"userId\":\"%s\",\"email\":\"%s@example.com\",\"timestamp\":\"%s\"}",
            userId, email.toLowerCase(), Instant.now()
        );
    }

    private boolean isValidJson(String json) {
        try {
            objectMapper.readTree(json);
            return true;
        } catch (Exception e) {
            return false;
        }
    }
}
