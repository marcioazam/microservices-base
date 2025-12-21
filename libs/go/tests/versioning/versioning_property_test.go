package versioning_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/auth-platform/libs/go/versioning"
	"pgregory.net/rapid"
)

// TestVersionStringFormat verifies version string format.
func TestVersionStringFormat(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		major := rapid.IntRange(1, 100).Draw(t, "major")
		minor := rapid.IntRange(0, 100).Draw(t, "minor")

		v := versioning.Version{Major: major, Minor: minor}
		str := v.String()

		if str[0] != 'v' {
			t.Errorf("version string should start with 'v', got %s", str)
		}
	})
}

// TestParseVersionRoundtrip verifies parse/string roundtrip.
func TestParseVersionRoundtrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		major := rapid.IntRange(1, 99).Draw(t, "major")

		v := &versioning.Version{Major: major}
		str := v.String()

		parsed := versioning.ParseVersion(str)
		if parsed.Major != major {
			t.Errorf("major mismatch: %d != %d", parsed.Major, major)
		}
	})
}

// TestPathVersionExtractor verifies path version extraction.
func TestPathVersionExtractor(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		major := rapid.IntRange(1, 9).Draw(t, "major")
		path := rapid.StringMatching(`/[a-z]+`).Draw(t, "path")

		extractor := versioning.PathVersionExtractor()
		req := httptest.NewRequest("GET", "/v"+itoa(major)+path, nil)

		version := extractor(req)
		expected := "v" + itoa(major)

		if version != expected {
			t.Errorf("expected %s, got %s", expected, version)
		}
	})
}

// TestHeaderVersionExtractor verifies header version extraction.
func TestHeaderVersionExtractor(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		headerName := rapid.StringMatching(`X-[A-Z][a-z]+-Version`).Draw(t, "headerName")
		version := rapid.StringMatching(`v[1-9]`).Draw(t, "version")

		extractor := versioning.HeaderVersionExtractor(headerName)
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set(headerName, version)

		extracted := extractor(req)
		if extracted != version {
			t.Errorf("expected %s, got %s", version, extracted)
		}
	})
}

// TestRouterVersionRouting verifies router routes to correct handler.
func TestRouterVersionRouting(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		major := rapid.IntRange(1, 5).Draw(t, "major")

		router := versioning.NewRouter(versioning.PathVersionExtractor())

		var handledVersion int
		for i := 1; i <= 5; i++ {
			v := i
			router.Register(&versioning.Version{Major: v}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				handledVersion = v
			}))
		}

		req := httptest.NewRequest("GET", "/v"+itoa(major)+"/test", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if handledVersion != major {
			t.Errorf("expected handler v%d, got v%d", major, handledVersion)
		}
	})
}

// TestRouterDefaultVersion verifies default version fallback.
func TestRouterDefaultVersion(t *testing.T) {
	router := versioning.NewRouter(versioning.PathVersionExtractor())

	var handled bool
	router.Register(&versioning.Version{Major: 1}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handled = true
	}))
	router.SetDefault("v1")

	// Request without version in path
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if !handled {
		t.Error("default version handler should be called")
	}
}

// TestRouterDeprecationHeaders verifies deprecation headers are set.
func TestRouterDeprecationHeaders(t *testing.T) {
	router := versioning.NewRouter(versioning.PathVersionExtractor())

	sunset := time.Now().Add(30 * 24 * time.Hour)
	router.Register(&versioning.Version{
		Major:      1,
		Deprecated: true,
		SunsetDate: &sunset,
	}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	req := httptest.NewRequest("GET", "/v1/test", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Header().Get("Deprecation") != "true" {
		t.Error("deprecation header should be set")
	}

	if rec.Header().Get("Sunset") == "" {
		t.Error("sunset header should be set")
	}
}

// TestRouterUnsupportedVersion verifies unsupported version returns error.
func TestRouterUnsupportedVersion(t *testing.T) {
	router := versioning.NewRouter(versioning.PathVersionExtractor())

	router.Register(&versioning.Version{Major: 1}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	req := httptest.NewRequest("GET", "/v99/test", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	result := ""
	for n > 0 {
		result = string(rune('0'+n%10)) + result
		n /= 10
	}
	return result
}
