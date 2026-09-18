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

// UserRepository manages database operations for users.
type UserRepository interface {
	UpsertUser(ctx context.Context, user *models.User) (*models.User, error)
	GetUserByID(ctx context.Context, id int64) (*models.User, error)
}

type mongoUserRepository struct {
	usersCol *mongo.Collection
}

// NewUserRepository creates a UserRepository instance.
func NewUserRepository(db *database.Client) UserRepository {
	return &mongoUserRepository{
		usersCol: db.Collection("users"),
	}
}

// UpsertUser inserts or updates a user document based on user ID.
func (r *mongoUserRepository) UpsertUser(ctx context.Context, u *models.User) (*models.User, error) {
	now := float64(time.Now().Unix())
	filter := bson.M{"_id": u.ID}

	update := bson.M{
		"$set": bson.M{
			"first_name":  u.FirstName,
			"username":    u.Username,
			"photo_url":   u.PhotoURL,
			"profile_url": u.ProfileURL,
			"updated_at":  now,
		},
		"$setOnInsert": bson.M{
			"created_at": now,
		},
	}

	if u.Telegram != nil {
		update["$set"].(bson.M)["telegram"] = u.Telegram
	}
	if u.Status != "" {
		update["$set"].(bson.M)["status"] = u.Status
	}

	opts := options.FindOneAndUpdate().
		SetUpsert(true).
		SetReturnDocument(options.After)

	var result models.User
	err := r.usersCol.FindOneAndUpdate(ctx, filter, update, opts).Decode(&result)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert user %d: %w", u.ID, err)
	}

	return &result, nil
}

// GetUserByID fetches a user by Telegram numeric ID.
func (r *mongoUserRepository) GetUserByID(ctx context.Context, id int64) (*models.User, error) {
	filter := bson.M{"_id": id}
	var u models.User
	err := r.usersCol.FindOne(ctx, filter).Decode(&u)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to fetch user %d: %w", id, err)
	}
	return &u, nil
}
