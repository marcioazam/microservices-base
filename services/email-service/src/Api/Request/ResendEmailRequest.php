<?php

declare(strict_types=1);

namespace EmailService\Api\Request;

final readonly class ResendEmailRequest
{
    public function __construct(
        public string $emailId
    ) {
    }

    public static function fromArray(array $data): self
    {
        return new self(
            emailId: $data['email_id'] ?? ''
        );
    }

    /**
     * @return array<string, string>
     */
    public function validate(): array
    {
        $errors = [];

        if (empty($this->emailId)) {
            $errors['email_id'] = 'Email ID is required';
        }

        return $errors;
    }
}
