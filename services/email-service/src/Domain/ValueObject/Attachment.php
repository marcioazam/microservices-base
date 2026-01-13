<?php

declare(strict_types=1);

namespace EmailService\Domain\ValueObject;

use EmailService\Domain\Exception\AttachmentTooLargeException;

final readonly class Attachment
{
    public const MAX_TOTAL_SIZE = 26214400; // 25MB

    public function __construct(
        public string $filename,
        public string $content,
        public string $mimeType,
        public int $size
    ) {
        if ($size > self::MAX_TOTAL_SIZE) {
            throw new AttachmentTooLargeException($size);
        }
    }

    public static function fromBase64(string $filename, string $base64Content, string $mimeType): self
    {
        $content = base64_decode($base64Content, true);
        if ($content === false) {
            throw new \InvalidArgumentException('Invalid base64 content');
        }
        
        return new self($filename, $content, $mimeType, strlen($content));
    }

    public static function fromPath(string $path, ?string $filename = null, ?string $mimeType = null): self
    {
        if (!file_exists($path)) {
            throw new \InvalidArgumentException("File not found: {$path}");
        }

        $content = file_get_contents($path);
        if ($content === false) {
            throw new \RuntimeException("Cannot read file: {$path}");
        }

        return new self(
            $filename ?? basename($path),
            $content,
            $mimeType ?? mime_content_type($path) ?: 'application/octet-stream',
            strlen($content)
        );
    }

    public function getBase64Content(): string
    {
        return base64_encode($this->content);
    }

    public static function validateTotalSize(array $attachments): void
    {
        $totalSize = array_reduce(
            $attachments,
            fn(int $carry, Attachment $attachment) => $carry + $attachment->size,
            0
        );

        if ($totalSize > self::MAX_TOTAL_SIZE) {
            throw new AttachmentTooLargeException($totalSize);
        }
    }
}
