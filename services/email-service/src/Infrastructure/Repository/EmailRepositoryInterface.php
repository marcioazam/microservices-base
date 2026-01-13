<?php

declare(strict_types=1);

namespace EmailService\Infrastructure\Repository;

use EmailService\Domain\Entity\Email;

interface EmailRepositoryInterface
{
    public function save(Email $email): void;

    public function findById(string $id): ?Email;

    /**
     * @return Email[]
     */
    public function findByStatus(string $status): array;
}
