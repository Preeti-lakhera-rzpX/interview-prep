package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"interview-prep/internal/dedup"
	"interview-prep/internal/model"
	"interview-prep/internal/queue"
	"interview-prep/internal/ratelimit"
	"interview-prep/internal/service"
	"interview-prep/internal/store"
)

func setupTestServer() (*httptest.Server, *queue.PriorityQueue) {
	st := store.NewMemory()
	dd := dedup.New(1 * time.Minute)
	rl := ratelimit.New(ratelimit.Config{Rate: 100, Capacity: 100})
	q := queue.New(queue.Config{CriticalSize: 100, HighSize: 100, NormalSize: 100, LowSize: 100})
	svc := service.NewNotifier(st, dd, rl, q)
	h := NewHandler(svc, st, q)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	return httptest.NewServer(mux), q
}

func TestHandler_SubmitNotification(t *testing.T) {
	srv, _ := setupTestServer()
	defer srv.Close()

	body := `{"user_id":"u1","channel":"email","priority":2,"payload":{"to":"a@b.com","body":"hi"}}`
	resp, err := http.Post(srv.URL+"/notifications", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		t.Errorf("status: got %d, want %d", resp.StatusCode, http.StatusAccepted)
	}

	var result map[string]string
	_ = json.NewDecoder(resp.Body).Decode(&result)
	if result["id"] == "" {
		t.Error("expected notification ID in response")
	}
}

func TestHandler_SubmitInvalidBody(t *testing.T) {
	srv, _ := setupTestServer()
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/notifications", "application/json", bytes.NewBufferString("not json"))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status: got %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandler_SubmitValidationError(t *testing.T) {
	srv, _ := setupTestServer()
	defer srv.Close()

	body := `{"user_id":"","channel":"email","payload":{"to":"x","body":"y"}}`
	resp, err := http.Post(srv.URL+"/notifications", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status: got %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandler_GetNotification(t *testing.T) {
	srv, _ := setupTestServer()
	defer srv.Close()

	// Submit first
	body := `{"user_id":"u1","channel":"push","payload":{"to":"token123","body":"hey"}}`
	postResp, _ := http.Post(srv.URL+"/notifications", "application/json", bytes.NewBufferString(body))
	var created map[string]string
	_ = json.NewDecoder(postResp.Body).Decode(&created)
	postResp.Body.Close()

	// Get
	getResp, err := http.Get(srv.URL + "/notifications/" + created["id"])
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer getResp.Body.Close()

	if getResp.StatusCode != http.StatusOK {
		t.Errorf("status: got %d, want %d", getResp.StatusCode, http.StatusOK)
	}

	var n model.Notification
	_ = json.NewDecoder(getResp.Body).Decode(&n)
	if n.ID != created["id"] {
		t.Errorf("GET: got id=%s, want %s", n.ID, created["id"])
	}
}

func TestHandler_GetNotFound(t *testing.T) {
	srv, _ := setupTestServer()
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/notifications/nonexistent")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status: got %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestHandler_Health(t *testing.T) {
	srv, _ := setupTestServer()
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/health")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status: got %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var result map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&result)
	if result["status"] != "ok" {
		t.Errorf("health: got status=%v", result["status"])
	}
}
