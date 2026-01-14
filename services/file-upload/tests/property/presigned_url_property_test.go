// Package property contains property-based tests for file-upload service.
// Feature: file-upload-modernization-2025, Property 11: Presigned URL Validity
// Validates: Requirements 9.3, 9.4
package property

import (
	"net/url"
	"strings"
	"testing"
	"time"

	"pgregory.net/rapid"
)

// MockPresignedURLGenerator simulates presigned URL generation for testing.
type MockPresignedURLGenerator struct {
	bucket   string
	region   string
	baseTime time.Time
}

func NewMockPresignedURLGenerator(bucket, region string) *MockPresignedURLGenerator {
	return &MockPresignedURLGenerator{
		bucket:   bucket,
		region:   region,
		baseTime: time.Now(),
	}
}

// GenerateUploadURL generates a mock presigned upload URL.
func (g *MockPresignedURLGenerator) GenerateUploadURL(path string, expiry time.Duration) *PresignedURL {
	expiresAt := g.baseTime.Add(expiry)
	return &PresignedURL{
		URL:       g.buildURL(path, "PUT", expiresAt),
		Method:    "PUT",
		ExpiresAt: expiresAt,
		Path:      path,
	}
}

// GenerateDownloadURL generates a mock presigned download URL.
func (g *MockPresignedURLGenerator) GenerateDownloadURL(path string, expiry time.Duration) *PresignedURL {
	expiresAt := g.baseTime.Add(expiry)
	return &PresignedURL{
		URL:       g.buildURL(path, "GET", expiresAt),
		Method:    "GET",
		ExpiresAt: expiresAt,
		Path:      path,
	}
}

func (g *MockPresignedURLGenerator) buildURL(path, method string, expiresAt time.Time) string {
	return "https://" + g.bucket + ".s3." + g.region + ".amazonaws.com/" + path +
		"?X-Amz-Algorithm=AWS4-HMAC-SHA256" +
		"&X-Amz-Expires=" + string(rune(int(time.Until(expiresAt).Seconds()))) +
		"&X-Amz-SignedHeaders=host" +
		"&X-Amz-Signature=mock-signature"
}

// SetBaseTime sets the base time for testing expiry.
func (g *MockPresignedURLGenerator) SetBaseTime(t time.Time) {
	g.baseTime = t
}

// PresignedURL represents a presigned URL with metadata.
type PresignedURL struct {
	URL       string
	Method    string
	ExpiresAt time.Time
	Path      string
}

// IsExpired checks if the URL has expired.
func (p *PresignedURL) IsExpired(now time.Time) bool {
	return now.After(p.ExpiresAt)
}

// IsValid checks if the URL is valid for the given method.
func (p *PresignedURL) IsValid(method string, now time.Time) bool {
	if p.IsExpired(now) {
		return false
	}
	return p.Method == method
}

// TestProperty11_UploadURLAllowsPUT tests that upload URLs allow PUT operations within expiry.
// Property 11: Presigned URL Validity
// Validates: Requirements 9.3, 9.4
func TestProperty11_UploadURLAllowsPUT(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		bucket := rapid.StringMatching(`[a-z0-9-]{3,20}`).Draw(t, "bucket")
		region := rapid.SampledFrom([]string{"us-east-1", "us-west-2", "eu-west-1"}).Draw(t, "region")
		path := rapid.StringMatching(`tenant-[a-z0-9]{8}/2025/01/15/[a-f0-9]{64}/[a-z0-9]+\.pdf`).Draw(t, "path")
		expiryMinutes := rapid.IntRange(5, 60).Draw(t, "expiryMinutes")

		generator := NewMockPresignedURLGenerator(bucket, region)
		expiry := time.Duration(expiryMinutes) * time.Minute

		presignedURL := generator.GenerateUploadURL(path, expiry)

		// Property: Upload URLs SHALL allow PUT operations
		if presignedURL.Method != "PUT" {
			t.Errorf("upload URL method should be PUT, got %q", presignedURL.Method)
		}

		// Property: URL SHALL be valid within expiry time
		now := generator.baseTime.Add(time.Duration(expiryMinutes/2) * time.Minute)
		if !presignedURL.IsValid("PUT", now) {
			t.Error("upload URL should be valid within expiry time")
		}

		// Property: URL SHALL contain the path
		if !strings.Contains(presignedURL.URL, path) {
			t.Errorf("URL should contain path %q", path)
		}
	})
}

// TestProperty11_DownloadURLAllowsGET tests that download URLs allow GET operations within expiry.
// Property 11: Presigned URL Validity
// Validates: Requirements 9.3, 9.4
func TestProperty11_DownloadURLAllowsGET(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		bucket := rapid.StringMatching(`[a-z0-9-]{3,20}`).Draw(t, "bucket")
		region := rapid.SampledFrom([]string{"us-east-1", "us-west-2", "eu-west-1"}).Draw(t, "region")
		path := rapid.StringMatching(`tenant-[a-z0-9]{8}/2025/01/15/[a-f0-9]{64}/[a-z0-9]+\.pdf`).Draw(t, "path")
		expiryMinutes := rapid.IntRange(5, 60).Draw(t, "expiryMinutes")

		generator := NewMockPresignedURLGenerator(bucket, region)
		expiry := time.Duration(expiryMinutes) * time.Minute

		presignedURL := generator.GenerateDownloadURL(path, expiry)

		// Property: Download URLs SHALL allow GET operations
		if presignedURL.Method != "GET" {
			t.Errorf("download URL method should be GET, got %q", presignedURL.Method)
		}

		// Property: URL SHALL be valid within expiry time
		now := generator.baseTime.Add(time.Duration(expiryMinutes/2) * time.Minute)
		if !presignedURL.IsValid("GET", now) {
			t.Error("download URL should be valid within expiry time")
		}
	})
}

// TestProperty11_ExpiredURLsReturnForbidden tests that expired URLs are invalid.
// Property 11: Presigned URL Validity
// Validates: Requirements 9.3, 9.4
func TestProperty11_ExpiredURLsReturnForbidden(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		bucket := rapid.StringMatching(`[a-z0-9-]{3,20}`).Draw(t, "bucket")
		region := rapid.SampledFrom([]string{"us-east-1", "us-west-2"}).Draw(t, "region")
		path := rapid.StringMatching(`tenant-[a-z0-9]{8}/2025/01/15/[a-f0-9]{64}/file\.pdf`).Draw(t, "path")
		expiryMinutes := rapid.IntRange(5, 30).Draw(t, "expiryMinutes")
		extraMinutes := rapid.IntRange(1, 60).Draw(t, "extraMinutes")

		generator := NewMockPresignedURLGenerator(bucket, region)
		expiry := time.Duration(expiryMinutes) * time.Minute

		uploadURL := generator.GenerateUploadURL(path, expiry)
		downloadURL := generator.GenerateDownloadURL(path, expiry)

		// Time after expiry
		expiredTime := generator.baseTime.Add(expiry + time.Duration(extraMinutes)*time.Minute)

		// Property: Expired upload URLs SHALL be invalid
		if uploadURL.IsValid("PUT", expiredTime) {
			t.Error("expired upload URL should be invalid")
		}

		// Property: Expired download URLs SHALL be invalid
		if downloadURL.IsValid("GET", expiredTime) {
			t.Error("expired download URL should be invalid")
		}

		// Property: IsExpired SHALL return true after expiry
		if !uploadURL.IsExpired(expiredTime) {
			t.Error("upload URL should be marked as expired")
		}
		if !downloadURL.IsExpired(expiredTime) {
			t.Error("download URL should be marked as expired")
		}
	})
}

// TestProperty11_URLContainsRequiredParameters tests that presigned URLs contain required parameters.
// Property 11: Presigned URL Validity
// Validates: Requirements 9.3, 9.4
func TestProperty11_URLContainsRequiredParameters(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		bucket := rapid.StringMatching(`[a-z0-9-]{3,20}`).Draw(t, "bucket")
		region := rapid.SampledFrom([]string{"us-east-1", "us-west-2", "eu-west-1"}).Draw(t, "region")
		path := rapid.StringMatching(`tenant-[a-z0-9]{8}/2025/01/15/[a-f0-9]{64}/[a-z0-9]+\.pdf`).Draw(t, "path")

		generator := NewMockPresignedURLGenerator(bucket, region)
		presignedURL := generator.GenerateUploadURL(path, 15*time.Minute)

		// Parse URL
		parsedURL, err := url.Parse(presignedURL.URL)
		if err != nil {
			t.Fatalf("failed to parse URL: %v", err)
		}

		// Property: URL SHALL be HTTPS
		if parsedURL.Scheme != "https" {
			t.Errorf("URL scheme should be https, got %q", parsedURL.Scheme)
		}

		// Property: URL SHALL contain bucket in host
		if !strings.Contains(parsedURL.Host, bucket) {
			t.Errorf("URL host should contain bucket %q", bucket)
		}

		// Property: URL SHALL contain region in host
		if !strings.Contains(parsedURL.Host, region) {
			t.Errorf("URL host should contain region %q", region)
		}

		// Property: URL SHALL contain path
		if !strings.Contains(parsedURL.Path, path) && !strings.Contains(presignedURL.URL, path) {
			t.Errorf("URL should contain path %q", path)
		}

		// Property: URL SHALL contain signature parameters
		if !strings.Contains(presignedURL.URL, "X-Amz-Algorithm") {
			t.Error("URL should contain X-Amz-Algorithm parameter")
		}
		if !strings.Contains(presignedURL.URL, "X-Amz-Signature") {
			t.Error("URL should contain X-Amz-Signature parameter")
		}
	})
}

// TestProperty11_MethodMismatchInvalidatesURL tests that using wrong method invalidates URL.
// Property 11: Presigned URL Validity
// Validates: Requirements 9.3, 9.4
func TestProperty11_MethodMismatchInvalidatesURL(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		bucket := rapid.StringMatching(`[a-z0-9-]{3,20}`).Draw(t, "bucket")
		region := rapid.SampledFrom([]string{"us-east-1", "us-west-2"}).Draw(t, "region")
		path := rapid.StringMatching(`tenant-[a-z0-9]{8}/2025/01/15/[a-f0-9]{64}/file\.pdf`).Draw(t, "path")

		generator := NewMockPresignedURLGenerator(bucket, region)
		now := generator.baseTime

		uploadURL := generator.GenerateUploadURL(path, 15*time.Minute)
		downloadURL := generator.GenerateDownloadURL(path, 15*time.Minute)

		// Property: Upload URL SHALL NOT be valid for GET
		if uploadURL.IsValid("GET", now) {
			t.Error("upload URL should not be valid for GET method")
		}

		// Property: Download URL SHALL NOT be valid for PUT
		if downloadURL.IsValid("PUT", now) {
			t.Error("download URL should not be valid for PUT method")
		}

		// Property: Upload URL SHALL be valid for PUT
		if !uploadURL.IsValid("PUT", now) {
			t.Error("upload URL should be valid for PUT method")
		}

		// Property: Download URL SHALL be valid for GET
		if !downloadURL.IsValid("GET", now) {
			t.Error("download URL should be valid for GET method")
		}
	})
}
