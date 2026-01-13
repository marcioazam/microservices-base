package com.authplatform.usersvc.domain.service;

import com.authplatform.usersvc.api.dto.request.ProfileUpdateRequest;
import com.authplatform.usersvc.api.dto.response.ProfileResponse;
import java.util.UUID;

public interface ProfileService {
    ProfileResponse getProfile(UUID userId);
    ProfileResponse updateProfile(UUID userId, ProfileUpdateRequest request);
}
