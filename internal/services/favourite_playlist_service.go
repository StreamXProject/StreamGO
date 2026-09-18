package services

import (
	"context"
	"errors"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"

	"streamgo/internal/models"
	"streamgo/internal/repository"
)

// FavouritePlaylistService orchestrates user favorites and playlist logic.
type FavouritePlaylistService struct {
	favRepo   repository.FavouritePlaylistRepository
	trackRepo repository.TrackRepository
}

// NewFavouritePlaylistService creates a new FavouritePlaylistService.
func NewFavouritePlaylistService(favRepo repository.FavouritePlaylistRepository, trackRepo repository.TrackRepository) *FavouritePlaylistService {
	return &FavouritePlaylistService{
		favRepo:   favRepo,
		trackRepo: trackRepo,
	}
}

// AddFavourite likes a track for a user.
func (s *FavouritePlaylistService) AddFavourite(ctx context.Context, userID int64, trackID string) (bool, error) {
	trackID = strings.TrimSpace(trackID)
	if trackID == "" {
		return false, errors.New("track_id is required")
	}
	track, err := s.trackRepo.GetByID(ctx, trackID)
	if err != nil {
		return false, err
	}
	if track == nil {
		return false, errors.New("track not found")
	}
	return s.favRepo.AddFavourite(ctx, userID, trackID)
}

// RemoveFavourite unlikes a track.
func (s *FavouritePlaylistService) RemoveFavourite(ctx context.Context, userID int64, trackID string) (bool, error) {
	trackID = strings.TrimSpace(trackID)
	if trackID == "" {
		return false, errors.New("track_id is required")
	}
	return s.favRepo.RemoveFavourite(ctx, userID, trackID)
}

// GetFavouriteIDs returns array of favorite track IDs.
func (s *FavouritePlaylistService) GetFavouriteIDs(ctx context.Context, userID int64, page, limit int) (*models.FavouriteIDsResponse, error) {
	ids, _, _, err := s.favRepo.GetFavouriteTrackIDs(ctx, userID, page, limit)
	if err != nil {
		return nil, err
	}
	if ids == nil {
		ids = []string{}
	}
	return &models.FavouriteIDsResponse{
		OK:  true,
		IDs: ids,
	}, nil
}

// GetFavourites returns paginated favorites converted to BrowseItems.
func (s *FavouritePlaylistService) GetFavourites(ctx context.Context, userID int64, page, limit int) (*models.BrowseResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	ids, total, _, err := s.favRepo.GetFavouriteTrackIDs(ctx, userID, page, limit)
	if err != nil {
		return nil, err
	}

	tracks, err := s.trackRepo.GetByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	items := make([]*models.BrowseItem, 0, len(tracks))
	for _, t := range tracks {
		items = append(items, t.ToBrowseItem())
	}

	totalPages := 0
	if total > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(limit)))
	}

	return &models.BrowseResponse{
		Items:      items,
		Total:      total,
		Page:       page,
		PerPage:    limit,
		TotalPages: totalPages,
	}, nil
}

// AddFavouriteArtist follows an artist.
func (s *FavouritePlaylistService) AddFavouriteArtist(ctx context.Context, userID int64, artistID string) (bool, error) {
	artistID = strings.TrimSpace(artistID)
	if artistID == "" {
		return false, errors.New("artist_id is required")
	}
	return s.favRepo.AddFavouriteArtist(ctx, userID, artistID)
}

// RemoveFavouriteArtist unfollows an artist.
func (s *FavouritePlaylistService) RemoveFavouriteArtist(ctx context.Context, userID int64, artistID string) (bool, error) {
	artistID = strings.TrimSpace(artistID)
	if artistID == "" {
		return false, errors.New("artist_id is required")
	}
	return s.favRepo.RemoveFavouriteArtist(ctx, userID, artistID)
}

// GetFavouriteArtistIDs returns list of followed artist IDs.
func (s *FavouritePlaylistService) GetFavouriteArtistIDs(ctx context.Context, userID int64) (*models.FavouriteArtistsResponse, error) {
	ids, err := s.favRepo.GetFavouriteArtistIDs(ctx, userID)
	if err != nil {
		return nil, err
	}
	if ids == nil {
		ids = []string{}
	}
	return &models.FavouriteArtistsResponse{
		OK:  true,
		IDs: ids,
	}, nil
}

// CreatePlaylist creates a new custom playlist.
func (s *FavouritePlaylistService) CreatePlaylist(ctx context.Context, userID int64, title, description, coverURL string) (*models.UserPlaylist, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, errors.New("playlist title is required")
	}

	p := &models.UserPlaylist{
		ID:          uuid.New().String(),
		UserID:      userID,
		Title:       title,
		Description: strings.TrimSpace(description),
		CoverURL:    strings.TrimSpace(coverURL),
		TrackCount:  0,
		CreatedAt:   float64(time.Now().Unix()),
		UpdatedAt:   float64(time.Now().Unix()),
	}

	if err := s.favRepo.CreatePlaylist(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

// GetUserPlaylists lists all playlists owned by a user.
func (s *FavouritePlaylistService) GetUserPlaylists(ctx context.Context, userID int64) (*models.PlaylistsResponse, error) {
	playlists, err := s.favRepo.GetUserPlaylists(ctx, userID)
	if err != nil {
		return nil, err
	}
	if playlists == nil {
		playlists = []*models.UserPlaylist{}
	}
	return &models.PlaylistsResponse{
		OK:    true,
		Total: len(playlists),
		Items: playlists,
	}, nil
}

// GetPlaylist retrieves a playlist along with its items.
func (s *FavouritePlaylistService) GetPlaylist(ctx context.Context, playlistID string) (*models.PlaylistDetail, error) {
	playlist, err := s.favRepo.GetPlaylistByID(ctx, playlistID)
	if err != nil {
		return nil, err
	}
	if playlist == nil {
		return nil, nil
	}

	trackIDs, err := s.favRepo.GetPlaylistTrackIDs(ctx, playlistID)
	if err != nil {
		return nil, err
	}

	tracks, err := s.trackRepo.GetByIDs(ctx, trackIDs)
	if err != nil {
		return nil, err
	}

	items := make([]*models.BrowseItem, 0, len(tracks))
	for _, t := range tracks {
		items = append(items, t.ToBrowseItem())
	}

	return &models.PlaylistDetail{
		Playlist: playlist,
		Tracks:   items,
	}, nil
}

// UpdatePlaylist updates metadata of a user playlist.
func (s *FavouritePlaylistService) UpdatePlaylist(ctx context.Context, playlistID string, userID int64, title, description, coverURL string) (*models.UserPlaylist, error) {
	return s.favRepo.UpdatePlaylist(ctx, playlistID, userID, title, description, coverURL)
}

// DeletePlaylist removes a playlist and its track relations.
func (s *FavouritePlaylistService) DeletePlaylist(ctx context.Context, playlistID string, userID int64) (bool, error) {
	return s.favRepo.DeletePlaylist(ctx, playlistID, userID)
}

// AddTracksToPlaylist adds multiple tracks to an existing playlist.
func (s *FavouritePlaylistService) AddTracksToPlaylist(ctx context.Context, playlistID string, trackIDs []string) error {
	return s.favRepo.AddTracksToPlaylist(ctx, playlistID, trackIDs)
}

// RemoveTrackFromPlaylist removes a single track from a playlist.
func (s *FavouritePlaylistService) RemoveTrackFromPlaylist(ctx context.Context, playlistID string, trackID string) error {
	return s.favRepo.RemoveTrackFromPlaylist(ctx, playlistID, trackID)
}

// GetPlaylistTracks returns the ordered tracks inside a playlist.
func (s *FavouritePlaylistService) GetPlaylistTracks(ctx context.Context, playlistID string) ([]*models.BrowseItem, error) {
	trackIDs, err := s.favRepo.GetPlaylistTrackIDs(ctx, playlistID)
	if err != nil {
		return nil, err
	}

	tracks, err := s.trackRepo.GetByIDs(ctx, trackIDs)
	if err != nil {
		return nil, err
	}

	items := make([]*models.BrowseItem, 0, len(tracks))
	for _, t := range tracks {
		items = append(items, t.ToBrowseItem())
	}
	return items, nil
}

// ReorderPlaylistTracks updates the order of tracks in a playlist.
func (s *FavouritePlaylistService) ReorderPlaylistTracks(ctx context.Context, playlistID string, trackIDs []string) error {
	return s.favRepo.ReorderPlaylistTracks(ctx, playlistID, trackIDs)
}

// RecordHistory stores an entry in listening history.
func (s *FavouritePlaylistService) RecordHistory(ctx context.Context, userID int64, trackID string, playedAt float64) error {
	return s.favRepo.RecordHistory(ctx, userID, trackID, playedAt)
}

// GetUserHistory returns unique recently played tracks for a user.
func (s *FavouritePlaylistService) GetUserHistory(ctx context.Context, userID int64, limit int) (*models.BrowseResponse, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}

	trackIDs, err := s.favRepo.GetUserHistoryTrackIDs(ctx, userID, limit)
	if err != nil {
		return nil, err
	}

	tracks, err := s.trackRepo.GetByIDs(ctx, trackIDs)
	if err != nil {
		return nil, err
	}

	items := make([]*models.BrowseItem, 0, len(tracks))
	for _, t := range tracks {
		items = append(items, t.ToBrowseItem())
	}

	return &models.BrowseResponse{
		Items:      items,
		Total:      int64(len(items)),
		Page:       1,
		PerPage:    limit,
		TotalPages: 1,
	}, nil
}

// RecordListeningEvents records telemetry playback events.
func (s *FavouritePlaylistService) RecordListeningEvents(ctx context.Context, userID int64, events []models.ListeningEventItem) error {
	return s.favRepo.RecordListeningEvents(ctx, userID, events)
}
