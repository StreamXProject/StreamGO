package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"streamgo/internal/api"
	"streamgo/internal/api/middleware"
	"streamgo/internal/config"
	"streamgo/internal/models"
	"streamgo/internal/services"
)

// AuthHandler handles authentication routes for Telegram WebApp and widgets.
type AuthHandler struct {
	cfg     *config.Config
	authSvc *services.AuthService
}

// NewAuthHandler creates an AuthHandler instance.
func NewAuthHandler(cfg *config.Config, authSvc *services.AuthService) *AuthHandler {
	return &AuthHandler{
		cfg:     cfg,
		authSvc: authSvc,
	}
}

// Routes registers authentication endpoints on Chi router.
func (h *AuthHandler) Routes(r chi.Router) {
	r.Post("/auth/telegram", h.TelegramLogin)
	r.Post("/auth/tg/login", h.TelegramLogin)
	r.Post("/auth/login", h.TelegramLogin)
	r.Post("/webapp/verify", h.TelegramLogin)

	r.Post("/auth/telegram/widget", h.WidgetLogin)

	r.Get("/auth/telegram/config", h.Config)
	r.Get("/webapp/config", h.Config)

	// Authenticated profile route
	r.Group(func(sub chi.Router) {
		sub.Use(middleware.RequireAuth(h.authSvc))
		sub.Get("/auth/me", h.Me)
	})
}

// TelegramLogin handles POST /auth/telegram and /webapp/verify.
func (h *AuthHandler) TelegramLogin(w http.ResponseWriter, r *http.Request) {
	var initData string

	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/x-www-form-urlencoded") || strings.Contains(contentType, "multipart/form-data") {
		_ = r.ParseForm()
		initData = r.FormValue("init_data")
		if initData == "" {
			initData = r.FormValue("initData")
		}
	} else {
		var payload models.TelegramWebAppRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err == nil {
			initData = payload.InitData
		}
	}

	initData = strings.TrimSpace(initData)
	if initData == "" {
		api.RespondError(w, http.StatusBadRequest, "init_data is required")
		return
	}

	resp, err := h.authSvc.AuthenticateTelegram(r.Context(), initData)
	if err != nil {
		api.RespondError(w, http.StatusUnauthorized, err.Error())
		return
	}

	api.RespondJSON(w, http.StatusOK, resp)
}

// WidgetLogin handles POST /auth/telegram/widget.
func (h *AuthHandler) WidgetLogin(w http.ResponseWriter, r *http.Request) {
	var payload models.TelegramWidgetRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		api.RespondError(w, http.StatusBadRequest, "invalid widget payload")
		return
	}

	resp, err := h.authSvc.AuthenticateWidget(r.Context(), payload)
	if err != nil {
		api.RespondError(w, http.StatusUnauthorized, err.Error())
		return
	}

	api.RespondJSON(w, http.StatusOK, resp)
}

// Me handles GET /auth/me.
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		api.RespondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	user, err := h.authSvc.GetUserByID(r.Context(), userID)
	if err != nil {
		api.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if user == nil {
		api.RespondError(w, http.StatusNotFound, "user not found")
		return
	}

	api.RespondJSON(w, http.StatusOK, map[string]any{
		"ok":   true,
		"user": user,
	})
}

// Config handles GET /auth/telegram/config and GET /webapp/config.
func (h *AuthHandler) Config(w http.ResponseWriter, r *http.Request) {
	botID := ""
	if parts := strings.Split(h.cfg.BotToken, ":"); len(parts) > 0 {
		botID = parts[0]
	}

	api.RespondJSON(w, http.StatusOK, map[string]any{
		"ok":       true,
		"bot_id":   botID,
		"has_auth": true,
	})
}
