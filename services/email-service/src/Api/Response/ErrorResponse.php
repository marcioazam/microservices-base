<?php

declare(strict_types=1);

namespace EmailService\Api\Response;

/**
 * Centralized error response factory.
 * Single source of truth for all error response formats.
 */
final readonly class ErrorResponse implements \JsonSerializable
{
    private function __construct(
        public string $code,
        public string $message,
        public ?string $correlationId = null,
        public array $details = [],
        public ?int $retryAfter = null,
    ) {}

    public static function validation(string $message, array $details = [], ?string $correlationId = null): self
    {
        return new self(
            code: 'VALIDATION_ERROR',
            message: $message,
            correlationId: $correlationId,
            details: $details,
        );
    }

    public static function invalidEmailFormat(string $email, ?string $correlationId = null): self
    {
        return new self(
            code: 'INVALID_EMAIL_FORMAT',
            message: 'Email format does not match RFC 5322',
            correlationId: $correlationId,
            details: ['email' => $email],
        );
    }

    public static function disposableEmail(string $domain, ?string $correlationId = null): self
    {
        return new self(
            code: 'DISPOSABLE_EMAIL',
            message: 'Disposable email domains are not allowed',
            correlationId: $correlationId,
            details: ['domain' => $domain],
        );
    }

    public static function invalidDomain(string $domain, ?string $correlationId = null): self
    {
        return new self(
            code: 'INVALID_DOMAIN',
            message: 'Domain has no valid MX records',
            correlationId: $correlationId,
            details: ['domain' => $domain],
        );
    }

    public static function rateLimit(int $retryAfter, ?string $correlationId = null): self
    {
        return new self(
            code: 'RATE_LIMIT_EXCEEDED',
            message: 'Rate limit exceeded. Please retry later.',
            correlationId: $correlationId,
            retryAfter: $retryAfter,
        );
    }

    public static function notFound(string $resource, ?string $id = null, ?string $correlationId = null): self
    {
        $details = $id !== null ? ['id' => $id] : [];

        return new self(
            code: 'NOT_FOUND',
            message: "{$resource} not found",
            correlationId: $correlationId,
            details: $details,
        );
    }

    public static function emailNotFound(string $emailId, ?string $correlationId = null): self
    {
        return new self(
            code: 'EMAIL_NOT_FOUND',
            message: 'Email not found',
            correlationId: $correlationId,
            details: ['email_id' => $emailId],
        );
    }

    public static function internal(?string $correlationId = null): self
    {
        return new self(
            code: 'INTERNAL_ERROR',
            message: 'An internal error occurred. Please contact support if the problem persists.',
            correlationId: $correlationId,
        );
    }

    public static function provider(string $providerName, string $message, ?string $correlationId = null): self
    {
        return new self(
            code: 'PROVIDER_ERROR',
            message: "Email provider error: {$message}",
            correlationId: $correlationId,
            details: ['provider' => $providerName],
        );
    }

    public static function unauthorized(?string $correlationId = null): self
    {
        return new self(
            code: 'UNAUTHORIZED',
            message: 'Authentication required',
            correlationId: $correlationId,
        );
    }

    public static function forbidden(?string $correlationId = null): self
    {
        return new self(
            code: 'FORBIDDEN',
            message: 'Access denied',
            correlationId: $correlationId,
        );
    }

    public static function serviceUnavailable(string $service, ?string $correlationId = null): self
    {
        return new self(
            code: 'SERVICE_UNAVAILABLE',
            message: "Service temporarily unavailable: {$service}",
            correlationId: $correlationId,
            details: ['service' => $service],
        );
    }

    public function getHttpStatusCode(): int
    {
        return match ($this->code) {
            'VALIDATION_ERROR', 'INVALID_EMAIL_FORMAT', 'DISPOSABLE_EMAIL', 'INVALID_DOMAIN' => 400,
            'UNAUTHORIZED' => 401,
            'FORBIDDEN' => 403,
            'NOT_FOUND', 'EMAIL_NOT_FOUND' => 404,
            'RATE_LIMIT_EXCEEDED' => 429,
            'PROVIDER_ERROR' => 502,
            'SERVICE_UNAVAILABLE' => 503,
            default => 500,
        };
    }

    public function jsonSerialize(): array
    {
        $data = [
            'error' => [
                'code' => $this->code,
                'message' => $this->message,
            ],
        ];

        if ($this->correlationId !== null) {
            $data['error']['correlation_id'] = $this->correlationId;
        }

        if (!empty($this->details)) {
            $data['error']['details'] = $this->details;
        }

        if ($this->retryAfter !== null) {
            $data['error']['retry_after'] = $this->retryAfter;
        }

        return $data;
    }

    public function toJson(): string
    {
        return json_encode($this->jsonSerialize(), JSON_THROW_ON_ERROR | JSON_UNESCAPED_SLASHES);
    }
}
