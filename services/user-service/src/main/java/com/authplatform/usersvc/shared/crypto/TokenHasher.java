package com.authplatform.usersvc.shared.crypto;

import org.springframework.stereotype.Component;

import java.nio.charset.StandardCharsets;
import java.security.MessageDigest;
import java.security.NoSuchAlgorithmException;
import java.security.SecureRandom;
import java.util.HexFormat;

/**
 * Token generation and hashing using SHA-256.
 * Single source of truth for verification token operations.
 */
@Component
public class TokenHasher {

    private static final int TOKEN_BYTES = 32;
    private final SecureRandom secureRandom;
    private final HexFormat hexFormat;

    public TokenHasher() {
        this.secureRandom = new SecureRandom();
        this.hexFormat = HexFormat.of();
    }

    /**
     * Generates a secure random 32-byte token as hex string.
     */
    public String generateToken() {
        byte[] bytes = new byte[TOKEN_BYTES];
        secureRandom.nextBytes(bytes);
        return hexFormat.formatHex(bytes);
    }

    /**
     * Hashes a token using SHA-256, returning 64-char hex string.
     */
    public String hash(String token) {
        if (token == null || token.isEmpty()) {
            throw new IllegalArgumentException("Token cannot be null or empty");
        }
        try {
            MessageDigest digest = MessageDigest.getInstance("SHA-256");
            byte[] hashBytes = digest.digest(token.getBytes(StandardCharsets.UTF_8));
            return hexFormat.formatHex(hashBytes);
        } catch (NoSuchAlgorithmException e) {
            throw new RuntimeException("SHA-256 not available", e);
        }
    }

    /**
     * Verifies a token against a hash using constant-time comparison.
     */
    public boolean verify(String token, String expectedHash) {
        if (token == null || expectedHash == null) {
            return false;
        }
        String actualHash = hash(token);
        return MessageDigest.isEqual(
                actualHash.getBytes(StandardCharsets.UTF_8),
                expectedHash.getBytes(StandardCharsets.UTF_8)
        );
    }
}
