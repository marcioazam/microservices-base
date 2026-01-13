package com.authplatform.usersvc.api.controller;

import com.authplatform.usersvc.api.dto.request.ProfileUpdateRequest;
import com.authplatform.usersvc.api.dto.response.ProfileResponse;
import com.authplatform.usersvc.domain.profile.ProfileService;
import io.swagger.v3.oas.annotations.Operation;
import io.swagger.v3.oas.annotations.responses.ApiResponse;
import io.swagger.v3.oas.annotations.security.SecurityRequirement;
import io.swagger.v3.oas.annotations.tags.Tag;
import jakarta.validation.Valid;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.http.ResponseEntity;
import org.springframework.security.core.annotation.AuthenticationPrincipal;
import org.springframework.security.oauth2.jwt.Jwt;
import org.springframework.web.bind.annotation.*;

import java.util.UUID;

@RestController
@RequestMapping("/api/v1/me")
@RequiredArgsConstructor
@Slf4j
@Tag(name = "Profile", description = "User profile endpoints")
@SecurityRequirement(name = "bearer-jwt")
public class MeController {

    private final ProfileService profileService;

    @GetMapping
    @Operation(summary = "Get profile", description = "Returns authenticated user's profile")
    @ApiResponse(responseCode = "200", description = "Profile retrieved successfully")
    @ApiResponse(responseCode = "404", description = "User not found")
    public ResponseEntity<ProfileResponse> getProfile(@AuthenticationPrincipal Jwt jwt) {
        UUID userId = UUID.fromString(jwt.getSubject());
        log.info("Getting profile for user: {}", userId);
        
        var profile = profileService.getProfile(userId);
        
        return ResponseEntity.ok(new ProfileResponse(
                profile.id(),
                profile.email(),
                profile.displayName(),
                profile.emailVerified(),
                profile.status(),
                profile.createdAt(),
                profile.updatedAt()
        ));
    }

    @PatchMapping
    @Operation(summary = "Update profile", description = "Updates authenticated user's profile")
    @ApiResponse(responseCode = "200", description = "Profile updated successfully")
    @ApiResponse(responseCode = "400", description = "Invalid input")
    @ApiResponse(responseCode = "404", description = "User not found")
    public ResponseEntity<ProfileResponse> updateProfile(
            @AuthenticationPrincipal Jwt jwt,
            @Valid @RequestBody ProfileUpdateRequest request) {
        
        UUID userId = UUID.fromString(jwt.getSubject());
        log.info("Updating profile for user: {}", userId);
        
        var profile = profileService.updateDisplayName(userId, request.displayName());
        
        return ResponseEntity.ok(new ProfileResponse(
                profile.id(),
                profile.email(),
                profile.displayName(),
                profile.emailVerified(),
                profile.status(),
                profile.createdAt(),
                profile.updatedAt()
        ));
    }
}
