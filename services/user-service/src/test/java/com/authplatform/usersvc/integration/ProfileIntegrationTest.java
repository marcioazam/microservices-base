package com.authplatform.usersvc.integration;

import com.authplatform.usersvc.api.dto.request.ProfileUpdateRequest;
import com.authplatform.usersvc.domain.model.User;
import com.authplatform.usersvc.domain.model.UserStatus;
import com.authplatform.usersvc.infra.persistence.UserRepository;
import com.fasterxml.jackson.databind.ObjectMapper;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.autoconfigure.web.servlet.AutoConfigureMockMvc;
import org.springframework.http.MediaType;
import org.springframework.security.test.web.servlet.request.SecurityMockMvcRequestPostProcessors;
import org.springframework.test.web.servlet.MockMvc;

import java.util.UUID;

import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.*;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.*;

@AutoConfigureMockMvc
class ProfileIntegrationTest extends BaseIntegrationTest {

    @Autowired
    private MockMvc mockMvc;

    @Autowired
    private ObjectMapper objectMapper;

    @Autowired
    private UserRepository userRepository;

    private UUID testUserId;

    @BeforeEach
    void setUp() {
        // Create test user
        var user = User.builder()
                .email("profile@example.com")
                .passwordHash("$argon2id$v=19$m=19456,t=2,p=1$test")
                .displayName("Test User")
                .emailVerified(true)
                .status(UserStatus.ACTIVE)
                .build();
        user = userRepository.save(user);
        testUserId = user.getId();
    }

    @Test
    void shouldGetProfileSuccessfully() throws Exception {
        mockMvc.perform(get("/api/v1/me")
                        .with(SecurityMockMvcRequestPostProcessors.jwt()
                                .jwt(jwt -> jwt.subject(testUserId.toString()))))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.id").value(testUserId.toString()))
                .andExpect(jsonPath("$.email").value("profile@example.com"))
                .andExpect(jsonPath("$.displayName").value("Test User"))
                .andExpect(jsonPath("$.emailVerified").value(true))
                .andExpect(jsonPath("$.status").value("ACTIVE"));
    }

    @Test
    void shouldReturn404ForNonExistentUser() throws Exception {
        UUID nonExistentId = UUID.randomUUID();

        mockMvc.perform(get("/api/v1/me")
                        .with(SecurityMockMvcRequestPostProcessors.jwt()
                                .jwt(jwt -> jwt.subject(nonExistentId.toString()))))
                .andExpect(status().isNotFound())
                .andExpect(jsonPath("$.errorCode").value("USER_NOT_FOUND"));
    }

    @Test
    void shouldUpdateDisplayNameSuccessfully() throws Exception {
        var request = new ProfileUpdateRequest("Updated Name");

        mockMvc.perform(patch("/api/v1/me")
                        .with(SecurityMockMvcRequestPostProcessors.jwt()
                                .jwt(jwt -> jwt.subject(testUserId.toString())))
                        .contentType(MediaType.APPLICATION_JSON)
                        .content(objectMapper.writeValueAsString(request)))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.displayName").value("Updated Name"));
    }

    @Test
    void shouldRejectInvalidDisplayName() throws Exception {
        var request = new ProfileUpdateRequest("A"); // Too short

        mockMvc.perform(patch("/api/v1/me")
                        .with(SecurityMockMvcRequestPostProcessors.jwt()
                                .jwt(jwt -> jwt.subject(testUserId.toString())))
                        .contentType(MediaType.APPLICATION_JSON)
                        .content(objectMapper.writeValueAsString(request)))
                .andExpect(status().isBadRequest())
                .andExpect(jsonPath("$.errorCode").value("VALIDATION_ERROR"));
    }

    @Test
    void shouldSanitizeHtmlInDisplayName() throws Exception {
        var request = new ProfileUpdateRequest("<script>alert('xss')</script>Safe Name");

        mockMvc.perform(patch("/api/v1/me")
                        .with(SecurityMockMvcRequestPostProcessors.jwt()
                                .jwt(jwt -> jwt.subject(testUserId.toString())))
                        .contentType(MediaType.APPLICATION_JSON)
                        .content(objectMapper.writeValueAsString(request)))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.displayName").value("Safe Name"));
    }

    @Test
    void shouldRequireAuthentication() throws Exception {
        mockMvc.perform(get("/api/v1/me"))
                .andExpect(status().isUnauthorized());
    }
}
