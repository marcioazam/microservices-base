/**
 * Manual tests for SSRF protection improvements.
 *
 * Tests the following HIGH priority fixes:
 * 1. DNS caching with TTL
 * 2. IP pinning to prevent TOCTOU attacks
 * 3. DNS cache management
 *
 * Run with: npx ts-node tests/manual/test-ssrf-improvements.ts
 */

import { UrlValidator, ValidatedUrl } from '../../src/security/url-validator';

// Test utilities
let testsPassed = 0;
let testsFailed = 0;

function assert(condition: boolean, message: string): void {
  if (condition) {
    console.log(`  ✓ ${message}`);
  } else {
    console.log(`  ❌ ${message}`);
    testsFailed++;
    throw new Error(`Assertion failed: ${message}`);
  }
}

function separator(): void {
  console.log('='.repeat(70));
}

// Test 1: DNS Caching
async function test1_dns_caching(): Promise<boolean> {
  separator();
  console.log('Test 1: DNS Caching with TTL');
  separator();

  try {
    // Clear cache first
    UrlValidator.clearDnsCache();
    console.log('\n✓ Cache cleared');

    // First validation - should perform DNS lookup
    console.log('\n1a. First validation (cache miss)...');
    const start1 = Date.now();
    const result1 = await UrlValidator.validate('https://example.com');
    const duration1 = Date.now() - start1;

    assert(result1.url.hostname === 'example.com', 'Hostname is example.com');
    assert(result1.validatedIps.length > 0, 'Has validated IPs');
    assert(result1.primaryIp.length > 0, 'Has primary IP');
    console.log(`  ✓ First lookup took ${duration1}ms`);
    console.log(`  ✓ Primary IP: ${result1.primaryIp}`);
    console.log(`  ✓ All IPs: ${result1.validatedIps.join(', ')}`);

    // Second validation - should use cache
    console.log('\n1b. Second validation (cache hit)...');
    const start2 = Date.now();
    const result2 = await UrlValidator.validate('https://example.com');
    const duration2 = Date.now() - start2;

    assert(result2.url.hostname === 'example.com', 'Hostname is example.com');
    assert(result2.primaryIp === result1.primaryIp, 'Primary IP matches (cached)');
    assert(duration2 < duration1, `Cached lookup faster (${duration2}ms vs ${duration1}ms)`);
    console.log(`  ✓ Cached lookup took ${duration2}ms (${Math.round((1 - duration2/duration1) * 100)}% faster)`);

    // Clear cache and verify
    console.log('\n1c. Cache invalidation...');
    UrlValidator.clearDnsCache();
    console.log('  ✓ Cache cleared');

    const start3 = Date.now();
    await UrlValidator.validate('https://example.com');
    const duration3 = Date.now() - start3;

    assert(duration3 > duration2, `After clear, lookup slower (${duration3}ms vs ${duration2}ms)`);
    console.log(`  ✓ Lookup after clear took ${duration3}ms (cache miss)`);

    console.log('\n✅ Test 1 PASSED: DNS caching working correctly');
    testsPassed++;
    return true;
  } catch (error) {
    console.log(`\n❌ Test 1 FAILED: ${error}`);
    return false;
  }
}

// Test 2: IP Pinning
async function test2_ip_pinning(): Promise<boolean> {
  separator();
  console.log('Test 2: IP Pinning for TOCTOU Prevention');
  separator();

  try {
    console.log('\n2a. Validating URL and getting pinned IP...');
    const validatedUrl = await UrlValidator.validate('https://example.com');

    assert(validatedUrl.url.hostname === 'example.com', 'Hostname is example.com');
    assert(validatedUrl.validatedIps.length > 0, 'Has validated IPs');
    assert(validatedUrl.primaryIp.length > 0, 'Has primary IP');

    console.log(`  ✓ Validated URL: ${validatedUrl.url.toString()}`);
    console.log(`  ✓ Validated IPs: ${validatedUrl.validatedIps.join(', ')}`);
    console.log(`  ✓ Primary (pinned) IP: ${validatedUrl.primaryIp}`);

    console.log('\n2b. Building pinned URL...');
    const pinned = UrlValidator.buildPinnedUrl(validatedUrl);

    assert(pinned.url.includes(validatedUrl.primaryIp), 'Pinned URL contains IP');
    assert(pinned.headers.Host === 'example.com', 'Host header is original hostname');

    console.log(`  ✓ Pinned URL: ${pinned.url}`);
    console.log(`  ✓ Host header: ${pinned.headers.Host}`);

    // Verify IP is used instead of hostname
    const isIpInUrl = pinned.url.includes(validatedUrl.primaryIp);
    const isHostnameNotInUrl = !pinned.url.includes('example.com');
    assert(isIpInUrl && isHostnameNotInUrl, 'URL uses IP, not hostname');

    console.log('\n2c. Testing IPv6 handling...');
    // Simulate IPv6 address
    const ipv6Result: ValidatedUrl = {
      url: new URL('https://example.com'),
      validatedIps: ['2606:2800:220:1:248:1893:25c8:1946'],
      primaryIp: '2606:2800:220:1:248:1893:25c8:1946',
    };

    const ipv6Pinned = UrlValidator.buildPinnedUrl(ipv6Result);
    assert(ipv6Pinned.url.includes('['), 'IPv6 wrapped in brackets');
    assert(ipv6Pinned.url.includes(']'), 'IPv6 brackets closed');
    console.log(`  ✓ IPv6 pinned URL: ${ipv6Pinned.url}`);

    console.log('\n✅ Test 2 PASSED: IP pinning working correctly');
    testsPassed++;
    return true;
  } catch (error) {
    console.log(`\n❌ Test 2 FAILED: ${error}`);
    return false;
  }
}

// Test 3: Safe Fetch Options
async function test3_safe_fetch_options(): Promise<boolean> {
  separator();
  console.log('Test 3: Safe Fetch Options with IP Pinning');
  separator();

  try {
    console.log('\n3a. Creating safe fetch options without validated URL...');
    const options1 = UrlValidator.createSafeFetchOptions(5000);

    assert(options1.redirect === 'manual', 'Redirects set to manual');
    assert(options1.follow === 0, 'Follow redirects disabled');
    assert('User-Agent' in (options1.headers as Record<string, string>), 'Has User-Agent header');
    console.log('  ✓ Basic options created correctly');

    console.log('\n3b. Creating safe fetch options with validated URL...');
    const validatedUrl = await UrlValidator.validate('https://example.com');
    const options2 = UrlValidator.createSafeFetchOptions(5000, validatedUrl);

    assert(options2.redirect === 'manual', 'Redirects set to manual');
    assert('Host' in (options2.headers as Record<string, string>), 'Has Host header for IP pinning');
    assert((options2.headers as Record<string, string>).Host === 'example.com', 'Host header is original hostname');
    console.log(`  ✓ Options with IP pinning: Host=${(options2.headers as Record<string, string>).Host}`);

    console.log('\n✅ Test 3 PASSED: Safe fetch options working correctly');
    testsPassed++;
    return true;
  } catch (error) {
    console.log(`\n❌ Test 3 FAILED: ${error}`);
    return false;
  }
}

// Test 4: DNS Cache Cleanup
async function test4_dns_cache_cleanup(): Promise<boolean> {
  separator();
  console.log('Test 4: DNS Cache Cleanup');
  separator();

  try {
    console.log('\n4a. Populating cache with multiple entries...');
    UrlValidator.clearDnsCache();

    await UrlValidator.validate('https://example.com');
    await UrlValidator.validate('https://google.com');
    await UrlValidator.validate('https://github.com');

    console.log('  ✓ Cache populated with 3 entries');

    console.log('\n4b. Testing cache cleanup (should not remove fresh entries)...');
    UrlValidator.cleanupDnsCache();
    console.log('  ✓ Cleanup executed');

    // Validate again - should be fast (cached)
    const start = Date.now();
    await UrlValidator.validate('https://example.com');
    const duration = Date.now() - start;

    assert(duration < 50, `Entry still cached (${duration}ms < 50ms)`);
    console.log(`  ✓ Cache entries preserved (lookup: ${duration}ms)`);

    console.log('\n4c. Testing full cache clear...');
    UrlValidator.clearDnsCache();
    console.log('  ✓ Cache cleared');

    const start2 = Date.now();
    await UrlValidator.validate('https://example.com');
    const duration2 = Date.now() - start2;

    assert(duration2 > duration, `After clear, lookup slower (${duration2}ms vs ${duration}ms)`);
    console.log(`  ✓ Cache cleared successfully (lookup: ${duration2}ms)`);

    console.log('\n✅ Test 4 PASSED: DNS cache cleanup working correctly');
    testsPassed++;
    return true;
  } catch (error) {
    console.log(`\n❌ Test 4 FAILED: ${error}`);
    return false;
  }
}

// Test 5: SSRF Protection Still Active
async function test5_ssrf_protection_active(): Promise<boolean> {
  separator();
  console.log('Test 5: SSRF Protection Still Active');
  separator();

  const testCases = [
    { url: 'http://localhost/admin', name: 'localhost' },
    { url: 'http://127.0.0.1/admin', name: '127.0.0.1' },
    { url: 'http://10.0.0.1/internal', name: 'private IP 10.x' },
    { url: 'http://192.168.1.1/router', name: 'private IP 192.168.x' },
    { url: 'http://169.254.169.254/metadata', name: 'cloud metadata' },
    { url: 'file:///etc/passwd', name: 'file protocol' },
    { url: 'http://[::1]/admin', name: 'IPv6 localhost' },
  ];

  let blocked = 0;

  for (const testCase of testCases) {
    try {
      await UrlValidator.validate(testCase.url);
      console.log(`  ❌ FAIL: ${testCase.name} was not blocked`);
    } catch (error) {
      console.log(`  ✓ PASS: ${testCase.name} blocked correctly`);
      blocked++;
    }
  }

  const allBlocked = blocked === testCases.length;

  if (allBlocked) {
    console.log(`\n✅ Test 5 PASSED: All ${testCases.length} dangerous URLs blocked`);
    testsPassed++;
    return true;
  } else {
    console.log(`\n❌ Test 5 FAILED: Only ${blocked}/${testCases.length} dangerous URLs blocked`);
    return false;
  }
}

// Main test runner
async function main(): Promise<void> {
  console.log('\n' + '#'.repeat(70));
  console.log('# SSRF Protection Improvements - Validation Tests');
  console.log('#'.repeat(70));

  console.log('\nRunning validation tests...\n');

  try {
    await test1_dns_caching();
    await test2_ip_pinning();
    await test3_safe_fetch_options();
    await test4_dns_cache_cleanup();
    await test5_ssrf_protection_active();

    // Summary
    separator();
    console.log('TEST SUMMARY');
    separator();

    const total = testsPassed + testsFailed;
    console.log(`\nTotal: ${testsPassed} passed, ${testsFailed} failed out of ${total} tests`);

    if (testsFailed === 0) {
      console.log('\n' + '='.repeat(70));
      console.log('🎉 ALL TESTS PASSED!');
      console.log('='.repeat(70) + '\n');
      process.exit(0);
    } else {
      console.log(`\n⚠️  ${testsFailed} test(s) failed. Please review the output above.\n`);
      process.exit(1);
    }
  } catch (error) {
    console.log(`\n❌ Error running tests: ${error}`);
    process.exit(1);
  }
}

// Run tests
main();
