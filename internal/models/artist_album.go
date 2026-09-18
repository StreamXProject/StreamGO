package models

// Artist represents an artist record in the artists collection.
type Artist struct {
	ID          string  `bson:"_id" json:"_id"`
	Name        string  `bson:"name" json:"name"`
	MatchArtist string  `bson:"match_artist" json:"match_artist"`
	TracksCount int64   `bson:"tracks_count" json:"tracks_count"`
	CoverURL    string  `bson:"cover_url" json:"cover_url"`
	Followers   int64   `bson:"followers,omitempty" json:"followers"`
	CreatedAt   float64 `bson:"created_at,omitempty" json:"created_at"`
	UpdatedAt   float64 `bson:"updated_at,omitempty" json:"updated_at"`
}

// Album represents an album record in the albums collection.
type Album struct {
	ID            string   `bson:"_id" json:"_id"`
	Title         string   `bson:"title" json:"title"`
	Artist        string   `bson:"artist" json:"artist"`
	Artists       []string `bson:"artists,omitempty" json:"artists,omitempty"`
	CoverURL      string   `bson:"cover_url" json:"cover_url"`
	DurationTotal float64  `bson:"duration_total,omitempty" json:"duration_total"`
	MatchAlbum    string   `bson:"match_album,omitempty" json:"match_album"`
	MatchArtist   string   `bson:"match_artist,omitempty" json:"match_artist"`
	TracksCount   int64    `bson:"tracks_count" json:"tracks_count"`
	CreatedAt     float64  `bson:"created_at,omitempty" json:"created_at"`
	UpdatedAt     float64  `bson:"updated_at,omitempty" json:"updated_at"`
}

// ArtistDetail contains full artist information along with their top tracks and albums.
type ArtistDetail struct {
	Artist    *Artist       `json:"artist"`
	TopTracks []*BrowseItem `json:"top_tracks"`
	Albums    []*Album      `json:"albums"`
}

// AlbumDetail contains full album metadata and its ordered tracklist.
type AlbumDetail struct {
	Album  *Album        `json:"album"`
	Tracks []*BrowseItem `json:"tracks"`
}

// ArtistsResponse is a paginated list of artists.
type ArtistsResponse struct {
	Items      []*Artist `json:"items"`
	Total      int64     `json:"total"`
	Page       int       `json:"page"`
	PerPage    int       `json:"per_page"`
	TotalPages int       `json:"total_pages"`
}

// AlbumsResponse is a paginated list of albums.
type AlbumsResponse struct {
	Items      []*Album `json:"items"`
	Total      int64    `json:"total"`
	Page       int      `json:"page"`
	PerPage    int      `json:"per_page"`
	TotalPages int      `json:"total_pages"`
}
