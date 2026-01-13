<?php

declare(strict_types=1);

namespace EmailService\Infrastructure\Repository;

use EmailService\Domain\Entity\Email;

class InMemoryEmailRepository implements EmailRepositoryInterface
{
    /** @var array<string, Email> */
    private array $emails = [];

    public function save(Email $email): void
    {
        $this->emails[$email->id] = $email;
    }

    public function findById(string $id): ?Email
    {
        return $this->emails[$id] ?? null;
    }

    public function findByStatus(string $status): array
    {
        return array_values(array_filter(
            $this->emails,
            fn(Email $email) => $email->getStatus()->value === $status
        ));
    }

    public function clear(): void
    {
        $this->emails = [];
    }

    public function getAll(): array
    {
        return array_values($this->emails);
    }
}
