package repository

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"streamgo/internal/database"
	"streamgo/internal/models"
)

// ArtistAlbumRepository manages data access for artists and albums.
type ArtistAlbumRepository interface {
	ListArtists(ctx context.Context, page, perPage int) ([]*models.Artist, int64, error)
	GetArtistByID(ctx context.Context, id string) (*models.Artist, error)
	GetArtistTracks(ctx context.Context, artistName string, limit int) ([]*models.Track, error)
	GetArtistAlbums(ctx context.Context, artistName string) ([]*models.Album, error)

	ListAlbums(ctx context.Context, page, perPage int, artistFilter string) ([]*models.Album, int64, error)
	GetAlbumByID(ctx context.Context, id string) (*models.Album, error)
	GetAlbumTracks(ctx context.Context, albumID string) ([]*models.Track, error)
}

type mongoArtistAlbumRepository struct {
	artistsCol *mongo.Collection
	albumsCol  *mongo.Collection
	tracksCol  *mongo.Collection
}

// NewArtistAlbumRepository creates an ArtistAlbumRepository backed by MongoDB.
func NewArtistAlbumRepository(db *database.Client) ArtistAlbumRepository {
	return &mongoArtistAlbumRepository{
		artistsCol: db.Collection("artists"),
		albumsCol:  db.Collection("albums"),
		tracksCol:  db.Collection("audioTracks"),
	}
}

func (r *mongoArtistAlbumRepository) ListArtists(ctx context.Context, page, perPage int) ([]*models.Artist, int64, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	filter := bson.M{"tracks_count": bson.M{"$gt": 0}}

	total, err := r.artistsCol.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count artists: %w", err)
	}

	skip := int64((page - 1) * perPage)
	opts := options.Find().
		SetSort(bson.D{{Key: "tracks_count", Value: -1}, {Key: "updated_at", Value: -1}}).
		SetSkip(skip).
		SetLimit(int64(perPage))

	cursor, err := r.artistsCol.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list artists: %w", err)
	}
	defer cursor.Close(ctx)

	var artists []*models.Artist
	if err := cursor.All(ctx, &artists); err != nil {
		return nil, 0, fmt.Errorf("failed to decode artists: %w", err)
	}

	return artists, total, nil
}

func (r *mongoArtistAlbumRepository) GetArtistByID(ctx context.Context, id string) (*models.Artist, error) {
	id = strings.TrimSpace(id)
	filter := bson.M{"$or": []bson.M{{"_id": id}, {"match_artist": strings.ToLower(id)}}}

	var artist models.Artist
	err := r.artistsCol.FindOne(ctx, filter).Decode(&artist)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to fetch artist %s: %w", id, err)
	}

	return &artist, nil
}

func (r *mongoArtistAlbumRepository) GetArtistTracks(ctx context.Context, artistName string, limit int) ([]*models.Track, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	escaped := regexp.QuoteMeta(artistName)
	regex := bson.M{"$regex": "^" + escaped + "$", "$options": "i"}

	filter := bson.M{
		"deleted": bson.M{"$ne": true},
		"$or": []bson.M{
			{"audio.artist": regex},
			{"audio.artists": regex},
			{"audio.performer": regex},
		},
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "play_count", Value: -1}, {Key: "updated_at", Value: -1}}).
		SetLimit(int64(limit))

	cursor, err := r.tracksCol.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var tracks []*models.Track
	if err := cursor.All(ctx, &tracks); err != nil {
		return nil, err
	}
	return tracks, nil
}

func (r *mongoArtistAlbumRepository) GetArtistAlbums(ctx context.Context, artistName string) ([]*models.Album, error) {
	escaped := regexp.QuoteMeta(artistName)
	regex := bson.M{"$regex": "^" + escaped + "$", "$options": "i"}

	filter := bson.M{
		"$or": []bson.M{
			{"artist": regex},
			{"artists": regex},
			{"match_artist": strings.ToLower(artistName)},
		},
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "updated_at", Value: -1}}).
		SetLimit(50)

	cursor, err := r.albumsCol.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var albums []*models.Album
	if err := cursor.All(ctx, &albums); err != nil {
		return nil, err
	}
	return albums, nil
}

func (r *mongoArtistAlbumRepository) ListAlbums(ctx context.Context, page, perPage int, artistFilter string) ([]*models.Album, int64, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	filter := bson.M{}
	if artistFilter != "" {
		escaped := regexp.QuoteMeta(artistFilter)
		filter["artist"] = bson.M{"$regex": escaped, "$options": "i"}
	}

	total, err := r.albumsCol.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	skip := int64((page - 1) * perPage)
	opts := options.Find().
		SetSort(bson.D{{Key: "updated_at", Value: -1}}).
		SetSkip(skip).
		SetLimit(int64(perPage))

	cursor, err := r.albumsCol.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var albums []*models.Album
	if err := cursor.All(ctx, &albums); err != nil {
		return nil, 0, err
	}
	return albums, total, nil
}

func (r *mongoArtistAlbumRepository) GetAlbumByID(ctx context.Context, id string) (*models.Album, error) {
	id = strings.TrimSpace(id)
	var album models.Album
	err := r.albumsCol.FindOne(ctx, bson.M{"_id": id}).Decode(&album)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &album, nil
}

func (r *mongoArtistAlbumRepository) GetAlbumTracks(ctx context.Context, albumID string) ([]*models.Track, error) {
	filter := bson.M{
		"deleted":        bson.M{"$ne": true},
		"audio.album_id": albumID,
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "updated_at", Value: 1}}).
		SetLimit(100)

	cursor, err := r.tracksCol.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var tracks []*models.Track
	if err := cursor.All(ctx, &tracks); err != nil {
		return nil, err
	}
	return tracks, nil
}
