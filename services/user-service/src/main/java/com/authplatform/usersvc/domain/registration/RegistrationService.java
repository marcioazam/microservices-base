package com.authplatform.usersvc.domain.registration;

import com.authplatform.usersvc.domain.model.EmailVerificationToken;
import com.authplatform.usersvc.domain.model.User;
import com.authplatform.usersvc.domain.model.UserStatus;
import com.authplatform.usersvc.domain.ratelimit.RateLimitService;
import com.authplatform.usersvc.infra.persistence.EmailVerificationTokenRepository;
import com.authplatform.usersvc.infra.persistence.UserRepository;
import com.authplatform.usersvc.infrastructure.logging.AuditEvent;
import com.authplatform.usersvc.infrastructure.logging.LoggingServiceClient;
import com.authplatform.usersvc.infrastructure.outbox.OutboxPublisher;
import com.authplatform.usersvc.shared.crypto.PasswordService;
import com.authplatform.usersvc.shared.crypto.TokenHasher;
import com.authplatform.usersvc.shared.exception.EmailExistsException;
import com.authplatform.usersvc.shared.exception.ValidationException;
import com.authplatform.usersvc.shared.security.SecurityUtils;
import com.authplatform.usersvc.shared.validation.ValidationService;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.time.Duration;
import java.time.Instant;
import java.util.Map;
import java.util.UUID;

/**
 * Service for user registration with email verification.
 */
@Service
public class RegistrationService {

    private final UserRepository userRepository;
    private final EmailVerificationTokenRepository tokenRepository;
    private final ValidationService validationService;
    private final PasswordService passwordService;
    private final TokenHasher tokenHasher;
    private final RateLimitService rateLimitService;
    private final OutboxPublisher outboxPublisher;
    private final LoggingServiceClient loggingClient;
    private final SecurityUtils securityUtils;
    private final Duration tokenTtl;

    public RegistrationService(
            UserRepository userRepository,
            EmailVerificationTokenRepository tokenRepository,
            ValidationService validationService,
            PasswordService passwordService,
            TokenHasher tokenHasher,
            RateLimitService rateLimitService,
            OutboxPublisher outboxPublisher,
            LoggingServiceClient loggingClient,
            SecurityUtils securityUtils,
            @Value("${app.email-token.ttl-minutes:60}") int tokenTtlMinutes) {
        this.userRepository = userRepository;
        this.tokenRepository = tokenRepository;
        this.validationService = validationService;
        this.passwordService = passwordService;
        this.tokenHasher = tokenHasher;
        this.rateLimitService = rateLimitService;
        this.outboxPublisher = outboxPublisher;
        this.loggingClient = loggingClient;
        this.securityUtils = securityUtils;
        this.tokenTtl = Duration.ofMinutes(tokenTtlMinutes);
    }

    @Transactional
    public RegistrationResult register(String email, String password, String displayName, String ipAddress) {
        // Check rate limit
        rateLimitService.checkRegistrationLimit(ipAddress);
        
        // Validate input
        var validation = validationService.validateRegistration(email, password, displayName);
        if (!validation.valid()) {
            throw new ValidationException(validation.errors());
        }
        
        // Normalize email
        String normalizedEmail = validationService.normalizeEmail(email);
        
        // Check email uniqueness
        if (userRepository.existsByEmail(normalizedEmail)) {
            throw new EmailExistsException();
        }
        
        // Hash password
        String passwordHash = passwordService.hash(password);
        
        // Sanitize display name
        String sanitizedDisplayName = validationService.sanitizeDisplayName(displayName);
        
        // Create user
        User user = User.builder()
                .email(normalizedEmail)
                .passwordHash(passwordHash)
                .displayName(sanitizedDisplayName)
                .status(UserStatus.PENDING_EMAIL)
                .emailVerified(false)
                .build();
        user = userRepository.save(user);
        
        // Generate verification token
        String token = tokenHasher.generateToken();
        String tokenHash = tokenHasher.hash(token);
        
        EmailVerificationToken verificationToken = EmailVerificationToken.builder()
                .userId(user.getId())
                .tokenHash(tokenHash)
                .expiresAt(Instant.now().plus(tokenTtl))
                .attemptCount(0)
                .build();
        tokenRepository.save(verificationToken);
        
        // Publish events
        outboxPublisher.publish("User", user.getId(), "UserRegistered", 
                Map.of("userId", user.getId().toString(), "email", normalizedEmail));
        outboxPublisher.publish("User", user.getId(), "EmailVerificationRequested",
                Map.of("userId", user.getId().toString(), "email", normalizedEmail, "token", token));
        
        // Log audit event
        String correlationId = securityUtils.getCurrentCorrelationId();
        loggingClient.logAudit(AuditEvent.of(
                "USER_REGISTERED", user.getId().toString(), correlationId,
                "User registered successfully",
                Map.of("email", securityUtils.maskEmail(normalizedEmail))
        ));
        
        return new RegistrationResult(user.getId(), user.getStatus());
    }

    public record RegistrationResult(UUID userId, UserStatus status) {}
}
