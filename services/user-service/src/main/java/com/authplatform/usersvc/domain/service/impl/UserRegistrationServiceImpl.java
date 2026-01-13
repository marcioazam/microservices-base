package com.authplatform.usersvc.domain.service.impl;

import com.authplatform.usersvc.api.dto.request.UserRegistrationRequest;
import com.authplatform.usersvc.api.dto.response.UserRegistrationResponse;
import com.authplatform.usersvc.common.errors.EmailAlreadyExistsException;
import com.authplatform.usersvc.common.errors.ValidationException;
import com.authplatform.usersvc.common.util.*;
import com.authplatform.usersvc.domain.model.*;
import com.authplatform.usersvc.domain.service.PasswordService;
import com.authplatform.usersvc.domain.service.UserRegistrationService;
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
public class UserRegistrationServiceImpl implements UserRegistrationService {

    private final UserRepository userRepository;
    private final EmailVerificationTokenRepository tokenRepository;
    private final PasswordService passwordService;
    private final EmailNormalizer emailNormalizer;
    private final EmailValidator emailValidator;
    private final PasswordValidator passwordValidator;
    private final DisplayNameValidator displayNameValidator;
    private final TokenHasher tokenHasher;
    private final OutboxPublisher outboxPublisher;

    @Value("${app.email-token.ttl-minutes:60}")
    private int tokenTtlMinutes;

    @Value("${app.verification.base-url:http://localhost:3000/verify}")
    private String verificationBaseUrl;

    @Override
    @Transactional
    public UserRegistrationResponse register(UserRegistrationRequest request) {
        // Validate inputs
        validateInputs(request);

        // Normalize email
        String normalizedEmail = emailNormalizer.normalize(request.email());

        // Check for existing user
        if (userRepository.existsByEmail(normalizedEmail)) {
            log.warn("Registration attempt with existing email");
            throw new EmailAlreadyExistsException("Email already registered");
        }

        // Hash password
        String passwordHash = passwordService.hash(request.password());

        // Create user
        User user = User.builder()
                .email(normalizedEmail)
                .emailVerified(false)
                .passwordHash(passwordHash)
                .displayName(request.displayName().trim())
                .status(UserStatus.PENDING_EMAIL)
                .build();

        user = userRepository.save(user);
        log.info("User created: userId={}", user.getId());

        // Generate verification token
        String rawToken = tokenHasher.generateToken();
        String tokenHash = tokenHasher.hash(rawToken);

        EmailVerificationToken token = EmailVerificationToken.builder()
                .userId(user.getId())
                .tokenHash(tokenHash)
                .expiresAt(Instant.now().plus(Duration.ofMinutes(tokenTtlMinutes)))
                .build();

        tokenRepository.save(token);

        // Publish events
        outboxPublisher.publishUserRegistered(user.getId(), normalizedEmail, user.getDisplayName());
        
        String verificationLink = verificationBaseUrl + "?token=" + rawToken;
        outboxPublisher.publishEmailVerificationRequested(user.getId(), normalizedEmail, verificationLink);

        return new UserRegistrationResponse(
                user.getId(),
                user.getEmail(),
                user.getStatus().name()
        );
    }

    private void validateInputs(UserRegistrationRequest request) {
        var emailResult = emailValidator.validate(request.email());
        if (!emailResult.isValid()) {
            throw new ValidationException("email", emailResult.getFirstError());
        }

        var passwordResult = passwordValidator.validate(request.password());
        if (!passwordResult.isValid()) {
            throw new ValidationException("password", passwordResult.getFirstError());
        }

        var displayNameResult = displayNameValidator.validate(request.displayName());
        if (!displayNameResult.isValid()) {
            throw new ValidationException("displayName", displayNameResult.getFirstError());
        }
    }
}
