package com.authplatform.usersvc.domain.service.impl;

import com.authplatform.usersvc.common.errors.TokenException;
import com.authplatform.usersvc.common.util.EmailNormalizer;
import com.authplatform.usersvc.common.util.TokenHasher;
import com.authplatform.usersvc.domain.model.*;
import com.authplatform.usersvc.domain.service.EmailVerificationService;
import com.authplatform.usersvc.domain.service.RateLimitService;
import com.authplatform.usersvc.infra.outbox.OutboxPublisher;
import com.authplatform.usersvc.infra.persistence.*;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;
import java.time.Duration;
import java.time.Instant;

@Service
@RequiredArgsConstructor
@Slf4j
public class EmailVerificationServiceImpl implements EmailVerificationService {

    private final EmailVerificationTokenRepository tokenRepository;
    private final UserRepository userRepository;
    private final TokenHasher tokenHasher;
    private final EmailNormalizer emailNormalizer;
    private final OutboxPublisher outboxPublisher;
    private final RateLimitService rateLimitService;

    @Value("${app.email-token.ttl-minutes:60}")
    private int tokenTtlMinutes;

    @Value("${app.verification.base-url:http://localhost:3000/verify}")
    private String verificationBaseUrl;

    @Override
    @Transactional
    public void verify(String token) {
        String tokenHash = tokenHasher.hash(token);
        
        EmailVerificationToken verificationToken = tokenRepository.findByTokenHash(tokenHash)
                .orElseThrow(() -> new TokenException(TokenException.TokenErrorType.INVALID, "Invalid token"));

        if (verificationToken.isUsed()) {
            throw new TokenException(TokenException.TokenErrorType.ALREADY_USED, "Token already used");
        }

        if (verificationToken.isExpired()) {
            throw new TokenException(TokenException.TokenErrorType.EXPIRED, "Token expired");
        }

        verificationToken.markAsUsed();
        tokenRepository.save(verificationToken);

        User user = userRepository.findById(verificationToken.getUserId())
                .orElseThrow(() -> new TokenException(TokenException.TokenErrorType.INVALID, "User not found"));

        user.activate();
        userRepository.save(user);

        outboxPublisher.publishUserEmailVerified(user.getId(), user.getEmail());
        log.info("Email verified for user: userId={}", user.getId());
    }

    @Override
    @Transactional
    public void resend(String email, String ipAddress) {
        String normalizedEmail = emailNormalizer.normalize(email);
        
        rateLimitService.checkResendLimit(normalizedEmail, ipAddress);

        userRepository.findByEmail(normalizedEmail).ifPresent(user -> {
            if (!user.isEmailVerified()) {
                tokenRepository.invalidateUnusedTokensForUser(user.getId());

                String rawToken = tokenHasher.generateToken();
                String tokenHash = tokenHasher.hash(rawToken);

                EmailVerificationToken token = EmailVerificationToken.builder()
                        .userId(user.getId())
                        .tokenHash(tokenHash)
                        .expiresAt(Instant.now().plus(Duration.ofMinutes(tokenTtlMinutes)))
                        .build();

                tokenRepository.save(token);

                String verificationLink = verificationBaseUrl + "?token=" + rawToken;
                outboxPublisher.publishEmailVerificationRequested(user.getId(), normalizedEmail, verificationLink);
                log.info("Verification email resent for user: userId={}", user.getId());
            }
        });
    }
}
