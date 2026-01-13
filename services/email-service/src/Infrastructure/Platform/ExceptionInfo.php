<?php

declare(strict_types=1);

namespace EmailService\Infrastructure\Platform;

/**
 * Exception information for structured logging.
 */
final readonly class ExceptionInfo
{
    public function __construct(
        public string $type,
        public string $message,
        public ?string $stackTrace = null,
        public ?self $innerException = null,
    ) {}

    public static function fromThrowable(\Throwable $e): self
    {
        $inner = $e->getPrevious() !== null
            ? self::fromThrowable($e->getPrevious())
            : null;

        return new self(
            type: $e::class,
            message: $e->getMessage(),
            stackTrace: $e->getTraceAsString(),
            innerException: $inner,
        );
    }

    public function toArray(): array
    {
        $data = [
            'type' => $this->type,
            'message' => $this->message,
        ];

        if ($this->stackTrace !== null) {
            $data['stack_trace'] = $this->stackTrace;
        }

        if ($this->innerException !== null) {
            $data['inner_exception'] = $this->innerException->toArray();
        }

        return $data;
    }
}
