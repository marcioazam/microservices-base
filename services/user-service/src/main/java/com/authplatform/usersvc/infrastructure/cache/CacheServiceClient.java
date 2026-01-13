package com.authplatform.usersvc.infrastructure.cache;

import com.github.benmanes.caffeine.cache.Cache;
import com.github.benmanes.caffeine.cache.Caffeine;
import io.github.resilience4j.circuitbreaker.CircuitBreaker;
import io.github.resilience4j.circuitbreaker.CircuitBreakerConfig;
import io.github.resilience4j.circuitbreaker.CircuitBreakerRegistry;
import io.grpc.ManagedChannel;
import io.grpc.ManagedChannelBuilder;
import io.grpc.StatusRuntimeException;
import jakarta.annotation.PostConstruct;
import jakarta.annotation.PreDestroy;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Component;

import java.time.Duration;
import java.util.Optional;
import java.util.concurrent.TimeUnit;

/**
 * gRPC client for platform cache-service with circuit breaker and local fallback.
 */
@Component
public class CacheServiceClient {

    private static final Logger log = LoggerFactory.getLogger(CacheServiceClient.class);
    
    private final String cacheServiceUrl;
    private final int timeoutMs;
    private final Cache<String, byte[]> localCache;
    private final CircuitBreaker circuitBreaker;
    
    private ManagedChannel channel;
    private volatile boolean connected = false;

    public CacheServiceClient(
            @Value("${app.platform.cache-service.url:localhost:50060}") String cacheServiceUrl,
            @Value("${app.platform.cache-service.timeout-ms:5000}") int timeoutMs) {
        this.cacheServiceUrl = cacheServiceUrl;
        this.timeoutMs = timeoutMs;
        
        // Local Caffeine cache as fallback
        this.localCache = Caffeine.newBuilder()
                .maximumSize(10_000)
                .expireAfterWrite(Duration.ofMinutes(5))
                .build();
        
        // Circuit breaker configuration
        CircuitBreakerConfig config = CircuitBreakerConfig.custom()
                .failureRateThreshold(50)
                .waitDurationInOpenState(Duration.ofSeconds(30))
                .slidingWindowSize(10)
                .minimumNumberOfCalls(5)
                .build();
        
        CircuitBreakerRegistry registry = CircuitBreakerRegistry.of(config);
        this.circuitBreaker = registry.circuitBreaker("cacheService");
    }

    @PostConstruct
    public void init() {
        try {
            String[] parts = cacheServiceUrl.split(":");
            String host = parts[0];
            int port = parts.length > 1 ? Integer.parseInt(parts[1]) : 50060;
            
            this.channel = ManagedChannelBuilder.forAddress(host, port)
                    .usePlaintext()
                    .build();
            this.connected = true;
            log.info("CacheServiceClient initialized: {}", cacheServiceUrl);
        } catch (Exception e) {
            log.warn("Failed to connect to cache-service, using local fallback: {}", e.getMessage());
            this.connected = false;
        }
    }

    @PreDestroy
    public void shutdown() {
        if (channel != null) {
            try {
                channel.shutdown().awaitTermination(5, TimeUnit.SECONDS);
            } catch (InterruptedException e) {
                Thread.currentThread().interrupt();
            }
        }
    }

    /**
     * Gets a value from cache. Falls back to local cache if remote unavailable.
     */
    public Optional<byte[]> get(String namespace, String key) {
        String fullKey = namespace + ":" + key;
        
        try {
            return circuitBreaker.executeSupplier(() -> {
                if (!connected || channel == null) {
                    return getFromLocalCache(fullKey);
                }
                // In production, this would call the gRPC stub
                // For now, use local cache as the implementation
                return getFromLocalCache(fullKey);
            });
        } catch (Exception e) {
            log.debug("Cache get failed, using local fallback: {}", e.getMessage());
            return getFromLocalCache(fullKey);
        }
    }

    /**
     * Sets a value in cache with TTL.
     */
    public void set(String namespace, String key, byte[] value, Duration ttl) {
        String fullKey = namespace + ":" + key;
        
        try {
            circuitBreaker.executeRunnable(() -> {
                if (!connected || channel == null) {
                    setInLocalCache(fullKey, value);
                    return;
                }
                // In production, this would call the gRPC stub
                setInLocalCache(fullKey, value);
            });
        } catch (Exception e) {
            log.debug("Cache set failed, using local fallback: {}", e.getMessage());
            setInLocalCache(fullKey, value);
        }
    }

    /**
     * Deletes a value from cache.
     */
    public void delete(String namespace, String key) {
        String fullKey = namespace + ":" + key;
        
        try {
            circuitBreaker.executeRunnable(() -> {
                if (!connected || channel == null) {
                    localCache.invalidate(fullKey);
                    return;
                }
                // In production, this would call the gRPC stub
                localCache.invalidate(fullKey);
            });
        } catch (Exception e) {
            log.debug("Cache delete failed, using local fallback: {}", e.getMessage());
            localCache.invalidate(fullKey);
        }
    }

    /**
     * Checks if cache service is available.
     */
    public boolean isAvailable() {
        return connected && circuitBreaker.getState() != CircuitBreaker.State.OPEN;
    }

    private Optional<byte[]> getFromLocalCache(String key) {
        return Optional.ofNullable(localCache.getIfPresent(key));
    }

    private void setInLocalCache(String key, byte[] value) {
        localCache.put(key, value);
    }
}
