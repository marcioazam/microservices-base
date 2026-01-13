<?php

declare(strict_types=1);

namespace EmailService\Application\Service;

use EmailService\Application\DTO\DomainValidationResult;

interface DnsResolverInterface
{
    public function getMxRecords(string $domain): DomainValidationResult;
}
