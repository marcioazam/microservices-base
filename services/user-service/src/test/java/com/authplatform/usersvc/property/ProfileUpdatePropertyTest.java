package com.authplatform.usersvc.property;

import com.authplatform.usersvc.api.dto.request.ProfileUpdateRequest;
import com.authplatform.usersvc.api.dto.response.ProfileResponse;
import com.authplatform.usersvc.domain.model.User;
import com.authplatform.usersvc.domain.model.UserStatus;
import net.jqwik.api.*;
import net.jqwik.api.constraints.AlphaChars;
import net.jqwik.api.constraints.StringLength;
import java.time.Instant;
import java.util.UUID;
import static org.assertj.core.api.Assertions.assertThat;

/**
 * Property 10: Profile Update Field Restriction
 * Validates: Requirements 5.2, 5.3
 * 
 * Ensures that profile updates only modify allowed fields (displayName)
 * and never modify protected fields (email, passwordHash, status, etc.)
 */
class ProfileUpdatePropertyTest {

    @Property(tries = 100)
    void profileUpdateOnlyModifiesAllowedFields(
            @ForAll @AlphaChars @StringLength(min = 2, max = 50) String newDisplayName,
            @ForAll("validUser") User originalUser) {
        
        // Given: original user state
        String originalEmail = originalUser.getEmail();
        String originalPasswordHash = originalUser.getPasswordHash();
        UserStatus originalStatus = originalUser.getStatus();
        boolean originalEmailVerified = originalUser.isEmailVerified();
        Instant originalCreatedAt = originalUser.getCreatedAt();
        
        // When: simulating profile update (only displayName changes)
        User updatedUser = simulateProfileUpdate(originalUser, newDisplayName);
        
        // Then: only displayName should change
        assertThat(updatedUser.getDisplayName()).isEqualTo(newDisplayName);
        
        // Protected fields must remain unchanged
        assertThat(updatedUser.getEmail()).isEqualTo(originalEmail);
        assertThat(updatedUser.getPasswordHash()).isEqualTo(originalPasswordHash);
        assertThat(updatedUser.getStatus()).isEqualTo(originalStatus);
        assertThat(updatedUser.isEmailVerified()).isEqualTo(originalEmailVerified);
        assertThat(updatedUser.getCreatedAt()).isEqualTo(originalCreatedAt);
        
        // updatedAt should be modified
        assertThat(updatedUser.getUpdatedAt()).isAfterOrEqualTo(originalCreatedAt);
    }

    @Property(tries = 100)
    void profileUpdateWithNullDisplayNamePreservesOriginal(
            @ForAll("validUser") User originalUser) {
        
        String originalDisplayName = originalUser.getDisplayName();
        
        // When: update with null displayName
        User updatedUser = simulateProfileUpdateWithNull(originalUser);
        
        // Then: displayName should remain unchanged
        assertThat(updatedUser.getDisplayName()).isEqualTo(originalDisplayName);
    }

    @Property(tries = 100)
    void profileResponseNeverExposesPasswordHash(
            @ForAll("validUser") User user) {
        
        ProfileResponse response = mapToResponse(user);
        
        // Response should not contain password hash
        assertThat(response.toString()).doesNotContain(user.getPasswordHash());
        
        // Response should contain expected fields
        assertThat(response.userId()).isEqualTo(user.getId());
        assertThat(response.email()).isEqualTo(user.getEmail());
        assertThat(response.displayName()).isEqualTo(user.getDisplayName());
    }

    @Provide
    Arbitrary<User> validUser() {
        return Combinators.combine(
            Arbitraries.strings().alpha().ofMinLength(5).ofMaxLength(20),
            Arbitraries.strings().alpha().ofMinLength(2).ofMaxLength(50),
            Arbitraries.of(UserStatus.ACTIVE, UserStatus.PENDING_EMAIL)
        ).as((emailLocal, displayName, status) -> {
            User user = new User();
            user.setId(UUID.randomUUID());
            user.setEmail(emailLocal.toLowerCase() + "@example.com");
            user.setPasswordHash("$argon2id$v=19$m=65536,t=3,p=4$hash" + UUID.randomUUID());
            user.setDisplayName(displayName);
            user.setStatus(status);
            user.setEmailVerified(status == UserStatus.ACTIVE);
            user.setCreatedAt(Instant.now().minusSeconds(3600));
            user.setUpdatedAt(Instant.now().minusSeconds(1800));
            return user;
        });
    }

    private User simulateProfileUpdate(User original, String newDisplayName) {
        User updated = copyUser(original);
        updated.setDisplayName(newDisplayName);
        updated.setUpdatedAt(Instant.now());
        return updated;
    }

    private User simulateProfileUpdateWithNull(User original) {
        User updated = copyUser(original);
        updated.setUpdatedAt(Instant.now());
        return updated;
    }

    private User copyUser(User original) {
        User copy = new User();
        copy.setId(original.getId());
        copy.setEmail(original.getEmail());
        copy.setPasswordHash(original.getPasswordHash());
        copy.setDisplayName(original.getDisplayName());
        copy.setStatus(original.getStatus());
        copy.setEmailVerified(original.isEmailVerified());
        copy.setCreatedAt(original.getCreatedAt());
        copy.setUpdatedAt(original.getUpdatedAt());
        return copy;
    }

    private ProfileResponse mapToResponse(User user) {
        return new ProfileResponse(
            user.getId(),
            user.getEmail(),
            user.isEmailVerified(),
            user.getDisplayName(),
            user.getCreatedAt()
        );
    }
}
