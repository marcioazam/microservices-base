package com.authplatform.usersvc.property;

import com.authplatform.usersvc.domain.service.PasswordService;
import com.authplatform.usersvc.domain.service.impl.PasswordServiceImpl;
import net.jqwik.api.*;
import net.jqwik.api.constraints.NotBlank;
import net.jqwik.api.constraints.StringLength;
import org.junit.jupiter.api.Tag;
import static org.assertj.core.api.Assertions.assertThat;

/**
 * Property-based tests for password hashing with Argon2id.
 * Validates: Requirements 1.5, 6.1
 */
@Tag("Feature: user-service, Property 2: Password Hashing Round-Trip")
class PasswordHashingPropertyTest {

    // Use lower memory for tests to run faster
    private final PasswordService passwordService = new PasswordServiceImpl(2, 16384, 1);

    @Property(tries = 50) // Reduced tries due to expensive hashing
    @Label("Password hash round-trip verification succeeds")
    void passwordHashRoundTripVerificationSucceeds(
            @ForAll @NotBlank @StringLength(min = 8, max = 64) String password) {
        String hash = passwordService.hash(password);
        
        assertThat(passwordService.verify(password, hash)).isTrue();
    }

    @Property(tries = 50)
    @Label("Hash produces valid Argon2id format")
    void hashProducesValidArgon2idFormat(
            @ForAll @NotBlank @StringLength(min = 8, max = 64) String password) {
        String hash = passwordService.hash(password);
        
        assertThat(hash).startsWith("$argon2id$");
        assertThat(passwordService.isArgon2idHash(hash)).isTrue();
    }

    @Property(tries = 50)
    @Label("Same password produces different hashes (salt)")
    void samePasswordProducesDifferentHashes(
            @ForAll @NotBlank @StringLength(min = 8, max = 64) String password) {
        String hash1 = passwordService.hash(password);
        String hash2 = passwordService.hash(password);
        
        assertThat(hash1).isNotEqualTo(hash2);
        // But both should verify
        assertThat(passwordService.verify(password, hash1)).isTrue();
        assertThat(passwordService.verify(password, hash2)).isTrue();
    }

    @Property(tries = 50)
    @Label("Wrong password fails verification")
    void wrongPasswordFailsVerification(
            @ForAll @NotBlank @StringLength(min = 8, max = 32) String password,
            @ForAll @NotBlank @StringLength(min = 8, max = 32) String wrongPassword) {
        Assume.that(!password.equals(wrongPassword));
        
        String hash = passwordService.hash(password);
        
        assertThat(passwordService.verify(wrongPassword, hash)).isFalse();
    }

    @Property(tries = 50)
    @Label("Hash contains algorithm parameters")
    void hashContainsAlgorithmParameters(
            @ForAll @NotBlank @StringLength(min = 8, max = 64) String password) {
        String hash = passwordService.hash(password);
        
        // Argon2id hash format: $argon2id$v=19$m=...,t=...,p=...$salt$hash
        assertThat(hash).contains("$v=");
        assertThat(hash).contains("$m=");
        assertThat(hash).contains(",t=");
        assertThat(hash).contains(",p=");
    }

    @Example
    @Label("Null password throws exception")
    void nullPasswordThrowsException() {
        org.junit.jupiter.api.Assertions.assertThrows(
                IllegalArgumentException.class,
                () -> passwordService.hash(null)
        );
    }

    @Example
    @Label("Empty password throws exception")
    void emptyPasswordThrowsException() {
        org.junit.jupiter.api.Assertions.assertThrows(
                IllegalArgumentException.class,
                () -> passwordService.hash("")
        );
    }

    @Example
    @Label("Verify with null returns false")
    void verifyWithNullReturnsFalse() {
        assertThat(passwordService.verify(null, "somehash")).isFalse();
        assertThat(passwordService.verify("password", null)).isFalse();
    }
}
