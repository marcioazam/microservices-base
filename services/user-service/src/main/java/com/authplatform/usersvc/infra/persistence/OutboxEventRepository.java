package com.authplatform.usersvc.infra.persistence;

import com.authplatform.usersvc.domain.model.OutboxEvent;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.data.jpa.repository.Query;
import org.springframework.data.repository.query.Param;
import org.springframework.stereotype.Repository;
import java.util.List;
import java.util.UUID;

@Repository
public interface OutboxEventRepository extends JpaRepository<OutboxEvent, UUID> {

    @Query("SELECT e FROM OutboxEvent e WHERE e.processedAt IS NULL ORDER BY e.createdAt ASC")
    List<OutboxEvent> findUnprocessedEvents();

    @Query(value = "SELECT e FROM OutboxEvent e WHERE e.processedAt IS NULL ORDER BY e.createdAt ASC LIMIT :limit")
    List<OutboxEvent> findUnprocessedEvents(@Param("limit") int limit);

    @Query("SELECT e FROM OutboxEvent e WHERE e.processedAt IS NULL AND e.retryCount < :maxRetries ORDER BY e.createdAt ASC")
    List<OutboxEvent> findUnprocessedEventsWithRetryLimit(@Param("maxRetries") int maxRetries);

    List<OutboxEvent> findByAggregateTypeAndAggregateId(String aggregateType, UUID aggregateId);

    long countByProcessedAtIsNull();
}
