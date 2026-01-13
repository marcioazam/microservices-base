using PasswordRecovery.Application.Interfaces;

namespace PasswordRecovery.Api.Services;

public class TokenCleanupService : BackgroundService
{
    private readonly IServiceProvider _serviceProvider;
    private readonly ILogger<TokenCleanupService> _logger;
    private readonly TimeSpan _cleanupInterval;
    private readonly TimeSpan _retentionPeriod;

    public TokenCleanupService(
        IServiceProvider serviceProvider,
        ILogger<TokenCleanupService> logger,
        IConfiguration configuration)
    {
        _serviceProvider = serviceProvider;
        _logger = logger;
        _cleanupInterval = TimeSpan.FromMinutes(
            int.Parse(configuration["TokenCleanup:IntervalMinutes"] ?? "60"));
        _retentionPeriod = TimeSpan.FromDays(
            int.Parse(configuration["TokenCleanup:RetentionDays"] ?? "30"));
    }

    protected override async Task ExecuteAsync(CancellationToken stoppingToken)
    {
        _logger.LogInformation("Token cleanup service started. Interval: {Interval}, Retention: {Retention}",
            _cleanupInterval, _retentionPeriod);

        while (!stoppingToken.IsCancellationRequested)
        {
            try
            {
                await CleanupExpiredTokensAsync(stoppingToken);
            }
            catch (Exception ex)
            {
                _logger.LogError(ex, "Error during token cleanup");
            }

            await Task.Delay(_cleanupInterval, stoppingToken);
        }
    }

    private async Task CleanupExpiredTokensAsync(CancellationToken ct)
    {
        using var scope = _serviceProvider.CreateScope();
        var tokenRepository = scope.ServiceProvider.GetRequiredService<ITokenRepository>();

        var cutoffDate = DateTime.UtcNow.Subtract(_retentionPeriod);
        await tokenRepository.CleanupExpiredAsync(cutoffDate, ct);

        _logger.LogInformation("Cleaned up expired tokens older than {CutoffDate}", cutoffDate);
    }
}
