package services

import (
	"context"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"streamgo/internal/config"
	"streamgo/internal/database"
)

const (
	FilterModeGroupOnly = 0
	FilterModeAnyone    = 1
	FilterModeHybrid    = 2
)

// AccessFilter checks chat access rights before indexing or streaming media.
type AccessFilter struct {
	cfg       *config.Config
	col       *mongo.Collection
	bannedCol *mongo.Collection

	mu           sync.RWMutex
	allowedChats map[int64]bool
	bannedUsers  map[int64]bool
	lastRefresh  time.Time
}

// NewAccessFilter creates an AccessFilter instance.
func NewAccessFilter(cfg *config.Config, db *database.Client) *AccessFilter {
	var col, bannedCol *mongo.Collection
	if db != nil {
		col = db.Collection("source_chats")
		bannedCol = db.Collection("banned_users")
	}

	af := &AccessFilter{
		cfg:          cfg,
		col:          col,
		bannedCol:    bannedCol,
		allowedChats: make(map[int64]bool),
		bannedUsers:  make(map[int64]bool),
	}

	if cfg.ChannelID != 0 {
		af.allowedChats[cfg.ChannelID] = true
	}
	if cfg.DumpChannelID != 0 {
		af.allowedChats[cfg.DumpChannelID] = true
	}

	return af
}

// RefreshCache loads allowed and banned lists from MongoDB.
func (f *AccessFilter) RefreshCache(ctx context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.col != nil {
		cursor, err := f.col.Find(ctx, bson.M{})
		if err == nil {
			defer cursor.Close(ctx)
			newAllowed := make(map[int64]bool)
			if f.cfg.ChannelID != 0 {
				newAllowed[f.cfg.ChannelID] = true
			}
			if f.cfg.DumpChannelID != 0 {
				newAllowed[f.cfg.DumpChannelID] = true
			}

			for cursor.Next(ctx) {
				var doc struct {
					ChatID int64 `bson:"chat_id"`
					ID     int64 `bson:"_id"`
				}
				if err := cursor.Decode(&doc); err == nil {
					cid := doc.ChatID
					if cid == 0 {
						cid = doc.ID
					}
					if cid != 0 {
						newAllowed[cid] = true
					}
				}
			}
			f.allowedChats = newAllowed
		}
	}

	f.lastRefresh = time.Now()
	return nil
}

// IsChatAllowed determines if an incoming message or channel is authorized.
func (f *AccessFilter) IsChatAllowed(ctx context.Context, chatID int64) bool {
	mode := f.cfg.FilterMode

	// Mode 1: Anyone is allowed
	if mode == FilterModeAnyone {
		return true
	}

	f.mu.RLock()
	if time.Since(f.lastRefresh) > 60*time.Second {
		f.mu.RUnlock()
		_ = f.RefreshCache(ctx)
		f.mu.RLock()
	}
	allowed := f.allowedChats[chatID]
	f.mu.RUnlock()

	return allowed
}
