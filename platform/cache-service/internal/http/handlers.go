// Package http provides HTTP REST API handlers.
package http

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/auth-platform/cache-service/internal/cache"
)

// Handler provides HTTP handlers for cache operations.
type Handler struct {
	cacheService cache.Service
}

// NewHandler creates a new HTTP handler.
func NewHandler(cacheService cache.Service) *Handler {
	return &Handler{cacheService: cacheService}
}

// GetRequest represents a GET request body (for batch).
type GetRequest struct {
	Keys []string `json:"keys"`
}

// SetRequest represents a SET request body.
type SetRequest struct {
	Value      []byte `json:"value"`
	TTLSeconds int64  `json:"ttl_seconds,omitempty"`
	Encrypt    bool   `json:"encrypt,omitempty"`
}

// BatchSetRequest represents a batch SET request body.
type BatchSetRequest struct {
	Entries    map[string][]byte `json:"entries"`
	TTLSeconds int64             `json:"ttl_seconds,omitempty"`
}

// GetResponse represents a GET response.
type GetResponse struct {
	Found  bool   `json:"found"`
	Value  []byte `json:"value,omitempty"`
	Source string `json:"source,omitempty"`
}

// SetResponse represents a SET response.
type SetResponse struct {
	Success bool `json:"success"`
}

// DeleteResponse represents a DELETE response.
type DeleteResponse struct {
	Deleted bool `json:"deleted"`
}

// BatchGetResponse represents a batch GET response.
type BatchGetResponse struct {
	Values      map[string][]byte `json:"values"`
	MissingKeys []string          `json:"missing_keys,omitempty"`
}

// BatchSetResponse represents a batch SET response.
type BatchSetResponse struct {
	Success     bool `json:"success"`
	StoredCount int  `json:"stored_count"`
}

// HealthResponse represents a health check response.
type HealthResponse struct {
	Healthy           bool   `json:"healthy"`
	RedisStatus       string `json:"redis_status"`
	BrokerStatus      string `json:"broker_status,omitempty"`
	LocalCacheEnabled bool   `json:"local_cache_enabled"`
}

// Get handles GET /api/v1/cache/{namespace}/{key}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	namespace := chi.URLParam(r, "namespace")
	key := chi.URLParam(r, "key")

	if namespace == "" || key == "" {
		WriteBadRequest(w, r, "namespace and key are required")
		return
	}

	entry, err := h.cacheService.Get(r.Context(), namespace, key)
	if err != nil {
		if cache.IsNotFound(err) {
			writeJSON(w, http.StatusOK, GetResponse{Found: false})
			return
		}
		WriteError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, GetResponse{
		Found:  true,
		Value:  entry.Value,
		Source: entry.Source.String(),
	})
}

// Set handles PUT /api/v1/cache/{namespace}/{key}
func (h *Handler) Set(w http.ResponseWriter, r *http.Request) {
	namespace := chi.URLParam(r, "namespace")
	key := chi.URLParam(r, "key")

	if namespace == "" || key == "" {
		WriteBadRequest(w, r, "namespace and key are required")
		return
	}

	var req SetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteBadRequest(w, r, "invalid request body")
		return
	}

	if len(req.Value) == 0 {
		WriteBadRequest(w, r, "value is required")
		return
	}

	ttl := time.Duration(req.TTLSeconds) * time.Second

	var opts []cache.SetOption
	if req.Encrypt {
		opts = append(opts, cache.WithEncryption())
	}

	err := h.cacheService.Set(r.Context(), namespace, key, req.Value, ttl, opts...)
	if err != nil {
		WriteError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, SetResponse{Success: true})
}

// Delete handles DELETE /api/v1/cache/{namespace}/{key}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	namespace := chi.URLParam(r, "namespace")
	key := chi.URLParam(r, "key")

	if namespace == "" || key == "" {
		WriteBadRequest(w, r, "namespace and key are required")
		return
	}

	deleted, err := h.cacheService.Delete(r.Context(), namespace, key)
	if err != nil {
		WriteError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, DeleteResponse{Deleted: deleted})
}

// BatchGet handles POST /api/v1/cache/{namespace}/batch/get
func (h *Handler) BatchGet(w http.ResponseWriter, r *http.Request) {
	namespace := chi.URLParam(r, "namespace")

	if namespace == "" {
		WriteBadRequest(w, r, "namespace is required")
		return
	}

	var req GetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteBadRequest(w, r, "invalid request body")
		return
	}

	if len(req.Keys) == 0 {
		writeJSON(w, http.StatusOK, BatchGetResponse{Values: map[string][]byte{}})
		return
	}

	found, missing, err := h.cacheService.BatchGet(r.Context(), namespace, req.Keys)
	if err != nil {
		WriteError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, BatchGetResponse{
		Values:      found,
		MissingKeys: missing,
	})
}

// BatchSet handles POST /api/v1/cache/{namespace}/batch/set
func (h *Handler) BatchSet(w http.ResponseWriter, r *http.Request) {
	namespace := chi.URLParam(r, "namespace")

	if namespace == "" {
		WriteBadRequest(w, r, "namespace is required")
		return
	}

	var req BatchSetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteBadRequest(w, r, "invalid request body")
		return
	}

	if len(req.Entries) == 0 {
		writeJSON(w, http.StatusOK, BatchSetResponse{Success: true, StoredCount: 0})
		return
	}

	ttl := time.Duration(req.TTLSeconds) * time.Second

	count, err := h.cacheService.BatchSet(r.Context(), namespace, req.Entries, ttl)
	if err != nil {
		WriteError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, BatchSetResponse{
		Success:     true,
		StoredCount: count,
	})
}

// Health handles GET /health
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	status, err := h.cacheService.Health(r.Context())
	if err != nil {
		WriteInternalError(w, r, "health check failed")
		return
	}

	httpStatus := http.StatusOK
	if !status.Healthy {
		httpStatus = http.StatusServiceUnavailable
	}

	writeJSON(w, httpStatus, HealthResponse{
		Healthy:           status.Healthy,
		RedisStatus:       status.RedisStatus,
		BrokerStatus:      status.BrokerStatus,
		LocalCacheEnabled: status.LocalCache,
	})
}

// Ready handles GET /ready
func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	status, err := h.cacheService.Health(r.Context())
	if err != nil || !status.Healthy {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data) // Error intentionally ignored - response already committed
}
