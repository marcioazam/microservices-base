package com.authplatform.usersvc.property.domain;

import com.authplatform.usersvc.domain.ratelimit.RateLimitService;
import net.jqwik.api.*;
import org.junit.jupiter.api.Tag;

import static org.assertj.core.api.Assertions.assertThat;

/**
 * Property-based tests for RateLimitService.
 * Feature: user-service-modernization-2025
 */
class RateLimitPropertyTest {

    // Property 4: Rate Limit Namespace Consistency
    @Property(tries = 100)
    @Tag("Feature: user-service-modernization-2025, Property 4: Rate Limit Namespace Consistency")
    @Label("Property 4: All rate limit keys are prefixed with namespace")
    void allRateLimitKeysArePrefixedWithNamespace(@ForAll("rateLimitKeys") String key) {
        String fullKey = RateLimitService.NAMESPACE + ":" + key;
        
        assertThat(fullKey).startsWith("user-service:ratelimit:");
        assertThat(fullKey).contains(key);
    }

    @Property(tries = 100)
    @Tag("Feature: user-service-modernization-2025, Property 4: Rate Limit Namespace Consistency")
    @Label("Property 4: Registration keys follow pattern")
    void registrationKeysFollowPattern(@ForAll("ipAddresses") String ip) {
        String key = "registration:ip:" + ip;
        String fullKey = RateLimitService.NAMESPACE + ":" + key;
        
        assertThat(fullKey).startsWith("user-service:ratelimit:registration:ip:");
    }

    @Property(tries = 100)
    @Tag("Feature: user-service-modernization-2025, Property 4: Rate Limit Namespace Consistency")
    @Label("Property 4: Resend email keys follow pattern")
    void resendEmailKeysFollowPattern(@ForAll("emails") String email) {
        String key = "resend:email:" + email.toLowerCase();
        String fullKey = RateLimitService.NAMESPACE + ":" + key;
        
        assertThat(fullKey).startsWith("user-service:ratelimit:resend:email:");
    }

    @Property(tries = 100)
    @Tag("Feature: user-service-modernization-2025, Property 4: Rate Limit Namespace Consistency")
    @Label("Property 4: Verify keys follow pattern")
    void verifyKeysFollowPattern(@ForAll("ipAddresses") String ip) {
        String key = "verify:ip:" + ip;
        String fullKey = RateLimitService.NAMESPACE + ":" + key;
        
        assertThat(fullKey).startsWith("user-service:ratelimit:verify:ip:");
    }

    // Providers
    @Provide
    Arbitrary<String> rateLimitKeys() {
        return Arbitraries.of(
                "registration:ip:192.168.1.1",
                "resend:email:test@example.com",
                "resend:ip:10.0.0.1",
                "verify:ip:172.16.0.1"
        );
    }

    @Provide
    Arbitrary<String> ipAddresses() {
        return Arbitraries.integers().between(0, 255)
                .tuple4()
                .map(t -> t.get1() + "." + t.get2() + "." + t.get3() + "." + t.get4());
    }

    @Provide
    Arbitrary<String> emails() {
        return Arbitraries.strings()
                .withCharRange('a', 'z')
                .ofMinLength(3)
                .ofMaxLength(10)
                .map(s -> s + "@example.com");
    }
}
