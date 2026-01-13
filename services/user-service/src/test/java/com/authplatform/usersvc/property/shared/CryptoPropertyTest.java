package com.authplatform.usersvc.property.shared;

import com.authplatform.usersvc.shared.crypto.PasswordService;
import com.authplatform.usersvc.shared.crypto.TokenHasher;
import net.jqwik.api.*;
import net.jqwik.api.constraints.StringLength;
import org.junit.jupiter.api.Tag;

import static org.assertj.core.api.Assertions.assertThat;

/**
 * Property-based tests for PasswordService and TokenHasher.
 * Feature: user-service-modernization-2025
 */
class CryptoPropertyTest {

    private final PasswordService passwordService = new PasswordService(19456, 2, 1);
    private final TokenHasher tokenHasher = new TokenHasher();

    // Property 5: Password Hash Format Compliance
    @Property(tries = 100)
    @Tag("Feature: user-service-modernization-2025, Property 5: Password Hash Format Compliance")
    @Label("Property 5: Password hash starts with $argon2id$")
    void passwordHashStartsWithArgon2id(@ForAll("validPasswords") String password) {
        String hash = passwordService.hash(password);
        
        assertThat(hash).startsWith("$argon2id$");
        assertThat(hash).contains("$v=");
        assertThat(hash).contains("$m=");
        assertThat(hash).contains(",t=");
        assertThat(hash).contains(",p=");
    }

    // Property 6: Password Hash Round-Trip Verification
    @Property(tries = 100)
    @Tag("Feature: user-service-modernization-2025, Property 6: Password Hash Round-Trip Verification")
    @Label("Property 6: Same password verifies true")
    void samePasswordVerifiesTrue(@ForAll("validPasswords") String password) {
        String hash = passwordService.hash(password);
        
        assertThat(passwordService.verify(password, hash)).isTrue();
    }

    @Property(tries = 100)
    @Tag("Feature: user-service-modernization-2025, Property 6: Password Hash Round-Trip Verification")
    @Label("Property 6: Different password verifies false")
    void differentPasswordVerifiesFalse(
            @ForAll("validPasswords") String password1,
            @ForAll("validPasswords") String password2) {
        
        Assume.that(!password1.equals(password2));
        
        String hash = passwordService.hash(password1);
        assertThat(passwordService.verify(password2, hash)).isFalse();
    }

    // Property 7: Token Hash Determinism
    @Property(tries = 100)
    @Tag("Feature: user-service-modernization-2025, Property 7: Token Hash Determinism")
    @Label("Property 7: Same token produces same hash")
    void sameTokenProducesSameHash(@ForAll("tokens") String token) {
        String hash1 = tokenHasher.hash(token);
        String hash2 = tokenHasher.hash(token);
        
        assertThat(hash1).isEqualTo(hash2);
        assertThat(hash1).hasSize(64);
        assertThat(hash1).matches("[0-9a-f]{64}");
    }

    @Property(tries = 100)
    @Tag("Feature: user-service-modernization-2025, Property 7: Token Hash Determinism")
    @Label("Property 7: Different tokens produce different hashes")
    void differentTokensProduceDifferentHashes(
            @ForAll("tokens") String token1,
            @ForAll("tokens") String token2) {
        
        Assume.that(!token1.equals(token2));
        
        String hash1 = tokenHasher.hash(token1);
        String hash2 = tokenHasher.hash(token2);
        
        assertThat(hash1).isNotEqualTo(hash2);
    }

    // Property 8: Token Verification Round-Trip
    @Property(tries = 100)
    @Tag("Feature: user-service-modernization-2025, Property 8: Token Verification Round-Trip")
    @Label("Property 8: Generated token verifies correctly")
    void generatedTokenVerifiesCorrectly() {
        String token = tokenHasher.generateToken();
        String hash = tokenHasher.hash(token);
        
        assertThat(tokenHasher.verify(token, hash)).isTrue();
        assertThat(token).hasSize(64);
    }

    @Property(tries = 100)
    @Tag("Feature: user-service-modernization-2025, Property 8: Token Verification Round-Trip")
    @Label("Property 8: Wrong token fails verification")
    void wrongTokenFailsVerification() {
        String token1 = tokenHasher.generateToken();
        String token2 = tokenHasher.generateToken();
        String hash = tokenHasher.hash(token1);
        
        assertThat(tokenHasher.verify(token2, hash)).isFalse();
    }

    // Providers
    @Provide
    Arbitrary<String> validPasswords() {
        return Arbitraries.strings()
                .withCharRange('a', 'z')
                .withCharRange('A', 'Z')
                .withCharRange('0', '9')
                .withChars('!', '@', '#', '$')
                .ofMinLength(8)
                .ofMaxLength(32);
    }

    @Provide
    Arbitrary<String> tokens() {
        return Arbitraries.strings()
                .withCharRange('a', 'f')
                .withCharRange('0', '9')
                .ofLength(64);
    }
}
