package com.authplatform.usersvc.property.shared;

import com.authplatform.usersvc.shared.validation.ValidationService;
import net.jqwik.api.*;
import net.jqwik.api.constraints.IntRange;
import net.jqwik.api.constraints.StringLength;
import org.junit.jupiter.api.Tag;

import static org.assertj.core.api.Assertions.assertThat;

/**
 * Property-based tests for ValidationService.
 * Feature: user-service-modernization-2025
 */
@Tag("Feature: user-service-modernization-2025, Property 11: Validation Service Completeness")
class ValidationServicePropertyTest {

    private final ValidationService validationService = new ValidationService();

    // Property 11: Email Validation Completeness
    @Property(tries = 100)
    @Label("Property 11: Valid emails return valid=true")
    void validEmailsReturnValid(
            @ForAll("validLocalParts") String localPart,
            @ForAll("validDomains") String domain) {
        
        String email = localPart + "@" + domain;
        var result = validationService.validateEmail(email);
        
        assertThat(result.valid()).isTrue();
        assertThat(result.errors()).isEmpty();
    }

    @Property(tries = 100)
    @Label("Property 11: Invalid emails return valid=false with errors")
    void invalidEmailsReturnInvalidWithErrors(@ForAll("invalidEmails") String email) {
        var result = validationService.validateEmail(email);
        
        assertThat(result.valid()).isFalse();
        assertThat(result.errors()).isNotEmpty();
        assertThat(result.errors().get(0).field()).isEqualTo("email");
    }

    @Property(tries = 100)
    @Label("Property 11: Disposable emails are rejected")
    void disposableEmailsAreRejected(@ForAll("disposableDomains") String domain) {
        String email = "test@" + domain;
        var result = validationService.validateEmail(email);
        
        assertThat(result.valid()).isFalse();
        assertThat(result.errors()).anyMatch(e -> e.code().equals("DISPOSABLE_EMAIL"));
    }

    // Property 11: Password Validation Completeness
    @Property(tries = 100)
    @Label("Property 11: Valid passwords return valid=true")
    void validPasswordsReturnValid(@ForAll("validPasswords") String password) {
        var result = validationService.validatePassword(password);
        
        assertThat(result.valid()).isTrue();
        assertThat(result.errors()).isEmpty();
    }

    @Property(tries = 100)
    @Label("Property 11: Short passwords return valid=false")
    void shortPasswordsReturnInvalid(@ForAll @StringLength(min = 1, max = 7) String password) {
        var result = validationService.validatePassword(password);
        
        assertThat(result.valid()).isFalse();
        assertThat(result.errors()).anyMatch(e -> e.code().equals("TOO_SHORT"));
    }

    @Property(tries = 100)
    @Label("Property 11: Passwords without uppercase return error")
    void passwordsWithoutUppercaseReturnError() {
        String password = "password1!";
        var result = validationService.validatePassword(password);
        
        assertThat(result.valid()).isFalse();
        assertThat(result.errors()).anyMatch(e -> e.code().equals("MISSING_UPPERCASE"));
    }

    // Property 11: Display Name Validation Completeness
    @Property(tries = 100)
    @Label("Property 11: Valid display names return valid=true")
    void validDisplayNamesReturnValid(@ForAll("validDisplayNames") String displayName) {
        var result = validationService.validateDisplayName(displayName);
        
        assertThat(result.valid()).isTrue();
        assertThat(result.errors()).isEmpty();
    }

    @Property(tries = 100)
    @Label("Property 11: Short display names return valid=false")
    void shortDisplayNamesReturnInvalid(@ForAll @StringLength(max = 1) String displayName) {
        if (displayName.isBlank()) return;
        var result = validationService.validateDisplayName(displayName);
        
        assertThat(result.valid()).isFalse();
        assertThat(result.errors()).anyMatch(e -> e.code().equals("TOO_SHORT"));
    }

    @Property(tries = 100)
    @Label("Property 11: Display names with scripts are rejected")
    void displayNamesWithScriptsAreRejected(@ForAll("scriptInjections") String script) {
        var result = validationService.validateDisplayName(script);
        
        assertThat(result.valid()).isFalse();
        assertThat(result.errors()).anyMatch(e -> e.code().equals("INVALID_CONTENT"));
    }

    @Property(tries = 100)
    @Label("Property 11: Sanitization escapes HTML")
    void sanitizationEscapesHtml(@ForAll("htmlContent") String html) {
        String sanitized = validationService.sanitizeDisplayName(html);
        
        assertThat(sanitized).doesNotContain("<script>");
        assertThat(sanitized).doesNotContain("</script>");
    }

    // Providers
    @Provide
    Arbitrary<String> validLocalParts() {
        return Arbitraries.strings()
                .withCharRange('a', 'z')
                .ofMinLength(3)
                .ofMaxLength(15);
    }

    @Provide
    Arbitrary<String> validDomains() {
        return Arbitraries.of("example.com", "test.org", "company.net", "mail.io");
    }

    @Provide
    Arbitrary<String> invalidEmails() {
        return Arbitraries.of(
                "invalid", "no@domain", "@nodomain.com", "spaces in@email.com",
                "missing.at.sign", "", "   "
        );
    }

    @Provide
    Arbitrary<String> disposableDomains() {
        return Arbitraries.of(
                "tempmail.com", "throwaway.com", "mailinator.com", "guerrillamail.com"
        );
    }

    @Provide
    Arbitrary<String> validPasswords() {
        return Arbitraries.of(
                "Password1!", "SecureP@ss123", "MyP@ssw0rd!", "Test1234!@#"
        );
    }

    @Provide
    Arbitrary<String> validDisplayNames() {
        return Arbitraries.strings()
                .withCharRange('a', 'z')
                .ofMinLength(2)
                .ofMaxLength(50);
    }

    @Provide
    Arbitrary<String> scriptInjections() {
        return Arbitraries.of(
                "<script>alert('xss')</script>",
                "javascript:alert(1)",
                "onclick=alert(1)",
                "<script src='evil.js'></script>"
        );
    }

    @Provide
    Arbitrary<String> htmlContent() {
        return Arbitraries.of(
                "<script>alert('xss')</script>",
                "<b>bold</b>",
                "<img src='x' onerror='alert(1)'>"
        );
    }
}
