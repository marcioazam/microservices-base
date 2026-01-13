package com.authplatform.usersvc.integration;

import com.authplatform.usersvc.api.dto.request.EmailResendRequest;
import com.authplatform.usersvc.api.dto.request.EmailVerificationRequest;
import com.authplatform.usersvc.api.dto.request.UserRegistrationRequest;
import com.authplatform.usersvc.domain.model.EmailVerificationToken;
import com.authplatform.usersvc.infra.persistence.EmailVerificationTokenRepository;
import com.authplatform.usersvc.shared.crypto.TokenHasher;
import com.fasterxml.jackson.databind.ObjectMapper;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.autoconfigure.web.servlet.AutoConfigureMockMvc;
import org.springframework.http.MediaType;
import org.springframework.test.web.servlet.MockMvc;

import java.time.Instant;
import java.time.temporal.ChronoUnit;
import java.util.UUID;

import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.*;

@AutoConfigureMockMvc
class EmailVerificationIntegrationTest extends BaseIntegrationTest {

    @Autowired
    private MockMvc mockMvc;

    @Autowired
    private ObjectMapper objectMapper;

    @Autowired
    private EmailVerificationTokenRepository tokenRepository;

    @Autowired
    private TokenHasher tokenHasher;

    @Test
    void shouldVerifyEmailSuccessfully() throws Exception {
        // Register user first
        var regRequest = new UserRegistrationRequest(
                "verify@example.com", "Password123!", "Test User");
        
        mockMvc.perform(post("/api/v1/users")
                        .contentType(MediaType.APPLICATION_JSON)
                        .content(objectMapper.writeValueAsString(regRequest)))
                .andExpect(status().isCreated());

        // Get the token from database (in real scenario, sent via email)
        var tokens = tokenRepository.findAll();
        var token = tokens.stream()
                .filter(t -> !t.isUsed() && !t.isExpired())
                .findFirst()
                .orElseThrow();

        // Create raw token for verification (we need to reverse-engineer or use test helper)
        String rawToken = createTestToken(token.getUserId());

        var verifyRequest = new EmailVerificationRequest(rawToken);

        mockMvc.perform(post("/api/v1/users/verify")
                        .contentType(MediaType.APPLICATION_JSON)
                        .content(objectMapper.writeValueAsString(verifyRequest)))
                .andExpect(status().isOk());
    }

    @Test
    void shouldRejectInvalidToken() throws Exception {
        var request = new EmailVerificationRequest("invalid-token-that-does-not-exist-in-db");

        mockMvc.perform(post("/api/v1/users/verify")
                        .contentType(MediaType.APPLICATION_JSON)
                        .content(objectMapper.writeValueAsString(request)))
                .andExpect(status().isBadRequest())
                .andExpect(jsonPath("$.errorCode").value("INVALID_TOKEN"));
    }

    @Test
    void shouldRejectExpiredToken() throws Exception {
        // Create expired token directly in DB
        String rawToken = tokenHasher.generateToken();
        String tokenHash = tokenHasher.hash(rawToken);
        
        var expiredToken = EmailVerificationToken.builder()
                .userId(UUID.randomUUID())
                .tokenHash(tokenHash)
                .expiresAt(Instant.now().minus(1, ChronoUnit.HOURS))
                .attemptCount(0)
                .build();
        tokenRepository.save(expiredToken);

        var request = new EmailVerificationRequest(rawToken);

        mockMvc.perform(post("/api/v1/users/verify")
                        .contentType(MediaType.APPLICATION_JSON)
                        .content(objectMapper.writeValueAsString(request)))
                .andExpect(status().isBadRequest())
                .andExpect(jsonPath("$.errorCode").value("EXPIRED_TOKEN"));
    }

    @Test
    void shouldRejectAlreadyUsedToken() throws Exception {
        // Create already used token
        String rawToken = tokenHasher.generateToken();
        String tokenHash = tokenHasher.hash(rawToken);
        
        var usedToken = EmailVerificationToken.builder()
                .userId(UUID.randomUUID())
                .tokenHash(tokenHash)
                .expiresAt(Instant.now().plus(24, ChronoUnit.HOURS))
                .usedAt(Instant.now().minus(1, ChronoUnit.HOURS))
                .attemptCount(1)
                .build();
        tokenRepository.save(usedToken);

        var request = new EmailVerificationRequest(rawToken);

        mockMvc.perform(post("/api/v1/users/verify")
                        .contentType(MediaType.APPLICATION_JSON)
                        .content(objectMapper.writeValueAsString(request)))
                .andExpect(status().isBadRequest())
                .andExpect(jsonPath("$.errorCode").value("ALREADY_USED"));
    }

    @Test
    void shouldAcceptResendRequestForAnyEmail() throws Exception {
        // Should always return 202 to prevent email enumeration
        var request = new EmailResendRequest("nonexistent@example.com");

        mockMvc.perform(post("/api/v1/users/resend-verification")
                        .contentType(MediaType.APPLICATION_JSON)
                        .content(objectMapper.writeValueAsString(request)))
                .andExpect(status().isAccepted());
    }

    @Test
    void shouldAcceptResendForExistingUser() throws Exception {
        // Register user first
        var regRequest = new UserRegistrationRequest(
                "resend@example.com", "Password123!", "Test User");
        
        mockMvc.perform(post("/api/v1/users")
                        .contentType(MediaType.APPLICATION_JSON)
                        .content(objectMapper.writeValueAsString(regRequest)))
                .andExpect(status().isCreated());

        var resendRequest = new EmailResendRequest("resend@example.com");

        mockMvc.perform(post("/api/v1/users/resend-verification")
                        .contentType(MediaType.APPLICATION_JSON)
                        .content(objectMapper.writeValueAsString(resendRequest)))
                .andExpect(status().isAccepted());
    }

    private String createTestToken(UUID userId) {
        String rawToken = tokenHasher.generateToken();
        String tokenHash = tokenHasher.hash(rawToken);
        
        var token = EmailVerificationToken.builder()
                .userId(userId)
                .tokenHash(tokenHash)
                .expiresAt(Instant.now().plus(24, ChronoUnit.HOURS))
                .attemptCount(0)
                .build();
        tokenRepository.save(token);
        
        return rawToken;
    }
}
