package com.authplatform.usersvc.api.dto.response;

import java.util.UUID;

public record UserRegistrationResponse(
        UUID userId,
        String status
) {}
