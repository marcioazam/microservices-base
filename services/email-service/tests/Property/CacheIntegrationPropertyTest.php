<?php

declare(strict_types=1);

namespace EmailService\Tests\Property;

use EmailService\Infrastructure\Platform\BatchGetResult;
use EmailService\Infrastructure\Platform\CacheSource;
use EmailService\Infrastructure\Platform\CacheValue;
use EmailService\Infrastructure\Platform\InMemoryCacheClient;
use Eris\Generator;
use Eris\TestTrait;
use PHPUnit\Framework\TestCase;

/**
 * Feature: email-service-modernization-2025
 * Property 2: Cache Integration Round-Trip
 * 
 * For any cache key-value pair with valid namespace, when stored via CacheClient
 * and subsequently retrieved, the returned value SHALL equal the original value.
 * 
 * Validates: Requirements 1.2
 */
class CacheIntegrationPropertyTest extends TestCase
{
    use TestTrait;

    private InMemoryCacheClient $cacheClient;

    protected function setUp(): void
    {
        $this->cacheClient = new InMemoryCacheClient();
    }

    /**
     * @test
     * Property 2: Cache round-trip preserves string values
     */
    public function cacheRoundTripPreservesStringValues(): void
    {
        $this->forAll(
            Generator\suchThat(
                fn($s) => strlen($s) >= 1 && strlen($s) <= 100,
                Generator\string()
            ),
            Generator\suchThat(
                fn($s) => strlen($s) >= 1 && strlen($s) <= 500,
                Generator\string()
            )
        )
        ->withMaxSize(100)
        ->then(function (string $key, string $value): void {
            $this->cacheClient->clear();
            
            // Set value
            $setResult = $this->cacheClient->set($key, $value);
            $this->assertTrue($setResult);
            
            // Get value
            $cached = $this->cacheClient->get($key);
            
            $this->assertNotNull($cached);
            $this->assertEquals($value, $cached->value);
            $this->assertEquals(CacheSource::MEMORY, $cached->source);
        });
    }

    /**
     * @test
     * Property 2: Cache round-trip preserves integer values
     */
    public function cacheRoundTripPreservesIntegerValues(): void
    {
        $this->forAll(
            Generator\suchThat(
                fn($s) => strlen($s) >= 1 && strlen($s) <= 50 && preg_match('/^[a-z0-9]+$/', $s),
                Generator\string()
            ),
            Generator\int()
        )
        ->withMaxSize(100)
        ->then(function (string $key, int $value): void {
            $this->cacheClient->clear();
            
            $this->cacheClient->set($key, $value);
            $cached = $this->cacheClient->get($key);
            
            $this->assertNotNull($cached);
            $this->assertEquals($value, $cached->value);
        });
    }

    /**
     * @test
     * Property 2: Cache round-trip preserves array values
     */
    public function cacheRoundTripPreservesArrayValues(): void
    {
        $this->forAll(
            Generator\suchThat(
                fn($s) => strlen($s) >= 1 && strlen($s) <= 50 && preg_match('/^[a-z0-9]+$/', $s),
                Generator\string()
            ),
            Generator\associative([
                'id' => Generator\int(),
                'name' => Generator\suchThat(
                    fn($s) => strlen($s) >= 1 && strlen($s) <= 50,
                    Generator\string()
                ),
                'active' => Generator\bool(),
            ])
        )
        ->withMaxSize(100)
        ->then(function (string $key, array $value): void {
            $this->cacheClient->clear();
            
            $this->cacheClient->set($key, $value);
            $cached = $this->cacheClient->get($key);
            
            $this->assertNotNull($cached);
            $this->assertEquals($value, $cached->value);
        });
    }

    /**
     * @test
     * Property 2: Cache respects namespace isolation
     */
    public function cacheRespectsNamespaceIsolation(): void
    {
        $this->forAll(
            Generator\suchThat(
                fn($s) => strlen($s) >= 1 && strlen($s) <= 30 && preg_match('/^[a-z]+$/', $s),
                Generator\string()
            ),
            Generator\suchThat(
                fn($s) => strlen($s) >= 1 && strlen($s) <= 100,
                Generator\string()
            )
        )
        ->withMaxSize(100)
        ->then(function (string $key, string $value): void {
            $this->cacheClient->clear();
            
            // Set in namespace1
            $this->cacheClient->set($key, $value, 0, 'namespace1');
            
            // Should not exist in namespace2
            $cached2 = $this->cacheClient->get($key, 'namespace2');
            $this->assertNull($cached2);
            
            // Should exist in namespace1
            $cached1 = $this->cacheClient->get($key, 'namespace1');
            $this->assertNotNull($cached1);
            $this->assertEquals($value, $cached1->value);
        });
    }

    /**
     * @test
     * Property 2: Batch get returns all stored values
     */
    public function batchGetReturnsAllStoredValues(): void
    {
        $this->cacheClient->clear();
        
        $entries = [
            'key1' => 'value1',
            'key2' => 'value2',
            'key3' => 'value3',
        ];
        
        // Store all entries
        foreach ($entries as $key => $value) {
            $this->cacheClient->set($key, $value);
        }
        
        // Batch get
        $result = $this->cacheClient->batchGet(array_keys($entries));
        
        $this->assertInstanceOf(BatchGetResult::class, $result);
        $this->assertEmpty($result->missingKeys);
        
        foreach ($entries as $key => $value) {
            $this->assertTrue($result->hasKey($key));
            $this->assertEquals($value, $result->get($key));
        }
    }

    /**
     * @test
     * Property 2: Batch set stores all values
     */
    public function batchSetStoresAllValues(): void
    {
        $this->forAll(
            Generator\choose(1, 10)
        )
        ->withMaxSize(100)
        ->then(function (int $count): void {
            $this->cacheClient->clear();
            
            $entries = [];
            for ($i = 0; $i < $count; $i++) {
                $entries["key_{$i}"] = "value_{$i}";
            }
            
            $result = $this->cacheClient->batchSet($entries);
            $this->assertTrue($result);
            
            // Verify all stored
            foreach ($entries as $key => $value) {
                $cached = $this->cacheClient->get($key);
                $this->assertNotNull($cached);
                $this->assertEquals($value, $cached->value);
            }
        });
    }

    /**
     * @test
     * Property 2: Delete removes value
     */
    public function deleteRemovesValue(): void
    {
        $this->forAll(
            Generator\suchThat(
                fn($s) => strlen($s) >= 1 && strlen($s) <= 50 && preg_match('/^[a-z0-9]+$/', $s),
                Generator\string()
            ),
            Generator\suchThat(
                fn($s) => strlen($s) >= 1 && strlen($s) <= 100,
                Generator\string()
            )
        )
        ->withMaxSize(100)
        ->then(function (string $key, string $value): void {
            $this->cacheClient->clear();
            
            // Set and verify
            $this->cacheClient->set($key, $value);
            $this->assertNotNull($this->cacheClient->get($key));
            
            // Delete
            $deleted = $this->cacheClient->delete($key);
            $this->assertTrue($deleted);
            
            // Verify deleted
            $this->assertNull($this->cacheClient->get($key));
        });
    }

    /**
     * @test
     * Property 2: Health check returns true for in-memory client
     */
    public function healthCheckReturnsTrue(): void
    {
        $this->assertTrue($this->cacheClient->isHealthy());
    }

    /**
     * @test
     * Property 2: CacheValue factory methods work correctly
     */
    public function cacheValueFactoryMethodsWorkCorrectly(): void
    {
        $value = 'test-value';
        
        $fromRedis = CacheValue::fromRedis($value);
        $this->assertEquals($value, $fromRedis->value);
        $this->assertEquals(CacheSource::REDIS, $fromRedis->source);
        
        $fromLocal = CacheValue::fromLocal($value);
        $this->assertEquals($value, $fromLocal->value);
        $this->assertEquals(CacheSource::LOCAL, $fromLocal->source);
        
        $fromMemory = CacheValue::fromMemory($value);
        $this->assertEquals($value, $fromMemory->value);
        $this->assertEquals(CacheSource::MEMORY, $fromMemory->source);
    }
}
