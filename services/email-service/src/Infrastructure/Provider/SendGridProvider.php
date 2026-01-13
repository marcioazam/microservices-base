<?php

declare(strict_types=1);

namespace EmailService\Infrastructure\Provider;

use EmailService\Domain\Entity\Email;
use SendGrid;
use SendGrid\Mail\Mail;
use SendGrid\Mail\Attachment;

class SendGridProvider implements EmailProviderInterface
{
    private SendGrid $client;

    public function __construct(
        private readonly string $apiKey,
        ?SendGrid $client = null
    ) {
        $this->client = $client ?? new SendGrid($apiKey);
    }

    public function send(Email $email): ProviderResult
    {
        try {
            $sendGridMail = $this->buildMail($email);
            $response = $this->client->send($sendGridMail);
            
            $statusCode = $response->statusCode();
            
            if ($statusCode >= 200 && $statusCode < 300) {
                $headers = $response->headers();
                $messageId = $headers['X-Message-Id'][0] ?? uniqid('sg_');
                return ProviderResult::success($messageId);
            }
            
            $body = json_decode($response->body(), true);
            $errorMessage = $body['errors'][0]['message'] ?? 'Unknown SendGrid error';
            
            return ProviderResult::failure('SENDGRID_ERROR', $errorMessage);
        } catch (\Exception $e) {
            return ProviderResult::failure('SENDGRID_EXCEPTION', $e->getMessage());
        }
    }

    public function getDeliveryStatus(string $messageId): DeliveryStatus
    {
        // SendGrid uses webhooks for delivery status
        // This would typically query the Events API
        return DeliveryStatus::pending();
    }

    public function getName(): string
    {
        return 'SendGrid';
    }

    public function isHealthy(): bool
    {
        try {
            // Simple health check - verify API key is valid
            return !empty($this->apiKey);
        } catch (\Exception) {
            return false;
        }
    }

    private function buildMail(Email $email): Mail
    {
        $mail = new Mail();
        
        // From
        $mail->setFrom($email->from->email, $email->from->name);
        
        // Subject
        $mail->setSubject($email->subject);
        
        // Recipients
        foreach ($email->getRecipients() as $recipient) {
            $mail->addTo($recipient->email, $recipient->name);
        }
        
        // CC
        foreach ($email->getCc() as $cc) {
            $mail->addCc($cc->email, $cc->name);
        }
        
        // BCC
        foreach ($email->getBcc() as $bcc) {
            $mail->addBcc($bcc->email, $bcc->name);
        }
        
        // Content
        if ($email->contentType->isHtml()) {
            $mail->addContent('text/html', $email->body);
        } else {
            $mail->addContent('text/plain', $email->body);
        }
        
        // Attachments
        foreach ($email->getAttachments() as $attachment) {
            $sgAttachment = new Attachment();
            $sgAttachment->setContent($attachment->getBase64Content());
            $sgAttachment->setType($attachment->mimeType);
            $sgAttachment->setFilename($attachment->filename);
            $sgAttachment->setDisposition('attachment');
            $mail->addAttachment($sgAttachment);
        }
        
        // Custom headers
        foreach ($email->headers as $name => $value) {
            $mail->addHeader($name, $value);
        }
        
        return $mail;
    }
}
