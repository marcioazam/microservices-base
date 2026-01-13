<?php

declare(strict_types=1);

/**
 * Service configuration for Email Microservice
 * 
 * Environment variables should be set in .env or docker-compose
 */

return [
    'database' => [
        'driver' => 'pgsql',
        'host' => getenv('DB_HOST') ?: 'postgres',
        'port' => (int) (getenv('DB_PORT') ?: 5432),
        'database' => getenv('DB_NAME') ?: 'email_db',
        'username' => getenv('DB_USER') ?: 'postgres',
        'password' => getenv('DB_PASSWORD') ?: 'postgres',
    ],

    'redis' => [
        'host' => getenv('REDIS_HOST') ?: 'redis',
        'port' => (int) (getenv('REDIS_PORT') ?: 6379),
        'prefix' => 'email:',
    ],

    'rabbitmq' => [
        'host' => getenv('RABBITMQ_HOST') ?: 'rabbitmq',
        'port' => (int) (getenv('RABBITMQ_PORT') ?: 5672),
        'user' => getenv('RABBITMQ_USER') ?: 'guest',
        'password' => getenv('RABBITMQ_PASSWORD') ?: 'guest',
        'vhost' => getenv('RABBITMQ_VHOST') ?: '/',
    ],

    'providers' => [
        'sendgrid' => [
            'api_key' => getenv('SENDGRID_API_KEY') ?: '',
            'enabled' => !empty(getenv('SENDGRID_API_KEY')),
        ],
        'mailgun' => [
            'api_key' => getenv('MAILGUN_API_KEY') ?: '',
            'domain' => getenv('MAILGUN_DOMAIN') ?: '',
            'enabled' => !empty(getenv('MAILGUN_API_KEY')),
        ],
        'ses' => [
            'region' => getenv('AWS_SES_REGION') ?: 'us-east-1',
            'access_key' => getenv('AWS_ACCESS_KEY_ID') ?: '',
            'secret_key' => getenv('AWS_SECRET_ACCESS_KEY') ?: '',
            'enabled' => !empty(getenv('AWS_ACCESS_KEY_ID')),
        ],
    ],

    'rate_limit' => [
        'per_sender' => (int) (getenv('RATE_LIMIT_PER_SENDER') ?: 100),
        'window_seconds' => (int) (getenv('RATE_LIMIT_WINDOW_SECONDS') ?: 3600),
        'global_limit' => (int) (getenv('RATE_LIMIT_GLOBAL') ?: 10000),
    ],

    'queue' => [
        'max_retries' => (int) (getenv('QUEUE_MAX_RETRIES') ?: 5),
        'retry_delays' => [1, 2, 4, 8, 16], // seconds
    ],

    'templates' => [
        'cache_dir' => getenv('TEMPLATE_CACHE_DIR') ?: '/tmp/twig_cache',
        'auto_reload' => getenv('APP_ENV') !== 'production',
    ],

    'logging' => [
        'level' => getenv('LOG_LEVEL') ?: 'info',
        'format' => 'json',
    ],
];
