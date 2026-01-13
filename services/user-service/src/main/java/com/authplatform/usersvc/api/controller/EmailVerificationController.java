package com.authplatform.usersvc.api.controller;

import com.authplatform.usersvc.api.dto.request.EmailResendRequest;
import com.authplatform.usersvc.api.dto.request.EmailVerificationRequest;
import com.authplatform.usersvc.domain.verification.EmailVerificationService;
import com.authplatform.usersvc.domain.verification.ResendVerificationService;
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
@Tag(name = "Email Verification", description = "Email verification endpoints")
public class EmailVerificationController {

    private final EmailVerificationService verificationService;
    private final ResendVerificationService resendService;

    @PostMapping("/verify")
    @Operation(summary = "Verify email", description = "Verifies user email with token")
    @ApiResponse(responseCode = "200", description = "Email verified successfully")
    @ApiResponse(responseCode = "400", description = "Invalid, expired, or already used token")
    @ApiResponse(responseCode = "429", description = "Rate limit exceeded")
    public ResponseEntity<Void> verify(
            @Valid @RequestBody EmailVerificationRequest request,
            HttpServletRequest httpRequest) {
        
        String ipAddress = getClientIp(httpRequest);
        log.info("Processing email verification request");
        
        verificationService.verify(request.token(), ipAddress);
        
        log.info("Email verified successfully");
        return ResponseEntity.ok().build();
    }

    @PostMapping("/resend-verification")
    @Operation(summary = "Resend verification email", description = "Resends verification email to user")
    @ApiResponse(responseCode = "202", description = "Request accepted")
    @ApiResponse(responseCode = "429", description = "Rate limit exceeded")
    public ResponseEntity<Void> resend(
            @Valid @RequestBody EmailResendRequest request,
            HttpServletRequest httpRequest) {
        
        String ipAddress = getClientIp(httpRequest);
        log.info("Processing resend verification request");
        
        resendService.resend(request.email(), ipAddress);
        
        // Always return 202 to prevent email enumeration
        return ResponseEntity.status(HttpStatus.ACCEPTED).build();
    }

    private String getClientIp(HttpServletRequest request) {
        String xForwardedFor = request.getHeader("X-Forwarded-For");
        if (xForwardedFor != null && !xForwardedFor.isEmpty()) {
            return xForwardedFor.split(",")[0].trim();
        }
        return request.getRemoteAddr();
    }
}
