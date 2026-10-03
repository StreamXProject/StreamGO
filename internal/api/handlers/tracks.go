package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"streamgo/internal/api"
	"streamgo/internal/api/middleware"
	"streamgo/internal/config"
	"streamgo/internal/services"
)

// TrackHandler handles HTTP routes for track operations.
type TrackHandler struct {
	trackService *services.TrackService
	authSvc      *services.AuthService
	cfg          *config.Config
}

// NewTrackHandler creates a new TrackHandler.
func NewTrackHandler(svc *services.TrackService, authSvc *services.AuthService, cfg *config.Config) *TrackHandler {
	return &TrackHandler{
		trackService: svc,
		authSvc:      authSvc,
		cfg:          cfg,
	}
}

// Routes mounts track endpoints on the given Chi router.
func (h *TrackHandler) Routes(r chi.Router) {
	r.Get("/tracks", h.List)
	r.Get("/browse", h.List)
	r.Get("/search", h.Search)
	r.Get("/tracks/search", h.Search)
	r.Get("/tracks/random", h.Random)
	r.Get("/tracks/shuffle", h.Random)
	r.Get("/library/shuffle", h.Random)
	r.Get("/tracks/{id}", h.GetByID)

	r.Group(func(sub chi.Router) {
		if h.authSvc != nil {
			sub.Use(middleware.RequireAuth(h.authSvc))
		}
		sub.Delete("/tracks/{id}", h.Delete)
	})
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

	if effType := track.EffectiveType(); effType != "" {
		track.Audio.Type = effType
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

// checkAdmin verifies caller has administrator privileges (consistent with access_control.go).
func (h *TrackHandler) checkAdmin(r *http.Request) error {
	if middleware.IsGuest(r.Context()) {
		return nil
	}

	userID, ok := middleware.GetUserID(r.Context())
	if !ok || userID <= 0 {
		return fmt.Errorf("authentication required")
	}

	if h.cfg == nil || (len(h.cfg.OwnerIDs) == 0 && len(h.cfg.SudoUsers) == 0) {
		return nil
	}

	for _, oid := range h.cfg.OwnerIDs {
		if oid == userID {
			return nil
		}
	}
	for _, sid := range h.cfg.SudoUsers {
		if sid == userID {
			return nil
		}
	}

	return fmt.Errorf("forbidden: administrator privileges required")
}

// Delete handles DELETE /tracks/{id}.
func (h *TrackHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if err := h.checkAdmin(r); err != nil {
		if strings.Contains(err.Error(), "forbidden") {
			api.RespondError(w, http.StatusForbidden, err.Error())
		} else {
			api.RespondError(w, http.StatusUnauthorized, err.Error())
		}
		return
	}

	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		api.RespondError(w, http.StatusBadRequest, "track id is required")
		return
	}

	hard := api.ParseQueryBool(r, "hard", false)
	purgeCache := api.ParseQueryBool(r, "purge_cache", true)
	purgeArtwork := api.ParseQueryBool(r, "purge_artwork", true)

	opts := services.DeleteTrackOptions{
		Hard:         hard,
		PurgeCache:   purgeCache,
		PurgeArtwork: purgeArtwork,
	}

	res, err := h.trackService.DeleteTrack(r.Context(), id, opts)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			api.RespondError(w, http.StatusNotFound, "track not found")
			return
		}
		api.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	api.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"ok":              true,
		"status":          http.StatusOK,
		"track_id":        res.TrackID,
		"mode":            res.Mode,
		"cache_purged":    res.CachePurged,
		"purged_artworks": res.PurgedArtworks,
		"message":         res.Message,
	})
}
