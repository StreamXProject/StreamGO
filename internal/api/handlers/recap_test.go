package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"streamgo/internal/config"
	"streamgo/internal/models"
	"streamgo/internal/services"
)

// mockRecapRepo implements repository.RecapRepository for testing handlers.
type mockRecapRepo struct {
	events       []models.ListeningEventItem
	publicShare  *models.RecapPublicSummary
	shares       []*models.RecapShareItem
	revokeResult bool
}

func (m *mockRecapRepo) RecordEvents(ctx context.Context, userID int64, events []models.ListeningEventItem) (int, error) {
	m.events = append(m.events, events...)
	return len(events), nil
}

func (m *mockRecapRepo) LoadEvents(ctx context.Context, userID int64, start float64, end float64) ([]*models.ListeningEventDoc, error) {
	return nil, nil
}

func (m *mockRecapRepo) FirstEventTimestamp(ctx context.Context, userID int64) (float64, bool, error) {
	return 0, false, nil
}

func (m *mockRecapRepo) HasPlays(ctx context.Context, userID int64, start float64, end float64) (bool, error) {
	return false, nil
}

func (m *mockRecapRepo) FirstSeenBefore(ctx context.Context, userID int64, trackIDs []string, before float64) (map[string]bool, error) {
	return make(map[string]bool), nil
}

func (m *mockRecapRepo) GetSnapshot(ctx context.Context, userID int64, ptype models.RecapPeriodType, period string, tz int) (*models.RecapSnapshot, float64, error) {
	return nil, 0, nil
}

func (m *mockRecapRepo) SaveSnapshot(ctx context.Context, userID int64, ptype models.RecapPeriodType, period string, tz int, snap *models.RecapSnapshot) error {
	return nil
}

func (m *mockRecapRepo) CreateShare(ctx context.Context, userID int64, ptype models.RecapPeriodType, period string, summary *models.RecapPublicSummary) (*models.RecapShareItem, error) {
	return &models.RecapShareItem{
		Token:  "test_token_123",
		Type:   ptype,
		Period: period,
	}, nil
}

func (m *mockRecapRepo) ListShares(ctx context.Context, userID int64) ([]*models.RecapShareItem, error) {
	return m.shares, nil
}

func (m *mockRecapRepo) RevokeShare(ctx context.Context, userID int64, token string) (bool, error) {
	return m.revokeResult, nil
}

func (m *mockRecapRepo) GetPublicShare(ctx context.Context, token string) (*models.RecapPublicSummary, error) {
	return m.publicShare, nil
}

func (m *mockRecapRepo) DeleteUserData(ctx context.Context, userID int64) (int64, int64, int64, error) {
	return 5, 2, 1, nil
}

func setupRecapTestRouter(mock *mockRecapRepo) (http.Handler, string) {
	recapSvc := services.NewRecapService(mock, nil)
	authSvc := services.NewAuthService(&config.Config{SecretKey: "test_recap_secret"}, nil)
	handler := NewRecapHandler(recapSvc, authSvc)

	token, _ := authSvc.CreateToken(&models.User{
		ID:       12345,
		Username: "recapuser",
	})

	r := chi.NewRouter()
	handler.Routes(r)
	return r, token
}

func TestRecapHandlerPublicShare(t *testing.T) {
	mock := &mockRecapRepo{
		publicShare: &models.RecapPublicSummary{
			Type:         models.RecapPeriodMonthly,
			Period:       "2026-09",
			Label:        "September 2026",
			TotalMinutes: 120,
			TotalPlays:   30,
		},
	}
	router, _ := setupRecapTestRouter(mock)

	// 1. Existing public share link
	req := httptest.NewRequest("GET", "/recaps/share/test_token_123", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rr.Code, rr.Body.String())
	}
	var res models.RecapPublicSummary
	if err := json.Unmarshal(rr.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if res.Period != "2026-09" || res.TotalMinutes != 120 {
		t.Errorf("unexpected public summary: %+v", res)
	}

	// 2. Missing/expired share link
	mock.publicShare = nil
	req404 := httptest.NewRequest("GET", "/recaps/share/expired_token", nil)
	rr404 := httptest.NewRecorder()
	router.ServeHTTP(rr404, req404)

	if rr404.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found, got %d", rr404.Code)
	}
}

func TestRecapHandlerUnauthenticated(t *testing.T) {
	mock := &mockRecapRepo{}
	router, _ := setupRecapTestRouter(mock)

	// GET /me/recaps without user context -> 401
	req := httptest.NewRequest("GET", "/me/recaps", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized, got %d", rr.Code)
	}

	// POST /me/listening-events without user context -> 401
	reqPost := httptest.NewRequest("POST", "/me/listening-events", strings.NewReader(`{"events":[]}`))
	rrPost := httptest.NewRecorder()
	router.ServeHTTP(rrPost, reqPost)

	if rrPost.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized for /me/listening-events, got %d", rrPost.Code)
	}
}

func TestRecapHandlerAuthenticatedFlows(t *testing.T) {
	mock := &mockRecapRepo{
		revokeResult: true,
	}
	router, token := setupRecapTestRouter(mock)

	// Helper to attach Auth token header
	authReq := func(method, target string, body string) *http.Request {
		r := httptest.NewRequest(method, target, strings.NewReader(body))
		r.Header.Set("Authorization", "Bearer "+token)
		return r
	}

	// 1. POST /me/listening-events
	evPayload := `{"events":[{"id":"e1","track_id":"t1","played_at":1700000000,"played_ms":45000,"duration_ms":120000}]}`
	reqEv := authReq("POST", "/me/listening-events", evPayload)
	rrEv := httptest.NewRecorder()
	router.ServeHTTP(rrEv, reqEv)

	if rrEv.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rrEv.Code, rrEv.Body.String())
	}
	var evRes map[string]any
	_ = json.Unmarshal(rrEv.Body.Bytes(), &evRes)
	if evRes["ok"] != true || int(evRes["stored"].(float64)) != 1 {
		t.Errorf("expected ok=true, stored=1, got %+v", evRes)
	}

	// 2. DELETE /me/recaps/shares/token123
	reqRevoke := authReq("DELETE", "/me/recaps/shares/token123", "")
	rrRevoke := httptest.NewRecorder()
	router.ServeHTTP(rrRevoke, reqRevoke)

	if rrRevoke.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on revoke, got %d: %s", rrRevoke.Code, rrRevoke.Body.String())
	}

	// 3. DELETE /me/recaps/data
	reqDel := authReq("DELETE", "/me/recaps/data", "")
	rrDel := httptest.NewRecorder()
	router.ServeHTTP(rrDel, reqDel)

	if rrDel.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on delete data, got %d: %s", rrDel.Code, rrDel.Body.String())
	}
	var delRes map[string]any
	_ = json.Unmarshal(rrDel.Body.Bytes(), &delRes)
	if delRes["ok"] != true || int(delRes["events"].(float64)) != 5 {
		t.Errorf("expected ok=true, events=5, got %+v", delRes)
	}

	// 4. Invalid period type
	reqInv := authReq("GET", "/me/recaps/daily/2026-09-21", "")
	rrInv := httptest.NewRecorder()
	router.ServeHTTP(rrInv, reqInv)

	if rrInv.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request for invalid period type, got %d", rrInv.Code)
	}
}
