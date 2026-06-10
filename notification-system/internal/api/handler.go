package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"interview-prep/internal/model"
	"interview-prep/internal/queue"
	"interview-prep/internal/service"
	"interview-prep/internal/store"
)

// Handler holds HTTP route handlers.
type Handler struct {
	service *service.Notifier
	store   store.Store
	queue   *queue.PriorityQueue
}

// NewHandler creates an HTTP handler with its dependencies.
func NewHandler(svc *service.Notifier, st store.Store, q *queue.PriorityQueue) *Handler {
	return &Handler{service: svc, store: st, queue: q}
}

// RegisterRoutes wires all API routes onto the given mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /notifications", h.submitNotification)
	mux.HandleFunc("GET /notifications/{id}", h.getNotification)
	mux.HandleFunc("GET /health", h.health)
}

func (h *Handler) submitNotification(w http.ResponseWriter, r *http.Request) {
	var req service.SubmitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}

	resp, err := h.service.Submit(r.Context(), req)
	if err != nil {
		h.handleError(w, err)
		return
	}

	writeJSON(w, http.StatusAccepted, resp)
}

func (h *Handler) getNotification(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing notification id"})
		return
	}

	n, err := h.store.Get(r.Context(), id)
	if err != nil {
		h.handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, n)
}

func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":      "ok",
		"queue_depth": h.queue.Len(),
	})
}

func (h *Handler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, model.ErrNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
	case errors.Is(err, model.ErrRateLimited):
		w.Header().Set("Retry-After", "1")
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "rate limit exceeded"})
	case errors.Is(err, model.ErrQueueFull):
		w.Header().Set("Retry-After", "5")
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "service at capacity"})
	case errors.Is(err, model.ErrDuplicate):
		writeJSON(w, http.StatusConflict, map[string]string{"error": "duplicate notification"})
	case errors.Is(err, model.ErrInvalidInput):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": errMsg(err)})
	default:
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func errMsg(err error) string {
	msg := err.Error()
	if i := strings.Index(msg, ": "); i >= 0 {
		return msg[i+2:]
	}
	return msg
}
