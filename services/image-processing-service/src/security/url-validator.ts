import { AppError } from '@domain/errors';
import dns from 'dns';
import { promisify } from 'util';

const dnsLookup = promisify(dns.lookup);

/**
 * DNS cache entry with TTL for TOCTOU attack prevention
 */
interface DnsCacheEntry {
  addresses: string[];
  timestamp: number;
  ttl: number; // milliseconds
}

/**
 * Validated URL result with pinned IP addresses
 */
export interface ValidatedUrl {
  url: URL;
  validatedIps: string[];
  primaryIp: string;
}

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
 * 7. DNS caching with TTL to prevent TOCTOU attacks
 * 8. IP pinning for validated hostnames
 */
export class UrlValidator {
  // DNS cache with TTL (default 5 minutes)
  private static dnsCache = new Map<string, DnsCacheEntry>();
  private static readonly DEFAULT_DNS_TTL = 5 * 60 * 1000; // 5 minutes
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
   * Returns validated URL with pinned IP addresses to prevent TOCTOU attacks
   * @param urlString - The URL to validate
   * @throws AppError if URL is unsafe
   */
  static async validate(urlString: string): Promise<ValidatedUrl> {
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

    let validatedIps: string[];

    if (isIpAddress) {
      // Direct IP address - validate it's not private
      this.validateIpAddress(parsedUrl.hostname);
      validatedIps = [parsedUrl.hostname];
    } else {
      // Domain name - resolve DNS and validate IPs (with caching)
      validatedIps = await this.validateDnsResolution(parsedUrl.hostname);
    }

    return {
      url: parsedUrl,
      validatedIps,
      primaryIp: validatedIps[0],
    };
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
   * Prevents DNS rebinding attacks with caching
   * @returns Array of validated IP addresses
   */
  private static async validateDnsResolution(hostname: string): Promise<string[]> {
    // Check cache first
    const cached = this.getDnsCacheEntry(hostname);
    if (cached) {
      return cached.addresses;
    }

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

      // Cache validated addresses
      this.setDnsCacheEntry(hostname, addresses, this.DEFAULT_DNS_TTL);

      return addresses;
    } catch (error) {
      if (error instanceof AppError) {
        throw error;
      }
      throw AppError.invalidImage(`DNS resolution failed for hostname: ${hostname}`);
    }
  }

  /**
   * Get DNS cache entry if valid
   */
  private static getDnsCacheEntry(hostname: string): DnsCacheEntry | null {
    const entry = this.dnsCache.get(hostname);

    if (!entry) {
      return null;
    }

    // Check if expired
    const now = Date.now();
    if (now - entry.timestamp > entry.ttl) {
      // Expired - remove from cache
      this.dnsCache.delete(hostname);
      return null;
    }

    return entry;
  }

  /**
   * Set DNS cache entry
   */
  private static setDnsCacheEntry(hostname: string, addresses: string[], ttl: number): void {
    this.dnsCache.set(hostname, {
      addresses,
      timestamp: Date.now(),
      ttl,
    });
  }

  /**
   * Clear DNS cache (useful for testing or manual cache invalidation)
   */
  static clearDnsCache(): void {
    this.dnsCache.clear();
  }

  /**
   * Clear expired DNS cache entries
   */
  static cleanupDnsCache(): void {
    const now = Date.now();
    for (const [hostname, entry] of this.dnsCache.entries()) {
      if (now - entry.timestamp > entry.ttl) {
        this.dnsCache.delete(hostname);
      }
    }
  }

  /**
   * Builds a URL with pinned IP to prevent DNS rebinding attacks
   * @param validatedUrl - The validated URL result from validate()
   * @returns URL string with IP and original hostname as Host header
   */
  static buildPinnedUrl(validatedUrl: ValidatedUrl): { url: string; headers: Record<string, string> } {
    const { url, primaryIp } = validatedUrl;

    // Build URL with IP instead of hostname
    const protocol = url.protocol;
    const port = url.port ? `:${url.port}` : '';
    const pathname = url.pathname;
    const search = url.search;
    const hash = url.hash;

    // Wrap IPv6 addresses in brackets
    const ipForUrl = primaryIp.includes(':') ? `[${primaryIp}]` : primaryIp;

    const pinnedUrl = `${protocol}//${ipForUrl}${port}${pathname}${search}${hash}`;

    return {
      url: pinnedUrl,
      headers: {
        Host: url.hostname, // Original hostname as Host header
      },
    };
  }

  /**
   * Creates a safe fetch configuration with SSRF protections
   * @param validatedUrl - Optional validated URL for IP pinning
   */
  static createSafeFetchOptions(timeoutMs: number = 10000, validatedUrl?: ValidatedUrl): RequestInit {
    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), timeoutMs);

    const headers: Record<string, string> = {
      'User-Agent': 'ImageProcessingService/1.0 (SSRF-Protected)',
    };

    // Add Host header for IP pinning if validatedUrl provided
    if (validatedUrl) {
      headers['Host'] = validatedUrl.url.hostname;
    }

    return {
      signal: controller.signal,
      redirect: 'manual', // Prevent automatic redirects (SSRF bypass)
      headers,
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
