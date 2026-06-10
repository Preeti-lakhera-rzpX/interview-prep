package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"kv-cache/internal/cache"
	"kv-cache/internal/model"
)

func setupTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	cfg := model.Config{
		MaxEntries:     1000,
		EvictionPolicy: model.PolicyLRU,
		ShardCount:     4,
		VirtualNodes:   32,
	}
	c, err := cache.New(cfg)
	if err != nil {
		t.Fatalf("cache.New: %v", err)
	}
	t.Cleanup(func() { c.Close() })

	h := NewHandler(c)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	return httptest.NewServer(mux)
}

func TestHandler_SetAndGet(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	// Set
	body := `{"value":"aGVsbG8=","ttl_ms":60000}`
	resp, err := http.NewRequest("PUT", srv.URL+"/cache/greeting", bytes.NewBufferString(body))
	if err != nil {
		t.Fatal(err)
	}
	resp.Header.Set("Content-Type", "application/json")
	putResp, err := http.DefaultClient.Do(resp)
	if err != nil {
		t.Fatal(err)
	}
	if putResp.StatusCode != http.StatusCreated {
		t.Fatalf("PUT status = %d, want %d", putResp.StatusCode, http.StatusCreated)
	}

	// Get
	getResp, err := http.Get(srv.URL + "/cache/greeting")
	if err != nil {
		t.Fatal(err)
	}
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("GET status = %d, want %d", getResp.StatusCode, http.StatusOK)
	}

	var result map[string]any
	json.NewDecoder(getResp.Body).Decode(&result)
	if result["key"] != "greeting" {
		t.Errorf("key = %v, want 'greeting'", result["key"])
	}
	if result["value"] != "aGVsbG8=" {
		t.Errorf("value = %v, want 'aGVsbG8='", result["value"])
	}
}

func TestHandler_GetNotFound(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/cache/nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestHandler_Delete(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	// Set a key first
	body := `{"value":"dGVzdA=="}`
	req, _ := http.NewRequest("PUT", srv.URL+"/cache/todelete", bytes.NewBufferString(body))
	http.DefaultClient.Do(req)

	// Delete
	req, _ = http.NewRequest("DELETE", srv.URL+"/cache/todelete", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("DELETE status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	// Verify gone
	getResp, _ := http.Get(srv.URL + "/cache/todelete")
	if getResp.StatusCode != http.StatusNotFound {
		t.Errorf("GET after DELETE status = %d, want %d", getResp.StatusCode, http.StatusNotFound)
	}
}

func TestHandler_DeleteNotFound(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	req, _ := http.NewRequest("DELETE", srv.URL+"/cache/ghost", nil)
	resp, _ := http.DefaultClient.Do(req)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestHandler_Health(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)
	if result["status"] != "ok" {
		t.Errorf("status = %v, want 'ok'", result["status"])
	}
}

func TestHandler_Stats(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	// Generate some activity
	body := `{"value":"eA=="}`
	req, _ := http.NewRequest("PUT", srv.URL+"/cache/x", bytes.NewBufferString(body))
	http.DefaultClient.Do(req)
	http.Get(srv.URL + "/cache/x")    // hit
	http.Get(srv.URL + "/cache/miss") // miss

	resp, _ := http.Get(srv.URL + "/stats")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var s model.Stats
	json.NewDecoder(resp.Body).Decode(&s)
	if s.Hits < 1 {
		t.Errorf("Hits = %d, want >= 1", s.Hits)
	}
	if s.Misses < 1 {
		t.Errorf("Misses = %d, want >= 1", s.Misses)
	}
}

func TestHandler_InvalidJSON(t *testing.T) {
	srv := setupTestServer(t)
	defer srv.Close()

	req, _ := http.NewRequest("PUT", srv.URL+"/cache/bad", bytes.NewBufferString("not json"))
	resp, _ := http.DefaultClient.Do(req)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}
