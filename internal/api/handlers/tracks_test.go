package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"streamgo/internal/api/middleware"
	"streamgo/internal/config"
	"streamgo/internal/models"
	"streamgo/internal/services"
)

type mockTrackRepoForHandler struct {
	track        *models.Track
	softDeleted  bool
	hardDeleted  bool
	artRefCounts map[string]int64
}

func (m *mockTrackRepoForHandler) GetByID(ctx context.Context, id string) (*models.Track, error) {
	if m.track != nil && m.track.ID == id {
		if m.softDeleted || m.hardDeleted {
			return nil, errors.New("track not found")
		}
		return m.track, nil
	}
	return nil, errors.New("track not found")
}

func (m *mockTrackRepoForHandler) GetByIDs(ctx context.Context, ids []string) ([]*models.Track, error) {
	return nil, nil
}
func (m *mockTrackRepoForHandler) List(ctx context.Context, page, perPage int, sortField, topicName string, channelID int64) ([]*models.Track, int64, error) {
	return nil, 0, nil
}
func (m *mockTrackRepoForHandler) Search(ctx context.Context, query string, limit int) ([]*models.Track, error) {
	return nil, nil
}
func (m *mockTrackRepoForHandler) Random(ctx context.Context, limit int, channelID int64) ([]*models.Track, error) {
	return nil, nil
}
func (m *mockTrackRepoForHandler) GetTopics(ctx context.Context, channelID int64, limit int) ([]*models.TopicItem, error) {
	return nil, nil
}
func (m *mockTrackRepoForHandler) GetChannelIDs(ctx context.Context) ([]int64, error) {
	return nil, nil
}
func (m *mockTrackRepoForHandler) IncrementPlayCount(ctx context.Context, id string) error {
	return nil
}
func (m *mockTrackRepoForHandler) UpdateWorkerFileID(ctx context.Context, trackID, workerID, fileID string) error {
	return nil
}
func (m *mockTrackRepoForHandler) UpdateLyricsCache(ctx context.Context, id string, text, kind, source string, telegraphURL string) error {
	return nil
}
func (m *mockTrackRepoForHandler) SoftDelete(ctx context.Context, id string) error {
	if m.track != nil && m.track.ID == id {
		m.softDeleted = true
		return nil
	}
	return errors.New("track not found")
}
func (m *mockTrackRepoForHandler) HardDelete(ctx context.Context, id string) error {
	if m.track != nil && m.track.ID == id {
		m.hardDeleted = true
		return nil
	}
	return errors.New("track not found")
}
func (m *mockTrackRepoForHandler) CountArtworkReferences(ctx context.Context, excludeTrackID, coverURL string) (int64, error) {
	if m.artRefCounts != nil {
		return m.artRefCounts[coverURL], nil
	}
	return 0, nil
}

func setupTestRouter(handler *TrackHandler) *chi.Mux {
	r := chi.NewRouter()
	handler.Routes(r)
	return r
}

func TestTrackHandler_Delete_AdminSuccess(t *testing.T) {
	repo := &mockTrackRepoForHandler{
		track: &models.Track{
			ID: "trk_del_1",
		},
	}
	trackSvc := services.NewTrackService(repo)
	cfg := &config.Config{
		OwnerIDs: []int64{111},
	}
	h := NewTrackHandler(trackSvc, nil, cfg)
	router := setupTestRouter(h)

	req := httptest.NewRequest(http.MethodDelete, "/tracks/trk_del_1", nil)
	// Inject Owner ID
	ctx := middleware.WithUserID(req.Context(), 111)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["ok"] != true || resp["mode"] != "soft" {
		t.Fatalf("unexpected response payload: %v", resp)
	}
	if !repo.softDeleted {
		t.Fatal("expected track to be soft deleted")
	}
}

func TestTrackHandler_Delete_HardDeleteOption(t *testing.T) {
	repo := &mockTrackRepoForHandler{
		track: &models.Track{
			ID: "trk_del_hard",
		},
	}
	trackSvc := services.NewTrackService(repo)
	cfg := &config.Config{
		SudoUsers: []int64{222},
	}
	h := NewTrackHandler(trackSvc, nil, cfg)
	router := setupTestRouter(h)

	req := httptest.NewRequest(http.MethodDelete, "/tracks/trk_del_hard?hard=true", nil)
	ctx := middleware.WithUserID(req.Context(), 222)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["mode"] != "hard" {
		t.Fatalf("expected mode hard, got %v", resp["mode"])
	}
	if !repo.hardDeleted {
		t.Fatal("expected track to be hard deleted")
	}
}

func TestTrackHandler_Delete_ForbiddenNonAdmin(t *testing.T) {
	repo := &mockTrackRepoForHandler{
		track: &models.Track{ID: "trk_1"},
	}
	trackSvc := services.NewTrackService(repo)
	cfg := &config.Config{
		OwnerIDs: []int64{111},
	}
	h := NewTrackHandler(trackSvc, nil, cfg)
	router := setupTestRouter(h)

	req := httptest.NewRequest(http.MethodDelete, "/tracks/trk_1", nil)
	// Regular non-admin user
	ctx := middleware.WithUserID(req.Context(), 999)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403 Forbidden, got %d", rec.Code)
	}
}

func TestTrackHandler_Delete_UnauthorizedNoUser(t *testing.T) {
	repo := &mockTrackRepoForHandler{
		track: &models.Track{ID: "trk_1"},
	}
	trackSvc := services.NewTrackService(repo)
	cfg := &config.Config{
		OwnerIDs: []int64{111},
	}
	h := NewTrackHandler(trackSvc, nil, cfg)
	router := setupTestRouter(h)

	req := httptest.NewRequest(http.MethodDelete, "/tracks/trk_1", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401 Unauthorized, got %d", rec.Code)
	}
}

func TestTrackHandler_Delete_NotFound(t *testing.T) {
	repo := &mockTrackRepoForHandler{
		track: nil,
	}
	trackSvc := services.NewTrackService(repo)
	cfg := &config.Config{
		OwnerIDs: []int64{111},
	}
	h := NewTrackHandler(trackSvc, nil, cfg)
	router := setupTestRouter(h)

	req := httptest.NewRequest(http.MethodDelete, "/tracks/trk_missing", nil)
	ctx := middleware.WithUserID(req.Context(), 111)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404 Not Found, got %d: %s", rec.Code, rec.Body.String())
	}
}

