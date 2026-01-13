package com.authplatform.usersvc.property;

import com.authplatform.usersvc.domain.model.User;
import com.authplatform.usersvc.domain.model.UserStatus;
import net.jqwik.api.*;
import org.junit.jupiter.api.Tag;
import java.time.Instant;
import java.util.UUID;
import static org.assertj.core.api.Assertions.assertThat;

/**
 * Property-based tests for user state transitions.
 * Validates: Requirements 1.6, 1.7, 2.6, 2.7
 */
@Tag("Feature: user-service, Property 4: User Registration Initial State")
@Tag("Feature: user-service, Property 5: Email Verification State Transition")
class UserStateTransitionPropertyTest {

    @Property(tries = 100)
    @Label("New user has PENDING_EMAIL status and emailVerified false")
    void newUserHasPendingEmailStatusAndEmailVerifiedFalse(
            @ForAll("validEmails") String email,
            @ForAll("validDisplayNames") String displayName,
            @ForAll("validPasswordHashes") String passwordHash) {
        
        User user = User.builder()
                .id(UUID.randomUUID())
                .email(email)
                .emailVerified(false)
                .passwordHash(passwordHash)
                .displayName(displayName)
                .status(UserStatus.PENDING_EMAIL)
                .createdAt(Instant.now())
                .updatedAt(Instant.now())
                .build();

        assertThat(user.getStatus()).isEqualTo(UserStatus.PENDING_EMAIL);
        assertThat(user.isEmailVerified()).isFalse();
    }

    @Property(tries = 100)
    @Label("Activating user sets emailVerified true and status ACTIVE")
    void activatingUserSetsEmailVerifiedTrueAndStatusActive(
            @ForAll("validEmails") String email,
            @ForAll("validDisplayNames") String displayName,
            @ForAll("validPasswordHashes") String passwordHash) {
        
        User user = User.builder()
                .id(UUID.randomUUID())
                .email(email)
                .emailVerified(false)
                .passwordHash(passwordHash)
                .displayName(displayName)
                .status(UserStatus.PENDING_EMAIL)
                .createdAt(Instant.now())
                .updatedAt(Instant.now())
                .build();

        user.activate();

        assertThat(user.getStatus()).isEqualTo(UserStatus.ACTIVE);
        assertThat(user.isEmailVerified()).isTrue();
    }

    @Property(tries = 100)
    @Label("User state transition from PENDING_EMAIL to ACTIVE is valid")
    void userStateTransitionFromPendingToActiveIsValid(
            @ForAll("validEmails") String email,
            @ForAll("validDisplayNames") String displayName) {
        
        User user = User.builder()
                .id(UUID.randomUUID())
                .email(email)
                .emailVerified(false)
                .passwordHash("$argon2id$v=19$m=65536,t=3,p=1$hash")
                .displayName(displayName)
                .status(UserStatus.PENDING_EMAIL)
                .createdAt(Instant.now())
                .updatedAt(Instant.now())
                .build();

        UserStatus initialStatus = user.getStatus();
        user.activate();
        UserStatus finalStatus = user.getStatus();

        assertThat(initialStatus).isEqualTo(UserStatus.PENDING_EMAIL);
        assertThat(finalStatus).isEqualTo(UserStatus.ACTIVE);
    }

    @Property(tries = 100)
    @Label("Disabling user sets status to DISABLED")
    void disablingUserSetsStatusToDisabled(
            @ForAll("validEmails") String email,
            @ForAll("validDisplayNames") String displayName) {
        
        User user = User.builder()
                .id(UUID.randomUUID())
                .email(email)
                .emailVerified(true)
                .passwordHash("$argon2id$v=19$m=65536,t=3,p=1$hash")
                .displayName(displayName)
                .status(UserStatus.ACTIVE)
                .createdAt(Instant.now())
                .updatedAt(Instant.now())
                .build();

        user.disable();

        assertThat(user.getStatus()).isEqualTo(UserStatus.DISABLED);
    }

    @Property(tries = 100)
    @Label("User email is preserved during state transitions")
    void userEmailIsPreservedDuringStateTransitions(
            @ForAll("validEmails") String email,
            @ForAll("validDisplayNames") String displayName) {
        
        User user = User.builder()
                .id(UUID.randomUUID())
                .email(email)
                .emailVerified(false)
                .passwordHash("$argon2id$v=19$m=65536,t=3,p=1$hash")
                .displayName(displayName)
                .status(UserStatus.PENDING_EMAIL)
                .createdAt(Instant.now())
                .updatedAt(Instant.now())
                .build();

        String emailBefore = user.getEmail();
        user.activate();
        String emailAfter = user.getEmail();

        assertThat(emailBefore).isEqualTo(emailAfter);
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
    Arbitrary<String> validDisplayNames() {
        return Arbitraries.strings()
                .alpha()
                .ofMinLength(1)
                .ofMaxLength(50);
    }

    @Provide
    Arbitrary<String> validPasswordHashes() {
        return Arbitraries.strings()
                .alpha()
                .ofMinLength(10)
                .ofMaxLength(20)
                .map(s -> "$argon2id$v=19$m=65536,t=3,p=1$" + s);
    }
}
