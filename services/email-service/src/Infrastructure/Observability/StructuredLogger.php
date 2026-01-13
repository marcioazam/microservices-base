<?php

declare(strict_types=1);

namespace EmailService\Infrastructure\Observability;

use Psr\Log\LoggerInterface;
use Psr\Log\LogLevel;

class StructuredLogger implements LoggerInterface
{
    private string $correlationId;
    private string $serviceName = 'email-service';

    /** @var array<string, string> */
    private static array $piiPatterns = [
        'email' => '/[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}/',
        'phone' => '/\b\d{3}[-.]?\d{3}[-.]?\d{4}\b/',
        'ip' => '/\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b/',
    ];

    public function __construct(?string $correlationId = null)
    {
        $this->correlationId = $correlationId ?? $this->generateCorrelationId();
    }

    public function emergency(\Stringable|string $message, array $context = []): void
    {
        $this->log(LogLevel::EMERGENCY, $message, $context);
    }

    public function alert(\Stringable|string $message, array $context = []): void
    {
        $this->log(LogLevel::ALERT, $message, $context);
    }

    public function critical(\Stringable|string $message, array $context = []): void
    {
        $this->log(LogLevel::CRITICAL, $message, $context);
    }

    public function error(\Stringable|string $message, array $context = []): void
    {
        $this->log(LogLevel::ERROR, $message, $context);
    }

    public function warning(\Stringable|string $message, array $context = []): void
    {
        $this->log(LogLevel::WARNING, $message, $context);
    }

    public function notice(\Stringable|string $message, array $context = []): void
    {
        $this->log(LogLevel::NOTICE, $message, $context);
    }

    public function info(\Stringable|string $message, array $context = []): void
    {
        $this->log(LogLevel::INFO, $message, $context);
    }

    public function debug(\Stringable|string $message, array $context = []): void
    {
        $this->log(LogLevel::DEBUG, $message, $context);
    }

    public function log($level, \Stringable|string $message, array $context = []): void
    {
        $entry = $this->buildLogEntry($level, (string) $message, $context);
        $this->write($entry);
    }

    public function getCorrelationId(): string
    {
        return $this->correlationId;
    }

    public function setCorrelationId(string $correlationId): void
    {
        $this->correlationId = $correlationId;
    }

    private function buildLogEntry(string $level, string $message, array $context): array
    {
        $maskedContext = $this->maskPii($context);
        $maskedMessage = $this->maskPiiInString($message);

        return [
            'timestamp' => gmdate('Y-m-d\TH:i:s\Z'),
            'level' => strtoupper($level),
            'service' => $this->serviceName,
            'correlation_id' => $this->correlationId,
            'message' => $maskedMessage,
            'context' => $maskedContext,
        ];
    }

    private function maskPii(array $data): array
    {
        $masked = [];
        foreach ($data as $key => $value) {
            if (is_array($value)) {
                $masked[$key] = $this->maskPii($value);
            } elseif (is_string($value)) {
                $masked[$key] = $this->maskPiiInString($value);
            } else {
                $masked[$key] = $value;
            }
        }
        return $masked;
    }

    private function maskPiiInString(string $value): string
    {
        foreach (self::$piiPatterns as $type => $pattern) {
            $value = preg_replace_callback($pattern, fn($m) => $this->maskValue($m[0], $type), $value);
        }
        return $value;
    }

    private function maskValue(string $value, string $type): string
    {
        return match ($type) {
            'email' => $this->maskEmail($value),
            'phone' => '***-***-' . substr($value, -4),
            'ip' => '***.***.***.***',
            default => '***',
        };
    }

    private function maskEmail(string $email): string
    {
        $parts = explode('@', $email);
        if (count($parts) !== 2) {
            return '***@***.***';
        }
        $local = $parts[0];
        $domain = $parts[1];
        $maskedLocal = strlen($local) > 2 ? substr($local, 0, 2) . '***' : '***';
        return $maskedLocal . '@' . $domain;
    }

    private function generateCorrelationId(): string
    {
        return sprintf('%s-%s', bin2hex(random_bytes(8)), time());
    }

    protected function write(array $entry): void
    {
        fwrite(STDOUT, json_encode($entry, JSON_UNESCAPED_SLASHES) . "\n");
    }
}
