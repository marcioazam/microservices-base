package com.authplatform.usersvc.api.dto.request;

import jakarta.validation.constraints.Size;

public record ProfileUpdateRequest(
    @Size(min = 2, max = 100, message = "Display name must be 2-100 characters")
    String displayName
) {}
