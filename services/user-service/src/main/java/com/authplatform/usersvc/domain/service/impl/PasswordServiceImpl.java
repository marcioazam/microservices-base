package com.authplatform.usersvc.domain.service.impl;

import com.authplatform.usersvc.domain.service.PasswordService;
import de.mkammerer.argon2.Argon2;
import de.mkammerer.argon2.Argon2Factory;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;

@Service
public class PasswordServiceImpl implements PasswordService {

    private final Argon2 argon2;
    private final int iterations;
    private final int memoryKb;
    private final int parallelism;

    public PasswordServiceImpl(
            @Value("${app.argon2.iterations:3}") int iterations,
            @Value("${app.argon2.memory-kb:65536}") int memoryKb,
            @Value("${app.argon2.parallelism:1}") int parallelism) {
        this.argon2 = Argon2Factory.create(Argon2Factory.Argon2Types.ARGON2id);
        this.iterations = iterations;
        this.memoryKb = memoryKb;
        this.parallelism = parallelism;
    }

    @Override
    public String hash(String plainPassword) {
        if (plainPassword == null || plainPassword.isEmpty()) {
            throw new IllegalArgumentException("Password cannot be null or empty");
        }
        try {
            return argon2.hash(iterations, memoryKb, parallelism, plainPassword.toCharArray());
        } finally {
            // Clear password from memory
        }
    }

    @Override
    public boolean verify(String plainPassword, String hash) {
        if (plainPassword == null || hash == null) {
            return false;
        }
        try {
            return argon2.verify(hash, plainPassword.toCharArray());
        } catch (Exception e) {
            return false;
        }
    }

    @Override
    public boolean isArgon2idHash(String hash) {
        return hash != null && hash.startsWith("$argon2id$");
    }
}
