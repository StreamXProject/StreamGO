package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"streamgo/internal/database"
	"streamgo/internal/models"
)

// FavouritePlaylistRepository defines data operations for favourites, playlists, and history.
type FavouritePlaylistRepository interface {
	// Track Favourites
	AddFavourite(ctx context.Context, userID int64, trackID string) (alreadyExists bool, err error)
	RemoveFavourite(ctx context.Context, userID int64, trackID string) (deleted bool, err error)
	GetFavouriteTrackIDs(ctx context.Context, userID int64, page, limit int) ([]string, int64, *float64, error)
	GetFavourites(ctx context.Context, userID int64, page, limit int) ([]*models.UserFavourite, int64, *float64, error)

	// Artist Favourites
	AddFavouriteArtist(ctx context.Context, userID int64, artistID string) (alreadyExists bool, err error)
	RemoveFavouriteArtist(ctx context.Context, userID int64, artistID string) (deleted bool, err error)
	GetFavouriteArtistIDs(ctx context.Context, userID int64) ([]string, error)

	// Playlists
	CreatePlaylist(ctx context.Context, p *models.UserPlaylist) error
	GetUserPlaylists(ctx context.Context, userID int64) ([]*models.UserPlaylist, error)
	GetPlaylistByID(ctx context.Context, playlistID string) (*models.UserPlaylist, error)
	UpdatePlaylist(ctx context.Context, playlistID string, userID int64, title, description, coverURL string) (*models.UserPlaylist, error)
	DeletePlaylist(ctx context.Context, playlistID string, userID int64) (bool, error)
	AddTracksToPlaylist(ctx context.Context, playlistID string, trackIDs []string) error
	RemoveTrackFromPlaylist(ctx context.Context, playlistID string, trackID string) error
	GetPlaylistTrackIDs(ctx context.Context, playlistID string) ([]string, error)
	ReorderPlaylistTracks(ctx context.Context, playlistID string, trackIDs []string) error

	// History & Listening Events
	RecordHistory(ctx context.Context, userID int64, trackID string, playedAt float64) error
	GetUserHistoryTrackIDs(ctx context.Context, userID int64, limit int) ([]string, error)
	RecordListeningEvents(ctx context.Context, userID int64, events []models.ListeningEventItem) error
}

type mongoFavouritePlaylistRepository struct {
	favCol         *mongo.Collection
	favArtistCol   *mongo.Collection
	artistsCol     *mongo.Collection
	playlistsCol   *mongo.Collection
	playlistTrkCol *mongo.Collection
	historyCol     *mongo.Collection
	eventsCol      *mongo.Collection
}

// NewFavouritePlaylistRepository returns an instance of FavouritePlaylistRepository.
func NewFavouritePlaylistRepository(db *database.Client) FavouritePlaylistRepository {
	return &mongoFavouritePlaylistRepository{
		favCol:         db.Collection("userFavourites"),
		favArtistCol:   db.Collection("user_favourite_artists"),
		artistsCol:     db.Collection("artists"),
		playlistsCol:   db.Collection("userPlaylists"),
		playlistTrkCol: db.Collection("playlistTracks"),
		historyCol:     db.Collection("userHistory"),
		eventsCol:      db.Collection("user_listening_events"),
	}
}

// AddFavourite adds a track to user favourites.
func (r *mongoFavouritePlaylistRepository) AddFavourite(ctx context.Context, userID int64, trackID string) (bool, error) {
	now := float64(time.Now().Unix())
	filter := bson.M{"user_id": userID, "track_id": trackID}
	update := bson.M{
		"$setOnInsert": bson.M{"created_at": now},
		"$set":         bson.M{"user_id": userID, "track_id": trackID, "updated_at": now},
	}
	opts := options.UpdateOne().SetUpsert(true)
	res, err := r.favCol.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return false, fmt.Errorf("failed to add favourite: %w", err)
	}
	alreadyExists := res.UpsertedID == nil
	return alreadyExists, nil
}

// RemoveFavourite deletes a track from user favourites.
func (r *mongoFavouritePlaylistRepository) RemoveFavourite(ctx context.Context, userID int64, trackID string) (bool, error) {
	filter := bson.M{"user_id": userID, "track_id": trackID}
	res, err := r.favCol.DeleteOne(ctx, filter)
	if err != nil {
		return false, fmt.Errorf("failed to remove favourite: %w", err)
	}
	return res.DeletedCount > 0, nil
}

// GetFavouriteTrackIDs returns paginated track IDs liked by the user.
func (r *mongoFavouritePlaylistRepository) GetFavouriteTrackIDs(ctx context.Context, userID int64, page, limit int) ([]string, int64, *float64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 1000 {
		limit = 200
	}
	filter := bson.M{"user_id": userID}
	total, err := r.favCol.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, nil, fmt.Errorf("failed to count favourites: %w", err)
	}

	skip := int64((page - 1) * limit)
	opts := options.Find().
		SetProjection(bson.M{"_id": 0, "track_id": 1, "created_at": 1, "updated_at": 1}).
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetSkip(skip).
		SetLimit(int64(limit))

	cursor, err := r.favCol.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, nil, fmt.Errorf("failed to query favourite track IDs: %w", err)
	}
	defer cursor.Close(ctx)

	var ids []string
	var maxTS *float64
	for cursor.Next(ctx) {
		var doc struct {
			TrackID   string   `bson:"track_id"`
			CreatedAt *float64 `bson:"created_at"`
			UpdatedAt *float64 `bson:"updated_at"`
		}
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		if doc.TrackID != "" {
			ids = append(ids, doc.TrackID)
		}
		ts := doc.UpdatedAt
		if ts == nil {
			ts = doc.CreatedAt
		}
		if ts != nil {
			if maxTS == nil || *ts > *maxTS {
				t := *ts
				maxTS = &t
			}
		}
	}

	return ids, total, maxTS, nil
}

// GetFavourites returns paginated user favourite entries.
func (r *mongoFavouritePlaylistRepository) GetFavourites(ctx context.Context, userID int64, page, limit int) ([]*models.UserFavourite, int64, *float64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	filter := bson.M{"user_id": userID}
	total, err := r.favCol.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, nil, err
	}

	skip := int64((page - 1) * limit)
	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetSkip(skip).
		SetLimit(int64(limit))

	cursor, err := r.favCol.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, nil, err
	}
	defer cursor.Close(ctx)

	var items []*models.UserFavourite
	var maxTS *float64
	for cursor.Next(ctx) {
		var fav models.UserFavourite
		if err := cursor.Decode(&fav); err != nil {
			continue
		}
		items = append(items, &fav)
		if maxTS == nil || fav.CreatedAt > *maxTS {
			t := fav.CreatedAt
			maxTS = &t
		}
	}
	return items, total, maxTS, nil
}

// AddFavouriteArtist adds an artist to user favourites and updates follower count.
func (r *mongoFavouritePlaylistRepository) AddFavouriteArtist(ctx context.Context, userID int64, artistID string) (bool, error) {
	now := float64(time.Now().Unix())
	filter := bson.M{"user_id": userID, "artist_id": artistID}
	update := bson.M{
		"$setOnInsert": bson.M{"created_at": now},
		"$set":         bson.M{"user_id": userID, "artist_id": artistID, "updated_at": now},
	}
	opts := options.UpdateOne().SetUpsert(true)
	res, err := r.favArtistCol.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return false, err
	}
	alreadyExists := res.UpsertedID == nil
	if !alreadyExists {
		_, _ = r.artistsCol.UpdateOne(ctx, bson.M{"_id": artistID}, bson.M{"$inc": bson.M{"followers": 1}})
	}
	return alreadyExists, nil
}

// RemoveFavouriteArtist removes an artist from favourites and decrements follower count.
func (r *mongoFavouritePlaylistRepository) RemoveFavouriteArtist(ctx context.Context, userID int64, artistID string) (bool, error) {
	filter := bson.M{"user_id": userID, "artist_id": artistID}
	res, err := r.favArtistCol.DeleteOne(ctx, filter)
	if err != nil {
		return false, err
	}
	if res.DeletedCount > 0 {
		_, _ = r.artistsCol.UpdateOne(ctx, bson.M{"_id": artistID}, bson.M{"$inc": bson.M{"followers": -1}})
		return true, nil
	}
	return false, nil
}

// GetFavouriteArtistIDs returns list of followed artist IDs for a user.
func (r *mongoFavouritePlaylistRepository) GetFavouriteArtistIDs(ctx context.Context, userID int64) ([]string, error) {
	opts := options.Find().SetProjection(bson.M{"_id": 0, "artist_id": 1})
	cursor, err := r.favArtistCol.Find(ctx, bson.M{"user_id": userID}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var ids []string
	for cursor.Next(ctx) {
		var doc struct {
			ArtistID string `bson:"artist_id"`
		}
		if err := cursor.Decode(&doc); err == nil && doc.ArtistID != "" {
			ids = append(ids, doc.ArtistID)
		}
	}
	return ids, nil
}

// CreatePlaylist stores a new custom playlist.
func (r *mongoFavouritePlaylistRepository) CreatePlaylist(ctx context.Context, p *models.UserPlaylist) error {
	now := float64(time.Now().Unix())
	if p.CreatedAt == 0 {
		p.CreatedAt = now
	}
	p.UpdatedAt = now
	_, err := r.playlistsCol.InsertOne(ctx, p)
	return err
}

// GetUserPlaylists lists all playlists owned by a user.
func (r *mongoFavouritePlaylistRepository) GetUserPlaylists(ctx context.Context, userID int64) ([]*models.UserPlaylist, error) {
	opts := options.Find().SetSort(bson.D{{Key: "updated_at", Value: -1}})
	cursor, err := r.playlistsCol.Find(ctx, bson.M{"user_id": userID}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var items []*models.UserPlaylist
	if err := cursor.All(ctx, &items); err != nil {
		return nil, err
	}
	return items, nil
}

// GetPlaylistByID fetches a playlist document by ID.
func (r *mongoFavouritePlaylistRepository) GetPlaylistByID(ctx context.Context, playlistID string) (*models.UserPlaylist, error) {
	var p models.UserPlaylist
	err := r.playlistsCol.FindOne(ctx, bson.M{"_id": playlistID}).Decode(&p)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

// UpdatePlaylist updates title, description, or cover of a playlist.
func (r *mongoFavouritePlaylistRepository) UpdatePlaylist(ctx context.Context, playlistID string, userID int64, title, description, coverURL string) (*models.UserPlaylist, error) {
	now := float64(time.Now().Unix())
	update := bson.M{
		"$set": bson.M{
			"updated_at": now,
		},
	}
	setFields := update["$set"].(bson.M)
	if title != "" {
		setFields["title"] = title
	}
	if description != "" {
		setFields["description"] = description
	}
	if coverURL != "" {
		setFields["cover_url"] = coverURL
	}

	filter := bson.M{"_id": playlistID, "user_id": userID}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	var updated models.UserPlaylist
	err := r.playlistsCol.FindOneAndUpdate(ctx, filter, update, opts).Decode(&updated)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &updated, nil
}

// DeletePlaylist deletes a playlist and all its associated track links.
func (r *mongoFavouritePlaylistRepository) DeletePlaylist(ctx context.Context, playlistID string, userID int64) (bool, error) {
	filter := bson.M{"_id": playlistID, "user_id": userID}
	res, err := r.playlistsCol.DeleteOne(ctx, filter)
	if err != nil {
		return false, err
	}
	if res.DeletedCount > 0 {
		_, _ = r.playlistTrkCol.DeleteMany(ctx, bson.M{"playlist_id": playlistID})
		return true, nil
	}
	return false, nil
}

// AddTracksToPlaylist inserts tracks into playlistTracks and updates playlist track_count.
func (r *mongoFavouritePlaylistRepository) AddTracksToPlaylist(ctx context.Context, playlistID string, trackIDs []string) error {
	if len(trackIDs) == 0 {
		return nil
	}
	now := float64(time.Now().Unix())

	// Find current max position
	opts := options.FindOne().SetSort(bson.D{{Key: "position", Value: -1}}).SetProjection(bson.M{"position": 1})
	var lastDoc struct {
		Position int `bson:"position"`
	}
	pos := 0
	if err := r.playlistTrkCol.FindOne(ctx, bson.M{"playlist_id": playlistID}, opts).Decode(&lastDoc); err == nil {
		pos = lastDoc.Position + 1
	}

	for _, tid := range trackIDs {
		id := fmt.Sprintf("%s_%s", playlistID, tid)
		pt := models.PlaylistTrack{
			ID:         id,
			PlaylistID: playlistID,
			TrackID:    tid,
			Position:   pos,
			AddedAt:    now,
		}
		_, err := r.playlistTrkCol.UpdateOne(
			ctx,
			bson.M{"_id": id},
			bson.M{"$set": pt},
			options.UpdateOne().SetUpsert(true),
		)
		if err != nil {
			return err
		}
		pos++
	}

	count, _ := r.playlistTrkCol.CountDocuments(ctx, bson.M{"playlist_id": playlistID})
	_, _ = r.playlistsCol.UpdateOne(ctx, bson.M{"_id": playlistID}, bson.M{"$set": bson.M{"track_count": count, "updated_at": now}})
	return nil
}

// RemoveTrackFromPlaylist deletes a track relation from playlist.
func (r *mongoFavouritePlaylistRepository) RemoveTrackFromPlaylist(ctx context.Context, playlistID string, trackID string) error {
	id := fmt.Sprintf("%s_%s", playlistID, trackID)
	_, err := r.playlistTrkCol.DeleteOne(ctx, bson.M{"$or": []bson.M{{"_id": id}, {"playlist_id": playlistID, "track_id": trackID}}})
	if err != nil {
		return err
	}
	now := float64(time.Now().Unix())
	count, _ := r.playlistTrkCol.CountDocuments(ctx, bson.M{"playlist_id": playlistID})
	_, _ = r.playlistsCol.UpdateOne(ctx, bson.M{"_id": playlistID}, bson.M{"$set": bson.M{"track_count": count, "updated_at": now}})
	return nil
}

// GetPlaylistTrackIDs returns ordered track IDs for a playlist.
func (r *mongoFavouritePlaylistRepository) GetPlaylistTrackIDs(ctx context.Context, playlistID string) ([]string, error) {
	opts := options.Find().
		SetSort(bson.D{{Key: "position", Value: 1}, {Key: "added_at", Value: 1}}).
		SetProjection(bson.M{"track_id": 1})

	cursor, err := r.playlistTrkCol.Find(ctx, bson.M{"playlist_id": playlistID}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var ids []string
	for cursor.Next(ctx) {
		var doc struct {
			TrackID string `bson:"track_id"`
		}
		if err := cursor.Decode(&doc); err == nil && doc.TrackID != "" {
			ids = append(ids, doc.TrackID)
		}
	}
	return ids, nil
}

// ReorderPlaylistTracks updates the position of tracks in the playlist according to trackIDs order.
func (r *mongoFavouritePlaylistRepository) ReorderPlaylistTracks(ctx context.Context, playlistID string, trackIDs []string) error {
	now := float64(time.Now().Unix())
	for i, tid := range trackIDs {
		id := fmt.Sprintf("%s_%s", playlistID, tid)
		_, _ = r.playlistTrkCol.UpdateOne(
			ctx,
			bson.M{"$or": []bson.M{{"_id": id}, {"playlist_id": playlistID, "track_id": tid}}},
			bson.M{"$set": bson.M{"position": i}},
		)
	}
	_, _ = r.playlistsCol.UpdateOne(ctx, bson.M{"_id": playlistID}, bson.M{"$set": bson.M{"updated_at": now}})
	return nil
}

// RecordHistory stores an entry in userHistory.
func (r *mongoFavouritePlaylistRepository) RecordHistory(ctx context.Context, userID int64, trackID string, playedAt float64) error {
	if playedAt == 0 {
		playedAt = float64(time.Now().Unix())
	}
	doc := bson.M{
		"user_id":   userID,
		"track_id":  trackID,
		"played_at": playedAt,
	}
	_, err := r.historyCol.InsertOne(ctx, doc)
	return err
}

// GetUserHistoryTrackIDs returns recent unique played track IDs.
func (r *mongoFavouritePlaylistRepository) GetUserHistoryTrackIDs(ctx context.Context, userID int64, limit int) ([]string, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	opts := options.Find().
		SetSort(bson.D{{Key: "played_at", Value: -1}}).
		SetLimit(int64(limit * 2)).
		SetProjection(bson.M{"track_id": 1})

	filter := bson.M{"$or": []bson.M{{"user_id": userID}, {"user_id": fmt.Sprintf("%d", userID)}}}
	cursor, err := r.historyCol.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	seen := make(map[string]bool)
	var ids []string
	for cursor.Next(ctx) {
		var doc struct {
			TrackID string `bson:"track_id"`
		}
		if err := cursor.Decode(&doc); err == nil && doc.TrackID != "" {
			if !seen[doc.TrackID] {
				seen[doc.TrackID] = true
				ids = append(ids, doc.TrackID)
				if len(ids) >= limit {
					break
				}
			}
		}
	}
	return ids, nil
}

// RecordListeningEvents saves a batch of telemetry listening events into user_listening_events.
func (r *mongoFavouritePlaylistRepository) RecordListeningEvents(ctx context.Context, userID int64, events []models.ListeningEventItem) error {
	if len(events) == 0 {
		return nil
	}
	var docs []any
	for _, ev := range events {
		d := bson.M{
			"id":          ev.ID,
			"user_id":     userID,
			"track_id":    ev.TrackID,
			"played_at":   ev.PlayedAt,
			"started_at":  ev.StartedAt,
			"played_ms":   ev.PlayedMs,
			"duration_ms": ev.DurationMs,
			"completed":   ev.Completed,
			"skipped":     ev.Skipped,
			"source":      ev.Source,
			"session_id":  ev.SessionID,
			"created_at":  float64(time.Now().Unix()),
		}
		docs = append(docs, d)
	}
	_, err := r.eventsCol.InsertMany(ctx, docs)
	return err
}
