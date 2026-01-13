<?php

declare(strict_types=1);

namespace EmailService\Tests\Property;

use EmailService\Infrastructure\Platform\BatchLogResult;
use EmailService\Infrastructure\Platform\ExceptionInfo;
use EmailService\Infrastructure\Platform\FallbackLogger;
use EmailService\Infrastructure\Platform\LogEntry;
use EmailService\Infrastructure\Platform\LogLevel;
use Eris\Generator;
use Eris\TestTrait;
use PHPUnit\Framework\TestCase;

/**
 * Feature: email-service-modernization-2025
 * Property 1: Logging Integration Round-Trip
 * 
 * For any log entry with valid fields (correlationId, serviceId, level, message),
 * when sent to the LoggingClient, the entry SHALL be transmitted to the
 * Logging_Service with all fields preserved.
 * 
 * Validates: Requirements 1.1
 */
class LoggingIntegrationPropertyTest extends TestCase
{
    use TestTrait;

    private string $testLogPath;
    private FallbackLogger $logger;

    protected function setUp(): void
    {
        $this->testLogPath = sys_get_temp_dir() . '/email-service-test-' . uniqid() . '.json';
        $this->logger = new FallbackLogger($this->testLogPath);
    }

    protected function tearDown(): void
    {
        if (file_exists($this->testLogPath)) {
            unlink($this->testLogPath);
        }
    }

    /**
     * @test
     * Property 1: Log entry preserves all required fields
     */
    public function logEntryPreservesAllRequiredFields(): void
    {
        $this->forAll(
            Generator\suchThat(
                fn($s) => strlen($s) === 36 && preg_match('/^[a-f0-9-]+$/', $s),
                Generator\map(
                    fn() => sprintf(
                        '%s-%s-%s-%s-%s',
                        bin2hex(random_bytes(4)),
                        bin2hex(random_bytes(2)),
                        bin2hex(random_bytes(2)),
                        bin2hex(random_bytes(2)),
                        bin2hex(random_bytes(6))
                    ),
                    Generator\constant(null)
                )
            ),
            Generator\suchThat(
                fn($s) => strlen($s) >= 1 && strlen($s) <= 200,
                Generator\string()
            )
        )
        ->withMaxSize(100)
        ->then(function (string $correlationId, string $message): void {
            $entry = LogEntry::info($correlationId, $message);
            
            $this->logger->log($entry);
            
            // Read back from file
            $content = file_get_contents($this->testLogPath);
            $logged = json_decode(trim($content), true);
            
            $this->assertEquals($correlationId, $logged['correlation_id']);
            $this->assertEquals('email-service', $logged['service_id']);
            $this->assertEquals(LogLevel::INFO->value, $logged['level']);
            $this->assertEquals($message, $logged['message']);
            $this->assertArrayHasKey('timestamp', $logged);
            
            // Clean up for next iteration
            file_put_contents($this->testLogPath, '');
        });
    }

    /**
     * @test
     * Property 1: Log entry preserves metadata
     */
    public function logEntryPreservesMetadata(): void
    {
        $correlationId = 'test-correlation-id-123';
        $message = 'Test message with metadata';
        $metadata = [
            'email_id' => 'email-123',
            'recipient' => 'masked@example.com',
            'provider' => 'sendgrid',
        ];

        $entry = new LogEntry(
            correlationId: $correlationId,
            serviceId: 'email-service',
            level: LogLevel::INFO,
            message: $message,
            metadata: $metadata,
        );

        $this->logger->log($entry);

        $content = file_get_contents($this->testLogPath);
        $logged = json_decode(trim($content), true);

        $this->assertEquals($metadata, $logged['metadata']);
    }

    /**
     * @test
     * Property 1: Error log entry preserves exception info
     */
    public function errorLogEntryPreservesExceptionInfo(): void
    {
        $correlationId = 'error-correlation-id';
        $message = 'An error occurred';
        $exception = new \RuntimeException('Test exception message');

        $entry = LogEntry::error($correlationId, $message, $exception);

        $this->logger->log($entry);

        $content = file_get_contents($this->testLogPath);
        $logged = json_decode(trim($content), true);

        $this->assertEquals(LogLevel::ERROR->value, $logged['level']);
        $this->assertArrayHasKey('exception', $logged);
        $this->assertEquals(\RuntimeException::class, $logged['exception']['type']);
        $this->assertEquals('Test exception message', $logged['exception']['message']);
        $this->assertArrayHasKey('stack_trace', $logged['exception']);
    }

    /**
     * @test
     * Property 1: All log levels are correctly preserved
     */
    public function allLogLevelsAreCorrectlyPreserved(): void
    {
        $correlationId = 'level-test-id';

        $levels = [
            ['method' => 'debug', 'level' => LogLevel::DEBUG],
            ['method' => 'info', 'level' => LogLevel::INFO],
            ['method' => 'warn', 'level' => LogLevel::WARN],
            ['method' => 'error', 'level' => LogLevel::ERROR],
        ];

        foreach ($levels as $levelData) {
            file_put_contents($this->testLogPath, '');

            $entry = match ($levelData['method']) {
                'debug' => LogEntry::debug($correlationId, 'Debug message'),
                'info' => LogEntry::info($correlationId, 'Info message'),
                'warn' => LogEntry::warn($correlationId, 'Warn message'),
                'error' => LogEntry::error($correlationId, 'Error message'),
            };

            $this->logger->log($entry);

            $content = file_get_contents($this->testLogPath);
            $logged = json_decode(trim($content), true);

            $this->assertEquals($levelData['level']->value, $logged['level']);
        }
    }

    /**
     * @test
     * Property 1: Batch logging processes all entries
     */
    public function batchLoggingProcessesAllEntries(): void
    {
        $this->forAll(
            Generator\choose(1, 10)
        )
        ->withMaxSize(100)
        ->then(function (int $count): void {
            file_put_contents($this->testLogPath, '');

            $entries = [];
            for ($i = 0; $i < $count; $i++) {
                $entries[] = LogEntry::info("correlation-{$i}", "Message {$i}");
            }

            $result = $this->logger->logBatch($entries);

            $this->assertInstanceOf(BatchLogResult::class, $result);
            $this->assertEquals($count, $result->acceptedCount);
            $this->assertEquals(0, $result->rejectedCount);
            $this->assertTrue($result->isFullySuccessful());

            // Verify all entries were written
            $content = file_get_contents($this->testLogPath);
            $lines = array_filter(explode("\n", trim($content)));
            $this->assertCount($count, $lines);
        });
    }

    /**
     * @test
     * Property 1: ExceptionInfo correctly captures nested exceptions
     */
    public function exceptionInfoCorrectlyCapturesNestedExceptions(): void
    {
        $inner = new \InvalidArgumentException('Inner exception');
        $outer = new \RuntimeException('Outer exception', 0, $inner);

        $exceptionInfo = ExceptionInfo::fromThrowable($outer);

        $this->assertEquals(\RuntimeException::class, $exceptionInfo->type);
        $this->assertEquals('Outer exception', $exceptionInfo->message);
        $this->assertNotNull($exceptionInfo->stackTrace);
        $this->assertNotNull($exceptionInfo->innerException);
        $this->assertEquals(\InvalidArgumentException::class, $exceptionInfo->innerException->type);
        $this->assertEquals('Inner exception', $exceptionInfo->innerException->message);
    }

    /**
     * @test
     * Property 1: LogEntry toArray includes all optional fields when set
     */
    public function logEntryToArrayIncludesAllOptionalFieldsWhenSet(): void
    {
        $entry = new LogEntry(
            correlationId: 'test-id',
            serviceId: 'email-service',
            level: LogLevel::INFO,
            message: 'Test message',
            traceId: 'trace-123',
            spanId: 'span-456',
            userId: 'user-789',
            requestId: 'request-abc',
            method: 'POST',
            path: '/api/emails',
            statusCode: 200,
            durationMs: 150,
            metadata: ['key' => 'value'],
        );

        $array = $entry->toArray();

        $this->assertEquals('trace-123', $array['trace_id']);
        $this->assertEquals('span-456', $array['span_id']);
        $this->assertEquals('user-789', $array['user_id']);
        $this->assertEquals('request-abc', $array['request_id']);
        $this->assertEquals('POST', $array['method']);
        $this->assertEquals('/api/emails', $array['path']);
        $this->assertEquals(200, $array['status_code']);
        $this->assertEquals(150, $array['duration_ms']);
        $this->assertEquals(['key' => 'value'], $array['metadata']);
    }

    /**
     * @test
     * Property 1: FallbackLogger health check works
     */
    public function fallbackLoggerHealthCheckWorks(): void
    {
        $this->assertTrue($this->logger->isHealthy());
    }

    /**
     * @test
     * Property 1: LogLevel toString returns correct string
     */
    public function logLevelToStringReturnsCorrectString(): void
    {
        $this->assertEquals('DEBUG', LogLevel::DEBUG->toString());
        $this->assertEquals('INFO', LogLevel::INFO->toString());
        $this->assertEquals('WARN', LogLevel::WARN->toString());
        $this->assertEquals('ERROR', LogLevel::ERROR->toString());
        $this->assertEquals('FATAL', LogLevel::FATAL->toString());
    }
}
