package com.authplatform.usersvc.domain.service.impl;

import com.authplatform.usersvc.api.dto.request.ProfileUpdateRequest;
import com.authplatform.usersvc.api.dto.response.ProfileResponse;
import com.authplatform.usersvc.common.errors.UserNotFoundException;
import com.authplatform.usersvc.common.util.DisplayNameValidator;
import com.authplatform.usersvc.common.util.InputSanitizer;
import com.authplatform.usersvc.domain.model.User;
import com.authplatform.usersvc.domain.service.ProfileService;
import com.authplatform.usersvc.infra.persistence.UserRepository;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;
import java.time.Instant;
import java.util.UUID;

@Service
@RequiredArgsConstructor
@Slf4j
public class ProfileServiceImpl implements ProfileService {

    private final UserRepository userRepository;
    private final DisplayNameValidator displayNameValidator;
    private final InputSanitizer inputSanitizer;

    @Override
    @Transactional(readOnly = true)
    public ProfileResponse getProfile(UUID userId) {
        log.debug("Fetching profile for userId: {}", userId);
        User user = userRepository.findById(userId)
            .orElseThrow(() -> new UserNotFoundException("User not found"));
        return mapToResponse(user);
    }

    @Override
    @Transactional
    public ProfileResponse updateProfile(UUID userId, ProfileUpdateRequest request) {
        log.debug("Updating profile for userId: {}", userId);
        User user = userRepository.findById(userId)
            .orElseThrow(() -> new UserNotFoundException("User not found"));
        
        if (request.displayName() != null) {
            String sanitized = inputSanitizer.sanitize(request.displayName());
            displayNameValidator.validate(sanitized);
            user.setDisplayName(sanitized);
        }
        
        user.setUpdatedAt(Instant.now());
        User saved = userRepository.save(user);
        log.info("Profile updated for userId: {}", userId);
        return mapToResponse(saved);
    }

    private ProfileResponse mapToResponse(User user) {
        return new ProfileResponse(
            user.getId(),
            user.getEmail(),
            user.isEmailVerified(),
            user.getDisplayName(),
            user.getCreatedAt()
        );
    }
}
