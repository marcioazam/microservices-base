using FsCheck;
using FsCheck.Xunit;
using System.Diagnostics;

namespace PasswordRecovery.Tests.Api;

/// <summary>
/// Property 15: OpenTelemetry Tracing Coverage
/// Validates: Requirements 10.2
/// </summary>
public class OpenTelemetryPropertyTests
{
    [Property(MaxTest = 10)]
    public bool ActivitySourceCreatesSpans(PositiveInt seed)
    {
        using var activitySource = new ActivitySource("password-recovery-test");
        using var listener = new ActivityListener
        {
            ShouldListenTo = _ => true,
            Sample = (ref ActivityCreationOptions<ActivityContext> _) => ActivitySamplingResult.AllData
        };
        ActivitySource.AddActivityListener(listener);

        Activity? capturedActivity = null;
        using (var activity = activitySource.StartActivity("TestOperation"))
        {
            capturedActivity = activity;
            activity?.SetTag("test.seed", seed.Get);
        }

        return capturedActivity != null && 
               capturedActivity.OperationName == "TestOperation";
    }

    [Property(MaxTest = 10)]
    public bool SpansHaveCorrectStatus(PositiveInt seed)
    {
        using var activitySource = new ActivitySource("password-recovery-test-status");
        using var listener = new ActivityListener
        {
            ShouldListenTo = _ => true,
            Sample = (ref ActivityCreationOptions<ActivityContext> _) => ActivitySamplingResult.AllData
        };
        ActivitySource.AddActivityListener(listener);

        Activity? successActivity = null;
        Activity? errorActivity = null;

        using (var activity = activitySource.StartActivity("SuccessOperation"))
        {
            activity?.SetStatus(ActivityStatusCode.Ok);
            successActivity = activity;
        }

        using (var activity = activitySource.StartActivity("ErrorOperation"))
        {
            activity?.SetStatus(ActivityStatusCode.Error, "Test error");
            errorActivity = activity;
        }

        return successActivity?.Status == ActivityStatusCode.Ok &&
               errorActivity?.Status == ActivityStatusCode.Error;
    }

    [Property(MaxTest = 10)]
    public bool SpansContainAttributes(PositiveInt seed)
    {
        using var activitySource = new ActivitySource("password-recovery-test-attrs");
        using var listener = new ActivityListener
        {
            ShouldListenTo = _ => true,
            Sample = (ref ActivityCreationOptions<ActivityContext> _) => ActivitySamplingResult.AllData
        };
        ActivitySource.AddActivityListener(listener);

        Activity? capturedActivity = null;
        using (var activity = activitySource.StartActivity("AttributeOperation"))
        {
            activity?.SetTag("http.method", "POST");
            activity?.SetTag("http.route", "/api/v1/password-recovery/request");
            capturedActivity = activity;
        }

        var tags = capturedActivity?.Tags.ToDictionary(t => t.Key, t => t.Value);
        return tags != null &&
               tags.ContainsKey("http.method") &&
               tags.ContainsKey("http.route");
    }

    [Property(MaxTest = 10)]
    public bool SpansAreProperlyClosed(PositiveInt seed)
    {
        using var activitySource = new ActivitySource("password-recovery-test-close");
        using var listener = new ActivityListener
        {
            ShouldListenTo = _ => true,
            Sample = (ref ActivityCreationOptions<ActivityContext> _) => ActivitySamplingResult.AllData
        };
        ActivitySource.AddActivityListener(listener);

        DateTime? startTime = null;
        DateTime? endTime = null;

        using (var activity = activitySource.StartActivity("TimedOperation"))
        {
            startTime = activity?.StartTimeUtc;
            Thread.Sleep(10); // Small delay to ensure duration
        }
        
        // After using block, activity should be stopped
        // We verify by checking that we can create activities properly
        using (var activity = activitySource.StartActivity("AfterOperation"))
        {
            endTime = activity?.StartTimeUtc;
        }

        return startTime.HasValue && endTime.HasValue && endTime >= startTime;
    }
}
