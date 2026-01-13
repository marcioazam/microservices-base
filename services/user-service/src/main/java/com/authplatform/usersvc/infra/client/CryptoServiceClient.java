package com.authplatform.usersvc.infra.client;

import io.github.resilience4j.circuitbreaker.annotation.CircuitBreaker;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Component;
import de.mkammerer.argon2.Argon2;
import de.mkammerer.argon2.Argon2Factory;

@Component
@Slf4j
public class CryptoServiceClient {

    @Value("${app.crypto.service.enabled:false}")
    private boolean cryptoServiceEnabled;

    private final Argon2 argon2 = Argon2Factory.create(Argon2Factory.Argon2Types.ARGON2id);

    @CircuitBreaker(name = "cryptoService", fallbackMethod = "hashPasswordFallback")
    public String hashPassword(String password) {
        if (!cryptoServiceEnabled) {
            return hashPasswordLocal(password);
        }
        // TODO: gRPC call to crypto-service
        log.debug("Calling crypto-service for password hashing");
        return hashPasswordLocal(password);
    }

    @CircuitBreaker(name = "cryptoService", fallbackMethod = "verifyPasswordFallback")
    public boolean verifyPassword(String password, String hash) {
        if (!cryptoServiceEnabled) {
            return verifyPasswordLocal(password, hash);
        }
        // TODO: gRPC call to crypto-service
        log.debug("Calling crypto-service for password verification");
        return verifyPasswordLocal(password, hash);
    }

    public String hashPasswordFallback(String password, Throwable t) {
        log.warn("Crypto service unavailable, using local fallback: {}", t.getMessage());
        return hashPasswordLocal(password);
    }

    public boolean verifyPasswordFallback(String password, String hash, Throwable t) {
        log.warn("Crypto service unavailable, using local fallback: {}", t.getMessage());
        return verifyPasswordLocal(password, hash);
    }

    private String hashPasswordLocal(String password) {
        return argon2.hash(3, 65536, 4, password.toCharArray());
    }

    private boolean verifyPasswordLocal(String password, String hash) {
        return argon2.verify(hash, password.toCharArray());
    }
}
