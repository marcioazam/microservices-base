<?php

declare(strict_types=1);

namespace EmailService\Application\DTO;

final readonly class DomainValidationResult
{
    /**
     * @param string[] $mxRecords
     */
    public function __construct(
        public bool $isValid,
        public array $mxRecords = [],
        public ?string $errorMessage = null
    ) {
    }

    public static function valid(array $mxRecords): self
    {
        return new self(isValid: true, mxRecords: $mxRecords);
    }

    public static function invalid(string $errorMessage): self
    {
        return new self(isValid: false, errorMessage: $errorMessage);
    }
}
