using System.Security.Cryptography;
using System.Text;
using System.Text.RegularExpressions;
using LoggingService.Core.Configuration;
using LoggingService.Core.Interfaces;
using LoggingService.Core.Models;
using Microsoft.Extensions.Options;

namespace LoggingService.Core.Services;

/// <summary>
/// Masks personally identifiable information (PII) in log entries.
/// </summary>
public sealed partial class PiiMasker : IPiiMasker
{
    private readonly SecurityOptions _options;

    // Compiled regex patterns for PII detection
    private static readonly Regex EmailPattern = EmailRegex();
    private static readonly Regex PhonePattern = PhoneRegex();
    private static readonly Regex IpPattern = IpRegex();
    private static readonly Regex SsnPattern = SsnRegex();

    public PiiMasker(IOptions<SecurityOptions> options)
    {
        _options = options.Value;
    }

    /// <summary>
    /// Constructor for testing without DI.
    /// </summary>
    public PiiMasker(SecurityOptions options)
    {
        _options = options;
    }

    /// <inheritdoc />
    public LogEntry MaskSensitiveData(LogEntry entry)
    {
        if (_options.MaskingMode == PiiMaskingMode.None)
        {
            return entry;
        }

        return entry with
        {
            Message = MaskString(entry.Message),
            Path = entry.Path != null ? MaskString(entry.Path) : null,
            UserId = entry.UserId != null ? MaskString(entry.UserId) : null,
            Metadata = entry.Metadata != null ? MaskMetadata(entry.Metadata) : null
        };
    }

    private string MaskString(string input)
    {
        if (string.IsNullOrEmpty(input))
        {
            return input;
        }

        var result = input;

        if (_options.PiiPatterns.Contains("email"))
        {
            result = EmailPattern.Replace(result, match => GetMaskedValue("email", match.Value));
        }

        if (_options.PiiPatterns.Contains("phone"))
        {
            result = PhonePattern.Replace(result, match => GetMaskedValue("phone", match.Value));
        }

        if (_options.PiiPatterns.Contains("ip"))
        {
            result = IpPattern.Replace(result, match => GetMaskedValue("ip", match.Value));
        }

        if (_options.PiiPatterns.Contains("ssn"))
        {
            result = SsnPattern.Replace(result, match => GetMaskedValue("ssn", match.Value));
        }

        return result;
    }

    private string GetMaskedValue(string type, string originalValue)
    {
        return _options.MaskingMode switch
        {
            PiiMaskingMode.Mask => $"[MASKED_{type.ToUpperInvariant()}]",
            PiiMaskingMode.Redact => "[REDACTED]",
            PiiMaskingMode.Hash => $"[HASH:{ComputeHash(originalValue)}]",
            _ => originalValue
        };
    }

    private static string ComputeHash(string value)
    {
        var bytes = SHA256.HashData(Encoding.UTF8.GetBytes(value));
        return Convert.ToHexString(bytes)[..16].ToLowerInvariant();
    }

    private Dictionary<string, object> MaskMetadata(Dictionary<string, object> metadata)
    {
        return metadata.ToDictionary(
            kvp => kvp.Key,
            kvp => kvp.Value is string s ? (object)MaskString(s) : kvp.Value);
    }

    // Regex patterns using source generators for better performance
    [GeneratedRegex(@"[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}", RegexOptions.Compiled)]
    private static partial Regex EmailRegex();

    [GeneratedRegex(@"\b\d{3}[-.]?\d{3}[-.]?\d{4}\b", RegexOptions.Compiled)]
    private static partial Regex PhoneRegex();

    [GeneratedRegex(@"\b(?:\d{1,3}\.){3}\d{1,3}\b", RegexOptions.Compiled)]
    private static partial Regex IpRegex();

    [GeneratedRegex(@"\b\d{3}-\d{2}-\d{4}\b", RegexOptions.Compiled)]
    private static partial Regex SsnRegex();
}
