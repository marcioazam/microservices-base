package com.authplatform.usersvc.property;

import com.authplatform.usersvc.common.util.*;
import net.jqwik.api.*;
import org.junit.jupiter.api.Tag;
import static org.assertj.core.api.Assertions.assertThat;

/**
 * Property-based tests for input validation.
 * Validates: Requirements 1.1, 5.1, 8.1, 8.2, 8.3, 8.4
 */
@Tag("Feature: user-service, Property 9: Input Validation Rejection")
class InputValidationPropertyTest {

    private final PasswordValidator passwordValidator = new PasswordValidator();
    private final EmailValidator emailValidator = new EmailValidator("");
    private final DisplayNameValidator displayNameValidator = new DisplayNameValidator();
    private final InputSanitizer sanitizer = new InputSanitizer();

    @Property(tries = 100)
    @Label("Valid passwords pass validation")
    void validPasswordsPassValidation(@ForAll("validPasswords") String password) {
        var result = passwordValidator.validate(password);
        assertThat(result.isValid()).isTrue();
    }

    @Property(tries = 100)
    @Label("Short passwords fail validation")
    void shortPasswordsFailValidation(@ForAll("shortPasswords") String password) {
        var result = passwordValidator.validate(password);
        assertThat(result.isValid()).isFalse();
        assertThat(result.errors()).anyMatch(e -> e.contains("at least 8"));
    }

    @Property(tries = 100)
    @Label("Passwords without uppercase fail validation")
    void passwordsWithoutUppercaseFailValidation(@ForAll("passwordsWithoutUppercase") String password) {
        var result = passwordValidator.validate(password);
        assertThat(result.isValid()).isFalse();
        assertThat(result.errors()).anyMatch(e -> e.contains("uppercase"));
    }

    @Property(tries = 100)
    @Label("Valid emails pass validation")
    void validEmailsPassValidation(@ForAll("validEmails") String email) {
        var result = emailValidator.validate(email);
        assertThat(result.isValid()).isTrue();
    }

    @Property(tries = 100)
    @Label("Invalid email formats fail validation")
    void invalidEmailFormatsFailValidation(@ForAll("invalidEmails") String email) {
        var result = emailValidator.validate(email);
        assertThat(result.isValid()).isFalse();
    }

    @Property(tries = 100)
    @Label("Disposable emails fail validation")
    void disposableEmailsFailValidation(@ForAll("disposableEmails") String email) {
        var result = emailValidator.validate(email);
        assertThat(result.isValid()).isFalse();
        assertThat(result.errors()).anyMatch(e -> e.contains("Disposable"));
    }

    @Property(tries = 100)
    @Label("Valid display names pass validation")
    void validDisplayNamesPassValidation(@ForAll("validDisplayNames") String displayName) {
        var result = displayNameValidator.validate(displayName);
        assertThat(result.isValid()).isTrue();
    }

    @Property(tries = 100)
    @Label("Too long display names fail validation")
    void tooLongDisplayNamesFailValidation(@ForAll("tooLongDisplayNames") String displayName) {
        var result = displayNameValidator.validate(displayName);
        assertThat(result.isValid()).isFalse();
        assertThat(result.errors()).anyMatch(e -> e.contains("at most 100"));
    }

    @Property(tries = 100)
    @Label("Sanitizer removes script tags")
    void sanitizerRemovesScriptTags(@ForAll("stringsWithScripts") String input) {
        String sanitized = sanitizer.sanitize(input);
        assertThat(sanitized).doesNotContainIgnoringCase("<script");
        assertThat(sanitized).doesNotContainIgnoringCase("</script>");
    }

    @Property(tries = 100)
    @Label("Sanitizer escapes HTML entities")
    void sanitizerEscapesHtmlEntities(@ForAll("stringsWithHtml") String input) {
        String sanitized = sanitizer.sanitize(input);
        assertThat(sanitized).doesNotContain("<div>");
        assertThat(sanitized).doesNotContain("</div>");
    }

    @Provide
    Arbitrary<String> validPasswords() {
        return Arbitraries.strings()
                .withCharRange('a', 'z')
                .ofMinLength(6)
                .ofMaxLength(10)
                .map(s -> "A1" + s);
    }

    @Provide
    Arbitrary<String> shortPasswords() {
        return Arbitraries.strings()
                .alpha()
                .ofMinLength(1)
                .ofMaxLength(7);
    }

    @Provide
    Arbitrary<String> passwordsWithoutUppercase() {
        return Arbitraries.strings()
                .withCharRange('a', 'z')
                .ofMinLength(8)
                .ofMaxLength(20)
                .map(s -> s + "1");
    }

    @Provide
    Arbitrary<String> validEmails() {
        Arbitrary<String> local = Arbitraries.strings()
                .alpha()
                .ofMinLength(3)
                .ofMaxLength(10);
        Arbitrary<String> domain = Arbitraries.strings()
                .alpha()
                .ofMinLength(3)
                .ofMaxLength(8);
        return Combinators.combine(local, domain)
                .as((l, d) -> l.toLowerCase() + "@" + d.toLowerCase() + ".com");
    }

    @Provide
    Arbitrary<String> invalidEmails() {
        return Arbitraries.of(
                "notanemail",
                "missing@domain",
                "@nodomain.com",
                "spaces in@email.com",
                "double@@at.com"
        );
    }

    @Provide
    Arbitrary<String> disposableEmails() {
        return Arbitraries.strings()
                .alpha()
                .ofMinLength(3)
                .ofMaxLength(10)
                .map(s -> s.toLowerCase() + "@tempmail.com");
    }

    @Provide
    Arbitrary<String> validDisplayNames() {
        return Arbitraries.strings()
                .alpha()
                .ofMinLength(1)
                .ofMaxLength(50);
    }

    @Provide
    Arbitrary<String> tooLongDisplayNames() {
        return Arbitraries.strings()
                .alpha()
                .ofMinLength(101)
                .ofMaxLength(150);
    }

    @Provide
    Arbitrary<String> stringsWithScripts() {
        return Arbitraries.strings()
                .alpha()
                .ofMinLength(5)
                .ofMaxLength(20)
                .map(s -> "<script>alert('" + s + "')</script>");
    }

    @Provide
    Arbitrary<String> stringsWithHtml() {
        return Arbitraries.strings()
                .alpha()
                .ofMinLength(5)
                .ofMaxLength(20)
                .map(s -> "<div>" + s + "</div>");
    }
}
