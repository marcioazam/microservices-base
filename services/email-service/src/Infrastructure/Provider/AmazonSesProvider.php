<?php

declare(strict_types=1);

namespace EmailService\Infrastructure\Provider;

use Aws\Ses\SesClient;
use EmailService\Domain\Entity\Email;

class AmazonSesProvider implements EmailProviderInterface
{
    private SesClient $client;

    public function __construct(
        private readonly string $region,
        private readonly string $accessKeyId,
        private readonly string $secretAccessKey,
        ?SesClient $client = null
    ) {
        $this->client = $client ?? new SesClient([
            'version' => 'latest',
            'region' => $region,
            'credentials' => [
                'key' => $accessKeyId,
                'secret' => $secretAccessKey,
            ],
        ]);
    }

    public function send(Email $email): ProviderResult
    {
        try {
            if ($email->hasAttachments()) {
                return $this->sendRawEmail($email);
            }
            
            return $this->sendSimpleEmail($email);
        } catch (\Exception $e) {
            return ProviderResult::failure('SES_EXCEPTION', $e->getMessage());
        }
    }

    private function sendSimpleEmail(Email $email): ProviderResult
    {
        $params = [
            'Source' => $email->from->getFormatted(),
            'Destination' => [
                'ToAddresses' => array_map(fn($r) => $r->email, $email->getRecipients()),
            ],
            'Message' => [
                'Subject' => [
                    'Data' => $email->subject,
                    'Charset' => 'UTF-8',
                ],
                'Body' => [],
            ],
        ];
        
        // CC
        if (!empty($email->getCc())) {
            $params['Destination']['CcAddresses'] = array_map(fn($r) => $r->email, $email->getCc());
        }
        
        // BCC
        if (!empty($email->getBcc())) {
            $params['Destination']['BccAddresses'] = array_map(fn($r) => $r->email, $email->getBcc());
        }
        
        // Content
        if ($email->contentType->isHtml()) {
            $params['Message']['Body']['Html'] = [
                'Data' => $email->body,
                'Charset' => 'UTF-8',
            ];
        } else {
            $params['Message']['Body']['Text'] = [
                'Data' => $email->body,
                'Charset' => 'UTF-8',
            ];
        }
        
        $result = $this->client->sendEmail($params);
        $messageId = $result->get('MessageId');
        
        return ProviderResult::success($messageId);
    }

    private function sendRawEmail(Email $email): ProviderResult
    {
        $rawMessage = $this->buildRawMessage($email);
        
        $result = $this->client->sendRawEmail([
            'RawMessage' => [
                'Data' => $rawMessage,
            ],
        ]);
        
        $messageId = $result->get('MessageId');
        return ProviderResult::success($messageId);
    }

    private function buildRawMessage(Email $email): string
    {
        $boundary = uniqid('boundary_');
        $headers = [];
        
        $headers[] = "From: {$email->from->getFormatted()}";
        $headers[] = "To: " . implode(', ', array_map(fn($r) => $r->getFormatted(), $email->getRecipients()));
        
        if (!empty($email->getCc())) {
            $headers[] = "Cc: " . implode(', ', array_map(fn($r) => $r->getFormatted(), $email->getCc()));
        }
        
        $headers[] = "Subject: {$email->subject}";
        $headers[] = "MIME-Version: 1.0";
        $headers[] = "Content-Type: multipart/mixed; boundary=\"{$boundary}\"";
        
        // Custom headers
        foreach ($email->headers as $name => $value) {
            $headers[] = "{$name}: {$value}";
        }
        
        $message = implode("\r\n", $headers) . "\r\n\r\n";
        
        // Body
        $message .= "--{$boundary}\r\n";
        $message .= "Content-Type: {$email->contentType->value}; charset=UTF-8\r\n";
        $message .= "Content-Transfer-Encoding: 7bit\r\n\r\n";
        $message .= $email->body . "\r\n\r\n";
        
        // Attachments
        foreach ($email->getAttachments() as $attachment) {
            $message .= "--{$boundary}\r\n";
            $message .= "Content-Type: {$attachment->mimeType}; name=\"{$attachment->filename}\"\r\n";
            $message .= "Content-Disposition: attachment; filename=\"{$attachment->filename}\"\r\n";
            $message .= "Content-Transfer-Encoding: base64\r\n\r\n";
            $message .= chunk_split($attachment->getBase64Content(), 76, "\r\n");
        }
        
        $message .= "--{$boundary}--";
        
        return $message;
    }

    public function getDeliveryStatus(string $messageId): DeliveryStatus
    {
        // SES uses SNS notifications for delivery status
        return DeliveryStatus::pending();
    }

    public function getName(): string
    {
        return 'AmazonSES';
    }

    public function isHealthy(): bool
    {
        try {
            $this->client->getSendQuota();
            return true;
        } catch (\Exception) {
            return false;
        }
    }
}
