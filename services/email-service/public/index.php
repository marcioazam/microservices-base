<?php

declare(strict_types=1);

require_once __DIR__ . '/../vendor/autoload.php';

use EmailService\Api\Controller\EmailController;
use EmailService\Api\Controller\HealthController;
use EmailService\Infrastructure\Observability\StructuredLogger;

// Bootstrap application
$logger = new StructuredLogger();

// Simple router
$requestUri = $_SERVER['REQUEST_URI'] ?? '/';
$requestMethod = $_SERVER['REQUEST_METHOD'] ?? 'GET';

// Parse path
$path = parse_url($requestUri, PHP_URL_PATH);

// Route handling
try {
    $response = match (true) {
        $path === '/health/live' => handleLiveness(),
        $path === '/health/ready' => handleReadiness(),
        $path === '/metrics' => handleMetrics(),
        str_starts_with($path, '/api/v1/') => handleApi($path, $requestMethod),
        default => ['statusCode' => 404, 'body' => ['error' => 'Not Found']],
    };

    http_response_code($response['statusCode'] ?? 200);
    header('Content-Type: application/json');
    echo json_encode($response['body'] ?? $response);
} catch (\Throwable $e) {
    $logger->error('Unhandled exception', [
        'exception' => $e->getMessage(),
        'trace' => $e->getTraceAsString(),
    ]);
    
    http_response_code(500);
    header('Content-Type: application/json');
    echo json_encode(['error' => 'Internal Server Error']);
}

function handleLiveness(): array
{
    return ['status' => 'ok', 'timestamp' => gmdate('Y-m-d\TH:i:s\Z')];
}

function handleReadiness(): array
{
    // In production, wire up actual health checks
    return ['status' => 'ok', 'checks' => []];
}

function handleMetrics(): array
{
    return ['statusCode' => 200, 'body' => ['metrics' => 'prometheus_format']];
}

function handleApi(string $path, string $method): array
{
    // API routing - in production use a proper router
    return match (true) {
        $path === '/api/v1/emails/send' && $method === 'POST' => handleSendEmail(),
        $path === '/api/v1/emails/verify' && $method === 'POST' => handleVerifyEmail(),
        $path === '/api/v1/emails/resend' && $method === 'POST' => handleResendEmail(),
        str_starts_with($path, '/api/v1/emails/') && str_ends_with($path, '/status') => handleGetStatus($path),
        $path === '/api/v1/emails/audit' && $method === 'GET' => handleAuditQuery(),
        default => ['statusCode' => 404, 'body' => ['error' => 'Endpoint not found']],
    };
}

function handleSendEmail(): array
{
    return ['statusCode' => 501, 'body' => ['error' => 'Wire up EmailController']];
}

function handleVerifyEmail(): array
{
    return ['statusCode' => 501, 'body' => ['error' => 'Wire up EmailController']];
}

function handleResendEmail(): array
{
    return ['statusCode' => 501, 'body' => ['error' => 'Wire up EmailController']];
}

function handleGetStatus(string $path): array
{
    return ['statusCode' => 501, 'body' => ['error' => 'Wire up EmailController']];
}

function handleAuditQuery(): array
{
    return ['statusCode' => 501, 'body' => ['error' => 'Wire up EmailController']];
}
