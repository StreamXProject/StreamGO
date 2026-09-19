package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"streamgo/internal/config"
	"streamgo/internal/models"
	"streamgo/internal/services"
)

type mockFavRepoForHandler struct {
	playlists map[string]*models.UserPlaylist
	tracks    map[string][]string
	favs      map[int64][]string
	artists   map[int64][]string
	albums    map[int64][]string
}

func (m *mockFavRepoForHandler) AddFavourite(ctx context.Context, userID int64, trackID string) (bool, error) {
	for _, id := range m.favs[userID] {
		if id == trackID {
			return true, nil
		}
	}
	m.favs[userID] = append(m.favs[userID], trackID)
	return false, nil
}

func (m *mockFavRepoForHandler) RemoveFavourite(ctx context.Context, userID int64, trackID string) (bool, error) {
	ids := m.favs[userID]
	for i, id := range ids {
		if id == trackID {
			m.favs[userID] = append(ids[:i], ids[i+1:]...)
			return true, nil
		}
	}
	return false, nil
}

func (m *mockFavRepoForHandler) GetFavouriteTrackIDs(ctx context.Context, userID int64, page, limit int) ([]string, int64, *float64, error) {
	ids := m.favs[userID]
	return ids, int64(len(ids)), nil, nil
}

func (m *mockFavRepoForHandler) GetAllFavouriteTrackIDs(ctx context.Context, userID int64, page, limit int) ([]string, int64, *float64, error) {
	ids := m.favs[userID]
	return ids, int64(len(ids)), nil, nil
}

func (m *mockFavRepoForHandler) AddFavouriteArtist(ctx context.Context, userID int64, artistID string) (bool, error) {
	for _, id := range m.artists[userID] {
		if id == artistID {
			return true, nil
		}
	}
	m.artists[userID] = append(m.artists[userID], artistID)
	return false, nil
}

func (m *mockFavRepoForHandler) RemoveFavouriteArtist(ctx context.Context, userID int64, artistID string) (bool, error) {
	ids := m.artists[userID]
	for i, id := range ids {
		if id == artistID {
			m.artists[userID] = append(ids[:i], ids[i+1:]...)
			return true, nil
		}
	}
	return false, nil
}

func (m *mockFavRepoForHandler) GetFavouriteArtistIDs(ctx context.Context, userID int64) ([]string, error) {
	return m.artists[userID], nil
}

func (m *mockFavRepoForHandler) CreatePlaylist(ctx context.Context, p *models.UserPlaylist) error {
	m.playlists[p.ID] = p
	return nil
}

func (m *mockFavRepoForHandler) GetUserPlaylists(ctx context.Context, userID int64) ([]*models.UserPlaylist, error) {
	var res []*models.UserPlaylist
	for _, p := range m.playlists {
		if p.UserID == userID {
			res = append(res, p)
		}
	}
	return res, nil
}

func (m *mockFavRepoForHandler) GetPlaylistByID(ctx context.Context, playlistID string, userID int64) (*models.UserPlaylist, error) {
	p, ok := m.playlists[playlistID]
	if !ok || p.UserID != userID {
		return nil, nil
	}
	return p, nil
}

func (m *mockFavRepoForHandler) GetPublicPlaylistByID(ctx context.Context, playlistID string) (*models.UserPlaylist, error) {
	return m.playlists[playlistID], nil
}

func (m *mockFavRepoForHandler) RenamePlaylist(ctx context.Context, playlistID string, userID int64, name, coverID, coverURL, normalThumbnail string) (*models.UserPlaylist, error) {
	p := m.playlists[playlistID]
	if p == nil || p.UserID != userID {
		return nil, nil
	}
	p.Name = name
	return p, nil
}

func (m *mockFavRepoForHandler) DeletePlaylist(ctx context.Context, playlistID string, userID int64) error {
	delete(m.playlists, playlistID)
	return nil
}

func (m *mockFavRepoForHandler) AddTracksToPlaylist(ctx context.Context, playlistID string, trackIDs []string) (int, error) {
	m.tracks[playlistID] = append(m.tracks[playlistID], trackIDs...)
	return len(trackIDs), nil
}

func (m *mockFavRepoForHandler) RemoveTrackFromPlaylist(ctx context.Context, playlistID string, trackID string) error {
	ids := m.tracks[playlistID]
	for i, id := range ids {
		if id == trackID {
			m.tracks[playlistID] = append(ids[:i], ids[i+1:]...)
			return nil
		}
	}
	return nil
}

func (m *mockFavRepoForHandler) GetPlaylistTrackIDs(ctx context.Context, playlistID string, page, limit int) ([]string, int64, error) {
	ids := m.tracks[playlistID]
	return ids, int64(len(ids)), nil
}

func (m *mockFavRepoForHandler) GetPlaylistFirstTrackIDs(ctx context.Context, playlistID string, limit int) ([]string, error) {
	ids := m.tracks[playlistID]
	if len(ids) > limit {
		return ids[:limit], nil
	}
	return ids, nil
}

func (m *mockFavRepoForHandler) SaveAlbum(ctx context.Context, userID int64, albumID string) (bool, error) {
	for _, id := range m.albums[userID] {
		if id == albumID {
			return true, nil
		}
	}
	m.albums[userID] = append(m.albums[userID], albumID)
	return false, nil
}

func (m *mockFavRepoForHandler) GetSavedAlbumIDs(ctx context.Context, userID int64, page, limit int) ([]string, []float64, int64, error) {
	ids := m.albums[userID]
	ts := make([]float64, len(ids))
	return ids, ts, int64(len(ids)), nil
}

func (m *mockFavRepoForHandler) RecordListeningEvents(ctx context.Context, userID *int64, events []models.ListeningEventItem) (int, error) {
	return len(events), nil
}

func (m *mockFavRepoForHandler) GetUserHistoryTrackIDs(ctx context.Context, userID int64, limit int) ([]string, error) {
	return []string{}, nil
}

func (m *mockFavRepoForHandler) GetUserTopPlayedTrackIDs(ctx context.Context, userID int64, page, limit int) ([]string, int64, error) {
	return []string{}, 0, nil
}

type mockTrackRepoForHandler struct{}

func (m *mockTrackRepoForHandler) GetByID(ctx context.Context, id string) (*models.Track, error) {
	return &models.Track{ID: id, Audio: models.AudioMeta{Title: "Test", Artist: "Artist"}}, nil
}

func (m *mockTrackRepoForHandler) GetByIDs(ctx context.Context, ids []string) ([]*models.Track, error) {
	var res []*models.Track
	for _, id := range ids {
		res = append(res, &models.Track{ID: id, Audio: models.AudioMeta{Title: "Test", Artist: "Artist"}})
	}
	return res, nil
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

func (m *mockTrackRepoForHandler) GetTopics(ctx context.Context, limit int) ([]*models.TopicItem, error) {
	return []*models.TopicItem{
		{
			Name:            "EDM",
			TopicName:       "EDM",
			Count:           15,
			TracksCount:     15,
			CoverURL:        "https://example.com/edm.jpg",
			Endpoint:        "/topics/EDM/tracks",
			Thumbnails:      []string{"https://example.com/edm.jpg"},
		},
	}, nil
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

type mockUserRepoForHandler struct{}

func (m *mockUserRepoForHandler) UpsertUser(ctx context.Context, user *models.User) (*models.User, error) {
	return user, nil
}
func (m *mockUserRepoForHandler) GetUserByID(ctx context.Context, id int64) (*models.User, error) {
	return &models.User{ID: id, Username: "test"}, nil
}
func (m *mockUserRepoForHandler) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	return &models.User{ID: 100, Username: username}, nil
}
func (m *mockUserRepoForHandler) UpdateUserCredentials(ctx context.Context, userID int64, username string, pwd *models.PasswordHash) error {
	return nil
}
func (m *mockUserRepoForHandler) UpdateIntegrations(ctx context.Context, userID int64, req models.IntegrationsUpdateRequest) (*models.UserIntegrations, error) {
	return &models.UserIntegrations{}, nil
}
func (m *mockUserRepoForHandler) SaveRegistrationOTP(ctx context.Context, otp *models.RegistrationOTP) error {
	return nil
}
func (m *mockUserRepoForHandler) GetRegistrationOTP(ctx context.Context, userID int64) (*models.RegistrationOTP, error) {
	return nil, nil
}
func (m *mockUserRepoForHandler) DeleteRegistrationOTP(ctx context.Context, userID int64) error {
	return nil
}
func (m *mockUserRepoForHandler) SaveOIDCSession(ctx context.Context, session *models.OIDCSession) error {
	return nil
}
func (m *mockUserRepoForHandler) GetAndDeleteOIDCSession(ctx context.Context, state string) (*models.OIDCSession, error) {
	return nil, nil
}
func (m *mockUserRepoForHandler) GetOwnerPassword(ctx context.Context) (*models.PasswordHash, error) {
	return nil, nil
}
func (m *mockUserRepoForHandler) SetOwnerPassword(ctx context.Context, pwd *models.PasswordHash, updatedBy int64) error {
	return nil
}
func (m *mockUserRepoForHandler) UpdateFCMToken(ctx context.Context, userID int64, fcmToken string) error {
	return nil
}
func (m *mockUserRepoForHandler) SaveBotAuthSession(ctx context.Context, session *models.BotAuthSession) error {
	return nil
}
func (m *mockUserRepoForHandler) GetBotAuthSession(ctx context.Context, sessionID string) (*models.BotAuthSession, error) {
	return nil, nil
}

func TestPlaylistsAndFavouritesHandlers(t *testing.T) {
	cfg := &config.Config{SecretKey: "secret"}
	authSvc := services.NewAuthService(cfg, &mockUserRepoForHandler{})
	token, _ := authSvc.CreateToken(&models.User{ID: 12345, Username: "testuser"})

	favRepo := &mockFavRepoForHandler{
		playlists: make(map[string]*models.UserPlaylist),
		tracks:    make(map[string][]string),
		favs:      make(map[int64][]string),
		artists:   make(map[int64][]string),
		albums:    make(map[int64][]string),
	}
	trackRepo := &mockTrackRepoForHandler{}
	favSvc := services.NewFavouritePlaylistService(favRepo, trackRepo, nil)
	discordSvc := services.NewDiscordService()

	playlistH := NewPlaylistHandler(favSvc, authSvc)
	favH := NewFavouriteHandler(favSvc, authSvc)
	discordH := NewDiscordHandler(discordSvc)

	r := chi.NewRouter()
	playlistH.Routes(r)
	favH.Routes(r)
	discordH.Routes(r)

	// 1. Create playlist
	body := `{"name": "Vibes"}`
	req := httptest.NewRequest(http.MethodPost, "/me/playlists", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var pResp models.PlaylistItem
	_ = json.Unmarshal(w.Body.Bytes(), &pResp)
	if pResp.Name != "Vibes" {
		t.Errorf("expected playlist name Vibes, got %s", pResp.Name)
	}

	// 2. Add Track to playlist
	addBody := `{"track_id": "trk_99"}`
	req = httptest.NewRequest(http.MethodPost, "/me/playlists/"+pResp.PlaylistID+"/tracks", bytes.NewBufferString(addBody))
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// 3. Add favourite track
	favBody := `{"track_id": "trk_fav_1"}`
	req = httptest.NewRequest(http.MethodPost, "/me/favourites", bytes.NewBufferString(favBody))
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// 4. Get favourite track IDs
	req = httptest.NewRequest(http.MethodGet, "/me/favourites/ids", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// 5. Discord remote auth status for missing session -> 404
	req = httptest.NewRequest(http.MethodGet, "/api/discord/remote-auth/status/non-existent", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for non-existent discord session, got %d", w.Code)
	}
}

func TestTopicsAndAuthEndpoints(t *testing.T) {
	cfg := &config.Config{SecretKey: "secret"}
	authSvc := services.NewAuthService(cfg, &mockUserRepoForHandler{})
	trackRepo := &mockTrackRepoForHandler{}
	trackSvc := services.NewTrackService(trackRepo)
	topicH := NewTopicHandler(trackSvc)
	authH := NewAuthHandler(cfg, authSvc)

	r := chi.NewRouter()
	topicH.Routes(r)
	authH.Routes(r)

	// 1. GET /topics
	req := httptest.NewRequest(http.MethodGet, "/topics", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var topicsResp models.TopicsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &topicsResp); err != nil {
		t.Fatalf("failed to unmarshal topics: %v", err)
	}
	if !topicsResp.OK || topicsResp.Total != 1 || len(topicsResp.Items) != 1 || len(topicsResp.Topics) != 1 {
		t.Fatalf("unexpected topics response: %+v", topicsResp)
	}
	if topicsResp.Topics[0] != "EDM" || topicsResp.Items[0].Name != "EDM" || topicsResp.Items[0].Count != 15 {
		t.Fatalf("unexpected topic item data: %+v", topicsResp.Items[0])
	}

	// 2. POST /auth/fcm-token with valid auth token
	token, _ := authSvc.CreateToken(&models.User{ID: 100, Username: "test"})
	req = httptest.NewRequest(http.MethodPost, "/auth/fcm-token", bytes.NewBufferString(`{"fcm_token": "token_xyz"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// 3. POST /auth/telegram/bot-session
	req = httptest.NewRequest(http.MethodPost, "/auth/telegram/bot-session", bytes.NewBufferString(`{"invite_code": "INV123"}`))
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var sessResp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &sessResp)
	if sessResp["ok"] != true || sessResp["session_id"] == nil {
		t.Fatalf("unexpected bot session response: %+v", sessResp)
	}
}
