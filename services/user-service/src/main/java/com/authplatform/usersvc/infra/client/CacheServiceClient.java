package com.authplatform.usersvc.infra.client;

import com.github.benmanes.caffeine.cache.Cache;
import com.github.benmanes.caffeine.cache.Caffeine;
import io.github.resilience4j.circuitbreaker.annotation.CircuitBreaker;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Component;
import java.time.Duration;
import java.util.Optional;

@Component
@Slf4j
public class CacheServiceClient {

    @Value("${app.cache.service.enabled:false}")
    private boolean cacheServiceEnabled;

    private final Cache<String, String> localCache = Caffeine.newBuilder()
            .maximumSize(10_000)
            .expireAfterWrite(Duration.ofMinutes(15))
            .build();

    @CircuitBreaker(name = "cacheService", fallbackMethod = "getFallback")
    public Optional<String> get(String key) {
        if (!cacheServiceEnabled) {
            return getLocal(key);
        }
        // TODO: gRPC call to cache-service
        log.debug("Calling cache-service for key: {}", key);
        return getLocal(key);
    }

    @CircuitBreaker(name = "cacheService", fallbackMethod = "setFallback")
    public void set(String key, String value, Duration ttl) {
        if (!cacheServiceEnabled) {
            setLocal(key, value);
            return;
        }
        // TODO: gRPC call to cache-service
        log.debug("Calling cache-service to set key: {}", key);
        setLocal(key, value);
    }

    @CircuitBreaker(name = "cacheService", fallbackMethod = "deleteFallback")
    public void delete(String key) {
        if (!cacheServiceEnabled) {
            deleteLocal(key);
            return;
        }
        // TODO: gRPC call to cache-service
        log.debug("Calling cache-service to delete key: {}", key);
        deleteLocal(key);
    }

    public Optional<String> getFallback(String key, Throwable t) {
        log.warn("Cache service unavailable, using local fallback: {}", t.getMessage());
        return getLocal(key);
    }

    public void setFallback(String key, String value, Duration ttl, Throwable t) {
        log.warn("Cache service unavailable, using local fallback: {}", t.getMessage());
        setLocal(key, value);
    }

    public void deleteFallback(String key, Throwable t) {
        log.warn("Cache service unavailable, using local fallback: {}", t.getMessage());
        deleteLocal(key);
    }

    private Optional<String> getLocal(String key) {
        return Optional.ofNullable(localCache.getIfPresent(key));
    }

    private void setLocal(String key, String value) {
        localCache.put(key, value);
    }

    private void deleteLocal(String key) {
        localCache.invalidate(key);
    }
}
