<?php

declare(strict_types=1);

namespace EmailService\Tests\Property;

use DateTimeImmutable;
use EmailService\Application\DTO\AuditQuery;
use EmailService\Application\Service\AuditService;
use EmailService\Application\Util\PiiMasker;
use EmailService\Domain\Entity\AuditLog;
use EmailService\Domain\Entity\Email;
use EmailService\Domain\Enum\AuditAction;
use EmailService\Domain\Enum\EmailStatus;
use EmailService\Domain\ValueObject\Recipient;
use EmailService\Infrastructure\Platform\LoggingClientInterface;
use EmailService\Infrastructure\Repository\AuditLogRepositoryInterface;
use Eris\Generator;
use Eris\TestTrait;
use PHPUnit\Framework\TestCase;

/**
 * Property 19: Audit Log Completeness
 * For any email operation, the Audit_Logger SHALL create a log entry containing:
 * emailId, action, senderId, recipientEmail, subject, status, timestamp, and errorMessage.
 *
 * Property 24: PII Masking in Logs
 * For any log entry containing email addresses, the logged value SHALL be masked.
 *
 * Property 20: Audit Log Filtering
 * For any audit query with filters, the returned results SHALL contain only entries
 * matching ALL specified filter criteria.
 *
 * Validates: Requirements 5.1, 5.2, 5.4, 7.5
 */
final class AuditLoggingPropertyTest extends TestCase
{
    use TestTrait;

    private AuditService $auditService;
    private AuditLogRepositoryInterface $repository;
    private LoggingClientInterface $loggingClient;

    protected function setUp(): void
    {
        $this->repository = $this->createInMemoryRepository();
        $this->loggingClient = $this->createMock(LoggingClientInterface::class);
        $this->loggingClient->method('log')->willReturn(null);
        $this->loggingClient->method('isHealthy')->willReturn(true);

        $this->auditService = new AuditService($this->loggingClient, $this->repository);
    }

    /**
     * @test
     * Property 19: Audit log contains all required fields for email created
     */
    public function auditLogContainsAllRequiredFieldsForEmailCreated(): void
    {
        $this->forAll(
            Generator\suchThat(
                fn($s) => strlen($s) >= 1 && strlen($s) <= 20 && preg_match('/^[a-z]+$/', $s),
                Generator\string()
            ),
            Generator\suchThat(
                fn($s) => strlen($s) >= 1 && strlen($s) <= 50 && preg_match('/^[a-zA-Z0-9 ]+$/', $s),
                Generator\string()
            )
        )
            ->withMaxSize(100)
            ->then(function (string $localPart, string $subject): void {
                $email = Email::create(
                    from: new Recipient('sender@example.com'),
                    recipients: [new Recipient("{$localPart}@example.com")],
                    subject: $subject,
                    body: '<p>Test body</p>'
                );

                $senderId = 'user-123';
                $auditLog = AuditLog::forEmailCreated($email, $senderId);

                $this->assertNotEmpty($auditLog->id);
                $this->assertEquals($email->id, $auditLog->emailId);
                $this->assertNotNull($auditLog->action);
                $this->assertEquals($senderId, $auditLog->senderId);
                $this->assertNotEmpty($auditLog->recipientEmail);
                $this->assertEquals($subject, $auditLog->subject);
                $this->assertNotNull($auditLog->status);
                $this->assertInstanceOf(DateTimeImmutable::class, $auditLog->timestamp);
            });
    }

    /**
     * @test
     * Property 19: Audit log contains error message for failed emails
     */
    public function auditLogContainsErrorMessageForFailedEmails(): void
    {
        $this->forAll(
            Generator\suchThat(
                fn($s) => strlen($s) >= 1 && strlen($s) <= 100 && preg_match('/^[a-zA-Z0-9 ]+$/', $s),
                Generator\string()
            )
        )
            ->withMaxSize(100)
            ->then(function (string $errorMessage): void {
                $email = Email::create(
                    from: new Recipient('sender@example.com'),
                    recipients: [new Recipient('recipient@example.com')],
                    subject: 'Test',
                    body: 'Body'
                );

                $auditLog = AuditLog::forEmailFailed($email, 'user-123', $errorMessage, 'SendGrid');

                $this->assertEquals($errorMessage, $auditLog->errorMessage);
                $this->assertEquals(EmailStatus::FAILED, $auditLog->status);
                $this->assertEquals('SendGrid', $auditLog->providerName);
            });
    }

    /**
     * @test
     * Property 24: Email addresses are masked using PiiMasker
     */
    public function emailAddressesAreMaskedUsingPiiMasker(): void
    {
        $testCases = [
            'john.doe@example.com' => 'j***@example.com',
            'a@example.com' => '*@example.com',
            'ab@example.com' => 'a***@example.com',
            'test@domain.org' => 't***@domain.org',
        ];

        foreach ($testCases as $original => $expected) {
            $masked = PiiMasker::maskEmail($original);
            $this->assertEquals($expected, $masked);
        }
    }

    /**
     * @test
     * Property 24: Masked emails don't reveal full local part
     */
    public function maskedEmailsDontRevealFullLocalPart(): void
    {
        $this->forAll(
            Generator\suchThat(
                fn($s) => strlen($s) >= 2 && strlen($s) <= 30 && preg_match('/^[a-z]+$/', $s),
                Generator\string()
            )
        )
            ->withMaxSize(100)
            ->then(function (string $localPart): void {
                $email = "{$localPart}@example.com";
                $masked = PiiMasker::maskEmail($email);

                $this->assertStringNotContainsString($localPart, $masked);
                $this->assertStringContainsString('@example.com', $masked);
                $this->assertStringStartsWith($localPart[0], $masked);
            });
    }

    /**
     * @test
     * Property 20: Filtering by status returns only matching entries
     */
    public function filteringByStatusReturnsOnlyMatchingEntries(): void
    {
        $this->repository->clear();

        $email = Email::create(
            from: new Recipient('sender@example.com'),
            recipients: [new Recipient('recipient@example.com')],
            subject: 'Test',
            body: 'Body'
        );

        $logPending = new AuditLog(
            id: 'log-pending',
            emailId: $email->id,
            action: AuditAction::CREATED,
            senderId: 'user-1',
            recipientEmail: 'r***@example.com',
            subject: 'Test',
            status: EmailStatus::PENDING
        );

        $logSent = new AuditLog(
            id: 'log-sent',
            emailId: $email->id,
            action: AuditAction::SENT,
            senderId: 'user-1',
            recipientEmail: 'r***@example.com',
            subject: 'Test',
            status: EmailStatus::SENT
        );

        $logFailed = new AuditLog(
            id: 'log-failed',
            emailId: $email->id,
            action: AuditAction::FAILED,
            senderId: 'user-1',
            recipientEmail: 'r***@example.com',
            subject: 'Test',
            status: EmailStatus::FAILED,
            errorMessage: 'Provider error'
        );

        $this->auditService->log($logPending);
        $this->auditService->log($logSent);
        $this->auditService->log($logFailed);

        $query = new AuditQuery(status: EmailStatus::FAILED);
        $result = $this->auditService->query($query);

        $this->assertCount(1, $result->items);
        $this->assertEquals(EmailStatus::FAILED, $result->items[0]->status);
    }

    /**
     * @test
     * Property 20: Filtering by sender returns only matching entries
     */
    public function filteringBySenderReturnsOnlyMatchingEntries(): void
    {
        $this->repository->clear();

        $email = Email::create(
            from: new Recipient('sender@example.com'),
            recipients: [new Recipient('recipient@example.com')],
            subject: 'Test',
            body: 'Body'
        );

        $logUser1 = new AuditLog(
            id: 'log-user1',
            emailId: $email->id,
            action: AuditAction::CREATED,
            senderId: 'user-1',
            recipientEmail: 'r***@example.com',
            subject: 'Test',
            status: EmailStatus::PENDING
        );

        $logUser2 = new AuditLog(
            id: 'log-user2',
            emailId: $email->id,
            action: AuditAction::CREATED,
            senderId: 'user-2',
            recipientEmail: 'r***@example.com',
            subject: 'Test',
            status: EmailStatus::PENDING
        );

        $this->auditService->log($logUser1);
        $this->auditService->log($logUser2);

        $query = new AuditQuery(senderId: 'user-1');
        $result = $this->auditService->query($query);

        $this->assertCount(1, $result->items);
        $this->assertEquals('user-1', $result->items[0]->senderId);
    }

    /**
     * @test
     * Property: getByEmailId returns all logs for an email
     */
    public function getByEmailIdReturnsAllLogsForAnEmail(): void
    {
        $this->repository->clear();

        $email = Email::create(
            from: new Recipient('sender@example.com'),
            recipients: [new Recipient('recipient@example.com')],
            subject: 'Test',
            body: 'Body'
        );

        $log1 = AuditLog::forEmailCreated($email, 'user-1');
        $log2 = AuditLog::forEmailQueued($email, 'user-1');
        $log3 = AuditLog::forEmailSent($email, 'user-1', 'SendGrid', 'msg-123');

        $this->auditService->log($log1);
        $this->auditService->log($log2);
        $this->auditService->log($log3);

        $logs = $this->auditService->getByEmailId($email->id);

        $this->assertCount(3, $logs);
    }

    private function createInMemoryRepository(): AuditLogRepositoryInterface
    {
        return new class implements AuditLogRepositoryInterface {
            /** @var AuditLog[] */
            private array $logs = [];

            public function save(AuditLog $log): void
            {
                $this->logs[$log->id] = $log;
            }

            public function findByEmailId(string $emailId): array
            {
                return array_values(array_filter(
                    $this->logs,
                    fn(AuditLog $log) => $log->emailId === $emailId
                ));
            }

            public function query(AuditQuery $query): array
            {
                return array_values(array_filter($this->logs, function (AuditLog $log) use ($query) {
                    if ($query->senderId !== null && $log->senderId !== $query->senderId) {
                        return false;
                    }
                    if ($query->status !== null && $log->status !== $query->status) {
                        return false;
                    }
                    return true;
                }));
            }

            public function clear(): void
            {
                $this->logs = [];
            }
        };
    }
}
