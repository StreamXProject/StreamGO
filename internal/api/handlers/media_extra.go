package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"

	"streamgo/internal/api"
	"streamgo/internal/services"
)

// MediaExtraHandler handles static cover images and track lyrics.
type MediaExtraHandler struct {
	trackSvc *services.TrackService
}

// NewMediaExtraHandler creates a new MediaExtraHandler.
func NewMediaExtraHandler(trackSvc *services.TrackService) *MediaExtraHandler {
	return &MediaExtraHandler{trackSvc: trackSvc}
}

// Routes mounts cover and lyrics endpoints on Chi router.
func (h *MediaExtraHandler) Routes(r chi.Router) {
	r.Get("/covers/file/{file_key}.png", h.GetCoverFile)
	r.Head("/covers/file/{file_key}.png", h.GetCoverFile)
	r.Get("/cover/{id}", h.GetCoverRedirect)
	r.Head("/cover/{id}", h.GetCoverRedirect)
	r.Get("/lyrics/{id}", h.GetLyrics)
	r.Get("/tracks/{id}/lyrics", h.GetLyrics)
}


// GetCoverFile serves cached generated cover PNG files.
func (h *MediaExtraHandler) GetCoverFile(w http.ResponseWriter, r *http.Request) {
	fileKey := strings.TrimSpace(chi.URLParam(r, "file_key"))
	if fileKey == "" {
		api.RespondError(w, http.StatusBadRequest, "file_key is required")
		return
	}

	// Look in GenCovers in current or parent dirs
	searchDirs := []string{
		"GenCovers",
		"../GenCovers",
		"../../GenCovers",
		"/home/misfit/Work/StreamXBot/GenCovers",
	}

	var foundPath string
	for _, dir := range searchDirs {
		candidate := filepath.Join(dir, fmt.Sprintf("%s.png", fileKey))
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			foundPath = candidate
			break
		}
	}

	if foundPath == "" {
		api.RespondError(w, http.StatusNotFound, "cover not found")
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	http.ServeFile(w, r, foundPath)
}

// GetCoverRedirect resolves cover URL for a track and redirects.
func (h *MediaExtraHandler) GetCoverRedirect(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		api.RespondError(w, http.StatusBadRequest, "id is required")
		return
	}

	track, err := h.trackSvc.GetTrack(r.Context(), id)
	if err != nil || track == nil {
		api.RespondError(w, http.StatusNotFound, "track not found")
		return
	}

	item := track.ToBrowseItem()
	coverURL := item.CoverURL
	if coverURL == "" {
		coverURL = track.EffectiveCoverURL()
	}

	if coverURL == "" {
		api.RespondError(w, http.StatusNotFound, "cover not available")
		return
	}

	http.Redirect(w, r, coverURL, http.StatusFound)
}

// GetLyrics returns lyrics for a track either in plain text or JSON.
func (h *MediaExtraHandler) GetLyrics(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		api.RespondError(w, http.StatusBadRequest, "track id is required")
		return
	}

	track, err := h.trackSvc.GetTrack(r.Context(), id)
	if err != nil || track == nil {
		api.RespondError(w, http.StatusNotFound, "track not found")
		return
	}

	lyrics := track.Lyrics
	if lyrics == "" {
		lyrics = track.Audio.Lyrics
	}

	if lyrics == "" {
		api.RespondError(w, http.StatusNotFound, "lyrics not found")
		return
	}


	format := r.URL.Query().Get("format")
	accept := r.Header.Get("Accept")
	if format == "plain" || strings.Contains(accept, "text/plain") {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(lyrics))
		return
	}

	api.RespondJSON(w, http.StatusOK, map[string]any{
		"ok":       true,
		"track_id": id,
		"lyrics":   lyrics,
	})
}
