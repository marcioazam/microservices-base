package com.authplatform.usersvc.domain.profile;

import com.authplatform.usersvc.domain.model.User;
import com.authplatform.usersvc.infra.persistence.UserRepository;
import com.authplatform.usersvc.infrastructure.cache.CacheServiceClient;
import com.authplatform.usersvc.infrastructure.logging.AuditEvent;
import com.authplatform.usersvc.infrastructure.logging.LoggingServiceClient;
import com.authplatform.usersvc.shared.exception.UserNotFoundException;
import com.authplatform.usersvc.shared.exception.ValidationException;
import com.authplatform.usersvc.shared.security.SecurityUtils;
import com.authplatform.usersvc.shared.validation.ValidationService;
import com.fasterxml.jackson.databind.ObjectMapper;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.time.Duration;
import java.util.Map;
import java.util.Optional;
import java.util.UUID;

/**
 * Service for profile management with caching.
 */
@Service
public class ProfileService {

    private static final String CACHE_NAMESPACE = "user-service:profile";
    private static final Duration CACHE_TTL = Duration.ofMinutes(5);

    private final UserRepository userRepository;
    private final CacheServiceClient cacheClient;
    private final ValidationService validationService;
    private final LoggingServiceClient loggingClient;
    private final SecurityUtils securityUtils;
    private final ObjectMapper objectMapper;

    public ProfileService(
            UserRepository userRepository,
            CacheServiceClient cacheClient,
            ValidationService validationService,
            LoggingServiceClient loggingClient,
            SecurityUtils securityUtils,
            ObjectMapper objectMapper) {
        this.userRepository = userRepository;
        this.cacheClient = cacheClient;
        this.validationService = validationService;
        this.loggingClient = loggingClient;
        this.securityUtils = securityUtils;
        this.objectMapper = objectMapper;
    }

    public ProfileData getProfile(UUID userId) {
        // Check cache first
        String cacheKey = userId.toString();
        Optional<byte[]> cached = cacheClient.get(CACHE_NAMESPACE, cacheKey);
        
        if (cached.isPresent()) {
            try {
                return objectMapper.readValue(cached.get(), ProfileData.class);
            } catch (Exception e) {
                // Cache miss on deserialization error
            }
        }
        
        // Load from database
        User user = userRepository.findById(userId)
                .orElseThrow(UserNotFoundException::new);
        
        ProfileData profile = new ProfileData(
                user.getId(),
                user.getEmail(),
                user.getDisplayName(),
                user.isEmailVerified(),
                user.getStatus().name(),
                user.getCreatedAt(),
                user.getUpdatedAt()
        );
        
        // Cache result
        try {
            byte[] data = objectMapper.writeValueAsBytes(profile);
            cacheClient.set(CACHE_NAMESPACE, cacheKey, data, CACHE_TTL);
        } catch (Exception e) {
            // Log but don't fail on cache error
        }
        
        return profile;
    }

    @Transactional
    public ProfileData updateDisplayName(UUID userId, String displayName) {
        // Validate input
        var validation = validationService.validateDisplayName(displayName);
        if (!validation.valid()) {
            throw new ValidationException(validation.errors());
        }
        
        // Load user
        User user = userRepository.findById(userId)
                .orElseThrow(UserNotFoundException::new);
        
        // Sanitize and update
        String sanitized = validationService.sanitizeDisplayName(displayName);
        user.setDisplayName(sanitized);
        user = userRepository.save(user);
        
        // Invalidate cache
        cacheClient.delete(CACHE_NAMESPACE, userId.toString());
        
        // Log audit event
        String correlationId = securityUtils.getCurrentCorrelationId();
        loggingClient.logAudit(AuditEvent.of(
                "PROFILE_UPDATED", userId.toString(), correlationId,
                "Display name updated",
                Map.of("email", securityUtils.maskEmail(user.getEmail()))
        ));
        
        return new ProfileData(
                user.getId(),
                user.getEmail(),
                user.getDisplayName(),
                user.isEmailVerified(),
                user.getStatus().name(),
                user.getCreatedAt(),
                user.getUpdatedAt()
        );
    }

    public record ProfileData(
            UUID id,
            String email,
            String displayName,
            boolean emailVerified,
            String status,
            java.time.Instant createdAt,
            java.time.Instant updatedAt
    ) {}
}
