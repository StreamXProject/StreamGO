package models

// UserFavourite represents a track liked by a user in the userFavourites collection.
type UserFavourite struct {
	ID        string  `bson:"_id,omitempty" json:"_id,omitempty"`
	UserID    int64   `bson:"user_id" json:"user_id"`
	TrackID   string  `bson:"track_id" json:"track_id"`
	CreatedAt float64 `bson:"created_at" json:"created_at"`
}

// UserFavouriteArtist represents a followed artist in user_favourite_artists.
type UserFavouriteArtist struct {
	ID        string  `bson:"_id,omitempty" json:"_id,omitempty"`
	UserID    int64   `bson:"user_id" json:"user_id"`
	ArtistID  string  `bson:"artist_id" json:"artist_id"`
	CreatedAt float64 `bson:"created_at" json:"created_at"`
}

// UserPlaylist represents a custom playlist in the userPlaylists collection.
type UserPlaylist struct {
	ID          string  `bson:"_id" json:"_id"`
	UserID      int64   `bson:"user_id" json:"user_id"`
	Title       string  `bson:"title" json:"title"`
	Description string  `bson:"description,omitempty" json:"description"`
	CoverURL    string  `bson:"cover_url,omitempty" json:"cover_url"`
	TrackCount  int64   `bson:"track_count,omitempty" json:"track_count"`
	CreatedAt   float64 `bson:"created_at,omitempty" json:"created_at"`
	UpdatedAt   float64 `bson:"updated_at,omitempty" json:"updated_at"`
}

// PlaylistTrack represents a track association in the playlistTracks collection.
type PlaylistTrack struct {
	ID         string  `bson:"_id" json:"_id"`
	PlaylistID string  `bson:"playlist_id" json:"playlist_id"`
	TrackID    string  `bson:"track_id" json:"track_id"`
	Position   int     `bson:"position" json:"position"`
	AddedAt    float64 `bson:"added_at" json:"added_at"`
}

// PlaylistDetail bundles a playlist with its full track entities.
type PlaylistDetail struct {
	Playlist *UserPlaylist `json:"playlist"`
	Tracks   []*BrowseItem `json:"tracks"`
}

// PlaylistsResponse is the list of user playlists.
type PlaylistsResponse struct {
	OK    bool            `json:"ok"`
	Total int             `json:"total"`
	Items []*UserPlaylist `json:"items"`
}

// FavouriteIDsResponse contains array of liked track IDs.
type FavouriteIDsResponse struct {
	OK  bool     `json:"ok"`
	IDs []string `json:"ids"`
}

// FavouriteArtistsResponse contains array of followed artist IDs.
type FavouriteArtistsResponse struct {
	OK  bool     `json:"ok"`
	IDs []string `json:"ids"`
}

// ListeningEventItem represents a single telemetry playback event.
type ListeningEventItem struct {
	ID         string  `json:"id" bson:"id"`
	TrackID    string  `json:"track_id" bson:"track_id"`
	PlayedAt   float64 `json:"played_at" bson:"played_at"`
	StartedAt  float64 `json:"started_at" bson:"started_at"`
	PlayedMs   float64 `json:"played_ms" bson:"played_ms"`
	DurationMs float64 `json:"duration_ms" bson:"duration_ms"`
	Completed  bool    `json:"completed" bson:"completed"`
	Skipped    bool    `json:"skipped" bson:"skipped"`
	Source     string  `json:"source" bson:"source"`
	SessionID  string  `json:"session_id,omitempty" bson:"session_id,omitempty"`
}

// ListeningEventsPayload is the payload for batch listening events.
type ListeningEventsPayload struct {
	Events []ListeningEventItem `json:"events"`
}

