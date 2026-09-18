package handlers

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"streamgo/internal/api"
	"streamgo/internal/services"
)

// TrackHandler handles HTTP routes for track operations.
type TrackHandler struct {
	trackService *services.TrackService
}

// NewTrackHandler creates a new TrackHandler.
func NewTrackHandler(svc *services.TrackService) *TrackHandler {
	return &TrackHandler{trackService: svc}
}

// Routes mounts track endpoints on the given Chi router.
func (h *TrackHandler) Routes(r chi.Router) {
	r.Get("/tracks", h.List)
	r.Get("/browse", h.List)
	r.Get("/search", h.Search)
	r.Get("/tracks/random", h.Random)
	r.Get("/tracks/{id}", h.GetByID)
}

// List handles GET /tracks and GET /browse with pagination, sorting, and filters.
func (h *TrackHandler) List(w http.ResponseWriter, r *http.Request) {
	page := api.ParseQueryInt(r, "page", 1)
	perPage := api.ParseQueryInt(r, "per_page", 20)
	if limit := api.ParseQueryInt(r, "limit", 0); limit > 0 {
		perPage = limit
	}

	sortField := strings.TrimSpace(r.URL.Query().Get("sort"))
	topicName := strings.TrimSpace(r.URL.Query().Get("topic"))
	channelID := api.ParseQueryInt64(r, "channel_id", 0)

	resp, err := h.trackService.Browse(r.Context(), page, perPage, sortField, topicName, channelID)
	if err != nil {
		api.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	api.RespondJSON(w, http.StatusOK, resp)
}

// GetByID handles GET /tracks/{id}.
func (h *TrackHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if strings.TrimSpace(id) == "" {
		api.RespondError(w, http.StatusBadRequest, "track id is required")
		return
	}

	track, err := h.trackService.GetTrack(r.Context(), id)
	if err != nil {
		api.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if track == nil {
		api.RespondError(w, http.StatusNotFound, "track not found")
		return
	}

	api.RespondJSON(w, http.StatusOK, track)
}

// Search handles GET /search with text queries.
func (h *TrackHandler) Search(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		query = strings.TrimSpace(r.URL.Query().Get("query"))
	}
	limit := api.ParseQueryInt(r, "limit", 50)

	items, err := h.trackService.Search(r.Context(), query, limit)
	if err != nil {
		api.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	api.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"query": query,
		"total": len(items),
		"items": items,
	})
}

// Random handles GET /tracks/random.
func (h *TrackHandler) Random(w http.ResponseWriter, r *http.Request) {
	limit := api.ParseQueryInt(r, "limit", 20)
	channelID := api.ParseQueryInt64(r, "channel_id", 0)

	items, err := h.trackService.GetRandom(r.Context(), limit, channelID)
	if err != nil {
		api.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	api.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"total": len(items),
		"items": items,
	})
}
