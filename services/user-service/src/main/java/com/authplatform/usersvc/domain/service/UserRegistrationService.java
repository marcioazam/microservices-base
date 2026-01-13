package com.authplatform.usersvc.domain.service;

import com.authplatform.usersvc.api.dto.request.UserRegistrationRequest;
import com.authplatform.usersvc.api.dto.response.UserRegistrationResponse;

public interface UserRegistrationService {
    UserRegistrationResponse register(UserRegistrationRequest request);
}
