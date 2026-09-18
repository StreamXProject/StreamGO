package models

// AudioMeta represents metadata extracted from the audio stream or tags.
type AudioMeta struct {
	Title          string   `bson:"title,omitempty" json:"title"`
	Artist         string   `bson:"artist,omitempty" json:"artist"`
	Performer      string   `bson:"performer,omitempty" json:"performer,omitempty"`
	Album          string   `bson:"album,omitempty" json:"album,omitempty"`
	AlbumID        string   `bson:"album_id,omitempty" json:"album_id,omitempty"`
	Artists        []string `bson:"artists,omitempty" json:"artists,omitempty"`
	DurationSec    float64  `bson:"duration_sec,omitempty" json:"duration_sec"`
	Type           string   `bson:"type,omitempty" json:"type,omitempty"`
	SamplingRateHz int      `bson:"sampling_rate_hz,omitempty" json:"sampling_rate_hz,omitempty"`
	Bitrate        int      `bson:"bitrate,omitempty" json:"bitrate,omitempty"`
	MimeType       string   `bson:"mime_type,omitempty" json:"mime_type,omitempty"`
	FileSize       int64    `bson:"file_size,omitempty" json:"file_size,omitempty"`
	CoverURL       string         `bson:"cover_url,omitempty" json:"cover_url,omitempty"`
	Lyrics         string         `bson:"lyrics,omitempty" json:"lyrics,omitempty"`
	Titles         map[string]any `bson:"titles,omitempty" json:"titles,omitempty"`
}

// TelegramMeta stores Telegram file references and chat mapping.
type TelegramMeta struct {
	FileID       string            `bson:"file_id,omitempty" json:"file_id"`
	FileUniqueID string            `bson:"file_unique_id,omitempty" json:"file_unique_id,omitempty"`
	FileName     string            `bson:"file_name,omitempty" json:"file_name,omitempty"`
	FileSize     int64             `bson:"file_size,omitempty" json:"file_size,omitempty"`
	MimeType     string            `bson:"mime_type,omitempty" json:"mime_type,omitempty"`
	FileIDs      map[string]string `bson:"file_ids,omitempty" json:"file_ids,omitempty"`
}

// SpotifyMeta stores resolved Spotify match data and cover URLs.
type SpotifyMeta struct {
	URL            string `bson:"url,omitempty" json:"url,omitempty"`
	CoverURL       string `bson:"cover_url,omitempty" json:"cover_url,omitempty"`
	TrackSpotifyID string `bson:"track_spotify_id,omitempty" json:"track_spotify_id,omitempty"`
}

// Track represents the full track document stored in the audioTracks MongoDB collection.
type Track struct {
	ID              string         `bson:"_id" json:"_id"`
	Audio           AudioMeta      `bson:"audio" json:"audio"`
	Telegram        TelegramMeta   `bson:"telegram" json:"telegram"`
	Spotify         SpotifyMeta    `bson:"spotify,omitempty" json:"spotify,omitempty"`
	Lyrics          string         `bson:"lyrics,omitempty" json:"lyrics,omitempty"`
	Titles          map[string]any `bson:"titles,omitempty" json:"titles,omitempty"`
	SourceChatID    int64          `bson:"source_chat_id,omitempty" json:"source_chat_id,omitempty"`
	SourceMessageID int64          `bson:"source_message_id,omitempty" json:"source_message_id,omitempty"`
	CacheChatID     int64          `bson:"cache_chat_id,omitempty" json:"cache_chat_id,omitempty"`
	CacheMessageID  int64          `bson:"cache_message_id,omitempty" json:"cache_message_id,omitempty"`
	TopicID         int64          `bson:"topic_id,omitempty" json:"topic_id,omitempty"`
	TopicName       string         `bson:"topic_name,omitempty" json:"topic_name,omitempty"`
	PlayCount       int64          `bson:"play_count,omitempty" json:"play_count"`
	LikesCount      int64          `bson:"likes_count,omitempty" json:"likes_count"`
	CreatedAt       float64        `bson:"created_at,omitempty" json:"created_at"`
	UpdatedAt       float64        `bson:"updated_at,omitempty" json:"updated_at"`
	Deleted         bool           `bson:"deleted,omitempty" json:"deleted"`
	Liked           bool           `bson:"-" json:"liked"`
}

// EffectiveCoverURL returns the best available cover art URL.
func (t *Track) EffectiveCoverURL() string {
	if t.Spotify.CoverURL != "" {
		return t.Spotify.CoverURL
	}
	return t.Audio.CoverURL
}

// EffectiveArtist returns the best artist/performer string.
func (t *Track) EffectiveArtist() string {
	if t.Audio.Artist != "" {
		return t.Audio.Artist
	}
	return t.Audio.Performer
}

// EffectiveTitles returns the best available alternative titles map (original, romanized, etc.).
func (t *Track) EffectiveTitles() map[string]any {
	if len(t.Titles) > 0 {
		return t.Titles
	}
	if len(t.Audio.Titles) > 0 {
		return t.Audio.Titles
	}
	return nil
}

// BrowseItem represents a flattened track item optimized for list/feed UI views.
type BrowseItem struct {
	ID              string         `json:"_id"`
	SourceChatID    int64          `json:"source_chat_id,omitempty"`
	SourceMessageID int64          `json:"source_message_id,omitempty"`
	TopicID         int64          `json:"topic_id,omitempty"`
	TopicName       string         `json:"topic_name,omitempty"`
	Title           string         `json:"title"`
	Artist          string         `json:"artist"`
	Album           string         `json:"album,omitempty"`
	AlbumID         string         `json:"album_id,omitempty"`
	DurationSec     float64        `json:"duration_sec"`
	Type            string         `json:"type,omitempty"`
	SamplingRateHz  int            `json:"sampling_rate_hz,omitempty"`
	SpotifyURL      string         `json:"spotify_url,omitempty"`
	CoverURL        string         `json:"cover_url,omitempty"`
	Titles          map[string]any `json:"titles,omitempty"`
	CreatedAt       float64        `json:"created_at"`
	UpdatedAt       float64        `json:"updated_at"`
	Liked           bool           `json:"liked"`
}

// ToBrowseItem converts a full Track entity into a lightweight BrowseItem.
func (t *Track) ToBrowseItem() *BrowseItem {
	return &BrowseItem{
		ID:              t.ID,
		SourceChatID:    t.SourceChatID,
		SourceMessageID: t.SourceMessageID,
		TopicID:         t.TopicID,
		TopicName:       t.TopicName,
		Title:           t.Audio.Title,
		Artist:          t.EffectiveArtist(),
		Album:           t.Audio.Album,
		AlbumID:         t.Audio.AlbumID,
		DurationSec:     t.Audio.DurationSec,
		Type:            t.Audio.Type,
		SamplingRateHz:  t.Audio.SamplingRateHz,
		SpotifyURL:      t.Spotify.URL,
		CoverURL:        t.EffectiveCoverURL(),
		Titles:          t.EffectiveTitles(),
		CreatedAt:       t.CreatedAt,
		UpdatedAt:       t.UpdatedAt,
		Liked:           t.Liked,
	}
}

// BrowseResponse represents a paginated response of BrowseItem objects.
type BrowseResponse struct {
	Items      []*BrowseItem `json:"items"`
	Total      int64         `json:"total"`
	Page       int           `json:"page"`
	PerPage    int           `json:"per_page"`
	TotalPages int           `json:"total_pages"`
}

// TopicItem represents an aggregated topic with track counts and cover image.
type TopicItem struct {
	TopicID     int64  `json:"topic_id"`
	TopicName   string `json:"topic_name"`
	TracksCount int64  `json:"tracks_count"`
	CoverURL    string `json:"cover_url,omitempty"`
}

// TopicsResponse is the response payload for GET /topics.
type TopicsResponse struct {
	OK    bool         `json:"ok"`
	Total int          `json:"total"`
	Items []*TopicItem `json:"items"`
}

// ChannelItem represents an indexed Telegram source channel.
type ChannelItem struct {
	ID       int64  `json:"id"`
	Title    string `json:"title,omitempty"`
	Username string `json:"username,omitempty"`
	Type     string `json:"type,omitempty"`
}

// ChannelIDsResponse is the response payload for GET /channelids.
type ChannelIDsResponse struct {
	OK    bool          `json:"ok"`
	Items []ChannelItem `json:"items"`
}
