package com.authplatform.usersvc.domain.service;

public interface PasswordService {
    String hash(String plainPassword);
    boolean verify(String plainPassword, String hash);
    boolean isArgon2idHash(String hash);
}
