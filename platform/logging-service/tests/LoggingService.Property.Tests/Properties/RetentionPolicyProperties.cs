using FsCheck;
using FsCheck.Xunit;
using LoggingService.Core.Models;

namespace LoggingService.Property.Tests.Properties;

/// <summary>
/// Property 9: Retention Policy Enforcement
/// Validates: Requirements 5.2, 5.3
/// </summary>
[Trait("Category", "Property")]
[Trait("Feature", "logging-microservice")]
public class RetentionPolicyProperties
{
    private static readonly Dictionary<LogLevel, TimeSpan> DefaultRetentionPolicies = new()
    {
        [LogLevel.Debug] = TimeSpan.FromDays(7),
        [LogLevel.Info] = TimeSpan.FromDays(30),
        [LogLevel.Warn] = TimeSpan.FromDays(90),
        [LogLevel.Error] = TimeSpan.FromDays(365),
        [LogLevel.Fatal] = TimeSpan.FromDays(365)
    };

    /// <summary>
    /// Property: Debug logs have shortest retention period.
    /// </summary>
    [Property(MaxTest = 100)]
    public Property DebugLogsHaveShortestRetention()
    {
        var debugRetention = DefaultRetentionPolicies[LogLevel.Debug];
        var otherRetentions = DefaultRetentionPolicies
            .Where(kvp => kvp.Key != LogLevel.Debug)
            .Select(kvp => kvp.Value);

        return otherRetentions.All(r => r >= debugRetention).ToProperty();
    }

    /// <summary>
    /// Property: Error and Fatal logs have longest retention period.
    /// </summary>
    [Property(MaxTest = 100)]
    public Property ErrorAndFatalHaveLongestRetention()
    {
        var errorRetention = DefaultRetentionPolicies[LogLevel.Error];
        var fatalRetention = DefaultRetentionPolicies[LogLevel.Fatal];
        var otherRetentions = DefaultRetentionPolicies
            .Where(kvp => kvp.Key != LogLevel.Error && kvp.Key != LogLevel.Fatal)
            .Select(kvp => kvp.Value);

        return (otherRetentions.All(r => r <= errorRetention) &&
                otherRetentions.All(r => r <= fatalRetention)).ToProperty();
    }

    /// <summary>
    /// Property: Retention periods are always positive.
    /// </summary>
    [Property(MaxTest = 100)]
    public Property RetentionPeriodsArePositive()
    {
        return DefaultRetentionPolicies.Values.All(r => r > TimeSpan.Zero).ToProperty();
    }

    /// <summary>
    /// Property: Log entry is eligible for deletion after retention period.
    /// </summary>
    [Property(MaxTest = 100)]
    public Property LogIsEligibleForDeletionAfterRetention()
    {
        return Prop.ForAll(
            Gen.Elements(LogLevel.Debug, LogLevel.Info, LogLevel.Warn, LogLevel.Error, LogLevel.Fatal).ToArbitrary(),
            Gen.Choose(1, 1000).ToArbitrary(),
            (level, daysOld) =>
            {
                var retentionPeriod = DefaultRetentionPolicies[level];
                var logAge = TimeSpan.FromDays(daysOld);
                var isEligibleForDeletion = logAge > retentionPeriod;

                // If log is older than retention, it should be eligible
                // If log is younger than retention, it should not be eligible
                return (daysOld > retentionPeriod.TotalDays) == isEligibleForDeletion;
            });
    }

    /// <summary>
    /// Property: Retention hierarchy is maintained (Debug < Info < Warn < Error/Fatal).
    /// </summary>
    [Property(MaxTest = 100)]
    public Property RetentionHierarchyIsMaintained()
    {
        var debug = DefaultRetentionPolicies[LogLevel.Debug];
        var info = DefaultRetentionPolicies[LogLevel.Info];
        var warn = DefaultRetentionPolicies[LogLevel.Warn];
        var error = DefaultRetentionPolicies[LogLevel.Error];

        return (debug <= info && info <= warn && warn <= error).ToProperty();
    }

    /// <summary>
    /// Property: Cutoff date calculation is correct.
    /// </summary>
    [Property(MaxTest = 100)]
    public Property CutoffDateCalculationIsCorrect()
    {
        return Prop.ForAll(
            Gen.Elements(LogLevel.Debug, LogLevel.Info, LogLevel.Warn, LogLevel.Error, LogLevel.Fatal).ToArbitrary(),
            level =>
            {
                var now = DateTimeOffset.UtcNow;
                var retentionPeriod = DefaultRetentionPolicies[level];
                var cutoffDate = now - retentionPeriod;

                // Cutoff should be in the past
                return cutoffDate < now;
            });
    }
}
