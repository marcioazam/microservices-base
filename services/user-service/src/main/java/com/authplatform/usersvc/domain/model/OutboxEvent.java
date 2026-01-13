package com.authplatform.usersvc.domain.model;

import jakarta.persistence.*;
import lombok.*;
import java.time.Instant;
import java.util.UUID;

@Entity
@Table(name = "outbox_events")
@Getter
@Setter
@NoArgsConstructor
@AllArgsConstructor
@Builder
public class OutboxEvent {

    @Id
    @GeneratedValue(strategy = GenerationType.UUID)
    private UUID id;

    @Column(name = "aggregate_type", nullable = false, length = 50)
    private String aggregateType;

    @Column(name = "aggregate_id", nullable = false)
    private UUID aggregateId;

    @Column(name = "event_type", nullable = false, length = 50)
    private String eventType;

    @Column(name = "payload_json", nullable = false, columnDefinition = "jsonb")
    private String payloadJson;

    @Column(name = "created_at", nullable = false, updatable = false)
    private Instant createdAt;

    @Column(name = "processed_at")
    private Instant processedAt;

    @Column(name = "retry_count", nullable = false)
    private int retryCount;

    @Column(name = "last_error")
    private String lastError;

    @PrePersist
    protected void onCreate() {
        createdAt = Instant.now();
        retryCount = 0;
    }

    public boolean isProcessed() {
        return processedAt != null;
    }

    public void markAsProcessed() {
        this.processedAt = Instant.now();
    }

    public void recordFailure(String error) {
        this.retryCount++;
        this.lastError = error;
    }
}
