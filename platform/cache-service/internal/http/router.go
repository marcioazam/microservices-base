package http

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/auth-platform/cache-service/internal/auth"
	"github.com/auth-platform/cache-service/internal/cache"
	libhttp "github.com/authcorp/libs/go/src/http"
)

// RouterConfig holds router configuration.
type RouterConfig struct {
	MetricsEnabled bool
	MetricsPath    string
	AuthMiddleware *auth.Middleware
	RequestTimeout time.Duration
}

// NewRouter creates a new HTTP router.
func NewRouter(cacheService cache.Service, cfg RouterConfig) http.Handler {
	r := chi.NewRouter()

	// Chi middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5))

	// Lib middleware for timeout
	if cfg.RequestTimeout > 0 {
		r.Use(libhttp.TimeoutMiddleware(cfg.RequestTimeout))
	}

	handler := NewHandler(cacheService)

	// Health endpoints using lib health handler
	healthHandler := libhttp.NewHealthHandler()
	healthHandler.Register("cache", func() error {
		status, err := cacheService.Health(context.Background())
		if err != nil {
			return err
		}
		if !status.Healthy {
			return errors.New("cache service unhealthy")
		}
		return nil
	})

	r.Get("/health", healthHandler.LivenessHandler())
	r.Get("/ready", healthHandler.ReadinessHandler())

	// Metrics endpoint (no auth)
	if cfg.MetricsEnabled {
		r.Handle(cfg.MetricsPath, promhttp.Handler())
	}

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		// Apply auth middleware if configured
		if cfg.AuthMiddleware != nil {
			r.Use(cfg.AuthMiddleware.Authenticate)
		}

		r.Route("/cache/{namespace}", func(r chi.Router) {
			// Single key operations
			r.Get("/{key}", handler.Get)
			r.Put("/{key}", handler.Set)
			r.Delete("/{key}", handler.Delete)

			// Batch operations
			r.Post("/batch/get", handler.BatchGet)
			r.Post("/batch/set", handler.BatchSet)
		})
	})

	return r
}
