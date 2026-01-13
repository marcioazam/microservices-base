<?php

declare(strict_types=1);

namespace EmailService\Application\DTO;

use EmailService\Domain\Entity\AuditLog;

final readonly class AuditResultSet
{
    /**
     * @param AuditLog[] $items
     */
    public function __construct(
        public array $items,
        public int $total,
        public int $limit,
        public int $offset
    ) {
    }

    public function hasMore(): bool
    {
        return ($this->offset + count($this->items)) < $this->total;
    }

    public function isEmpty(): bool
    {
        return empty($this->items);
    }
}
