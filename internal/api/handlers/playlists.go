package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"streamgo/internal/api"
	"streamgo/internal/api/middleware"
	"streamgo/internal/services"
)

// PlaylistHandler handles custom user playlist endpoints.
type PlaylistHandler struct {
	favSvc  *services.FavouritePlaylistService
	authSvc *services.AuthService
}

// NewPlaylistHandler creates a PlaylistHandler instance.
func NewPlaylistHandler(favSvc *services.FavouritePlaylistService, authSvc *services.AuthService) *PlaylistHandler {
	return &PlaylistHandler{
		favSvc:  favSvc,
		authSvc: authSvc,
	}
}

// Routes registers all playlist routes on the Chi router.
func (h *PlaylistHandler) Routes(r chi.Router) {
	requireAuth := middleware.RequireAuth(h.authSvc)
	optAuth := middleware.OptionalAuth(h.authSvc)

	// Protected routes (creation, modification, deletion)
	r.Group(func(sub chi.Router) {
		sub.Use(requireAuth)
		sub.Post("/playlists", h.Create)
		sub.Post("/me/playlists", h.Create)
		sub.Get("/playlists", h.ListMine)
		sub.Get("/me/playlists", h.ListMine)

		sub.Patch("/playlists/{id}", h.Update)
		sub.Patch("/me/playlists/{id}", h.Update)
		sub.Put("/playlists/{id}", h.Update)
		sub.Put("/me/playlists/{id}", h.Update)

		sub.Delete("/playlists/{id}", h.Delete)
		sub.Delete("/me/playlists/{id}", h.Delete)

		sub.Post("/playlists/{id}/tracks", h.AddTracks)
		sub.Post("/me/playlists/{id}/tracks", h.AddTracks)

		sub.Delete("/playlists/{id}/tracks/{track_id}", h.RemoveTrack)
		sub.Delete("/me/playlists/{id}/tracks/{track_id}", h.RemoveTrack)

		sub.Post("/playlists/{id}/reorder", h.Reorder)
		sub.Post("/me/playlists/{id}/reorder", h.Reorder)
	})

	// Public / Optional-auth routes (viewing playlist tracks)
	r.Group(func(sub chi.Router) {
		sub.Use(optAuth)
		sub.Get("/playlists/{id}", h.GetByID)
		sub.Get("/me/playlists/{id}", h.GetByID)
		sub.Get("/playlists/{id}/tracks", h.GetTracks)
		sub.Get("/me/playlists/{id}/tracks", h.GetTracks)
	})
}

// Create handles POST /playlists.
func (h *PlaylistHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		api.RespondError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	var payload struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		CoverURL    string `json:"cover_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		api.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	p, err := h.favSvc.CreatePlaylist(r.Context(), userID, payload.Title, payload.Description, payload.CoverURL)
	if err != nil {
		api.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	api.RespondJSON(w, http.StatusOK, p)
}

// ListMine handles GET /playlists and GET /me/playlists.
func (h *PlaylistHandler) ListMine(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		api.RespondError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	resp, err := h.favSvc.GetUserPlaylists(r.Context(), userID)
	if err != nil {
		api.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	api.RespondJSON(w, http.StatusOK, resp)
}

// GetByID handles GET /playlists/{id}.
func (h *PlaylistHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		api.RespondError(w, http.StatusBadRequest, "playlist id is required")
		return
	}

	detail, err := h.favSvc.GetPlaylist(r.Context(), id)
	if err != nil {
		api.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if detail == nil {
		api.RespondError(w, http.StatusNotFound, "playlist not found")
		return
	}

	api.RespondJSON(w, http.StatusOK, detail)
}

// Update handles PATCH /playlists/{id}.
func (h *PlaylistHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		api.RespondError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		api.RespondError(w, http.StatusBadRequest, "playlist id is required")
		return
	}

	var payload struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		CoverURL    string `json:"cover_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		api.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	updated, err := h.favSvc.UpdatePlaylist(r.Context(), id, userID, payload.Title, payload.Description, payload.CoverURL)
	if err != nil {
		api.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if updated == nil {
		api.RespondError(w, http.StatusNotFound, "playlist not found or unauthorized")
		return
	}

	api.RespondJSON(w, http.StatusOK, updated)
}

// Delete handles DELETE /playlists/{id}.
func (h *PlaylistHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		api.RespondError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		api.RespondError(w, http.StatusBadRequest, "playlist id is required")
		return
	}

	deleted, err := h.favSvc.DeletePlaylist(r.Context(), id, userID)
	if err != nil {
		api.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	api.RespondJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"deleted": deleted,
	})
}

// GetTracks handles GET /playlists/{id}/tracks.
func (h *PlaylistHandler) GetTracks(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		api.RespondError(w, http.StatusBadRequest, "playlist id is required")
		return
	}

	tracks, err := h.favSvc.GetPlaylistTracks(r.Context(), id)
	if err != nil {
		api.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	api.RespondJSON(w, http.StatusOK, map[string]any{
		"ok":    true,
		"total": len(tracks),
		"items": tracks,
	})
}

// AddTracks handles POST /playlists/{id}/tracks.
func (h *PlaylistHandler) AddTracks(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		api.RespondError(w, http.StatusBadRequest, "playlist id is required")
		return
	}

	var payload struct {
		TrackID  string   `json:"track_id"`
		TrackIDs []string `json:"track_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		api.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var trackIDs []string
	if len(payload.TrackIDs) > 0 {
		trackIDs = payload.TrackIDs
	} else if payload.TrackID != "" {
		trackIDs = []string{payload.TrackID}
	}

	if len(trackIDs) == 0 {
		api.RespondError(w, http.StatusBadRequest, "track_ids cannot be empty")
		return
	}

	if err := h.favSvc.AddTracksToPlaylist(r.Context(), id, trackIDs); err != nil {
		api.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	api.RespondJSON(w, http.StatusOK, map[string]any{
		"ok":    true,
		"count": len(trackIDs),
	})
}

// RemoveTrack handles DELETE /playlists/{id}/tracks/{track_id}.
func (h *PlaylistHandler) RemoveTrack(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	trackID := strings.TrimSpace(chi.URLParam(r, "track_id"))
	if id == "" || trackID == "" {
		api.RespondError(w, http.StatusBadRequest, "playlist id and track id are required")
		return
	}

	if err := h.favSvc.RemoveTrackFromPlaylist(r.Context(), id, trackID); err != nil {
		api.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	api.RespondJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"deleted": true,
	})
}

// Reorder handles POST /playlists/{id}/reorder.
func (h *PlaylistHandler) Reorder(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		api.RespondError(w, http.StatusBadRequest, "playlist id is required")
		return
	}

	var payload struct {
		TrackIDs []string `json:"track_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		api.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.favSvc.ReorderPlaylistTracks(r.Context(), id, payload.TrackIDs); err != nil {
		api.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	api.RespondJSON(w, http.StatusOK, map[string]any{
		"ok": true,
	})
}
