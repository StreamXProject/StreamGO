package services

import (
	"context"
	"strconv"
	"strings"
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

// ParseFilterMode parses string or int values into valid filter mode constants.
func ParseFilterMode(val any) int {
	if val == nil {
		return FilterModeGroupOnly
	}
	switch v := val.(type) {
	case int:
		if v == FilterModeAnyone || v == FilterModeHybrid {
			return v
		}
		return FilterModeGroupOnly
	case int64:
		if v == int64(FilterModeAnyone) || v == int64(FilterModeHybrid) {
			return int(v)
		}
		return FilterModeGroupOnly
	case string:
		s := strings.ToLower(strings.TrimSpace(v))
		switch s {
		case "0", "group_only", "group", "channel_only":
			return FilterModeGroupOnly
		case "1", "anyone", "all", "any":
			return FilterModeAnyone
		case "2", "hybrid", "allowlist", "whitelist":
			return FilterModeHybrid
		default:
			if n, err := strconv.Atoi(s); err == nil {
				if n == FilterModeAnyone || n == FilterModeHybrid {
					return n
				}
			}
			return FilterModeGroupOnly
		}
	}
	return FilterModeGroupOnly
}

// FilterModeToString converts a filter mode integer to its canonical string representation.
func FilterModeToString(mode int) string {
	switch mode {
	case FilterModeAnyone:
		return "anyone"
	case FilterModeHybrid:
		return "hybrid"
	default:
		return "group_only"
	}
}

// AccessFilter checks chat access rights before indexing or streaming media.
type AccessFilter struct {
	cfg       *config.Config
	col       *mongo.Collection
	bannedCol *mongo.Collection

	mu           sync.RWMutex
	mode         int
	allowedChats map[int64]bool
	bannedChats  map[int64]bool
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

	mode := FilterModeGroupOnly
	if cfg != nil {
		mode = cfg.FilterMode
	}

	af := &AccessFilter{
		cfg:          cfg,
		col:          col,
		bannedCol:    bannedCol,
		mode:         mode,
		allowedChats: make(map[int64]bool),
		bannedChats:  make(map[int64]bool),
		bannedUsers:  make(map[int64]bool),
	}

	if cfg != nil {
		if cfg.ChannelID != 0 {
			af.allowedChats[cfg.ChannelID] = true
		}
		if cfg.DumpChannelID != 0 {
			af.allowedChats[cfg.DumpChannelID] = true
		}
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
			if f.cfg != nil {
				if f.cfg.ChannelID != 0 {
					newAllowed[f.cfg.ChannelID] = true
				}
				if f.cfg.DumpChannelID != 0 {
					newAllowed[f.cfg.DumpChannelID] = true
				}
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

	if f.bannedCol != nil {
		bCursor, err := f.bannedCol.Find(ctx, bson.M{})
		if err == nil {
			defer bCursor.Close(ctx)
			newBanned := make(map[int64]bool)
			for bCursor.Next(ctx) {
				var bDoc struct {
					ChatID int64 `bson:"chat_id"`
					UserID int64 `bson:"user_id"`
					ID     int64 `bson:"_id"`
				}
				if err := bCursor.Decode(&bDoc); err == nil {
					id := bDoc.ChatID
					if id == 0 {
						id = bDoc.UserID
					}
					if id == 0 {
						id = bDoc.ID
					}
					if id != 0 {
						newBanned[id] = true
					}
				}
			}
			f.bannedChats = newBanned
			f.bannedUsers = newBanned
		}
	}

	f.lastRefresh = time.Now()
	return nil
}

// IsChatAllowed determines if an incoming message or channel is authorized.
func (f *AccessFilter) IsChatAllowed(ctx context.Context, chatID int64) bool {
	f.mu.RLock()
	if f.bannedChats[chatID] || f.bannedUsers[chatID] {
		f.mu.RUnlock()
		return false
	}

	mode := FilterModeGroupOnly
	if f.cfg != nil {
		mode = f.cfg.FilterMode
	}
	if f.mode != FilterModeGroupOnly {
		mode = f.mode
	}

	if mode == FilterModeAnyone {
		f.mu.RUnlock()
		return true
	}

	if time.Since(f.lastRefresh) > 60*time.Second && f.col != nil {
		f.mu.RUnlock()
		_ = f.RefreshCache(ctx)
		f.mu.RLock()
	}

	allowed := f.allowedChats[chatID]
	f.mu.RUnlock()

	return allowed
}
