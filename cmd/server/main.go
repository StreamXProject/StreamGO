package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"streamgo/internal/config"
	"streamgo/internal/database"
	"streamgo/internal/server"
	"streamgo/internal/telegram"
)

func main() {
	log.Println("==================================================")
	log.Println("        StreamGO - High Performance Media API     ")
	log.Println("==================================================")

	// 1. Load Configuration
	cfg := config.Load()
	log.Printf("[main] loaded config: PORT=%s, DB=%s", cfg.Port, cfg.DatabaseName)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 2. Initialize MongoDB Connection
	var dbClient *database.Client
	if cfg.MongoURI != "" {
		dbCtx, dbCancel := context.WithTimeout(ctx, 10*time.Second)
		client, err := database.Connect(dbCtx, cfg.MongoURI, cfg.DatabaseName)
		dbCancel()
		if err != nil {
			log.Printf("[main] WARNING: MongoDB connection failed: %v", err)
			log.Println("[main] Continuing with MongoDB in disabled state...")
		} else {
			dbClient = client
			defer func() {
				closeCtx, closeCancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer closeCancel()
				_ = dbClient.Close(closeCtx)
			}()
		}
	}

	// 3. Initialize Telegram Service (optional / background)
	var tgService *telegram.Service
	if cfg.ApiID > 0 && cfg.ApiHash != "" {
		svc, err := telegram.New(cfg)
		if err != nil {
			log.Printf("[main] WARNING: Failed to initialize Telegram service: %v", err)
		} else {
			tgService = svc
			log.Println("[main] Starting Telegram MTProto client in background...")
			go func() {
				if err := tgService.Start(ctx); err != nil {
					log.Printf("[main] Telegram client start failed: %v", err)
				}
			}()
			defer tgService.Stop()
		}
	} else {
		log.Println("[main] Telegram credentials (API_ID/API_HASH) not configured; skipping Telegram init.")
	}

	// 4. Initialize HTTP Server with Chi Router
	srv := server.New(cfg, dbClient)
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      srv.Router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// 5. Start HTTP Listener in a separate goroutine
	go func() {
		log.Printf("[main] HTTP server listening on http://0.0.0.0:%s", cfg.Port)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("[main] HTTP server error: %v", err)
		}
	}()

	// 6. Graceful Shutdown on OS Signal (SIGINT, SIGTERM)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	log.Printf("[main] received shutdown signal (%s), starting graceful shutdown...", sig)

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("[main] HTTP server forced shutdown error: %v", err)
	} else {
		log.Println("[main] HTTP server shutdown cleanly.")
	}

	log.Println("[main] StreamGO stopped. Goodbye!")
}
