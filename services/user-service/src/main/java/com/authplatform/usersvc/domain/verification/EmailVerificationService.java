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
import com.authplatform.usersvc.shared.exception.AlreadyUsedException;
import com.authplatform.usersvc.shared.exception.ExpiredTokenException;
import com.authplatform.usersvc.shared.exception.InvalidTokenException;
import com.authplatform.usersvc.shared.exception.UserNotFoundException;
import com.authplatform.usersvc.shared.security.SecurityUtils;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.util.Map;

/**
 * Service for email verification.
 */
@Service
public class EmailVerificationService {

    private final UserRepository userRepository;
    private final EmailVerificationTokenRepository tokenRepository;
    private final TokenHasher tokenHasher;
    private final RateLimitService rateLimitService;
    private final OutboxPublisher outboxPublisher;
    private final LoggingServiceClient loggingClient;
    private final SecurityUtils securityUtils;

    public EmailVerificationService(
            UserRepository userRepository,
            EmailVerificationTokenRepository tokenRepository,
            TokenHasher tokenHasher,
            RateLimitService rateLimitService,
            OutboxPublisher outboxPublisher,
            LoggingServiceClient loggingClient,
            SecurityUtils securityUtils) {
        this.userRepository = userRepository;
        this.tokenRepository = tokenRepository;
        this.tokenHasher = tokenHasher;
        this.rateLimitService = rateLimitService;
        this.outboxPublisher = outboxPublisher;
        this.loggingClient = loggingClient;
        this.securityUtils = securityUtils;
    }

    @Transactional
    public void verify(String token, String ipAddress) {
        // Check rate limit
        rateLimitService.checkVerifyLimit(ipAddress);
        
        // Hash token and lookup
        String tokenHash = tokenHasher.hash(token);
        EmailVerificationToken verificationToken = tokenRepository.findByTokenHash(tokenHash)
                .orElseThrow(InvalidTokenException::new);
        
        // Check if already used
        if (verificationToken.isUsed()) {
            throw new AlreadyUsedException();
        }
        
        // Check if expired
        if (verificationToken.isExpired()) {
            throw new ExpiredTokenException();
        }
        
        // Mark token as used
        verificationToken.markAsUsed();
        tokenRepository.save(verificationToken);
        
        // Update user status
        User user = userRepository.findById(verificationToken.getUserId())
                .orElseThrow(UserNotFoundException::new);
        user.setEmailVerified(true);
        user.setStatus(UserStatus.ACTIVE);
        userRepository.save(user);
        
        // Publish event
        outboxPublisher.publish("User", user.getId(), "UserEmailVerified",
                Map.of("userId", user.getId().toString(), "email", user.getEmail()));
        
        // Log audit event
        String correlationId = securityUtils.getCurrentCorrelationId();
        loggingClient.logAudit(AuditEvent.of(
                "EMAIL_VERIFIED", user.getId().toString(), correlationId,
                "Email verified successfully",
                Map.of("email", securityUtils.maskEmail(user.getEmail()))
        ));
    }
}
