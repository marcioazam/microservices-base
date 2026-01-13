<?php

declare(strict_types=1);

namespace EmailService\Tests\Property;

use EmailService\Infrastructure\Platform\ExceptionInfo;
use EmailService\Infrastructure\Platform\LogEntry;
use EmailService\Infrastructure\Platform\LogLevel;
use Eris\Generator;
use Eris\TestTrait;
use PHPUnit\Framework\TestCase;

/**
 * Feature: email-service-modernization-2025
 * Property 6: Structured Error Logging
 * 
 * For any exception thrown during email processing, the logged error entry
 * SHALL contain:
 * - Exception type
 * - Exception message
 * - Stack trace (when available)
 * - Correlation ID
 * 
 * Validates: Requirements 7.4
 */
class StructuredErrorLoggingPropertyTest extends TestCase
{
    use TestTrait;

    /**
     * @test
     * Property 6: Error log contains exception type
     */
    public function errorLogContainsExceptionType(): void
    {
        $exceptions = [
            new \RuntimeException('Runtime error'),
            new \InvalidArgumentException('Invalid argument'),
            new \LogicException('Logic error'),
            new \DomainException('Domain error'),
        ];

        foreach ($exceptions as $exception) {
            $correlationId = $this->generateUuid();
            $entry = LogEntry::error($correlationId, 'Error occurred', $exception);

            $this->assertNotNull($entry->exception);
            $this->assertEquals($exception::class, $entry->exception->type);

            $array = $entry->toArray();
            $this->assertEquals($exception::class, $array['exception']['type']);
        }
    }

    /**
     * @test
     * Property 6: Error log contains exception message
     */
    public function errorLogContainsExceptionMessage(): void
    {
        $this->forAll(
            Generator\suchThat(
                fn($s) => strlen($s) >= 1 && strlen($s) <= 200,
                Generator\string()
            )
        )
        ->withMaxSize(100)
        ->then(function (string $errorMessage): void {
            $exception = new \RuntimeException($errorMessage);
            $correlationId = $this->generateUuid();
            $entry = LogEntry::error($correlationId, 'Error occurred', $exception);

            $this->assertNotNull($entry->exception);
            $this->assertEquals($errorMessage, $entry->exception->message);

            $array = $entry->toArray();
            $this->assertEquals($errorMessage, $array['exception']['message']);
        });
    }

    /**
     * @test
     * Property 6: Error log contains stack trace
     */
    public function errorLogContainsStackTrace(): void
    {
        $exception = new \RuntimeException('Test error');
        $correlationId = $this->generateUuid();
        $entry = LogEntry::error($correlationId, 'Error occurred', $exception);

        $this->assertNotNull($entry->exception);
        $this->assertNotNull($entry->exception->stackTrace);
        $this->assertNotEmpty($entry->exception->stackTrace);

        $array = $entry->toArray();
        $this->assertArrayHasKey('stack_trace', $array['exception']);
        $this->assertNotEmpty($array['exception']['stack_trace']);
    }

    /**
     * @test
     * Property 6: Error log contains correlation ID
     */
    public function errorLogContainsCorrelationId(): void
    {
        $this->forAll(
            Generator\suchThat(
                fn($s) => strlen($s) >= 1 && strlen($s) <= 100,
                Generator\string()
            )
        )
        ->withMaxSize(100)
        ->then(function (string $errorMessage): void {
            $exception = new \RuntimeException($errorMessage);
            $correlationId = $this->generateUuid();
            $entry = LogEntry::error($correlationId, 'Error occurred', $exception);

            $this->assertEquals($correlationId, $entry->correlationId);

            $array = $entry->toArray();
            $this->assertArrayHasKey('correlation_id', $array);
            $this->assertEquals($correlationId, $array['correlation_id']);
        });
    }

    /**
     * @test
     * Property 6: Error log has ERROR level
     */
    public function errorLogHasErrorLevel(): void
    {
        $exception = new \RuntimeException('Test error');
        $correlationId = $this->generateUuid();
        $entry = LogEntry::error($correlationId, 'Error occurred', $exception);

        $this->assertEquals(LogLevel::ERROR, $entry->level);

        $array = $entry->toArray();
        $this->assertEquals(LogLevel::ERROR->value, $array['level']);
    }

    /**
     * @test
     * Property 6: Nested exceptions are captured
     */
    public function nestedExceptionsAreCaptured(): void
    {
        $innerException = new \InvalidArgumentException('Inner error');
        $outerException = new \RuntimeException('Outer error', 0, $innerException);

        $correlationId = $this->generateUuid();
        $entry = LogEntry::error($correlationId, 'Error occurred', $outerException);

        $this->assertNotNull($entry->exception);
        $this->assertEquals(\RuntimeException::class, $entry->exception->type);
        $this->assertEquals('Outer error', $entry->exception->message);

        $this->assertNotNull($entry->exception->innerException);
        $this->assertEquals(\InvalidArgumentException::class, $entry->exception->innerException->type);
        $this->assertEquals('Inner error', $entry->exception->innerException->message);

        $array = $entry->toArray();
        $this->assertArrayHasKey('inner_exception', $array['exception']);
        $this->assertEquals(\InvalidArgumentException::class, $array['exception']['inner_exception']['type']);
    }

    /**
     * @test
     * Property 6: ExceptionInfo fromThrowable captures all details
     */
    public function exceptionInfoFromThrowableCapturesAllDetails(): void
    {
        $this->forAll(
            Generator\suchThat(
                fn($s) => strlen($s) >= 1 && strlen($s) <= 100,
                Generator\string()
            )
        )
        ->withMaxSize(100)
        ->then(function (string $message): void {
            $exception = new \RuntimeException($message);
            $info = ExceptionInfo::fromThrowable($exception);

            $this->assertEquals(\RuntimeException::class, $info->type);
            $this->assertEquals($message, $info->message);
            $this->assertNotNull($info->stackTrace);
            $this->assertNull($info->innerException);
        });
    }

    /**
     * @test
     * Property 6: ExceptionInfo toArray produces valid structure
     */
    public function exceptionInfoToArrayProducesValidStructure(): void
    {
        $exception = new \RuntimeException('Test error');
        $info = ExceptionInfo::fromThrowable($exception);

        $array = $info->toArray();

        $this->assertArrayHasKey('type', $array);
        $this->assertArrayHasKey('message', $array);
        $this->assertArrayHasKey('stack_trace', $array);

        $this->assertEquals(\RuntimeException::class, $array['type']);
        $this->assertEquals('Test error', $array['message']);
        $this->assertNotEmpty($array['stack_trace']);
    }

    /**
     * @test
     * Property 6: Deeply nested exceptions are captured
     */
    public function deeplyNestedExceptionsAreCaptured(): void
    {
        $level3 = new \LogicException('Level 3');
        $level2 = new \InvalidArgumentException('Level 2', 0, $level3);
        $level1 = new \RuntimeException('Level 1', 0, $level2);

        $info = ExceptionInfo::fromThrowable($level1);

        $this->assertEquals('Level 1', $info->message);
        $this->assertNotNull($info->innerException);
        $this->assertEquals('Level 2', $info->innerException->message);
        $this->assertNotNull($info->innerException->innerException);
        $this->assertEquals('Level 3', $info->innerException->innerException->message);
    }

    /**
     * @test
     * Property 6: Error log without exception still has correlation ID
     */
    public function errorLogWithoutExceptionStillHasCorrelationId(): void
    {
        $correlationId = $this->generateUuid();
        $entry = LogEntry::error($correlationId, 'Error without exception');

        $this->assertEquals($correlationId, $entry->correlationId);
        $this->assertNull($entry->exception);

        $array = $entry->toArray();
        $this->assertArrayHasKey('correlation_id', $array);
        $this->assertArrayNotHasKey('exception', $array);
    }

    /**
     * @test
     * Property 6: All required fields present in error log
     */
    public function allRequiredFieldsPresentInErrorLog(): void
    {
        $exception = new \RuntimeException('Test error');
        $correlationId = $this->generateUuid();
        $entry = LogEntry::error($correlationId, 'Error occurred', $exception);

        $array = $entry->toArray();

        // Required fields
        $this->assertArrayHasKey('correlation_id', $array);
        $this->assertArrayHasKey('service_id', $array);
        $this->assertArrayHasKey('level', $array);
        $this->assertArrayHasKey('message', $array);
        $this->assertArrayHasKey('timestamp', $array);
        $this->assertArrayHasKey('exception', $array);

        // Exception fields
        $this->assertArrayHasKey('type', $array['exception']);
        $this->assertArrayHasKey('message', $array['exception']);
        $this->assertArrayHasKey('stack_trace', $array['exception']);
    }

    private function generateUuid(): string
    {
        return sprintf(
            '%s-%s-%s-%s-%s',
            bin2hex(random_bytes(4)),
            bin2hex(random_bytes(2)),
            bin2hex(random_bytes(2)),
            bin2hex(random_bytes(2)),
            bin2hex(random_bytes(6))
        );
    }
}
