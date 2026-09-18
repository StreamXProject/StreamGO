package handlers

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"streamgo/internal/api"
	"streamgo/internal/services"
)

// StreamHandler handles audio streaming endpoints.
type StreamHandler struct {
	streamService *services.StreamService
}

// NewStreamHandler creates a new StreamHandler.
func NewStreamHandler(svc *services.StreamService) *StreamHandler {
	return &StreamHandler{streamService: svc}
}

// Routes mounts the streaming endpoints.
func (h *StreamHandler) Routes(r chi.Router) {
	r.Get("/tracks/{id}/stream", h.Stream)
	r.Head("/tracks/{id}/stream", h.Stream)
	r.Get("/stream/{id}", h.Stream)
	r.Head("/stream/{id}", h.Stream)
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
