package com.authplatform.usersvc.property;

import com.authplatform.usersvc.domain.model.EmailVerificationToken;
import net.jqwik.api.*;
import org.junit.jupiter.api.Tag;
import java.time.Instant;
import java.time.temporal.ChronoUnit;
import java.util.UUID;
import static org.assertj.core.api.Assertions.assertThat;

/**
 * Property-based tests for email verification state transitions.
 * Validates: Requirements 2.6, 2.7
 */
@Tag("Feature: user-service, Property 5: Email Verification State Transition")
class EmailVerificationPropertyTest {

    @Property(tries = 100)
    @Label("Valid token is not expired and not used")
    void validTokenIsNotExpiredAndNotUsed(@ForAll("validTokens") EmailVerificationToken token) {
        assertThat(token.isExpired()).isFalse();
        assertThat(token.isUsed()).isFalse();
    }

    @Property(tries = 100)
    @Label("Expired token is detected correctly")
    void expiredTokenIsDetectedCorrectly(@ForAll("expiredTokens") EmailVerificationToken token) {
        assertThat(token.isExpired()).isTrue();
    }

    @Property(tries = 100)
    @Label("Used token is detected correctly")
    void usedTokenIsDetectedCorrectly(@ForAll("usedTokens") EmailVerificationToken token) {
        assertThat(token.isUsed()).isTrue();
    }

    @Property(tries = 100)
    @Label("Marking token as used sets usedAt timestamp")
    void markingTokenAsUsedSetsUsedAtTimestamp(@ForAll("validTokens") EmailVerificationToken token) {
        assertThat(token.getUsedAt()).isNull();
        
        token.markAsUsed();
        
        assertThat(token.getUsedAt()).isNotNull();
        assertThat(token.isUsed()).isTrue();
    }

    @Property(tries = 100)
    @Label("Token hash is preserved after marking as used")
    void tokenHashIsPreservedAfterMarkingAsUsed(@ForAll("validTokens") EmailVerificationToken token) {
        String hashBefore = token.getTokenHash();
        
        token.markAsUsed();
        
        assertThat(token.getTokenHash()).isEqualTo(hashBefore);
    }

    @Property(tries = 100)
    @Label("Incrementing attempt count increases by one")
    void incrementingAttemptCountIncreasesByOne(@ForAll("validTokens") EmailVerificationToken token) {
        int countBefore = token.getAttemptCount();
        
        token.incrementAttemptCount();
        
        assertThat(token.getAttemptCount()).isEqualTo(countBefore + 1);
    }

    @Provide
    Arbitrary<EmailVerificationToken> validTokens() {
        return Arbitraries.of(1).map(i -> EmailVerificationToken.builder()
                .id(UUID.randomUUID())
                .userId(UUID.randomUUID())
                .tokenHash(generateRandomHash())
                .expiresAt(Instant.now().plus(1, ChronoUnit.HOURS))
                .usedAt(null)
                .createdAt(Instant.now())
                .attemptCount(0)
                .build());
    }

    @Provide
    Arbitrary<EmailVerificationToken> expiredTokens() {
        return Arbitraries.of(1).map(i -> EmailVerificationToken.builder()
                .id(UUID.randomUUID())
                .userId(UUID.randomUUID())
                .tokenHash(generateRandomHash())
                .expiresAt(Instant.now().minus(1, ChronoUnit.HOURS))
                .usedAt(null)
                .createdAt(Instant.now().minus(2, ChronoUnit.HOURS))
                .attemptCount(0)
                .build());
    }

    @Provide
    Arbitrary<EmailVerificationToken> usedTokens() {
        return Arbitraries.of(1).map(i -> EmailVerificationToken.builder()
                .id(UUID.randomUUID())
                .userId(UUID.randomUUID())
                .tokenHash(generateRandomHash())
                .expiresAt(Instant.now().plus(1, ChronoUnit.HOURS))
                .usedAt(Instant.now().minus(30, ChronoUnit.MINUTES))
                .createdAt(Instant.now().minus(1, ChronoUnit.HOURS))
                .attemptCount(1)
                .build());
    }

    private String generateRandomHash() {
        return UUID.randomUUID().toString().replace("-", "") + 
               UUID.randomUUID().toString().replace("-", "");
    }
}
