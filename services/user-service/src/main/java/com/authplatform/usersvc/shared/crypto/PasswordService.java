package com.authplatform.usersvc.shared.crypto;

import de.mkammerer.argon2.Argon2;
import de.mkammerer.argon2.Argon2Factory;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;

/**
 * Password hashing service using Argon2id with OWASP-recommended parameters.
 * Single source of truth for password hashing and verification.
 */
@Service
public class PasswordService {

    private final Argon2 argon2;
    private final int memoryKb;
    private final int iterations;
    private final int parallelism;

    public PasswordService(
            @Value("${app.argon2.memory-kb:19456}") int memoryKb,
            @Value("${app.argon2.iterations:2}") int iterations,
            @Value("${app.argon2.parallelism:1}") int parallelism) {
        this.argon2 = Argon2Factory.create(Argon2Factory.Argon2Types.ARGON2id);
        this.memoryKb = memoryKb;
        this.iterations = iterations;
        this.parallelism = parallelism;
    }

    /**
     * Hashes a password using Argon2id.
     * Returns a string starting with $argon2id$ containing algorithm parameters.
     */
    public String hash(String password) {
        if (password == null || password.isEmpty()) {
            throw new IllegalArgumentException("Password cannot be null or empty");
        }
        return argon2.hash(iterations, memoryKb, parallelism, password.toCharArray());
    }

    /**
     * Verifies a password against a hash.
     * Uses constant-time comparison to prevent timing attacks.
     */
    public boolean verify(String password, String hash) {
        if (password == null || hash == null) {
            return false;
        }
        return argon2.verify(hash, password.toCharArray());
    }
}
