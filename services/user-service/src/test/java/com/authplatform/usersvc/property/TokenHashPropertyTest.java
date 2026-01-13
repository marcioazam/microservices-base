package com.authplatform.usersvc.property;

import com.authplatform.usersvc.common.util.TokenHasher;
import net.jqwik.api.*;
import net.jqwik.api.constraints.NotBlank;
import org.junit.jupiter.api.Tag;
import static org.assertj.core.api.Assertions.assertThat;

/**
 * Property-based tests for token hashing.
 * Validates: Requirements 1.8, 2.1
 */
@Tag("Feature: user-service, Property 3: Token Hash Consistency")
class TokenHashPropertyTest {

    private final TokenHasher hasher = new TokenHasher();

    @Property(tries = 100)
    @Label("Hash produces consistent 64-character hex string")
    void hashProducesConsistent64CharHex(@ForAll @NotBlank String token) {
        String hash = hasher.hash(token);

        assertThat(hash).hasSize(64);
        assertThat(hash).matches("[0-9a-f]{64}");
    }

    @Property(tries = 100)
    @Label("Same token always produces same hash")
    void sameTokenProducesSameHash(@ForAll @NotBlank String token) {
        String hash1 = hasher.hash(token);
        String hash2 = hasher.hash(token);

        assertThat(hash1).isEqualTo(hash2);
    }

    @Property(tries = 100)
    @Label("Different tokens produce different hashes")
    void differentTokensProduceDifferentHashes(
            @ForAll @NotBlank String token1,
            @ForAll @NotBlank String token2) {
        Assume.that(!token1.equals(token2));

        String hash1 = hasher.hash(token1);
        String hash2 = hasher.hash(token2);

        assertThat(hash1).isNotEqualTo(hash2);
    }

    @Property(tries = 100)
    @Label("Verify returns true for matching token and hash")
    void verifyReturnsTrueForMatchingTokenAndHash(@ForAll @NotBlank String token) {
        String hash = hasher.hash(token);

        assertThat(hasher.verify(token, hash)).isTrue();
    }

    @Property(tries = 100)
    @Label("Verify returns false for non-matching token")
    void verifyReturnsFalseForNonMatchingToken(
            @ForAll @NotBlank String token,
            @ForAll @NotBlank String wrongToken) {
        Assume.that(!token.equals(wrongToken));

        String hash = hasher.hash(token);

        assertThat(hasher.verify(wrongToken, hash)).isFalse();
    }

    @Property(tries = 100)
    @Label("Generated tokens have sufficient entropy")
    void generatedTokensHaveSufficientEntropy() {
        String token1 = hasher.generateToken();
        String token2 = hasher.generateToken();

        assertThat(token1).isNotEqualTo(token2);
        assertThat(token1.length()).isGreaterThanOrEqualTo(32);
    }

    @Property(tries = 100)
    @Label("Generated token hash round-trip works")
    void generatedTokenHashRoundTripWorks() {
        String token = hasher.generateToken();
        String hash = hasher.hash(token);

        assertThat(hasher.verify(token, hash)).isTrue();
    }

    @Example
    @Label("Null token throws exception")
    void nullTokenThrowsException() {
        org.junit.jupiter.api.Assertions.assertThrows(
                IllegalArgumentException.class,
                () -> hasher.hash(null)
        );
    }

    @Example
    @Label("Verify with null returns false")
    void verifyWithNullReturnsFalse() {
        assertThat(hasher.verify(null, "somehash")).isFalse();
        assertThat(hasher.verify("sometoken", null)).isFalse();
    }
}
