package handlers

import (
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"

	"streamgo/internal/api"
	"streamgo/internal/services"
)

// StreamHandler handles audio streaming and download endpoints.
type StreamHandler struct {
	streamService *services.StreamService
}

// NewStreamHandler creates a new StreamHandler.
func NewStreamHandler(svc *services.StreamService) *StreamHandler {
	return &StreamHandler{streamService: svc}
}

// Routes mounts the streaming and download endpoints.
func (h *StreamHandler) Routes(r chi.Router) {
	r.Get("/tracks/{id}/stream", h.Stream)
	r.Head("/tracks/{id}/stream", h.Stream)
	r.Get("/stream/{id}", h.Stream)
	r.Head("/stream/{id}", h.Stream)

	r.Get("/tracks/{id}/download", h.Download)
	r.Head("/tracks/{id}/download", h.Download)
	r.Get("/download/{id}", h.Download)
	r.Head("/download/{id}", h.Download)

	r.Get("/tracks/{id}/warm", h.Warm)
}

// Stream handles GET and HEAD /tracks/{id}/stream with HTTP Range support.
func (h *StreamHandler) Stream(w http.ResponseWriter, r *http.Request) {
	trackID := chi.URLParam(r, "id")
	trackID = strings.TrimSpace(trackID)
	if trackID == "" {
		api.RespondError(w, http.StatusBadRequest, "track id is required")
		return
	}

	h.streamService.StreamTrack(w, r, trackID)
}

// Download handles GET and HEAD /tracks/{id}/download with Content-Disposition attachment.
func (h *StreamHandler) Download(w http.ResponseWriter, r *http.Request) {
	trackID := chi.URLParam(r, "id")
	trackID = strings.TrimSpace(trackID)
	if trackID == "" {
		api.RespondError(w, http.StatusBadRequest, "track id is required")
		return
	}

	track, _ := h.streamService.GetTrack(r.Context(), trackID)
	if track != nil {
		filename := track.Audio.Title
		if track.Audio.Artist != "" {
			filename = fmt.Sprintf("%s - %s", track.Audio.Artist, track.Audio.Title)
		}
		if filename == "" {
			filename = track.Telegram.FileName
		}
		if filename == "" {
			filename = fmt.Sprintf("track-%s", track.ID)
		}
		ext := track.Audio.Type
		if (h.streamService.ALACService() != nil && h.streamService.ALACService().ShouldDecodeALAC(r, track)) || strings.ToLower(r.URL.Query().Get("format")) == "flac" {
			ext = "flac"
		}
		if ext == "" {
			ext = "mp3"
		}
		cleanFilename := strings.TrimSuffix(filename, filepath.Ext(filename))
		filename = fmt.Sprintf("%s.%s", cleanFilename, ext)

		fallback := strings.ReplaceAll(filename, `"`, `_`)
		encoded := url.QueryEscape(filename)
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"; filename*=UTF-8''%s`, fallback, encoded))
	} else {
		w.Header().Set("Content-Disposition", `attachment; filename="track.mp3"`)
	}

	h.streamService.StreamTrack(w, r, trackID)
}

// Warm handles GET /tracks/{id}/warm to prewarm playback cache.
func (h *StreamHandler) Warm(w http.ResponseWriter, r *http.Request) {
	trackID := chi.URLParam(r, "id")
	trackID = strings.TrimSpace(trackID)
	if trackID == "" {
		api.RespondError(w, http.StatusBadRequest, "track id is required")
		return
	}

	h.streamService.WarmTrack(r.Context(), trackID)
	api.RespondJSON(w, http.StatusOK, map[string]any{"ok": true, "ready": true})
}

