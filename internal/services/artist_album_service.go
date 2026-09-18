package services

import (
	"context"
	"math"

	"streamgo/internal/models"
	"streamgo/internal/repository"
)

// ArtistAlbumService orchestrates artist and album operations.
type ArtistAlbumService struct {
	repo      repository.ArtistAlbumRepository
	trackRepo repository.TrackRepository
}

// NewArtistAlbumService creates a new ArtistAlbumService.
func NewArtistAlbumService(repo repository.ArtistAlbumRepository, trackRepo repository.TrackRepository) *ArtistAlbumService {
	return &ArtistAlbumService{
		repo:      repo,
		trackRepo: trackRepo,
	}
}

// ListArtists returns a paginated list of artists.
func (s *ArtistAlbumService) ListArtists(ctx context.Context, page, perPage int) (*models.ArtistsResponse, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	artists, total, err := s.repo.ListArtists(ctx, page, perPage)
	if err != nil {
		return nil, err
	}

	totalPages := 0
	if total > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(perPage)))
	}

	return &models.ArtistsResponse{
		Items:      artists,
		Total:      total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: totalPages,
	}, nil
}

// GetArtistDetail returns detailed artist information with top tracks and albums.
func (s *ArtistAlbumService) GetArtistDetail(ctx context.Context, id string) (*models.ArtistDetail, error) {
	artist, err := s.repo.GetArtistByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if artist == nil {
		return nil, nil
	}

	tracks, err := s.repo.GetArtistTracks(ctx, artist.Name, 50)
	if err != nil {
		return nil, err
	}

	browseItems := make([]*models.BrowseItem, 0, len(tracks))
	for _, t := range tracks {
		browseItems = append(browseItems, t.ToBrowseItem())
	}

	albums, err := s.repo.GetArtistAlbums(ctx, artist.Name)
	if err != nil {
		return nil, err
	}

	return &models.ArtistDetail{
		Artist:    artist,
		TopTracks: browseItems,
		Albums:    albums,
	}, nil
}

// GetArtistTracks returns the top tracks for a specific artist ID.
func (s *ArtistAlbumService) GetArtistTracks(ctx context.Context, id string, limit int) ([]*models.BrowseItem, error) {
	artist, err := s.repo.GetArtistByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if artist == nil {
		return nil, nil
	}

	tracks, err := s.repo.GetArtistTracks(ctx, artist.Name, limit)
	if err != nil {
		return nil, err
	}

	items := make([]*models.BrowseItem, 0, len(tracks))
	for _, t := range tracks {
		items = append(items, t.ToBrowseItem())
	}
	return items, nil
}

// ListAlbums returns a paginated list of albums.
func (s *ArtistAlbumService) ListAlbums(ctx context.Context, page, perPage int, artistFilter string) (*models.AlbumsResponse, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	albums, total, err := s.repo.ListAlbums(ctx, page, perPage, artistFilter)
	if err != nil {
		return nil, err
	}

	totalPages := 0
	if total > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(perPage)))
	}

	return &models.AlbumsResponse{
		Items:      albums,
		Total:      total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: totalPages,
	}, nil
}

// GetAlbumDetail returns full album metadata along with its tracklist.
func (s *ArtistAlbumService) GetAlbumDetail(ctx context.Context, id string) (*models.AlbumDetail, error) {
	album, err := s.repo.GetAlbumByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if album == nil {
		return nil, nil
	}

	tracks, err := s.repo.GetAlbumTracks(ctx, album.ID)
	if err != nil {
		return nil, err
	}

	browseItems := make([]*models.BrowseItem, 0, len(tracks))
	for _, t := range tracks {
		browseItems = append(browseItems, t.ToBrowseItem())
	}

	return &models.AlbumDetail{
		Album:  album,
		Tracks: browseItems,
	}, nil
}

// GetAlbumTracks returns only the tracks of an album.
func (s *ArtistAlbumService) GetAlbumTracks(ctx context.Context, id string) ([]*models.BrowseItem, error) {
	tracks, err := s.repo.GetAlbumTracks(ctx, id)
	if err != nil {
		return nil, err
	}

	browseItems := make([]*models.BrowseItem, 0, len(tracks))
	for _, t := range tracks {
		browseItems = append(browseItems, t.ToBrowseItem())
	}
	return browseItems, nil
}
