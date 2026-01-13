<?php

declare(strict_types=1);

namespace EmailService\Infrastructure\Platform;

enum CacheSource: int
{
    case REDIS = 1;
    case LOCAL = 2;
    case MEMORY = 3;
}
