// Package versioning provides API versioning utilities.
package versioning

import (
	"net/http"
	"regexp"
	"strings"
	"time"
)

// Version represents an API version.
type Version struct {
	Major      int
	Minor      int
	Deprecated bool
	SunsetDate *time.Time
}

// String returns the version string.
func (v Version) String() string {
	return "v" + itoa(v.Major)
}

// VersionExtractor extracts version from request.
type VersionExtractor func(*http.Request) string

// PathVersionExtractor extracts version from URL path.
func PathVersionExtractor() VersionExtractor {
	re := regexp.MustCompile(`/v(\d+)`)
	return func(r *http.Request) string {
		matches := re.FindStringSubmatch(r.URL.Path)
		if len(matches) > 1 {
			return "v" + matches[1]
		}
		return ""
	}
}

// HeaderVersionExtractor extracts version from header.
func HeaderVersionExtractor(header string) VersionExtractor {
	return func(r *http.Request) string {
		return r.Header.Get(header)
	}
}

// Router routes requests to versioned handlers.
type Router struct {
	versions   map[string]*Version
	handlers   map[string]http.Handler
	extractor  VersionExtractor
	defaultVer string
}

// NewRouter creates a new version router.
func NewRouter(extractor VersionExtractor) *Router {
	return &Router{
		versions:  make(map[string]*Version),
		handlers:  make(map[string]http.Handler),
		extractor: extractor,
	}
}

// Register registers a version handler.
func (r *Router) Register(version *Version, handler http.Handler) {
	key := version.String()
	r.versions[key] = version
	r.handlers[key] = handler
}

// SetDefault sets the default version.
func (r *Router) SetDefault(version string) {
	r.defaultVer = version
}

// ServeHTTP implements http.Handler.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	version := r.extractor(req)
	if version == "" {
		version = r.defaultVer
	}

	ver, ok := r.versions[version]
	if !ok {
		http.Error(w, "unsupported API version", http.StatusBadRequest)
		return
	}

	// Add deprecation headers
	if ver.Deprecated {
		w.Header().Set("Deprecation", "true")
		if ver.SunsetDate != nil {
			w.Header().Set("Sunset", ver.SunsetDate.Format(http.TimeFormat))
		}
	}

	handler, ok := r.handlers[version]
	if !ok {
		http.Error(w, "version handler not found", http.StatusInternalServerError)
		return
	}

	handler.ServeHTTP(w, req)
}

// ParseVersion parses a version string.
func ParseVersion(s string) *Version {
	s = strings.TrimPrefix(s, "v")
	parts := strings.Split(s, ".")

	v := &Version{}
	if len(parts) > 0 {
		v.Major = atoi(parts[0])
	}
	if len(parts) > 1 {
		v.Minor = atoi(parts[1])
	}
	return v
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

func atoi(s string) int {
	n := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}
	return n
}
