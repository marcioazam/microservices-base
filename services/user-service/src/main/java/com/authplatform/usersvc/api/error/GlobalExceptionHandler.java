package com.authplatform.usersvc.api.error;

import com.authplatform.usersvc.shared.exception.*;
import com.authplatform.usersvc.shared.security.SecurityUtils;
import jakarta.servlet.http.HttpServletRequest;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.http.HttpHeaders;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.ExceptionHandler;
import org.springframework.web.bind.annotation.RestControllerAdvice;

import java.util.HashMap;
import java.util.Map;

/**
 * Global exception handler mapping exceptions to RFC 7807 Problem Detail responses.
 */
@RestControllerAdvice
public class GlobalExceptionHandler {

    private static final Logger log = LoggerFactory.getLogger(GlobalExceptionHandler.class);
    private static final String PROBLEM_TYPE_BASE = "https://api.auth-platform.com/problems/";

    private final SecurityUtils securityUtils;

    public GlobalExceptionHandler(SecurityUtils securityUtils) {
        this.securityUtils = securityUtils;
    }

    @ExceptionHandler(EmailExistsException.class)
    public ResponseEntity<ProblemDetail> handleEmailExists(EmailExistsException ex, HttpServletRequest request) {
        return buildResponse(ex, request, "Email address is already registered");
    }

    @ExceptionHandler(InvalidTokenException.class)
    public ResponseEntity<ProblemDetail> handleInvalidToken(InvalidTokenException ex, HttpServletRequest request) {
        return buildResponse(ex, request, "The verification token is invalid");
    }

    @ExceptionHandler(ExpiredTokenException.class)
    public ResponseEntity<ProblemDetail> handleExpiredToken(ExpiredTokenException ex, HttpServletRequest request) {
        return buildResponse(ex, request, "The verification token has expired");
    }

    @ExceptionHandler(AlreadyUsedException.class)
    public ResponseEntity<ProblemDetail> handleAlreadyUsed(AlreadyUsedException ex, HttpServletRequest request) {
        return buildResponse(ex, request, "The verification token has already been used");
    }

    @ExceptionHandler(UserNotFoundException.class)
    public ResponseEntity<ProblemDetail> handleUserNotFound(UserNotFoundException ex, HttpServletRequest request) {
        return buildResponse(ex, request, "The requested user was not found");
    }

    @ExceptionHandler(RateLimitedException.class)
    public ResponseEntity<ProblemDetail> handleRateLimited(RateLimitedException ex, HttpServletRequest request) {
        String correlationId = securityUtils.getCurrentCorrelationId();
        
        Map<String, Object> extensions = new HashMap<>();
        extensions.put("retryAfter", ex.getRetryAfterSeconds());
        
        ProblemDetail problem = ProblemDetail.of(
                PROBLEM_TYPE_BASE + "rate-limited",
                "Rate Limit Exceeded",
                ex.getHttpStatus(),
                "Too many requests. Please try again later.",
                request.getRequestURI(),
                correlationId,
                ex.getErrorCode(),
                extensions
        );
        
        log.warn("Rate limit exceeded: correlationId={}", correlationId);
        
        HttpHeaders headers = new HttpHeaders();
        headers.add("Retry-After", String.valueOf(ex.getRetryAfterSeconds()));
        
        return ResponseEntity.status(HttpStatus.TOO_MANY_REQUESTS)
                .headers(headers)
                .body(problem);
    }

    @ExceptionHandler(ValidationException.class)
    public ResponseEntity<ProblemDetail> handleValidation(ValidationException ex, HttpServletRequest request) {
        String correlationId = securityUtils.getCurrentCorrelationId();
        
        Map<String, Object> extensions = new HashMap<>();
        extensions.put("errors", ex.getErrors());
        
        ProblemDetail problem = ProblemDetail.of(
                PROBLEM_TYPE_BASE + "validation-error",
                "Validation Error",
                ex.getHttpStatus(),
                "One or more validation errors occurred",
                request.getRequestURI(),
                correlationId,
                ex.getErrorCode(),
                extensions
        );
        
        log.debug("Validation error: correlationId={}, errors={}", correlationId, ex.getErrors());
        
        return ResponseEntity.status(HttpStatus.BAD_REQUEST).body(problem);
    }

    @ExceptionHandler(Exception.class)
    public ResponseEntity<ProblemDetail> handleGeneric(Exception ex, HttpServletRequest request) {
        String correlationId = securityUtils.getCurrentCorrelationId();
        
        log.error("Unexpected error: correlationId={}", correlationId, ex);
        
        ProblemDetail problem = ProblemDetail.of(
                PROBLEM_TYPE_BASE + "internal-error",
                "Internal Server Error",
                500,
                "An unexpected error occurred",
                request.getRequestURI(),
                correlationId,
                "INTERNAL_ERROR"
        );
        
        return ResponseEntity.status(HttpStatus.INTERNAL_SERVER_ERROR).body(problem);
    }

    private ResponseEntity<ProblemDetail> buildResponse(UserServiceException ex, HttpServletRequest request, String detail) {
        String correlationId = securityUtils.getCurrentCorrelationId();
        
        ProblemDetail problem = ProblemDetail.of(
                PROBLEM_TYPE_BASE + ex.getErrorCode().toLowerCase().replace("_", "-"),
                toTitle(ex.getErrorCode()),
                ex.getHttpStatus(),
                detail,
                request.getRequestURI(),
                correlationId,
                ex.getErrorCode()
        );
        
        log.debug("Handled exception: type={}, correlationId={}", ex.getErrorCode(), correlationId);
        
        return ResponseEntity.status(ex.getHttpStatus()).body(problem);
    }

    private String toTitle(String errorCode) {
        return errorCode.replace("_", " ")
                .toLowerCase()
                .substring(0, 1).toUpperCase() + 
                errorCode.replace("_", " ").toLowerCase().substring(1);
    }
}
