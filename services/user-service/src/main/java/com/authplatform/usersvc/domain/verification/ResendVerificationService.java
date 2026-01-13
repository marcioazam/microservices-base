package com.authplatform.usersvc.domain.verification;

import com.authplatform.usersvc.domain.model.EmailVerificationToken;
import com.authplatform.usersvc.domain.model.User;
import com.authplatform.usersvc.domain.model.UserStatus;
import com.authplatform.usersvc.domain.ratelimit.RateLimitService;
import com.authplatform.usersvc.infra.persistence.EmailVerificationTokenRepository;
import com.authplatform.usersvc.infra.persistence.UserRepository;
import com.authplatform.usersvc.infrastructure.logging.AuditEvent;
import com.authplatform.usersvc.infrastructure.logging.LoggingServiceClient;
import com.authplatform.usersvc.infrastructure.outbox.OutboxPublisher;
import com.authplatform.usersvc.shared.crypto.TokenHasher;
import com.authplatform.usersvc.shared.security.SecurityUtils;
import com.authplatform.usersvc.shared.validation.ValidationService;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.time.Duration;
import java.time.Instant;
import java.util.Map;
import java.util.Optional;

/**
 * Service for resending verification emails.
 * Always returns success to prevent email enumeration.
 */
@Service
public class ResendVerificationService {

    private final UserRepository userRepository;
    private final EmailVerificationTokenRepository tokenRepository;
    private final TokenHasher tokenHasher;
    private final RateLimitService rateLimitService;
    private final OutboxPublisher outboxPublisher;
    private final LoggingServiceClient loggingClient;
    private final SecurityUtils securityUtils;
    private final ValidationService validationService;
    private final Duration tokenTtl;

    public ResendVerificationService(
            UserRepository userRepository,
            EmailVerificationTokenRepository tokenRepository,
            TokenHasher tokenHasher,
            RateLimitService rateLimitService,
            OutboxPublisher outboxPublisher,
            LoggingServiceClient loggingClient,
            SecurityUtils securityUtils,
            ValidationService validationService,
            @Value("${app.email-token.ttl-minutes:60}") int tokenTtlMinutes) {
        this.userRepository = userRepository;
        this.tokenRepository = tokenRepository;
        this.tokenHasher = tokenHasher;
        this.rateLimitService = rateLimitService;
        this.outboxPublisher = outboxPublisher;
        this.loggingClient = loggingClient;
        this.securityUtils = securityUtils;
        this.validationService = validationService;
        this.tokenTtl = Duration.ofMinutes(tokenTtlMinutes);
    }

    @Transactional
    public void resend(String email, String ipAddress) {
        String normalizedEmail = validationService.normalizeEmail(email);
        
        // Check rate limit (stricter: 3 per hour per email)
        rateLimitService.checkResendLimit(normalizedEmail, ipAddress);
        
        // Always log the attempt for audit
        String correlationId = securityUtils.getCurrentCorrelationId();
        
        // Find user - but don't reveal if exists
        Optional<User> userOpt = userRepository.findByEmail(normalizedEmail);
        
        if (userOpt.isPresent()) {
            User user = userOpt.get();
            
            // Only resend if user is pending email verification
            if (user.getStatus() == UserStatus.PENDING_EMAIL && !user.isEmailVerified()) {
                // Invalidate previous tokens
                tokenRepository.invalidateUnusedTokensForUser(user.getId());
                
                // Generate new token
                String token = tokenHasher.generateToken();
                String tokenHash = tokenHasher.hash(token);
                
                EmailVerificationToken verificationToken = EmailVerificationToken.builder()
                        .userId(user.getId())
                        .tokenHash(tokenHash)
                        .expiresAt(Instant.now().plus(tokenTtl))
                        .attemptCount(0)
                        .build();
                tokenRepository.save(verificationToken);
                
                // Publish event
                outboxPublisher.publish("User", user.getId(), "EmailVerificationRequested",
                        Map.of("userId", user.getId().toString(), "email", normalizedEmail, "token", token));
                
                loggingClient.logAudit(AuditEvent.of(
                        "VERIFICATION_RESENT", user.getId().toString(), correlationId,
                        "Verification email resent",
                        Map.of("email", securityUtils.maskEmail(normalizedEmail))
                ));
            }
        }
        
        // Always log the request (without revealing if user exists)
        loggingClient.logAudit(AuditEvent.of(
                "RESEND_REQUESTED", null, correlationId,
                "Resend verification requested",
                Map.of("maskedEmail", securityUtils.maskEmail(normalizedEmail))
        ));
    }
}
