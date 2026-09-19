package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"streamgo/internal/database"
	"streamgo/internal/models"
)

// HistoryRepository manages user playback history and listening telemetry.
type HistoryRepository interface {
	RecordPlay(ctx context.Context, userID int64, trackID string, source string) error
	GetUserHistory(ctx context.Context, userID int64, limit int) ([]string, error)
	GetUserTopPlayed(ctx context.Context, userID int64, limit int) ([]string, error)
	RecordListeningEvents(ctx context.Context, userID int64, events []models.ListeningEventItem) error
}

type mongoHistoryRepository struct {
	histCol   *mongo.Collection
	userPbCol *mongo.Collection
	globPbCol *mongo.Collection
	eventsCol *mongo.Collection
	tracksCol *mongo.Collection
}

// NewHistoryRepository creates a MongoDB HistoryRepository.
func NewHistoryRepository(db *database.Client) HistoryRepository {
	return &mongoHistoryRepository{
		histCol:   db.Collection("userHistory"),
		userPbCol: db.Collection("userPlayback"),
		globPbCol: db.Collection("globalPlayback"),
		eventsCol: db.Collection("listening_events"),
		tracksCol: db.Collection("audioTracks"),
	}
}

// RecordPlay records a playback entry in userHistory, userPlayback, and globalPlayback.
func (r *mongoHistoryRepository) RecordPlay(ctx context.Context, userID int64, trackID string, source string) error {
	trackID = strings.TrimSpace(trackID)
	if trackID == "" {
		return nil
	}

	now := float64(time.Now().Unix())

	// 1. Log to userHistory
	if userID > 0 {
		_, _ = r.histCol.InsertOne(ctx, bson.M{
			"user_id":   userID,
			"track_id":  trackID,
			"played_at": now,
		})

		// 2. Update userPlayback bucket (bucket = 30-min window)
		bucket := int64(now) / 1800
		userPlayID := fmt.Sprintf("%d:%s:%d", userID, trackID, bucket)
		_, _ = r.userPbCol.UpdateOne(
			ctx,
			bson.M{"_id": userPlayID},
			bson.M{
				"$setOnInsert": bson.M{
					"_id":       userPlayID,
					"user_id":   userID,
					"track_id":  trackID,
					"played_at": now,
					"bucket":    bucket,
					"source":    source,
				},
				"$inc": bson.M{"plays": 1},
			},
			options.UpdateOne().SetUpsert(true),
		)
	}

	// 3. Update globalPlayback
	inc := bson.M{"plays": 1}
	if source != "" {
		inc["sources."+source] = 1
	}
	_, _ = r.globPbCol.UpdateOne(
		ctx,
		bson.M{"_id": trackID},
		bson.M{
			"$inc": inc,
			"$set": bson.M{
				"last_played_at": now,
				"updated_at":     now,
			},
		},
		options.UpdateOne().SetUpsert(true),
	)

	// 4. Increment track play_count in tracks collection
	_, _ = r.tracksCol.UpdateOne(
		ctx,
		bson.M{"_id": trackID},
		bson.M{"$inc": bson.M{"audio.play_count": 1}},
	)

	return nil
}

// GetUserHistory retrieves unique track IDs for a user sorted by played_at desc.
func (r *mongoHistoryRepository) GetUserHistory(ctx context.Context, userID int64, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	filter := bson.M{"$or": []bson.M{
		{"user_id": userID},
		{"user_id": fmt.Sprintf("%d", userID)},
	}}

	opts := options.Find().
		SetSort(bson.D{{Key: "played_at", Value: -1}}).
		SetLimit(int64(limit * 3))

	cursor, err := r.histCol.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var orderedIDs []string
	seen := make(map[string]bool)

	for cursor.Next(ctx) {
		var doc struct {
			TrackID string `bson:"track_id"`
		}
		if err := cursor.Decode(&doc); err == nil && doc.TrackID != "" {
			if !seen[doc.TrackID] {
				seen[doc.TrackID] = true
				orderedIDs = append(orderedIDs, doc.TrackID)
				if len(orderedIDs) >= limit {
					break
				}
			}
		}
	}

	return orderedIDs, nil
}

// GetUserTopPlayed retrieves track IDs most frequently played by the user.
func (r *mongoHistoryRepository) GetUserTopPlayed(ctx context.Context, userID int64, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"user_id": userID}}},
		{{Key: "$group", Value: bson.M{
			"_id":   "$track_id",
			"plays": bson.M{"$sum": "$plays"},
		}}},
		{{Key: "$sort", Value: bson.D{{Key: "plays", Value: -1}}}},
		{{Key: "$limit", Value: limit}},
	}

	cursor, err := r.userPbCol.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var trackIDs []string
	for cursor.Next(ctx) {
		var doc struct {
			TrackID string `bson:"_id"`
		}
		if err := cursor.Decode(&doc); err == nil && doc.TrackID != "" {
			trackIDs = append(trackIDs, doc.TrackID)
		}
	}

	return trackIDs, nil
}

// RecordListeningEvents saves telemetry events and triggers play count updates.
func (r *mongoHistoryRepository) RecordListeningEvents(ctx context.Context, userID int64, events []models.ListeningEventItem) error {
	now := float64(time.Now().Unix())
	for _, ev := range events {
		eid := strings.TrimSpace(ev.ID)
		if eid == "" {
			eid = fmt.Sprintf("ev_%d_%s_%d", userID, ev.TrackID, time.Now().UnixNano())
		}

		filter := bson.M{"event_id": eid}
		if userID > 0 {
			filter["user_id"] = userID
		}

		update := bson.M{
			"$setOnInsert": bson.M{"created_at": now},
			"$set": bson.M{
				"event_id":    eid,
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
			},
		}

		_, _ = r.eventsCol.UpdateOne(ctx, filter, update, options.UpdateOne().SetUpsert(true))

		// Record play in userHistory, userPlayback, and global metrics
		if ev.TrackID != "" {
			_ = r.RecordPlay(ctx, userID, ev.TrackID, ev.Source)
		}
	}

	return nil
}
