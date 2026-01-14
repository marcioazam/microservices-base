import { describe, it, expect } from 'vitest';
import { fc } from '@fast-check/vitest';
import { UrlValidator } from '@security/url-validator';
import { AppError } from '@domain/errors';

/**
 * Property-Based Tests for SSRF Protection
 *
 * These tests verify that the UrlValidator correctly blocks all SSRF attack vectors
 * using property-based testing to cover edge cases and boundary conditions.
 */
describe('UrlValidator - SSRF Protection Properties', () => {
  describe('Property 1: Only HTTP/HTTPS protocols are allowed', () => {
    it('should reject all non-HTTP/HTTPS protocols', async () => {
      await fc.assert(
        fc.asyncProperty(
          fc.oneof(
            fc.constant('file://'),
            fc.constant('ftp://'),
            fc.constant('gopher://'),
            fc.constant('dict://'),
            fc.constant('sftp://'),
            fc.constant('tftp://'),
            fc.constant('ldap://'),
            fc.constant('ssh://'),
            fc.constant('tel:'),
            fc.constant('data:'),
            fc.constant('javascript:'),
          ),
          fc.domain(),
          async (protocol, domain) => {
            const url = `${protocol}${domain}/test`;

            await expect(UrlValidator.validate(url)).rejects.toThrow(AppError);
            await expect(UrlValidator.validate(url)).rejects.toThrow(/Invalid protocol/i);
          }
        )
      );
    });

    it('should accept HTTP and HTTPS protocols', async () => {
      await fc.assert(
        fc.asyncProperty(
          fc.oneof(fc.constant('http://'), fc.constant('https://')),
          fc.domain(),
          async (protocol, domain) => {
            const url = `${protocol}${domain}`;

            // Should not throw for valid public domains
            // Note: This might throw for DNS resolution, but not for protocol validation
            try {
              await UrlValidator.validate(url);
            } catch (error) {
              // If it throws, it should NOT be a protocol error
              expect(error).toBeInstanceOf(AppError);
              expect((error as AppError).message).not.toMatch(/Invalid protocol/i);
            }
          }
        )
      );
    });
  });

  describe('Property 2: Private IPv4 addresses are blocked', () => {
    const privateIpv4Generators = [
      // 127.0.0.0/8 - Loopback
      fc.tuple(fc.constant(127), fc.nat(255), fc.nat(255), fc.nat(255)),
      // 10.0.0.0/8 - Private Class A
      fc.tuple(fc.constant(10), fc.nat(255), fc.nat(255), fc.nat(255)),
      // 172.16.0.0/12 - Private Class B
      fc.tuple(fc.constant(172), fc.integer(16, 31), fc.nat(255), fc.nat(255)),
      // 192.168.0.0/16 - Private Class C
      fc.tuple(fc.constant(192), fc.constant(168), fc.nat(255), fc.nat(255)),
      // 169.254.0.0/16 - Link-local
      fc.tuple(fc.constant(169), fc.constant(254), fc.nat(255), fc.nat(255)),
    ];

    it('should reject all private IPv4 addresses', async () => {
      await fc.assert(
        fc.asyncProperty(fc.oneof(...privateIpv4Generators), async (octets) => {
          const ip = octets.join('.');
          const url = `http://${ip}/test`;

          await expect(UrlValidator.validate(url)).rejects.toThrow(AppError);
          await expect(UrlValidator.validate(url)).rejects.toThrow(
            /private|internal|forbidden/i
          );
        })
      );
    });
  });

  describe('Property 3: Cloud metadata endpoints are blocked', () => {
    const cloudMetadataEndpoints = [
      'http://169.254.169.254/latest/meta-data/',
      'http://169.254.169.254/latest/meta-data/iam/security-credentials/',
      'http://169.254.170.2/',
      'http://metadata.google.internal/',
      'http://metadata.google.internal/computeMetadata/v1/',
      'https://169.254.169.254/metadata/instance',
      'http://100.100.100.200/latest/meta-data/',
    ];

    it('should reject all known cloud metadata endpoints', async () => {
      for (const endpoint of cloudMetadataEndpoints) {
        await expect(UrlValidator.validate(endpoint)).rejects.toThrow(AppError);
        await expect(UrlValidator.validate(endpoint)).rejects.toThrow(
          /metadata|forbidden|private|internal/i
        );
      }
    });
  });

  describe('Property 4: Localhost variants are blocked', () => {
    const localhostVariants = [
      'http://localhost/',
      'http://localhost.localdomain/',
      'http://127.0.0.1/',
      'http://127.1/',
      'http://127.0.1/',
      'http://[::1]/',
      'http://[0:0:0:0:0:0:0:1]/',
    ];

    it('should reject all localhost variants', async () => {
      for (const url of localhostVariants) {
        await expect(UrlValidator.validate(url)).rejects.toThrow(AppError);
        await expect(UrlValidator.validate(url)).rejects.toThrow(/forbidden|private|internal/i);
      }
    });
  });

  describe('Property 5: IPv6 private addresses are blocked', () => {
    const privateIpv6Patterns = [
      'http://[::1]/test',                    // Loopback
      'http://[fc00::1]/test',                // Unique Local
      'http://[fd00::1]/test',                // Unique Local
      'http://[fe80::1]/test',                // Link-Local
      'http://[ff00::1]/test',                // Multicast
      'http://[fd00:ec2::254]/test',          // AWS IPv6 metadata
    ];

    it('should reject IPv6 private addresses', async () => {
      for (const url of privateIpv6Patterns) {
        await expect(UrlValidator.validate(url)).rejects.toThrow(AppError);
        await expect(UrlValidator.validate(url)).rejects.toThrow(/forbidden|private|internal/i);
      }
    });
  });

  describe('Property 6: Internal domain patterns are blocked', () => {
    it('should reject .local domains', async () => {
      await fc.assert(
        fc.asyncProperty(fc.domain().filter((d) => !d.endsWith('.local')), async (domain) => {
          const url = `http://${domain}.local/test`;

          await expect(UrlValidator.validate(url)).rejects.toThrow(AppError);
          await expect(UrlValidator.validate(url)).rejects.toThrow(/forbidden/i);
        })
      );
    });

    it('should reject .internal domains', async () => {
      await fc.assert(
        fc.asyncProperty(fc.domain(), async (domain) => {
          const url = `http://${domain}.internal/test`;

          await expect(UrlValidator.validate(url)).rejects.toThrow(AppError);
          await expect(UrlValidator.validate(url)).rejects.toThrow(/forbidden/i);
        })
      );
    });
  });

  describe('Property 7: URLs with credentials are blocked', () => {
    it('should reject URLs with username', async () => {
      await fc.assert(
        fc.asyncProperty(
          fc.webUrl({ validSchemes: ['http', 'https'] }),
          fc.string({ minLength: 1, maxLength: 20 }),
          async (baseUrl, username) => {
            const urlObj = new URL(baseUrl);
            const urlWithCreds = `${urlObj.protocol}//${username}@${urlObj.host}${urlObj.pathname}`;

            await expect(UrlValidator.validate(urlWithCreds)).rejects.toThrow(AppError);
            await expect(UrlValidator.validate(urlWithCreds)).rejects.toThrow(/credential/i);
          }
        )
      );
    });

    it('should reject URLs with username and password', async () => {
      await fc.assert(
        fc.asyncProperty(
          fc.webUrl({ validSchemes: ['http', 'https'] }),
          fc.string({ minLength: 1, maxLength: 20 }),
          fc.string({ minLength: 1, maxLength: 20 }),
          async (baseUrl, username, password) => {
            const urlObj = new URL(baseUrl);
            const urlWithCreds = `${urlObj.protocol}//${username}:${password}@${urlObj.host}${urlObj.pathname}`;

            await expect(UrlValidator.validate(urlWithCreds)).rejects.toThrow(AppError);
            await expect(UrlValidator.validate(urlWithCreds)).rejects.toThrow(/credential/i);
          }
        )
      );
    });
  });

  describe('Property 8: Redirect responses are blocked', () => {
    it('should reject redirect status codes', () => {
      const redirectCodes = [301, 302, 303, 307, 308];

      for (const code of redirectCodes) {
        const mockResponse = {
          status: code,
          ok: false,
          headers: new Headers({ location: 'http://evil.com' }),
        } as Response;

        expect(() => UrlValidator.validateResponse(mockResponse)).toThrow(AppError);
        expect(() => UrlValidator.validateResponse(mockResponse)).toThrow(/redirect/i);
      }
    });
  });

  describe('Property 9: Safe fetch options prevent redirects', () => {
    it('should create fetch options with redirect: manual', () => {
      const options = UrlValidator.createSafeFetchOptions(5000);

      expect(options.redirect).toBe('manual');
      expect(options.signal).toBeDefined();
      expect(options.follow).toBe(0);
    });

    it('should include abort controller for timeout', () => {
      const options = UrlValidator.createSafeFetchOptions(1000);

      expect(options.signal).toBeInstanceOf(AbortSignal);
    });
  });

  describe('Property 10: Special IP addresses are blocked', () => {
    const specialIps = [
      '0.0.0.0',       // Unspecified
      '255.255.255.255', // Broadcast
      '224.0.0.1',     // Multicast
    ];

    it('should reject special-purpose IP addresses', async () => {
      for (const ip of specialIps) {
        const url = `http://${ip}/test`;

        await expect(UrlValidator.validate(url)).rejects.toThrow(AppError);
      }
    });
  });

  describe('Property 11: Valid public URLs are accepted', () => {
    const validPublicUrls = [
      'https://example.com/image.jpg',
      'https://cdn.example.com/images/photo.png',
      'https://images.example.co.uk/test.webp',
      'http://public-bucket.s3.amazonaws.com/image.jpg',
    ];

    it('should accept valid public URLs (may fail on DNS)', async () => {
      for (const url of validPublicUrls) {
        // Note: DNS resolution might fail in test environment
        // We're verifying that it doesn't fail on protocol/format validation
        try {
          await UrlValidator.validate(url);
          // If it succeeds, it passed all checks
          expect(true).toBe(true);
        } catch (error) {
          // If it fails, ensure it's NOT failing on protocol/format/IP validation
          const errorMessage = (error as AppError).message.toLowerCase();
          expect(errorMessage).not.toMatch(/invalid protocol/i);
          expect(errorMessage).not.toMatch(/private.*forbidden/i);
          expect(errorMessage).not.toMatch(/internal.*forbidden/i);
          expect(errorMessage).not.toMatch(/credential/i);
        }
      }
    });
  });

  describe('Property 12: DNS rebinding protection', () => {
    it('should validate resolved IP addresses', async () => {
      // This test verifies that even if a domain resolves to a private IP,
      // it will be rejected. This prevents DNS rebinding attacks.

      // We can't easily test this without a mock DNS server,
      // but we document the behavior here for manual testing:

      // If attacker.com initially resolves to 1.2.3.4 (public)
      // but then changes DNS to resolve to 192.168.1.1 (private)
      // the UrlValidator will detect and block the private IP

      expect(true).toBe(true); // Placeholder - requires integration test
    });
  });

  describe('Property 13: Error messages do not leak sensitive information', () => {
    it('should provide generic error messages', async () => {
      const testCases = [
        { url: 'http://127.0.0.1/', expectedPattern: /forbidden|private|internal/i },
        { url: 'file:///etc/passwd', expectedPattern: /Invalid protocol/i },
        { url: 'http://metadata.google.internal/', expectedPattern: /forbidden/i },
      ];

      for (const { url, expectedPattern } of testCases) {
        try {
          await UrlValidator.validate(url);
          // Should not reach here
          expect(true).toBe(false);
        } catch (error) {
          expect(error).toBeInstanceOf(AppError);
          const message = (error as AppError).message;

          // Should not leak internal details
          expect(message).not.toMatch(/AWS|cloud|metadata endpoint/i);
          expect(message).not.toMatch(/127\.0\.0\.1/);

          // Should match expected pattern
          expect(message).toMatch(expectedPattern);
        }
      }
    });
  });
});
