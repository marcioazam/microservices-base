package com.authplatform.usersvc.common.util;

import org.springframework.stereotype.Component;
import java.nio.charset.StandardCharsets;
import java.security.MessageDigest;
import java.security.NoSuchAlgorithmException;
import java.security.SecureRandom;
import java.util.Base64;
import java.util.HexFormat;

@Component
public class TokenHasher {

    private static final int TOKEN_BYTES = 32;
    private static final String HASH_ALGORITHM = "SHA-256";
    private final SecureRandom secureRandom;

    public TokenHasher() {
        this.secureRandom = new SecureRandom();
    }

    public String generateToken() {
        byte[] tokenBytes = new byte[TOKEN_BYTES];
        secureRandom.nextBytes(tokenBytes);
        return Base64.getUrlEncoder().withoutPadding().encodeToString(tokenBytes);
    }

    public String hash(String token) {
        if (token == null) {
            throw new IllegalArgumentException("Token cannot be null");
        }
        try {
            MessageDigest digest = MessageDigest.getInstance(HASH_ALGORITHM);
            byte[] hashBytes = digest.digest(token.getBytes(StandardCharsets.UTF_8));
            return HexFormat.of().formatHex(hashBytes);
        } catch (NoSuchAlgorithmException e) {
            throw new RuntimeException("SHA-256 algorithm not available", e);
        }
    }

    public boolean verify(String token, String expectedHash) {
        if (token == null || expectedHash == null) {
            return false;
        }
        String computedHash = hash(token);
        return MessageDigest.isEqual(
                computedHash.getBytes(StandardCharsets.UTF_8),
                expectedHash.getBytes(StandardCharsets.UTF_8)
        );
    }

    public int getTokenLength() {
        return TOKEN_BYTES;
    }

    public int getHashLength() {
        return 64;
    }
}
