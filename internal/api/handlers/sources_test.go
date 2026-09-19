package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"streamgo/internal/config"
	"streamgo/internal/services"
)

func setupSourcesTestRouter() (*chi.Mux, *services.AccessFilter) {
	cfg := &config.Config{
		ChannelID:     -1003961478268,
		DumpChannelID: -1004300252384,
		SecretKey:     "test-secret",
	}

	filter := services.NewAccessFilter(cfg, nil)
	handler := NewSourcesHandler(cfg, filter, nil)

	r := chi.NewRouter()
	handler.Routes(r)
	return r, filter
}

func TestSourcesOverviewAndModeEndpoints(t *testing.T) {
	router, _ := setupSourcesTestRouter()

	// 1. GET /admin/sources without auth -> 401
	req := httptest.NewRequest(http.MethodGet, "/admin/sources/", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 unauthorized, got %d", w.Code)
	}

	// 2. GET /admin/sources with X-Secret-Key -> 200
	req = httptest.NewRequest(http.MethodGet, "/admin/sources/", nil)
	req.Header.Set("X-Secret-Key", "test-secret")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}

	var overviewResp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &overviewResp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if overviewResp["ok"] != true {
		t.Errorf("expected ok: true")
	}

	// 3. GET /admin/sources/mode
	req = httptest.NewRequest(http.MethodGet, "/admin/sources/mode", nil)
	req.Header.Set("X-Secret-Key", "test-secret")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}

	// 4. POST /admin/sources/mode to "anyone"
	setModeBody := `{"mode": "anyone"}`
	req = httptest.NewRequest(http.MethodPost, "/admin/sources/mode", bytes.NewBufferString(setModeBody))
	req.Header.Set("X-Secret-Key", "test-secret")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}

	var modeResp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &modeResp)
	if modeResp["filter_mode_name"] != "anyone" {
		t.Errorf("expected filter_mode_name anyone, got %v", modeResp["filter_mode_name"])
	}
}

func TestSourcesAllowedAndBannedEndpoints(t *testing.T) {
	router, _ := setupSourcesTestRouter()

	// 1. POST /admin/sources/allowed
	allowedBody := `{"source_id": -100123456789, "source_type": "channel", "name": "Test Channel"}`
	req := httptest.NewRequest(http.MethodPost, "/admin/sources/allowed", bytes.NewBufferString(allowedBody))
	req.Header.Set("X-Secret-Key", "test-secret")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}

	// 2. GET /admin/sources/allowed
	req = httptest.NewRequest(http.MethodGet, "/admin/sources/allowed", nil)
	req.Header.Set("X-Secret-Key", "test-secret")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}

	// 3. DELETE /admin/sources/allowed/-100123456789
	req = httptest.NewRequest(http.MethodDelete, "/admin/sources/allowed/-100123456789", nil)
	req.Header.Set("X-Secret-Key", "test-secret")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}

	// 4. POST /admin/sources/banned
	banBody := `{"source_id": 987654321, "source_type": "user", "name": "Bad Actor", "reason": "Spamming"}`
	req = httptest.NewRequest(http.MethodPost, "/admin/sources/banned", bytes.NewBufferString(banBody))
	req.Header.Set("X-Secret-Key", "test-secret")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}

	// 5. GET /admin/sources/banned
	req = httptest.NewRequest(http.MethodGet, "/admin/sources/banned", nil)
	req.Header.Set("X-Secret-Key", "test-secret")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}

	// 6. DELETE /admin/sources/banned/987654321
	req = httptest.NewRequest(http.MethodDelete, "/admin/sources/banned/987654321", nil)
	req.Header.Set("X-Secret-Key", "test-secret")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}
}
