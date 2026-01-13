<?php

declare(strict_types=1);

namespace EmailService\Infrastructure\Repository;

use EmailService\Application\DTO\AuditQuery;
use EmailService\Domain\Entity\AuditLog;

class InMemoryAuditLogRepository implements AuditLogRepositoryInterface
{
    /** @var AuditLog[] */
    private array $logs = [];

    public function save(AuditLog $auditLog): void
    {
        $this->logs[$auditLog->id] = $auditLog;
    }

    public function findByQuery(AuditQuery $query): array
    {
        $filtered = $this->applyFilters($query);
        
        // Sort by timestamp descending
        usort($filtered, fn(AuditLog $a, AuditLog $b) => 
            $b->timestamp <=> $a->timestamp
        );
        
        return array_slice($filtered, $query->offset, $query->limit);
    }

    public function countByQuery(AuditQuery $query): int
    {
        return count($this->applyFilters($query));
    }

    public function findByEmailId(string $emailId): array
    {
        return array_values(array_filter(
            $this->logs,
            fn(AuditLog $log) => $log->emailId === $emailId
        ));
    }

    /**
     * @return AuditLog[]
     */
    private function applyFilters(AuditQuery $query): array
    {
        return array_values(array_filter($this->logs, function (AuditLog $log) use ($query): bool {
            if ($query->startDate !== null && $log->timestamp < $query->startDate) {
                return false;
            }
            
            if ($query->endDate !== null && $log->timestamp > $query->endDate) {
                return false;
            }
            
            if ($query->status !== null && $log->status !== $query->status) {
                return false;
            }
            
            if ($query->senderId !== null && $log->senderId !== $query->senderId) {
                return false;
            }
            
            if ($query->emailId !== null && $log->emailId !== $query->emailId) {
                return false;
            }
            
            return true;
        }));
    }

    public function clear(): void
    {
        $this->logs = [];
    }

    public function getAll(): array
    {
        return array_values($this->logs);
    }
}
