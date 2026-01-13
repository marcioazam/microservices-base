using System.Net;
using PasswordRecovery.Application.Interfaces;

namespace PasswordRecovery.Api.Middleware;

public class RateLimitingMiddleware
{
    private readonly RequestDelegate _next;
    private readonly ILogger<RateLimitingMiddleware> _logger;

    public RateLimitingMiddleware(RequestDelegate next, ILogger<RateLimitingMiddleware> logger)
    {
        _next = next;
        _logger = logger;
    }

    public async Task InvokeAsync(HttpContext context, IRateLimiter rateLimiter)
    {
        var ipAddress = GetClientIpAddress(context);
        var path = context.Request.Path.Value ?? "";

        if (path.StartsWith("/api/v1/password-recovery"))
        {
            var result = await rateLimiter.CheckAsync(
                $"global:{ipAddress}", 
                100, 
                TimeSpan.FromMinutes(1), 
                context.RequestAborted);

            if (!result.IsAllowed)
            {
                _logger.LogWarning("Rate limit exceeded for IP: {IpAddress}", ipAddress);
                
                context.Response.StatusCode = (int)HttpStatusCode.TooManyRequests;
                context.Response.Headers.RetryAfter = result.RetryAfter?.TotalSeconds.ToString("F0") ?? "60";
                
                await context.Response.WriteAsJsonAsync(new
                {
                    Code = "RATE_LIMIT_EXCEEDED",
                    Message = "Too many requests. Please try again later.",
                    CorrelationId = context.Items["CorrelationId"]?.ToString()
                });
                return;
            }

            await rateLimiter.IncrementAsync($"global:{ipAddress}", TimeSpan.FromMinutes(1), context.RequestAborted);
        }

        await _next(context);
    }

    private static string GetClientIpAddress(HttpContext context)
    {
        var forwardedFor = context.Request.Headers["X-Forwarded-For"].FirstOrDefault();
        if (!string.IsNullOrEmpty(forwardedFor))
        {
            return forwardedFor.Split(',')[0].Trim();
        }
        return context.Connection.RemoteIpAddress?.ToString() ?? "unknown";
    }
}

public static class RateLimitingMiddlewareExtensions
{
    public static IApplicationBuilder UseRateLimiting(this IApplicationBuilder builder)
    {
        return builder.UseMiddleware<RateLimitingMiddleware>();
    }
}
