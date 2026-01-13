<?php

declare(strict_types=1);

namespace EmailService\Application\Service;

use EmailService\Application\DTO\EmailDTO;
use EmailService\Application\DTO\EmailResult;
use EmailService\Domain\Entity\AuditLog;
use EmailService\Domain\Entity\Email;
use EmailService\Domain\Enum\EmailStatus;
use EmailService\Domain\ValueObject\Attachment;
use EmailService\Domain\ValueObject\Recipient;
use EmailService\Infrastructure\Provider\ProviderRouter;
use EmailService\Infrastructure\Queue\EmailJob;
use EmailService\Infrastructure\Queue\QueueServiceInterface;
use EmailService\Infrastructure\RateLimiter\RateLimiterInterface;
use EmailService\Infrastructure\Repository\EmailRepositoryInterface;

class EmailService implements EmailServiceInterface
{
    public function __construct(
        private readonly ValidationServiceInterface $validationService,
        private readonly ?TemplateServiceInterface $templateService,
        private readonly ?AuditServiceInterface $auditService,
        private readonly ProviderRouter $providerRouter,
        private readonly QueueServiceInterface $queueService,
        private readonly RateLimiterInterface $rateLimiter,
        private readonly ?EmailRepositoryInterface $emailRepository = null,
        private readonly string $defaultSenderId = 'system'
    ) {
    }

    public function send(EmailDTO $dto): EmailResult
    {
        // Validate sender rate limit
        $rateLimitResult = $this->rateLimiter->check($this->defaultSenderId);
        if (!$rateLimitResult->isAllowed) {
            return EmailResult::failure('RATE_LIMIT_EXCEEDED', 'Rate limit exceeded');
        }

        // Validate recipients
        foreach ($dto->recipients as $recipientEmail) {
            $validationResult = $this->validationService->validateEmail($recipientEmail);
            if (!$validationResult->isValid) {
                return EmailResult::failure(
                    $validationResult->errorCode ?? 'VALIDATION_ERROR',
                    $validationResult->errorMessage ?? 'Validation failed'
                );
            }
        }

        // Build email entity
        $email = $this->buildEmail($dto);

        // Record rate limit hit
        $this->rateLimiter->hit($this->defaultSenderId);

        // Log creation
        $this->logAudit(AuditLog::forEmailCreated($email, $this->defaultSenderId));

        // Save email
        $this->saveEmail($email);

        // Send via provider
        try {
            $result = $this->providerRouter->send($email);

            if ($result->success) {
                $email->markAsSent($result->messageId);
                $this->saveEmail($email);
                
                $this->logAudit(AuditLog::forEmailSent(
                    $email,
                    $this->defaultSenderId,
                    $this->providerRouter->getProviderForEmail($email)->getName(),
                    $result->messageId
                ));

                return EmailResult::success($email->id, $result->messageId);
            }

            $email->markAsFailed($result->errorMessage ?? 'Unknown error');
            $this->saveEmail($email);
            
            $this->logAudit(AuditLog::forEmailFailed(
                $email,
                $this->defaultSenderId,
                $result->errorMessage ?? 'Unknown error'
            ));

            return EmailResult::failure(
                $result->errorCode ?? 'PROVIDER_ERROR',
                $result->errorMessage ?? 'Failed to send email'
            );
        } catch (\Exception $e) {
            $email->markAsFailed($e->getMessage());
            $this->saveEmail($email);
            
            $this->logAudit(AuditLog::forEmailFailed(
                $email,
                $this->defaultSenderId,
                $e->getMessage()
            ));

            return EmailResult::failure('SEND_ERROR', $e->getMessage());
        }
    }

    public function sendAsync(EmailDTO $dto): string
    {
        // Validate recipients
        foreach ($dto->recipients as $recipientEmail) {
            $validationResult = $this->validationService->validateEmail($recipientEmail);
            if (!$validationResult->isValid) {
                throw new \InvalidArgumentException(
                    $validationResult->errorMessage ?? 'Invalid recipient'
                );
            }
        }

        // Build email entity
        $email = $this->buildEmail($dto);
        $email->markAsQueued();

        // Save email
        $this->saveEmail($email);

        // Create job and enqueue
        $job = EmailJob::create($email);
        $this->queueService->enqueue($job);

        // Log queued
        $this->logAudit(AuditLog::forEmailQueued($email, $this->defaultSenderId));

        return $email->id;
    }

    public function resend(string $emailId): EmailResult
    {
        $email = $this->emailRepository?->findById($emailId);
        
        if ($email === null) {
            return EmailResult::failure('EMAIL_NOT_FOUND', "Email not found: {$emailId}");
        }

        if (!$email->canRetry()) {
            return EmailResult::failure('CANNOT_RESEND', 'Email cannot be resent');
        }

        // Create new job for resend
        $job = EmailJob::create($email);
        $this->queueService->enqueue($job);

        $this->logAudit(AuditLog::forEmailRetried($email, $this->defaultSenderId, $email->getAttempts() + 1));

        return EmailResult::queued($email->id);
    }

    public function getStatus(string $emailId): EmailStatus
    {
        $email = $this->emailRepository?->findById($emailId);
        
        if ($email === null) {
            throw new \RuntimeException("Email not found: {$emailId}");
        }

        return $email->getStatus();
    }

    private function buildEmail(EmailDTO $dto): Email
    {
        $from = new Recipient($dto->from, $dto->fromName);
        
        $recipients = array_map(
            fn(string $email) => new Recipient($email),
            $dto->recipients
        );

        $cc = array_map(
            fn(string $email) => new Recipient($email),
            $dto->cc
        );

        $bcc = array_map(
            fn(string $email) => new Recipient($email),
            $dto->bcc
        );

        $attachments = [];
        foreach ($dto->attachments as $filename => $base64Content) {
            $attachments[] = Attachment::fromBase64(
                $filename,
                $base64Content,
                'application/octet-stream'
            );
        }

        // Render template if specified
        $subject = $dto->subject;
        $body = $dto->body;

        if ($dto->templateId !== null && $this->templateService !== null) {
            $rendered = $this->templateService->render($dto->templateId, $dto->templateVariables);
            $subject = $rendered->subject;
            $body = $rendered->bodyHtml;
        }

        return Email::create(
            from: $from,
            recipients: $recipients,
            subject: $subject,
            body: $body,
            contentType: $dto->contentType,
            cc: $cc,
            bcc: $bcc,
            attachments: $attachments,
            metadata: $dto->metadata,
            type: $dto->type,
            templateId: $dto->templateId
        );
    }

    private function saveEmail(Email $email): void
    {
        $this->emailRepository?->save($email);
    }

    private function logAudit(AuditLog $log): void
    {
        $this->auditService?->log($log);
    }
}
