<?php

declare(strict_types=1);

namespace EmailService\Infrastructure\Platform;

/**
 * Fallback logger for when platform logging-service is unavailable.
 * Writes structured JSON logs to local file.
 */
final class FallbackLogger implements LoggingClientInterface
{
    private const DEFAULT_LOG_PATH = '/var/log/email-service/fallback.json';

    /** @var LogEntry[] */
    private array $buffer = [];

    public function __construct(
        private readonly string $logPath = self::DEFAULT_LOG_PATH,
        private readonly int $bufferSize = 100,
        private readonly bool $immediateFlush = true,
    ) {}

    public function log(LogEntry $entry): void
    {
        if ($this->immediateFlush) {
            $this->writeToFile($entry);
            return;
        }

        $this->buffer[] = $entry;

        if (count($this->buffer) >= $this->bufferSize) {
            $this->flush();
        }
    }

    public function logBatch(array $entries): BatchLogResult
    {
        $accepted = 0;
        $errors = [];

        foreach ($entries as $entry) {
            try {
                $this->log($entry);
                $accepted++;
            } catch (\Throwable $e) {
                $errors[] = $e->getMessage();
            }
        }

        $rejected = count($entries) - $accepted;

        return new BatchLogResult($accepted, $rejected, $errors);
    }

    public function isHealthy(): bool
    {
        $dir = dirname($this->logPath);

        if (!is_dir($dir)) {
            return @mkdir($dir, 0755, true);
        }

        return is_writable($dir);
    }

    public function flush(): void
    {
        foreach ($this->buffer as $entry) {
            $this->writeToFile($entry);
        }

        $this->buffer = [];
    }

    public function getBuffer(): array
    {
        return $this->buffer;
    }

    public function clearBuffer(): void
    {
        $this->buffer = [];
    }

    private function writeToFile(LogEntry $entry): void
    {
        $json = json_encode($entry->toArray(), JSON_THROW_ON_ERROR | JSON_UNESCAPED_SLASHES);
        $line = $json . PHP_EOL;

        $dir = dirname($this->logPath);
        if (!is_dir($dir)) {
            @mkdir($dir, 0755, true);
        }

        file_put_contents($this->logPath, $line, FILE_APPEND | LOCK_EX);
    }
}
