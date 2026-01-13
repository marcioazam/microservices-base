<?php

declare(strict_types=1);

namespace EmailService\Domain\Exception;

use InvalidArgumentException;

class AttachmentTooLargeException extends InvalidArgumentException
{
    public const MAX_SIZE_BYTES = 26214400; // 25MB

    public function __construct(
        public readonly int $actualSize,
        public readonly int $maxSize = self::MAX_SIZE_BYTES
    ) {
        $actualMb = round($actualSize / 1048576, 2);
        $maxMb = round($maxSize / 1048576, 2);
        parent::__construct("Attachment size {$actualMb}MB exceeds maximum {$maxMb}MB");
    }
}
