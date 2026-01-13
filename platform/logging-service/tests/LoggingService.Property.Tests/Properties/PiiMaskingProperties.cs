using FsCheck;
using FsCheck.Xunit;
using LoggingService.Core.Configuration;
using LoggingService.Core.Models;
using LoggingService.Core.Services;
using LoggingService.Property.Tests.Generators;
using Shouldly;

namespace LoggingService.Property.Tests.Properties;

/// <summary>
/// Property tests for PII masking.
/// Feature: logging-service-modernization, Property 3: PII Masking Completeness
/// Validates: Requirements 8.1
/// </summary>
public class PiiMaskingProperties
{
    private readonly PiiMasker _maskerRedact;
    private readonly PiiMasker _maskerMask;
    private readonly PiiMasker _maskerHash;
    private readonly PiiMasker _maskerNone;

    public PiiMaskingProperties()
    {
        _maskerRedact = new PiiMasker(new SecurityOptions
        {
            MaskingMode = PiiMaskingMode.Redact,
            PiiPatterns = ["email", "phone", "ip", "ssn"]
        });

        _maskerMask = new PiiMasker(new SecurityOptions
        {
            MaskingMode = PiiMaskingMode.Mask,
            PiiPatterns = ["email", "phone", "ip", "ssn"]
        });

        _maskerHash = new PiiMasker(new SecurityOptions
        {
            MaskingMode = PiiMaskingMode.Hash,
            PiiPatterns = ["email", "phone", "ip", "ssn"]
        });

        _maskerNone = new PiiMasker(new SecurityOptions
        {
            MaskingMode = PiiMaskingMode.None,
            PiiPatterns = ["email", "phone", "ip", "ssn"]
        });
    }

    /// <summary>
    /// Property 3: PII Masking Completeness
    /// For any log entry containing PII patterns (email addresses, phone numbers, IP addresses),
    /// after processing by the PiiMasker, those patterns SHALL be masked/redacted.
    /// </summary>
    [Fact]
    [Trait("Category", "Property")]
    [Trait("Feature", "logging-service-modernization")]
    public void EmailAddresses_AreMasked()
    {
        var entry = CreateEntryWithMessage("User [email] logged in from 192.168.1.1");

        var masked = _maskerRedact.MaskSensitiveData(entry);

        masked.Message.ShouldNotContain("[email]");
        masked.Message.ShouldContain("[REDACTED]");
    }

    /// <summary>
    /// Property: Phone numbers are masked.
    /// </summary>
    [Fact]
    [Trait("Category", "Property")]
    [Trait("Feature", "logging-service-modernization")]
    public void PhoneNumbers_AreMasked()
    {
        var entry = CreateEntryWithMessage("Contact phone: 555-123-4567");

        var masked = _maskerRedact.MaskSensitiveData(entry);

        masked.Message.ShouldNotContain("555-123-4567");
        masked.Message.ShouldContain("[REDACTED]");
    }

    /// <summary>
    /// Property: IP addresses are masked.
    /// </summary>
    [Fact]
    [Trait("Category", "Property")]
    [Trait("Feature", "logging-service-modernization")]
    public void IpAddresses_AreMasked()
    {
        var entry = CreateEntryWithMessage("Request from IP: 192.168.1.100");

        var masked = _maskerRedact.MaskSensitiveData(entry);

        masked.Message.ShouldNotContain("192.168.1.100");
        masked.Message.ShouldContain("[REDACTED]");
    }

    /// <summary>
    /// Property: SSN numbers are masked.
    /// </summary>
    [Fact]
    [Trait("Category", "Property")]
    [Trait("Feature", "logging-service-modernization")]
    public void SsnNumbers_AreMasked()
    {
        var entry = CreateEntryWithMessage("SSN: 123-45-6789");

        var masked = _maskerRedact.MaskSensitiveData(entry);

        masked.Message.ShouldNotContain("123-45-6789");
        masked.Message.ShouldContain("[REDACTED]");
    }

    /// <summary>
    /// Property: Mask mode uses type-specific placeholders.
    /// </summary>
    [Fact]
    [Trait("Category", "Property")]
    [Trait("Feature", "logging-service-modernization")]
    public void MaskMode_UsesTypeSpecificPlaceholders()
    {
        var entry = CreateEntryWithMessage("Email: [email], Phone: 555-123-4567, IP: 10.0.0.1");

        var masked = _maskerMask.MaskSensitiveData(entry);

        masked.Message.ShouldContain("[MASKED_EMAIL]");
        masked.Message.ShouldContain("[MASKED_PHONE]");
        masked.Message.ShouldContain("[MASKED_IP]");
    }

    /// <summary>
    /// Property: Hash mode produces consistent hashes.
    /// </summary>
    [Fact]
    [Trait("Category", "Property")]
    [Trait("Feature", "logging-service-modernization")]
    public void HashMode_ProducesConsistentHashes()
    {
        var entry1 = CreateEntryWithMessage("Email: [email]");
        var entry2 = CreateEntryWithMessage("Email: [email]");

        var masked1 = _maskerHash.MaskSensitiveData(entry1);
        var masked2 = _maskerHash.MaskSensitiveData(entry2);

        // Same input should produce same hash
        masked1.Message.ShouldBe(masked2.Message);
        masked1.Message.ShouldContain("[HASH:");
    }

    /// <summary>
    /// Property: None mode preserves original values.
    /// </summary>
    [Fact]
    [Trait("Category", "Property")]
    [Trait("Feature", "logging-service-modernization")]
    public void NoneMode_PreservesOriginalValues()
    {
        var originalMessage = "Email: [email], Phone: 555-123-4567";
        var entry = CreateEntryWithMessage(originalMessage);

        var masked = _maskerNone.MaskSensitiveData(entry);

        masked.Message.ShouldBe(originalMessage);
    }

    /// <summary>
    /// Property: Multiple PII patterns in same message are all masked.
    /// </summary>
    [Fact]
    [Trait("Category", "Property")]
    [Trait("Feature", "logging-service-modernization")]
    public void MultiplePiiPatterns_AreAllMasked()
    {
        var entry = CreateEntryWithMessage(
            "User [email] from 192.168.1.1 called 555-123-4567 with SSN 123-45-6789");

        var masked = _maskerRedact.MaskSensitiveData(entry);

        masked.Message.ShouldNotContain("[email]");
        masked.Message.ShouldNotContain("192.168.1.1");
        masked.Message.ShouldNotContain("555-123-4567");
        masked.Message.ShouldNotContain("123-45-6789");

        // Should have 4 redacted placeholders
        var redactedCount = masked.Message.Split("[REDACTED]").Length - 1;
        redactedCount.ShouldBe(4);
    }

    /// <summary>
    /// Property: PII in metadata is also masked.
    /// </summary>
    [Fact]
    [Trait("Category", "Property")]
    [Trait("Feature", "logging-service-modernization")]
    public void PiiInMetadata_IsMasked()
    {
        var entry = new LogEntry
        {
            Id = "test-id",
            Timestamp = DateTimeOffset.UtcNow,
            CorrelationId = Guid.NewGuid().ToString(),
            ServiceId = "test-service",
            Level = LogLevel.Info,
            Message = "Test message",
            Metadata = new Dictionary<string, object>
            {
                ["userEmail"] = "[email]",
                ["clientIp"] = "10.0.0.1",
                ["count"] = 42 // Non-string should be preserved
            }
        };

        var masked = _maskerRedact.MaskSensitiveData(entry);

        masked.Metadata.ShouldNotBeNull();
        masked.Metadata!["userEmail"].ShouldBe("[REDACTED]");
        masked.Metadata["clientIp"].ShouldBe("[REDACTED]");
        masked.Metadata["count"].ShouldBe(42);
    }

    /// <summary>
    /// Property: Path field is masked.
    /// </summary>
    [Fact]
    [Trait("Category", "Property")]
    [Trait("Feature", "logging-service-modernization")]
    public void PathField_IsMasked()
    {
        var entry = new LogEntry
        {
            Id = "test-id",
            Timestamp = DateTimeOffset.UtcNow,
            CorrelationId = Guid.NewGuid().ToString(),
            ServiceId = "test-service",
            Level = LogLevel.Info,
            Message = "Test message",
            Path = "/api/users/[email]/profile"
        };

        var masked = _maskerRedact.MaskSensitiveData(entry);

        masked.Path.ShouldNotContain("[email]");
        masked.Path.ShouldContain("[REDACTED]");
    }

    /// <summary>
    /// Property: Original PII values are not recoverable from masked output.
    /// </summary>
    [Property(MaxTest = 100)]
    [Trait("Category", "Property")]
    [Trait("Feature", "logging-service-modernization")]
    public Property OriginalPii_NotRecoverableFromMaskedOutput()
    {
        return Prop.ForAll(
            LogEntryGenerator.Generate(),
            entry =>
            {
                // Add known PII to the entry
                var piiEmail = "[email]";
                var piiPhone = "555-999-8888";
                var piiIp = "172.16.0.100";

                var entryWithPii = entry with
                {
                    Message = $"User {piiEmail} from {piiIp} called {piiPhone}"
                };

                var masked = _maskerRedact.MaskSensitiveData(entryWithPii);

                // Original values should not be present
                masked.Message.ShouldNotContain(piiEmail);
                masked.Message.ShouldNotContain(piiPhone);
                masked.Message.ShouldNotContain(piiIp);

                return true;
            });
    }

    private static LogEntry CreateEntryWithMessage(string message)
    {
        return new LogEntry
        {
            Id = "test-id",
            Timestamp = DateTimeOffset.UtcNow,
            CorrelationId = Guid.NewGuid().ToString(),
            ServiceId = "test-service",
            Level = LogLevel.Info,
            Message = message
        };
    }
}
