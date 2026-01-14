import { AppError } from '@domain/errors';
import dns from 'dns';
import { promisify } from 'util';

const dnsLookup = promisify(dns.lookup);

/**
 * SSRF Protection - URL Validator
 *
 * Protects against Server-Side Request Forgery attacks by:
 * 1. Allowlisting only HTTP/HTTPS protocols
 * 2. Blocking private/internal IP ranges (RFC 1918, RFC 4193, RFC 3927)
 * 3. Blocking cloud metadata endpoints (AWS, GCP, Azure)
 * 4. Blocking localhost and link-local addresses
 * 5. Validating resolved DNS addresses to prevent DNS rebinding
 * 6. Preventing redirect-based SSRF bypasses
 */
export class UrlValidator {
  private static readonly PRIVATE_IP_PATTERNS = [
    // IPv4 Loopback
    /^127\./,
    // IPv4 Private Class A (10.0.0.0/8)
    /^10\./,
    // IPv4 Private Class B (172.16.0.0/12)
    /^172\.(1[6-9]|2[0-9]|3[01])\./,
    // IPv4 Private Class C (192.168.0.0/16)
    /^192\.168\./,
    // IPv4 Link-Local (169.254.0.0/16)
    /^169\.254\./,
    // IPv4 Broadcast
    /^255\.255\.255\.255$/,
    // IPv6 Loopback
    /^::1$/,
    /^0:0:0:0:0:0:0:1$/,
    // IPv6 Private/Unique Local (fc00::/7)
    /^fc00:/i,
    /^fd00:/i,
    // IPv6 Link-Local (fe80::/10)
    /^fe80:/i,
    // IPv6 Multicast (ff00::/8)
    /^ff00:/i,
  ];

  private static readonly BLOCKED_HOSTNAMES = [
    // Localhost
    'localhost',
    'localhost.localdomain',
    // Internal domains
    '*.local',
    '*.localhost',
    '*.internal',
    '*.corp',
    '*.intranet',
    // Cloud metadata endpoints
    'metadata.google.internal',
    'metadata',
    'instance-data',
  ];

  private static readonly CLOUD_METADATA_IPS = [
    '169.254.169.254', // AWS, Azure, GCP, Oracle Cloud
    '169.254.170.2',   // ECS Task metadata endpoint
    '100.100.100.200', // Alibaba Cloud
    'fd00:ec2::254',   // AWS IPv6 metadata
  ];

  private static readonly ALLOWED_PROTOCOLS = ['http:', 'https:'];

  private static readonly MAX_REDIRECTS = 0; // No redirects allowed

  /**
   * Validates a URL for SSRF safety before making HTTP requests
   * @param urlString - The URL to validate
   * @throws AppError if URL is unsafe
   */
  static async validate(urlString: string): Promise<URL> {
    // Parse URL
    let parsedUrl: URL;
    try {
      parsedUrl = new URL(urlString);
    } catch (error) {
      throw AppError.invalidImage('Invalid URL format');
    }

    // Validate protocol
    if (!this.ALLOWED_PROTOCOLS.includes(parsedUrl.protocol)) {
      throw AppError.invalidImage(
        `Invalid protocol: ${parsedUrl.protocol}. Only HTTP and HTTPS are allowed`
      );
    }

    // Validate hostname exists
    if (!parsedUrl.hostname) {
      throw AppError.invalidImage('URL must have a hostname');
    }

    // Check for username/password in URL (potential credential leak)
    if (parsedUrl.username || parsedUrl.password) {
      throw AppError.invalidImage('URLs with embedded credentials are not allowed');
    }

    // Validate hostname is not blocked
    this.validateHostname(parsedUrl.hostname);

    // Check if hostname is an IP address
    const isIpAddress = this.isIpAddress(parsedUrl.hostname);

    if (isIpAddress) {
      // Direct IP address - validate it's not private
      this.validateIpAddress(parsedUrl.hostname);
    } else {
      // Domain name - resolve DNS and validate IPs
      await this.validateDnsResolution(parsedUrl.hostname);
    }

    return parsedUrl;
  }

  /**
   * Validates hostname against blocked patterns
   */
  private static validateHostname(hostname: string): void {
    const lowerHostname = hostname.toLowerCase();

    for (const blocked of this.BLOCKED_HOSTNAMES) {
      if (blocked.startsWith('*.')) {
        // Wildcard pattern
        const suffix = blocked.substring(1); // Remove *
        if (lowerHostname.endsWith(suffix) || lowerHostname === suffix.substring(1)) {
          throw AppError.invalidImage(`Access to ${hostname} is forbidden`);
        }
      } else {
        // Exact match
        if (lowerHostname === blocked) {
          throw AppError.invalidImage(`Access to ${hostname} is forbidden`);
        }
      }
    }
  }

  /**
   * Checks if string is an IP address (IPv4 or IPv6)
   */
  private static isIpAddress(hostname: string): boolean {
    // IPv4 pattern
    const ipv4Pattern = /^(\d{1,3}\.){3}\d{1,3}$/;
    // IPv6 pattern (simplified - matches most forms)
    const ipv6Pattern = /^([0-9a-f]{0,4}:){2,7}[0-9a-f]{0,4}$/i;

    return ipv4Pattern.test(hostname) || ipv6Pattern.test(hostname);
  }

  /**
   * Validates an IP address is not private/internal
   */
  private static validateIpAddress(ip: string): void {
    // Check cloud metadata IPs
    if (this.CLOUD_METADATA_IPS.includes(ip)) {
      throw AppError.invalidImage('Access to cloud metadata endpoints is forbidden');
    }

    // Check private IP patterns
    for (const pattern of this.PRIVATE_IP_PATTERNS) {
      if (pattern.test(ip)) {
        throw AppError.invalidImage('Access to private/internal IP addresses is forbidden');
      }
    }

    // Additional check for 0.0.0.0
    if (ip === '0.0.0.0' || ip === '::') {
      throw AppError.invalidImage('Access to unspecified addresses is forbidden');
    }
  }

  /**
   * Resolves DNS and validates all returned IPs are safe
   * Prevents DNS rebinding attacks
   */
  private static async validateDnsResolution(hostname: string): Promise<void> {
    try {
      // Resolve both IPv4 and IPv6
      const addresses: string[] = [];

      // Try IPv4
      try {
        const ipv4 = await dnsLookup(hostname, { family: 4 });
        addresses.push(ipv4.address);
      } catch (error) {
        // IPv4 not available, that's ok
      }

      // Try IPv6
      try {
        const ipv6 = await dnsLookup(hostname, { family: 6 });
        addresses.push(ipv6.address);
      } catch (error) {
        // IPv6 not available, that's ok
      }

      // Must resolve to at least one address
      if (addresses.length === 0) {
        throw AppError.invalidImage(`Unable to resolve hostname: ${hostname}`);
      }

      // Validate all resolved addresses
      for (const address of addresses) {
        this.validateIpAddress(address);
      }
    } catch (error) {
      if (error instanceof AppError) {
        throw error;
      }
      throw AppError.invalidImage(`DNS resolution failed for hostname: ${hostname}`);
    }
  }

  /**
   * Creates a safe fetch configuration with SSRF protections
   */
  static createSafeFetchOptions(timeoutMs: number = 10000): RequestInit {
    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), timeoutMs);

    return {
      signal: controller.signal,
      redirect: 'manual', // Prevent automatic redirects (SSRF bypass)
      headers: {
        'User-Agent': 'ImageProcessingService/1.0 (SSRF-Protected)',
      },
      // Prevent following redirects
      follow: 0,
    };
  }

  /**
   * Validates HTTP response to prevent redirect-based SSRF
   */
  static validateResponse(response: Response): void {
    // Block redirect responses (301, 302, 303, 307, 308)
    if (response.status >= 300 && response.status < 400) {
      const location = response.headers.get('location');
      throw AppError.invalidImage(
        `Redirects are not allowed. Attempted redirect to: ${location || 'unknown'}`
      );
    }

    // Ensure successful response
    if (!response.ok) {
      throw AppError.invalidImage(
        `Failed to fetch image: ${response.status} ${response.statusText}`
      );
    }
  }
}
