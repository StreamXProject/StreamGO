package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"streamgo/internal/config"
	"streamgo/internal/database"
	"streamgo/internal/logger"
	"streamgo/internal/server"
	"streamgo/internal/services"
	"streamgo/internal/telegram"
)

var log = logger.New("main")

func main() {
	fmt.Println("==================================================")
	fmt.Println("        StreamGO - High Performance Media API     ")
	fmt.Println("==================================================")

	// 1. Load Configuration
	cfg := config.Load()
	log.Infof("loaded config: PORT=%s, DB=%s, API_LOGS=%v", cfg.Port, cfg.DatabaseName, cfg.APILogs)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 2. Initialize MongoDB Connection
	var dbClient *database.Client
	if cfg.MongoURI != "" {
		dbCtx, dbCancel := context.WithTimeout(ctx, 10*time.Second)
		client, err := database.Connect(dbCtx, cfg.MongoURI, cfg.DatabaseName)
		dbCancel()
		if err != nil {
			log.Warnf("MongoDB connection failed: %v", err)
			log.Warn("Continuing with MongoDB in disabled state...")
		} else {
			dbClient = client
			defer func() {
				closeCtx, closeCancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer closeCancel()
				_ = dbClient.Close(closeCtx)
			}()
		}
	}

	// 3. Initialize Internal Auxiliary Services
	coverSearch := services.NewCoverSearchService()
	lyricsSvc := services.NewLyricsEnrichmentService()
	dedupSvc := services.NewDedupService(dbClient)
	accessFilter := services.NewAccessFilter(cfg, dbClient)
	enrichSvc := services.NewEnrichmentService(dbClient, coverSearch, lyricsSvc)

	if dbClient != nil {
		enrichSvc.Start(ctx, 2)
	}

	// 4. Initialize Telegram Service (Multi-Client MTProto Pool)
	var tgService *telegram.Service
	if cfg.ApiID > 0 && cfg.ApiHash != "" {
		svc, err := telegram.New(cfg)
		if err != nil {
			log.Warnf("Failed to initialize Telegram service: %v", err)
		} else {
			tgService = svc
			enrichSvc.SetDownloader(tgService)

			// Attach Ingestion Listener for live channel/group audio uploads
			listener := telegram.NewIngestionListener(cfg, dbClient, accessFilter, dedupSvc, enrichSvc.TriggerEnrich)
			listener.SetupDispatcher(&tgService.Dispatcher)

			log.Info("Starting Telegram MTProto multi-client pool in background...")
			go func() {
				if err := tgService.Start(ctx); err != nil {
					log.Errorf("Telegram client start failed: %v", err)
				}
			}()
			defer tgService.Stop()
		}
	} else {
		log.Info("Telegram credentials (API_ID/API_HASH) not configured; skipping Telegram init.")
	}

	// 4. Initialize HTTP Server with Chi Router
	srv := server.New(cfg, dbClient, tgService)
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      srv.Router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// 5. Start HTTP Listener in a separate goroutine
	go func() {
		log.Infof("HTTP server listening on http://0.0.0.0:%s", cfg.Port)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// 6. Graceful Shutdown on OS Signal (SIGINT, SIGTERM)
	signal.Ignore(syscall.SIGHUP)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	log.Infof("received shutdown signal (%s), starting graceful shutdown...", sig)

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Errorf("HTTP server forced shutdown error: %v", err)
	} else {
		log.Info("HTTP server shutdown cleanly.")
	}

	log.Info("StreamGO stopped. Goodbye!")
}
