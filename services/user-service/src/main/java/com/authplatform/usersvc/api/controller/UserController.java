package com.authplatform.usersvc.api.controller;

import com.authplatform.usersvc.api.dto.request.UserRegistrationRequest;
import com.authplatform.usersvc.api.dto.response.UserRegistrationResponse;
import com.authplatform.usersvc.domain.registration.RegistrationService;
import com.authplatform.usersvc.shared.security.SecurityUtils;
import io.swagger.v3.oas.annotations.Operation;
import io.swagger.v3.oas.annotations.responses.ApiResponse;
import io.swagger.v3.oas.annotations.tags.Tag;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.validation.Valid;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

@RestController
@RequestMapping("/api/v1/users")
@RequiredArgsConstructor
@Slf4j
@Tag(name = "User Registration", description = "User registration endpoints")
public class UserController {

    private final RegistrationService registrationService;
    private final SecurityUtils securityUtils;

    @PostMapping
    @Operation(summary = "Register a new user", description = "Creates a new user account with pending email verification")
    @ApiResponse(responseCode = "201", description = "User created successfully")
    @ApiResponse(responseCode = "400", description = "Invalid input")
    @ApiResponse(responseCode = "409", description = "Email already exists")
    @ApiResponse(responseCode = "429", description = "Rate limit exceeded")
    public ResponseEntity<UserRegistrationResponse> register(
            @Valid @RequestBody UserRegistrationRequest request,
            HttpServletRequest httpRequest) {
        
        String ipAddress = getClientIp(httpRequest);
        log.info("Processing registration request for email: {}", securityUtils.maskEmail(request.email()));
        
        var result = registrationService.register(
                request.email(),
                request.password(),
                request.displayName(),
                ipAddress
        );
        
        log.info("User registered successfully: userId={}", result.userId());
        
        return ResponseEntity.status(HttpStatus.CREATED)
                .body(new UserRegistrationResponse(result.userId(), result.status().name()));
    }

    private String getClientIp(HttpServletRequest request) {
        String xForwardedFor = request.getHeader("X-Forwarded-For");
        if (xForwardedFor != null && !xForwardedFor.isEmpty()) {
            return xForwardedFor.split(",")[0].trim();
        }
        return request.getRemoteAddr();
    }
}
