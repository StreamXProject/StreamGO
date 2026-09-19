package services

import (
	"context"
	"testing"

	"streamgo/internal/models"
)

type mockFavRepo struct {
	playlists    map[string]*models.UserPlaylist
	tracks       map[string][]string // playlistID -> trackIDs
	favourites   map[int64][]string  // userID -> trackIDs
	artists      map[int64][]string  // userID -> artistIDs
	savedAlbums  map[int64][]string  // userID -> albumIDs
}

func newMockFavRepo() *mockFavRepo {
	return &mockFavRepo{
		playlists:   make(map[string]*models.UserPlaylist),
		tracks:      make(map[string][]string),
		favourites:  make(map[int64][]string),
		artists:     make(map[int64][]string),
		savedAlbums: make(map[int64][]string),
	}
}

func (m *mockFavRepo) AddFavourite(ctx context.Context, userID int64, trackID string) (bool, error) {
	for _, id := range m.favourites[userID] {
		if id == trackID {
			return true, nil
		}
	}
	m.favourites[userID] = append(m.favourites[userID], trackID)
	return false, nil
}

func (m *mockFavRepo) RemoveFavourite(ctx context.Context, userID int64, trackID string) (bool, error) {
	ids := m.favourites[userID]
	for i, id := range ids {
		if id == trackID {
			m.favourites[userID] = append(ids[:i], ids[i+1:]...)
			return true, nil
		}
	}
	return false, nil
}

func (m *mockFavRepo) GetFavouriteTrackIDs(ctx context.Context, userID int64, page, limit int) ([]string, int64, *float64, error) {
	ids := m.favourites[userID]
	now := float64(1700000000)
	return ids, int64(len(ids)), &now, nil
}

func (m *mockFavRepo) GetAllFavouriteTrackIDs(ctx context.Context, userID int64, page, limit int) ([]string, int64, *float64, error) {
	ids := m.favourites[userID]
	now := float64(1700000000)
	return ids, int64(len(ids)), &now, nil
}

func (m *mockFavRepo) AddFavouriteArtist(ctx context.Context, userID int64, artistID string) (bool, error) {
	for _, id := range m.artists[userID] {
		if id == artistID {
			return true, nil
		}
	}
	m.artists[userID] = append(m.artists[userID], artistID)
	return false, nil
}

func (m *mockFavRepo) RemoveFavouriteArtist(ctx context.Context, userID int64, artistID string) (bool, error) {
	ids := m.artists[userID]
	for i, id := range ids {
		if id == artistID {
			m.artists[userID] = append(ids[:i], ids[i+1:]...)
			return true, nil
		}
	}
	return false, nil
}

func (m *mockFavRepo) GetFavouriteArtistIDs(ctx context.Context, userID int64) ([]string, error) {
	return m.artists[userID], nil
}

func (m *mockFavRepo) CreatePlaylist(ctx context.Context, p *models.UserPlaylist) error {
	m.playlists[p.ID] = p
	return nil
}

func (m *mockFavRepo) GetUserPlaylists(ctx context.Context, userID int64) ([]*models.UserPlaylist, error) {
	var out []*models.UserPlaylist
	for _, p := range m.playlists {
		if p.UserID == userID {
			out = append(out, p)
		}
	}
	return out, nil
}

func (m *mockFavRepo) GetPlaylistByID(ctx context.Context, playlistID string, userID int64) (*models.UserPlaylist, error) {
	p, ok := m.playlists[playlistID]
	if !ok || p.UserID != userID {
		return nil, nil
	}
	return p, nil
}

func (m *mockFavRepo) GetPublicPlaylistByID(ctx context.Context, playlistID string) (*models.UserPlaylist, error) {
	return m.playlists[playlistID], nil
}

func (m *mockFavRepo) RenamePlaylist(ctx context.Context, playlistID string, userID int64, name, coverID, coverURL, normalThumbnail string) (*models.UserPlaylist, error) {
	p := m.playlists[playlistID]
	if p == nil || p.UserID != userID {
		return nil, nil
	}
	p.Name = name
	p.CoverID = coverID
	p.CoverURL = coverURL
	p.NormalThumbnail = normalThumbnail
	return p, nil
}

func (m *mockFavRepo) DeletePlaylist(ctx context.Context, playlistID string, userID int64) error {
	delete(m.playlists, playlistID)
	delete(m.tracks, playlistID)
	return nil
}

func (m *mockFavRepo) AddTracksToPlaylist(ctx context.Context, playlistID string, trackIDs []string) (int, error) {
	m.tracks[playlistID] = append(m.tracks[playlistID], trackIDs...)
	return len(trackIDs), nil
}

func (m *mockFavRepo) RemoveTrackFromPlaylist(ctx context.Context, playlistID string, trackID string) error {
	ids := m.tracks[playlistID]
	for i, id := range ids {
		if id == trackID {
			m.tracks[playlistID] = append(ids[:i], ids[i+1:]...)
			return nil
		}
	}
	return nil
}

func (m *mockFavRepo) GetPlaylistTrackIDs(ctx context.Context, playlistID string, page, limit int) ([]string, int64, error) {
	ids := m.tracks[playlistID]
	return ids, int64(len(ids)), nil
}

func (m *mockFavRepo) GetPlaylistFirstTrackIDs(ctx context.Context, playlistID string, limit int) ([]string, error) {
	ids := m.tracks[playlistID]
	if len(ids) > limit {
		return ids[:limit], nil
	}
	return ids, nil
}

func (m *mockFavRepo) SaveAlbum(ctx context.Context, userID int64, albumID string) (bool, error) {
	for _, id := range m.savedAlbums[userID] {
		if id == albumID {
			return true, nil
		}
	}
	m.savedAlbums[userID] = append(m.savedAlbums[userID], albumID)
	return false, nil
}

func (m *mockFavRepo) GetSavedAlbumIDs(ctx context.Context, userID int64, page, limit int) ([]string, []float64, int64, error) {
	ids := m.savedAlbums[userID]
	ts := make([]float64, len(ids))
	return ids, ts, int64(len(ids)), nil
}

func (m *mockFavRepo) RecordListeningEvents(ctx context.Context, userID *int64, events []models.ListeningEventItem) (int, error) {
	return len(events), nil
}

func (m *mockFavRepo) GetUserHistoryTrackIDs(ctx context.Context, userID int64, limit int) ([]string, error) {
	return []string{}, nil
}

func (m *mockFavRepo) GetUserTopPlayedTrackIDs(ctx context.Context, userID int64, page, limit int) ([]string, int64, error) {
	return []string{}, 0, nil
}

type mockTrackRepo struct {
	tracks map[string]*models.Track
}

func (m *mockTrackRepo) GetByID(ctx context.Context, id string) (*models.Track, error) {
	return m.tracks[id], nil
}

func (m *mockTrackRepo) GetByIDs(ctx context.Context, ids []string) ([]*models.Track, error) {
	var out []*models.Track
	for _, id := range ids {
		if t, ok := m.tracks[id]; ok {
			out = append(out, t)
		}
	}
	return out, nil
}

func (m *mockTrackRepo) List(ctx context.Context, page, perPage int, sortField, topicName string, channelID int64) ([]*models.Track, int64, error) {
	return nil, 0, nil
}

func (m *mockTrackRepo) Search(ctx context.Context, query string, limit int) ([]*models.Track, error) {
	return nil, nil
}

func (m *mockTrackRepo) Random(ctx context.Context, limit int, channelID int64) ([]*models.Track, error) {
	return nil, nil
}

func (m *mockTrackRepo) GetTopics(ctx context.Context, limit int) ([]*models.TopicItem, error) {
	return nil, nil
}

func (m *mockTrackRepo) GetChannelIDs(ctx context.Context) ([]int64, error) {
	return nil, nil
}

func (m *mockTrackRepo) IncrementPlayCount(ctx context.Context, id string) error {
	return nil
}

func (m *mockTrackRepo) UpdateWorkerFileID(ctx context.Context, trackID, workerID, fileID string) error {
	return nil
}

func TestFavouritePlaylistService(t *testing.T) {
	ctx := context.Background()
	favRepo := newMockFavRepo()
	trackRepo := &mockTrackRepo{
		tracks: map[string]*models.Track{
			"trk1": {
				ID: "trk1",
				Audio: models.AudioMeta{
					Title:  "Track One",
					Artist: "Artist A",
				},
				Spotify: models.SpotifyMeta{
					CoverURL: "https://example.com/cover1.jpg",
				},
			},
			"trk2": {
				ID: "trk2",
				Audio: models.AudioMeta{
					Title:  "Track Two",
					Artist: "Artist B",
				},
				Spotify: models.SpotifyMeta{
					CoverURL: "https://example.com/cover2.jpg",
				},
			},
		},
	}

	svc := NewFavouritePlaylistService(favRepo, trackRepo, nil)
	userID := int64(12345)

	// 1. Create playlist
	p, err := svc.CreatePlaylist(ctx, userID, "My Chill Mix")
	if err != nil {
		t.Fatalf("failed to create playlist: %v", err)
	}
	if p.Name != "My Chill Mix" {
		t.Errorf("expected name 'My Chill Mix', got '%s'", p.Name)
	}

	// 2. Add tracks to playlist
	added, err := svc.AddTracksToPlaylist(ctx, p.PlaylistID, userID, []string{"trk1", "trk2"})
	if err != nil {
		t.Fatalf("failed to add tracks: %v", err)
	}
	if added != 2 {
		t.Errorf("expected 2 tracks added, got %d", added)
	}

	// 3. List playlists (check collage thumbnails)
	lists, err := svc.GetUserPlaylists(ctx, userID)
	if err != nil {
		t.Fatalf("failed to list playlists: %v", err)
	}
	if len(lists.Items) != 1 {
		t.Fatalf("expected 1 playlist, got %d", len(lists.Items))
	}
	if len(lists.Items[0].Thumbnails) < 2 {
		t.Errorf("expected at least 2 collage thumbnails, got %d", len(lists.Items[0].Thumbnails))
	}

	// 4. Rename playlist
	renamed, err := svc.RenamePlaylist(ctx, p.PlaylistID, userID, "Renamed Mix")
	if err != nil {
		t.Fatalf("failed to rename playlist: %v", err)
	}
	if renamed.Name != "Renamed Mix" {
		t.Errorf("expected renamed 'Renamed Mix', got '%s'", renamed.Name)
	}

	// 5. Delete playlist
	err = svc.DeletePlaylist(ctx, p.PlaylistID, userID)
	if err != nil {
		t.Fatalf("failed to delete playlist: %v", err)
	}

	// 6. Test Favourites
	already, err := svc.AddFavourite(ctx, userID, "trk1")
	if err != nil || already {
		t.Fatalf("failed to add favourite: %v (already: %v)", err, already)
	}

	favIDs, err := svc.GetFavouriteIDs(ctx, userID, 1, 100)
	if err != nil {
		t.Fatalf("failed to get favourite IDs: %v", err)
	}
	if len(favIDs.IDs) != 1 || favIDs.IDs[0] != "trk1" {
		t.Errorf("expected ['trk1'], got %v", favIDs.IDs)
	}

	deleted, err := svc.RemoveFavourite(ctx, userID, "trk1")
	if err != nil || !deleted {
		t.Fatalf("failed to remove favourite: %v", err)
	}

	// 7. Test Artist Favourites
	alreadyA, err := svc.AddFavouriteArtist(ctx, userID, "artist_xyz")
	if err != nil || alreadyA {
		t.Fatalf("failed to add favourite artist: %v", err)
	}

	artistIDs, err := svc.GetFavouriteArtistIDs(ctx, userID)
	if err != nil {
		t.Fatalf("failed to get favourite artist IDs: %v", err)
	}
	if len(artistIDs) != 1 || artistIDs[0] != "artist_xyz" {
		t.Errorf("expected ['artist_xyz'], got %v", artistIDs)
	}

	deletedA, err := svc.RemoveFavouriteArtist(ctx, userID, "artist_xyz")
	if err != nil || !deletedA {
		t.Fatalf("failed to remove favourite artist: %v", err)
	}

	// 8. Test Saved Albums
	count, err := svc.SaveAlbums(ctx, userID, []string{"alb_1", "alb_2"})
	if err != nil {
		t.Fatalf("failed to save albums: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 albums saved, got %d", count)
	}
}
