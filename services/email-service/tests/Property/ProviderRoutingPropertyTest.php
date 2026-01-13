<?php

declare(strict_types=1);

namespace EmailService\Tests\Property;

use EmailService\Domain\Entity\Email;
use EmailService\Domain\Enum\EmailType;
use EmailService\Domain\ValueObject\Recipient;
use EmailService\Infrastructure\Provider\DeliveryStatus;
use EmailService\Infrastructure\Provider\EmailProviderInterface;
use EmailService\Infrastructure\Provider\ProviderResult;
use EmailService\Infrastructure\Provider\ProviderRouter;
use Eris\Generator;
use Eris\TestTrait;
use PHPUnit\Framework\TestCase;

/**
 * Property 25: Provider Routing by Email Type
 * For any email with specified type, the Email_Service SHALL route to the
 * provider configured for that type.
 * 
 * Property 5: Provider Failover
 * For any email send attempt where the primary provider fails, the Email_Service
 * SHALL attempt delivery via the configured secondary provider.
 * 
 * Property 26: Per-Provider Metrics
 * For any email sent through a provider, the metrics system SHALL increment
 * counters for that specific provider.
 * 
 * Validates: Requirements 1.6, 8.2, 8.3, 8.5
 */
class ProviderRoutingPropertyTest extends TestCase
{
    use TestTrait;

    /**
     * @test
     * Property 25: Emails are routed to type-specific providers
     */
    public function emailsAreRoutedToTypeSpecificProviders(): void
    {
        $transactionalProvider = $this->createMockProvider('Transactional', true);
        $marketingProvider = $this->createMockProvider('Marketing', true);
        $verificationProvider = $this->createMockProvider('Verification', true);
        
        $router = new ProviderRouter();
        $router->registerProvider($transactionalProvider, true);
        $router->registerProvider($marketingProvider);
        $router->registerProvider($verificationProvider);
        
        $router->setTypeMapping(EmailType::TRANSACTIONAL, 'Transactional');
        $router->setTypeMapping(EmailType::MARKETING, 'Marketing');
        $router->setTypeMapping(EmailType::VERIFICATION, 'Verification');
        
        // Test transactional
        $email = $this->createEmail(EmailType::TRANSACTIONAL);
        $provider = $router->getProviderForEmail($email);
        $this->assertEquals('Transactional', $provider->getName());
        
        // Test marketing
        $email = $this->createEmail(EmailType::MARKETING);
        $provider = $router->getProviderForEmail($email);
        $this->assertEquals('Marketing', $provider->getName());
        
        // Test verification
        $email = $this->createEmail(EmailType::VERIFICATION);
        $provider = $router->getProviderForEmail($email);
        $this->assertEquals('Verification', $provider->getName());
    }

    /**
     * @test
     * Property 5: Failover to secondary provider on primary failure
     */
    public function failoverToSecondaryProviderOnPrimaryFailure(): void
    {
        $primaryProvider = $this->createMockProvider('Primary', true, false); // Healthy but fails to send
        $fallbackProvider = $this->createMockProvider('Fallback', true, true); // Healthy and succeeds
        
        $router = new ProviderRouter();
        $router->registerProvider($primaryProvider, true);
        $router->registerProvider($fallbackProvider);
        $router->setFallbackProvider('Fallback');
        
        $email = $this->createEmail(EmailType::TRANSACTIONAL);
        $result = $router->send($email);
        
        // Should succeed via fallback
        $this->assertTrue($result->success);
    }

    /**
     * @test
     * Property 5: Primary provider is used when healthy
     */
    public function primaryProviderIsUsedWhenHealthy(): void
    {
        $primaryProvider = $this->createMockProvider('Primary', true, true);
        $fallbackProvider = $this->createMockProvider('Fallback', true, true);
        
        $router = new ProviderRouter();
        $router->registerProvider($primaryProvider, true);
        $router->registerProvider($fallbackProvider);
        $router->setFallbackProvider('Fallback');
        
        $email = $this->createEmail(EmailType::TRANSACTIONAL);
        $result = $router->send($email);
        
        $this->assertTrue($result->success);
        
        // Primary should have been used
        $metrics = $router->getMetricsForProvider('Primary');
        $this->assertEquals(1, $metrics['sent']);
    }

    /**
     * @test
     * Property 26: Metrics are tracked per provider
     */
    public function metricsAreTrackedPerProvider(): void
    {
        $provider1 = $this->createMockProvider('Provider1', true, true);
        $provider2 = $this->createMockProvider('Provider2', true, true);
        
        $router = new ProviderRouter();
        $router->registerProvider($provider1, true);
        $router->registerProvider($provider2);
        
        $router->setTypeMapping(EmailType::TRANSACTIONAL, 'Provider1');
        $router->setTypeMapping(EmailType::MARKETING, 'Provider2');
        
        // Send via Provider1
        $email1 = $this->createEmail(EmailType::TRANSACTIONAL);
        $router->send($email1);
        $router->send($email1);
        
        // Send via Provider2
        $email2 = $this->createEmail(EmailType::MARKETING);
        $router->send($email2);
        
        $metrics1 = $router->getMetricsForProvider('Provider1');
        $metrics2 = $router->getMetricsForProvider('Provider2');
        
        $this->assertEquals(2, $metrics1['sent']);
        $this->assertEquals(1, $metrics2['sent']);
    }

    /**
     * @test
     * Property 26: Failed sends are tracked separately
     */
    public function failedSendsAreTrackedSeparately(): void
    {
        $failingProvider = $this->createMockProvider('Failing', true, false);
        
        $router = new ProviderRouter();
        $router->registerProvider($failingProvider, true);
        
        $email = $this->createEmail(EmailType::TRANSACTIONAL);
        $router->send($email);
        $router->send($email);
        
        $metrics = $router->getMetricsForProvider('Failing');
        
        $this->assertEquals(0, $metrics['sent']);
        $this->assertEquals(2, $metrics['failed']);
        $this->assertEquals(2, $metrics['total']);
    }

    /**
     * @test
     * Property: Unhealthy type-specific provider falls back to primary
     */
    public function unhealthyTypeSpecificProviderFallsBackToPrimary(): void
    {
        $primaryProvider = $this->createMockProvider('Primary', true, true);
        $unhealthyProvider = $this->createMockProvider('Unhealthy', false, true);
        
        $router = new ProviderRouter();
        $router->registerProvider($primaryProvider, true);
        $router->registerProvider($unhealthyProvider);
        
        $router->setTypeMapping(EmailType::MARKETING, 'Unhealthy');
        
        $email = $this->createEmail(EmailType::MARKETING);
        $provider = $router->getProviderForEmail($email);
        
        // Should fall back to primary since type-specific is unhealthy
        $this->assertEquals('Primary', $provider->getName());
    }

    private function createMockProvider(string $name, bool $healthy, bool $sendSuccess = true): EmailProviderInterface
    {
        $provider = $this->createMock(EmailProviderInterface::class);
        $provider->method('getName')->willReturn($name);
        $provider->method('isHealthy')->willReturn($healthy);
        $provider->method('getDeliveryStatus')->willReturn(DeliveryStatus::pending());
        
        if ($sendSuccess) {
            $provider->method('send')->willReturn(ProviderResult::success('msg-' . uniqid()));
        } else {
            $provider->method('send')->willReturn(ProviderResult::failure('ERROR', 'Send failed'));
        }
        
        return $provider;
    }

    private function createEmail(EmailType $type): Email
    {
        return Email::create(
            from: new Recipient('sender@example.com'),
            recipients: [new Recipient('recipient@example.com')],
            subject: 'Test',
            body: 'Body',
            type: $type
        );
    }
}
