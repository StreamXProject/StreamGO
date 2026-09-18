package server

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"streamgo/internal/api/handlers"
	"streamgo/internal/config"
	"streamgo/internal/database"
	"streamgo/internal/logger"
	"streamgo/internal/repository"
	"streamgo/internal/services"
	"streamgo/internal/telegram"
)

// Server encapsulates the Chi router, config, and database handle.
type Server struct {
	Router *chi.Mux
	Config *config.Config
	DB     *database.Client
	TG     *telegram.Service
	start  time.Time
}

// New creates and configures a new HTTP router with middlewares and modular routes.
func New(cfg *config.Config, db *database.Client, tg *telegram.Service) *Server {
	r := chi.NewRouter()

	s := &Server{
		Router: r,
		Config: cfg,
		DB:     db,
		TG:     tg,
		start:  time.Now(),
	}

	// Global Middlewares
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	// Toggleable HTTP request logger: completely silent when APILogs is false
	r.Use(logger.HTTPMiddleware(cfg.APILogs))
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// CORS Configuration
	corsOrigins := []string{"*"}
	if cfg.CorsOrigin != "" && cfg.CorsOrigin != "*" {
		corsOrigins = []string{cfg.CorsOrigin}
	}

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   corsOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "HEAD"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "Range"},
		ExposedHeaders:   []string{"Link", "Content-Length", "Content-Range", "Accept-Ranges"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Core System Routes
	r.Get("/", s.handleIndex)
	r.Get("/health", s.handleHealth)

	// API Documentation Routes (Swagger UI & ReDoc)
	r.Get("/docs", s.handleDocs)
	r.Get("/docs/*", s.handleDocs)
	r.Head("/docs", s.handleDocs)
	r.Get("/redoc", s.handleRedoc)
	r.Head("/redoc", s.handleRedoc)
	r.Get("/openapi.json", s.handleOpenAPISpec)
	r.Head("/openapi.json", s.handleOpenAPISpec)

	// Mount Domain Routes when MongoDB is available
	if db != nil {
		// Repositories
		trackRepo := repository.NewTrackRepository(db)
		artistAlbumRepo := repository.NewArtistAlbumRepository(db)
		favRepo := repository.NewFavouritePlaylistRepository(db)
		userRepo := repository.NewUserRepository(db)

		// Services
		trackSvc := services.NewTrackService(trackRepo)
		streamSvc := services.NewStreamService(trackRepo, tg)
		artistAlbumSvc := services.NewArtistAlbumService(artistAlbumRepo, trackRepo)
		favPlaylistSvc := services.NewFavouritePlaylistService(favRepo, trackRepo)
		authSvc := services.NewAuthService(cfg, userRepo)

		// Handlers
		trackHandler := handlers.NewTrackHandler(trackSvc)
		topicHandler := handlers.NewTopicHandler(trackSvc)
		streamHandler := handlers.NewStreamHandler(streamSvc)
		artistHandler := handlers.NewArtistHandler(artistAlbumSvc)
		albumHandler := handlers.NewAlbumHandler(artistAlbumSvc)
		favHandler := handlers.NewFavouriteHandler(favPlaylistSvc, authSvc)
		playlistHandler := handlers.NewPlaylistHandler(favPlaylistSvc, authSvc)
		authHandler := handlers.NewAuthHandler(cfg, authSvc)
		mediaExtraHandler := handlers.NewMediaExtraHandler(trackSvc)

		// Register routes
		trackHandler.Routes(r)
		topicHandler.Routes(r)
		streamHandler.Routes(r)
		artistHandler.Routes(r)
		albumHandler.Routes(r)
		favHandler.Routes(r)
		playlistHandler.Routes(r)
		authHandler.Routes(r)
		mediaExtraHandler.Routes(r)
	}

	return s
}


func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"service": "StreamGO",
		"status":  "running",
		"version": "1.0.0",
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	dbStatus := "disabled"
	if s.DB != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := s.DB.Ping(ctx); err != nil {
			dbStatus = "unhealthy: " + err.Error()
		} else {
			dbStatus = "healthy"
		}
	}

	tgStatus := "disabled"
	if s.TG != nil {
		if s.TG.IsReady() {
			self := s.TG.Self()
			if self != nil {
				tgStatus = "connected (" + self.FirstName + ")"
			} else {
				tgStatus = "connected"
			}
		} else {
			tgStatus = "connecting"
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":   "ok",
		"database": dbStatus,
		"telegram": tgStatus,
		"uptime":   time.Since(s.start).String(),
		"time":     time.Now().UTC().Format(time.RFC3339),
	})
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
