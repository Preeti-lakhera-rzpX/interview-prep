package api

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"kv-cache/internal/cache"
	"kv-cache/internal/model"
)

// Handler serves the cache HTTP API.
type Handler struct {
	cache cache.Cache
}

// NewHandler creates a handler backed by the given cache.
func NewHandler(c cache.Cache) *Handler {
	return &Handler{cache: c}
}

// RegisterRoutes wires all endpoints to the mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /cache/{key}", h.getKey)
	mux.HandleFunc("PUT /cache/{key}", h.setKey)
	mux.HandleFunc("DELETE /cache/{key}", h.deleteKey)
	mux.HandleFunc("GET /health", h.health)
	mux.HandleFunc("GET /stats", h.getStats)
}

func (h *Handler) getKey(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	val, err := h.cache.Get(r.Context(), key)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"key":   key,
		"value": base64.StdEncoding.EncodeToString(val),
	})
}

type setRequest struct {
	Value string `json:"value"`
	TTLMS int64  `json:"ttl_ms"`
}

func (h *Handler) setKey(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")

	var req setRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}

	value, err := base64.StdEncoding.DecodeString(req.Value)
	if err != nil {
		// Allow raw string values as a convenience
		value = []byte(req.Value)
	}

	var ttl time.Duration
	if req.TTLMS > 0 {
		ttl = time.Duration(req.TTLMS) * time.Millisecond
	}

	if err := h.cache.Set(r.Context(), key, value, ttl); err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"key": key, "stored": true})
}

func (h *Handler) deleteKey(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if err := h.cache.Delete(r.Context(), key); err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": true})
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	s := h.cache.Stats(r.Context())
	writeJSON(w, http.StatusOK, map[string]any{
		"status":  "ok",
		"entries": s.Entries,
	})
}

func (h *Handler) getStats(w http.ResponseWriter, r *http.Request) {
	s := h.cache.Stats(r.Context())
	writeJSON(w, http.StatusOK, s)
}

func (h *Handler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, model.ErrNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "key not found"})
	case errors.Is(err, model.ErrInvalidKey):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid key"})
	case errors.Is(err, model.ErrInvalidTTL):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid TTL"})
	default:
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
