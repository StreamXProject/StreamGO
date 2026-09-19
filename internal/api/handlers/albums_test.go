package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"streamgo/internal/models"
	"streamgo/internal/services"
)

type mockArtistAlbumRepoForTest struct {
	albums  []*models.Album
	artists []*models.Artist
	tracks  []*models.Track
}

func (m *mockArtistAlbumRepoForTest) ListArtists(ctx context.Context, page, perPage int, refresh bool) ([]*models.Artist, int64, error) {
	return m.artists, int64(len(m.artists)), nil
}

func (m *mockArtistAlbumRepoForTest) RefreshArtistsCache(ctx context.Context, limitTracks, limitArtists int) (int, error) {
	return len(m.artists), nil
}

func (m *mockArtistAlbumRepoForTest) GetArtistByID(ctx context.Context, id string) (*models.Artist, error) {
	for _, a := range m.artists {
		if a.ID == id || a.Name == id {
			return a, nil
		}
	}
	return nil, nil
}

func (m *mockArtistAlbumRepoForTest) GetArtistTracks(ctx context.Context, artistName string, limit int) ([]*models.Track, error) {
	return m.tracks, nil
}

func (m *mockArtistAlbumRepoForTest) GetArtistTracksPaginated(ctx context.Context, artistName string, page, perPage int) ([]*models.Track, int64, error) {
	return m.tracks, int64(len(m.tracks)), nil
}

func (m *mockArtistAlbumRepoForTest) GetArtistAlbums(ctx context.Context, artistName string) ([]*models.Album, error) {
	return m.albums, nil
}

func (m *mockArtistAlbumRepoForTest) ListAlbums(ctx context.Context, page, perPage int, artistFilter string, refresh bool) ([]*models.Album, int64, error) {
	return m.albums, int64(len(m.albums)), nil
}

func (m *mockArtistAlbumRepoForTest) GetAlbumByID(ctx context.Context, id string) (*models.Album, error) {
	for _, a := range m.albums {
		if a.ID == id {
			return a, nil
		}
	}
	return nil, nil
}

func (m *mockArtistAlbumRepoForTest) GetAlbumTracks(ctx context.Context, albumID string) ([]*models.Track, error) {
	return m.tracks, nil
}

func (m *mockArtistAlbumRepoForTest) GetAlbumTracksPaginated(ctx context.Context, albumID string, page, perPage int) ([]*models.Track, int64, error) {
	return m.tracks, int64(len(m.tracks)), nil
}

func (m *mockArtistAlbumRepoForTest) RefreshAlbumsCache(ctx context.Context, limit int) (int, error) {
	return len(m.albums), nil
}

func TestAlbumsAndArtistsPythonResponseShapes(t *testing.T) {
	repo := &mockArtistAlbumRepoForTest{
		albums: []*models.Album{
			{
				ID:          "album_delirium_deluxe_2015",
				Id:          "album_delirium_deluxe_2015",
				Title:       "Delirium (Deluxe)",
				Artist:      "Ellie Goulding",
				TracksCount: 1,
				TrackCount:  1,
				Year:        2015,
				CoverURL:    "https://example.com/cover.jpg",
			},
		},
		artists: []*models.Artist{
			{
				ID:          "artist_ellie_goulding",
				Id:          "artist_ellie_goulding",
				Name:        "Ellie Goulding",
				TracksCount: 1,
				TrackCount:  1,
				CoverURL:    "https://example.com/avatar.jpg",
			},
		},
		tracks: []*models.Track{
			{
				ID: "AgADkAYAAr1QcEU",
				Audio: models.AudioMeta{
					Title:       "Love Me Like You Do",
					Artist:      "Ellie Goulding",
					Album:       "Delirium (Deluxe)",
					AlbumID:     "album_delirium_deluxe_2015",
					DurationSec: 252,
				},
			},
		},
	}

	svc := services.NewArtistAlbumService(repo, nil)
	albumHandler := NewAlbumHandler(svc)
	artistHandler := NewArtistHandler(svc)

	r := chi.NewRouter()
	albumHandler.Routes(r)
	artistHandler.Routes(r)

	// 1. Test GET /albums
	t.Run("GET /albums response matches Python schema", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/albums?limit=200", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}

		var resp map[string]any
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode json: %v", err)
		}

		if resp["ok"] != true {
			t.Errorf("expected ok=true, got %v", resp["ok"])
		}
		if int(resp["per_page"].(float64)) != 200 {
			t.Errorf("expected per_page=200, got %v", resp["per_page"])
		}
		if int(resp["page"].(float64)) != 1 {
			t.Errorf("expected page=1, got %v", resp["page"])
		}
		if int(resp["total"].(float64)) != 1 {
			t.Errorf("expected total=1, got %v", resp["total"])
		}

		items, ok := resp["items"].([]any)
		if !ok || len(items) != 1 {
			t.Fatalf("expected 1 item, got %v", resp["items"])
		}
		first := items[0].(map[string]any)
		if first["id"] != "album_delirium_deluxe_2015" || first["_id"] != "album_delirium_deluxe_2015" {
			t.Errorf("expected id and _id to match, got id=%v, _id=%v", first["id"], first["_id"])
		}
		if first["title"] != "Delirium (Deluxe)" {
			t.Errorf("expected title='Delirium (Deluxe)', got %v", first["title"])
		}
	})

	// 2. Test GET /albums/{id}
	t.Run("GET /albums/{id} matches Python schema", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/albums/album_delirium_deluxe_2015", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}

		var resp map[string]any
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode json: %v", err)
		}

		if resp["ok"] != true {
			t.Errorf("expected ok=true, got %v", resp["ok"])
		}
		if resp["album"] == nil {
			t.Errorf("expected album object, got nil")
		}
		if resp["tracks"] == nil {
			t.Errorf("expected tracks array, got nil")
		}
	})

	// 3. Test GET /albums/{id}/tracks
	t.Run("GET /albums/{id}/tracks matches Python schema", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/albums/album_delirium_deluxe_2015/tracks", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}

		var resp map[string]any
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode json: %v", err)
		}

		if resp["ok"] != true {
			t.Errorf("expected ok=true, got %v", resp["ok"])
		}
		if int(resp["page"].(float64)) != 1 {
			t.Errorf("expected page=1, got %v", resp["page"])
		}
		if resp["items"] == nil {
			t.Errorf("expected items array, got nil")
		}
	})

	// 4. Test GET /artists
	t.Run("GET /artists matches Python schema", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/artists?limit=200", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}

		var resp map[string]any
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode json: %v", err)
		}

		if resp["ok"] != true {
			t.Errorf("expected ok=true, got %v", resp["ok"])
		}
		if int(resp["per_page"].(float64)) != 200 {
			t.Errorf("expected per_page=200, got %v", resp["per_page"])
		}
		if resp["items"] == nil {
			t.Errorf("expected items array, got nil")
		}
	})

	// 5. Test GET /artists/{id}
	t.Run("GET /artists/{id} matches Python schema", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/artists/artist_ellie_goulding", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}

		var resp map[string]any
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode json: %v", err)
		}

		if resp["ok"] != true {
			t.Errorf("expected ok=true, got %v", resp["ok"])
		}
		if resp["artist"] == nil {
			t.Errorf("expected artist object, got nil")
		}
		if resp["popular_tracks"] == nil {
			t.Errorf("expected popular_tracks object, got nil")
		}
		if resp["releases"] == nil {
			t.Errorf("expected releases object, got nil")
		}
	})
}
