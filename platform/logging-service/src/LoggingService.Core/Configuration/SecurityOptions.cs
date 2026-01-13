namespace LoggingService.Core.Configuration;

/// <summary>
/// Security configuration options.
/// </summary>
public sealed record SecurityOptions
{
    /// <summary>
    /// Configuration section name.
    /// </summary>
    public const string SectionName = "LoggingService:Security";

    /// <summary>
    /// Whether to require TLS for all communications.
    /// </summary>
    public bool RequireTls { get; init; } = true;

    /// <summary>
    /// Whether to encrypt sensitive data at rest.
    /// </summary>
    public bool EncryptAtRest { get; init; } = true;

    /// <summary>
    /// PII patterns to detect and mask.
    /// </summary>
    public string[] PiiPatterns { get; init; } = ["email", "phone", "ip", "ssn"];

    /// <summary>
    /// Mode for masking PII data.
    /// </summary>
    public PiiMaskingMode MaskingMode { get; init; } = PiiMaskingMode.Redact;
}

/// <summary>
/// PII masking modes.
/// </summary>
public enum PiiMaskingMode
{
    /// <summary>
    /// No masking applied.
    /// </summary>
    None,

    /// <summary>
    /// Replace with type-specific placeholder (e.g., [MASKED_EMAIL]).
    /// </summary>
    Mask,

    /// <summary>
    /// Replace with generic [REDACTED] placeholder.
    /// </summary>
    Redact,

    /// <summary>
    /// Replace with hash of original value.
    /// </summary>
    Hash
}
