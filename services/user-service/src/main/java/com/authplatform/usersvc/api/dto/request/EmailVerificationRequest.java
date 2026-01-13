package com.authplatform.usersvc.api.dto.request;

import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.Size;

public record EmailVerificationRequest(
    @NotBlank(message = "Token is required")
    @Size(min = 32, max = 128, message = "Invalid token format")
    String token
) {}
