<?php

declare(strict_types=1);

namespace EmailService\Infrastructure\Repository;

use EmailService\Application\DTO\AuditQuery;
use EmailService\Domain\Entity\AuditLog;

interface AuditLogRepositoryInterface
{
    public function save(AuditLog $auditLog): void;

    /**
     * @return AuditLog[]
     */
    public function findByQuery(AuditQuery $query): array;

    public function countByQuery(AuditQuery $query): int;

    /**
     * @return AuditLog[]
     */
    public function findByEmailId(string $emailId): array;
}
