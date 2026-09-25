package handlers

import (
	"context"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"streamgo/internal/config"
	"streamgo/internal/models"
	"streamgo/internal/services"
)

type mockStreamTrackRepo struct {
	track *models.Track
}

func (m *mockStreamTrackRepo) GetByID(ctx context.Context, id string) (*models.Track, error) {
	if m.track != nil && m.track.ID == id {
		return m.track, nil
	}
	return nil, nil
}
func (m *mockStreamTrackRepo) GetByIDs(ctx context.Context, ids []string) ([]*models.Track, error) {
	return nil, nil
}
func (m *mockStreamTrackRepo) List(ctx context.Context, page, perPage int, sortField, topicName string, channelID int64) ([]*models.Track, int64, error) {
	return nil, 0, nil
}
func (m *mockStreamTrackRepo) Search(ctx context.Context, query string, limit int) ([]*models.Track, error) {
	return nil, nil
}
func (m *mockStreamTrackRepo) Random(ctx context.Context, limit int, channelID int64) ([]*models.Track, error) {
	return nil, nil
}
func (m *mockStreamTrackRepo) GetTopics(ctx context.Context, channelID int64, limit int) ([]*models.TopicItem, error) {
	return nil, nil
}
func (m *mockStreamTrackRepo) GetChannelIDs(ctx context.Context) ([]int64, error) {
	return nil, nil
}
func (m *mockStreamTrackRepo) IncrementPlayCount(ctx context.Context, id string) error {
	return nil
}
func (m *mockStreamTrackRepo) UpdateWorkerFileID(ctx context.Context, trackID, workerID, fileID string) error {
	return nil
}
func (m *mockStreamTrackRepo) UpdateLyricsCache(ctx context.Context, id string, text, kind, source string, telegraphURL string) error {
	return nil
}

type mockHistRepoForStream struct {
	mu    sync.Mutex
	plays []string
}

func (m *mockHistRepoForStream) RecordPlay(ctx context.Context, userID int64, trackID string, source string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.plays = append(m.plays, trackID)
	return nil
}
func (m *mockHistRepoForStream) GetUserHistory(ctx context.Context, userID int64, limit int) ([]string, error) {
	return nil, nil
}
func (m *mockHistRepoForStream) GetUserTopPlayed(ctx context.Context, userID int64, limit int) ([]string, error) {
	return nil, nil
}
func (m *mockHistRepoForStream) RecordListeningEvents(ctx context.Context, userID int64, events []models.ListeningEventItem) error {
	return nil
}

func TestStreamHandlerRecordsHistory(t *testing.T) {
	tRepo := &mockStreamTrackRepo{
		track: &models.Track{
			ID: "test_track_1",
			Audio: models.AudioMeta{
				Title:       "Test Track",
				Artist:      "Test Artist",
				DurationSec: 180,
				FileSize:    1024,
				MimeType:    "audio/mpeg",
			},
		},
	}
	hRepo := &mockHistRepoForStream{}

	streamSvc := services.NewStreamService(tRepo, nil)
	authSvc := services.NewAuthService(&config.Config{SecretKey: "test_secret"}, nil)
	histSvc := services.NewHistoryService(hRepo, tRepo)

	handler := NewStreamHandler(streamSvc, authSvc, histSvc)

	token, _ := authSvc.CreateToken(&models.User{
		ID:       999,
		Username: "streamuser",
	})

	r := chi.NewRouter()
	handler.Routes(r)

	// 1. First stream request with ?token=...
	req := httptest.NewRequest("GET", "/tracks/test_track_1/stream?token="+token, nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	// Wait up to 200ms for background goroutine to record play
	time.Sleep(50 * time.Millisecond)

	hRepo.mu.Lock()
	count := len(hRepo.plays)
	hRepo.mu.Unlock()

	if count != 1 {
		t.Errorf("expected 1 play recorded, got %d", count)
	}

	// 2. Second immediate stream request within debounce window (<15s)
	req2 := httptest.NewRequest("GET", "/tracks/test_track_1/stream?token="+token, nil)
	rr2 := httptest.NewRecorder()
	r.ServeHTTP(rr2, req2)

	time.Sleep(50 * time.Millisecond)

	hRepo.mu.Lock()
	count2 := len(hRepo.plays)
	hRepo.mu.Unlock()

	if count2 != 1 {
		t.Errorf("expected still 1 play recorded after debounce, got %d", count2)
	}
}
