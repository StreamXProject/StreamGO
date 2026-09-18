package services

import (
	"context"
	"math"

	"streamgo/internal/models"
	"streamgo/internal/repository"
)

// TrackService coordinates track-related business operations.
type TrackService struct {
	repo repository.TrackRepository
}

// NewTrackService creates a new TrackService with the given repository.
func NewTrackService(repo repository.TrackRepository) *TrackService {
	return &TrackService{repo: repo}
}

// GetTrack retrieves a single track by its ID.
func (s *TrackService) GetTrack(ctx context.Context, id string) (*models.Track, error) {
	return s.repo.GetByID(ctx, id)
}

// Browse retrieves a paginated feed of tracks converted to lightweight BrowseItems.
func (s *TrackService) Browse(
	ctx context.Context,
	page, perPage int,
	sortField, topicName string,
	channelID int64,
) (*models.BrowseResponse, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	tracks, total, err := s.repo.List(ctx, page, perPage, sortField, topicName, channelID)
	if err != nil {
		return nil, err
	}

	items := make([]*models.BrowseItem, 0, len(tracks))
	for _, t := range tracks {
		items = append(items, t.ToBrowseItem())
	}

	totalPages := 0
	if total > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(perPage)))
	}

	return &models.BrowseResponse{
		Items:      items,
		Total:      total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: totalPages,
	}, nil
}

// Search queries tracks by artist, title, or album matching.
func (s *TrackService) Search(ctx context.Context, query string, limit int) ([]*models.BrowseItem, error) {
	tracks, err := s.repo.Search(ctx, query, limit)
	if err != nil {
		return nil, err
	}

	items := make([]*models.BrowseItem, 0, len(tracks))
	for _, t := range tracks {
		items = append(items, t.ToBrowseItem())
	}

	return items, nil
}

// GetRandom retrieves a randomized selection of tracks.
func (s *TrackService) GetRandom(ctx context.Context, limit int, channelID int64) ([]*models.BrowseItem, error) {
	tracks, err := s.repo.Random(ctx, limit, channelID)
	if err != nil {
		return nil, err
	}

	items := make([]*models.BrowseItem, 0, len(tracks))
	for _, t := range tracks {
		items = append(items, t.ToBrowseItem())
	}

	return items, nil
}

// GetTopics returns aggregated topic lists with counts.
func (s *TrackService) GetTopics(ctx context.Context, limit int) (*models.TopicsResponse, error) {
	topics, err := s.repo.GetTopics(ctx, limit)
	if err != nil {
		return nil, err
	}

	return &models.TopicsResponse{
		OK:    true,
		Total: len(topics),
		Items: topics,
	}, nil
}

// GetChannelIDs returns all unique indexed Telegram source channel IDs.
func (s *TrackService) GetChannelIDs(ctx context.Context) (*models.ChannelIDsResponse, error) {
	cids, err := s.repo.GetChannelIDs(ctx)
	if err != nil {
		return nil, err
	}

	items := make([]models.ChannelItem, 0, len(cids))
	for _, id := range cids {
		items = append(items, models.ChannelItem{
			ID: id,
		})
	}

	return &models.ChannelIDsResponse{
		OK:    true,
		Items: items,
	}, nil
}
