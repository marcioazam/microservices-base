<?php

declare(strict_types=1);

namespace EmailService\Infrastructure\Provider;

use RuntimeException;

class NoHealthyProviderException extends RuntimeException
{
}
