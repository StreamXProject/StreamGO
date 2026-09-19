package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"streamgo/internal/config"
	"streamgo/internal/models"
	"streamgo/internal/services"
)

// Mock History Repo
type mockHistRepo struct {
	events   []models.ListeningEventItem
	history  []string
	topPlays []string
}

func (m *mockHistRepo) RecordPlay(ctx context.Context, userID int64, trackID string, source string) error {
	m.history = append([]string{trackID}, m.history...)
	return nil
}

func (m *mockHistRepo) GetUserHistory(ctx context.Context, userID int64, limit int) ([]string, error) {
	return m.history, nil
}

func (m *mockHistRepo) GetUserTopPlayed(ctx context.Context, userID int64, limit int) ([]string, error) {
	return m.topPlays, nil
}

func (m *mockHistRepo) RecordListeningEvents(ctx context.Context, userID int64, events []models.ListeningEventItem) error {
	m.events = append(m.events, events...)
	for _, e := range events {
		m.history = append([]string{e.TrackID}, m.history...)
	}
	return nil
}

// Mock Daily Playlist Repo
type mockDailyRepo struct {
	cache map[string]*models.DailyPlaylistDoc
}

func (m *mockDailyRepo) GetCachedDailyPlaylist(ctx context.Context, docID string) (*models.DailyPlaylistDoc, error) {
	return m.cache[docID], nil
}

func (m *mockDailyRepo) SaveDailyPlaylist(ctx context.Context, doc *models.DailyPlaylistDoc) error {
	if m.cache == nil {
		m.cache = make(map[string]*models.DailyPlaylistDoc)
	}
	m.cache[doc.ID] = doc
	return nil
}

// Mock Access Control Repo
type mockAccessRepo struct {
	policy    *models.AccessPolicy
	invites   map[string]*models.InviteCode
	allowlist map[int64]*models.AllowlistEntry
	bypass    map[int64]*models.AllowlistEntry
	users     map[int64]*models.User
}

func newMockAccessRepo() *mockAccessRepo {
	return &mockAccessRepo{
		policy: &models.AccessPolicy{
			RegistrationMode:  "open",
			EnforceMembership: false,
			RequiredChats:     []models.RequiredChat{},
		},
		invites:   make(map[string]*models.InviteCode),
		allowlist: make(map[int64]*models.AllowlistEntry),
		bypass:    make(map[int64]*models.AllowlistEntry),
		users:     make(map[int64]*models.User),
	}
}

func (m *mockAccessRepo) GetPolicy(ctx context.Context) (*models.AccessPolicy, error) {
	return m.policy, nil
}

func (m *mockAccessRepo) SavePolicy(ctx context.Context, policy *models.AccessPolicy) error {
	m.policy = policy
	return nil
}

func (m *mockAccessRepo) CreateInvite(ctx context.Context, invite *models.InviteCode) error {
	m.invites[invite.Code] = invite
	return nil
}

func (m *mockAccessRepo) GetInvite(ctx context.Context, code string) (*models.InviteCode, error) {
	return m.invites[code], nil
}

func (m *mockAccessRepo) ConsumeInvite(ctx context.Context, code string) error {
	if inv := m.invites[code]; inv != nil {
		inv.UsedCount++
	}
	return nil
}

func (m *mockAccessRepo) RevokeInvite(ctx context.Context, code string) bool {
	if inv := m.invites[code]; inv != nil {
		inv.Revoked = true
		return true
	}
	return false
}

func (m *mockAccessRepo) ListInvites(ctx context.Context, includeDead bool) ([]*models.InviteCode, error) {
	var list []*models.InviteCode
	for _, inv := range m.invites {
		if includeDead || (!inv.Revoked) {
			list = append(list, inv)
		}
	}
	return list, nil
}

func (m *mockAccessRepo) AddToAllowlist(ctx context.Context, entry *models.AllowlistEntry) error {
	m.allowlist[entry.UserID] = entry
	return nil
}

func (m *mockAccessRepo) RemoveFromAllowlist(ctx context.Context, userID int64) bool {
	if _, ok := m.allowlist[userID]; ok {
		delete(m.allowlist, userID)
		return true
	}
	return false
}

func (m *mockAccessRepo) IsInAllowlist(ctx context.Context, userID int64) bool {
	_, ok := m.allowlist[userID]
	return ok
}

func (m *mockAccessRepo) ListAllowlist(ctx context.Context) ([]*models.AllowlistEntry, error) {
	var list []*models.AllowlistEntry
	for _, e := range m.allowlist {
		list = append(list, e)
	}
	return list, nil
}

func (m *mockAccessRepo) AddToBypass(ctx context.Context, entry *models.AllowlistEntry) error {
	m.bypass[entry.UserID] = entry
	return nil
}

func (m *mockAccessRepo) RemoveFromBypass(ctx context.Context, userID int64) bool {
	if _, ok := m.bypass[userID]; ok {
		delete(m.bypass, userID)
		return true
	}
	return false
}

func (m *mockAccessRepo) IsInBypass(ctx context.Context, userID int64) bool {
	_, ok := m.bypass[userID]
	return ok
}

func (m *mockAccessRepo) ListBypass(ctx context.Context) ([]*models.AllowlistEntry, error) {
	var list []*models.AllowlistEntry
	for _, e := range m.bypass {
		list = append(list, e)
	}
	return list, nil
}

func (m *mockAccessRepo) ListUsers(ctx context.Context, query string, status string, limit int, skip int) ([]*models.User, int64, error) {
	var list []*models.User
	for _, u := range m.users {
		list = append(list, u)
	}
	return list, int64(len(list)), nil
}

func (m *mockAccessRepo) SetUserStatus(ctx context.Context, userID int64, status string, reason string) error {
	if u := m.users[userID]; u != nil {
		u.Status = status
	}
	return nil
}

func (m *mockAccessRepo) RevokeUserSessions(ctx context.Context, userID int64) (int, error) {
	if u := m.users[userID]; u != nil {
		u.TokenVersion++
		return u.TokenVersion, nil
	}
	return 1, nil
}

func TestHistoryHandler(t *testing.T) {
	cfg := &config.Config{GuestPassword: "testguestpassword"}
	userRepo := &mockUserRepoForHandler{}
	authSvc := services.NewAuthService(cfg, userRepo)

	trackRepo := &mockTrackRepoForHandler{}
	histRepo := &mockHistRepo{
		topPlays: []string{"trk_1"},
	}
	histSvc := services.NewHistoryService(histRepo, trackRepo)
	handler := NewHistoryHandler(histSvc, authSvc)

	r := chi.NewRouter()
	handler.Routes(r)

	// Create test user and token
	user := &models.User{ID: 100, Username: "testuser"}
	token, _ := authSvc.CreateToken(user)

	// 1. Post listening event
	eventPayload := map[string]any{
		"events": []map[string]any{
			{
				"id":        "ev_1",
				"track_id":  "trk_1",
				"played_at": float64(time.Now().Unix()),
			},
		},
	}
	b, _ := json.Marshal(eventPayload)
	req := httptest.NewRequest("POST", "/me/listening-events", bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for listening events, got %d: %s", rec.Code, rec.Body.String())
	}

	// 2. Get history
	req = httptest.NewRequest("GET", "/history", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for /history, got %d", rec.Code)
	}

	// 3. Get top played
	req = httptest.NewRequest("GET", "/me/top-played", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for /me/top-played, got %d", rec.Code)
	}
}

func TestDailyPlaylistHandler(t *testing.T) {
	cfg := &config.Config{GuestPassword: "testguestpassword"}
	userRepo := &mockUserRepoForHandler{}
	authSvc := services.NewAuthService(cfg, userRepo)

	trackRepo := &mockTrackRepoForHandler{}
	dailyRepo := &mockDailyRepo{cache: make(map[string]*models.DailyPlaylistDoc)}
	dailySvc := services.NewDailyPlaylistService(dailyRepo, trackRepo)
	handler := NewDailyPlaylistHandler(dailySvc, authSvc)

	r := chi.NewRouter()
	handler.Routes(r)

	// 1. Get available playlists
	req := httptest.NewRequest("GET", "/playlists/available", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for /playlists/available, got %d", rec.Code)
	}

	// 2. Get daily playlist
	req = httptest.NewRequest("GET", "/daily-playlist/trending", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for /daily-playlist/trending, got %d", rec.Code)
	}
}

func TestAccessControlHandler(t *testing.T) {
	cfg := &config.Config{
		GuestPassword: "admin_guest_pwd",
		OwnerIDs:      []int64{999},
	}
	userRepo := &mockUserRepoForHandler{}
	authSvc := services.NewAuthService(cfg, userRepo)
	accessRepo := newMockAccessRepo()
	accessSvc := services.NewAccessControlService(accessRepo)

	accessRepo.users[555] = &models.User{ID: 555, Username: "targetuser", Status: "active"}

	handler := NewAccessControlHandler(accessSvc, authSvc, cfg)
	r := chi.NewRouter()
	handler.Routes(r)

	// Admin token
	adminUser := &models.User{ID: 999, Username: "admin"}
	adminToken, _ := authSvc.CreateToken(adminUser)

	// 1. Get policy
	req := httptest.NewRequest("GET", "/admin/access/policy", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for GET /admin/access/policy, got %d", rec.Code)
	}

	// 2. Patch policy to invite
	patchBody := `{"registration_mode": "invite"}`
	req = httptest.NewRequest("PATCH", "/admin/access/policy", bytes.NewBufferString(patchBody))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for PATCH /admin/access/policy, got %d", rec.Code)
	}

	// 3. Create invite
	invBody := `{"max_uses": 5, "ttl_days": 10, "note": "VIP invite"}`
	req = httptest.NewRequest("POST", "/admin/access/invites", bytes.NewBufferString(invBody))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for POST /admin/access/invites, got %d", rec.Code)
	}

	var invResp struct {
		OK     bool              `json:"ok"`
		Invite models.InviteCode `json:"invite"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&invResp)
	if invResp.Invite.Code == "" {
		t.Fatalf("expected generated invite code, got empty")
	}

	// 4. Lock user
	lockBody := `{"reason": "Rule violation", "revoke_sessions": true}`
	req = httptest.NewRequest("POST", "/admin/access/users/555/lock", bytes.NewBufferString(lockBody))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for locking user, got %d", rec.Code)
	}
	if accessRepo.users[555].Status != "locked" {
		t.Fatalf("expected user status locked, got %s", accessRepo.users[555].Status)
	}

	// 5. Reverify
	req = httptest.NewRequest("POST", "/admin/access/reverify", bytes.NewBufferString(`{"user_id": 555}`))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for /admin/access/reverify, got %d", rec.Code)
	}
}

func TestShareHandler(t *testing.T) {
	cfg := &config.Config{}
	trackRepo := &mockTrackRepoForHandler{}
	favRepo := &mockFavRepoForHandler{
		playlists: map[string]*models.UserPlaylist{
			"pl_share": {
				ID:   "pl_share",
				Name: "Chill Vibes",
			},
		},
		tracks: map[string][]string{
			"pl_share": {"trk_share"},
		},
	}

	handler := NewShareHandler(trackRepo, favRepo, cfg)
	r := chi.NewRouter()
	handler.Routes(r)

	// 1. Share track HTML preview
	req := httptest.NewRequest("GET", "/share/trk_share", nil)
	req.Header.Set("Accept", "text/html")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for HTML track share, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !bytes.Contains([]byte(body), []byte("og:title")) {
		t.Fatalf("expected OpenGraph meta tags in HTML preview")
	}

	// 2. Share track JSON metadata
	req = httptest.NewRequest("GET", "/share/trk_share?format=json", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for JSON track share, got %d", rec.Code)
	}

	// 3. Share playlist HTML preview
	req = httptest.NewRequest("GET", "/share/playlists/pl_share", nil)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for playlist share, got %d", rec.Code)
	}
}
