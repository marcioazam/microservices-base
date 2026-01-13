<?php

declare(strict_types=1);

namespace EmailService\Infrastructure\Provider;

use EmailService\Domain\Entity\Email;
use EmailService\Domain\Enum\EmailType;

class ProviderRouter
{
    /** @var array<string, EmailProviderInterface> */
    private array $providers = [];
    
    /** @var array<string, string> */
    private array $typeMapping = [];
    
    /** @var array<string, int> */
    private array $metrics = [];

    private ?string $primaryProvider = null;
    private ?string $fallbackProvider = null;

    public function registerProvider(EmailProviderInterface $provider, bool $isPrimary = false): void
    {
        $name = $provider->getName();
        $this->providers[$name] = $provider;
        
        if ($isPrimary || $this->primaryProvider === null) {
            $this->primaryProvider = $name;
        }
        
        if ($this->fallbackProvider === null && !$isPrimary) {
            $this->fallbackProvider = $name;
        }
        
        $this->initMetrics($name);
    }

    public function setTypeMapping(EmailType $type, string $providerName): void
    {
        if (!isset($this->providers[$providerName])) {
            throw new \InvalidArgumentException("Provider not registered: {$providerName}");
        }
        
        $this->typeMapping[$type->value] = $providerName;
    }

    public function setFallbackProvider(string $providerName): void
    {
        if (!isset($this->providers[$providerName])) {
            throw new \InvalidArgumentException("Provider not registered: {$providerName}");
        }
        
        $this->fallbackProvider = $providerName;
    }

    public function send(Email $email): ProviderResult
    {
        $provider = $this->getProviderForEmail($email);
        
        $result = $provider->send($email);
        $this->recordMetric($provider->getName(), $result->success);
        
        if (!$result->success && $this->fallbackProvider !== null) {
            $fallback = $this->providers[$this->fallbackProvider];
            
            if ($fallback->getName() !== $provider->getName()) {
                $result = $fallback->send($email);
                $this->recordMetric($fallback->getName(), $result->success);
            }
        }
        
        return $result;
    }

    public function getProviderForEmail(Email $email): EmailProviderInterface
    {
        // Check type-specific mapping
        $typeKey = $email->type->value;
        if (isset($this->typeMapping[$typeKey])) {
            $providerName = $this->typeMapping[$typeKey];
            if ($this->providers[$providerName]->isHealthy()) {
                return $this->providers[$providerName];
            }
        }
        
        // Use primary provider
        if ($this->primaryProvider !== null && $this->providers[$this->primaryProvider]->isHealthy()) {
            return $this->providers[$this->primaryProvider];
        }
        
        // Find any healthy provider
        foreach ($this->providers as $provider) {
            if ($provider->isHealthy()) {
                return $provider;
            }
        }
        
        throw new NoHealthyProviderException('No healthy email provider available');
    }

    public function getMetrics(): array
    {
        return $this->metrics;
    }

    public function getMetricsForProvider(string $providerName): array
    {
        return [
            'sent' => $this->metrics["{$providerName}_sent"] ?? 0,
            'failed' => $this->metrics["{$providerName}_failed"] ?? 0,
            'total' => ($this->metrics["{$providerName}_sent"] ?? 0) + ($this->metrics["{$providerName}_failed"] ?? 0),
        ];
    }

    private function initMetrics(string $providerName): void
    {
        $this->metrics["{$providerName}_sent"] = 0;
        $this->metrics["{$providerName}_failed"] = 0;
    }

    private function recordMetric(string $providerName, bool $success): void
    {
        $key = $success ? "{$providerName}_sent" : "{$providerName}_failed";
        $this->metrics[$key] = ($this->metrics[$key] ?? 0) + 1;
    }

    public function getProviders(): array
    {
        return $this->providers;
    }
}
