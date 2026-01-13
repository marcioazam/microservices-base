package com.authplatform.usersvc.domain.service;

public interface EmailVerificationService {
    void verify(String token);
    void resend(String email, String ipAddress);
}
