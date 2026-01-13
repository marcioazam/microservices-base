<?php

declare(strict_types=1);

namespace EmailService\Infrastructure\Provider;

use EmailService\Domain\Entity\Email;
use Mailgun\Mailgun;

class MailgunProvider implements EmailProviderInterface
{
    private Mailgun $client;

    public function __construct(
        private readonly string $apiKey,
        private readonly string $domain,
        ?Mailgun $client = null
    ) {
        $this->client = $client ?? Mailgun::create($apiKey);
    }

    public function send(Email $email): ProviderResult
    {
        try {
            $params = $this->buildParams($email);
            
            $response = $this->client->messages()->send($this->domain, $params);
            
            if ($response->getId()) {
                return ProviderResult::success($response->getId());
            }
            
            return ProviderResult::failure('MAILGUN_ERROR', $response->getMessage() ?? 'Unknown error');
        } catch (\Exception $e) {
            return ProviderResult::failure('MAILGUN_EXCEPTION', $e->getMessage());
        }
    }

    public function getDeliveryStatus(string $messageId): DeliveryStatus
    {
        try {
            $events = $this->client->events()->get($this->domain, [
                'message-id' => $messageId,
                'limit' => 1,
            ]);
            
            foreach ($events->getItems() as $event) {
                return match ($event->getEvent()) {
                    'delivered' => DeliveryStatus::delivered(),
                    'failed', 'rejected' => DeliveryStatus::failed($event->getReason() ?? 'Unknown'),
                    'bounced' => DeliveryStatus::bounced($event->getReason() ?? 'Bounced'),
                    default => DeliveryStatus::pending(),
                };
            }
            
            return DeliveryStatus::pending();
        } catch (\Exception) {
            return DeliveryStatus::pending();
        }
    }

    public function getName(): string
    {
        return 'Mailgun';
    }

    public function isHealthy(): bool
    {
        try {
            $this->client->domains()->show($this->domain);
            return true;
        } catch (\Exception) {
            return false;
        }
    }

    private function buildParams(Email $email): array
    {
        $params = [
            'from' => $email->from->getFormatted(),
            'to' => array_map(fn($r) => $r->getFormatted(), $email->getRecipients()),
            'subject' => $email->subject,
        ];
        
        // CC
        if (!empty($email->getCc())) {
            $params['cc'] = array_map(fn($r) => $r->getFormatted(), $email->getCc());
        }
        
        // BCC
        if (!empty($email->getBcc())) {
            $params['bcc'] = array_map(fn($r) => $r->getFormatted(), $email->getBcc());
        }
        
        // Content
        if ($email->contentType->isHtml()) {
            $params['html'] = $email->body;
        } else {
            $params['text'] = $email->body;
        }
        
        // Attachments
        if ($email->hasAttachments()) {
            $params['attachment'] = [];
            foreach ($email->getAttachments() as $attachment) {
                $params['attachment'][] = [
                    'fileContent' => $attachment->content,
                    'filename' => $attachment->filename,
                ];
            }
        }
        
        // Custom headers
        foreach ($email->headers as $name => $value) {
            $params["h:{$name}"] = $value;
        }
        
        return $params;
    }
}
