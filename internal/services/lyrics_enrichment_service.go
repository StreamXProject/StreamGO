package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"streamgo/internal/logger"
)

var logLyrics = logger.New("lyrics")

// LyricsResult bundles resolved lyrics and alternative multilingual titles.
type LyricsResult struct {
	Lyrics       string
	SyncedLyrics string
	Titles       map[string]any
}

// LyricsEnrichmentService fetches lyrics and alternative titles from LRCLIB and Musixmatch.
type LyricsEnrichmentService struct {
	httpClient *http.Client
}

// NewLyricsEnrichmentService creates a new LyricsEnrichmentService.
func NewLyricsEnrichmentService() *LyricsEnrichmentService {
	return &LyricsEnrichmentService{
		httpClient: &http.Client{
			Timeout: 8 * time.Second,
		},
	}
}

// FetchLyrics searches for lyrics and romanized/localized titles.
func (s *LyricsEnrichmentService) FetchLyrics(ctx context.Context, title, artist, album string) (*LyricsResult, error) {
	cleanTitle := StripNoise(title)
	cleanArtist := StripNoise(artist)
	cleanAlbum := StripNoise(album)

	if cleanTitle == "" {
		return nil, fmt.Errorf("title cannot be empty")
	}

	// 1. Query LRCLIB API
	res, err := s.queryLRCLIB(ctx, cleanTitle, cleanArtist, cleanAlbum)
	if err == nil && res != nil {
		return res, nil
	}

	return nil, fmt.Errorf("no lyrics found for %s - %s", artist, title)
}

func (s *LyricsEnrichmentService) queryLRCLIB(ctx context.Context, title, artist, album string) (*LyricsResult, error) {
	apiURL := fmt.Sprintf("https://lrclib.net/api/get?track_name=%s&artist_name=%s",
		url.QueryEscape(title),
		url.QueryEscape(artist),
	)
	if album != "" {
		apiURL += fmt.Sprintf("&album_name=%s", url.QueryEscape(album))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "StreamGO/1.0")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("lrclib status: %d", resp.StatusCode)
	}

	var payload struct {
		TrackName    string `json:"trackName"`
		ArtistName   string `json:"artistName"`
		PlainLyrics  string `json:"plainLyrics"`
		SyncedLyrics string `json:"syncedLyrics"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	bestLyrics := payload.SyncedLyrics
	if bestLyrics == "" {
		bestLyrics = payload.PlainLyrics
	}

	if bestLyrics == "" {
		return nil, fmt.Errorf("empty lyrics from lrclib")
	}

	return &LyricsResult{
		Lyrics:       bestLyrics,
		SyncedLyrics: payload.SyncedLyrics,
	}, nil
}
