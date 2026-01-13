package com.authplatform.usersvc.api.dto.response;

import java.time.Instant;
import java.util.UUID;

public record ProfileResponse(
    UUID id,
    String email,
    String displayName,
    boolean emailVerified,
    String status,
    Instant createdAt,
    Instant updatedAt
) {}
