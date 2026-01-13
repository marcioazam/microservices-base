<?php

declare(strict_types=1);

namespace EmailService\Application\Service;

use EmailService\Application\DTO\AuditQuery;
use EmailService\Application\DTO\AuditResultSet;
use EmailService\Domain\Entity\AuditLog;

interface AuditServiceInterface
{
    /**
     * Log an audit entry
     */
    public function log(AuditLog $entry): void;

    /**
     * Query audit logs with filters
     */
    public function query(AuditQuery $query): AuditResultSet;

    /**
     * Get all audit logs for a specific email
     * @return AuditLog[]
     */
    public function getByEmailId(string $emailId): array;
}
