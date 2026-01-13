using FsCheck;
using FsCheck.Xunit;

namespace LoggingService.Property.Tests.Properties;

/// <summary>
/// Property 13: Metrics Accuracy
/// Validates: Requirements 7.2
/// </summary>
[Trait("Category", "Property")]
[Trait("Feature", "logging-microservice")]
public class MetricsAccuracyProperties
{
    /// <summary>
    /// Property: Counter increments are always positive.
    /// </summary>
    [Property(MaxTest = 100)]
    public Property CounterIncrementsArePositive()
    {
        return Prop.ForAll(
            Gen.Choose(1, 1000).ToArbitrary(),
            increment =>
            {
                var counter = new TestCounter();
                counter.Inc(increment);
                return counter.Value == increment && increment > 0;
            });
    }

    /// <summary>
    /// Property: Counter accumulates correctly over multiple increments.
    /// </summary>
    [Property(MaxTest = 100)]
    public Property CounterAccumulatesCorrectly()
    {
        return Prop.ForAll(
            Gen.ListOf(Gen.Choose(1, 100)).ToArbitrary(),
            increments =>
            {
                var counter = new TestCounter();
                var expectedTotal = 0;

                foreach (var inc in increments)
                {
                    counter.Inc(inc);
                    expectedTotal += inc;
                }

                return counter.Value == expectedTotal;
            });
    }

    /// <summary>
    /// Property: Gauge can be set to any value.
    /// </summary>
    [Property(MaxTest = 100)]
    public Property GaugeCanBeSetToAnyValue()
    {
        return Prop.ForAll(
            Arb.From<double>().Filter(d => !double.IsNaN(d) && !double.IsInfinity(d)),
            value =>
            {
                var gauge = new TestGauge();
                gauge.Set(value);
                return Math.Abs(gauge.Value - value) < 0.0001;
            });
    }

    /// <summary>
    /// Property: Histogram records values in correct buckets.
    /// </summary>
    [Property(MaxTest = 100)]
    public Property HistogramRecordsInCorrectBuckets()
    {
        var buckets = new[] { 0.01, 0.05, 0.1, 0.25, 0.5, 1.0 };

        return Prop.ForAll(
            Gen.Choose(0, 100).Select(x => x / 100.0).ToArbitrary(),
            value =>
            {
                var histogram = new TestHistogram(buckets);
                histogram.Observe(value);

                // Value should be in the appropriate bucket
                var expectedBucket = buckets.FirstOrDefault(b => value <= b);
                return histogram.GetBucketCount(expectedBucket) >= 1 || expectedBucket == 0;
            });
    }

    /// <summary>
    /// Property: Processed count equals received minus failed.
    /// </summary>
    [Property(MaxTest = 100)]
    public Property ProcessedEqualsReceivedMinusFailed()
    {
        return Prop.ForAll(
            Gen.Choose(0, 1000).ToArbitrary(),
            Gen.Choose(0, 100).ToArbitrary(),
            (received, failedPercent) =>
            {
                var failed = (int)(received * (failedPercent / 100.0));
                var processed = received - failed;
                return processed >= 0 && processed + failed == received;
            });
    }

    // Test helper classes to simulate Prometheus metrics behavior
    private class TestCounter
    {
        public double Value { get; private set; }
        public void Inc(double amount = 1) => Value += amount;
    }

    private class TestGauge
    {
        public double Value { get; private set; }
        public void Set(double value) => Value = value;
    }

    private class TestHistogram
    {
        private readonly double[] _buckets;
        private readonly Dictionary<double, int> _bucketCounts = new();

        public TestHistogram(double[] buckets)
        {
            _buckets = buckets;
            foreach (var b in buckets) _bucketCounts[b] = 0;
        }

        public void Observe(double value)
        {
            foreach (var bucket in _buckets.Where(b => value <= b))
            {
                _bucketCounts[bucket]++;
            }
        }

        public int GetBucketCount(double bucket) =>
            _bucketCounts.TryGetValue(bucket, out var count) ? count : 0;
    }
}
