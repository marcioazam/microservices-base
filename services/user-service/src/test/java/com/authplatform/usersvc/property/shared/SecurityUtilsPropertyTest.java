package com.authplatform.usersvc.property.shared;

import com.authplatform.usersvc.shared.security.SecurityUtils;
import net.jqwik.api.*;
import net.jqwik.api.constraints.IntRange;
import org.junit.jupiter.api.Tag;

import static org.assertj.core.api.Assertions.assertThat;

/**
 * Property-based tests for SecurityUtils.
 * Feature: user-service-modernization-2025
 */
@Tag("Feature: user-service-modernization-2025, Property 2: Sensitive Data Masking Consistency")
class SecurityUtilsPropertyTest {

    private final SecurityUtils securityUtils = new SecurityUtils();

    // Property 2: IP Masking - For any valid IPv4, last octet replaced with ***
    @Property(tries = 100)
    @Label("Property 2: IP masking replaces last octet with ***")
    void ipMaskingReplacesLastOctetWithStars(
            @ForAll @IntRange(min = 0, max = 255) int o1,
            @ForAll @IntRange(min = 0, max = 255) int o2,
            @ForAll @IntRange(min = 0, max = 255) int o3,
            @ForAll @IntRange(min = 0, max = 255) int o4) {
        
        String ip = String.format("%d.%d.%d.%d", o1, o2, o3, o4);
        String masked = securityUtils.maskIp(ip);
        
        String expected = String.format("%d.%d.%d.***", o1, o2, o3);
        assertThat(masked).isEqualTo(expected);
        assertThat(masked).doesNotContain(String.valueOf(o4));
        assertThat(masked).endsWith(".***");
    }

    @Property(tries = 100)
    @Label("Property 2: IP masking never exposes full IP")
    void ipMaskingNeverExposesFullIp(
            @ForAll @IntRange(min = 0, max = 255) int o1,
            @ForAll @IntRange(min = 0, max = 255) int o2,
            @ForAll @IntRange(min = 0, max = 255) int o3,
            @ForAll @IntRange(min = 0, max = 255) int o4) {
        
        String ip = String.format("%d.%d.%d.%d", o1, o2, o3, o4);
        String masked = securityUtils.maskIp(ip);
        
        assertThat(masked).isNotEqualTo(ip);
        assertThat(masked).contains("***");
    }

    // Property 2: Email Masking - Characters after first 2 before @ replaced with ***
    @Property(tries = 100)
    @Label("Property 2: Email masking keeps first 2 chars and domain")
    void emailMaskingKeepsFirst2CharsAndDomain(
            @ForAll("validEmailLocalParts") String localPart,
            @ForAll("validDomains") String domain) {
        
        String email = localPart + "@" + domain;
        String masked = securityUtils.maskEmail(email);
        
        String expectedPrefix = localPart.substring(0, Math.min(2, localPart.length())).toLowerCase();
        assertThat(masked).startsWith(expectedPrefix);
        assertThat(masked).contains("***");
        assertThat(masked).endsWith("@" + domain.toLowerCase());
    }

    @Property(tries = 100)
    @Label("Property 2: Email masking never exposes full local part")
    void emailMaskingNeverExposesFullLocalPart(
            @ForAll("validEmailLocalParts") String localPart,
            @ForAll("validDomains") String domain) {
        
        if (localPart.length() <= 2) return; // Skip short local parts
        
        String email = localPart + "@" + domain;
        String masked = securityUtils.maskEmail(email);
        
        assertThat(masked).doesNotContain(localPart.toLowerCase());
    }

    @Provide
    Arbitrary<String> validEmailLocalParts() {
        return Arbitraries.strings()
                .withCharRange('a', 'z')
                .ofMinLength(3)
                .ofMaxLength(20);
    }

    @Provide
    Arbitrary<String> validDomains() {
        return Arbitraries.of("example.com", "test.org", "mail.io", "company.net");
    }

    // Correlation ID tests
    @Property(tries = 100)
    @Label("Correlation ID: provided ID is returned unchanged")
    void correlationIdReturnsProvidedId(@ForAll("uuids") String uuid) {
        String result = securityUtils.getOrCreateCorrelationId(uuid);
        assertThat(result).isEqualTo(uuid.trim());
    }

    @Property(tries = 100)
    @Label("Correlation ID: null/blank generates new UUID")
    void correlationIdGeneratesNewForNullOrBlank(@ForAll("nullOrBlank") String input) {
        String result = securityUtils.getOrCreateCorrelationId(input);
        assertThat(result).isNotBlank();
        assertThat(result).matches("[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}");
    }

    @Provide
    Arbitrary<String> uuids() {
        return Arbitraries.create(() -> java.util.UUID.randomUUID().toString());
    }

    @Provide
    Arbitrary<String> nullOrBlank() {
        return Arbitraries.of(null, "", "   ", "\t", "\n");
    }
}
