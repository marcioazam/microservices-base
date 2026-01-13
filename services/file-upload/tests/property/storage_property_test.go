package property

import (
	"regexp"
	"testing"
	"time"

	"github.com/auth-platform/file-upload/internal/storage"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// Feature: file-upload-service, Property 4: Storage Path Structure
// Validates: Requirements 4.5
// For any successfully stored file, the storage path SHALL follow the format
// /{tenant_id}/{year}/{month}/{day}/{file_hash}/{filename}

func TestStoragePathStructureProperty(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	pathGen := storage.NewPathGenerator()

	// Property: Generated paths follow the correct format
	properties.Property("paths follow correct format", prop.ForAll(
		func(tenantID, fileHash, filename string) bool {
			if tenantID == "" || fileHash == "" || filename == "" {
				return true // Skip empty inputs
			}

			path := pathGen.GeneratePath(tenantID, fileHash, filename)

			// Path should match pattern: {tenant}/{year}/{month}/{day}/{hash}/{filename}
			pattern := regexp.MustCompile(`^[^/]+/\d{4}/\d{2}/\d{2}/[^/]+/[^/]+$`)
			return pattern.MatchString(path)
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) <= 64 }),
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) <= 64 }),
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) <= 64 }),
	))

	// Property: Path contains tenant ID
	properties.Property("path contains tenant ID", prop.ForAll(
		func(tenantID, fileHash, filename string) bool {
			if tenantID == "" {
				return true
			}

			path := pathGen.GeneratePath(tenantID, fileHash, filename)
			components, err := pathGen.ParsePath(path)
			if err != nil {
				return false
			}

			return components.TenantID == tenantID
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) <= 64 }),
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) <= 64 }),
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) <= 64 }),
	))

	// Property: Path contains file hash
	properties.Property("path contains file hash", prop.ForAll(
		func(tenantID, fileHash, filename string) bool {
			if fileHash == "" {
				return true
			}

			path := pathGen.GeneratePath(tenantID, fileHash, filename)
			components, err := pathGen.ParsePath(path)
			if err != nil {
				return false
			}

			return components.FileHash == fileHash
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) <= 64 }),
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) <= 64 }),
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) <= 64 }),
	))

	// Property: Path contains filename
	properties.Property("path contains filename", prop.ForAll(
		func(tenantID, fileHash, filename string) bool {
			if filename == "" {
				return true
			}

			path := pathGen.GeneratePath(tenantID, fileHash, filename)
			components, err := pathGen.ParsePath(path)
			if err != nil {
				return false
			}

			return components.Filename == filename
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) <= 64 }),
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) <= 64 }),
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) <= 64 }),
	))

	// Property: Year/month/day correspond to current time
	properties.Property("date components are current", prop.ForAll(
		func(tenantID, fileHash, filename string) bool {
			if tenantID == "" || fileHash == "" || filename == "" {
				return true
			}

			now := time.Now().UTC()
			path := pathGen.GeneratePath(tenantID, fileHash, filename)
			components, err := pathGen.ParsePath(path)
			if err != nil {
				return false
			}

			expectedYear := now.Format("2006")
			expectedMonth := now.Format("01")
			expectedDay := now.Format("02")

			return components.Year == expectedYear &&
				components.Month == expectedMonth &&
				components.Day == expectedDay
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) <= 64 }),
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) <= 64 }),
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) <= 64 }),
	))

	// Property: Path validation works correctly
	properties.Property("valid paths pass validation", prop.ForAll(
		func(tenantID, fileHash, filename string) bool {
			if tenantID == "" || fileHash == "" || filename == "" {
				return true
			}

			path := pathGen.GeneratePath(tenantID, fileHash, filename)
			return storage.ValidatePath(path)
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) <= 64 }),
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) <= 64 }),
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) <= 64 }),
	))

	// Property: Invalid paths fail validation
	properties.Property("invalid paths fail validation", prop.ForAll(
		func(invalidPath string) bool {
			// Paths with fewer than 6 components should fail
			return !storage.ValidatePath(invalidPath)
		},
		gen.OneConstOf(
			"",
			"single",
			"two/parts",
			"three/parts/here",
			"four/parts/are/here",
			"five/parts/are/here/now",
		),
	))

	// Property: GeneratePathWithTime uses provided timestamp
	properties.Property("GeneratePathWithTime uses provided timestamp", prop.ForAll(
		func(tenantID, fileHash, filename string, year, month, day int) bool {
			if tenantID == "" || fileHash == "" || filename == "" {
				return true
			}

			// Normalize values
			if year < 2000 {
				year = 2000
			}
			if year > 2100 {
				year = 2100
			}
			if month < 1 {
				month = 1
			}
			if month > 12 {
				month = 12
			}
			if day < 1 {
				day = 1
			}
			if day > 28 {
				day = 28 // Safe for all months
			}

			t := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
			path := pathGen.GeneratePathWithTime(tenantID, fileHash, filename, t)
			components, err := pathGen.ParsePath(path)
			if err != nil {
				return false
			}

			expectedYear := t.Format("2006")
			expectedMonth := t.Format("01")
			expectedDay := t.Format("02")

			return components.Year == expectedYear &&
				components.Month == expectedMonth &&
				components.Day == expectedDay
		},
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) <= 64 }),
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) <= 64 }),
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) <= 64 }),
		gen.IntRange(2000, 2100),
		gen.IntRange(1, 12),
		gen.IntRange(1, 28),
	))

	properties.TestingRun(t)
}
