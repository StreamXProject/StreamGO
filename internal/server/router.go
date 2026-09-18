package server

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"streamgo/internal/config"
	"streamgo/internal/database"
	"streamgo/internal/logger"
)

// Server encapsulates the Chi router, config, and database handle.
type Server struct {
	Router *chi.Mux
	Config *config.Config
	DB     *database.Client
	start  time.Time
}

// New creates and configures a new HTTP router with middlewares and core routes.
func New(cfg *config.Config, db *database.Client) *Server {
	r := chi.NewRouter()

	s := &Server{
		Router: r,
		Config: cfg,
		DB:     db,
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
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "HEAD"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "Range"},
		ExposedHeaders:   []string{"Link", "Content-Length", "Content-Range", "Accept-Ranges"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Core Routes
	r.Get("/", s.handleIndex)
	r.Get("/health", s.handleHealth)

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

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":   "ok",
		"database": dbStatus,
		"uptime":   time.Since(s.start).String(),
		"time":     time.Now().UTC().Format(time.RFC3339),
	})
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
