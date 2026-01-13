package com.authplatform.usersvc.property;

import com.authplatform.usersvc.common.util.EmailNormalizer;
import net.jqwik.api.*;
import net.jqwik.api.constraints.NotBlank;
import org.junit.jupiter.api.Tag;
import static org.assertj.core.api.Assertions.assertThat;

/**
 * Property-based tests for email normalization.
 * Validates: Requirements 1.2, 3.1
 */
@Tag("Feature: user-service, Property 1: Email Normalization Idempotence")
class EmailNormalizationPropertyTest {

    private final EmailNormalizer normalizer = new EmailNormalizer();

    @Property(tries = 100)
    @Label("Email normalization is idempotent")
    void emailNormalizationIsIdempotent(@ForAll @NotBlank String email) {
        String normalized = normalizer.normalize(email);
        String normalizedAgain = normalizer.normalize(normalized);

        assertThat(normalized).isEqualTo(normalizedAgain);
    }

    @Property(tries = 100)
    @Label("Normalized email is lowercase and trimmed")
    void normalizedEmailIsLowercaseAndTrimmed(@ForAll @NotBlank String email) {
        String normalized = normalizer.normalize(email);

        assertThat(normalized).isEqualTo(email.toLowerCase().trim());
    }

    @Property(tries = 100)
    @Label("Normalized email has no leading or trailing whitespace")
    void normalizedEmailHasNoWhitespace(@ForAll("emailsWithWhitespace") String email) {
        String normalized = normalizer.normalize(email);

        assertThat(normalized).doesNotStartWith(" ");
        assertThat(normalized).doesNotEndWith(" ");
        assertThat(normalized).doesNotStartWith("\t");
        assertThat(normalized).doesNotEndWith("\t");
    }

    @Property(tries = 100)
    @Label("Normalized email preserves content")
    void normalizedEmailPreservesContent(@ForAll("validEmails") String email) {
        String normalized = normalizer.normalize(email);

        assertThat(normalized).contains("@");
        assertThat(normalized.split("@")).hasSize(2);
    }

    @Property(tries = 100)
    @Label("Null email returns null")
    void nullEmailReturnsNull() {
        assertThat(normalizer.normalize(null)).isNull();
    }

    @Provide
    Arbitrary<String> emailsWithWhitespace() {
        return Arbitraries.strings()
                .alpha()
                .ofMinLength(3)
                .ofMaxLength(20)
                .map(s -> "  " + s + "@example.com  ");
    }

    @Provide
    Arbitrary<String> validEmails() {
        Arbitrary<String> localPart = Arbitraries.strings()
                .alpha()
                .ofMinLength(1)
                .ofMaxLength(20);
        Arbitrary<String> domain = Arbitraries.strings()
                .alpha()
                .ofMinLength(2)
                .ofMaxLength(10);

        return Combinators.combine(localPart, domain)
                .as((local, dom) -> local + "@" + dom + ".com");
    }
}
