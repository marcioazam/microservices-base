<?php

declare(strict_types=1);

namespace EmailService\Infrastructure\Platform;

/**
 * Structured log entry for platform logging-service.
 */
final readonly class LogEntry
{
    public function __construct(
        public string $correlationId,
        public string $serviceId,
        public LogLevel $level,
        public string $message,
        public ?string $traceId = null,
        public ?string $spanId = null,
        public ?string $userId = null,
        public ?string $requestId = null,
        public ?string $method = null,
        public ?string $path = null,
        public ?int $statusCode = null,
        public ?int $durationMs = null,
        public array $metadata = [],
        public ?ExceptionInfo $exception = null,
        public ?\DateTimeImmutable $timestamp = null,
    ) {}

    public static function info(string $correlationId, string $message, array $metadata = []): self
    {
        return new self(
            correlationId: $correlationId,
            serviceId: 'email-service',
            level: LogLevel::INFO,
            message: $message,
            metadata: $metadata,
            timestamp: new \DateTimeImmutable(),
        );
    }

    public static function error(
        string $correlationId,
        string $message,
        ?\Throwable $exception = null,
        array $metadata = [],
    ): self {
        return new self(
            correlationId: $correlationId,
            serviceId: 'email-service',
            level: LogLevel::ERROR,
            message: $message,
            metadata: $metadata,
            exception: $exception !== null ? ExceptionInfo::fromThrowable($exception) : null,
            timestamp: new \DateTimeImmutable(),
        );
    }

    public static function warn(string $correlationId, string $message, array $metadata = []): self
    {
        return new self(
            correlationId: $correlationId,
            serviceId: 'email-service',
            level: LogLevel::WARN,
            message: $message,
            metadata: $metadata,
            timestamp: new \DateTimeImmutable(),
        );
    }

    public static function debug(string $correlationId, string $message, array $metadata = []): self
    {
        return new self(
            correlationId: $correlationId,
            serviceId: 'email-service',
            level: LogLevel::DEBUG,
            message: $message,
            metadata: $metadata,
            timestamp: new \DateTimeImmutable(),
        );
    }

    public function toArray(): array
    {
        $data = [
            'correlation_id' => $this->correlationId,
            'service_id' => $this->serviceId,
            'level' => $this->level->value,
            'message' => $this->message,
            'timestamp' => ($this->timestamp ?? new \DateTimeImmutable())->format(\DateTimeInterface::RFC3339_EXTENDED),
        ];

        if ($this->traceId !== null) {
            $data['trace_id'] = $this->traceId;
        }
        if ($this->spanId !== null) {
            $data['span_id'] = $this->spanId;
        }
        if ($this->userId !== null) {
            $data['user_id'] = $this->userId;
        }
        if ($this->requestId !== null) {
            $data['request_id'] = $this->requestId;
        }
        if ($this->method !== null) {
            $data['method'] = $this->method;
        }
        if ($this->path !== null) {
            $data['path'] = $this->path;
        }
        if ($this->statusCode !== null) {
            $data['status_code'] = $this->statusCode;
        }
        if ($this->durationMs !== null) {
            $data['duration_ms'] = $this->durationMs;
        }
        if (!empty($this->metadata)) {
            $data['metadata'] = $this->metadata;
        }
        if ($this->exception !== null) {
            $data['exception'] = $this->exception->toArray();
        }

        return $data;
    }
}
