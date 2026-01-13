using Microsoft.AspNetCore.Mvc;
using PasswordRecovery.Application.DTOs;
using PasswordRecovery.Application.Services;

namespace PasswordRecovery.Api.Controllers;

[ApiController]
[Route("api/v1/password-recovery")]
public class PasswordRecoveryController : ControllerBase
{
    private readonly IRecoveryService _recoveryService;

    public PasswordRecoveryController(IRecoveryService recoveryService)
    {
        _recoveryService = recoveryService;
    }

    [HttpPost("request")]
    [ProducesResponseType(typeof(RecoveryRequestResponse), StatusCodes.Status200OK)]
    [ProducesResponseType(typeof(ErrorResponse), StatusCodes.Status400BadRequest)]
    [ProducesResponseType(typeof(ErrorResponse), StatusCodes.Status429TooManyRequests)]
    public async Task<IActionResult> RequestRecovery(
        [FromBody] RecoveryRequest request,
        CancellationToken ct)
    {
        var correlationId = GetCorrelationId();
        var ipAddress = GetClientIpAddress();

        var result = await _recoveryService.RequestRecoveryAsync(
            request.Email,
            ipAddress,
            correlationId,
            ct);

        return result.Match<IActionResult>(
            success => Ok(success),
            error => error.Contains("Too many") 
                ? StatusCode(429, new ErrorResponse("RATE_LIMIT_EXCEEDED", error, correlationId.ToString()))
                : BadRequest(new ErrorResponse("BAD_REQUEST", error, correlationId.ToString())));
    }

    [HttpPost("validate")]
    [ProducesResponseType(typeof(TokenValidationResponse), StatusCodes.Status200OK)]
    [ProducesResponseType(typeof(ErrorResponse), StatusCodes.Status400BadRequest)]
    public async Task<IActionResult> ValidateToken(
        [FromBody] TokenValidationRequest request,
        CancellationToken ct)
    {
        var correlationId = GetCorrelationId();

        var result = await _recoveryService.ValidateTokenAsync(
            request.Token,
            correlationId,
            ct);

        return result.Match<IActionResult>(
            success => Ok(success),
            error => error.Contains("Too many")
                ? StatusCode(429, new ErrorResponse("RATE_LIMIT_EXCEEDED", error, correlationId.ToString()))
                : BadRequest(new ErrorResponse("INVALID_TOKEN", error, correlationId.ToString())));
    }

    [HttpPost("reset")]
    [ProducesResponseType(typeof(PasswordResetResponse), StatusCodes.Status200OK)]
    [ProducesResponseType(typeof(ErrorResponse), StatusCodes.Status400BadRequest)]
    public async Task<IActionResult> ResetPassword(
        [FromBody] PasswordResetRequest request,
        CancellationToken ct)
    {
        var correlationId = GetCorrelationId();

        if (request.NewPassword != request.ConfirmPassword)
        {
            return BadRequest(new ErrorResponse(
                "PASSWORD_MISMATCH",
                "Passwords do not match.",
                correlationId.ToString()));
        }

        var result = await _recoveryService.ResetPasswordAsync(
            request.Token,
            request.NewPassword,
            correlationId,
            ct);

        return result.Match<IActionResult>(
            success => Ok(success),
            error =>
            {
                if (result.ValidationErrors.Count > 0)
                {
                    return BadRequest(new ErrorResponse(
                        "WEAK_PASSWORD",
                        error,
                        correlationId.ToString(),
                        new Dictionary<string, string[]> { ["password"] = result.ValidationErrors.ToArray() }));
                }
                return BadRequest(new ErrorResponse("INVALID_TOKEN", error, correlationId.ToString()));
            });
    }

    private Guid GetCorrelationId()
    {
        if (HttpContext.Items.TryGetValue("CorrelationId", out var id) && id is Guid correlationId)
            return correlationId;
        return Guid.NewGuid();
    }

    private string GetClientIpAddress()
    {
        var forwardedFor = HttpContext.Request.Headers["X-Forwarded-For"].FirstOrDefault();
        if (!string.IsNullOrEmpty(forwardedFor))
            return forwardedFor.Split(',')[0].Trim();
        
        return HttpContext.Connection.RemoteIpAddress?.ToString() ?? "unknown";
    }
}
