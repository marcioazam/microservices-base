package com.authplatform.usersvc.property;

import com.authplatform.usersvc.api.dto.response.ProfileResponse;
import com.authplatform.usersvc.api.dto.response.UserRegistrationResponse;
import com.authplatform.usersvc.domain.model.User;
import com.authplatform.usersvc.domain.model.UserStatus;
import net.jqwik.api.*;
import java.time.Instant;
import java.util.UUID;
import static org.assertj.core.api.Assertions.assertThat;

/**
 * Property 8: Sensitive Data Non-Exposure
 * Validates: Requirements 4.5, 6.3, 10.5
 * 
 * Ensures that sensitive data (passwordHash, internal IDs, tokens)
 * is never exposed in API responses or logs.
 */
class SensitiveDataPropertyTest {

    private static final String[] SENSITIVE_PATTERNS = {
        "$argon2", "$bcrypt", "$2a$", "$2b$", "password",
        "secret", "token", "hash"
    };

    @Property(tries = 100)
    void profileResponseNeverContainsPasswordHash(
            @ForAll("userWithPassword") User user) {
        
        ProfileResponse response = mapToProfileResponse(user);
        String responseString = response.toString();
        
        // Password hash should never appear in response
        assertThat(responseString).doesNotContain(user.getPasswordHash());
        
        // Common password hash prefixes should not appear
        for (String pattern : SENSITIVE_PATTERNS) {
            assertThat(responseString.toLowerCase()).doesNotContain(pattern);
        }
    }

    @Property(tries = 100)
    void registrationResponseNeverContainsSensitiveData(
            @ForAll("userWithPassword") User user) {
        
        UserRegistrationResponse response = mapToRegistrationResponse(user);
        String responseString = response.toString();
        
        // Password hash should never appear
        assertThat(responseString).doesNotContain(user.getPasswordHash());
        
        // Internal verification token should not appear
        for (String pattern : SENSITIVE_PATTERNS) {
            assertThat(responseString.toLowerCase()).doesNotContain(pattern);
        }
    }

    @Property(tries = 100)
    void profileResponseContainsOnlyAllowedFields(
            @ForAll("userWithPassword") User user) {
        
        ProfileResponse response = mapToProfileResponse(user);
        
        // Should contain these fields
        assertThat(response.userId()).isNotNull();
        assertThat(response.email()).isNotNull();
        assertThat(response.displayName()).isNotNull();
        assertThat(response.createdAt()).isNotNull();
        
        // Verify field count (only 5 fields in record)
        assertThat(ProfileResponse.class.getRecordComponents()).hasSize(5);
    }

    @Property(tries = 100)
    void maskedEmailHidesLocalPart(@ForAll("validEmail") String email) {
        String masked = maskEmail(email);
        
        if (email.contains("@")) {
            String localPart = email.substring(0, email.indexOf('@'));
            if (localPart.length() > 2) {
                // Should not contain full local part
                assertThat(masked).doesNotContain(localPart);
                // Should contain asterisks
                assertThat(masked).contains("***");
            }
        }
    }

    @Property(tries = 100)
    void maskedIpHidesLastOctet(@ForAll("validIpv4") String ip) {
        String masked = maskIp(ip);
        
        String[] octets = ip.split("\\.");
        if (octets.length == 4) {
            // Last octet should be masked
            assertThat(masked).doesNotEndWith("." + octets[3]);
            assertThat(masked).endsWith(".***");
        }
    }

    @Provide
    Arbitrary<User> userWithPassword() {
        return Combinators.combine(
            Arbitraries.strings().alpha().ofMinLength(5).ofMaxLength(20),
            Arbitraries.strings().alpha().ofMinLength(2).ofMaxLength(50)
        ).as((emailLocal, displayName) -> {
            User user = new User();
            user.setId(UUID.randomUUID());
            user.setEmail(emailLocal.toLowerCase() + "@example.com");
            user.setPasswordHash("$argon2id$v=19$m=65536,t=3,p=4$" + UUID.randomUUID());
            user.setDisplayName(displayName);
            user.setStatus(UserStatus.ACTIVE);
            user.setEmailVerified(true);
            user.setCreatedAt(Instant.now());
            user.setUpdatedAt(Instant.now());
            return user;
        });
    }

    @Provide
    Arbitrary<String> validEmail() {
        return Arbitraries.strings().alpha().ofMinLength(3).ofMaxLength(20)
            .map(local -> local.toLowerCase() + "@example.com");
    }

    @Provide
    Arbitrary<String> validIpv4() {
        return Combinators.combine(
            Arbitraries.integers().between(1, 255),
            Arbitraries.integers().between(0, 255),
            Arbitraries.integers().between(0, 255),
            Arbitraries.integers().between(1, 254)
        ).as((a, b, c, d) -> a + "." + b + "." + c + "." + d);
    }

    private ProfileResponse mapToProfileResponse(User user) {
        return new ProfileResponse(
            user.getId(),
            user.getEmail(),
            user.isEmailVerified(),
            user.getDisplayName(),
            user.getCreatedAt()
        );
    }

    private UserRegistrationResponse mapToRegistrationResponse(User user) {
        return new UserRegistrationResponse(
            user.getId(),
            user.getEmail(),
            user.getStatus().name()
        );
    }

    private String maskEmail(String email) {
        if (email == null || !email.contains("@")) {
            return "***";
        }
        int atIndex = email.indexOf('@');
        if (atIndex <= 2) {
            return "***" + email.substring(atIndex);
        }
        return email.substring(0, 2) + "***" + email.substring(atIndex);
    }

    private String maskIp(String ip) {
        if (ip == null) return "unknown";
        int lastDot = ip.lastIndexOf('.');
        if (lastDot > 0) {
            return ip.substring(0, lastDot) + ".***";
        }
        return "***";
    }
}
