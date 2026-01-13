namespace PasswordRecovery.Application.DTOs;

public record RecoveryRequest(string Email);

public record RecoveryRequestResponse(string Message, string CorrelationId);

public record TokenValidationRequest(string Token);

public record TokenValidationResponse(bool IsValid, string? ResetToken, string CorrelationId);

public record PasswordResetRequest(string Token, string NewPassword, string ConfirmPassword);

public record PasswordResetResponse(bool Success, string Message, string CorrelationId);

public record ErrorResponse(
    string Code,
    string Message,
    string CorrelationId,
    Dictionary<string, string[]>? ValidationErrors = null);
