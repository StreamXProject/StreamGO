package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"streamgo/internal/models"
)

type mockTrackRepoForService struct {
	track        *models.Track
	softDeleted  bool
	hardDeleted  bool
	artRefCounts map[string]int64
}

func (m *mockTrackRepoForService) GetByID(ctx context.Context, id string) (*models.Track, error) {
	if m.track != nil && m.track.ID == id {
		if m.softDeleted || m.hardDeleted {
			return nil, errors.New("track not found")
		}
		return m.track, nil
	}
	return nil, errors.New("track not found")
}

func (m *mockTrackRepoForService) GetByIDs(ctx context.Context, ids []string) ([]*models.Track, error) {
	return nil, nil
}
func (m *mockTrackRepoForService) List(ctx context.Context, page, perPage int, sortField, topicName string, channelID int64) ([]*models.Track, int64, error) {
	return nil, 0, nil
}
func (m *mockTrackRepoForService) Search(ctx context.Context, query string, limit int) ([]*models.Track, error) {
	return nil, nil
}
func (m *mockTrackRepoForService) Random(ctx context.Context, limit int, channelID int64) ([]*models.Track, error) {
	return nil, nil
}
func (m *mockTrackRepoForService) GetTopics(ctx context.Context, channelID int64, limit int) ([]*models.TopicItem, error) {
	return nil, nil
}
func (m *mockTrackRepoForService) GetChannelIDs(ctx context.Context) ([]int64, error) {
	return nil, nil
}
func (m *mockTrackRepoForService) IncrementPlayCount(ctx context.Context, id string) error {
	return nil
}
func (m *mockTrackRepoForService) UpdateWorkerFileID(ctx context.Context, trackID, workerID, fileID string) error {
	return nil
}
func (m *mockTrackRepoForService) UpdateLyricsCache(ctx context.Context, id string, text, kind, source string, telegraphURL string) error {
	return nil
}

func (m *mockTrackRepoForService) SoftDelete(ctx context.Context, id string) error {
	if m.track != nil && m.track.ID == id {
		m.softDeleted = true
		return nil
	}
	return errors.New("track not found")
}

func (m *mockTrackRepoForService) HardDelete(ctx context.Context, id string) error {
	if m.track != nil && m.track.ID == id {
		m.hardDeleted = true
		return nil
	}
	return errors.New("track not found")
}

func (m *mockTrackRepoForService) CountArtworkReferences(ctx context.Context, excludeTrackID, coverURL string) (int64, error) {
	if m.artRefCounts != nil {
		return m.artRefCounts[coverURL], nil
	}
	return 0, nil
}

func TestTrackService_DeleteTrack_SoftAndHard(t *testing.T) {
	ctx := context.Background()

	// 1. Soft Delete
	mockRepo := &mockTrackRepoForService{
		track: &models.Track{
			ID: "trk_soft",
		},
	}
	svc := NewTrackService(mockRepo)

	res, err := svc.DeleteTrack(ctx, "trk_soft", DeleteTrackOptions{Hard: false})
	if err != nil {
		t.Fatalf("DeleteTrack failed: %v", err)
	}
	if res.Mode != "soft" {
		t.Fatalf("Expected soft mode, got %s", res.Mode)
	}
	if !mockRepo.softDeleted {
		t.Fatal("Expected SoftDelete to be called")
	}

	// 2. Hard Delete
	mockRepoHard := &mockTrackRepoForService{
		track: &models.Track{
			ID: "trk_hard",
		},
	}
	svcHard := NewTrackService(mockRepoHard)

	resHard, err := svcHard.DeleteTrack(ctx, "trk_hard", DeleteTrackOptions{Hard: true})
	if err != nil {
		t.Fatalf("DeleteTrack hard failed: %v", err)
	}
	if resHard.Mode != "hard" {
		t.Fatalf("Expected hard mode, got %s", resHard.Mode)
	}
	if !mockRepoHard.hardDeleted {
		t.Fatal("Expected HardDelete to be called")
	}

	// 3. Not Found
	_, err = svc.DeleteTrack(ctx, "nonexistent", DeleteTrackOptions{})
	if err == nil {
		t.Fatal("Expected error for non-existent track, got nil")
	}
}

func TestTrackService_DeleteTrack_PurgeCache(t *testing.T) {
	ctx := context.Background()
	trackID := "trk_cache_test"

	cacheDir := filepath.Join("stream_media", "alac_cache")
	_ = os.MkdirAll(cacheDir, 0755)
	dummyFile := filepath.Join(cacheDir, fmt.Sprintf("%s.flac", trackID))
	_ = os.WriteFile(dummyFile, []byte("dummy-flac-data"), 0644)
	defer os.Remove(dummyFile)

	mockRepo := &mockTrackRepoForService{
		track: &models.Track{ID: trackID},
	}
	svc := NewTrackService(mockRepo)

	res, err := svc.DeleteTrack(ctx, trackID, DeleteTrackOptions{PurgeCache: true})
	if err != nil {
		t.Fatalf("DeleteTrack failed: %v", err)
	}
	if !res.CachePurged {
		t.Fatal("Expected CachePurged to be true")
	}
	if _, err := os.Stat(dummyFile); !os.IsNotExist(err) {
		t.Fatal("Expected cached file to be removed from disk")
	}
}

func TestTrackService_DeleteTrack_OrphanArtworkPurging(t *testing.T) {
	ctx := context.Background()
	trackID := "trk_art_test"

	deletedKeys := make([]string, 0)
	mockTransport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method == http.MethodDelete {
			deletedKeys = append(deletedKeys, req.URL.Path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("")),
			Header:     make(http.Header),
		}, nil
	})

	r2Svc := &R2StorageService{
		accountID:       "acc123",
		accessKeyID:     "key123",
		secretAccessKey: "sec123",
		bucketName:      "test-bucket",
		publicURL:       "https://cdn.example.com",
		httpClient: &http.Client{
			Transport: mockTransport,
		},
		albumCoverCache: make(map[string]AlbumCovers),
	}

	uniqueCover := "https://cdn.example.com/covers/unique_hash.webp"
	sharedCover := "https://cdn.example.com/covers/shared_hash.webp"

	mockRepo := &mockTrackRepoForService{
		track: &models.Track{
			ID: trackID,
			Spotify: models.SpotifyMeta{
				CloudflareCoverURL:    uniqueCover,
				CloudflareBigCoverURL: sharedCover,
			},
		},
		artRefCounts: map[string]int64{
			uniqueCover: 0, // Orphaned! Safe to delete
			sharedCover: 2, // Shared with sibling! Must preserve
		},
	}

	svc := NewTrackService(mockRepo)
	svc.SetR2StorageService(r2Svc)

	res, err := svc.DeleteTrack(ctx, trackID, DeleteTrackOptions{PurgeArtwork: true})
	if err != nil {
		t.Fatalf("DeleteTrack failed: %v", err)
	}

	// Only unique cover should be deleted
	if len(res.PurgedArtworks) != 1 || res.PurgedArtworks[0] != uniqueCover {
		t.Fatalf("Expected only uniqueCover to be purged, got %v", res.PurgedArtworks)
	}
	if len(deletedKeys) != 1 || deletedKeys[0] != "/test-bucket/covers/unique_hash.webp" {
		t.Fatalf("Unexpected R2 delete calls: %v", deletedKeys)
	}
}

func TestTrackService_DeleteTrack_R2Unconfigured(t *testing.T) {
	ctx := context.Background()
	mockRepo := &mockTrackRepoForService{
		track: &models.Track{
			ID: "trk_no_r2",
			Spotify: models.SpotifyMeta{
				CoverURL: "https://is1-ssl.mzstatic.com/image.jpg",
			},
		},
	}
	svc := NewTrackService(mockRepo)
	// r2Storage is nil / unconfigured
	res, err := svc.DeleteTrack(ctx, "trk_no_r2", DeleteTrackOptions{PurgeArtwork: true})
	if err != nil {
		t.Fatalf("Expected success when R2 is unconfigured, got %v", err)
	}
	if res.Mode != "soft" {
		t.Fatalf("Unexpected mode: %s", res.Mode)
	}
}
